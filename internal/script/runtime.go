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

type HostReferenceResolver interface {
	ResolveHostReference(path string) (Value, error)
}

type DispatchRequest struct {
	Source        string
	Bindings      map[string]Value
	ModuleExports map[string]Value
}

type DispatchResult struct {
	Value Value
}

type Runtime struct {
	config        RuntimeConfig
	host          HostBindings
	globalBindings map[string]Value
}

type classicJSEnvironment struct {
	parent    *classicJSEnvironment
	bindings  map[string]classicJSBinding
	classDefs map[string]*classicJSClassDefinition
}

type classicJSBinding struct {
	value   jsValue
	mutable bool
}

type classicJSClassFieldDefinition struct {
	env              *classicJSEnvironment
	name             string
	init             string
	private          bool
	privateKeyPrefix string
}

type classicJSClassDefinition struct {
	env                 *classicJSEnvironment
	privateFieldPrefix  string
	instanceFields      []classicJSClassFieldDefinition
	superStaticTarget   Value
	superInstanceTarget Value
	hasSuper            bool
}

func newClassicJSEnvironment() *classicJSEnvironment {
	return &classicJSEnvironment{
		bindings:  make(map[string]classicJSBinding),
		classDefs: make(map[string]*classicJSClassDefinition),
	}
}

func (e *classicJSEnvironment) clone() *classicJSEnvironment {
	if e == nil {
		return newClassicJSEnvironment()
	}
	return &classicJSEnvironment{
		parent:    e,
		bindings:  make(map[string]classicJSBinding),
		classDefs: make(map[string]*classicJSClassDefinition),
	}
}

func (e *classicJSEnvironment) cloneDetached() *classicJSEnvironment {
	return e.cloneDetachedWithMapping(make(map[*classicJSEnvironment]*classicJSEnvironment))
}

func (e *classicJSEnvironment) cloneDetachedWithMapping(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSEnvironment {
	if e == nil {
		return newClassicJSEnvironment()
	}
	if cloned, ok := mapping[e]; ok {
		return cloned
	}
	clonedParent := e.parent.cloneDetachedWithMapping(mapping)
	cloned := &classicJSEnvironment{
		parent:    clonedParent,
		bindings:  make(map[string]classicJSBinding, len(e.bindings)),
		classDefs: make(map[string]*classicJSClassDefinition, len(e.classDefs)),
	}
	mapping[e] = cloned
	for name, binding := range e.bindings {
		cloned.bindings[name] = classicJSBinding{
			value:   cloneJSValueDetached(binding.value, mapping),
			mutable: binding.mutable,
		}
	}
	for name, classDef := range e.classDefs {
		cloned.classDefs[name] = classDef.cloneDetached(mapping)
	}
	return cloned
}

func (d *classicJSClassDefinition) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSClassDefinition {
	if d == nil {
		return nil
	}
	cloned := &classicJSClassDefinition{
		privateFieldPrefix:  d.privateFieldPrefix,
		instanceFields:      make([]classicJSClassFieldDefinition, 0, len(d.instanceFields)),
		superStaticTarget:   cloneValueDetached(d.superStaticTarget, mapping),
		superInstanceTarget: cloneValueDetached(d.superInstanceTarget, mapping),
		hasSuper:            d.hasSuper,
	}
	for _, field := range d.instanceFields {
		cloned.instanceFields = append(cloned.instanceFields, field.cloneDetached(mapping))
	}
	if d.env != nil {
		if clonedEnv, ok := mapping[d.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = d.env.cloneDetachedWithMapping(mapping)
		}
	}
	return cloned
}

func (f classicJSClassFieldDefinition) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSClassFieldDefinition {
	cloned := classicJSClassFieldDefinition{
		name:             f.name,
		init:             f.init,
		private:          f.private,
		privateKeyPrefix: f.privateKeyPrefix,
	}
	if f.env != nil {
		if clonedEnv, ok := mapping[f.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = f.env.cloneDetachedWithMapping(mapping)
		}
	}
	return cloned
}

func (d *classicJSClassDefinition) privateFieldKey(name string) string {
	if d == nil || d.privateFieldPrefix == "" {
		return name
	}
	return d.privateFieldPrefix + name
}

func (e *classicJSEnvironment) setClassDefinition(name string, classDef *classicJSClassDefinition) {
	if e == nil {
		return
	}
	if e.classDefs == nil {
		e.classDefs = make(map[string]*classicJSClassDefinition)
	}
	e.classDefs[name] = classDef
}

func (e *classicJSEnvironment) classDefinition(name string) (*classicJSClassDefinition, bool) {
	for current := e; current != nil; current = current.parent {
		if len(current.classDefs) == 0 {
			continue
		}
		classDef, ok := current.classDefs[name]
		if ok {
			return classDef, true
		}
	}
	return nil, false
}

func cloneJSValueDetached(value jsValue, mapping map[*classicJSEnvironment]*classicJSEnvironment) jsValue {
	switch value.kind {
	case jsValueScalar:
		cloned := scalarJSValue(cloneValueDetached(value.value, mapping))
		if value.hasReceiver {
			cloned.receiver = cloneValueDetached(value.receiver, mapping)
			cloned.hasReceiver = true
		}
		return cloned
	case jsValueHostObject:
		return hostObjectJSValue()
	case jsValueBuiltinExpr:
		return builtinExprJSValue()
	case jsValueHostMethod:
		return hostMethodJSValue(value.method)
	case jsValueSuper:
		cloned := superJSValue(cloneValueDetached(value.value, mapping), cloneValueDetached(value.receiver, mapping))
		return cloned
	default:
		return value
	}
}

func cloneValueDetached(value Value, mapping map[*classicJSEnvironment]*classicJSEnvironment) Value {
	switch value.Kind {
	case ValueKindArray:
		if len(value.Array) == 0 {
			return ArrayValue(nil)
		}
		cloned := make([]Value, len(value.Array))
		for i, element := range value.Array {
			cloned[i] = cloneValueDetached(element, mapping)
		}
		return ArrayValue(cloned)
	case ValueKindObject:
		if len(value.Object) == 0 {
			return ObjectValue(nil)
		}
		cloned := make([]ObjectEntry, len(value.Object))
		for i, entry := range value.Object {
			cloned[i] = ObjectEntry{Key: entry.Key, Value: cloneValueDetached(entry.Value, mapping)}
		}
		return ObjectValue(cloned)
	case ValueKindFunction:
		if value.NativeFunction != nil {
			return NativeFunctionValue(value.NativeFunction)
		}
		if value.Function == nil {
			return value
		}
		cloned := FunctionValue(value.Function.cloneDetached(mapping))
		cloned.NativeFunction = value.NativeFunction
		return cloned
	case ValueKindPromise:
		if value.Promise == nil {
			return PromiseValue(UndefinedValue())
		}
		return PromiseValue(cloneValueDetached(*value.Promise, mapping))
	default:
		return value
	}
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

func cloneBindingsMap(bindings map[string]Value) map[string]Value {
	if len(bindings) == 0 {
		return map[string]Value{}
	}
	cloned := make(map[string]Value, len(bindings))
	for name, value := range bindings {
		cloned[name] = cloneValueDetached(value, nil)
	}
	return cloned
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
	return NewRuntimeWithConfigAndBindings(DefaultRuntimeConfig(), host, nil)
}

func NewRuntimeWithConfig(config RuntimeConfig, host HostBindings) *Runtime {
	return NewRuntimeWithConfigAndBindings(config, host, nil)
}

func NewRuntimeWithBindings(host HostBindings, globalBindings map[string]Value) *Runtime {
	return NewRuntimeWithConfigAndBindings(DefaultRuntimeConfig(), host, globalBindings)
}

func NewRuntimeWithConfigAndBindings(config RuntimeConfig, host HostBindings, globalBindings map[string]Value) *Runtime {
	cfg := config
	if cfg.StepLimit <= 0 {
		cfg.StepLimit = DefaultRuntimeConfig().StepLimit
	}
	return &Runtime{
		config:        cfg,
		host:          host,
		globalBindings: cloneBindingsMap(globalBindings),
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

	baseEnv := newClassicJSEnvironment()
	if len(r.globalBindings) > 0 {
		if err := seedClassicJSEnvironment(baseEnv, r.globalBindings); err != nil {
			return DispatchResult{}, err
		}
	}
	env := baseEnv.clone()
	if len(request.Bindings) > 0 {
		if err := seedClassicJSEnvironment(env, request.Bindings); err != nil {
			return DispatchResult{}, err
		}
	}
	var last Value = UndefinedValue()
	for _, statement := range statements {
		result, err := r.dispatchStatement(statement, env, request.ModuleExports)
		if err != nil {
			return DispatchResult{}, err
		}
		last = result.Value
	}
	return DispatchResult{Value: last}, nil
}

func (r *Runtime) dispatchStatement(source string, env *classicJSEnvironment, moduleExports map[string]Value) (DispatchResult, error) {
	if source == "" || source == "noop" {
		return DispatchResult{Value: UndefinedValue()}, nil
	}

	if strings.HasPrefix(strings.TrimSpace(source), "host:") {
		return r.dispatchLegacyStatement(source, env)
	}

	value, err := evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, r.host, env, r.config.StepLimit, true, false, false, nil, nil, moduleExports)
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
	value, err := evalClassicJSProgram(trimmed, r.host, env, r.config.StepLimit, nil)
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Value: value}, nil
}

func seedClassicJSEnvironment(env *classicJSEnvironment, bindings map[string]Value) error {
	if env == nil {
		return NewError(ErrorKindRuntime, "classic-JS environment is unavailable")
	}
	for name, value := range bindings {
		if err := env.declare(name, scalarJSValue(value), false); err != nil {
			return err
		}
	}
	return nil
}

func evalClassicJSProgram(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSProgramWithAllowAwait(source, host, env, stepLimit, true, privateClass)
}

func evalClassicJSProgramWithAllowAwait(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, false, false, nil, privateClass, nil)
}

func evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, privateClass *classicJSClassDefinition, moduleExports map[string]Value) (Value, error) {
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
	for i, statement := range statements {
		value, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(statement, host, env, stepLimit, allowAwait, allowYield, allowReturn, resumeState, privateClass, moduleExports)
		if err != nil {
			if yieldedValue, state, ok := classicJSYieldSignalDetails(err); ok {
				if state == nil && i+1 < len(statements) {
					return UndefinedValue(), NewError(ErrorKindUnsupported, "yield inside a nested block must be the final statement in this bounded classic-JS slice")
				}
				return yieldedValue, err
			}
			if throwValue, ok := classicJSThrowSignalValue(err); ok {
				return UndefinedValue(), NewError(ErrorKindRuntime, ToJSString(throwValue))
			}
			if classicJSBreakSignalValue(err) {
				return UndefinedValue(), NewError(ErrorKindRuntime, "break statement is not within a loop or switch in this bounded classic-JS slice")
			}
			if classicJSContinueSignalValue(err) {
				return UndefinedValue(), NewError(ErrorKindRuntime, "continue statement is not within a loop in this bounded classic-JS slice")
			}
			return UndefinedValue(), err
		}
		last = value
	}
	return last, nil
}

func evalClassicJSProgramWithAllowAwaitAndYield(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, resumeState classicJSResumeState, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, allowYield, false, resumeState, privateClass, nil)
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

func SplitScriptStatementsForRuntime(source string) ([]string, error) {
	return splitScriptStatements(source)
}
