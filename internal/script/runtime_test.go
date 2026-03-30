package script

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"browsertester/internal/dom"
)

type hostCall struct {
	method string
	args   []Value
}

type fakeHost struct {
	values map[string]Value
	errs   map[string]error
	calls  []hostCall
}

func (h *fakeHost) Call(method string, args []Value) (Value, error) {
	copiedArgs := make([]Value, len(args))
	copy(copiedArgs, args)
	h.calls = append(h.calls, hostCall{method: method, args: copiedArgs})
	if err, ok := h.errs[method]; ok {
		return UndefinedValue(), err
	}
	if value, ok := h.values[method]; ok {
		return value, nil
	}
	return UndefinedValue(), errors.New("host method is not configured")
}

type hostReferenceDeleteHost struct {
	calls         []hostCall
	resolvedPaths []string
	deletedPaths  []string
	values        map[string]Value
}

func (h *hostReferenceDeleteHost) Call(method string, args []Value) (Value, error) {
	copiedArgs := make([]Value, len(args))
	copy(copiedArgs, args)
	h.calls = append(h.calls, hostCall{method: method, args: copiedArgs})

	switch method {
	case "echo":
		if len(args) != 1 {
			return UndefinedValue(), fmt.Errorf("echo expects 1 argument")
		}
		return args[0], nil
	default:
		return UndefinedValue(), errors.New("host method is not configured")
	}
}

func (h *hostReferenceDeleteHost) ResolveHostReference(path string) (Value, error) {
	h.resolvedPaths = append(h.resolvedPaths, path)

	switch path {
	case "button.dataset":
		return HostObjectReference(path), nil
	}

	if h.values != nil {
		if value, ok := h.values[path]; ok {
			return value, nil
		}
	}
	return UndefinedValue(), nil
}

func (h *hostReferenceDeleteHost) DeleteHostReference(path string) error {
	h.deletedPaths = append(h.deletedPaths, path)

	switch path {
	case "button.dataset":
		return NewError(ErrorKindUnsupported, "deletion of element.dataset is unsupported in this bounded classic-JS slice")
	case "button.dataset.foo":
		if h.values != nil {
			delete(h.values, path)
		}
		return nil
	default:
		return fmt.Errorf("unexpected delete path %q", path)
	}
}

type echoHost struct {
	calls []hostCall
}

func (h *echoHost) Call(method string, args []Value) (Value, error) {
	copiedArgs := make([]Value, len(args))
	copy(copiedArgs, args)
	h.calls = append(h.calls, hostCall{method: method, args: copiedArgs})

	switch method {
	case "echo":
		if len(args) == 0 {
			return UndefinedValue(), nil
		}
		return args[0], nil
	default:
		return UndefinedValue(), errors.New("host method is not configured")
	}
}

type loopHost struct {
	present map[string]bool
	text    map[string]string
	calls   []hostCall
}

func newLoopHost() *loopHost {
	return &loopHost{
		present: map[string]bool{
			"#step1": true,
			"#step2": true,
		},
		text: map[string]string{
			"#out": "old",
		},
	}
}

func (h *loopHost) Call(method string, args []Value) (Value, error) {
	copiedArgs := make([]Value, len(args))
	copy(copiedArgs, args)
	h.calls = append(h.calls, hostCall{method: method, args: copiedArgs})

	switch method {
	case "querySelector":
		if len(args) != 1 || args[0].Kind != ValueKindString {
			return UndefinedValue(), errors.New("querySelector argument must be a selector string")
		}
		if h.present[args[0].String] {
			return StringValue(args[0].String), nil
		}
		return UndefinedValue(), nil
	case "removeNode":
		if len(args) != 1 || args[0].Kind != ValueKindString {
			return UndefinedValue(), errors.New("removeNode argument must be a selector string")
		}
		delete(h.present, args[0].String)
		return UndefinedValue(), nil
	case "setTextContent":
		if len(args) != 2 || args[0].Kind != ValueKindString || args[1].Kind != ValueKindString {
			return UndefinedValue(), errors.New("setTextContent arguments must be selector and string")
		}
		if h.text == nil {
			h.text = map[string]string{}
		}
		h.text[args[0].String] = args[1].String
		return UndefinedValue(), nil
	default:
		return UndefinedValue(), errors.New("host method is not configured")
	}
}

func TestNewRuntimeUsesDefaultConfig(t *testing.T) {
	runtime := NewRuntime(nil)
	if runtime == nil {
		t.Fatalf("NewRuntime() = nil")
	}

	got := runtime.Config()
	want := DefaultRuntimeConfig()
	if got.StepLimit != want.StepLimit {
		t.Fatalf("Config().StepLimit = %d, want %d", got.StepLimit, want.StepLimit)
	}
}

func TestNewRuntimeWithConfigNormalizesStepLimit(t *testing.T) {
	runtime := NewRuntimeWithConfig(RuntimeConfig{StepLimit: 0}, nil)
	if runtime == nil {
		t.Fatalf("NewRuntimeWithConfig() = nil")
	}

	if got, want := runtime.Config().StepLimit, DefaultRuntimeConfig().StepLimit; got != want {
		t.Fatalf("Config().StepLimit = %d, want %d", got, want)
	}
}

func TestDispatchSupportsNoopAndHostCall(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"version": StringValue("v1"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "noop"})
	if err != nil {
		t.Fatalf("Dispatch(noop) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(noop) kind = %q, want %q", result.Value.Kind, ValueKindUndefined)
	}

	result, err = runtime.Dispatch(DispatchRequest{Source: "host:version"})
	if err != nil {
		t.Fatalf("Dispatch(host:version) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "v1" {
		t.Fatalf("Dispatch(host:version) value = %#v, want string v1", result.Value)
	}
	if len(host.calls) != 1 || host.calls[0].method != "version" {
		t.Fatalf("host calls = %#v, want [\"version\"]", host.calls)
	}
}

func TestDispatchSupportsClassicJSMemberCalls(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"documentCurrentScript": StringValue(`<script id="boot">host.setTextContent("#out", host.documentCurrentScript())</script>`),
			"setTextContent":        UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.setTextContent("#out", host.documentCurrentScript())`})
	if err != nil {
		t.Fatalf("Dispatch(classic JS member calls) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(classic JS member calls) kind = %q, want %q", result.Value.Kind, ValueKindUndefined)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "documentCurrentScript" {
		t.Fatalf("host calls[0].method = %q, want documentCurrentScript", host.calls[0].method)
	}
	if host.calls[1].method != "setTextContent" {
		t.Fatalf("host calls[1].method = %q, want setTextContent", host.calls[1].method)
	}
	if got := host.calls[1].args[1]; got.Kind != ValueKindString || got.String != `<script id="boot">host.setTextContent("#out", host.documentCurrentScript())</script>` {
		t.Fatalf("host calls[1].args[1] = %#v, want nested JS source string", got)
	}
}

func TestDispatchSupportsClassicJSStatementLists(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `debugger; host.setTextContent("#out", "first"); host.setTextContent("#out", "second")`})
	if err != nil {
		t.Fatalf("Dispatch(classic JS statement list) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[1].method != "setTextContent" {
		t.Fatalf("host call methods = %#v, want setTextContent twice", host.calls)
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "first" {
		t.Fatalf("host calls[0].args[1] = %#v, want first", host.calls[0].args[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "second" {
		t.Fatalf("host calls[1].args[1] = %#v, want second", host.calls[1].args[1])
	}
}

func TestDispatchSupportsWithStatementsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { value: "seed", count: 1 }; with (obj) { count++; value = value + "-" + count; } host.echo(obj.value)`})
	if err != nil {
		t.Fatalf("Dispatch(with statements) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "seed-2" {
		t.Fatalf("Dispatch(with statements) result = %#v, want string seed-2", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
}

func TestDispatchRejectsWithStatementTargetsOutsideObjectsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `with (1) {}`})
	if err == nil {
		t.Fatalf("Dispatch(with non-object target) error = nil, want error")
	}
	if scriptErr, ok := err.(Error); !ok {
		t.Fatalf("Dispatch(with non-object target) error type = %T, want script.Error", err)
	} else if scriptErr.Kind != ErrorKindUnsupported && scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(with non-object target) error kind = %q, want unsupported or runtime", scriptErr.Kind)
	}
}

func TestDispatchSupportsTopLevelFunctionDeclarationFollowedByStatement(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := `function parseBooleanLike(value) {
  return value === "yes"
}
const flag = parseBooleanLike("yes");
host.setTextContent("#out", flag ? "yes" : "no")`
	_, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(function declaration + statement) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host call method = %q, want setTextContent", host.calls[0].method)
	}
	if got := host.calls[0].args[1]; got.Kind != ValueKindString || got.String != "yes" {
		t.Fatalf("host call args[1] = %#v, want yes", got)
	}
}

func TestDispatchSupportsTopLevelFunctionDeclarationHoisting(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `const value = get(2); function get(input) { return input + 1; } value`})
	if err != nil {
		t.Fatalf("Dispatch(top-level function hoisting) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 3 {
		t.Fatalf("Dispatch(top-level function hoisting) value = %#v, want number 3", result.Value)
	}
}

func TestDispatchTreatsHoistedTopLevelFunctionDeclarationAsUndefinedStatement(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `const value = get(2); function get(input) { return input + 1; }`})
	if err != nil {
		t.Fatalf("Dispatch(top-level function declaration as trailing statement) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(top-level function declaration as trailing statement) value = %#v, want undefined", result.Value)
	}
}

func TestDispatchSupportsObjectLiteralShorthandPropertiesAndMethodsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let item = { kind: "box" }; let list = [1, 2]; let obj = { item, list, read() { return this.item.kind + "-" + this.list[1] } }; host.echo(obj.item.kind, obj.list[1], obj.read())`})
	if err != nil {
		t.Fatalf("Dispatch(object literal shorthand properties and methods) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object literal shorthand properties and methods) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 3 {
		t.Fatalf("host.calls[0].args len = %d, want 3", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "box" {
		t.Fatalf("host.calls[0].args[0] = %#v, want box", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindNumber || host.calls[0].args[1].Number != 2 {
		t.Fatalf("host.calls[0].args[1] = %#v, want 2", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindString || host.calls[0].args[2].String != "box-2" {
		t.Fatalf("host.calls[0].args[2] = %#v, want box-2", host.calls[0].args[2])
	}
}

func TestDispatchSupportsComputedObjectLiteralPropertiesAndMethodsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let key = "value"; let method = "read"; let value = "seed"; let obj = { [key]: value, [method]() { return this.value } }; host.echo(obj.value, obj.read())`})
	if err != nil {
		t.Fatalf("Dispatch(computed object literal properties and methods) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(computed object literal properties and methods) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "seed" {
		t.Fatalf("host.calls[0].args[0] = %#v, want seed", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "seed" {
		t.Fatalf("host.calls[0].args[1] = %#v, want seed", host.calls[0].args[1])
	}
}

func TestDispatchSupportsObjectLiteralAsyncAndGeneratorMethodsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	source := `let obj = { async: "seed", async read() { return await this.async }, *spin() { yield this.async }, async *drift() { yield await this.async; yield this.async } }; let asyncValue = await obj.read(); let syncValue = obj.spin().next().value; let asyncIt = obj.drift(); let asyncFirst = await asyncIt.next(); let asyncSecond = await asyncIt.next(); let asyncThird = await asyncIt.next(); ` + "`" + "${asyncValue}|${syncValue}|${asyncFirst.value}|${asyncSecond.value}|${asyncThird.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(object literal async and generator methods) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "seed|seed|seed|seed|true" {
		t.Fatalf("Dispatch(object literal async and generator methods) value = %#v, want string seed|seed|seed|seed|true", result.Value)
	}
}

func TestDispatchRejectsMalformedObjectLiteralAsyncGeneratorMethodsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { async *() { yield 1 } }`})
	if err == nil {
		t.Fatalf("Dispatch(malformed object literal async generator methods) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed object literal async generator methods) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed object literal async generator methods) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedObjectLiteralShorthandSequencesInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let value = "seed"; let obj = { value other }`})
	if err == nil {
		t.Fatalf("Dispatch(malformed object literal shorthand sequences) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed object literal shorthand sequences) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed object literal shorthand sequences) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsPrivateNamesInObjectLiteralMethodsAsParseErrorsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { async #read() { return 1 } }`})
	if err == nil {
		t.Fatalf("Dispatch(private names in object literal methods) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(private names in object literal methods) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(private names in object literal methods) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsObjectLiteralGetterAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let label = "score"; let obj = { get value() { return "seed" }, get [label]() { return this.value } }; host.echo(obj.value, obj.score)`})
	if err != nil {
		t.Fatalf("Dispatch(object literal getter accessors) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object literal getter accessors) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "seed" {
		t.Fatalf("host.calls[0].args[0] = %#v, want seed", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "seed" {
		t.Fatalf("host.calls[0].args[1] = %#v, want seed", host.calls[0].args[1])
	}
}

func TestDispatchSupportsObjectLiteralSuperMethodsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let proto = { read() { return "proto" }, label: "base" }; let obj = { __proto__: proto, read() { return super.read() + "-child" }, get label() { return super.label } }; let plain = { read() { return super.toString() } }; ` + "`" + "${obj.read()}|${obj.label}|${plain.read()}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(object literal super methods) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "proto-child|base|[object Object]" {
		t.Fatalf("Dispatch(object literal super methods) value = %#v, want string proto-child|base|[object Object]", result.Value)
	}
}

func TestDispatchSupportsNullPrototypeObjectLiteralSuperAccessInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { __proto__: null, read() { return super.label } }; obj.read()`})
	if err != nil {
		t.Fatalf("Dispatch(null prototype object literal super access) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(null prototype object literal super access) value = %#v, want undefined", result.Value)
	}
}

func TestDispatchSupportsNullPrototypeObjectLiteralSuperAssignmentInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { __proto__: null, value: "seed", write(v) { super.value = v; return this.value }, create(v) { super.extra = v; return this.extra } }; ` + "`" + "${obj.write(\"updated\")}|${obj.create(\"fresh\")}|${obj.value}|${obj.extra}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(null prototype object literal super assignment) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "updated|fresh|updated|fresh" {
		t.Fatalf("Dispatch(null prototype object literal super assignment) value = %#v, want string updated|fresh|updated|fresh", result.Value)
	}
}

func TestDispatchSupportsNullPrototypeObjectLiteralSuperCompoundAssignmentInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { __proto__: null, value: "seed", write(v) { super.value += v; return this.value } }; ` + "`" + "${obj.write(\"-updated\")}|${obj.value}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(null prototype object literal super compound assignment) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "seed-updated|seed-updated" {
		t.Fatalf("Dispatch(null prototype object literal super compound assignment) value = %#v, want string seed-updated|seed-updated", result.Value)
	}
}

func TestDispatchSupportsDeleteExpressionsOnSuperTargetsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `class Base { static value = "seed" }; class Derived extends Base { static value = "derived"; static zap() { return delete super.value }; static read() { return super.value } }; ` + "`" + "${Derived.zap()}|${Derived.read()}|${Base.value}|${Derived.value}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions on super targets) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "true|seed|seed|seed" {
		t.Fatalf("Dispatch(delete expressions on super targets) value = %#v, want string true|seed|seed|seed", result.Value)
	}
}

func TestDispatchSupportsDeleteExpressionsOnNullPrototypeSuperTargetsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { __proto__: null, value: "seed", zap() { return delete super.value }, read() { return this.value } }; ` + "`" + "${obj.zap()}|${obj.read()}|${obj.value}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions on null-prototype super targets) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "true|undefined|undefined" {
		t.Fatalf("Dispatch(delete expressions on null-prototype super targets) value = %#v, want string true|undefined|undefined", result.Value)
	}
}

func TestDispatchRejectsSuperInNonMethodObjectLiteralFunctionsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { read: function() { return super.toString() } }; obj.read()`})
	if err == nil {
		t.Fatalf("Dispatch(super in non-method object literal function) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(super in non-method object literal function) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(super in non-method object literal function) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchReportsRuntimeErrorForSuperCallInNullPrototypeObjectLiteralMethodInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { __proto__: null, read() { return super.toString() } }; obj.read()`})
	if err == nil {
		t.Fatalf("Dispatch(super call in null-prototype object literal method) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(super call in null-prototype object literal method) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(super call in null-prototype object literal method) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsObjectPropertyAssignmentInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { value: "seed" }; obj.value = "updated"; obj.nested = {}; obj.nested.count = 1; obj.nested.count = obj.nested.count + 1; host.echo(obj.value, obj.nested.count)`})
	if err != nil {
		t.Fatalf("Dispatch(object property assignment) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "updated" {
		t.Fatalf("host.calls[0].args[0] = %#v, want updated", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindNumber || host.calls[0].args[1].Number != 2 {
		t.Fatalf("host.calls[0].args[1] = %#v, want 2", host.calls[0].args[1])
	}
}

func TestDispatchSupportsFunctionOwnPropertyAssignmentInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function showToast(message) { showToast._timer = message; } showToast("done"); host.echo(showToast._timer)`})
	if err != nil {
		t.Fatalf("Dispatch(function own property assignment) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", call.method)
	}
	if len(call.args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "done" {
		t.Fatalf("host.calls[0].args[0] = %#v, want done", call.args[0])
	}
}

func TestDispatchSupportsSetConstructorFromNativeIteratorLikeSourceInClassicJS(t *testing.T) {
	index := 0
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"Set": BuiltinSetValue(),
		"source": ObjectValue([]ObjectEntry{
			{
				Key: "next",
				Value: NativeFunctionValue(func(args []Value) (Value, error) {
					switch index {
					case 0:
						index = 1
						return ObjectValue([]ObjectEntry{
							{Key: "value", Value: StringValue("left")},
							{Key: "done", Value: BoolValue(false)},
						}), nil
					case 1:
						index = 2
						return ObjectValue([]ObjectEntry{
							{Key: "value", Value: StringValue("right")},
							{Key: "done", Value: BoolValue(false)},
						}), nil
					default:
						return ObjectValue([]ObjectEntry{
							{Key: "done", Value: BoolValue(true)},
						}), nil
					}
				}),
			},
		}),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `const set = new Set(source); host.echo(set.size, set.has("left"), set.has("right"))`})
	if err != nil {
		t.Fatalf("Dispatch(Set constructor from iterator-like source) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host.calls[0].args len = %d, want 3", len(call.args))
	}
	if call.args[0].Kind != ValueKindNumber || call.args[0].Number != 2 {
		t.Fatalf("host.calls[0].args[0] = %#v, want 2", call.args[0])
	}
	if call.args[1].Kind != ValueKindBool || !call.args[1].Bool {
		t.Fatalf("host.calls[0].args[1] = %#v, want true", call.args[1])
	}
	if call.args[2].Kind != ValueKindBool || !call.args[2].Bool {
		t.Fatalf("host.calls[0].args[2] = %#v, want true", call.args[2])
	}
}

func TestDispatchSupportsArrayPropertyAssignmentInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let arr = [1, 2]; host.echo(arr[0]++, arr[0], ++arr[1], arr[1], arr[2] = 5, arr.length, arr[2]); arr.length = 2; host.echo(arr.length, arr[2])`})
	if err != nil {
		t.Fatalf("Dispatch(array property assignment) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	firstCall := host.calls[0]
	if firstCall.method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", firstCall.method)
	}
	if len(firstCall.args) != 7 {
		t.Fatalf("host.calls[0].args len = %d, want 7", len(firstCall.args))
	}
	wantNumbers := []float64{1, 2, 3, 3, 5, 3}
	for i, want := range wantNumbers {
		if firstCall.args[i].Kind != ValueKindNumber || firstCall.args[i].Number != want {
			t.Fatalf("host.calls[0].args[%d] = %#v, want number %v", i, firstCall.args[i], want)
		}
	}
	if firstCall.args[6].Kind != ValueKindNumber || firstCall.args[6].Number != 5 {
		t.Fatalf("host.calls[0].args[6] = %#v, want 5", firstCall.args[6])
	}
	secondCall := host.calls[1]
	if secondCall.method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", secondCall.method)
	}
	if len(secondCall.args) != 2 {
		t.Fatalf("host.calls[1].args len = %d, want 2", len(secondCall.args))
	}
	if secondCall.args[0].Kind != ValueKindNumber || secondCall.args[0].Number != 2 {
		t.Fatalf("host.calls[1].args[0] = %#v, want 2", secondCall.args[0])
	}
	if secondCall.args[1].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[1].args[1] = %#v, want undefined", secondCall.args[1])
	}
}

func TestDispatchSupportsIncrementAndDecrementExpressionsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let value = 1; let big = 1n; host.echo(value++, value, ++value, value, big++, big, ++big, big)`})
	if err != nil {
		t.Fatalf("Dispatch(increment and decrement expressions) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 1 {
		t.Fatalf("Dispatch(increment and decrement expressions) result = %#v, want first scalar 1", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 8 {
		t.Fatalf("host call args len = %d, want 8", len(call.args))
	}
	wantNumbers := []float64{1, 2, 3, 3}
	for i, want := range wantNumbers {
		if call.args[i].Kind != ValueKindNumber || call.args[i].Number != want {
			t.Fatalf("host call arg[%d] = %#v, want number %v", i, call.args[i], want)
		}
	}
	wantBigInts := []string{"1", "2", "3", "3"}
	for i, want := range wantBigInts {
		arg := call.args[i+4]
		if arg.Kind != ValueKindBigInt || arg.BigInt != want {
			t.Fatalf("host call arg[%d] = %#v, want bigint %s", i+4, arg, want)
		}
	}
}

func TestDispatchSupportsIncrementAndDecrementOnObjectPropertiesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { count: 1 }; host.echo(obj.count++, obj.count, ++obj.count, obj.count, obj["count"]--, obj.count, --obj["count"], obj.count)`})
	if err != nil {
		t.Fatalf("Dispatch(increment and decrement on object properties) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 1 {
		t.Fatalf("Dispatch(increment and decrement on object properties) result = %#v, want first scalar 1", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 8 {
		t.Fatalf("host call args len = %d, want 8", len(call.args))
	}
	wantNumbers := []float64{1, 2, 3, 3, 3, 2, 1, 1}
	for i, want := range wantNumbers {
		if call.args[i].Kind != ValueKindNumber || call.args[i].Number != want {
			t.Fatalf("host call arg[%d] = %#v, want number %v", i, call.args[i], want)
		}
	}
}

func TestDispatchRejectsAssignmentToGetterOnlyObjectPropertiesInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { get value() { return "seed" } }; obj.value = "updated"`})
	if err == nil {
		t.Fatalf("Dispatch(assignment to getter-only object property) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(assignment to getter-only object property) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(assignment to getter-only object property) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsAssignmentToUnsupportedArrayPropertiesInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let arr = [1, 2]; arr["foo"] = 1`})
	if err == nil {
		t.Fatalf("Dispatch(assignment to unsupported array property) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(assignment to unsupported array property) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(assignment to unsupported array property) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsIncrementAndDecrementOnUnsupportedTargetsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `1++`})
	if err == nil {
		t.Fatalf("Dispatch(increment on unsupported target) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(increment on unsupported target) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(increment on unsupported target) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsDeleteExpressionsOnObjectBindingsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { nested: { value: "seed" } }; host.echo(delete obj, delete obj.nested.value, obj.nested.value)`})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions on object bindings) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(delete expressions on object bindings) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 3 {
		t.Fatalf("host.calls[0].args len = %d, want 3", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindBool || host.calls[0].args[0].Bool {
		t.Fatalf("host.calls[0].args[0] = %#v, want bool false", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindBool || !host.calls[0].args[1].Bool {
		t.Fatalf("host.calls[0].args[1] = %#v, want bool true", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[2] = %#v, want undefined", host.calls[0].args[2])
	}
}

func TestDispatchSupportsDeleteExpressionsOnConstObjectBindingsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `const key = "remove"; const payload = { keep: "value", remove: "gone" }; delete payload[key]; [payload.remove, payload.keep].join("|")`})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions on const object bindings) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "|value" {
		t.Fatalf("Dispatch(delete expressions on const object bindings) value = %#v, want \"|value\"", result.Value)
	}
}

func TestDispatchSupportsDeleteExpressionsOnArrayBindingsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let arr = [1, 2, { value: "seed" }]; host.echo(delete arr[1], delete arr[2].value, arr[0], arr[1], arr[2].value, arr.length)`})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions on array bindings) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(delete expressions on array bindings) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 6 {
		t.Fatalf("host.calls[0].args len = %d, want 6", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindBool || !host.calls[0].args[0].Bool {
		t.Fatalf("host.calls[0].args[0] = %#v, want bool true", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindBool || !host.calls[0].args[1].Bool {
		t.Fatalf("host.calls[0].args[1] = %#v, want bool true", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindNumber || host.calls[0].args[2].Number != 1 {
		t.Fatalf("host.calls[0].args[2] = %#v, want 1", host.calls[0].args[2])
	}
	if host.calls[0].args[3].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[3] = %#v, want undefined", host.calls[0].args[3])
	}
	if host.calls[0].args[4].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[4] = %#v, want undefined", host.calls[0].args[4])
	}
	if host.calls[0].args[5].Kind != ValueKindNumber || host.calls[0].args[5].Number != 3 {
		t.Fatalf("host.calls[0].args[5] = %#v, want 3", host.calls[0].args[5])
	}
}

func TestDispatchSupportsDeleteExpressionsOnArrayLengthInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let arr = [1, 2]; host.echo(delete arr.length, arr.length)`})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions on array length) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(delete expressions on array length) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindBool || host.calls[0].args[0].Bool {
		t.Fatalf("host.calls[0].args[0] = %#v, want bool false", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindNumber || host.calls[0].args[1].Number != 2 {
		t.Fatalf("host.calls[0].args[1] = %#v, want 2", host.calls[0].args[1])
	}
}

func TestDispatchSupportsDeleteExpressionsOnStringBindingsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let text = "go"; host.echo(delete text[0], delete text["length"], delete text.foo, text[0], text["length"])`})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions on string bindings) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(delete expressions on string bindings) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 5 {
		t.Fatalf("host.calls[0].args len = %d, want 5", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindBool || host.calls[0].args[0].Bool {
		t.Fatalf("host.calls[0].args[0] = %#v, want bool false", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindBool || host.calls[0].args[1].Bool {
		t.Fatalf("host.calls[0].args[1] = %#v, want bool false", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindBool || !host.calls[0].args[2].Bool {
		t.Fatalf("host.calls[0].args[2] = %#v, want bool true", host.calls[0].args[2])
	}
	if host.calls[0].args[3].Kind != ValueKindString || host.calls[0].args[3].String != "g" {
		t.Fatalf("host.calls[0].args[3] = %#v, want g", host.calls[0].args[3])
	}
	if host.calls[0].args[4].Kind != ValueKindNumber || host.calls[0].args[4].Number != 2 {
		t.Fatalf("host.calls[0].args[4] = %#v, want 2", host.calls[0].args[4])
	}
}

func TestDispatchSupportsDeleteExpressionsWithOptionalChainingOnObjectBindingsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let maybe = null; let obj = { nested: { value: "seed" } }; host.echo(delete maybe?.value, delete obj?.nested?.value, obj.nested.value)`})
	if err != nil {
		t.Fatalf("Dispatch(delete expressions with optional chaining) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(delete expressions with optional chaining) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 3 {
		t.Fatalf("host.calls[0].args len = %d, want 3", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindBool || !host.calls[0].args[0].Bool {
		t.Fatalf("host.calls[0].args[0] = %#v, want bool true", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindBool || !host.calls[0].args[1].Bool {
		t.Fatalf("host.calls[0].args[1] = %#v, want bool true", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[2] = %#v, want undefined", host.calls[0].args[2])
	}
}

func TestDispatchRejectsDeleteExpressionsOnUnsupportedTargetsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `delete host.echo`})
	if err == nil {
		t.Fatalf("Dispatch(delete on unsupported target) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(delete on unsupported target) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(delete on unsupported target) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsDeleteExpressionsOnPrimitiveValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := "let num = 1; let bool = false; let big = 1n; let deletes = `${delete num.foo}|${delete bool.foo}|${delete big.foo}|${delete num?.foo}`; host.echo(deletes)"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(delete on primitive values) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(delete on primitive values) result kind = %q, want %q", result.Value.Kind, ValueKindUndefined)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want 1 call", host.calls)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "true|true|true|true" {
		t.Fatalf("host.calls[0].args[0] = %#v, want primitive delete expressions to return true", host.calls[0].args[0])
	}
}

func TestDispatchSupportsDeleteExpressionsOnArrayPropertiesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "let arr = [1, 2]; let deleted = delete arr.foo; host.echo(`${deleted}|${arr.length}`)"})
	if err != nil {
		t.Fatalf("Dispatch(delete on array property) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(delete on array property) result kind = %q, want %q", result.Value.Kind, ValueKindUndefined)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want 1 call", host.calls)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "true|2" {
		t.Fatalf("host.calls[0].args[0] = %#v, want array delete of unknown property to return true", host.calls[0].args[0])
	}
}

func TestDispatchSupportsDeleteExpressionsOnHostReferencesInClassicJS(t *testing.T) {
	host := &hostReferenceDeleteHost{
		values: map[string]Value{
			"button.dataset.foo": StringValue("seed"),
		},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"button": HostObjectReference("button"),
		},
		Source: "let deleted = delete button.dataset.foo; host.echo(" + "`" + "${deleted}|${button.dataset.foo}" + "`" + ")",
	})
	if err != nil {
		t.Fatalf("Dispatch(delete on host reference) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "true|undefined" {
		t.Fatalf("Dispatch(delete on host reference) result = %#v, want string true|undefined", result.Value)
	}
	if len(host.deletedPaths) != 1 {
		t.Fatalf("host.deletedPaths = %#v, want one delete path", host.deletedPaths)
	}
	if host.deletedPaths[0] != "button.dataset.foo" {
		t.Fatalf("host.deletedPaths[0] = %q, want button.dataset.foo", host.deletedPaths[0])
	}
	if len(host.calls) != 1 {
		t.Fatalf("host.calls = %#v, want one echo call", host.calls)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "true|undefined" {
		t.Fatalf("host.calls[0].args = %#v, want one true|undefined string", host.calls[0].args)
	}
}

func TestSkipHostBindingsPreservesMapState(t *testing.T) {
	state := &classicJSMapState{}
	state.set(StringValue("sku-1"), NumberValue(12))
	original := classicJSMapInstanceValue(state, "")

	host := &hostReferenceDeleteHost{
		values: map[string]Value{
			"map": original,
		},
	}
	skip := skipHostBindings{delegate: host}

	resolved, err := skip.ResolveHostReference("map")
	if err != nil {
		t.Fatalf("skip.ResolveHostReference(map) error = %v", err)
	}
	if resolved.Kind != ValueKindObject {
		t.Fatalf("skip.ResolveHostReference(map) kind = %q, want object", resolved.Kind)
	}
	if resolved.MapState == nil {
		t.Fatalf("skip.ResolveHostReference(map) MapState = nil, want map state")
	}
	if resolved.MapState == state {
		t.Fatalf("skip.ResolveHostReference(map) MapState = original pointer, want detached clone")
	}
	if got := resolved.MapState.size(); got != 1 {
		t.Fatalf("skip.ResolveHostReference(map) size = %d, want 1", got)
	}

	setFn, ok, err := classicJSMapVirtualProperty(resolved, "set")
	if err != nil {
		t.Fatalf("classicJSMapVirtualProperty(set) error = %v", err)
	}
	if !ok {
		t.Fatalf("classicJSMapVirtualProperty(set) ok = false, want true")
	}
	if _, err := InvokeCallableValue(nil, setFn, []Value{StringValue("sku-2"), NumberValue(5)}, UndefinedValue(), false); err != nil {
		t.Fatalf("InvokeCallableValue(map.set) error = %v", err)
	}
	if got := state.size(); got != 1 {
		t.Fatalf("original MapState size = %d, want 1 after skipped clone mutation", got)
	}
	if got := resolved.MapState.size(); got != 2 {
		t.Fatalf("resolved MapState size = %d, want 2 after mutation", got)
	}
}

func TestDispatchRejectsDeleteExpressionsOnHostDatasetSurfaceInClassicJS(t *testing.T) {
	host := &hostReferenceDeleteHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"button": HostObjectReference("button"),
		},
		Source: `delete button.dataset`,
	})
	if err == nil {
		t.Fatalf("Dispatch(delete on host dataset surface) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(delete on host dataset surface) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(delete on host dataset surface) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if len(host.deletedPaths) != 1 || host.deletedPaths[0] != "button.dataset" {
		t.Fatalf("host.deletedPaths = %#v, want one button.dataset delete path", host.deletedPaths)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host.calls = %#v, want no calls after rejected delete", host.calls)
	}
}

func TestDispatchSupportsDeleteExpressionsWithOptionalChainingOnPrimitiveTargetsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "let value = 1; let deleted = delete value?.prop; host.echo(`${deleted}`)"})
	if err != nil {
		t.Fatalf("Dispatch(delete with optional chaining on primitive targets) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(delete with optional chaining on primitive targets) result kind = %q, want %q", result.Value.Kind, ValueKindUndefined)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want 1 call", host.calls)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "true" {
		t.Fatalf("host.calls[0].args[0] = %#v, want optional delete on primitive target to return true", host.calls[0].args[0])
	}
}

func TestDispatchSupportsThrowStatementsWithCatchBindingsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let value = "seed"; try { throw value } catch (error) { host.echo(error) }`})
	if err != nil {
		t.Fatalf("Dispatch(throw statements with catch bindings) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(throw statements with catch bindings) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "seed" {
		t.Fatalf("host.calls[0].args[0] = %#v, want seed", host.calls[0].args[0])
	}
}

func TestDispatchRejectsThrowStatementsWithoutExpressions(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `throw`})
	if err == nil {
		t.Fatalf("Dispatch(throw without expression) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(throw without expression) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(throw without expression) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchReportsUncaughtThrowStatementsAsRuntimeErrors(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `throw "boom"`})
	if err == nil {
		t.Fatalf("Dispatch(uncaught throw) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(uncaught throw) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(uncaught throw) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsObjectLiteralGetterAccessorsWithParametersInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { get value(a) { return a } }`})
	if err == nil {
		t.Fatalf("Dispatch(object literal getter accessors with parameters) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(object literal getter accessors with parameters) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(object literal getter accessors with parameters) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsObjectLiteralSetterAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { _value: "seed", get value() { return this._value }, set value(v) { this._value = v } }; obj.value = "updated"; host.echo(obj.value, obj._value)`})
	if err != nil {
		t.Fatalf("Dispatch(object literal setter accessors) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object literal setter accessors) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "updated" {
		t.Fatalf("host.calls[0].args[0] = %#v, want updated", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "updated" {
		t.Fatalf("host.calls[0].args[1] = %#v, want updated", host.calls[0].args[1])
	}
}

func TestDispatchSupportsObjectLiteralSetterOnlyAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { set value(v) { this._value = v }, _value: "seed" }; obj.value = "updated"; host.echo(obj._value, obj.value)`})
	if err != nil {
		t.Fatalf("Dispatch(object literal setter-only accessors) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object literal setter-only accessors) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "updated" {
		t.Fatalf("host.calls[0].args[0] = %#v, want updated", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[1] = %#v, want undefined", host.calls[0].args[1])
	}
}

func TestDispatchDeleteExpressionsRemoveObjectLiteralSetterAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { _value: "seed", get value() { return this._value }, set value(v) { this._value = v } }; delete obj.value; host.echo(obj.value, obj._value)`})
	if err != nil {
		t.Fatalf("Dispatch(delete object literal setter accessors) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(delete object literal setter accessors) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[0] = %#v, want undefined", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "seed" {
		t.Fatalf("host.calls[0].args[1] = %#v, want seed", host.calls[0].args[1])
	}
}

func TestDispatchRejectsObjectLiteralSetterAccessorsWithMultipleParametersInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { set value(a, b) { } }`})
	if err == nil {
		t.Fatalf("Dispatch(object literal setter accessors with multiple parameters) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(object literal setter accessors with multiple parameters) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(object literal setter accessors with multiple parameters) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsComputedObjectLiteralShorthandWithoutInitializerOrMethod(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let key = "value"; let obj = { [key] }`})
	if err == nil {
		t.Fatalf("Dispatch(computed object literal shorthand without initializer or method) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(computed object literal shorthand without initializer or method) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(computed object literal shorthand without initializer or method) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsNumericSeparatorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(1_000)`})
	if err != nil {
		t.Fatalf("Dispatch(classic JS numeric separators) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(classic JS numeric separators) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindNumber || host.calls[0].args[0].Number != 1000 {
		t.Fatalf("host calls[0].args[0] = %#v, want 1000", host.calls[0].args[0])
	}
}

func TestDispatchSupportsNonDecimalNumericLiteralsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(0x10, 0b1010, 0o77, 0x1_0n, 0b10_10n, 0o7_7n)`})
	if err != nil {
		t.Fatalf("Dispatch(classic JS non-decimal numeric literals) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(classic JS non-decimal numeric literals) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 6 {
		t.Fatalf("host.calls[0].args len = %d, want 6", len(host.calls[0].args))
	}
	wantNumbers := []float64{16, 10, 63}
	for i, want := range wantNumbers {
		if host.calls[0].args[i].Kind != ValueKindNumber || host.calls[0].args[i].Number != want {
			t.Fatalf("host.calls[0].args[%d] = %#v, want %v", i, host.calls[0].args[i], want)
		}
	}
	wantBigInts := []string{"16", "10", "63"}
	for i, want := range wantBigInts {
		arg := host.calls[0].args[i+3]
		if arg.Kind != ValueKindBigInt || arg.BigInt != want {
			t.Fatalf("host.calls[0].args[%d] = %#v, want %s", i+3, arg, want)
		}
	}
}

func TestDispatchSupportsBigIntLiteralsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(-1_234n, !0n)`})
	if err != nil {
		t.Fatalf("Dispatch(classic JS BigInt literals) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(classic JS BigInt literals) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindBigInt || host.calls[0].args[0].BigInt != "-1234" {
		t.Fatalf("host.calls[0].args[0] = %#v, want -1234", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindBool || !host.calls[0].args[1].Bool {
		t.Fatalf("host.calls[0].args[1] = %#v, want true", host.calls[0].args[1])
	}
}

func TestDispatchSupportsUnaryPlusAndMinusOnScalarValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(+"0x10", -"2", +true, -false, -"foo", -1n)`})
	if err != nil {
		t.Fatalf("Dispatch(classic JS unary + and -) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(classic JS unary + and -) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 6 {
		t.Fatalf("host.calls[0].args len = %d, want 6", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindNumber || host.calls[0].args[0].Number != 16 {
		t.Fatalf("host.calls[0].args[0] = %#v, want 16", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindNumber || host.calls[0].args[1].Number != -2 {
		t.Fatalf("host.calls[0].args[1] = %#v, want -2", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindNumber || host.calls[0].args[2].Number != 1 {
		t.Fatalf("host.calls[0].args[2] = %#v, want 1", host.calls[0].args[2])
	}
	if host.calls[0].args[3].Kind != ValueKindNumber || host.calls[0].args[3].Number != 0 {
		t.Fatalf("host.calls[0].args[3] = %#v, want 0", host.calls[0].args[3])
	}
	if host.calls[0].args[4].Kind != ValueKindNumber || !math.IsNaN(host.calls[0].args[4].Number) {
		t.Fatalf("host.calls[0].args[4] = %#v, want NaN", host.calls[0].args[4])
	}
	if host.calls[0].args[5].Kind != ValueKindBigInt || host.calls[0].args[5].BigInt != "-1" {
		t.Fatalf("host.calls[0].args[5] = %#v, want -1", host.calls[0].args[5])
	}
}

func TestDispatchSupportsVoidOperatorInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(void "seed")`})
	if err != nil {
		t.Fatalf("Dispatch(void operator) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(void operator) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[0] = %#v, want undefined", host.calls[0].args[0])
	}
}

func TestDispatchSupportsLogicalNegationAcrossBoundedValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(!{}, ![], !host, !null, !undefined, !"", !0, !1)`})
	if err != nil {
		t.Fatalf("Dispatch(logical negation across bounded values) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 8 {
		t.Fatalf("host.calls[0].args len = %d, want 8", len(host.calls[0].args))
	}
	want := []bool{false, false, false, true, true, true, true, false}
	for i, wantBool := range want {
		if host.calls[0].args[i].Kind != ValueKindBool || host.calls[0].args[i].Bool != wantBool {
			t.Fatalf("host.calls[0].args[%d] = %#v, want bool %v", i, host.calls[0].args[i], wantBool)
		}
	}
}

func TestDispatchRejectsUnaryPlusOnBigIntInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `+1n`})
	if err == nil {
		t.Fatalf("Dispatch(unary plus on BigInt) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(unary plus on BigInt) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(unary plus on BigInt) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsVoidAsDeclarationNameInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let void = 1`})
	if err == nil {
		t.Fatalf("Dispatch(lexical declaration using void) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(lexical declaration using void) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(lexical declaration using void) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsThisExpressionInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(this)`})
	if err != nil {
		t.Fatalf("Dispatch(this expression) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(this expression) value = %#v, want undefined", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindUndefined {
		t.Fatalf("host.calls[0].args[0] = %#v, want undefined", host.calls[0].args[0])
	}
}

func TestDispatchRejectsThisAsDeclarationNameInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let this = 1`})
	if err == nil {
		t.Fatalf("Dispatch(lexical declaration using this) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(lexical declaration using this) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(lexical declaration using this) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsReservedBindingNamesAsParseErrorsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	for _, source := range []string{
		`let { this } = { this: 1 }`,
		`function this() {}`,
		`class this {}`,
		`let debugger = 1`,
	} {
		_, err := runtime.Dispatch(DispatchRequest{Source: source})
		if err == nil {
			t.Fatalf("Dispatch(%s) error = nil, want parse error", source)
		}
		scriptErr, ok := err.(Error)
		if !ok {
			t.Fatalf("Dispatch(%s) error type = %T, want script.Error", source, err)
		}
		if scriptErr.Kind != ErrorKindParse {
			t.Fatalf("Dispatch(%s) error kind = %q, want %q", source, scriptErr.Kind, ErrorKindParse)
		}
	}
}

func TestDispatchRejectsMalformedBigIntLiterals(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(1.0n)`})
	if err == nil {
		t.Fatalf("Dispatch(malformed BigInt literal) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed BigInt literal) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed BigInt literal) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedNonDecimalNumericLiteralsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	for _, source := range []string{`0x`, `0b2`, `0o8`, `0x1.2`, `0b1.1`, `0o7e1`} {
		_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(` + source + `)`})
		if err == nil {
			t.Fatalf("Dispatch(%s) error = nil, want parse error", source)
		}
		scriptErr, ok := err.(Error)
		if !ok {
			t.Fatalf("Dispatch(%s) error type = %T, want script.Error", source, err)
		}
		if scriptErr.Kind != ErrorKindParse {
			t.Fatalf("Dispatch(%s) error kind = %q, want %q", source, scriptErr.Kind, ErrorKindParse)
		}
	}
}

func TestDispatchSupportsLetAndConstBindingsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let target = "#out"; const value = "fresh"; host.setTextContent(target, value)`})
	if err != nil {
		t.Fatalf("Dispatch(let/const bindings) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0].args[0] = %#v, want #out", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "fresh" {
		t.Fatalf("host.calls[0].args[1] = %#v, want fresh", host.calls[0].args[1])
	}
}

func TestDispatchSupportsIfElseBlocksInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let flag = true; if (flag) { host.setTextContent("#out", "then") } else { host.setTextContent("#out", "else") }`})
	if err != nil {
		t.Fatalf("Dispatch(if/else blocks) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0].args[0] = %#v, want #out", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "then" {
		t.Fatalf("host.calls[0].args[1] = %#v, want then", host.calls[0].args[1])
	}
}

func TestDispatchSupportsIfElseIfChainsWithoutBracesInClassicJS(t *testing.T) {
	tests := []struct {
		name     string
		distance int
		want     int
	}{
		{name: "middle branch", distance: 12, want: 40},
		{name: "final else", distance: 25, want: 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &echoHost{}
			runtime := NewRuntime(host)

			result, err := runtime.Dispatch(DispatchRequest{Source: fmt.Sprintf(`function score(distance, margin) {
  let value = 0;
  if (distance <= margin) value = 70;
  else if (distance <= margin * 2) value = 40;
  else value = 10;
  return value;
}
host.echo(score(%d, 10))`, tc.distance)})
			if err != nil {
				t.Fatalf("Dispatch(if/else-if chains without braces) error = %v", err)
			}
			if result.Value.Kind != ValueKindNumber || result.Value.Number != float64(tc.want) {
				t.Fatalf("Dispatch(if/else-if chains without braces) result = %#v, want number %d", result.Value, tc.want)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
		})
	}
}

func TestDispatchRejectsConstWithoutInitializer(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `const value`})
	if err == nil {
		t.Fatalf("Dispatch(const without initializer) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(const without initializer) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(const without initializer) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsBlockBodiedWhileLoopsInClassicJS(t *testing.T) {
	host := newLoopHost()
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let step1 = host.querySelector("#step1"); let step2 = host.querySelector("#step2"); while (step1 ?? step2) { if (step1) { host.setTextContent("#out", "first"); host.removeNode("#step1"); step1 &&= undefined } else { host.setTextContent("#out", "second"); host.removeNode("#step2"); step2 &&= undefined } }`})
	if err != nil {
		t.Fatalf("Dispatch(while loop) error = %v", err)
	}
	if got, want := host.text["#out"], "second"; got != want {
		t.Fatalf("loop host text[#out] = %q, want %q", got, want)
	}
	if len(host.calls) != 6 {
		t.Fatalf("host calls = %#v, want six calls", host.calls)
	}
	if host.calls[0].method != "querySelector" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#step1" {
		t.Fatalf("host.calls[0] = %#v, want querySelector(#step1)", host.calls[0])
	}
	if host.calls[1].method != "querySelector" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#step2" {
		t.Fatalf("host.calls[1] = %#v, want querySelector(#step2)", host.calls[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "first" {
		t.Fatalf("host.calls[2] = %#v, want first iteration text", host.calls[2])
	}
	if host.calls[3].method != "removeNode" || host.calls[3].args[0].Kind != ValueKindString || host.calls[3].args[0].String != "#step1" {
		t.Fatalf("host.calls[3] = %#v, want removeNode(#step1)", host.calls[3])
	}
	if host.calls[4].method != "setTextContent" || host.calls[4].args[1].Kind != ValueKindString || host.calls[4].args[1].String != "second" {
		t.Fatalf("host.calls[4] = %#v, want second iteration text", host.calls[4])
	}
	if host.calls[5].method != "removeNode" || host.calls[5].args[0].Kind != ValueKindString || host.calls[5].args[0].String != "#step2" {
		t.Fatalf("host.calls[5] = %#v, want removeNode(#step2)", host.calls[5])
	}
}

func TestDispatchSupportsBlockBodiedForLoopsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let keepGoing = ` + "`" + `go${` + "`" + `now;later` + "`" + `}` + "`" + `; keepGoing; keepGoing &&= false) { host.setTextContent("#out", keepGoing) }`})
	if err != nil {
		t.Fatalf("Dispatch(for loop) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "gonow;later" {
		t.Fatalf("host.calls[0].args[1] = %#v, want gonow;later", host.calls[0].args[1])
	}
}

func TestDispatchSupportsSingleStatementLoopBodiesInClassicJS(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		want   Value
	}{
		{
			name:   "while",
			source: `let count = 0; while (count < 2) count++; count`,
			want:   NumberValue(2),
		},
		{
			name:   "do while",
			source: `let count = 0; do count++; while (count < 2); count`,
			want:   NumberValue(2),
		},
		{
			name:   "for",
			source: `let out = ""; for (let i = 0; i < 2; i++) out += ` + "`" + `x${` + "`" + `y;z` + "`" + `}` + "`" + `; out`,
			want:   StringValue("xy;zxy;z"),
		},
		{
			name:   "for of",
			source: `let out = ""; for (let value of [` + "`" + `prefix${` + "`" + `b of c` + "`" + `}` + "`" + `]) out += value; out`,
			want:   StringValue("prefixb of c"),
		},
		{
			name:   "for in",
			source: `let out = ""; for (let key in { alpha: 1, beta: 2 }) out += key; out`,
			want:   StringValue("alphabeta"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runtime := NewRuntime(nil)
			got, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.name, err)
			}
			if got.Value.Kind != tc.want.Kind {
				t.Fatalf("Dispatch(%s) result kind = %q, want %q", tc.name, got.Value.Kind, tc.want.Kind)
			}
			switch tc.want.Kind {
			case ValueKindNumber:
				if got.Value.Number != tc.want.Number {
					t.Fatalf("Dispatch(%s) result number = %v, want %v", tc.name, got.Value.Number, tc.want.Number)
				}
			case ValueKindString:
				if got.Value.String != tc.want.String {
					t.Fatalf("Dispatch(%s) result string = %q, want %q", tc.name, got.Value.String, tc.want.String)
				}
			default:
				t.Fatalf("Dispatch(%s) test wants unsupported kind %q", tc.name, tc.want.Kind)
			}
		})
	}
}

func TestDispatchSupportsSingleStatementIfBodiesInClassicJS(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		want   Value
	}{
		{
			name:   "if branch",
			source: `let count = 0; if (count < 1) count++; count`,
			want:   NumberValue(1),
		},
		{
			name:   "else branch",
			source: `let count = 0; if (count > 1) count++; else count += 2; count`,
			want:   NumberValue(2),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runtime := NewRuntime(nil)
			got, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.name, err)
			}
			if got.Value.Kind != tc.want.Kind {
				t.Fatalf("Dispatch(%s) result kind = %q, want %q", tc.name, got.Value.Kind, tc.want.Kind)
			}
			if got.Value.Number != tc.want.Number {
				t.Fatalf("Dispatch(%s) result number = %v, want %v", tc.name, got.Value.Number, tc.want.Number)
			}
		})
	}
}

func TestDispatchSupportsStandaloneBlockStatementsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `{ host.setTextContent("#out", "block"); 1 }`})
	if err != nil {
		t.Fatalf("Dispatch(standalone block statement) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 1 {
		t.Fatalf("Dispatch(standalone block statement) value = %#v, want number 1", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "block" {
		t.Fatalf("host.calls[0].args = %#v, want block", host.calls[0].args)
	}
}

func TestDispatchRejectsUnterminatedStandaloneBlockStatementsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `{ host.setTextContent("#out", "block"); `})
	if err == nil {
		t.Fatalf("Dispatch(unterminated standalone block statement) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(unterminated standalone block statement) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(unterminated standalone block statement) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsForOfLoopsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let [first, second = first] of [[1], [2, 3]]) { host.echo(first, second) }`})
	if err != nil {
		t.Fatalf("Dispatch(for...of loop) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindNumber || host.calls[0].args[0].Number != 1 {
		t.Fatalf("host.calls[0].args[0] = %#v, want number 1", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindNumber || host.calls[0].args[1].Number != 1 {
		t.Fatalf("host.calls[0].args[1] = %#v, want number 1", host.calls[0].args[1])
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 2 {
		t.Fatalf("host.calls[1].args len = %d, want 2", len(host.calls[1].args))
	}
	if host.calls[1].args[0].Kind != ValueKindNumber || host.calls[1].args[0].Number != 2 {
		t.Fatalf("host.calls[1].args[0] = %#v, want number 2", host.calls[1].args[0])
	}
	if host.calls[1].args[1].Kind != ValueKindNumber || host.calls[1].args[1].Number != 3 {
		t.Fatalf("host.calls[1].args[1] = %#v, want number 3", host.calls[1].args[1])
	}
}

func TestDispatchSupportsForOfLoopsOnIteratorLikeObjectsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let values = { index: 0, next() { if (this.index === 0) { this.index = 1; return { value: "left", done: false } }; if (this.index === 1) { this.index = 2; return { value: "right", done: false } }; return { done: true } } }; for (let value of values) { host.echo(value) }`})
	if err != nil {
		t.Fatalf("Dispatch(for...of loop on iterator-like object) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "left" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string left", host.calls[0].args[0])
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 {
		t.Fatalf("host.calls[1].args len = %d, want 1", len(host.calls[1].args))
	}
	if host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "right" {
		t.Fatalf("host.calls[1].args[0] = %#v, want string right", host.calls[1].args[0])
	}
}

func TestDispatchSupportsForOfLoopsOnStringValuesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let value of "go") { host.echo(value) }`})
	if err != nil {
		t.Fatalf("Dispatch(for...of loop on string value) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[1].method != "echo" {
		t.Fatalf("host call methods = %#v, want echo calls", host.calls)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "g" {
		t.Fatalf("host.calls[0].args = %#v, want string g", host.calls[0].args)
	}
	if len(host.calls[1].args) != 1 || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "o" {
		t.Fatalf("host.calls[1].args = %#v, want string o", host.calls[1].args)
	}
}

func TestDispatchSupportsForAwaitOfLoopsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `async function wrap(value) { return value }; for await (let value of [wrap("alpha"), wrap("beta")]) { host.echo(value) }`})
	if err != nil {
		t.Fatalf("Dispatch(for await...of loop) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "alpha" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string alpha", host.calls[0].args[0])
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 {
		t.Fatalf("host.calls[1].args len = %d, want 1", len(host.calls[1].args))
	}
	if host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "beta" {
		t.Fatalf("host.calls[1].args[0] = %#v, want string beta", host.calls[1].args[0])
	}
}

func TestDispatchSupportsForAwaitOfLoopsOnIteratorLikeObjectsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `async function wrap(value) { return value }; let values = { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: wrap("alpha"), done: false } }; if (this.index === 1) { this.index = 2; return { value: wrap("beta"), done: false } }; return { done: true } } }; for await (let value of values) { host.echo(value) }`})
	if err != nil {
		t.Fatalf("Dispatch(for await...of loop on iterator-like object) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "alpha" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string alpha", host.calls[0].args[0])
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 {
		t.Fatalf("host.calls[1].args len = %d, want 1", len(host.calls[1].args))
	}
	if host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "beta" {
		t.Fatalf("host.calls[1].args[0] = %#v, want string beta", host.calls[1].args[0])
	}
}

func TestDispatchSupportsForAwaitOfLoopsOnAsyncIteratorLikeObjectsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `async function run() { let values = { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: "alpha", done: false } }; if (this.index === 1) { this.index = 2; return { value: "beta", done: false } }; return { done: true } } }; for await (let value of values) { host.echo(value) } }; await run()`})
	if err != nil {
		t.Fatalf("Dispatch(for await...of loop on async iterator-like object) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "alpha" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string alpha", host.calls[0].args[0])
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 {
		t.Fatalf("host.calls[1].args len = %d, want 1", len(host.calls[1].args))
	}
	if host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "beta" {
		t.Fatalf("host.calls[1].args[0] = %#v, want string beta", host.calls[1].args[0])
	}
}

func TestDispatchSupportsForAwaitOfLoopsWithArrayBindingPatternsOnAsyncIteratorLikeObjectsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `async function run() { let values = { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: ["alpha", "beta"], done: false } }; return { done: true } } }; for await (let [first, second] of values) { host.echo(first + second) } }; await run()`})
	if err != nil {
		t.Fatalf("Dispatch(for await...of loop with array binding pattern on async iterator-like object) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "alphabeta" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string alphabeta", host.calls[0].args[0])
	}
}

func TestDispatchRejectsMalformedAsyncIteratorLikeObjectsInForAwaitOfLoopsWithBindingPatterns(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `async function run() { let values = { async next() { return "seed" } }; for await (let [value] of values) { value } }; await run()`})
	if err == nil {
		t.Fatalf("Dispatch(for await...of malformed async iterator-like object with binding pattern) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for await...of malformed async iterator-like object with binding pattern) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(for await...of malformed async iterator-like object with binding pattern) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsMalformedAsyncIteratorLikeObjectsInForAwaitOfLoops(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `async function run() { let values = { async next() { return "seed" } }; for await (let value of values) { value } }; await run()`})
	if err == nil {
		t.Fatalf("Dispatch(for await...of malformed async iterator-like object) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for await...of malformed async iterator-like object) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(for await...of malformed async iterator-like object) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsForAwaitOfLoopsOnStringValuesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `async function run() { for await (let value of "go") { host.echo(value) } }; await run()`})
	if err != nil {
		t.Fatalf("Dispatch(for await...of loop on string value) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[1].method != "echo" {
		t.Fatalf("host call methods = %#v, want echo calls", host.calls)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "g" {
		t.Fatalf("host.calls[0].args = %#v, want string g", host.calls[0].args)
	}
	if len(host.calls[1].args) != 1 || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "o" {
		t.Fatalf("host.calls[1].args = %#v, want string o", host.calls[1].args)
	}
}

func TestDispatchSupportsForInLoopsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let key in { alpha: 1, beta: 2 }) { host.echo(key) }`})
	if err != nil {
		t.Fatalf("Dispatch(for...in loop) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "alpha" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string alpha", host.calls[0].args[0])
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 {
		t.Fatalf("host.calls[1].args len = %d, want 1", len(host.calls[1].args))
	}
	if host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "beta" {
		t.Fatalf("host.calls[1].args[0] = %#v, want string beta", host.calls[1].args[0])
	}
}

func TestDispatchSupportsForInLoopsOnStringValuesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let index in "go") { host.echo(index) }`})
	if err != nil {
		t.Fatalf("Dispatch(for...in loop on string value) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[1].method != "echo" {
		t.Fatalf("host call methods = %#v, want echo calls", host.calls)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "0" {
		t.Fatalf("host.calls[0].args = %#v, want string 0", host.calls[0].args)
	}
	if len(host.calls[1].args) != 1 || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "1" {
		t.Fatalf("host.calls[1].args = %#v, want string 1", host.calls[1].args)
	}
}

func TestDispatchSupportsUsingDeclarationsInForHeadersInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (using value = 1; value; ) { host.echo(value); break }`})
	if err != nil {
		t.Fatalf("Dispatch(using declarations in for headers) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindNumber || host.calls[0].args[0].Number != 1 {
		t.Fatalf("host.calls[0].args = %#v, want number 1", host.calls[0].args)
	}
}

func TestDispatchSupportsUsingDeclarationsInForOfHeadersInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (using value of [1, 2]) { host.echo(value); break }`})
	if err != nil {
		t.Fatalf("Dispatch(using declarations in for...of headers) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindNumber || host.calls[0].args[0].Number != 1 {
		t.Fatalf("host.calls[0].args = %#v, want number 1", host.calls[0].args)
	}
}

func TestDispatchSupportsAwaitUsingDeclarationsInForAwaitOfHeadersInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for await (await using value of ["seed"]) { host.echo(value); break }`})
	if err != nil {
		t.Fatalf("Dispatch(await using declarations in for await...of headers) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "seed" {
		t.Fatalf("host.calls[0].args = %#v, want string seed", host.calls[0].args)
	}
}

func TestDispatchRejectsUsingDeclarationsWithInitializersInForOfHeadersInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (using value = 1 of [1]) { value }`})
	if err == nil {
		t.Fatalf("Dispatch(using declarations with initializer in for...of header) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(using declarations with initializer in for...of header) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(using declarations with initializer in for...of header) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsAwaitUsingDeclarationsInForOfHeadersInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (await using value of ["seed"]) { value }`})
	if err == nil {
		t.Fatalf("Dispatch(await using declarations in for...of header) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(await using declarations in for...of header) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(await using declarations in for...of header) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsForOfOverNonArrays(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let value of 1) { value }`})
	if err == nil {
		t.Fatalf("Dispatch(for...of over non-array) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for...of over non-array) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(for...of over non-array) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsMalformedIteratorLikeObjectsInForOfLoops(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let values = { next() { return { value: "seed" } } }; for (let value of values) { value }`})
	if err == nil {
		t.Fatalf("Dispatch(for...of malformed iterator-like object) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for...of malformed iterator-like object) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(for...of malformed iterator-like object) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsMalformedForAwaitOfHeaders(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for await (let value in [1]) { value }`})
	if err == nil {
		t.Fatalf("Dispatch(malformed for await...of header) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed for await...of header) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed for await...of header) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsForInOverNonObjects(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let key in 1) { key }`})
	if err == nil {
		t.Fatalf("Dispatch(for...in over non-object) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for...in over non-object) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(for...in over non-object) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsClassDeclarationsWithStaticBlocksInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { static { host.setTextContent("#out", "static") } }`})
	if err != nil {
		t.Fatalf("Dispatch(class declaration) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "static" {
		t.Fatalf("host.calls[0].args[1] = %#v, want static", host.calls[0].args[1])
	}
}

func TestDispatchSupportsStaticClassFieldsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { static value = host.setTextContent("#out", "field"); static { host.setTextContent("#out", "block") } }`})
	if err != nil {
		t.Fatalf("Dispatch(class static field) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "field" {
		t.Fatalf("host.calls[0] = %#v, want static field initializer call", host.calls[0])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "block" {
		t.Fatalf("host.calls[1] = %#v, want static block call", host.calls[1])
	}
}

func TestDispatchSupportsSuperInClassFieldInitializersInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	script := "class Base { constructor() {} static get label() { return \"base-static\" } get kind() { return \"base-instance\" } }; class Example extends Base { static label = super.label; value = super.kind; constructor() { super(); host.setTextContent(\"#out\", `" + "${Example.label}|${this.value}" + "`); } }; new Example()"
	_, err := runtime.Dispatch(DispatchRequest{Source: script})
	if err != nil {
		t.Fatalf("Dispatch(super in class field initializers) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0] = %#v, want class field initializer call", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "base-static|base-instance" {
		t.Fatalf("host.calls[0].args[1] = %#v, want base-static|base-instance", host.calls[0].args[1])
	}
}

func TestDispatchSupportsSuperInBaseClassFieldInitializersInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	script := "class Example { static label = super.label; value = super.kind; constructor() { host.setTextContent(\"#out\", `" + "${Example.label}|${this.value}" + "`); } }; new Example()"
	_, err := runtime.Dispatch(DispatchRequest{Source: script})
	if err != nil {
		t.Fatalf("Dispatch(super in base class field initializers) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0] = %#v, want base class field initializer call", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "undefined|undefined" {
		t.Fatalf("host.calls[0].args[1] = %#v, want undefined|undefined", host.calls[0].args[1])
	}
}

func TestDispatchReportsRuntimeErrorForSuperCallInBaseClassConstructorInClassicJS(t *testing.T) {
	runtime := NewRuntime(&fakeHost{})

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { constructor() { super() } }; new Example()`})
	if err == nil {
		t.Fatalf("Dispatch(super call in base class constructor) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(super call in base class constructor) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(super call in base class constructor) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if !strings.Contains(scriptErr.Error(), "requires a constructor on the base target") {
		t.Fatalf("Dispatch(super call in base class constructor) error = %q, want base-target message", scriptErr.Error())
	}
}

func TestDispatchSupportsClassMethodsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { static value = "field"; static writeStatic() { host.setTextContent("#out", Example.value) } writeInstance() { host.setTextContent("#side", "instance") } }; Example.writeStatic(); Example.prototype.writeInstance()`})
	if err != nil {
		t.Fatalf("Dispatch(class methods) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0] = %#v, want static method call", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "field" {
		t.Fatalf("host.calls[0].args[1] = %#v, want class static field value", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#side" {
		t.Fatalf("host.calls[1] = %#v, want instance method call", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "instance" {
		t.Fatalf("host.calls[1].args[1] = %#v, want instance", host.calls[1].args[1])
	}
}

func TestDispatchSupportsStaticPrototypeMembersInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: "class Example { static prototype = \"special\"; writeInstance() { return \"instance\" } }; let example = new Example(); `" + "${Example.prototype}|${example.writeInstance()}|${example instanceof Example}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(static prototype members) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "special|instance|true" {
		t.Fatalf("Dispatch(static prototype members) result = %#v, want string special|instance|true", result.Value)
	}
}

func TestDispatchSupportsStaticPrototypeSetterMembersInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "class Example { static set prototype(value) { host.setTextContent(\"#out\", value) } }; Example.prototype = \"special\"; `" + "${Example.prototype}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(static prototype setter members) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "undefined" {
		t.Fatalf("Dispatch(static prototype setter members) result = %#v, want string undefined", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0].args[0] = %#v, want #out", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "special" {
		t.Fatalf("host.calls[0].args[1] = %#v, want special", host.calls[0].args[1])
	}
}

func TestDispatchSupportsClassExpressionsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Base { static read() { return "base" } }; let Derived = class extends Base { static read() { return super.read() + "-expr" } }; host.echo(Derived.read())`})
	if err != nil {
		t.Fatalf("Dispatch(class expressions) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "base-expr" {
		t.Fatalf("host.calls[0].args[0] = %#v, want base-expr", host.calls[0].args[0])
	}
}

func TestDispatchSupportsExtendsNullInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: "class Example extends null { static read() { return \"ok\" } }; let example = new Example(); `" + "${example instanceof Example}|${Example.read()}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(extends null) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "true|ok" {
		t.Fatalf("Dispatch(extends null) result = %#v, want string true|ok", result.Value)
	}
}

func TestDispatchSupportsClassInheritanceFromClassExpressionValueInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function makeBase() { return class { static read() { return "base" } } }; class Derived extends makeBase() { static read() { return super.read() + "-expr" } }; host.echo(Derived.read())`})
	if err != nil {
		t.Fatalf("Dispatch(class inheritance from class expression value) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "base-expr" {
		t.Fatalf("host.calls[0].args[0] = %#v, want base-expr", host.calls[0].args[0])
	}
}

func TestDispatchSupportsNewOnClassExpressionsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let Named = class { constructor() { host.setTextContent("#named", "named") } }; new Named(); new (class { constructor() { host.setTextContent("#anon", "anon") } })()`})
	if err != nil {
		t.Fatalf("Dispatch(new on class expressions) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#named" {
		t.Fatalf("host.calls[0] = %#v, want named constructor call", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "named" {
		t.Fatalf("host.calls[0].args[1] = %#v, want named", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#anon" {
		t.Fatalf("host.calls[1] = %#v, want anonymous constructor call", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "anon" {
		t.Fatalf("host.calls[1].args[1] = %#v, want anon", host.calls[1].args[1])
	}
}

func TestDispatchRejectsNonClassExpressionInExtendsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function makeBase() { return {}; } class Derived extends makeBase() {}`})
	if err == nil {
		t.Fatalf("Dispatch(non-class expression in extends) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(non-class expression in extends) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(non-class expression in extends) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsClassExtendsConstructibleFunctionInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function Base(value = "base") { this.seed = value }; class Derived extends Base {}; host.echo(new Derived("seed").seed)`})
	if err != nil {
		t.Fatalf("Dispatch(class extends constructible function) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(class extends constructible function) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 1 || call.args[0].Kind != ValueKindString || call.args[0].String != "seed" {
		t.Fatalf("host call args = %#v, want seed", call.args)
	}
}

func TestDispatchSupportsConstructibleFunctionPrototypeAccessInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "function Base() {}; function Other() {}; class Derived extends Base { static read() { return `" + "${typeof super.prototype}|${typeof Base.prototype}|${new Base() instanceof Base}|${new Base() instanceof Other}" + "` } }; host.echo(Derived.read())"})
	if err != nil {
		t.Fatalf("Dispatch(constructible function prototype access) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(constructible function prototype access) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 1 || call.args[0].Kind != ValueKindString || call.args[0].String != "object|object|true|false" {
		t.Fatalf("host call args = %#v, want object|object|true|false", call.args)
	}
}

func TestDispatchSupportsNamedGeneratorFunctionPrototypeAccessInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "function Base() {}; host.echo((function* Base() { yield `" + "${Base.prototype === undefined}" + "` })().next().value)"})
	if err != nil {
		t.Fatalf("Dispatch(named generator function prototype access) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(named generator function prototype access) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 1 || call.args[0].Kind != ValueKindString || call.args[0].String != "true" {
		t.Fatalf("host call args = %#v, want true", call.args)
	}
}

func TestDispatchRejectsMalformedClassExpressionsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	for _, tc := range []struct {
		name   string
		source string
	}{
		{name: "unterminated", source: `let Example = class { static read() { return "ok" }`},
		{name: "instance-member-sequence", source: `class Example { foo bar }`},
		{name: "static-member-sequence", source: `class Example { static foo bar }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err == nil {
				t.Fatalf("Dispatch(%s) error = nil, want parse error", tc.name)
			}
			scriptErr, ok := err.(Error)
			if !ok {
				t.Fatalf("Dispatch(%s) error type = %T, want script.Error", tc.name, err)
			}
			if scriptErr.Kind != ErrorKindParse {
				t.Fatalf("Dispatch(%s) error kind = %q, want %q", tc.name, scriptErr.Kind, ErrorKindParse)
			}
		})
	}
}

func TestDispatchSupportsClassGetterAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let label = "kind"; class Example { static prefix = "static"; static get [label]() { return this.prefix } get = "plain"; value = "instance"; get read() { return this.value } }; let example = new Example(); host.echo(Example.kind, example.read, example.get)`})
	if err != nil {
		t.Fatalf("Dispatch(class getter accessors) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 3 {
		t.Fatalf("host.calls[0].args len = %d, want 3", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "static" {
		t.Fatalf("host.calls[0].args[0] = %#v, want static", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "instance" {
		t.Fatalf("host.calls[0].args[1] = %#v, want instance", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindString || host.calls[0].args[2].String != "plain" {
		t.Fatalf("host.calls[0].args[2] = %#v, want plain", host.calls[0].args[2])
	}
}

func TestDispatchSupportsPrivateClassGetterAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { static #prefix = "static"; static get #kind() { return this.#prefix } static revealed = Example.#kind; #value = "instance"; get #read() { return this.#value } snapshot = this.#read }; let example = new Example(); host.echo(Example.revealed, example.snapshot)`})
	if err != nil {
		t.Fatalf("Dispatch(private class getter accessors) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "static" {
		t.Fatalf("host.calls[0].args[0] = %#v, want static", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "instance" {
		t.Fatalf("host.calls[0].args[1] = %#v, want instance", host.calls[0].args[1])
	}
}

func TestDispatchSupportsClassSetterAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "class Example { static _value = \"static-seed\"; static get value() { return this._value } static set value(next) { this._value = next } _value = \"instance-seed\"; get value() { return this._value } set value(next) { this._value = next } }; Example.value = \"static-updated\"; let example = new Example(); example.value = \"instance-updated\"; host.setTextContent(\"#out\", `" + "${Example.value}|${example.value}" + "`)"})
	if err != nil {
		t.Fatalf("Dispatch(class setter accessors) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0] = %#v, want final output write", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "static-updated|instance-updated" {
		t.Fatalf("host.calls[0].args[1] = %#v, want static-updated|instance-updated", host.calls[0].args[1])
	}
}

func TestDispatchSupportsPrivateClassSetterAccessorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { static #prefix = "static-seed"; static get #kind() { return this.#prefix } static set #kind(next) { this.#prefix = next } static update(next) { this.#kind = next } static reveal() { return this.#kind } #value = "instance-seed"; get #read() { return this.#value } set #read(next) { this.#value = next } update(next) { this.#read = next } reveal() { return this.#read } }; Example.update("static-updated"); let example = new Example(); example.update("instance-updated"); host.echo(Example.reveal(), example.reveal())`})
	if err != nil {
		t.Fatalf("Dispatch(private class setter accessors) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "static-updated" {
		t.Fatalf("host.calls[0].args[0] = %#v, want static-updated", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "instance-updated" {
		t.Fatalf("host.calls[0].args[1] = %#v, want instance-updated", host.calls[0].args[1])
	}
}

func TestDispatchRejectsClassSetterAccessorsWithMultipleParametersInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { set value(next, extra) {} }`})
	if err == nil {
		t.Fatalf("Dispatch(class setter accessors with multiple parameters) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(class setter accessors with multiple parameters) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(class setter accessors with multiple parameters) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsSuperPropertyAccessInClassMethods(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: "class Base { static kind = \"base\"; greet() { return \"base\" } static label() { return \"label\" } }; class Derived extends Base { static seen = super.kind; static describe() { return `" + "${super[\"kind\"]}-${super[\"label\"]()}" + "` } read() { return `" + "${super.greet()}-${Derived.seen}" + "` } }; let instance = new Derived(); `" + "${Derived.seen}|${Derived.describe()}|${instance.read()}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(super property access in class methods) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "base|base-label|base-base" {
		t.Fatalf("Dispatch(super property access in class methods) result = %#v, want string base|base-label|base-base", result.Value)
	}
}

func TestDispatchSupportsSuperPropertyAssignmentInClassMethods(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: "class Base { static label = \"base\" }; class Derived extends Base { static label = \"derived\"; kind = \"initial\"; static update() { super.label = \"static-updated\"; return Derived.label } write() { super.kind = \"instance-updated\"; return this.kind } }; let instance = new Derived(); Derived.update(); instance.write(); `${Derived.label}|${Base.label}|${instance.kind}`"})
	if err != nil {
		t.Fatalf("Dispatch(super property assignment in class methods) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "static-updated|base|instance-updated" {
		t.Fatalf("Dispatch(super property assignment in class methods) result = %#v, want string static-updated|base|instance-updated", result.Value)
	}
}

func TestDispatchSupportsSuperPropertyAssignmentWhenReceiverPropertyIsMissingInClassMethods(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: "class Base { static label = \"base\" }; class Derived extends Base { static update() { super.label = \"static-updated\" } write() { super.kind = \"instance-updated\" } }; let instance = new Derived(); Derived.update(); instance.write(); `" + "${Derived.label}|${Base.label}|${instance.kind}" + "`"})
	if err != nil {
		t.Fatalf("Dispatch(super property assignment on missing receiver property) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "static-updated|base|instance-updated" {
		t.Fatalf("Dispatch(super property assignment on missing receiver property) result = %#v, want string static-updated|base|instance-updated", result.Value)
	}
}

func TestDispatchRejectsSuperPropertyAssignmentToGetterOnlyBasePropertyInClassMethods(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Base { get kind() { return "base" } }; class Derived extends Base { write() { super.kind = "instance-updated" } }; new Derived().write()`})
	if err == nil {
		t.Fatalf("Dispatch(super property assignment to getter-only base property) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(super property assignment to getter-only base property) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(super property assignment to getter-only base property) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsSuperCallsInClassConstructors(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Base { constructor(value = "base") { host.setTextContent("#base", value) } }; class Derived extends Base { constructor() { super("seed"); host.setTextContent("#derived", "done") } }; new Derived()`})
	if err != nil {
		t.Fatalf("Dispatch(super call in class constructor) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#base" {
		t.Fatalf("host.calls[0] = %#v, want base constructor side effect", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "seed" {
		t.Fatalf("host.calls[0].args[1] = %#v, want seed", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#derived" {
		t.Fatalf("host.calls[1] = %#v, want derived constructor side effect", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "done" {
		t.Fatalf("host.calls[1].args[1] = %#v, want done", host.calls[1].args[1])
	}
}

func TestDispatchSupportsClassInstanceFieldsAndNewInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { value = "field"; constructor() { host.setTextContent("#tail", "ctor") } write() { host.setTextContent("#side", "method") } }; let instance = new Example(); host.setTextContent("#out", instance.value); instance.write()`})
	if err != nil {
		t.Fatalf("Dispatch(class instance fields and new) error = %v", err)
	}
	if len(host.calls) != 3 {
		t.Fatalf("host calls = %#v, want three calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#tail" {
		t.Fatalf("host.calls[0] = %#v, want constructor side effect", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "ctor" {
		t.Fatalf("host.calls[0].args[1] = %#v, want ctor", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#out" {
		t.Fatalf("host.calls[1] = %#v, want instance field read", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "field" {
		t.Fatalf("host.calls[1].args[1] = %#v, want field", host.calls[1].args[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[0].Kind != ValueKindString || host.calls[2].args[0].String != "#side" {
		t.Fatalf("host.calls[2] = %#v, want instance method call", host.calls[2])
	}
	if host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "method" {
		t.Fatalf("host.calls[2].args[1] = %#v, want method", host.calls[2].args[1])
	}
}

func TestDispatchSupportsConstructibleFunctionConstructorsAndInstanceofInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function Plain(value) { this.value = value }; class Box { constructor(value) { this.value = value } }; let plain = new Plain("seed"); let box = new Box("class"); host.echo(plain.value, plain instanceof Plain, box.value, box instanceof Box)`})
	if err != nil {
		t.Fatalf("Dispatch(constructible function constructors and instanceof) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "seed" {
		t.Fatalf("Dispatch(constructible function constructors and instanceof) result = %#v, want string seed", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "seed" {
		t.Fatalf("host call arg[0] = %#v, want string seed", call.args[0])
	}
	if call.args[1].Kind != ValueKindBool || !call.args[1].Bool {
		t.Fatalf("host call arg[1] = %#v, want bool true", call.args[1])
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != "class" {
		t.Fatalf("host call arg[2] = %#v, want string class", call.args[2])
	}
	if call.args[3].Kind != ValueKindBool || !call.args[3].Bool {
		t.Fatalf("host call arg[3] = %#v, want bool true", call.args[3])
	}
}

func TestDispatchRejectsNewOnNonConstructibleFunctionsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	cases := []struct {
		name   string
		source string
	}{
		{name: "arrow", source: `new (() => {})()`},
		{name: "async", source: `new (async function () {})()`},
		{name: "generator", source: `new (function* () {})()`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err == nil {
				t.Fatalf("Dispatch(new on %s function) error = nil, want unsupported error", tc.name)
			}
			scriptErr, ok := err.(Error)
			if !ok {
				t.Fatalf("Dispatch(new on %s function) error type = %T, want script.Error", tc.name, err)
			}
			if scriptErr.Kind != ErrorKindUnsupported {
				t.Fatalf("Dispatch(new on %s function) error kind = %q, want %q", tc.name, scriptErr.Kind, ErrorKindUnsupported)
			}
		})
	}
}

func TestDispatchSupportsComputedClassMembersInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "let fieldName = \"value\"; let staticName = \"staticValue\"; let methodName = \"write\"; class Example { [fieldName] = \"field\"; static [staticName] = \"static\"; [methodName]() { host.setTextContent(\"#side\", \"method\") } }; let instance = new Example(); host.setTextContent(\"#out\", `${Example.staticValue}-${instance.value}`); instance[methodName]()"})
	if err != nil {
		t.Fatalf("Dispatch(computed class members) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0] = %#v, want computed field read", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "static-field" {
		t.Fatalf("host.calls[0].args[1] = %#v, want static-field", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#side" {
		t.Fatalf("host.calls[1] = %#v, want computed method call", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "method" {
		t.Fatalf("host.calls[1].args[1] = %#v, want method", host.calls[1].args[1])
	}
}

func TestDispatchSupportsSuperInComputedClassMemberNamesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Base { static get staticKey() { return "static-name" } get instanceKey() { return "instance-name" } }; class Example extends Base { static [super.staticKey] = "static"; [super.instanceKey] = "instance" }; let example = new Example(); host.echo(Example["static-name"], example["instance-name"])`})
	if err != nil {
		t.Fatalf("Dispatch(super in computed class member names) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "static" {
		t.Fatalf("host.calls[0].args[0] = %#v, want static", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "instance" {
		t.Fatalf("host.calls[0].args[1] = %#v, want instance", host.calls[0].args[1])
	}
}

func TestDispatchSupportsSuperInComputedClassMemberNamesWithoutSuperclassInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { static [super.name] = "value" } ; host.echo(Example.undefined)`})
	if err != nil {
		t.Fatalf("Dispatch(super in base class computed member name) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "value" {
		t.Fatalf("host.calls[0].args[0] = %#v, want value", host.calls[0].args[0])
	}
}

func TestDispatchSupportsBlockBodiedDoWhileLoopsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `do { host.setTextContent("#out", "ran") } while (false)`})
	if err != nil {
		t.Fatalf("Dispatch(do/while loop) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "ran" {
		t.Fatalf("host.calls[0].args = %#v, want ran", host.calls[0].args)
	}
}

func TestDispatchSupportsSwitchStatementsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `switch ("b") { case "a": host.setTextContent("#out", "a"); break; case "b": host.setTextContent("#out", "b"); case "c": host.setTextContent("#out", "c"); break; default: host.setTextContent("#out", "default") }`})
	if err != nil {
		t.Fatalf("Dispatch(switch statement) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "b" {
		t.Fatalf("host.calls[0] = %#v, want first matched case", host.calls[0])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "c" {
		t.Fatalf("host.calls[1] = %#v, want fallthrough case", host.calls[1])
	}
}

func TestDispatchSupportsTryCatchFinallyStatementsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{
			"fail": errors.New("boom"),
		},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `try { host.fail() } catch (err) { host.setTextContent("#out", err) } finally { host.setTextContent("#side", "finally") }`})
	if err != nil {
		t.Fatalf("Dispatch(try/catch/finally) error = %v", err)
	}
	if len(host.calls) != 3 {
		t.Fatalf("host calls = %#v, want three calls", host.calls)
	}
	if host.calls[0].method != "fail" {
		t.Fatalf("host.calls[0].method = %q, want fail", host.calls[0].method)
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#out" {
		t.Fatalf("host.calls[1] = %#v, want catch handler write to #out", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "host: boom" {
		t.Fatalf("host.calls[1].args[1] = %#v, want stringified host error", host.calls[1].args[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[0].Kind != ValueKindString || host.calls[2].args[0].String != "#side" {
		t.Fatalf("host.calls[2] = %#v, want finally handler write to #side", host.calls[2])
	}
	if host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "finally" {
		t.Fatalf("host.calls[2].args[1] = %#v, want finally", host.calls[2].args[1])
	}
}

func TestDispatchSupportsCatchBindingPatternsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `try { throw {kind: "box", count: 2} } catch ({kind, count}) { host.echo(kind, count) }`})
	if err != nil {
		t.Fatalf("Dispatch(catch binding patterns) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "box" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string box", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindNumber || host.calls[0].args[1].Number != 2 {
		t.Fatalf("host.calls[0].args[1] = %#v, want number 2", host.calls[0].args[1])
	}
}

func TestDispatchSupportsOptionalCatchBindingInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `try { throw "seed" } catch { host.echo("caught") }`})
	if err != nil {
		t.Fatalf("Dispatch(optional catch binding) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "caught" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string caught", host.calls[0].args[0])
	}
}

func TestDispatchRejectsMalformedCatchBindingPatternsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `try { throw {kind: "box"} } catch ({kind:}) { kind }`})
	if err == nil {
		t.Fatalf("Dispatch(malformed catch binding pattern) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed catch binding pattern) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed catch binding pattern) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsBreakAndContinueAcrossLoopSwitchAndTryInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let run = true; let first = true; let branch = true; while (run ?? false) { try { switch (branch) { case true: first &&= undefined; branch &&= false; host.setTextContent("#out", "first"); continue; case false: host.setTextContent("#side", "second"); break }; break } finally { host.setTextContent("#tail", "finally") } }`})
	if err != nil {
		t.Fatalf("Dispatch(break and continue across loop/switch/try) error = %v", err)
	}
	if len(host.calls) != 4 {
		t.Fatalf("host calls = %#v, want four calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0] = %#v, want first branch write", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "first" {
		t.Fatalf("host.calls[0].args[1] = %#v, want first", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#tail" {
		t.Fatalf("host.calls[1] = %#v, want finally after continue", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "finally" {
		t.Fatalf("host.calls[1].args[1] = %#v, want finally", host.calls[1].args[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[0].Kind != ValueKindString || host.calls[2].args[0].String != "#side" {
		t.Fatalf("host.calls[2] = %#v, want second branch write", host.calls[2])
	}
	if host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "second" {
		t.Fatalf("host.calls[2].args[1] = %#v, want second", host.calls[2].args[1])
	}
	if host.calls[3].method != "setTextContent" || host.calls[3].args[0].Kind != ValueKindString || host.calls[3].args[0].String != "#tail" {
		t.Fatalf("host.calls[3] = %#v, want finally after break", host.calls[3])
	}
	if host.calls[3].args[1].Kind != ValueKindString || host.calls[3].args[1].String != "finally" {
		t.Fatalf("host.calls[3].args[1] = %#v, want finally", host.calls[3].args[1])
	}
}

func TestDispatchSupportsLabeledBreakAndContinueAcrossLoopSwitchAndTryInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let run = true; let first = true; outer: while (run ?? false) { try { switch (first) { case true: first &&= false; host.setTextContent("#out", "first"); continue outer; case false: host.setTextContent("#side", "second"); break outer } } finally { host.setTextContent("#tail", "finally") } }`})
	if err != nil {
		t.Fatalf("Dispatch(labeled break and continue across loop/switch/try) error = %v", err)
	}
	if len(host.calls) != 4 {
		t.Fatalf("host calls = %#v, want four calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#out" {
		t.Fatalf("host.calls[0] = %#v, want first branch write", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "first" {
		t.Fatalf("host.calls[0].args[1] = %#v, want first", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#tail" {
		t.Fatalf("host.calls[1] = %#v, want finally after continue", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "finally" {
		t.Fatalf("host.calls[1].args[1] = %#v, want finally", host.calls[1].args[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[0].Kind != ValueKindString || host.calls[2].args[0].String != "#side" {
		t.Fatalf("host.calls[2] = %#v, want second branch write", host.calls[2])
	}
	if host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "second" {
		t.Fatalf("host.calls[2].args[1] = %#v, want second", host.calls[2].args[1])
	}
	if host.calls[3].method != "setTextContent" || host.calls[3].args[0].Kind != ValueKindString || host.calls[3].args[0].String != "#tail" {
		t.Fatalf("host.calls[3] = %#v, want finally after break", host.calls[3])
	}
	if host.calls[3].args[1].Kind != ValueKindString || host.calls[3].args[1].String != "finally" {
		t.Fatalf("host.calls[3].args[1] = %#v, want finally", host.calls[3].args[1])
	}
}

func TestDispatchReportsUncaughtBreakAndContinueAsParseErrorsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	for _, source := range []string{`break`, `continue`} {
		_, err := runtime.Dispatch(DispatchRequest{Source: source})
		if err == nil {
			t.Fatalf("Dispatch(%s) error = nil, want parse error", source)
		}
		scriptErr, ok := err.(Error)
		if !ok {
			t.Fatalf("Dispatch(%s) error type = %T, want script.Error", source, err)
		}
		if scriptErr.Kind != ErrorKindParse {
			t.Fatalf("Dispatch(%s) error kind = %q, want %q", source, scriptErr.Kind, ErrorKindParse)
		}
	}
}

func TestDispatchRejectsContinueTargetingLabeledSwitchInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `outer: switch (true) { case true: continue outer }`})
	if err == nil {
		t.Fatalf("Dispatch(continue targeting labeled switch) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(continue targeting labeled switch) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(continue targeting labeled switch) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsContinueTargetingLabeledTryInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `outer: try { continue outer } finally { }`})
	if err == nil {
		t.Fatalf("Dispatch(continue targeting labeled try) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(continue targeting labeled try) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(continue targeting labeled try) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsWhileLoopWhenStepLimitIsExceeded(t *testing.T) {
	host := newLoopHost()
	runtime := NewRuntimeWithConfig(RuntimeConfig{StepLimit: 1}, host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let step1 = host.querySelector("#step1"); let step2 = host.querySelector("#step2"); while (step1 ?? step2) { if (step1) { host.setTextContent("#out", "first"); host.removeNode("#step1"); step1 &&= undefined } else { host.setTextContent("#out", "second"); host.removeNode("#step2"); step2 &&= undefined } }`})
	if err == nil {
		t.Fatalf("Dispatch(while loop step limit) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(while loop step limit) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(while loop step limit) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if got, want := host.text["#out"], "first"; got != want {
		t.Fatalf("loop host text[#out] after step limit = %q, want %q", got, want)
	}
	if len(host.calls) != 4 {
		t.Fatalf("host calls = %#v, want four calls before step limit failure", host.calls)
	}
}

func TestDispatchRejectsForLoopWhenStepLimitIsExceeded(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntimeWithConfig(RuntimeConfig{StepLimit: 1}, host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (;;){ host.setTextContent("#out", "tick") }`})
	if err == nil {
		t.Fatalf("Dispatch(for loop step limit) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for loop step limit) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(for loop step limit) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call before step limit failure", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "tick" {
		t.Fatalf("host.calls[0].args[1] = %#v, want tick", host.calls[0].args[1])
	}
}

func TestDispatchRejectsMalformedSwitchStatements(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `switch ("x") { default: host.setTextContent("#out", "one"); default: host.setTextContent("#out", "two") }`})
	if err == nil {
		t.Fatalf("Dispatch(malformed switch statement) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed switch statement) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed switch statement) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedForStatements(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let keepGoing = true; keepGoing) { host.setTextContent("#out", "one") }`})
	if err == nil {
		t.Fatalf("Dispatch(malformed for statement) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed for statement) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed for statement) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsClassInheritanceInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "class Base { value = \"base\"; static kind = \"base\"; constructor() { host.setTextContent(\"#tail\", \"baseCtor\") } write() { host.setTextContent(\"#side\", \"baseMethod\") } static ping() { host.setTextContent(\"#extra\", \"ping\") } }; class Derived extends Base { value = \"derived\"; static kind = \"derived\" }; let instance = new Derived(); host.setTextContent(\"#out\", `${Derived.kind}-${instance.value}`); instance.write(); Derived.ping()"})
	if err != nil {
		t.Fatalf("Dispatch(class inheritance) error = %v", err)
	}
	if len(host.calls) != 4 {
		t.Fatalf("host calls = %#v, want four calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#tail" {
		t.Fatalf("host.calls[0] = %#v, want inherited constructor side effect", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "baseCtor" {
		t.Fatalf("host.calls[0].args[1] = %#v, want baseCtor", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#out" {
		t.Fatalf("host.calls[1] = %#v, want inherited and overridden fields", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "derived-derived" {
		t.Fatalf("host.calls[1].args[1] = %#v, want derived-derived", host.calls[1].args[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[0].Kind != ValueKindString || host.calls[2].args[0].String != "#side" {
		t.Fatalf("host.calls[2] = %#v, want inherited instance method call", host.calls[2])
	}
	if host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "baseMethod" {
		t.Fatalf("host.calls[2].args[1] = %#v, want baseMethod", host.calls[2].args[1])
	}
	if host.calls[3].method != "setTextContent" || host.calls[3].args[0].Kind != ValueKindString || host.calls[3].args[0].String != "#extra" {
		t.Fatalf("host.calls[3] = %#v, want inherited static method call", host.calls[3])
	}
	if host.calls[3].args[1].Kind != ValueKindString || host.calls[3].args[1].String != "ping" {
		t.Fatalf("host.calls[3].args[1] = %#v, want ping", host.calls[3].args[1])
	}
}

func TestDispatchRejectsMissingBaseClassInExtends(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Derived extends Missing {}`})
	if err == nil {
		t.Fatalf("Dispatch(missing base class in extends) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(missing base class in extends) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(missing base class in extends) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsConstructorArgumentsInNewExpressions(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "class Example { constructor(first = \"seed\", second = \"tail\") { host.setTextContent(\"#out\", `" + "${first}-${second}" + "` ) } }; new Example(\"picked\")"})
	if err != nil {
		t.Fatalf("Dispatch(new expression with constructor args) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 {
		t.Fatalf("host.calls[0].args len = %d, want 2", len(host.calls[0].args))
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "picked-tail" {
		t.Fatalf("host.calls[0].args[1] = %#v, want picked-tail", host.calls[0].args[1])
	}
}

func TestDispatchSupportsPrivateClassFieldsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Base { #base = "base"; #writeBase() { host.setTextContent("#base", this.#base) } readBase() { this.#writeBase() } }; class Derived extends Base { #derived = "derived"; mirrored = this.#derived; static #count = "7"; static #writeCount() { host.setTextContent("#count", this.#count) } static reveal() { this.#writeCount() } #writeDerived() { host.setTextContent("#derived", this.#derived) } readDerived() { this.#writeDerived() } }; let instance = new Derived(); host.setTextContent("#mirror", instance.mirrored); instance.readBase(); instance.readDerived(); Derived.reveal()`})
	if err != nil {
		t.Fatalf("Dispatch(private class fields) error = %v", err)
	}
	if len(host.calls) != 4 {
		t.Fatalf("Dispatch(private class fields) host calls = %#v, want four calls", host.calls)
	}
	if host.calls[0].method != "setTextContent" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#mirror" {
		t.Fatalf("host.calls[0] = %#v, want mirrored field write", host.calls[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "derived" {
		t.Fatalf("host.calls[0].args[1] = %#v, want derived", host.calls[0].args[1])
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "#base" {
		t.Fatalf("host.calls[1] = %#v, want base private field access", host.calls[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "base" {
		t.Fatalf("host.calls[1].args[1] = %#v, want base", host.calls[1].args[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[0].Kind != ValueKindString || host.calls[2].args[0].String != "#derived" {
		t.Fatalf("host.calls[2] = %#v, want derived private field access", host.calls[2])
	}
	if host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "derived" {
		t.Fatalf("host.calls[2].args[1] = %#v, want derived", host.calls[2].args[1])
	}
	if host.calls[3].method != "setTextContent" || host.calls[3].args[0].Kind != ValueKindString || host.calls[3].args[0].String != "#count" {
		t.Fatalf("host.calls[3] = %#v, want static private field access", host.calls[3])
	}
	if host.calls[3].args[1].Kind != ValueKindString || host.calls[3].args[1].String != "7" {
		t.Fatalf("host.calls[3].args[1] = %#v, want 7", host.calls[3].args[1])
	}
}

func TestDispatchSupportsPrivateClassFieldAssignmentInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Counter { #value = 1; constructor() { this.#value = this.#value + 1 } inc() { this.#value = this.#value + 1 } read() { return this.#value } }; let counter = new Counter(); counter.inc(); host.echo(counter.read())`})
	if err != nil {
		t.Fatalf("Dispatch(private class field assignment) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindNumber || host.calls[0].args[0].Number != 3 {
		t.Fatalf("host.calls[0].args[0] = %#v, want 3", host.calls[0].args[0])
	}
}

func TestDispatchSupportsPrivateInOperatorInClassicJS(t *testing.T) {
	runtime := NewRuntime(&echoHost{})

	result, err := runtime.Dispatch(DispatchRequest{Source: `class Example { #secret = 1; has(other) { return #secret in other } }; let example = new Example(); host.echo(example.has(example) + "-" + example.has({}))`})
	if err != nil {
		t.Fatalf("Dispatch(private `in` operator) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "true-false" {
		t.Fatalf("Dispatch(private `in` operator) value = %#v, want true-false", result.Value)
	}
}

func TestDispatchRejectsPrivateInOperatorOutsideClassBodyInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `#secret in {}`})
	if err == nil {
		t.Fatalf("Dispatch(private `in` operator outside class body) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(private `in` operator outside class body) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(private `in` operator outside class body) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsPrivateClassFieldAccessOutsideClass(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { #secret = 1 }; let example = new Example(); example.#secret`})
	if err == nil {
		t.Fatalf("Dispatch(private class field access outside class) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(private class field access outside class) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(private class field access outside class) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsTryWithoutCatchOrFinally(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{},
		errs:   map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `try { host.setTextContent("#out", "x") }`})
	if err == nil {
		t.Fatalf("Dispatch(try without catch/finally) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(try without catch/finally) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(try without catch/finally) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsLogicalAssignmentOperatorsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo":           StringValue("done"),
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let left = "kept"; left ||= host.echo("boom"); let middle = null; middle ??= "fresh"; let right = true; right &&= host.echo("done"); host.setTextContent("#left", left); host.setTextContent("#middle", middle); host.setTextContent("#right", right)`})
	if err != nil {
		t.Fatalf("Dispatch(logical assignment operators) error = %v", err)
	}
	if len(host.calls) != 4 {
		t.Fatalf("host calls = %#v, want four calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "done" {
		t.Fatalf("host calls[0].args = %#v, want echo(done)", host.calls[0].args)
	}
	if host.calls[1].method != "setTextContent" || host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "kept" {
		t.Fatalf("host calls[1] = %#v, want left kept", host.calls[1])
	}
	if host.calls[2].method != "setTextContent" || host.calls[2].args[1].Kind != ValueKindString || host.calls[2].args[1].String != "fresh" {
		t.Fatalf("host calls[2] = %#v, want middle fresh", host.calls[2])
	}
	if host.calls[3].method != "setTextContent" || host.calls[3].args[1].Kind != ValueKindString || host.calls[3].args[1].String != "done" {
		t.Fatalf("host calls[3] = %#v, want right done", host.calls[3])
	}
}

func TestDispatchSupportsConditionalPrecedenceAfterShortCircuitOperatorsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `[
		true || false ? "or-yes" : "or-no",
		false && true ? "and-yes" : "and-no",
		false ?? true ? "nullish-yes" : "nullish-no"
	].join("|")`})
	if err != nil {
		t.Fatalf("Dispatch(conditional precedence after short-circuit operators) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "or-yes|and-no|nullish-no" {
		t.Fatalf("Dispatch(conditional precedence after short-circuit operators) value = %#v, want string or-yes|and-no|nullish-no", result.Value)
	}
}

func TestDispatchSupportsLogicalOrAndAndOnNonScalarValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("boom"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { kind: "box" }; let arr = [1, 2]; let text = "seed"; let objectOr = obj || host.echo("boom"); let arrayAnd = arr && { kind: "fresh" }; let stringOr = text || host.echo("boom"); host.echo(objectOr.kind, arrayAnd.kind, stringOr)`})
	if err != nil {
		t.Fatalf("Dispatch(logical or/and on non-scalar values) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "boom" {
		t.Fatalf("Dispatch(logical or/and on non-scalar values) value = %#v, want string boom", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 3 {
		t.Fatalf("host.calls[0].args len = %d, want 3", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "box" {
		t.Fatalf("host.calls[0].args[0] = %#v, want box", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "fresh" {
		t.Fatalf("host.calls[0].args[1] = %#v, want fresh", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindString || host.calls[0].args[2].String != "seed" {
		t.Fatalf("host.calls[0].args[2] = %#v, want seed", host.calls[0].args[2])
	}
}

func TestDispatchSupportsLogicalAndShortCircuitBeforeAdditionInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `false && ({ kind: "box" } + 1)`})
	if err != nil {
		t.Fatalf("Dispatch(logical and short-circuit before addition) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || result.Value.Bool {
		t.Fatalf("Dispatch(logical and short-circuit before addition) value = %#v, want false", result.Value)
	}
}

func TestDispatchSupportsLogicalOrShortCircuitBeforeAdditionInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `true || ({ kind: "box" } + 1)`})
	if err != nil {
		t.Fatalf("Dispatch(logical or short-circuit before addition) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || !result.Value.Bool {
		t.Fatalf("Dispatch(logical or short-circuit before addition) value = %#v, want true", result.Value)
	}
}

func TestDispatchSupportsSwitchDiscriminantsOnNonScalarValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { kind: "box" }; let arr = [1, 2]; let out = ""; switch (obj) { case { kind: "box" }: out = "bad"; break; case obj: out = "obj"; break; default: out = "default" }; switch (arr) { case [1, 2]: out += "|bad"; break; case arr: out += "|arr"; break; default: out += "|default" }; switch ("seed") { case "seed": out += "|seed"; break; default: out += "|default" }; host.echo(out)`})
	if err != nil {
		t.Fatalf("Dispatch(switch discriminants on non-scalar values) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(switch discriminants on non-scalar values) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "obj|arr|seed" {
		t.Fatalf("host.calls[0].args[0] = %#v, want obj|arr|seed", host.calls[0].args[0])
	}
}

func TestDispatchSupportsLogicalAssignmentOnObjectPropertiesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { _value: "", get value() { return this._value }, set value(next) { this._value = next }, nested: { count: null } }; obj.value ||= "fresh"; obj.nested.count ??= 7; host.echo(obj.value, obj._value, obj.nested.count)`})
	if err != nil {
		t.Fatalf("Dispatch(logical assignment on object properties) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "fresh" {
		t.Fatalf("Dispatch(logical assignment on object properties) value = %#v, want string fresh", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host call args len = %d, want 3", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "fresh" {
		t.Fatalf("host call arg[0] = %#v, want string fresh", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "fresh" {
		t.Fatalf("host call arg[1] = %#v, want string fresh", call.args[1])
	}
	if call.args[2].Kind != ValueKindNumber || call.args[2].Number != 7 {
		t.Fatalf("host call arg[2] = %#v, want number 7", call.args[2])
	}
}

func TestDispatchSupportsArrayDestructuringAssignmentInsideElseIfBranchInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let state = { rows: [{ id: "a" }, { id: "b" }, { id: "c" }] };
function reorder(action, index) {
  if (action === "duplicate") {
    state.rows.splice(index + 1, 0, state.rows[index]);
  } else if (action === "delete") {
    state.rows.splice(index, 1);
  } else if (action === "up" && index > 0) {
    [state.rows[index - 1], state.rows[index]] = [state.rows[index], state.rows[index - 1]];
  } else if (action === "down" && index < state.rows.length - 1) {
    [state.rows[index + 1], state.rows[index]] = [state.rows[index], state.rows[index + 1]];
  }
}
reorder("up", 2);
host.echo(state.rows.map((row) => row.id).join(","))`})
	if err != nil {
		t.Fatalf("Dispatch(array destructuring assignment inside else-if branch) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "a,c,b" {
		t.Fatalf("Dispatch(array destructuring assignment inside else-if branch) result = %#v, want string a,c,b", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "a,c,b" {
		t.Fatalf("host.calls[0].args = %#v, want one string arg a,c,b", host.calls[0].args)
	}
}

func TestDispatchSupportsSingleStatementElseIfChainsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function evaluateMetricValue(def, row, profile) {
  let inside = false;
  let distance = 5;
  let margin = 8;
  let score = 100;
  if (!inside) {
    if (distance <= margin) score = 70;
    else if (distance <= margin * 2) score = 40;
    else score = 10;
  }
  return score;
}
evaluateMetricValue()`})
	if err != nil {
		t.Fatalf("Dispatch(single-statement else-if chain) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 70 {
		t.Fatalf("Dispatch(single-statement else-if chain) result = %#v, want number 70", result.Value)
	}
}

func TestDispatchRejectsArrayDestructuringAssignmentWithNonAssignableTargetInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[1] = [2]`})
	if err == nil {
		t.Fatalf("Dispatch(array destructuring assignment with non-assignable target) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(array destructuring assignment with non-assignable target) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(array destructuring assignment with non-assignable target) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsCompoundAssignmentOperatorsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let value = 1; value += 2; value *= 3; let obj = { count: 2, nested: { mask: 3 } }; obj.count -= 1; obj.nested.mask <<= 2; host.echo(value, obj.count, obj.nested.mask)`})
	if err != nil {
		t.Fatalf("Dispatch(compound assignment operators) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 9 {
		t.Fatalf("Dispatch(compound assignment operators) result = %#v, want number 9", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host call args len = %d, want 3", len(call.args))
	}
	if call.args[0].Kind != ValueKindNumber || call.args[0].Number != 9 {
		t.Fatalf("host call arg[0] = %#v, want number 9", call.args[0])
	}
	if call.args[1].Kind != ValueKindNumber || call.args[1].Number != 1 {
		t.Fatalf("host call arg[1] = %#v, want number 1", call.args[1])
	}
	if call.args[2].Kind != ValueKindNumber || call.args[2].Number != 12 {
		t.Fatalf("host call arg[2] = %#v, want number 12", call.args[2])
	}
}

func TestDispatchRejectsLogicalAssignmentOnGetterOnlyObjectProperty(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { get value() { return "" } }; obj.value ||= "fresh"`})
	if err == nil {
		t.Fatalf("Dispatch(logical assignment on getter-only object property) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(logical assignment on getter-only object property) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(logical assignment on getter-only object property) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsLogicalAssignmentOnConstBinding(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{},
		errs:   map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `const value = false; value ||= "other"`})
	if err == nil {
		t.Fatalf("Dispatch(logical assignment on const) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(logical assignment on const) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(logical assignment on const) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsCompoundAssignmentOnBigIntUnsignedShiftInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{},
		errs:   map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let value = 1n; value >>>= 1n`})
	if err == nil {
		t.Fatalf("Dispatch(compound assignment on bigint unsigned shift) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(compound assignment on bigint unsigned shift) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(compound assignment on bigint unsigned shift) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchDoesNotMutateSkippedLogicalAssignmentRhs(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let outer = "kept"; let inner = "seed"; outer ||= (inner ||= "changed"); host.setTextContent("#out", inner)`})
	if err != nil {
		t.Fatalf("Dispatch(skipped logical assignment rhs) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "seed" {
		t.Fatalf("host calls[0].args = %#v, want inner to remain seed", host.calls[0].args)
	}
}

func TestDispatchSupportsPlainTemplateLiterals(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "host.echo(`hello world`)"})
	if err != nil {
		t.Fatalf("Dispatch(plain template literal) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(plain template literal) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "hello world" {
		t.Fatalf("host calls[0].args[0] = %#v, want hello world", host.calls[0].args[0])
	}
}

func TestDispatchSupportsTemplateLiteralInterpolation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "let name = \"world\"; host.echo(`hello ${name}!`)"})
	if err != nil {
		t.Fatalf("Dispatch(template literal interpolation) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(template literal interpolation) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "hello world!" {
		t.Fatalf("host calls[0].args[0] = %#v, want hello world!", host.calls[0].args[0])
	}
}

func TestDispatchSupportsNestedTemplateLiteralInterpolation(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "host.echo(`${true ? ` / ${1}` : \"\"}`)"})
	if err != nil {
		t.Fatalf("Dispatch(nested template literal interpolation) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != " / 1" {
		t.Fatalf("Dispatch(nested template literal interpolation) value = %#v, want string ` / 1`", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != " / 1" {
		t.Fatalf("host.calls[0].args = %#v, want string ` / 1`", host.calls[0].args)
	}
}

func TestDispatchSupportsNestedTemplateLiteralScannerBoundaries(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := `class Box { label = ` + "`" + `outer${` + "`" + `inner` + "`" + `}` + "`" + `; read() { return this.label } }` +
		`; const pick = (value = ` + "`" + `default${` + "`" + `value` + "`" + `}` + "`" + `) => value;` +
		` const obj = { value: "done" }; let status = "";` +
		` switch (` + "`" + `state${` + "`" + `1` + "`" + `}` + "`" + `) {` +
		` case ` + "`" + `state${` + "`" + `1` + "`" + `}` + "`" + `:` +
		` status = ` + "`" + `hit${` + "`" + `!` + "`" + `}` + "`" + `; break;` +
		` default: status = "miss"; }` +
		` host.echo(` + "`" + `call${` + "`" + `arg` + "`" + `}` + "`" + `, new Box().read(), pick(), obj[` + "`" + `val${` + "`" + `ue` + "`" + `}` + "`" + `], status)`

	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(nested template literal scanner boundaries) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(nested template literal scanner boundaries) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 5 {
		t.Fatalf("host call args len = %d, want 5", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "callarg" {
		t.Fatalf("host call arg[0] = %#v, want string callarg", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "outerinner" {
		t.Fatalf("host call arg[1] = %#v, want string outerinner", call.args[1])
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != "defaultvalue" {
		t.Fatalf("host call arg[2] = %#v, want string defaultvalue", call.args[2])
	}
	if call.args[3].Kind != ValueKindString || call.args[3].String != "done" {
		t.Fatalf("host call arg[3] = %#v, want string done", call.args[3])
	}
	if call.args[4].Kind != ValueKindString || call.args[4].String != "hit!" {
		t.Fatalf("host call arg[4] = %#v, want string hit!", call.args[4])
	}
}

func TestDispatchSupportsTaggedTemplateLiterals(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "function tag(strings, left, right) { return strings[0] + left + strings[1] + right + strings[2] }; host.echo(tag`hello ${\"world\"} ${1 + 1}!`)"})
	if err != nil {
		t.Fatalf("Dispatch(tagged template literal) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(tagged template literal) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "hello world 2!" {
		t.Fatalf("host.calls[0].args[0] = %#v, want hello world 2!", host.calls[0].args[0])
	}
}

func TestDispatchSupportsRegularExpressionLiterals(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: "let re = /a\\/b/i; host.echo(`" + "${re}|${re.test(\"A/B\")}|${re.source}|${re.flags}" + "`)"})
	if err != nil {
		t.Fatalf("Dispatch(regular expression literal) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "/a\\/b/i|true|a\\/b|i" {
		t.Fatalf("Dispatch(regular expression literal) value = %#v, want regex string and helpers", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "/a\\/b/i|true|a\\/b|i" {
		t.Fatalf("host.calls[0].args = %#v, want regex string and helpers", host.calls[0].args)
	}
}

func TestDispatchSupportsRegularExpressionCommaLiteralInCallArguments(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo("1,234".replace(/,/g, ""))`})
	if err != nil {
		t.Fatalf("Dispatch(regular expression comma literal) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "1234" {
		t.Fatalf("Dispatch(regular expression comma literal) value = %#v, want string 1234", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "1234" {
		t.Fatalf("host.calls[0].args = %#v, want string 1234", host.calls[0].args)
	}
}

func TestDispatchSupportsRegularExpressionUnicodeEscapeLiterals(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(/[\uFF10-\uFF19]/.test("０"))`})
	if err != nil {
		t.Fatalf("Dispatch(regular expression unicode escape literal) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || !result.Value.Bool {
		t.Fatalf("Dispatch(regular expression unicode escape literal) value = %#v, want bool true", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindBool || !host.calls[0].args[0].Bool {
		t.Fatalf("host.calls[0].args = %#v, want bool true", host.calls[0].args)
	}
}

func TestDispatchRejectsMalformedRegularExpressionCommaLiteralInCallArguments(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo("1,234".replace(/,(/g, ""))`})
	if err == nil {
		t.Fatalf("Dispatch(malformed regular expression comma literal) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed regular expression comma literal) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed regular expression comma literal) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedRegularExpressionLiteralsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let re = /(/; re`})
	if err == nil {
		t.Fatalf("Dispatch(malformed regular expression literal) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed regular expression literal) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed regular expression literal) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsCommaOperatorSequenceExpressions(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let value = "left"; host.echo((value, "right"))`})
	if err != nil {
		t.Fatalf("Dispatch(comma operator sequence expression) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "right" {
		t.Fatalf("Dispatch(comma operator sequence expression) value = %#v, want string right", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host.calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "right" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string right", host.calls[0].args[0])
	}
}

func TestDispatchRejectsMalformedTemplateLiteralInterpolation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "host.echo(`hello ${1 + }`)"})
	if err == nil {
		t.Fatalf("Dispatch(malformed template literal interpolation) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed template literal interpolation) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed template literal interpolation) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchReportsRuntimeErrorForTaggedTemplateLiteralOnNonCallableTargetInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{},
		errs:   map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "host.echo((1)`hello`)"})
	if err == nil {
		t.Fatalf("Dispatch(non-callable tagged template) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(non-callable tagged template) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(non-callable tagged template) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchReportsRuntimeErrorForNonCallableCallExpressionsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{},
		errs:   map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "({})()"})
	if err == nil {
		t.Fatalf("Dispatch(non-callable call expression) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(non-callable call expression) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(non-callable call expression) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsArrayAndObjectLiterals(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo([1, "two", null], {kind: "box", count: 2})`})
	if err != nil {
		t.Fatalf("Dispatch(array/object literals) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(array/object literals) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	arrayValue := call.args[0]
	if arrayValue.Kind != ValueKindArray {
		t.Fatalf("host call arg[0].Kind = %q, want %q", arrayValue.Kind, ValueKindArray)
	}
	if len(arrayValue.Array) != 3 {
		t.Fatalf("host call arg[0] array len = %d, want 3", len(arrayValue.Array))
	}
	if arrayValue.Array[0].Kind != ValueKindNumber || arrayValue.Array[0].Number != 1 {
		t.Fatalf("host call arg[0].Array[0] = %#v, want number 1", arrayValue.Array[0])
	}
	if arrayValue.Array[1].Kind != ValueKindString || arrayValue.Array[1].String != "two" {
		t.Fatalf("host call arg[0].Array[1] = %#v, want string two", arrayValue.Array[1])
	}
	if arrayValue.Array[2].Kind != ValueKindNull {
		t.Fatalf("host call arg[0].Array[2] = %#v, want null", arrayValue.Array[2])
	}
	objectValue := call.args[1]
	if objectValue.Kind != ValueKindObject {
		t.Fatalf("host call arg[1].Kind = %q, want %q", objectValue.Kind, ValueKindObject)
	}
	if len(objectValue.Object) != 2 {
		t.Fatalf("host call arg[1] object len = %d, want 2", len(objectValue.Object))
	}
	if objectValue.Object[0].Key != "kind" || objectValue.Object[0].Value.Kind != ValueKindString || objectValue.Object[0].Value.String != "box" {
		t.Fatalf("host call arg[1].Object[0] = %#v, want kind: box", objectValue.Object[0])
	}
	if objectValue.Object[1].Key != "count" || objectValue.Object[1].Value.Kind != ValueKindNumber || objectValue.Object[1].Value.Number != 2 {
		t.Fatalf("host call arg[1].Object[1] = %#v, want count: 2", objectValue.Object[1])
	}
}

func TestDispatchSupportsArrayLiteralElisionsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let array = [1, , "two", , null]; host.echo(array.length, typeof array[1], typeof array[3])`})
	if err != nil {
		t.Fatalf("Dispatch(array literal elisions) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(array literal elisions) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host call args len = %d, want 3", len(call.args))
	}
	if call.args[0].Kind != ValueKindNumber || call.args[0].Number != 5 {
		t.Fatalf("host call arg[0] = %#v, want number 5", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "undefined" {
		t.Fatalf("host call arg[1] = %#v, want string undefined", call.args[1])
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != "undefined" {
		t.Fatalf("host call arg[2] = %#v, want string undefined", call.args[2])
	}
}

func TestDispatchSupportsArrayAndObjectDestructuring(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let [first, , third] = [1, 2, 3]; let {kind: label, count} = {kind: "box", count: 2}; host.echo(first, third, label, count)`})
	if err != nil {
		t.Fatalf("Dispatch(array/object destructuring) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(array/object destructuring) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[0].Kind != ValueKindNumber || call.args[0].Number != 1 {
		t.Fatalf("host call arg[0] = %#v, want number 1", call.args[0])
	}
	if call.args[1].Kind != ValueKindNumber || call.args[1].Number != 3 {
		t.Fatalf("host call arg[1] = %#v, want number 3", call.args[1])
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != "box" {
		t.Fatalf("host call arg[2] = %#v, want string box", call.args[2])
	}
	if call.args[3].Kind != ValueKindNumber || call.args[3].Number != 2 {
		t.Fatalf("host call arg[3] = %#v, want number 2", call.args[3])
	}
}

func TestDispatchSupportsComputedObjectDestructuringKeys(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let key = "kind"; let { [key]: label, ["count"]: total } = {kind: "box", count: 2}; host.echo(label, total)`})
	if err != nil {
		t.Fatalf("Dispatch(computed object destructuring keys) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(computed object destructuring keys) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "box" {
		t.Fatalf("host call arg[0] = %#v, want string box", call.args[0])
	}
	if call.args[1].Kind != ValueKindNumber || call.args[1].Number != 2 {
		t.Fatalf("host call arg[1] = %#v, want number 2", call.args[1])
	}
}

func TestDispatchSupportsDestructuringDefaults(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let [first = "fallback", second = first] = []; const {kind = "box", label: alias = kind} = {}; host.echo(first, second, kind, alias)`})
	if err != nil {
		t.Fatalf("Dispatch(destructuring defaults) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(destructuring defaults) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "fallback" {
		t.Fatalf("host call arg[0] = %#v, want string fallback", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "fallback" {
		t.Fatalf("host call arg[1] = %#v, want string fallback", call.args[1])
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != "box" {
		t.Fatalf("host call arg[2] = %#v, want string box", call.args[2])
	}
	if call.args[3].Kind != ValueKindString || call.args[3].String != "box" {
		t.Fatalf("host call arg[3] = %#v, want string box", call.args[3])
	}
}

func TestDispatchRejectsMalformedComputedObjectDestructuringKeys(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let { []: value } = {}; value`})
	if err == nil {
		t.Fatalf("Dispatch(malformed computed object destructuring key) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed computed object destructuring key) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed computed object destructuring key) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsVarDeclarationsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `var value = 1; var value = value + 1; value`})
	if err != nil {
		t.Fatalf("Dispatch(var declarations) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 2 {
		t.Fatalf("Dispatch(var declarations) value = %#v, want number 2", result.Value)
	}
}

func TestDispatchSupportsUsingDeclarationsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `using value = "seed"; await using asyncValue = "tail"; value + "-" + asyncValue`})
	if err != nil {
		t.Fatalf("Dispatch(using declarations) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "seed-tail" {
		t.Fatalf("Dispatch(using declarations) value = %#v, want string seed-tail", result.Value)
	}
}

func TestDispatchRejectsReservedVarDeclarationNames(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `var let = 1`})
	if err == nil {
		t.Fatalf("Dispatch(reserved var declaration name) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(reserved var declaration name) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(reserved var declaration name) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedUsingDeclaration(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `using value`})
	if err == nil {
		t.Fatalf("Dispatch(malformed using declaration) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed using declaration) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed using declaration) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedDestructuringDefaults(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let [value =] = []; value`})
	if err == nil {
		t.Fatalf("Dispatch(malformed destructuring default) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed destructuring default) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed destructuring default) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsArrayAndObjectSpreadAndRest(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let more = [2, 3]; let extra = {kind: "box"}; let [first, ...rest] = [1, ...more, 4]; let {kind, ...others} = {...extra, count: 2}; host.echo(first, rest, kind, others)`})
	if err != nil {
		t.Fatalf("Dispatch(spread/rest) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(spread/rest) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[0].Kind != ValueKindNumber || call.args[0].Number != 1 {
		t.Fatalf("host call arg[0] = %#v, want number 1", call.args[0])
	}
	if call.args[1].Kind != ValueKindArray {
		t.Fatalf("host call arg[1].Kind = %q, want %q", call.args[1].Kind, ValueKindArray)
	}
	if len(call.args[1].Array) != 3 {
		t.Fatalf("host call arg[1] array len = %d, want 3", len(call.args[1].Array))
	}
	if call.args[1].Array[0].Kind != ValueKindNumber || call.args[1].Array[0].Number != 2 {
		t.Fatalf("host call arg[1].Array[0] = %#v, want number 2", call.args[1].Array[0])
	}
	if call.args[1].Array[1].Kind != ValueKindNumber || call.args[1].Array[1].Number != 3 {
		t.Fatalf("host call arg[1].Array[1] = %#v, want number 3", call.args[1].Array[1])
	}
	if call.args[1].Array[2].Kind != ValueKindNumber || call.args[1].Array[2].Number != 4 {
		t.Fatalf("host call arg[1].Array[2] = %#v, want number 4", call.args[1].Array[2])
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != "box" {
		t.Fatalf("host call arg[2] = %#v, want string box", call.args[2])
	}
	if call.args[3].Kind != ValueKindObject {
		t.Fatalf("host call arg[3].Kind = %q, want %q", call.args[3].Kind, ValueKindObject)
	}
	if len(call.args[3].Object) != 1 {
		t.Fatalf("host call arg[3] object len = %d, want 1", len(call.args[3].Object))
	}
	if call.args[3].Object[0].Key != "count" || call.args[3].Object[0].Value.Kind != ValueKindNumber || call.args[3].Object[0].Value.Number != 2 {
		t.Fatalf("host call arg[3].Object[0] = %#v, want count: 2", call.args[3].Object[0])
	}
}

func TestDispatchSupportsObjectSpreadOnStringAndArrayValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let text = "go"; let array = [1, 2]; let spreadText = {...text}; let spreadArray = {...array}; host.echo(spreadText["0"], spreadText["1"], spreadArray["0"], spreadArray["1"])`})
	if err != nil {
		t.Fatalf("Dispatch(object spread on string and array) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object spread on string and array) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	wantValues := []string{"g", "o"}
	for i, want := range wantValues {
		if call.args[i].Kind != ValueKindString || call.args[i].String != want {
			t.Fatalf("host call text spread arg[%d] = %#v, want string %q", i, call.args[i], want)
		}
	}
	if call.args[2].Kind != ValueKindNumber || call.args[2].Number != 1 {
		t.Fatalf("host call array spread arg[2] = %#v, want number 1", call.args[2])
	}
	if call.args[3].Kind != ValueKindNumber || call.args[3].Number != 2 {
		t.Fatalf("host call array spread arg[3] = %#v, want number 2", call.args[3])
	}
}

func TestDispatchSupportsObjectSpreadOnPrimitiveValuesAsNoOpInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let spread = { seed: "ok", ...1, ...false, ...1n }; host.echo(spread.seed, typeof spread["0"], typeof spread["1"], typeof spread["2"])`})
	if err != nil {
		t.Fatalf("Dispatch(object spread on primitive values as no-op) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object spread on primitive values as no-op) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "ok" {
		t.Fatalf("host call primitive spread arg[0] = %#v, want string ok", call.args[0])
	}
	for i := 1; i < len(call.args); i++ {
		if call.args[i].Kind != ValueKindString || call.args[i].String != "undefined" {
			t.Fatalf("host call primitive spread arg[%d] = %#v, want string undefined", i, call.args[i])
		}
	}
}

func TestDispatchSupportsObjectSpreadOnNullishValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let spreadNull = {...null}; let spreadUndefined = {...undefined}; host.echo(spreadNull["0"], spreadUndefined["0"])`})
	if err != nil {
		t.Fatalf("Dispatch(object spread on nullish values) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object spread on nullish values) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindUndefined {
		t.Fatalf("host call null spread arg[0] = %#v, want undefined", call.args[0])
	}
	if call.args[1].Kind != ValueKindUndefined {
		t.Fatalf("host call undefined spread arg[1] = %#v, want undefined", call.args[1])
	}
}

func TestDispatchSupportsArraySpreadAndDestructuringFromIteratorLikeValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function* values() { yield "left"; yield "right" }; let [first, ...rest] = values(); let spread = [...values()]; host.echo(first, rest, spread)`})
	if err != nil {
		t.Fatalf("Dispatch(array spread/destructuring on iterator-like object) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(array spread/destructuring on iterator-like object) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host call args len = %d, want 3", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "left" {
		t.Fatalf("host call arg[0] = %#v, want string left", call.args[0])
	}
	if call.args[1].Kind != ValueKindArray {
		t.Fatalf("host call arg[1].Kind = %q, want %q", call.args[1].Kind, ValueKindArray)
	}
	if len(call.args[1].Array) != 1 {
		t.Fatalf("host call arg[1] array len = %d, want 1", len(call.args[1].Array))
	}
	if call.args[1].Array[0].Kind != ValueKindString || call.args[1].Array[0].String != "right" {
		t.Fatalf("host call arg[1].Array[0] = %#v, want string right", call.args[1].Array[0])
	}
	if call.args[2].Kind != ValueKindArray {
		t.Fatalf("host call arg[2].Kind = %q, want %q", call.args[2].Kind, ValueKindArray)
	}
	if len(call.args[2].Array) != 2 {
		t.Fatalf("host call arg[2] array len = %d, want 2", len(call.args[2].Array))
	}
	if call.args[2].Array[0].Kind != ValueKindString || call.args[2].Array[0].String != "left" {
		t.Fatalf("host call arg[2].Array[0] = %#v, want string left", call.args[2].Array[0])
	}
	if call.args[2].Array[1].Kind != ValueKindString || call.args[2].Array[1].String != "right" {
		t.Fatalf("host call arg[2].Array[1] = %#v, want string right", call.args[2].Array[1])
	}
}

func TestDispatchSupportsArraySpreadAndDestructuringFromStringValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let text = "go"; let [first, ...rest] = text; let spread = [...text]; host.echo(first, rest, spread, ...text)`})
	if err != nil {
		t.Fatalf("Dispatch(array spread/destructuring on string value) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(array spread/destructuring on string value) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 5 {
		t.Fatalf("host call args len = %d, want 5", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "g" {
		t.Fatalf("host call arg[0] = %#v, want string g", call.args[0])
	}
	if call.args[1].Kind != ValueKindArray {
		t.Fatalf("host call arg[1].Kind = %q, want %q", call.args[1].Kind, ValueKindArray)
	}
	if len(call.args[1].Array) != 1 {
		t.Fatalf("host call arg[1] array len = %d, want 1", len(call.args[1].Array))
	}
	if call.args[1].Array[0].Kind != ValueKindString || call.args[1].Array[0].String != "o" {
		t.Fatalf("host call arg[1].Array[0] = %#v, want string o", call.args[1].Array[0])
	}
	if call.args[2].Kind != ValueKindArray {
		t.Fatalf("host call arg[2].Kind = %q, want %q", call.args[2].Kind, ValueKindArray)
	}
	if len(call.args[2].Array) != 2 {
		t.Fatalf("host call arg[2] array len = %d, want 2", len(call.args[2].Array))
	}
	if call.args[2].Array[0].Kind != ValueKindString || call.args[2].Array[0].String != "g" {
		t.Fatalf("host call arg[2].Array[0] = %#v, want string g", call.args[2].Array[0])
	}
	if call.args[2].Array[1].Kind != ValueKindString || call.args[2].Array[1].String != "o" {
		t.Fatalf("host call arg[2].Array[1] = %#v, want string o", call.args[2].Array[1])
	}
	if call.args[3].Kind != ValueKindString || call.args[3].String != "g" {
		t.Fatalf("host call arg[3] = %#v, want string g", call.args[3])
	}
	if call.args[4].Kind != ValueKindString || call.args[4].String != "o" {
		t.Fatalf("host call arg[4] = %#v, want string o", call.args[4])
	}
}

func TestDispatchSupportsArrowFunctions(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let identity = x => x; let render = () => ` + "`" + `arrow${` + "`" + `body;value` + "`" + `}` + "`" + `; let collect = (...items) => items; host.echo(identity("fine"), collect(1, 2, 3), render())`})
	if err != nil {
		t.Fatalf("Dispatch(arrow functions) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(arrow functions) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host call args len = %d, want 3", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "fine" {
		t.Fatalf("host call arg[0] = %#v, want string fine", call.args[0])
	}
	if call.args[1].Kind != ValueKindArray {
		t.Fatalf("host call arg[1].Kind = %q, want %q", call.args[1].Kind, ValueKindArray)
	}
	if len(call.args[1].Array) != 3 {
		t.Fatalf("host call arg[1] array len = %d, want 3", len(call.args[1].Array))
	}
	if call.args[1].Array[0].Kind != ValueKindNumber || call.args[1].Array[0].Number != 1 {
		t.Fatalf("host call arg[1].Array[0] = %#v, want number 1", call.args[1].Array[0])
	}
	if call.args[1].Array[1].Kind != ValueKindNumber || call.args[1].Array[1].Number != 2 {
		t.Fatalf("host call arg[1].Array[1] = %#v, want number 2", call.args[1].Array[1])
	}
	if call.args[1].Array[2].Kind != ValueKindNumber || call.args[1].Array[2].Number != 3 {
		t.Fatalf("host call arg[1].Array[2] = %#v, want number 3", call.args[1].Array[2])
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != "arrowbody;value" {
		t.Fatalf("host call arg[2] = %#v, want string arrowbody;value", call.args[2])
	}
}

func TestDispatchSupportsAsyncArrowFunctionsAndAwait(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let run = async () => host.echo("fine"); run()`})
	if err != nil {
		t.Fatalf("Dispatch(async arrow functions and await) error = %v", err)
	}
	if result.Value.Kind != ValueKindPromise {
		t.Fatalf("Dispatch(async arrow functions and await) kind = %q, want %q", result.Value.Kind, ValueKindPromise)
	}
	if result.Value.Promise == nil {
		t.Fatalf("Dispatch(async arrow functions and await) promise = nil, want string fine")
	}
	if result.Value.Promise.Kind != ValueKindString || result.Value.Promise.String != "fine" {
		t.Fatalf("Dispatch(async arrow functions and await) promise = %#v, want string fine", *result.Value.Promise)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host call method = %q, want echo", host.calls[0].method)
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "fine" {
		t.Fatalf("host calls[0].args[0] = %#v, want string fine", host.calls[0].args[0])
	}
}

func TestDispatchSupportsAwaitInsideAsyncArrowFunctions(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let run = async () => host.echo("fine"); let unwrap = async () => await run(); unwrap()`})
	if err != nil {
		t.Fatalf("Dispatch(await inside async arrow functions) error = %v", err)
	}
	if result.Value.Kind != ValueKindPromise {
		t.Fatalf("Dispatch(await inside async arrow functions) kind = %q, want %q", result.Value.Kind, ValueKindPromise)
	}
	if result.Value.Promise == nil {
		t.Fatalf("Dispatch(await inside async arrow functions) promise = nil, want string fine")
	}
	if result.Value.Promise.Kind != ValueKindString || result.Value.Promise.String != "fine" {
		t.Fatalf("Dispatch(await inside async arrow functions) promise = %#v, want string fine", *result.Value.Promise)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host call method = %q, want echo", host.calls[0].method)
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "fine" {
		t.Fatalf("host calls[0].args[0] = %#v, want string fine", host.calls[0].args[0])
	}
}

func TestDispatchSupportsAsyncFunctionDeclarationsAndAwait(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `async function choose() { return await host.echo("fine") }; choose()`})
	if err != nil {
		t.Fatalf("Dispatch(async function declarations and await) error = %v", err)
	}
	if result.Value.Kind != ValueKindPromise {
		t.Fatalf("Dispatch(async function declarations and await) kind = %q, want %q", result.Value.Kind, ValueKindPromise)
	}
	if result.Value.Promise == nil {
		t.Fatalf("Dispatch(async function declarations and await) promise = nil, want string fine")
	}
	if result.Value.Promise.Kind != ValueKindString || result.Value.Promise.String != "fine" {
		t.Fatalf("Dispatch(async function declarations and await) promise = %#v, want string fine", *result.Value.Promise)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host call method = %q, want echo", host.calls[0].method)
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "fine" {
		t.Fatalf("host calls[0].args[0] = %#v, want string fine", host.calls[0].args[0])
	}
}

func TestDispatchSupportsAwaitAcrossBoundedValuesInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "async function read() { let obj = { value: \"kept\" }; let arr = [\"seed\"]; let text = \"go\"; return `" + "${(await obj).value}|${(await arr)[0]}|${await text}|${await null}" + "` }; read()"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(await across bounded values) error = %v", err)
	}
	if result.Value.Kind != ValueKindPromise {
		t.Fatalf("Dispatch(await across bounded values) kind = %q, want %q", result.Value.Kind, ValueKindPromise)
	}
	if result.Value.Promise == nil {
		t.Fatalf("Dispatch(await across bounded values) promise = nil, want string kept|seed|go|null")
	}
	if result.Value.Promise.Kind != ValueKindString || result.Value.Promise.String != "kept|seed|go|null" {
		t.Fatalf("Dispatch(await across bounded values) promise = %#v, want string kept|seed|go|null", *result.Value.Promise)
	}
}

func TestDispatchRejectsAwaitOutsideAsyncBodiesInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function read() { return await 1 }; read()`})
	if err == nil {
		t.Fatalf("Dispatch(await outside async bodies) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(await outside async bodies) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(await outside async bodies) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsAsyncGeneratorFunctionDeclarationsAndAwait(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	source := "async function* spin() { yield await host.echo(\"first\"); yield await host.echo(\"second\") }; let it = spin(); let first = await it.next(); let second = await it.next(); let third = await it.next(); `" + "${first.value}|${second.value}|${third.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator function declarations and await) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|second|true" {
		t.Fatalf("Dispatch(async generator function declarations and await) result = %#v, want string first|second|true", result.Value)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "first" {
		t.Fatalf("host.calls[0] = %#v, want echo(first)", host.calls[0])
	}
	if host.calls[1].method != "echo" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "second" {
		t.Fatalf("host.calls[1] = %#v, want echo(second)", host.calls[1])
	}
}

func TestDispatchSupportsAsyncClassMethodsAndAwait(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { static async readStatic() { return await host.echo("static") } async readInstance() { return await host.echo("instance") } }; let example = new Example(); await Example.readStatic(); await example.readInstance()`})
	if err != nil {
		t.Fatalf("Dispatch(async class methods and await) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "static" {
		t.Fatalf("host.calls[0] = %#v, want echo(static)", host.calls[0])
	}
	if host.calls[1].method != "echo" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "instance" {
		t.Fatalf("host.calls[1] = %#v, want echo(instance)", host.calls[1])
	}
}

func TestDispatchSupportsGeneratorClassMethodsAndYield(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "class Example { static *readStatic() { yield \"static\" } *readInstance() { yield \"instance\" } }; let example = new Example(); let staticIterator = Example.readStatic(); let instanceIterator = example.readInstance(); host.setTextContent(\"#out\", `" + "${staticIterator.next().value}-${instanceIterator.next().value}" + "`)"})
	if err != nil {
		t.Fatalf("Dispatch(generator class methods and yield) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "static-instance" {
		t.Fatalf("host.calls[0].args[1] = %#v, want string static-instance", host.calls[0].args[1])
	}
}

func TestDispatchSupportsAsyncGeneratorClassMethodsAndAwait(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	source := "class Example { static async *readStatic() { yield await host.echo(\"static\") } async *readInstance() { yield await host.echo(\"instance\") } }; let example = new Example(); let staticIterator = Example.readStatic(); let instanceIterator = example.readInstance(); let staticFirst = await staticIterator.next(); let instanceFirst = await instanceIterator.next(); `" + "${staticFirst.value}|${instanceFirst.value}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator class methods and await) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "static|instance" {
		t.Fatalf("Dispatch(async generator class methods and await) result = %#v, want string static|instance", result.Value)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "static" {
		t.Fatalf("host.calls[0] = %#v, want echo(static)", host.calls[0])
	}
	if host.calls[1].method != "echo" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "instance" {
		t.Fatalf("host.calls[1] = %#v, want echo(instance)", host.calls[1])
	}
}

func TestDispatchSupportsAsyncGeneratorYieldDelegation(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	source := "async function* spin() { yield \"first\"; yield* [await host.echo(\"second\")]; yield \"third\" }; let it = spin(); let first = await it.next(); let second = await it.next(); let third = await it.next(); let done = await it.next(); `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator yield delegation) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|second|third|true" {
		t.Fatalf("Dispatch(async generator yield delegation) result = %#v, want string first|second|third|true", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "second" {
		t.Fatalf("host.calls[0] = %#v, want echo(second)", host.calls[0])
	}
}

func TestDispatchSupportsAsyncGeneratorYieldDelegationOnString(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "async function* spin() { yield* \"go\"; yield \"done\" }; let it = spin(); let first = await it.next(); let second = await it.next(); let third = await it.next(); let done = await it.next(); `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator delegation on string) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "g|o|done|true" {
		t.Fatalf("Dispatch(async generator delegation on string) result = %#v, want string g|o|done|true", result.Value)
	}
}

func TestDispatchSupportsAsyncGeneratorYieldDelegationOnAsyncIteratorLikeObject(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "async function* spin() { yield* { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: \"go\", done: false } }; return { done: true } } }; yield \"done\" }; let it = spin(); let first = await it.next(); let second = await it.next(); let third = await it.next(); `" + "${first.value}|${second.value}|${third.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator delegation on async iterator-like object) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "go|done|true" {
		t.Fatalf("Dispatch(async generator delegation on async iterator-like object) result = %#v, want string go|done|true", result.Value)
	}
}

func TestDispatchSupportsAsyncGeneratorNextArgumentsOnYieldStarDelegatesInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "async function* spin() { yield* { index: 0, async next(value) { if (this.index === 0) { this.index = 1; return { value: value === undefined ? \"first\" : value, done: false } }; if (this.index === 1) { this.index = 2; return { value: value === undefined ? \"second\" : value, done: false } }; return { done: true } } }; yield \"done\" }; let it = spin(); let first = await it.next(); let second = await it.next(\"seed\"); let third = await it.next(); let done = await it.next(); `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator next arguments on yield* delegates) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|seed|done|true" {
		t.Fatalf("Dispatch(async generator next arguments on yield* delegates) result = %#v, want string first|seed|done|true", result.Value)
	}
}

func TestDispatchSupportsAsyncGeneratorNextArgumentsOnYieldExpressionsInDeclarationInitializersAndAssignmentsInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "async function* spin() { let first = yield \"first\"; let value; value = yield first; let box = { value: \"\" }; box.value = yield value; yield `${value}|${box.value}` }; let it = spin(); let one = await it.next(); let two = await it.next(\"seed\"); let three = await it.next(\"tail\"); let four = await it.next(\"final\"); let done = await it.next(); `" + "${one.value}|${two.value}|${three.value}|${four.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator next arguments on yield expressions in declaration initializers and assignments) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|seed|tail|tail|final|true" {
		t.Fatalf("Dispatch(async generator next arguments on yield expressions in declaration initializers and assignments) result = %#v, want string first|seed|tail|tail|final|true", result.Value)
	}
}

func TestDispatchSupportsGeneratorYieldDelegationImmediateIteratorReturnValue(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { let result = yield* { next() { return { value: \"finished\", done: true } } }; yield result }; let it = spin(); let first = it.next(); let done = it.next(); `" + "${first.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(generator delegation immediate return value) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "finished|true" {
		t.Fatalf("Dispatch(generator delegation immediate return value) result = %#v, want string finished|true", result.Value)
	}
}

func TestDispatchSupportsAsyncGeneratorYieldDelegationImmediateIteratorReturnValue(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "async function* spin() { let result = yield* { async next() { return { value: \"finished\", done: true } } }; yield result }; let it = spin(); let first = await it.next(); let done = await it.next(); `" + "${first.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(async generator delegation immediate return value) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "finished|true" {
		t.Fatalf("Dispatch(async generator delegation immediate return value) result = %#v, want string finished|true", result.Value)
	}
}

func TestDispatchSupportsNestedAsyncGeneratorYieldDelegationOnString(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "async function* spin() { if (true) { yield* \"go\"; }; yield \"done\" }; let it = spin(); let first = await it.next(); let second = await it.next(); let third = await it.next(); let done = await it.next(); `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(nested async generator delegation on string) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "g|o|done|true" {
		t.Fatalf("Dispatch(nested async generator delegation on string) result = %#v, want string g|o|done|true", result.Value)
	}
}

func TestDispatchSupportsFunctionDeclarationsAndReturnStatements(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function choose(flag) { if (flag) { return "yes" } return "no" }; choose(true)`})
	if err != nil {
		t.Fatalf("Dispatch(function declarations and return) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "yes" {
		t.Fatalf("Dispatch(function declarations and return) result = %#v, want string yes", result.Value)
	}
}

func TestDispatchSupportsNestedHelperCallInReturnExpression(t *testing.T) {
	runtime := NewRuntime(nil)

	source := `(() => {
		function renderLabel(label) {
			return ` + "`" + `<div class="field">${escapeHtml(label)}</div>` + "`" + `;
		}

		function escapeHtml(value) {
			return "" + (value || "")
				.replace(/&/g, "&amp;")
				.replace(/</g, "&lt;")
				.replace(/>/g, "&gt;")
				.replace(/"/g, "&quot;")
				.replace(/'/g, "&#39;");
		}

		return renderLabel("Holding rate");
	})()`
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(nested helper call in return expression) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != `<div class="field">Holding rate</div>` {
		t.Fatalf("Dispatch(nested helper call in return expression) result = %#v, want string <div class=\"field\">Holding rate</div>", result.Value)
	}
}

func TestDispatchSupportsArrayMapCallbackMutationsUpdateOuterLetBindings(t *testing.T) {
	runtime := NewRuntime(nil)

	source := `const rows = [
  { ok: true, label: "valid" },
  { ok: false, label: "invalid" }
];
let calculatedCount = 0;
let errorCount = 0;
const previewRows = rows.map((row) => {
  const notes = [];
  if (!row.ok) {
    notes.push("bad");
    errorCount += 1;
    return { label: row.label, notes };
  }
  calculatedCount += 1;
  return { label: row.label, notes };
});
` + "`" + `${calculatedCount}|${errorCount}|${previewRows.map((row) => row.label + ":" + row.notes.join(";")).join("|")}` + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(array map callback mutations update outer lets) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "1|1|valid:|invalid:bad" {
		t.Fatalf("Dispatch(array map callback mutations update outer lets) result = %#v, want string 1|1|valid:|invalid:bad", result.Value)
	}
}

func TestDispatchSupportsArrayArgumentMutationsInNestedHelpers(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `
		function addError(errors) {
			errors.push("too wide");
			host.echo("inner:" + errors.length);
		}

		function compute() {
			const errors = [];
			addError(errors);
			host.echo("outer:" + errors.length);
			return errors.length + "|" + errors.join(",");
		}

		host.echo(compute());
	`})
	if err != nil {
		t.Fatalf("Dispatch(array argument mutations in nested helpers) error = %v", err)
	}
	if len(host.calls) != 3 {
		t.Fatalf("host calls = %#v, want 3 calls", host.calls)
	}
	if host.calls[0].method != "echo" || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "inner:1" {
		t.Fatalf("host call[0] = %#v, want echo(inner:1)", host.calls[0])
	}
	if host.calls[1].method != "echo" || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "outer:1" {
		t.Fatalf("host call[1] = %#v, want echo(outer:1)", host.calls[1])
	}
	if host.calls[2].method != "echo" || host.calls[2].args[0].Kind != ValueKindString || host.calls[2].args[0].String != "1|too wide" {
		t.Fatalf("host call[2] = %#v, want echo(1|too wide)", host.calls[2])
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "1|too wide" {
		t.Fatalf("Dispatch(array argument mutations in nested helpers) result = %#v, want string 1|too wide", result.Value)
	}
}

func TestDispatchSupportsNestedHelperLocalIndexDoesNotPoisonPlainConstDeclaration(t *testing.T) {
	runtime := NewRuntime(nil)

	source := `(() => {
		const state = { nested: {} };

		function setDeepValue(obj, path, value) {
			const parts = path.split(".");
			let current = obj;
			for (let index = 0; index < parts.length - 1; index += 1) {
				const part = parts[index];
				if (!current[part] || typeof current[part] !== "object") {
					current[part] = {};
				}
				current = current[part];
			}
			current[parts[parts.length - 1]] = value;
		}

		setDeepValue(state.nested, "percent.rateRaw", "20");
		const index = 1;
		return "" + index + ":" + state.nested.percent.rateRaw;
	})()`
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(nested helper local index does not poison plain const declaration) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "1:20" {
		t.Fatalf("Dispatch(nested helper local index does not poison plain const declaration) result = %#v, want string 1:20", result.Value)
	}
}

func TestDispatchSupportsNewTargetInFunctionBodiesAndArrows(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "function outer() { let inspect = function () { return typeof new.target }; let capture = () => typeof new.target; host.setTextContent(\"#out\", `" + "${typeof new.target}:${inspect()}-${capture()}" + "`) }; new outer()"})
	if err != nil {
		t.Fatalf("Dispatch(new.target in function bodies and arrows) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "function:undefined-function" {
		t.Fatalf("host.calls[0].args = %#v, want function:undefined-function", host.calls[0].args)
	}
}

func TestDispatchRejectsNewTargetOutsideFunctionBodiesInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `typeof new.target`})
	if err == nil {
		t.Fatalf("Dispatch(new.target outside function bodies) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(new.target outside function bodies) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(new.target outside function bodies) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsGeneratorFunctionExpressionsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "let sync = function* () { yield \"sync\" }; let asyncGen = async function* () { yield \"async\" }; let syncValue = sync().next().value; let asyncIt = asyncGen(); let asyncFirst = await asyncIt.next(); host.setTextContent(\"#out\", `" + "${syncValue}-${asyncFirst.value}" + "`)"})
	if err != nil {
		t.Fatalf("Dispatch(generator function expressions) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "sync-async" {
		t.Fatalf("host.calls[0].args = %#v, want sync-async", host.calls[0].args)
	}
}

func TestDispatchSupportsDefaultParameterValues(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function choose(value = host.echo("seed")) { return value }; let arrow = (value = host.echo("fresh")) => value; choose(); arrow()`})
	if err != nil {
		t.Fatalf("Dispatch(default parameter values) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "fresh" {
		t.Fatalf("Dispatch(default parameter values) result = %#v, want string fresh", result.Value)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "seed" {
		t.Fatalf("host.calls[0].args[0] = %#v, want string seed", host.calls[0].args[0])
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "fresh" {
		t.Fatalf("host.calls[1].args[0] = %#v, want string fresh", host.calls[1].args[0])
	}
}

func TestDispatchSupportsDefaultParameterValuesInClassMethods(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `class Example { read(value = "seed") { return value } }; let example = new Example(); example.read()`})
	if err != nil {
		t.Fatalf("Dispatch(default parameter values in class methods) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "seed" {
		t.Fatalf("Dispatch(default parameter values in class methods) result = %#v, want string seed", result.Value)
	}
}

func TestDispatchSupportsDestructuringFunctionParameters(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function choose([first, second = first], {kind: label = "box"} = {}) { return first + "-" + second + "-" + label }; function* gather([first, second = first], {kind: label = "box"} = {}) { yield first + "-" + second + "-" + label }; let it = gather([1]); choose([1]) + "|" + it.next().value`})
	if err != nil {
		t.Fatalf("Dispatch(destructuring function parameters) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "1-1-box|1-1-box" {
		t.Fatalf("Dispatch(destructuring function parameters) result = %#v, want string 1-1-box|1-1-box", result.Value)
	}
}

func TestDispatchRejectsInvalidDefaultParameterSyntax(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function broken(value = ) {}`})
	if err == nil {
		t.Fatalf("Dispatch(invalid default parameter syntax) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(invalid default parameter syntax) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(invalid default parameter syntax) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedDestructuringParameterSyntax(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function broken([value =]) {}`})
	if err == nil {
		t.Fatalf("Dispatch(malformed destructuring parameter syntax) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed destructuring parameter syntax) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed destructuring parameter syntax) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsUnsupportedParameterSyntaxAsParseErrorInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function broken(value: label) {}`})
	if err == nil {
		t.Fatalf("Dispatch(unsupported parameter syntax) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(unsupported parameter syntax) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(unsupported parameter syntax) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsReturnInsideTryFinallyInFunctionBodies(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function run() { try { return "boom" } catch (e) { host.setTextContent("#out", "catch") } finally { host.setTextContent("#out", "finally") } }; run()`})
	if err != nil {
		t.Fatalf("Dispatch(return inside try/finally) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "boom" {
		t.Fatalf("Dispatch(return inside try/finally) result = %#v, want string boom", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "finally" {
		t.Fatalf("host.calls[0].args = %#v, want finally branch", host.calls[0].args)
	}
}

func TestDispatchRejectsTopLevelReturnStatements(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `return "boom"`})
	if err == nil {
		t.Fatalf("Dispatch(top-level return) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(top-level return) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(top-level return) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsReturnInsideClassStaticBlocks(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function outer() { class Example { static { return 1 } } }; outer()`})
	if err == nil {
		t.Fatalf("Dispatch(return inside class static block) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(return inside class static block) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(return inside class static block) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsGeneratorFunctionsAndYield(t *testing.T) {
	host := noopHostBindings{}
	env := newClassicJSEnvironment()

	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let make = function* () { let first = 1; yield first; let second = 2; yield second }`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("generator function setup error = %v", err)
	}
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("generator iterator setup error = %v", err)
	}

	first, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("first generator next error = %v", err)
	}
	if first.Kind != ValueKindObject {
		t.Fatalf("first generator next = %#v, want object", first)
	}
	if len(first.Object) != 2 {
		t.Fatalf("first generator next object len = %d, want 2", len(first.Object))
	}
	if first.Object[0].Key != "value" || first.Object[0].Value.Kind != ValueKindNumber || first.Object[0].Value.Number != 1 {
		t.Fatalf("first generator next value = %#v, want number 1", first.Object[0])
	}
	if first.Object[1].Key != "done" || first.Object[1].Value.Kind != ValueKindBool || first.Object[1].Value.Bool {
		t.Fatalf("first generator next done = %#v, want false", first.Object[1])
	}

	second, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("second generator next error = %v", err)
	}
	if second.Kind != ValueKindObject {
		t.Fatalf("second generator next = %#v, want object", second)
	}
	if len(second.Object) != 2 {
		t.Fatalf("second generator next object len = %d, want 2", len(second.Object))
	}
	if second.Object[0].Key != "value" || second.Object[0].Value.Kind != ValueKindNumber || second.Object[0].Value.Number != 2 {
		t.Fatalf("second generator next value = %#v, want number 2", second.Object[0])
	}
	if second.Object[1].Key != "done" || second.Object[1].Value.Kind != ValueKindBool || second.Object[1].Value.Bool {
		t.Fatalf("second generator next done = %#v, want false", second.Object[1])
	}

	third, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("third generator next error = %v", err)
	}
	if third.Kind != ValueKindObject {
		t.Fatalf("third generator next = %#v, want object", third)
	}
	if len(third.Object) != 2 {
		t.Fatalf("third generator next object len = %d, want 2", len(third.Object))
	}
	if third.Object[0].Key != "value" || third.Object[0].Value.Kind != ValueKindUndefined {
		t.Fatalf("third generator next value = %#v, want undefined", third.Object[0])
	}
	if third.Object[1].Key != "done" || third.Object[1].Value.Kind != ValueKindBool || !third.Object[1].Value.Bool {
		t.Fatalf("third generator next done = %#v, want true", third.Object[1])
	}
}

func TestDispatchSupportsGeneratorNextArgumentsAreIgnoredInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { yield \"first\"; yield \"second\" }; let it = spin(); let first = it.next(\"seed\"); let second = it.next(\"ignored\"); let third = it.next(); `" + "${first.value}|${second.value}|${third.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(generator next arguments) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|second|true" {
		t.Fatalf("Dispatch(generator next arguments) result = %#v, want string first|second|true", result.Value)
	}
}

func TestDispatchSupportsGeneratorNextArgumentsOnYieldStarDelegatesInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { yield* { index: 0, next(value) { if (this.index === 0) { this.index = 1; return { value: value === undefined ? \"first\" : value, done: false } }; if (this.index === 1) { this.index = 2; return { value: value === undefined ? \"second\" : value, done: false } }; return { done: true } } }; yield \"done\" }; let it = spin(); let first = it.next(); let second = it.next(\"seed\"); let third = it.next(); let done = it.next(); `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(generator next arguments on yield* delegates) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|seed|done|true" {
		t.Fatalf("Dispatch(generator next arguments on yield* delegates) result = %#v, want string first|seed|done|true", result.Value)
	}
}

func TestDispatchSupportsGeneratorNextArgumentsOnYieldExpressionsInDeclarationInitializersAndAssignmentsInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { let first = yield \"first\"; let value; value = yield first; let box = { value: \"\" }; box.value = yield value; yield `${value}|${box.value}` }; let it = spin(); let one = it.next(); let two = it.next(\"seed\"); let three = it.next(\"tail\"); let four = it.next(\"final\"); let done = it.next(); `" + "${one.value}|${two.value}|${three.value}|${four.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(generator next arguments on yield expressions in declaration initializers and assignments) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|seed|tail|tail|final|true" {
		t.Fatalf("Dispatch(generator next arguments on yield expressions in declaration initializers and assignments) result = %#v, want string first|seed|tail|tail|final|true", result.Value)
	}
}

func TestDispatchRejectsGeneratorNextArgumentsOnUndeclaredAssignmentsInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function* spin() { value = yield "first" }; spin().next()`})
	if err == nil {
		t.Fatalf("Dispatch(generator next arguments on undeclared assignments) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(generator next arguments on undeclared assignments) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(generator next arguments on undeclared assignments) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsGeneratorYieldExpressionsInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { yield \"first\"; (yield \"second\"); yield \"done\" }; let it = spin(); let first = it.next(); let second = it.next(\"ignored\"); let third = it.next(); let fourth = it.next(); `" + "${first.value}|${second.value}|${third.value}|${fourth.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(generator yield expressions) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|second|done|true" {
		t.Fatalf("Dispatch(generator yield expressions) result = %#v, want string first|second|done|true", result.Value)
	}
}

func TestDispatchSupportsGeneratorReturnClosesTheIteratorInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { yield \"first\"; yield \"second\" }; let it = spin(); let first = it.next(); let stopped = it.return(\"done\"); let third = it.next(); `" + "${first.value}|${stopped.value}|${stopped.done}|${third.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(generator return) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first|done|true|true" {
		t.Fatalf("Dispatch(generator return) result = %#v, want string first|done|true|true", result.Value)
	}
}

func TestDispatchRejectsGeneratorThrowInThisBoundedSlice(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function* spin() { yield "first" }; spin().throw("boom")`})
	if err == nil {
		t.Fatalf("Dispatch(generator throw) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(generator throw) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(generator throw) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsGeneratorThrowWithoutArgumentsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function* spin() { yield "first" }; spin().throw()`})
	if err == nil {
		t.Fatalf("Dispatch(generator throw without arguments) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(generator throw without arguments) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(generator throw without arguments) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if got := scriptErr.Error(); !strings.Contains(got, "undefined") {
		t.Fatalf("Dispatch(generator throw without arguments) error text = %q, want undefined", got)
	}
}

func TestDispatchSupportsNamedGeneratorFunctionsAndSelfBinding(t *testing.T) {
	host := noopHostBindings{}
	env := newClassicJSEnvironment()

	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let make = function* spin() { yield spin; yield 1 }`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("named generator function setup error = %v", err)
	}
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("named generator iterator setup error = %v", err)
	}

	first, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("first named generator next error = %v", err)
	}
	if first.Kind != ValueKindObject {
		t.Fatalf("first named generator next = %#v, want object", first)
	}
	if len(first.Object) != 2 {
		t.Fatalf("first named generator next object len = %d, want 2", len(first.Object))
	}
	if first.Object[0].Key != "value" || first.Object[0].Value.Kind != ValueKindFunction || first.Object[0].Value.Function == nil {
		t.Fatalf("first named generator next value = %#v, want function", first.Object[0])
	}
	if first.Object[1].Key != "done" || first.Object[1].Value.Kind != ValueKindBool || first.Object[1].Value.Bool {
		t.Fatalf("first named generator next done = %#v, want false", first.Object[1])
	}

	second, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("second named generator next error = %v", err)
	}
	if second.Kind != ValueKindObject {
		t.Fatalf("second named generator next = %#v, want object", second)
	}
	if len(second.Object) != 2 {
		t.Fatalf("second named generator next object len = %d, want 2", len(second.Object))
	}
	if second.Object[0].Key != "value" || second.Object[0].Value.Kind != ValueKindNumber || second.Object[0].Value.Number != 1 {
		t.Fatalf("second named generator next value = %#v, want number 1", second.Object[0])
	}
	if second.Object[1].Key != "done" || second.Object[1].Value.Kind != ValueKindBool || second.Object[1].Value.Bool {
		t.Fatalf("second named generator next done = %#v, want false", second.Object[1])
	}
}

func TestDispatchSupportsGeneratorDelegationWithYieldStar(t *testing.T) {
	host := noopHostBindings{}
	env := newClassicJSEnvironment()

	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let make = function* () { yield* [1, 2]; yield 3 }`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("generator delegation setup error = %v", err)
	}
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("generator delegation iterator setup error = %v", err)
	}

	first, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("first delegation next error = %v", err)
	}
	if first.Kind != ValueKindObject {
		t.Fatalf("first delegation next = %#v, want object", first)
	}
	if len(first.Object) != 2 {
		t.Fatalf("first delegation next object len = %d, want 2", len(first.Object))
	}
	if first.Object[0].Key != "value" || first.Object[0].Value.Kind != ValueKindNumber || first.Object[0].Value.Number != 1 {
		t.Fatalf("first delegation next value = %#v, want number 1", first.Object[0])
	}
	if first.Object[1].Key != "done" || first.Object[1].Value.Kind != ValueKindBool || first.Object[1].Value.Bool {
		t.Fatalf("first delegation next done = %#v, want false", first.Object[1])
	}

	second, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("second delegation next error = %v", err)
	}
	if second.Kind != ValueKindObject {
		t.Fatalf("second delegation next = %#v, want object", second)
	}
	if len(second.Object) != 2 {
		t.Fatalf("second delegation next object len = %d, want 2", len(second.Object))
	}
	if second.Object[0].Key != "value" || second.Object[0].Value.Kind != ValueKindNumber || second.Object[0].Value.Number != 2 {
		t.Fatalf("second delegation next value = %#v, want number 2", second.Object[0])
	}
	if second.Object[1].Key != "done" || second.Object[1].Value.Kind != ValueKindBool || second.Object[1].Value.Bool {
		t.Fatalf("second delegation next done = %#v, want false", second.Object[1])
	}

	third, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("third delegation next error = %v", err)
	}
	if third.Kind != ValueKindObject {
		t.Fatalf("third delegation next = %#v, want object", third)
	}
	if len(third.Object) != 2 {
		t.Fatalf("third delegation next object len = %d, want 2", len(third.Object))
	}
	if third.Object[0].Key != "value" || third.Object[0].Value.Kind != ValueKindNumber || third.Object[0].Value.Number != 3 {
		t.Fatalf("third delegation next value = %#v, want number 3", third.Object[0])
	}
	if third.Object[1].Key != "done" || third.Object[1].Value.Kind != ValueKindBool || third.Object[1].Value.Bool {
		t.Fatalf("third delegation next done = %#v, want false", third.Object[1])
	}

	fourth, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("fourth delegation next error = %v", err)
	}
	if fourth.Kind != ValueKindObject {
		t.Fatalf("fourth delegation next = %#v, want object", fourth)
	}
	if len(fourth.Object) != 2 {
		t.Fatalf("fourth delegation next object len = %d, want 2", len(fourth.Object))
	}
	if fourth.Object[0].Key != "value" || fourth.Object[0].Value.Kind != ValueKindUndefined {
		t.Fatalf("fourth delegation next value = %#v, want undefined", fourth.Object[0])
	}
	if fourth.Object[1].Key != "done" || fourth.Object[1].Value.Kind != ValueKindBool || !fourth.Object[1].Value.Bool {
		t.Fatalf("fourth delegation next done = %#v, want true", fourth.Object[1])
	}
}

func TestDispatchSupportsGeneratorDelegationWithYieldStarOnString(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { yield* \"go\"; yield \"done\" }; let it = spin(); let first = it.next(); let second = it.next(); let third = it.next(); let done = it.next(); `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(generator delegation on string) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "g|o|done|true" {
		t.Fatalf("Dispatch(generator delegation on string) result = %#v, want string g|o|done|true", result.Value)
	}
}

func TestDispatchSupportsNestedGeneratorYieldDelegationOnString(t *testing.T) {
	runtime := NewRuntime(nil)

	source := "function* spin() { if (true) { yield* \"go\"; }; yield \"done\" }; let it = spin(); let first = it.next(); let second = it.next(); let third = it.next(); let done = it.next(); `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(nested generator delegation on string) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "g|o|done|true" {
		t.Fatalf("Dispatch(nested generator delegation on string) result = %#v, want string g|o|done|true", result.Value)
	}
}

func TestDispatchRejectsNestedGeneratorYieldDelegationOnScalarInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function* spin() { if (true) { yield* 1; }; }; spin().next()`})
	if err == nil {
		t.Fatalf("Dispatch(nested generator delegation on scalar) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(nested generator delegation on scalar) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(nested generator delegation on scalar) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsNestedYieldInIfBranches(t *testing.T) {
	host := noopHostBindings{}
	env := newClassicJSEnvironment()

	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let make = function* () { if (true) { if (true) { yield 1 } }; yield 2 }`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("nested if yield setup error = %v", err)
	}
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("nested if yield iterator setup error = %v", err)
	}

	first, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("first nested if yield next error = %v", err)
	}
	if first.Kind != ValueKindObject {
		t.Fatalf("first nested if yield next = %#v, want object", first)
	}
	if len(first.Object) != 2 {
		t.Fatalf("first nested if yield next object len = %d, want 2", len(first.Object))
	}
	if first.Object[0].Key != "value" || first.Object[0].Value.Kind != ValueKindNumber || first.Object[0].Value.Number != 1 {
		t.Fatalf("first nested if yield next value = %#v, want number 1", first.Object[0])
	}
	if first.Object[1].Key != "done" || first.Object[1].Value.Kind != ValueKindBool || first.Object[1].Value.Bool {
		t.Fatalf("first nested if yield next done = %#v, want false", first.Object[1])
	}

	second, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err != nil {
		t.Fatalf("second nested if yield next error = %v", err)
	}
	if second.Kind != ValueKindObject {
		t.Fatalf("second nested if yield next = %#v, want object", second)
	}
	if len(second.Object) != 2 {
		t.Fatalf("second nested if yield next object len = %d, want 2", len(second.Object))
	}
	if second.Object[0].Key != "value" || second.Object[0].Value.Kind != ValueKindNumber || second.Object[0].Value.Number != 2 {
		t.Fatalf("second nested if yield next value = %#v, want number 2", second.Object[0])
	}
	if second.Object[1].Key != "done" || second.Object[1].Value.Kind != ValueKindBool || second.Object[1].Value.Bool {
		t.Fatalf("second nested if yield next done = %#v, want false", second.Object[1])
	}
}

func TestDispatchRejectsReservedGeneratorFunctionNames(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let make = function* yield() { yield 1 }`})
	if err == nil {
		t.Fatalf("Dispatch(reserved generator function name) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(reserved generator function name) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(reserved generator function name) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsYieldStarOnScalarValues(t *testing.T) {
	host := &echoHost{}

	env := newClassicJSEnvironment()
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let make = function* () { yield* 1 }`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("generator delegation setup error = %v", err)
	}
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("generator delegation iterator setup error = %v", err)
	}

	_, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
	if err == nil {
		t.Fatalf("generator delegation next error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("generator delegation next error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("generator delegation next error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsYieldInsideLoopBodies(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "while",
			source: `let make = function* () { while (true) { yield 1; yield 2 } }`,
		},
		{
			name:   "do_while",
			source: `let make = function* () { do { yield 1; yield 2 } while (true) }`,
		},
		{
			name:   "for",
			source: `let make = function* () { for (;;) { yield 1; yield 2 } }`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := noopHostBindings{}
			env := newClassicJSEnvironment()

			if _, err := evalClassicJSStatementWithEnvAndAllowAwait(tc.source, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
				t.Fatalf("loop yield setup error = %v", err)
			}
			if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
				t.Fatalf("loop yield iterator setup error = %v", err)
			}

			want := []int{1, 2, 1}
			for i, wantValue := range want {
				got, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
				if err != nil {
					t.Fatalf("loop yield next %d error = %v", i+1, err)
				}
				if got.Kind != ValueKindObject {
					t.Fatalf("loop yield next %d = %#v, want object", i+1, got)
				}
				if len(got.Object) != 2 {
					t.Fatalf("loop yield next %d object len = %d, want 2", i+1, len(got.Object))
				}
				if got.Object[0].Key != "value" || got.Object[0].Value.Kind != ValueKindNumber || got.Object[0].Value.Number != float64(wantValue) {
					t.Fatalf("loop yield next %d value = %#v, want number %d", i+1, got.Object[0], wantValue)
				}
				if got.Object[1].Key != "done" || got.Object[1].Value.Kind != ValueKindBool || got.Object[1].Value.Bool {
					t.Fatalf("loop yield next %d done = %#v, want false", i+1, got.Object[1])
				}
			}
		})
	}
}

func TestDispatchSupportsYieldInsideSwitchClauses(t *testing.T) {
	host := noopHostBindings{}
	env := newClassicJSEnvironment()

	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let make = function* () { switch ("b") { case "a": yield 1; break; case "b": yield 2; yield 3; break; default: yield 4 }; yield 5 }`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("switch clause yield setup error = %v", err)
	}
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("switch clause yield iterator setup error = %v", err)
	}

	want := []int{2, 3, 5}
	for i, wantValue := range want {
		got, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
		if err != nil {
			t.Fatalf("switch clause yield next %d error = %v", i+1, err)
		}
		if got.Kind != ValueKindObject {
			t.Fatalf("switch clause yield next %d = %#v, want object", i+1, got)
		}
		if len(got.Object) != 2 {
			t.Fatalf("switch clause yield next %d object len = %d, want 2", i+1, len(got.Object))
		}
		if got.Object[0].Key != "value" || got.Object[0].Value.Kind != ValueKindNumber || got.Object[0].Value.Number != float64(wantValue) {
			t.Fatalf("switch clause yield next %d value = %#v, want number %d", i+1, got.Object[0], wantValue)
		}
		if got.Object[1].Key != "done" || got.Object[1].Value.Kind != ValueKindBool || got.Object[1].Value.Bool {
			t.Fatalf("switch clause yield next %d done = %#v, want false", i+1, got.Object[1])
		}
	}
}

func TestDispatchSupportsYieldInsideTryCatchFinallyBlocks(t *testing.T) {
	host := &fakeHost{
		errs: map[string]error{
			"fail": errors.New("boom"),
		},
	}
	env := newClassicJSEnvironment()

	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let make = function* () { try { yield 1; host.fail(); yield 2 } catch (e) { yield e; yield 3 } finally { yield 4; yield 5 }; yield 6 }`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("try block yield setup error = %v", err)
	}
	if _, err := evalClassicJSStatementWithEnvAndAllowAwait(`let it = make()`, host, env, DefaultRuntimeConfig().StepLimit, false); err != nil {
		t.Fatalf("try block yield iterator setup error = %v", err)
	}

	want := []struct {
		kind   ValueKind
		number float64
		str    string
	}{
		{kind: ValueKindNumber, number: 1},
		{kind: ValueKindString, str: "host: boom"},
		{kind: ValueKindNumber, number: 3},
		{kind: ValueKindNumber, number: 4},
		{kind: ValueKindNumber, number: 5},
		{kind: ValueKindNumber, number: 6},
	}
	for i, wantValue := range want {
		got, err := evalClassicJSExpressionWithEnvAndAllowAwait(`it.next()`, host, env, DefaultRuntimeConfig().StepLimit, false)
		if err != nil {
			t.Fatalf("try block yield next %d error = %v", i+1, err)
		}
		if got.Kind != ValueKindObject {
			t.Fatalf("try block yield next %d = %#v, want object", i+1, got)
		}
		if len(got.Object) != 2 {
			t.Fatalf("try block yield next %d object len = %d, want 2", i+1, len(got.Object))
		}
		if got.Object[0].Key != "value" {
			t.Fatalf("try block yield next %d value key = %q, want value", i+1, got.Object[0].Key)
		}
		if got.Object[0].Value.Kind != wantValue.kind {
			t.Fatalf("try block yield next %d value kind = %q, want %q", i+1, got.Object[0].Value.Kind, wantValue.kind)
		}
		switch wantValue.kind {
		case ValueKindNumber:
			if got.Object[0].Value.Number != wantValue.number {
				t.Fatalf("try block yield next %d value = %#v, want number %v", i+1, got.Object[0].Value, wantValue.number)
			}
		case ValueKindString:
			if got.Object[0].Value.String != wantValue.str {
				t.Fatalf("try block yield next %d value = %#v, want string %q", i+1, got.Object[0].Value, wantValue.str)
			}
		default:
			t.Fatalf("try block yield next %d unsupported expectation kind %q", i+1, wantValue.kind)
		}
		if got.Object[1].Key != "done" || got.Object[1].Value.Kind != ValueKindBool || got.Object[1].Value.Bool {
			t.Fatalf("try block yield next %d done = %#v, want false", i+1, got.Object[1])
		}
	}
}

func TestDispatchRejectsArrowFunctionReservedParameterNames(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let bad = (host) => host; host.echo(bad("ok"))`})
	if err == nil {
		t.Fatalf("Dispatch(arrow function reserved parameter) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(arrow function reserved parameter) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(arrow function reserved parameter) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsTopLevelAwaitInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `await host.echo("boom")`})
	if err != nil {
		t.Fatalf("Dispatch(top-level await) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "boom" {
		t.Fatalf("host.calls[0].args = %#v, want echo(boom)", host.calls[0].args)
	}
}

func TestDispatchSupportsExportDeclarationsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	moduleExports := map[string]Value{}
	got, err := runtime.Dispatch(DispatchRequest{Source: `export var value = host.echo("boom"); export { value as alias }; export default value; value`, ModuleExports: moduleExports})
	if err != nil {
		t.Fatalf("Dispatch(export declarations) error = %v", err)
	}
	if got.Value.Kind != ValueKindString || got.Value.String != "boom" {
		t.Fatalf("Dispatch(export declarations) result = %#v, want string boom", got.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "boom" {
		t.Fatalf("host.calls[0].args = %#v, want echo(boom)", host.calls[0].args)
	}
	if got := moduleExports["value"]; got.Kind != ValueKindString || got.String != "boom" {
		t.Fatalf("moduleExports[\"value\"] = %#v, want string boom", got)
	}
	if got := moduleExports["alias"]; got.Kind != ValueKindString || got.String != "boom" {
		t.Fatalf("moduleExports[\"alias\"] = %#v, want string boom", got)
	}
	if got := moduleExports["default"]; got.Kind != ValueKindString || got.String != "boom" {
		t.Fatalf("moduleExports[\"default\"] = %#v, want string boom", got)
	}
}

func TestDispatchSupportsNamespaceReExportsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}
	moduleExports := map[string]Value{}
	got, err := runtime.Dispatch(DispatchRequest{Source: `export * as ns from "math" with { type: "json" }; 1`, Bindings: bindings, ModuleExports: moduleExports})
	if err != nil {
		t.Fatalf("Dispatch(namespace re-exports) error = %v", err)
	}
	if got.Value.Kind != ValueKindNumber || got.Value.Number != 1 {
		t.Fatalf("Dispatch(namespace re-exports) result = %#v, want number 1", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
	if got := moduleExports["ns"]; got.Kind != ValueKindObject {
		t.Fatalf("moduleExports[\"ns\"] = %#v, want object", got)
	}
	if got, ok := lookupObjectProperty(moduleExports["ns"].Object, "default"); !ok || got.Kind != ValueKindString || got.String != "seeded" {
		t.Fatalf("moduleExports[\"ns\"].default = %#v, want string seeded", got)
	}
	if got, ok := lookupObjectProperty(moduleExports["ns"].Object, "value"); !ok || got.Kind != ValueKindNumber || got.Number != 7 {
		t.Fatalf("moduleExports[\"ns\"].value = %#v, want number 7", got)
	}
}

func TestDispatchSupportsStarReExportsWithImportAttributesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}
	moduleExports := map[string]Value{}
	got, err := runtime.Dispatch(DispatchRequest{Source: `export * from "math" with { type: "json" }; 1`, Bindings: bindings, ModuleExports: moduleExports})
	if err != nil {
		t.Fatalf("Dispatch(star re-exports with attributes) error = %v", err)
	}
	if got.Value.Kind != ValueKindNumber || got.Value.Number != 1 {
		t.Fatalf("Dispatch(star re-exports with attributes) result = %#v, want number 1", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
	if _, ok := moduleExports["default"]; ok {
		t.Fatalf("moduleExports[\"default\"] = %#v, want default export omitted", moduleExports["default"])
	}
	if got := moduleExports["value"]; got.Kind != ValueKindNumber || got.Number != 7 {
		t.Fatalf("moduleExports[\"value\"] = %#v, want number 7", got)
	}
}

func TestDispatchSupportsSpecifierReExportsWithImportAttributesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}
	moduleExports := map[string]Value{}
	got, err := runtime.Dispatch(DispatchRequest{Source: `export { default as mirror, value as copy } from "math" with { type: "json" }; 1`, Bindings: bindings, ModuleExports: moduleExports})
	if err != nil {
		t.Fatalf("Dispatch(specifier re-exports with attributes) error = %v", err)
	}
	if got.Value.Kind != ValueKindNumber || got.Value.Number != 1 {
		t.Fatalf("Dispatch(specifier re-exports with attributes) result = %#v, want number 1", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
	if got := moduleExports["mirror"]; got.Kind != ValueKindString || got.String != "seeded" {
		t.Fatalf("moduleExports[\"mirror\"] = %#v, want string seeded", got)
	}
	if got := moduleExports["copy"]; got.Kind != ValueKindNumber || got.Number != 7 {
		t.Fatalf("moduleExports[\"copy\"] = %#v, want number 7", got)
	}
}

func TestDispatchSupportsExportDefaultClassDeclarationsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	moduleExports := map[string]Value{}
	got, err := runtime.Dispatch(DispatchRequest{Source: `export default class Box { static value = host.echo("boom"); };`, ModuleExports: moduleExports})
	if err != nil {
		t.Fatalf("Dispatch(export default class) error = %v", err)
	}
	if got.Value.Kind != ValueKindObject {
		t.Fatalf("Dispatch(export default class) result kind = %q, want object", got.Value.Kind)
	}
	if got, ok := lookupObjectProperty(got.Value.Object, "value"); !ok || got.Kind != ValueKindString || got.String != "boom" {
		t.Fatalf("Dispatch(export default class) result value = %#v, want string boom", got)
	}
	if got := moduleExports["default"]; got.Kind != ValueKindObject {
		t.Fatalf("moduleExports[\"default\"] = %#v, want object", got)
	}
	if got, ok := lookupObjectProperty(moduleExports["default"].Object, "value"); !ok || got.Kind != ValueKindString || got.String != "boom" {
		t.Fatalf("moduleExports[\"default\"].value = %#v, want string boom", got)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
}

func TestDispatchSupportsAnonymousDefaultFunctionExportsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	moduleExports := map[string]Value{}
	got, err := runtime.Dispatch(DispatchRequest{Source: `export default function () { return host.echo("boom"); };`, ModuleExports: moduleExports})
	if err != nil {
		t.Fatalf("Dispatch(export default function) error = %v", err)
	}
	if got.Value.Kind != ValueKindFunction {
		t.Fatalf("Dispatch(export default function) result kind = %q, want function", got.Value.Kind)
	}
	if got := moduleExports["default"]; got.Kind != ValueKindFunction {
		t.Fatalf("moduleExports[\"default\"] = %#v, want function", got)
	}

	call, err := runtime.Dispatch(DispatchRequest{Source: `import fn from "self"; fn()`, Bindings: map[string]Value{"self": ObjectValue([]ObjectEntry{{Key: "default", Value: moduleExports["default"]}})}})
	if err != nil {
		t.Fatalf("Dispatch(import default anonymous function export) error = %v", err)
	}
	if call.Value.Kind != ValueKindString || call.Value.String != "boom" {
		t.Fatalf("Dispatch(import default anonymous function export) result = %#v, want string boom", call.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
}

func TestDispatchSupportsDefaultModuleSpecifierAliasesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}
	moduleExports := map[string]Value{}
	got, err := runtime.Dispatch(DispatchRequest{
		Source:        `import { default as seeded } from "math"; export { value as default } from "math"; seeded`,
		Bindings:      bindings,
		ModuleExports: moduleExports,
	})
	if err != nil {
		t.Fatalf("Dispatch(default module specifier aliases) error = %v", err)
	}
	if got.Value.Kind != ValueKindString || got.Value.String != "seeded" {
		t.Fatalf("Dispatch(default module specifier aliases) result = %#v, want string seeded", got.Value)
	}
	if got := moduleExports["default"]; got.Kind != ValueKindNumber || got.Number != 7 {
		t.Fatalf("moduleExports[\"default\"] = %#v, want number 7", got)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsExportDefaultConstAndLetDeclarationsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	for _, source := range []string{`export default const value = 1`, `export default let value = 1`} {
		_, err := runtime.Dispatch(DispatchRequest{Source: source, ModuleExports: map[string]Value{}})
		if err == nil {
			t.Fatalf("Dispatch(%s) error = nil, want parse error", source)
		}
		scriptErr, ok := err.(Error)
		if !ok {
			t.Fatalf("Dispatch(%s) error type = %T, want script.Error", source, err)
		}
		if scriptErr.Kind != ErrorKindParse {
			t.Fatalf("Dispatch(%s) error kind = %q, want %q", source, scriptErr.Kind, ErrorKindParse)
		}
	}
}

func TestDispatchSupportsImportDeclarationsFromSeededModuleBindings(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `import seeded, { value as alias } from "math"; alias`, Bindings: bindings})
	if err != nil {
		t.Fatalf("Dispatch(import declarations) error = %v", err)
	}
	if got.Value.Kind != ValueKindNumber || got.Value.Number != 7 {
		t.Fatalf("Dispatch(import declarations) result = %#v, want number 7", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsDefaultAndNamespaceImportsFromSeededModuleBindings(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `import seeded, * as ns from "math"; seeded + "-" + ns.value`, Bindings: bindings})
	if err != nil {
		t.Fatalf("Dispatch(default + namespace import) error = %v", err)
	}
	if got.Value.Kind != ValueKindString || got.Value.String != "seeded-7" {
		t.Fatalf("Dispatch(default + namespace import) result = %#v, want string seeded-7", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsDefaultAndNamespaceImportsWithImportAttributesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `import seeded, * as ns from "math" with { type: "json" }; seeded + "-" + ns.value`, Bindings: bindings})
	if err != nil {
		t.Fatalf("Dispatch(default + namespace import with attributes) error = %v", err)
	}
	if got.Value.Kind != ValueKindString || got.Value.String != "seeded-7" {
		t.Fatalf("Dispatch(default + namespace import with attributes) result = %#v, want string seeded-7", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsImportAttributesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `import { default as seeded } from "math" with { type: "json" }; seeded`, Bindings: bindings})
	if err != nil {
		t.Fatalf("Dispatch(import attributes) error = %v", err)
	}
	if got.Value.Kind != ValueKindString || got.Value.String != "seeded" {
		t.Fatalf("Dispatch(import attributes) result = %#v, want string seeded", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsDefaultAndNamespaceImportsWithoutDefaultExportInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{
		Source: `import seeded, * as ns from "math"; 1`,
		Bindings: map[string]Value{
			"math": ObjectValue([]ObjectEntry{
				{Key: "value", Value: NumberValue(7)},
			}),
		},
	})
	if err == nil {
		t.Fatalf("Dispatch(default + namespace import missing default export) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(default + namespace import missing default export) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(default + namespace import missing default export) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsBareDefaultImportSpecifierInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{
		Source: `import { default } from "math"; 1`,
		Bindings: map[string]Value{
			"math": ObjectValue([]ObjectEntry{
				{Key: "default", Value: StringValue("seeded")},
			}),
		},
	})
	if err == nil {
		t.Fatalf("Dispatch(bare default import specifier) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(bare default import specifier) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(bare default import specifier) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsDynamicImportExpressionsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `let moduleName = "ma" + "th"; await import(moduleName)`, Bindings: bindings})
	if err != nil {
		t.Fatalf("Dispatch(dynamic import) error = %v", err)
	}
	if got.Value.Kind != ValueKindObject {
		t.Fatalf("Dispatch(dynamic import) result kind = %q, want object", got.Value.Kind)
	}
	if got, ok := lookupObjectProperty(got.Value.Object, "value"); !ok || got.Kind != ValueKindNumber || got.Number != 7 {
		t.Fatalf("Dispatch(dynamic import) result value = %#v, want number 7", got)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsBareDynamicImportExpressionsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `let moduleName = "ma" + "th"; import(moduleName)`, Bindings: bindings})
	if err != nil {
		t.Fatalf("Dispatch(bare dynamic import) error = %v", err)
	}
	if got.Value.Kind != ValueKindPromise {
		t.Fatalf("Dispatch(bare dynamic import) result kind = %q, want %q", got.Value.Kind, ValueKindPromise)
	}
	if got.Value.Promise == nil {
		t.Fatalf("Dispatch(bare dynamic import) promise = nil, want module object")
	}
	if got.Value.Promise.Kind != ValueKindObject {
		t.Fatalf("Dispatch(bare dynamic import) promise kind = %q, want object", got.Value.Promise.Kind)
	}
	if value, ok := lookupObjectProperty(got.Value.Promise.Object, "value"); !ok || value.Kind != ValueKindNumber || value.Number != 7 {
		t.Fatalf("Dispatch(bare dynamic import) promise value = %#v, want number 7", value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsDynamicImportOptionsObjectInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `let moduleName = "ma" + "th"; let ns = await import(moduleName, { with: { type: "json" } }); ns.value`, Bindings: bindings})
	if err != nil {
		t.Fatalf("Dispatch(dynamic import options object) error = %v", err)
	}
	if got.Value.Kind != ValueKindNumber || got.Value.Number != 7 {
		t.Fatalf("Dispatch(dynamic import options object) result = %#v, want number 7", got.Value)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsDynamicImportOptionsNonObjectInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `await import("math", 1)`})
	if err == nil {
		t.Fatalf("Dispatch(dynamic import options non-object) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(dynamic import options non-object) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(dynamic import options non-object) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsNamespaceReExportsWithoutAliasInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `export * as ns;`})
	if err == nil {
		t.Fatalf("Dispatch(namespace re-export) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(namespace re-export) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(namespace re-export) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsImportMetaUrlInModuleContext(t *testing.T) {
	runtime := NewRuntime(nil)

	got, err := runtime.Dispatch(DispatchRequest{
		Source: `import.meta.url`,
		Bindings: map[string]Value{
			ClassicJSModuleMetaURLBindingName: StringValue("inline-module:math"),
		},
	})
	if err != nil {
		t.Fatalf("Dispatch(import.meta.url) error = %v", err)
	}
	if got.Value.Kind != ValueKindString || got.Value.String != "inline-module:math" {
		t.Fatalf("Dispatch(import.meta.url) result = %#v, want string inline-module:math", got.Value)
	}
}

func TestDispatchSupportsBareImportMetaInModuleContext(t *testing.T) {
	runtime := NewRuntime(nil)

	got, err := runtime.Dispatch(DispatchRequest{
		Source: `import.meta`,
		Bindings: map[string]Value{
			ClassicJSModuleMetaURLBindingName: StringValue("inline-module:math"),
		},
	})
	if err != nil {
		t.Fatalf("Dispatch(import.meta) error = %v", err)
	}
	if got.Value.Kind != ValueKindObject {
		t.Fatalf("Dispatch(import.meta) result kind = %q, want object", got.Value.Kind)
	}
	if got, ok := lookupObjectProperty(got.Value.Object, "url"); !ok || got.Kind != ValueKindString || got.String != "inline-module:math" {
		t.Fatalf("Dispatch(import.meta) result url = %#v, want string inline-module:math", got)
	}
}

func TestDispatchRejectsImportMetaOutsideModuleContext(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `import.meta.url`})
	if err == nil {
		t.Fatalf("Dispatch(import.meta.url outside module) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(import.meta.url outside module) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(import.meta.url outside module) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsImportDeclarationsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `import { echo } from "./mod.js"`})
	if err == nil {
		t.Fatalf("Dispatch(import declaration) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(import declaration) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(import declaration) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsMissingDynamicImportModuleInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `await import("missing")`})
	if err == nil {
		t.Fatalf("Dispatch(dynamic import) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(dynamic import) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(dynamic import) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsYieldOutsideGeneratorFunctions(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `yield 1`})
	if err == nil {
		t.Fatalf("Dispatch(yield outside generator) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(yield outside generator) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(yield outside generator) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsSpreadOnScalarValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo([1, ...1])`})
	if err == nil {
		t.Fatalf("Dispatch(spread on scalar) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(spread on scalar) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(spread on scalar) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsObjectSpreadOnScalarValuesAsNoOp(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let spread = {...1}; typeof spread["0"]`})
	if err != nil {
		t.Fatalf("Dispatch(object spread on scalar as no-op) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "undefined" {
		t.Fatalf("Dispatch(object spread on scalar as no-op) value = %#v, want string undefined", result.Value)
	}
}

func TestDispatchRejectsMalformedArrayLiteralElisionsWithSpread(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let values = [1, , ...];`})
	if err == nil {
		t.Fatalf("Dispatch(malformed array literal elisions with spread) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed array literal elisions with spread) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed array literal elisions with spread) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedIteratorLikeArraySpread(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[...({ next() { return { value: "seed" } } })]`})
	if err == nil {
		t.Fatalf("Dispatch(malformed iterator-like array spread) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed iterator-like array spread) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(malformed iterator-like array spread) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsCallArgumentSpreadOnIteratorLikeObjects(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `function* values() { yield "first"; yield "second" }; host.echo(...values())`})
	if err != nil {
		t.Fatalf("Dispatch(call argument spread on iterator-like object) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "first" {
		t.Fatalf("Dispatch(call argument spread on iterator-like object) value = %#v, want string first", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "first" {
		t.Fatalf("host call arg[0] = %#v, want string first", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "second" {
		t.Fatalf("host call arg[1] = %#v, want string second", call.args[1])
	}
}

func TestDispatchRejectsMalformedIteratorLikeCallArgumentSpread(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(...({ next() { return { value: "seed" } } }))`})
	if err == nil {
		t.Fatalf("Dispatch(malformed iterator-like call argument spread) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed iterator-like call argument spread) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(malformed iterator-like call argument spread) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsTypeofOperatorInClassicJS(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{name: "undefined", expr: "typeof undefined", want: "undefined"},
		{name: "null", expr: "typeof null", want: "object"},
		{name: "string", expr: `typeof "x"`, want: "string"},
		{name: "number", expr: "typeof 1", want: "number"},
		{name: "bigint", expr: "typeof 1n", want: "bigint"},
		{name: "boolean", expr: "typeof true", want: "boolean"},
		{name: "host", expr: "typeof host", want: "object"},
		{name: "host_method", expr: "typeof host.echo", want: "function"},
		{name: "array", expr: "typeof [1, 2]", want: "object"},
		{name: "object", expr: "typeof ({ x: 1 })", want: "object"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runtime := NewRuntime(nil)

			got, err := runtime.Dispatch(DispatchRequest{Source: tc.expr})
			if err != nil {
				t.Fatalf("Dispatch(typeof operator %s) error = %v", tc.name, err)
			}
			if got.Value.Kind != ValueKindString || got.Value.String != tc.want {
				t.Fatalf("Dispatch(typeof operator %s) = %#v, want %q", tc.name, got.Value, tc.want)
			}
		})
	}
}

func TestDispatchSupportsInOperatorInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { set value(next) { this._value = next }, nested: { count: 1 }, items: [1, 2] }; host.echo("value" in obj, "missing" in obj, "count" in obj.nested, 0 in obj.items, 2 in obj.items, "length" in obj.items)`})
	if err != nil {
		t.Fatalf("Dispatch(in operator) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(in operator) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 6 {
		t.Fatalf("host call args len = %d, want 6", len(call.args))
	}
	wantBools := []bool{true, false, true, true, false, true}
	for i, want := range wantBools {
		if call.args[i].Kind != ValueKindBool || call.args[i].Bool != want {
			t.Fatalf("host call arg[%d] = %#v, want bool %v", i, call.args[i], want)
		}
	}
}

func TestDispatchSupportsInstanceofOperatorInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `class Base {}; class Derived extends Base {}; let base = new Base(); let derived = new Derived(); let plain = {}; host.echo(base instanceof Base, derived instanceof Base, derived instanceof Derived, plain instanceof Base)`})
	if err != nil {
		t.Fatalf("Dispatch(instanceof operator) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || !result.Value.Bool {
		t.Fatalf("Dispatch(instanceof operator) result = %#v, want bool true", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	wantBools := []bool{true, true, true, false}
	for i, want := range wantBools {
		if call.args[i].Kind != ValueKindBool || call.args[i].Bool != want {
			t.Fatalf("host call arg[%d] = %#v, want bool %v", i, call.args[i], want)
		}
	}
}

func TestDispatchSupportsConditionalOperatorInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(({} ? "object" : "no") + "-" + ([] ? "array" : "no") + "-" + (0 ? "zero" : "falsy") + "-" + ("" ? "string" : "falsy") + "-" + (false ? "bool" : "falsy") + "-" + (false ? "left" : true ? "middle" : "right"))`})
	if err != nil {
		t.Fatalf("Dispatch(conditional operator) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "object-array-falsy-falsy-falsy-middle" {
		t.Fatalf("Dispatch(conditional operator) value = %#v, want string result", result.Value)
	}
	if len(host.calls) != 1 || host.calls[0].method != "echo" {
		t.Fatalf("host calls = %#v, want one echo call", host.calls)
	}
}

func TestDispatchSupportsConditionalOperatorWithObjectLiteralBranchesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `const viewState = false ? { showVisibleWhitespace: true, showDiff: true } : { showVisibleWhitespace: false, showDiff: false }; host.echo(viewState.showVisibleWhitespace, viewState.showDiff)`})
	if err != nil {
		t.Fatalf("Dispatch(conditional operator with object literal branches) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || result.Value.Bool {
		t.Fatalf("Dispatch(conditional operator with object literal branches) result = %#v, want bool false", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	for i, arg := range call.args {
		if arg.Kind != ValueKindBool || arg.Bool {
			t.Fatalf("host call arg[%d] = %#v, want bool false", i, arg)
		}
	}
}

func TestDispatchSupportsNestedConditionalOperatorInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(true ? false ? "inner" : "middle" : "outer")`})
	if err != nil {
		t.Fatalf("Dispatch(nested conditional operator) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "middle" {
		t.Fatalf("Dispatch(nested conditional operator) value = %#v, want string middle", result.Value)
	}
	if len(host.calls) != 1 || host.calls[0].method != "echo" {
		t.Fatalf("host calls = %#v, want one echo call", host.calls)
	}
}

func TestDispatchSupportsEqualityComparisonsOnBoundedValuesInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { value: 1 }; let alias = obj; let arr = [1, 2]; let arrAlias = arr; host.echo(obj === alias, obj === { value: 1 }, obj == alias, obj == { value: 1 }, arr === arrAlias, arr == arrAlias, 1 == "1", 1 === "1", 1 != "1", 1 !== "1", null == undefined, null === undefined)`})
	if err != nil {
		t.Fatalf("Dispatch(equality comparisons on bounded values) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || !result.Value.Bool {
		t.Fatalf("Dispatch(equality comparisons on bounded values) result = %#v, want bool true", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 12 {
		t.Fatalf("host call args len = %d, want 12", len(call.args))
	}
	wantBools := []bool{true, false, true, false, true, true, true, false, false, true, true, false}
	for i, want := range wantBools {
		if call.args[i].Kind != ValueKindBool || call.args[i].Bool != want {
			t.Fatalf("host call arg[%d] = %#v, want bool %v", i, call.args[i], want)
		}
	}
}

func TestDispatchSupportsLooseEqualityWithNullishValues(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(0 == null, 0 != null, null == 0, null != 0, undefined == 0, undefined != 0)`})
	if err != nil {
		t.Fatalf("Dispatch(loose equality with nullish values) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || result.Value.Bool {
		t.Fatalf("Dispatch(loose equality with nullish values) result = %#v, want bool false", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 6 {
		t.Fatalf("host call args len = %d, want 6", len(call.args))
	}
	wantBools := []bool{false, true, false, true, false, true}
	for i, want := range wantBools {
		if call.args[i].Kind != ValueKindBool || call.args[i].Bool != want {
			t.Fatalf("host call arg[%d] = %#v, want bool %v", i, call.args[i], want)
		}
	}
}

func TestDispatchShortCircuitsConditionalOperatorBranches(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(true ? "safe" : host.echo("boom"))`})
	if err != nil {
		t.Fatalf("Dispatch(conditional short-circuit) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "safe" {
		t.Fatalf("Dispatch(conditional short-circuit) value = %#v, want string safe", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("Dispatch(conditional short-circuit) host calls = %#v, want one call", host.calls)
	}
}

func TestDispatchRejectsConditionalOperatorWithoutAlternateBranch(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(true ? "yes")`})
	if err == nil {
		t.Fatalf("Dispatch(conditional missing alternate) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(conditional missing alternate) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(conditional missing alternate) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedEqualityComparisonsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(1 ===)`})
	if err == nil {
		t.Fatalf("Dispatch(malformed equality comparison) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed equality comparison) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed equality comparison) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsExponentiationOperatorsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let value = 2; value **= 3; let big = 2n; big **= 3n; host.echo(value, big, 2 ** 3 ** 2, 2 ** -1)`})
	if err != nil {
		t.Fatalf("Dispatch(exponentiation operators) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 8 {
		t.Fatalf("Dispatch(exponentiation operators) result = %#v, want number 8", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[0].Kind != ValueKindNumber || call.args[0].Number != 8 {
		t.Fatalf("host call arg[0] = %#v, want number 8", call.args[0])
	}
	if call.args[1].Kind != ValueKindBigInt || call.args[1].BigInt != "8" {
		t.Fatalf("host call arg[1] = %#v, want bigint 8", call.args[1])
	}
	if call.args[2].Kind != ValueKindNumber || call.args[2].Number != 512 {
		t.Fatalf("host call arg[2] = %#v, want number 512", call.args[2])
	}
	if call.args[3].Kind != ValueKindNumber || call.args[3].Number != 0.5 {
		t.Fatalf("host call arg[3] = %#v, want number 0.5", call.args[3])
	}
}

func TestDispatchRejectsExponentiationOnMixedNumericKinds(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let value = 1n; value **= 2`})
	if err == nil {
		t.Fatalf("Dispatch(exponentiation on mixed numeric kinds) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(exponentiation on mixed numeric kinds) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(exponentiation on mixed numeric kinds) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsExponentiationOnNegativeBigIntExponent(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `1n ** -1n`})
	if err == nil {
		t.Fatalf("Dispatch(exponentiation on negative bigint exponent) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(exponentiation on negative bigint exponent) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(exponentiation on negative bigint exponent) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchSupportsBitwiseAndShiftOperatorsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(5 & 3, 5 | 2, 5 ^ 1, 1 << 3, 8 >> 1, 8 >>> 1, ~1, 1n & 3n, 1n << 2n)`})
	if err != nil {
		t.Fatalf("Dispatch(bitwise operators) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 1 {
		t.Fatalf("Dispatch(bitwise operators) result = %#v, want first scalar 1", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 9 {
		t.Fatalf("host call args len = %d, want 9", len(call.args))
	}
	wantNumbers := []float64{1, 7, 4, 8, 4, 4, -2}
	for i, want := range wantNumbers {
		if call.args[i].Kind != ValueKindNumber || call.args[i].Number != want {
			t.Fatalf("host call arg[%d] = %#v, want number %v", i, call.args[i], want)
		}
	}
	if call.args[7].Kind != ValueKindBigInt || call.args[7].BigInt != "1" {
		t.Fatalf("host call arg[7] = %#v, want bigint 1", call.args[7])
	}
	if call.args[8].Kind != ValueKindBigInt || call.args[8].BigInt != "4" {
		t.Fatalf("host call arg[8] = %#v, want bigint 4", call.args[8])
	}
}

func TestDispatchRejectsUnsignedShiftOnBigIntValues(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `1n >>> 1n`})
	if err == nil {
		t.Fatalf("Dispatch(unsigned shift on bigint) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(unsigned shift on bigint) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(unsigned shift on bigint) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsInOperatorOnNonObjectValues(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `1 in 2`})
	if err == nil {
		t.Fatalf("Dispatch(in operator on non-object) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(in operator on non-object) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(in operator on non-object) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsInstanceofOnNonClassObjects(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `({}) instanceof ({})`})
	if err == nil {
		t.Fatalf("Dispatch(instanceof on non-class objects) error = nil, want runtime error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(instanceof on non-class objects) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(instanceof on non-class objects) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsReservedTypeofDeclarationNames(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let typeof = 1`})
	if err == nil {
		t.Fatalf("Dispatch(reserved typeof declaration name) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(reserved typeof declaration name) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(reserved typeof declaration name) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchSupportsNullishCoalescingInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(null ?? "fallback")`})
	if err != nil {
		t.Fatalf("Dispatch(nullish coalescing) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(nullish coalescing) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "fallback" {
		t.Fatalf("host calls[0].args[0] = %#v, want fallback", host.calls[0].args[0])
	}
}

func TestDispatchSupportsNullishCoalescingOnNonScalarValuesInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := `let obj = { value: "kept" }; let arr = ["seed"]; let text = "go"; host.echo((obj ?? host.echo("boom")).value, (arr ?? host.echo("boom"))[0], text ?? host.echo("boom"), null ?? "fallback")`
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(nullish coalescing on non-scalar values) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(nullish coalescing on non-scalar values) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 4 {
		t.Fatalf("host.calls[0].args len = %d, want 4", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "kept" {
		t.Fatalf("host.calls[0].args[0] = %#v, want kept", host.calls[0].args[0])
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "seed" {
		t.Fatalf("host.calls[0].args[1] = %#v, want seed", host.calls[0].args[1])
	}
	if host.calls[0].args[2].Kind != ValueKindString || host.calls[0].args[2].String != "go" {
		t.Fatalf("host.calls[0].args[2] = %#v, want go", host.calls[0].args[2])
	}
	if host.calls[0].args[3].Kind != ValueKindString || host.calls[0].args[3].String != "fallback" {
		t.Fatalf("host.calls[0].args[3] = %#v, want fallback", host.calls[0].args[3])
	}
}

func TestDispatchShortCircuitsNullishCoalescingInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo("kept" ?? host.echo("boom"))`})
	if err != nil {
		t.Fatalf("Dispatch(nullish short-circuit) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(nullish short-circuit) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "kept" {
		t.Fatalf("host calls[0].args[0] = %#v, want kept", host.calls[0].args[0])
	}
}

func TestDispatchSupportsOptionalChainingOnHostMethodCalls(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host?.echo("ok")`})
	if err != nil {
		t.Fatalf("Dispatch(optional chaining host method) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(optional chaining host method) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 {
		t.Fatalf("host calls[0].args len = %d, want 1", len(host.calls[0].args))
	}
	if host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "ok" {
		t.Fatalf("host calls[0].args[0] = %#v, want ok", host.calls[0].args[0])
	}
}

func TestDispatchShortCircuitsOptionalChainingOnNullishBase(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `null?.echo(host.echo("boom"))`})
	if err != nil {
		t.Fatalf("Dispatch(optional chaining short-circuit) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(optional chaining short-circuit) kind = %q, want %q", result.Value.Kind, ValueKindUndefined)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchSupportsObjectPropertyAccessAndOptionalChaining(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let payload = { title: "ready", nested: { value: "changed" }, items: [1, 2, 3] }; host.echo(payload.title, payload?.nested?.value ?? "fallback", payload.items.length, payload?.items?.length, payload?.missing?.value ?? "seed")`})
	if err != nil {
		t.Fatalf("Dispatch(object property access and optional chaining) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(object property access and optional chaining) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 5 {
		t.Fatalf("host call args len = %d, want 5", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "ready" {
		t.Fatalf("host call arg[0] = %#v, want ready", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "changed" {
		t.Fatalf("host call arg[1] = %#v, want changed", call.args[1])
	}
	if call.args[2].Kind != ValueKindNumber || call.args[2].Number != 3 {
		t.Fatalf("host call arg[2] = %#v, want array length 3", call.args[2])
	}
	if call.args[3].Kind != ValueKindNumber || call.args[3].Number != 3 {
		t.Fatalf("host call arg[3] = %#v, want optional array length 3", call.args[3])
	}
	if call.args[4].Kind != ValueKindString || call.args[4].String != "seed" {
		t.Fatalf("host call arg[4] = %#v, want seed", call.args[4])
	}
}

func TestDispatchSupportsOptionalBracketAccessAndOptionalCalls(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
			"log":  StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let payload = { nested: { value: "changed" }, items: [1, 2, 3] }; let ops = { write: x => x }; host?.["log"]("alpha"); host.echo(payload?.["nested"]?.["value"], payload?.["items"]?.[1], ops.write?.("fresh"))`})
	if err != nil {
		t.Fatalf("Dispatch(optional bracket access and optional calls) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(optional bracket access and optional calls) value = %#v, want string ok", result.Value)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "log" {
		t.Fatalf("host calls[0].method = %q, want log", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "alpha" {
		t.Fatalf("host calls[0].args = %#v, want alpha", host.calls[0].args)
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 3 {
		t.Fatalf("host calls[1].args len = %d, want 3", len(host.calls[1].args))
	}
	if host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "changed" {
		t.Fatalf("host calls[1].args[0] = %#v, want changed", host.calls[1].args[0])
	}
	if host.calls[1].args[1].Kind != ValueKindNumber || host.calls[1].args[1].Number != 2 {
		t.Fatalf("host calls[1].args[1] = %#v, want array index 1", host.calls[1].args[1])
	}
	if host.calls[1].args[2].Kind != ValueKindString || host.calls[1].args[2].String != "fresh" {
		t.Fatalf("host calls[1].args[2] = %#v, want fresh", host.calls[1].args[2])
	}
}

func TestDispatchSupportsBracketAccessOnStringValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := "let text = \"go\"; let first = text[0]; let second = text[1]; let length = text[\"length\"]; `" + "${first}|${second}|${length}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(bracket access on string values) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "g|o|2" {
		t.Fatalf("Dispatch(bracket access on string values) result = %#v, want string g|o|2", result.Value)
	}
}

func TestDispatchSupportsMemberAccessOnStringAndArrayValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := "let text = \"go\"; let arr = [1, 2]; let textFoo = text.foo; let arrFoo = arr.foo; let textLength = text.length; let arrLength = arr.length; `" + "${textFoo}|${arrFoo}|${textLength}|${arrLength}" + "`"
	result, err := runtime.Dispatch(DispatchRequest{Source: source})
	if err != nil {
		t.Fatalf("Dispatch(member access on string and array values) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "undefined|undefined|2|2" {
		t.Fatalf("Dispatch(member access on string and array values) result = %#v, want string undefined|undefined|2|2", result.Value)
	}
}

func TestDispatchSupportsMemberAccessOnPrimitiveValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := "let num = 1; let bool = false; let big = 1n; let numValue = num.foo; let boolValue = bool.foo; let bigValue = big.foo; host.echo(`${numValue}|${boolValue}|${bigValue}`)"
	if _, err := runtime.Dispatch(DispatchRequest{Source: source}); err != nil {
		t.Fatalf("Dispatch(member access on primitive values) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want 1 call", host.calls)
	}
	call := host.calls[0]
	if len(call.args) != 1 {
		t.Fatalf("host call args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "undefined|undefined|undefined" {
		t.Fatalf("host call arg[0] = %#v, want primitive properties to resolve to undefined", call.args[0])
	}
}

func TestDispatchSupportsBracketAccessOnPrimitiveValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	source := "let num = 1; let bool = false; let big = 1n; let numValue = num[\"foo\"]; let boolValue = bool[\"foo\"]; let bigValue = big[\"foo\"]; host.echo(`${numValue}|${boolValue}|${bigValue}`)"
	if _, err := runtime.Dispatch(DispatchRequest{Source: source}); err != nil {
		t.Fatalf("Dispatch(bracket access on primitive values) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want 1 call", host.calls)
	}
	call := host.calls[0]
	if len(call.args) != 1 {
		t.Fatalf("host call args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "undefined|undefined|undefined" {
		t.Fatalf("host call arg[0] = %#v, want primitive properties to resolve to undefined", call.args[0])
	}
}

func TestDispatchShortCircuitsOptionalCallAndBracketAccessOnNullishBase(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `null?.(host.echo("boom")); null?.[host.echo("boom")]`})
	if err != nil {
		t.Fatalf("Dispatch(optional call/bracket short-circuit) error = %v", err)
	}
	if result.Value.Kind != ValueKindUndefined {
		t.Fatalf("Dispatch(optional call/bracket short-circuit) kind = %q, want %q", result.Value.Kind, ValueKindUndefined)
	}
	if len(host.calls) != 0 {
		t.Fatalf("host calls = %#v, want no calls", host.calls)
	}
}

func TestDispatchRejectsMalformedOptionalChaining(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `null?.`})
	if err == nil {
		t.Fatalf("Dispatch(malformed optional chaining) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed optional chaining) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed optional chaining) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedNullishCoalescing(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo("value" ??)`})
	if err == nil {
		t.Fatalf("Dispatch(malformed nullish coalescing) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed nullish coalescing) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed nullish coalescing) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchRejectsMalformedNumericSeparators(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo(1_)`})
	if err == nil {
		t.Fatalf("Dispatch(malformed numeric separators) error = nil, want parse error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(malformed numeric separators) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(malformed numeric separators) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchParsesHostArguments(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:echo("div > section > p.primary", true, 2)`})
	if err != nil {
		t.Fatalf("Dispatch(host:echo(...)) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "echo" {
		t.Fatalf("host call method = %q, want echo", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host call args len = %d, want 3", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "div > section > p.primary" {
		t.Fatalf("host call arg[0] = %#v, want selector string", call.args[0])
	}
	if call.args[1].Kind != ValueKindBool || !call.args[1].Bool {
		t.Fatalf("host call arg[1] = %#v, want true", call.args[1])
	}
	if call.args[2].Kind != ValueKindNumber || call.args[2].Number != 2 {
		t.Fatalf("host call arg[2] = %#v, want 2", call.args[2])
	}
}

func TestParseHostInvocationNestedArguments(t *testing.T) {
	method, args, err := parseHostInvocation(`setTimeout(expr(host:historyPushState("step-1", "", "https://example.test/step-1")), 25)`)
	if err != nil {
		t.Fatalf("parseHostInvocation() error = %v", err)
	}
	if method != "setTimeout" {
		t.Fatalf("method = %q, want setTimeout", method)
	}
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2", len(args))
	}
	if args[0].Kind != ValueKindInvocation {
		t.Fatalf("args[0].Kind = %q, want %q", args[0].Kind, ValueKindInvocation)
	}
	if args[0].Invocation != `host:historyPushState("step-1", "", "https://example.test/step-1")` {
		t.Fatalf("args[0].Invocation = %q, want host:historyPushState(...)", args[0].Invocation)
	}
}

func TestParseHostInvocationQuotedCommaArgument(t *testing.T) {
	method, args, err := parseHostInvocation(`setTextContent("#out", "first,part")`)
	if err != nil {
		t.Fatalf("parseHostInvocation() error = %v", err)
	}
	if method != "setTextContent" {
		t.Fatalf("method = %q, want setTextContent", method)
	}
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2", len(args))
	}
	if args[1].Kind != ValueKindString || args[1].String != "first,part" {
		t.Fatalf("args[1] = %#v, want quoted string with comma", args[1])
	}
}

func TestDispatchParsesTextContentMutationInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:setTextContent("#out", "clicked")`})
	if err != nil {
		t.Fatalf("Dispatch(setTextContent) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "setTextContent" {
		t.Fatalf("host call method = %q, want setTextContent", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "#out" {
		t.Fatalf("host call arg[0] = %#v, want selector string", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "clicked" {
		t.Fatalf("host call arg[1] = %#v, want clicked", call.args[1])
	}
}

func TestDispatchParsesTextContentGetterInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"textContent": StringValue("seed text"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host:textContent("#src")`})
	if err != nil {
		t.Fatalf("Dispatch(textContent) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "seed text" {
		t.Fatalf("Dispatch(textContent) value = %#v, want seed text", result.Value)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "textContent" {
		t.Fatalf("host call method = %q, want textContent", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "#src" {
		t.Fatalf("host call args = %#v, want selector argument", host.calls[0].args)
	}
}

func TestDispatchParsesTreeMutationInvocations(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"replaceChildren": UndefinedValue(),
			"cloneNode":       UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:replaceChildren("#out", "<span>fresh</span>"); host:cloneNode("#src", true)`})
	if err != nil {
		t.Fatalf("Dispatch(tree mutations) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "replaceChildren" {
		t.Fatalf("host call[0].method = %q, want replaceChildren", host.calls[0].method)
	}
	if len(host.calls[0].args) != 2 || host.calls[0].args[0].String != "#out" || host.calls[0].args[1].String != "<span>fresh</span>" {
		t.Fatalf("host call[0].args = %#v, want selector + markup", host.calls[0].args)
	}
	if host.calls[1].method != "cloneNode" {
		t.Fatalf("host call[1].method = %q, want cloneNode", host.calls[1].method)
	}
	if len(host.calls[1].args) != 2 || host.calls[1].args[0].String != "#src" || host.calls[1].args[1].Kind != ValueKindBool || !host.calls[1].args[1].Bool {
		t.Fatalf("host call[1].args = %#v, want selector + true", host.calls[1].args)
	}
}

func TestDispatchParsesQuotedEventListenerSource(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"addEventListener": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", "clicked")')`})
	if err != nil {
		t.Fatalf("Dispatch(addEventListener) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "addEventListener" {
		t.Fatalf("host call method = %q, want addEventListener", call.method)
	}
	if len(call.args) != 3 {
		t.Fatalf("host call args len = %d, want 3", len(call.args))
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != `host:setInnerHTML("#out", "clicked")` {
		t.Fatalf("host call arg[2] = %#v, want quoted source string", call.args[2])
	}
}

func TestDispatchParsesEventListenerPhaseArgument(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"addEventListener": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", "clicked")', "capture")`})
	if err != nil {
		t.Fatalf("Dispatch(addEventListener capture) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[3].Kind != ValueKindString || call.args[3].String != "capture" {
		t.Fatalf("host call arg[3] = %#v, want capture", call.args[3])
	}
}

func TestDispatchParsesEventListenerOnceArgument(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"addEventListener": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", "clicked")', "capture", true)`})
	if err != nil {
		t.Fatalf("Dispatch(addEventListener once) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if len(call.args) != 5 {
		t.Fatalf("host call args len = %d, want 5", len(call.args))
	}
	if call.args[4].Kind != ValueKindBool || !call.args[4].Bool {
		t.Fatalf("host call arg[4] = %#v, want true", call.args[4])
	}
}

func TestDispatchParsesEventListenerRemoval(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"removeEventListener": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:removeEventListener("#btn", "click", 'host:removeNode("#btn")', "capture")`})
	if err != nil {
		t.Fatalf("Dispatch(removeEventListener) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if len(call.args) != 4 {
		t.Fatalf("host call args len = %d, want 4", len(call.args))
	}
	if call.args[2].Kind != ValueKindString || call.args[2].String != `host:removeNode("#btn")` {
		t.Fatalf("host call arg[2] = %#v, want raw source string", call.args[2])
	}
	if call.args[3].Kind != ValueKindString || call.args[3].String != "capture" {
		t.Fatalf("host call arg[3] = %#v, want capture", call.args[3])
	}
}

func TestDispatchParsesQueueMicrotaskInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"queueMicrotask": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:queueMicrotask('host:setInnerHTML(#out, micro)')`})
	if err != nil {
		t.Fatalf("Dispatch(queueMicrotask) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if len(call.args) != 1 {
		t.Fatalf("host call args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != `host:setInnerHTML(#out, micro)` {
		t.Fatalf("host call arg[0] = %#v, want raw source string", call.args[0])
	}
}

func TestDispatchParsesSetTimeoutInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTimeout": NumberValue(1),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:setTimeout('host:setInnerHTML(#out, later)', 25)`})
	if err != nil {
		t.Fatalf("Dispatch(setTimeout) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "setTimeout" {
		t.Fatalf("host call method = %q, want setTimeout", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != `host:setInnerHTML(#out, later)` {
		t.Fatalf("host call arg[0] = %#v, want raw source string", call.args[0])
	}
	if call.args[1].Kind != ValueKindNumber || call.args[1].Number != 25 {
		t.Fatalf("host call arg[1] = %#v, want 25", call.args[1])
	}
}

func TestDispatchParsesWriteHTMLInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"writeHTML": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:writeHTML('<main><div data-x="a;b">ok</div></main>')`})
	if err != nil {
		t.Fatalf("Dispatch(writeHTML) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "writeHTML" {
		t.Fatalf("host call method = %q, want writeHTML", call.method)
	}
	if len(call.args) != 1 {
		t.Fatalf("host call args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != `<main><div data-x="a;b">ok</div></main>` {
		t.Fatalf("host call arg[0] = %#v, want raw markup string", call.args[0])
	}
}

func TestDispatchParsesLocationNavigationInvocations(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		method     string
		wantArgs   int
		wantString string
	}{
		{
			name:       "assign",
			source:     `host:locationAssign("https://example.test/assign")`,
			method:     "locationAssign",
			wantArgs:   1,
			wantString: "https://example.test/assign",
		},
		{
			name:       "replace",
			source:     `host:locationReplace("https://example.test/replace")`,
			method:     "locationReplace",
			wantArgs:   1,
			wantString: "https://example.test/replace",
		},
		{
			name:     "reload",
			source:   `host:locationReload()`,
			method:   "locationReload",
			wantArgs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					tc.method: UndefinedValue(),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			_, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != tc.wantArgs {
				t.Fatalf("host call args len = %d, want %d", len(call.args), tc.wantArgs)
			}
			if tc.wantArgs == 1 {
				if call.args[0].Kind != ValueKindString || call.args[0].String != tc.wantString {
					t.Fatalf("host call arg[0] = %#v, want %q", call.args[0], tc.wantString)
				}
			}
		})
	}
}

func TestDispatchParsesLocationGetterInvocations(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		method     string
		wantString string
	}{
		{
			name:       "href",
			source:     `host:locationHref()`,
			method:     "locationHref",
			wantString: "https://example.test:8443/path/name?mode=full#step-1",
		},
		{
			name:       "origin",
			source:     `host:locationOrigin()`,
			method:     "locationOrigin",
			wantString: "https://example.test:8443",
		},
		{
			name:       "protocol",
			source:     `host:locationProtocol()`,
			method:     "locationProtocol",
			wantString: "https:",
		},
		{
			name:       "host",
			source:     `host:locationHost()`,
			method:     "locationHost",
			wantString: "example.test:8443",
		},
		{
			name:       "hostname",
			source:     `host:locationHostname()`,
			method:     "locationHostname",
			wantString: "example.test",
		},
		{
			name:       "port",
			source:     `host:locationPort()`,
			method:     "locationPort",
			wantString: "8443",
		},
		{
			name:       "pathname",
			source:     `host:locationPathname()`,
			method:     "locationPathname",
			wantString: "/path/name",
		},
		{
			name:       "search",
			source:     `host:locationSearch()`,
			method:     "locationSearch",
			wantString: "?mode=full",
		},
		{
			name:       "hash",
			source:     `host:locationHash()`,
			method:     "locationHash",
			wantString: "#step-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					tc.method: StringValue(tc.wantString),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			result, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != 0 {
				t.Fatalf("host call args len = %d, want 0", len(call.args))
			}
			if result.Value.Kind != ValueKindString || result.Value.String != tc.wantString {
				t.Fatalf("Dispatch(%s) value = %#v, want string %q", tc.method, result.Value, tc.wantString)
			}
		})
	}
}

func TestDispatchParsesLocationPropertyInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"locationSet": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:locationSet("hash", "#next")`})
	if err != nil {
		t.Fatalf("Dispatch(locationSet) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "locationSet" {
		t.Fatalf("host call method = %q, want locationSet", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != "hash" {
		t.Fatalf("host call arg[0] = %#v, want hash", call.args[0])
	}
	if call.args[1].Kind != ValueKindString || call.args[1].String != "#next" {
		t.Fatalf("host call arg[1] = %#v, want #next", call.args[1])
	}
}

func TestDispatchParsesLocationPropertyInvocationSymbolArgs(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"locationSet": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:locationSet(expr(Symbol("property")), expr(Symbol("value")))`})
	if err != nil {
		t.Fatalf("Dispatch(locationSet symbol args) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "locationSet" {
		t.Fatalf("host call method = %q, want locationSet", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindSymbol || call.args[0].SymbolDescription != "property" {
		t.Fatalf("host call arg[0] = %#v, want Symbol(property)", call.args[0])
	}
	if call.args[1].Kind != ValueKindSymbol || call.args[1].SymbolDescription != "value" {
		t.Fatalf("host call arg[1] = %#v, want Symbol(value)", call.args[1])
	}
}

func TestDispatchParsesHistoryScrollRestorationInvocationSymbolArgs(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"historySetScrollRestoration": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:historySetScrollRestoration(expr(Symbol("token")))`})
	if err != nil {
		t.Fatalf("Dispatch(historySetScrollRestoration symbol arg) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "historySetScrollRestoration" {
		t.Fatalf("host call method = %q, want historySetScrollRestoration", call.method)
	}
	if len(call.args) != 1 {
		t.Fatalf("host call args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindSymbol || call.args[0].SymbolDescription != "token" {
		t.Fatalf("host call arg[0] = %#v, want Symbol(token)", call.args[0])
	}
}

func TestDispatchParsesHistoryNavigationInvocations(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		method   string
		wantArgs int
		check    func(t *testing.T, call hostCall)
	}{
		{
			name:     "pushState",
			source:   `host:historyPushState("step-1", "", "https://example.test/step-1")`,
			method:   "historyPushState",
			wantArgs: 3,
			check: func(t *testing.T, call hostCall) {
				if call.args[0].Kind != ValueKindString || call.args[0].String != "step-1" {
					t.Fatalf("host call arg[0] = %#v, want step-1", call.args[0])
				}
				if call.args[1].Kind != ValueKindString || call.args[1].String != "" {
					t.Fatalf("host call arg[1] = %#v, want empty title", call.args[1])
				}
				if call.args[2].Kind != ValueKindString || call.args[2].String != "https://example.test/step-1" {
					t.Fatalf("host call arg[2] = %#v, want url", call.args[2])
				}
			},
		},
		{
			name:     "replaceState",
			source:   `host:historyReplaceState("step-2", "", "https://example.test/step-2")`,
			method:   "historyReplaceState",
			wantArgs: 3,
			check: func(t *testing.T, call hostCall) {
				if call.args[0].Kind != ValueKindString || call.args[0].String != "step-2" {
					t.Fatalf("host call arg[0] = %#v, want step-2", call.args[0])
				}
				if call.args[2].Kind != ValueKindString || call.args[2].String != "https://example.test/step-2" {
					t.Fatalf("host call arg[2] = %#v, want url", call.args[2])
				}
			},
		},
		{
			name:     "back",
			source:   `host:historyBack()`,
			method:   "historyBack",
			wantArgs: 0,
		},
		{
			name:     "forward",
			source:   `host:historyForward()`,
			method:   "historyForward",
			wantArgs: 0,
		},
		{
			name:     "go",
			source:   `host:historyGo(-1)`,
			method:   "historyGo",
			wantArgs: 1,
			check: func(t *testing.T, call hostCall) {
				if call.args[0].Kind != ValueKindNumber || call.args[0].Number != -1 {
					t.Fatalf("host call arg[0] = %#v, want -1", call.args[0])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					tc.method: UndefinedValue(),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			_, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != tc.wantArgs {
				t.Fatalf("host call args len = %d, want %d", len(call.args), tc.wantArgs)
			}
			if tc.check != nil {
				tc.check(t, call)
			}
		})
	}
}

func TestDispatchParsesHistoryAccessorsAndScrollRestoration(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		method     string
		wantArgs   int
		wantKind   ValueKind
		wantString string
		wantNumber float64
	}{
		{
			name:       "length",
			source:     `host:historyLength()`,
			method:     "historyLength",
			wantArgs:   0,
			wantKind:   ValueKindNumber,
			wantNumber: 2,
		},
		{
			name:       "state",
			source:     `host:historyState()`,
			method:     "historyState",
			wantArgs:   0,
			wantKind:   ValueKindString,
			wantString: "current",
		},
		{
			name:       "scrollRestoration",
			source:     `host:historyScrollRestoration()`,
			method:     "historyScrollRestoration",
			wantArgs:   0,
			wantKind:   ValueKindString,
			wantString: "auto",
		},
		{
			name:     "setScrollRestoration",
			source:   `host:historySetScrollRestoration("manual")`,
			method:   "historySetScrollRestoration",
			wantArgs: 1,
			wantKind: ValueKindUndefined,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					tc.method: func() Value {
						switch tc.method {
						case "historyLength":
							return NumberValue(2)
						case "historyState":
							return StringValue(tc.wantString)
						case "historyScrollRestoration":
							return StringValue(tc.wantString)
						default:
							return UndefinedValue()
						}
					}(),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			result, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != tc.wantArgs {
				t.Fatalf("host call args len = %d, want %d", len(call.args), tc.wantArgs)
			}
			if result.Value.Kind != tc.wantKind {
				t.Fatalf("Dispatch(%s) kind = %q, want %q", tc.method, result.Value.Kind, tc.wantKind)
			}
			if tc.wantKind == ValueKindString && result.Value.String != tc.wantString {
				t.Fatalf("Dispatch(%s) value = %q, want %q", tc.method, result.Value.String, tc.wantString)
			}
			if tc.wantKind == ValueKindNumber && result.Value.Number != tc.wantNumber {
				t.Fatalf("Dispatch(%s) value = %v, want %v", tc.method, result.Value.Number, tc.wantNumber)
			}
		})
	}
}

func TestDispatchParsesDocumentCookieInvocations(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		method   string
		wantArgs int
		check    func(t *testing.T, call hostCall, result Value)
	}{
		{
			name:     "getter",
			source:   `host:documentCookie()`,
			method:   "documentCookie",
			wantArgs: 0,
			check: func(t *testing.T, call hostCall, result Value) {
				if result.Kind != ValueKindString || result.String != "theme=light" {
					t.Fatalf("Dispatch(documentCookie) value = %#v, want string theme=light", result)
				}
			},
		},
		{
			name:     "setter",
			source:   `host:setDocumentCookie("theme=dark")`,
			method:   "setDocumentCookie",
			wantArgs: 1,
			check: func(t *testing.T, call hostCall, result Value) {
				if call.args[0].Kind != ValueKindString || call.args[0].String != "theme=dark" {
					t.Fatalf("host call arg[0] = %#v, want theme=dark", call.args[0])
				}
				if result.Kind != ValueKindUndefined {
					t.Fatalf("Dispatch(setDocumentCookie) value = %#v, want undefined", result)
				}
			},
		},
		{
			name:     "navigatorCookieEnabled",
			source:   `host:navigatorCookieEnabled()`,
			method:   "navigatorCookieEnabled",
			wantArgs: 0,
			check: func(t *testing.T, call hostCall, result Value) {
				if result.Kind != ValueKindBool || !result.Bool {
					t.Fatalf("Dispatch(navigatorCookieEnabled) value = %#v, want true", result)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					"documentCookie":         StringValue("theme=light"),
					"setDocumentCookie":      UndefinedValue(),
					"navigatorCookieEnabled": BoolValue(true),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			result, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != tc.wantArgs {
				t.Fatalf("host call args len = %d, want %d", len(call.args), tc.wantArgs)
			}
			if tc.check != nil {
				tc.check(t, call, result.Value)
			}
		})
	}
}

func TestDispatchParsesDocumentCurrentScriptInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"documentCurrentScript": StringValue(`<script id="boot">host:documentCurrentScript()</script>`),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host:documentCurrentScript()`})
	if err != nil {
		t.Fatalf("Dispatch(documentCurrentScript) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "documentCurrentScript" {
		t.Fatalf("host call method = %q, want documentCurrentScript", call.method)
	}
	if len(call.args) != 0 {
		t.Fatalf("host call args len = %d, want 0", len(call.args))
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(documentCurrentScript) kind = %q, want string", result.Value.Kind)
	}
}

func TestDispatchParsesEventTargetValueInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"eventTargetValue": StringValue("Ada"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host:eventTargetValue()`})
	if err != nil {
		t.Fatalf("Dispatch(eventTargetValue) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "eventTargetValue" {
		t.Fatalf("host call method = %q, want eventTargetValue", call.method)
	}
	if len(call.args) != 0 {
		t.Fatalf("host call args len = %d, want 0", len(call.args))
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "Ada" {
		t.Fatalf("Dispatch(eventTargetValue) value = %#v, want string Ada", result.Value)
	}
}

func TestDispatchParsesWindowNameInvocations(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		method   string
		wantArgs int
		check    func(t *testing.T, call hostCall, result Value)
	}{
		{
			name:     "getter",
			source:   `host:windowName()`,
			method:   "windowName",
			wantArgs: 0,
			check: func(t *testing.T, call hostCall, result Value) {
				if result.Kind != ValueKindString || result.String != "alpha" {
					t.Fatalf("Dispatch(windowName) value = %#v, want string alpha", result)
				}
			},
		},
		{
			name:     "setter",
			source:   `host:setWindowName("alpha")`,
			method:   "setWindowName",
			wantArgs: 1,
			check: func(t *testing.T, call hostCall, result Value) {
				if call.args[0].Kind != ValueKindString || call.args[0].String != "alpha" {
					t.Fatalf("host call arg[0] = %#v, want alpha", call.args[0])
				}
				if result.Kind != ValueKindUndefined {
					t.Fatalf("Dispatch(setWindowName) value = %#v, want undefined", result)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					"windowName":    StringValue("alpha"),
					"setWindowName": UndefinedValue(),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			result, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != tc.wantArgs {
				t.Fatalf("host call args len = %d, want %d", len(call.args), tc.wantArgs)
			}
			if tc.check != nil {
				tc.check(t, call, result.Value)
			}
		})
	}
}

func TestDispatchParsesWindowNameInvocationSymbolArgs(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setWindowName": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:setWindowName(expr(Symbol("token")))`})
	if err != nil {
		t.Fatalf("Dispatch(setWindowName symbol arg) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "setWindowName" {
		t.Fatalf("host call method = %q, want setWindowName", call.method)
	}
	if len(call.args) != 1 {
		t.Fatalf("host call args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindSymbol || call.args[0].SymbolDescription != "token" {
		t.Fatalf("host call arg[0] = %#v, want Symbol(token)", call.args[0])
	}
}

func TestDispatchParsesWebStorageInvocations(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		method     string
		wantArgs   int
		wantKind   ValueKind
		wantString string
		wantNumber float64
	}{
		{
			name:       "localStorageGetItem",
			source:     `host:localStorageGetItem("theme")`,
			method:     "localStorageGetItem",
			wantArgs:   1,
			wantKind:   ValueKindString,
			wantString: "dark",
		},
		{
			name:     "localStorageSetItem",
			source:   `host:localStorageSetItem("theme", "light")`,
			method:   "localStorageSetItem",
			wantArgs: 2,
			wantKind: ValueKindUndefined,
		},
		{
			name:       "localStorageLength",
			source:     `host:localStorageLength()`,
			method:     "localStorageLength",
			wantArgs:   0,
			wantKind:   ValueKindNumber,
			wantNumber: 2,
		},
		{
			name:       "localStorageKey",
			source:     `host:localStorageKey(0)`,
			method:     "localStorageKey",
			wantArgs:   1,
			wantKind:   ValueKindString,
			wantString: "theme",
		},
		{
			name:       "sessionStorageGetItem",
			source:     `host:sessionStorageGetItem("tab")`,
			method:     "sessionStorageGetItem",
			wantArgs:   1,
			wantKind:   ValueKindString,
			wantString: "main",
		},
		{
			name:     "sessionStorageSetItem",
			source:   `host:sessionStorageSetItem("tab", "main")`,
			method:   "sessionStorageSetItem",
			wantArgs: 2,
			wantKind: ValueKindUndefined,
		},
		{
			name:     "sessionStorageRemoveItem",
			source:   `host:sessionStorageRemoveItem("tab")`,
			method:   "sessionStorageRemoveItem",
			wantArgs: 1,
			wantKind: ValueKindUndefined,
		},
		{
			name:     "sessionStorageClear",
			source:   `host:sessionStorageClear()`,
			method:   "sessionStorageClear",
			wantArgs: 0,
			wantKind: ValueKindUndefined,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					tc.method: func() Value {
						switch tc.wantKind {
						case ValueKindString:
							return StringValue(tc.wantString)
						case ValueKindNumber:
							return NumberValue(tc.wantNumber)
						default:
							return UndefinedValue()
						}
					}(),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			result, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != tc.wantArgs {
				t.Fatalf("host call args len = %d, want %d", len(call.args), tc.wantArgs)
			}
			if result.Value.Kind != tc.wantKind {
				t.Fatalf("Dispatch(%s) kind = %q, want %q", tc.method, result.Value.Kind, tc.wantKind)
			}
			if tc.wantKind == ValueKindString && result.Value.String != tc.wantString {
				t.Fatalf("Dispatch(%s) value = %q, want %q", tc.method, result.Value.String, tc.wantString)
			}
			if tc.wantKind == ValueKindNumber && result.Value.Number != tc.wantNumber {
				t.Fatalf("Dispatch(%s) value = %v, want %v", tc.method, result.Value.Number, tc.wantNumber)
			}
		})
	}
}

func TestDispatchParsesSetIntervalInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setInterval": NumberValue(2),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:setInterval('host:insertAdjacentHTML("#log", "beforeend", "<span>tick</span>")', 50)`})
	if err != nil {
		t.Fatalf("Dispatch(setInterval) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "setInterval" {
		t.Fatalf("host call method = %q, want setInterval", call.method)
	}
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != `host:insertAdjacentHTML("#log", "beforeend", "<span>tick</span>")` {
		t.Fatalf("host call arg[0] = %#v, want raw source string", call.args[0])
	}
	if call.args[1].Kind != ValueKindNumber || call.args[1].Number != 50 {
		t.Fatalf("host call arg[1] = %#v, want 50", call.args[1])
	}
}

func TestDispatchParsesTimerCancellationInvocations(t *testing.T) {
	tests := []struct {
		name   string
		source string
		method string
	}{
		{
			name:   "clearTimeout",
			source: `host:clearTimeout(1)`,
			method: "clearTimeout",
		},
		{
			name:   "clearTimeout interval alias",
			source: `host:clearInterval(2)`,
			method: "clearInterval",
		},
		{
			name:   "clearInterval alias only",
			source: `host:clearInterval(3)`,
			method: "clearInterval",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{
				values: map[string]Value{
					tc.method: UndefinedValue(),
				},
				errs: map[string]error{},
			}
			runtime := NewRuntime(host)

			_, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.method, err)
			}
			if len(host.calls) != 1 {
				t.Fatalf("host calls = %#v, want one call", host.calls)
			}
			call := host.calls[0]
			if call.method != tc.method {
				t.Fatalf("host call method = %q, want %s", call.method, tc.method)
			}
			if len(call.args) != 1 {
				t.Fatalf("host call args len = %d, want 1", len(call.args))
			}
			if call.args[0].Kind != ValueKindNumber {
				t.Fatalf("host call arg[0] = %#v, want number", call.args[0])
			}
		})
	}
}

func TestDispatchParsesAnimationFrameInvocations(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"requestAnimationFrame": NumberValue(3),
			"cancelAnimationFrame":  UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:requestAnimationFrame('host:insertAdjacentHTML("#log", "beforeend", "<span>frame</span>")')`})
	if err != nil {
		t.Fatalf("Dispatch(requestAnimationFrame) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "requestAnimationFrame" {
		t.Fatalf("host call method = %q, want requestAnimationFrame", call.method)
	}
	if len(call.args) != 1 {
		t.Fatalf("host call args len = %d, want 1", len(call.args))
	}
	if call.args[0].Kind != ValueKindString || call.args[0].String != `host:insertAdjacentHTML("#log", "beforeend", "<span>frame</span>")` {
		t.Fatalf("host call arg[0] = %#v, want raw source string", call.args[0])
	}

	host.calls = nil
	_, err = runtime.Dispatch(DispatchRequest{Source: `host:cancelAnimationFrame(3)`})
	if err != nil {
		t.Fatalf("Dispatch(cancelAnimationFrame) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls after cancel = %#v, want one call", host.calls)
	}
	call = host.calls[0]
	if call.method != "cancelAnimationFrame" {
		t.Fatalf("host call method = %q, want cancelAnimationFrame", call.method)
	}
	if len(call.args) != 1 || call.args[0].Kind != ValueKindNumber || call.args[0].Number != 3 {
		t.Fatalf("host call args = %#v, want number 3", call.args)
	}
}

func TestDispatchParsesPreventDefaultInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"preventDefault": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:preventDefault()`})
	if err != nil {
		t.Fatalf("Dispatch(preventDefault) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "preventDefault" {
		t.Fatalf("host call method = %q, want preventDefault", call.method)
	}
	if len(call.args) != 0 {
		t.Fatalf("host call args len = %d, want 0", len(call.args))
	}
}

func TestDispatchParsesStopPropagationInvocation(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"stopPropagation": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:stopPropagation()`})
	if err != nil {
		t.Fatalf("Dispatch(stopPropagation) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	call := host.calls[0]
	if call.method != "stopPropagation" {
		t.Fatalf("host call method = %q, want stopPropagation", call.method)
	}
	if len(call.args) != 0 {
		t.Fatalf("host call args len = %d, want 0", len(call.args))
	}
}

func TestDispatchSupportsMultipleHostStatements(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setInnerHTML": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host:setInnerHTML("#out", "first;part"); host:setInnerHTML("#out", "second")`})
	if err != nil {
		t.Fatalf("Dispatch(multiple host statements) error = %v", err)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two calls", host.calls)
	}
	if host.calls[0].method != "setInnerHTML" || host.calls[1].method != "setInnerHTML" {
		t.Fatalf("host call methods = %#v, want setInnerHTML twice", host.calls)
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "first;part" {
		t.Fatalf("host call[0] arg[1] = %#v, want first;part", host.calls[0].args[1])
	}
	if host.calls[1].args[1].Kind != ValueKindString || host.calls[1].args[1].String != "second" {
		t.Fatalf("host call[1] arg[1] = %#v, want second", host.calls[1].args[1])
	}
}

type domQueryHost struct {
	store *dom.Store
}

func (h *domQueryHost) Call(method string, args []Value) (Value, error) {
	if h == nil || h.store == nil {
		return UndefinedValue(), fmt.Errorf("dom query host is unavailable")
	}
	switch method {
	case "querySelector":
		if len(args) != 1 || args[0].Kind != ValueKindString {
			return UndefinedValue(), fmt.Errorf("querySelector requires one selector string")
		}
		nodeID, ok, err := h.store.QuerySelector(args[0].String)
		if err != nil {
			return UndefinedValue(), err
		}
		if !ok {
			return UndefinedValue(), nil
		}
		return StringValue(fmt.Sprintf("%d", nodeID)), nil
	case "querySelectorAll":
		if len(args) != 1 || args[0].Kind != ValueKindString {
			return UndefinedValue(), fmt.Errorf("querySelectorAll requires one selector string")
		}
		nodes, err := h.store.QuerySelectorAll(args[0].String)
		if err != nil {
			return UndefinedValue(), err
		}
		return NumberValue(float64(nodes.Length())), nil
	case "matches":
		if len(args) != 2 || args[0].Kind != ValueKindNumber || args[1].Kind != ValueKindString {
			return UndefinedValue(), fmt.Errorf("matches requires a node id and selector string")
		}
		matched, err := h.store.Matches(dom.NodeID(args[0].Number), args[1].String)
		if err != nil {
			return UndefinedValue(), err
		}
		return BoolValue(matched), nil
	case "closest":
		if len(args) != 2 || args[0].Kind != ValueKindNumber || args[1].Kind != ValueKindString {
			return UndefinedValue(), fmt.Errorf("closest requires a node id and selector string")
		}
		nodeID, ok, err := h.store.Closest(dom.NodeID(args[0].Number), args[1].String)
		if err != nil {
			return UndefinedValue(), err
		}
		if !ok {
			return UndefinedValue(), nil
		}
		return StringValue(fmt.Sprintf("%d", nodeID)), nil
	default:
		return UndefinedValue(), fmt.Errorf("host method is not configured")
	}
}

func TestDispatchSupportsDOMQueryHostCalls(t *testing.T) {
	store := dom.NewStore()
	if err := store.BootstrapHTML(`<main><section><p id="first">one</p></section><p id="second">two</p></main>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	firstID, ok, err := store.QuerySelector("#first")
	if err != nil {
		t.Fatalf("QuerySelector(#first) error = %v", err)
	}
	if !ok {
		t.Fatalf("QuerySelector(#first) ok = false, want true")
	}
	sectionID, ok, err := store.QuerySelector("section")
	if err != nil {
		t.Fatalf("QuerySelector(section) error = %v", err)
	}
	if !ok {
		t.Fatalf("QuerySelector(section) ok = false, want true")
	}

	runtime := NewRuntime(&domQueryHost{store: store})

	result, err := runtime.Dispatch(DispatchRequest{Source: `host:querySelector("#first")`})
	if err != nil {
		t.Fatalf("Dispatch(querySelector) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != fmt.Sprintf("%d", firstID) {
		t.Fatalf("Dispatch(querySelector) value = %#v, want node id string", result.Value)
	}

	result, err = runtime.Dispatch(DispatchRequest{Source: `host:querySelectorAll("main p")`})
	if err != nil {
		t.Fatalf("Dispatch(querySelectorAll) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 2 {
		t.Fatalf("Dispatch(querySelectorAll) value = %#v, want count 2", result.Value)
	}

	result, err = runtime.Dispatch(DispatchRequest{Source: fmt.Sprintf(`host:matches(%d, "main > section > p")`, firstID)})
	if err != nil {
		t.Fatalf("Dispatch(matches) error = %v", err)
	}
	if result.Value.Kind != ValueKindBool || !result.Value.Bool {
		t.Fatalf("Dispatch(matches) value = %#v, want true", result.Value)
	}

	result, err = runtime.Dispatch(DispatchRequest{Source: fmt.Sprintf(`host:closest(%d, "main > section")`, firstID)})
	if err != nil {
		t.Fatalf("Dispatch(closest) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != fmt.Sprintf("%d", sectionID) {
		t.Fatalf("Dispatch(closest) value = %#v, want section node id string", result.Value)
	}
}

func TestDispatchReturnsParseForMalformedSource(t *testing.T) {
	runtime := NewRuntime(nil)
	_, err := runtime.Dispatch(DispatchRequest{Source: "function foo("})
	if err == nil {
		t.Fatalf("Dispatch() error = nil, want parse error")
	}

	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch() error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch() error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestDispatchReturnsHostErrorsExplicitly(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{},
		errs: map[string]error{
			"boom": errors.New("host failed"),
		},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: "host:boom"})
	if err == nil {
		t.Fatalf("Dispatch() error = nil, want host error")
	}

	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch() error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindHost {
		t.Fatalf("Dispatch() error kind = %q, want %q", scriptErr.Kind, ErrorKindHost)
	}
}

func TestDispatchIsNilSafe(t *testing.T) {
	var runtime *Runtime

	if got, want := runtime.Config().StepLimit, DefaultRuntimeConfig().StepLimit; got != want {
		t.Fatalf("nil Config().StepLimit = %d, want %d", got, want)
	}

	_, err := runtime.Dispatch(DispatchRequest{Source: "noop"})
	if err == nil {
		t.Fatalf("nil Dispatch() error = nil, want runtime unavailable error")
	}

	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("nil Dispatch() error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("nil Dispatch() error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchValidatesHostMethodName(t *testing.T) {
	runtime := NewRuntime(&fakeHost{
		values: map[string]Value{},
		errs:   map[string]error{},
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: "host:   "})
	if err == nil {
		t.Fatalf("Dispatch(host:<blank>) error = nil, want parse error")
	}

	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(host:<blank>) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindParse {
		t.Fatalf("Dispatch(host:<blank>) error kind = %q, want %q", scriptErr.Kind, ErrorKindParse)
	}
}

func TestSplitScriptStatementsPreservesQuotedRegularExpressionLiteral(t *testing.T) {
	source := `const text = 'a"b'; document.getElementById("out").textContent = text.replace(/\"/g, "&quot;");`

	got, err := SplitScriptStatementsForRuntime(source)
	if err != nil {
		t.Fatalf("SplitScriptStatementsForRuntime() error = %v", err)
	}

	want := []string{
		`const text = 'a"b'`,
		`document.getElementById("out").textContent = text.replace(/\"/g, "&quot;")`,
	}
	if len(got) != len(want) {
		t.Fatalf("SplitScriptStatementsForRuntime() len = %d, want %d; got %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SplitScriptStatementsForRuntime()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitScriptStatementsPreservesRegexLiteralAfterWhitespace(t *testing.T) {
	source := `
  const one = (value) => {
    if (!value) return "";
    return value + "!";
  };
  const two = (value) => {
    if (!value) return false;
    return /^server\/?[^/]+$/.test(value);
  };
  const normalizedPath = one("a");
  const root = document.getElementById("root");
`

	got, err := SplitScriptStatementsForRuntime(source)
	if err != nil {
		t.Fatalf("SplitScriptStatementsForRuntime() error = %v", err)
	}

	want := []string{
		`const one = (value) => {
    if (!value) return "";
    return value + "!";
  }`,
		`const two = (value) => {
    if (!value) return false;
    return /^server\/?[^/]+$/.test(value);
  }`,
		`const normalizedPath = one("a")`,
		`const root = document.getElementById("root")`,
	}
	if len(got) != len(want) {
		t.Fatalf("SplitScriptStatementsForRuntime() len = %d, want %d; got %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SplitScriptStatementsForRuntime()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestClassicJSStatementParserPreservesRegexLiteralAfterWhitespace(t *testing.T) {
	checkString := func(t *testing.T, name, source, want string, fn func(*classicJSStatementParser) (string, error)) {
		t.Helper()

		parser := &classicJSStatementParser{source: source}
		got, err := fn(parser)
		if err != nil {
			t.Fatalf("%s error = %v", name, err)
		}
		if got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}

	t.Run("block", func(t *testing.T) {
		checkString(t, "consumeBlockSource", `{
  foo = /^server\/?[^/]+$/.test(value);
}`, `foo = /^server\/?[^/]+$/.test(value);`, func(parser *classicJSStatementParser) (string, error) {
			return parser.consumeBlockSource()
		})
	})

	t.Run("parenthesized", func(t *testing.T) {
		checkString(t, "consumeParenthesizedSource", `(
  foo = /^server\/?[^/]+$/.test(value)
)`, `foo = /^server\/?[^/]+$/.test(value)`, func(parser *classicJSStatementParser) (string, error) {
			return parser.consumeParenthesizedSource("test")
		})
	})

	t.Run("call-argument", func(t *testing.T) {
		checkString(t, "consumeAssignmentCallArgumentSource", `foo = /^server\/?[^/]+$/.test(value), next`, `foo = /^server\/?[^/]+$/.test(value)`, func(parser *classicJSStatementParser) (string, error) {
			return parser.consumeAssignmentCallArgumentSource()
		})
	})

	t.Run("array-destructuring", func(t *testing.T) {
		parser := &classicJSStatementParser{source: `[value = ` + "`" + `arr${` + "`" + `ay` + "`" + `}` + "`" + `] = rhs`}
		elements, _, ok, err := parser.tryParseArrayDestructuringAssignmentTarget()
		if err != nil {
			t.Fatalf("tryParseArrayDestructuringAssignmentTarget() error = %v", err)
		}
		if !ok {
			t.Fatalf("tryParseArrayDestructuringAssignmentTarget() ok = false, want true")
		}
		want := []string{`value = ` + "`" + `arr${` + "`" + `ay` + "`" + `}` + "`" + ``}
		if len(elements) != len(want) {
			t.Fatalf("tryParseArrayDestructuringAssignmentTarget() len = %d, want %d; got %#v", len(elements), len(want), elements)
		}
		for i := range want {
			if elements[i] != want[i] {
				t.Fatalf("tryParseArrayDestructuringAssignmentTarget()[%d] = %q, want %q", i, elements[i], want[i])
			}
		}
	})
}
