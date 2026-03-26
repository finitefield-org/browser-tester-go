package script

import (
	"fmt"
	"strings"
)

type RuntimeConfig struct {
	StepLimit int
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		StepLimit: 10_000,
	}
}

type HostBindings interface {
	Call(method string, args []Value) (Value, error)
}

type DispatchRequest struct {
	Source string
}

type DispatchResult struct {
	Value Value
}

type Runtime struct {
	config RuntimeConfig
	host   HostBindings
}

type classicJSEnvironment struct {
	parent   *classicJSEnvironment
	bindings map[string]classicJSBinding
}

type classicJSBinding struct {
	value   jsValue
	mutable bool
}

func newClassicJSEnvironment() *classicJSEnvironment {
	return &classicJSEnvironment{
		bindings: make(map[string]classicJSBinding),
	}
}

func (e *classicJSEnvironment) clone() *classicJSEnvironment {
	if e == nil {
		return newClassicJSEnvironment()
	}
	return &classicJSEnvironment{
		parent:   e,
		bindings: make(map[string]classicJSBinding),
	}
}

func (e *classicJSEnvironment) cloneDetached() *classicJSEnvironment {
	if e == nil {
		return newClassicJSEnvironment()
	}
	clonedParent := e.parent.cloneDetached()
	cloned := &classicJSEnvironment{
		parent:   clonedParent,
		bindings: make(map[string]classicJSBinding, len(e.bindings)),
	}
	for name, binding := range e.bindings {
		cloned.bindings[name] = binding
	}
	return cloned
}

func (e *classicJSEnvironment) declare(name string, value jsValue, mutable bool) error {
	if e == nil {
		return NewError(ErrorKindRuntime, "classic-JS environment is unavailable")
	}
	if e.bindings == nil {
		e.bindings = make(map[string]classicJSBinding)
	}
	if _, exists := e.bindings[name]; exists {
		return NewError(ErrorKindParse, fmt.Sprintf("duplicate lexical declaration for %q in this bounded classic-JS slice", name))
	}
	e.bindings[name] = classicJSBinding{value: value, mutable: mutable}
	return nil
}

func (e *classicJSEnvironment) lookup(name string) (jsValue, bool) {
	for current := e; current != nil; current = current.parent {
		if len(current.bindings) == 0 {
			continue
		}
		binding, ok := current.bindings[name]
		if ok {
			return binding.value, true
		}
	}
	return jsValue{}, false
}

func (e *classicJSEnvironment) assign(name string, value jsValue) error {
	for current := e; current != nil; current = current.parent {
		if len(current.bindings) == 0 {
			continue
		}
		binding, ok := current.bindings[name]
		if !ok {
			continue
		}
		if !binding.mutable {
			return NewError(ErrorKindRuntime, fmt.Sprintf("cannot assign to immutable binding %q in this bounded classic-JS slice", name))
		}
		binding.value = value
		current.bindings[name] = binding
		return nil
	}
	return NewError(ErrorKindUnsupported, fmt.Sprintf("assignment target %q is not a declared local binding in this bounded classic-JS slice", name))
}

func NewRuntime(host HostBindings) *Runtime {
	return NewRuntimeWithConfig(DefaultRuntimeConfig(), host)
}

func NewRuntimeWithConfig(config RuntimeConfig, host HostBindings) *Runtime {
	cfg := config
	if cfg.StepLimit <= 0 {
		cfg.StepLimit = DefaultRuntimeConfig().StepLimit
	}
	return &Runtime{
		config: cfg,
		host:   host,
	}
}

func (r *Runtime) Config() RuntimeConfig {
	if r == nil {
		return DefaultRuntimeConfig()
	}
	return r.config
}

func (r *Runtime) Dispatch(request DispatchRequest) (DispatchResult, error) {
	if r == nil {
		return DispatchResult{}, NewError(ErrorKindRuntime, "script runtime is unavailable")
	}

	source := strings.TrimSpace(request.Source)
	if source == "" || source == "noop" {
		return DispatchResult{Value: UndefinedValue()}, nil
	}

	statements, err := splitScriptStatements(source)
	if err != nil {
		if scriptErr, ok := err.(Error); ok {
			return DispatchResult{}, scriptErr
		}
		return DispatchResult{}, NewError(ErrorKindParse, err.Error())
	}

	if len(statements) == 0 {
		return DispatchResult{Value: UndefinedValue()}, nil
	}

	env := newClassicJSEnvironment()
	var last Value = UndefinedValue()
	for _, statement := range statements {
		result, err := r.dispatchStatement(statement, env)
		if err != nil {
			return DispatchResult{}, err
		}
		last = result.Value
	}
	return DispatchResult{Value: last}, nil
}

func (r *Runtime) dispatchStatement(source string, env *classicJSEnvironment) (DispatchResult, error) {
	if source == "" || source == "noop" {
		return DispatchResult{Value: UndefinedValue()}, nil
	}

	if strings.HasPrefix(strings.TrimSpace(source), "host:") {
		return r.dispatchLegacyStatement(source, env)
	}

	value, err := evalClassicJSProgram(source, r.host, env, r.config.StepLimit)
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Value: value}, nil
}

func (r *Runtime) dispatchLegacyStatement(source string, env *classicJSEnvironment) (DispatchResult, error) {
	method, args, err := parseHostInvocation(strings.TrimPrefix(strings.TrimSpace(source), "host:"))
	if err != nil {
		return DispatchResult{}, NewError(ErrorKindParse, err.Error())
	}
	if r.host == nil {
		return DispatchResult{}, NewError(ErrorKindHost, "host bindings are unavailable")
	}
	resolvedArgs, err := r.resolveArgs(args, env)
	if err != nil {
		return DispatchResult{}, err
	}
	value, err := r.host.Call(method, resolvedArgs)
	if err != nil {
		return DispatchResult{}, NewError(ErrorKindHost, err.Error())
	}
	return DispatchResult{Value: value}, nil
}

func (r *Runtime) resolveArgs(args []Value, env *classicJSEnvironment) ([]Value, error) {
	if len(args) == 0 {
		return nil, nil
	}
	resolved := make([]Value, len(args))
	for i, arg := range args {
		if arg.Kind != ValueKindInvocation {
			resolved[i] = arg
			continue
		}
		result, err := r.dispatchSourceWithEnv(arg.Invocation, env)
		if err != nil {
			return nil, err
		}
		resolved[i] = result.Value
	}
	return resolved, nil
}

func (r *Runtime) dispatchSourceWithEnv(source string, env *classicJSEnvironment) (DispatchResult, error) {
	if r == nil {
		return DispatchResult{}, NewError(ErrorKindRuntime, "script runtime is unavailable")
	}
	trimmed := strings.TrimSpace(source)
	if trimmed == "" || trimmed == "noop" {
		return DispatchResult{Value: UndefinedValue()}, nil
	}
	if strings.HasPrefix(trimmed, "host:") {
		return r.dispatchLegacyStatement(trimmed, env)
	}
	value, err := evalClassicJSProgram(trimmed, r.host, env, r.config.StepLimit)
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Value: value}, nil
}

func evalClassicJSProgram(source string, host HostBindings, env *classicJSEnvironment, stepLimit int) (Value, error) {
	if env == nil {
		env = newClassicJSEnvironment()
	}
	if stepLimit <= 0 {
		stepLimit = DefaultRuntimeConfig().StepLimit
	}

	statements, err := splitScriptStatements(source)
	if err != nil {
		return UndefinedValue(), NewError(ErrorKindParse, err.Error())
	}
	if len(statements) == 0 {
		return UndefinedValue(), nil
	}

	var last Value = UndefinedValue()
	for _, statement := range statements {
		value, err := evalClassicJSStatementWithEnv(statement, host, env, stepLimit)
		if err != nil {
			return UndefinedValue(), err
		}
		last = value
	}
	return last, nil
}

func splitScriptStatements(source string) ([]string, error) {
	text := strings.TrimSpace(source)
	if text == "" {
		return nil, nil
	}

	statements := make([]string, 0, 4)
	start := 0
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if lineComment {
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && i+1 < len(text) && text[i+1] == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case '`':
			quote = ch
		case '/':
			if i+1 < len(text) {
				switch text[i+1] {
				case '/':
					lineComment = true
					i++
				case '*':
					blockComment = true
					i++
				}
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ';':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				statement := strings.TrimSpace(text[start:i])
				if statement != "" {
					statements = append(statements, statement)
				}
				start = i + 1
			}
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted string in script source")
	}
	if blockComment {
		return nil, fmt.Errorf("unterminated block comment in script source")
	}

	if tail := strings.TrimSpace(text[start:]); tail != "" {
		statements = append(statements, tail)
	}
	return statements, nil
}
