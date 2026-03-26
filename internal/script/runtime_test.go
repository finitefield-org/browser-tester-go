package script

import (
	"errors"
	"fmt"
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

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.setTextContent("#out", "first"); host.setTextContent("#out", "second")`})
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

func TestDispatchSupportsObjectLiteralShorthandPropertiesAndMethodsInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let value = "seed"; let obj = { value, read() { return this.value } }; host.echo(obj.value, obj.read())`})
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

func TestDispatchSupportsObjectPropertyAssignmentInClassicJS(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { value: "seed", nested: { count: 1 } }; obj.value = "updated"; obj.nested.count = obj.nested.count + 1; host.echo(obj.value, obj.nested.count)`})
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

func TestDispatchRejectsAssignmentToGetterOnlyObjectPropertiesInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { get value() { return "seed" } }; obj.value = "updated"`})
	if err == nil {
		t.Fatalf("Dispatch(assignment to getter-only object property) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(assignment to getter-only object property) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(assignment to getter-only object property) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let keepGoing = true; keepGoing; keepGoing &&= false) { host.setTextContent("#out", "ran") }`})
	if err != nil {
		t.Fatalf("Dispatch(for loop) error = %v", err)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one call", host.calls)
	}
	if host.calls[0].method != "setTextContent" {
		t.Fatalf("host.calls[0].method = %q, want setTextContent", host.calls[0].method)
	}
	if host.calls[0].args[1].Kind != ValueKindString || host.calls[0].args[1].String != "ran" {
		t.Fatalf("host.calls[0].args[1] = %#v, want ran", host.calls[0].args[1])
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

func TestDispatchRejectsForOfOverNonArrays(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `for (let value of 1) { value }`})
	if err == nil {
		t.Fatalf("Dispatch(for...of over non-array) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for...of over non-array) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(for...of over non-array) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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
		t.Fatalf("Dispatch(for...in over non-object) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(for...in over non-object) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(for...in over non-object) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

func TestDispatchReportsUncaughtBreakAndContinueAsRuntimeErrorsInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	for _, source := range []string{`break`, `continue`} {
		_, err := runtime.Dispatch(DispatchRequest{Source: source})
		if err == nil {
			t.Fatalf("Dispatch(%s) error = nil, want runtime error", source)
		}
		scriptErr, ok := err.(Error)
		if !ok {
			t.Fatalf("Dispatch(%s) error type = %T, want script.Error", source, err)
		}
		if scriptErr.Kind != ErrorKindRuntime {
			t.Fatalf("Dispatch(%s) error kind = %q, want %q", source, scriptErr.Kind, ErrorKindRuntime)
		}
	}
}

func TestDispatchRejectsContinueTargetingLabeledSwitchInClassicJS(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `outer: switch (true) { case true: continue outer }`})
	if err == nil {
		t.Fatalf("Dispatch(continue targeting labeled switch) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(continue targeting labeled switch) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(continue targeting labeled switch) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

func TestDispatchRejectsConstructorArgumentsInNewExpressions(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"setTextContent": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `class Example { value = "field" }; new Example(1)`})
	if err == nil {
		t.Fatalf("Dispatch(new expression with constructor args) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(new expression with constructor args) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(new expression with constructor args) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

func TestDispatchRejectsLogicalAssignmentOnGetterOnlyObjectProperty(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let obj = { get value() { return "" } }; obj.value ||= "fresh"`})
	if err == nil {
		t.Fatalf("Dispatch(logical assignment on getter-only object property) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(logical assignment on getter-only object property) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(logical assignment on getter-only object property) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

func TestDispatchRejectsReservedVarDeclarationNames(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `var let = 1`})
	if err == nil {
		t.Fatalf("Dispatch(reserved var declaration name) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(reserved var declaration name) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(reserved var declaration name) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

func TestDispatchSupportsArrowFunctions(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": StringValue("ok"),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	result, err := runtime.Dispatch(DispatchRequest{Source: `let identity = x => x; let collect = (...items) => items; host.echo(identity("fine"), collect(1, 2, 3))`})
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
	if len(call.args) != 2 {
		t.Fatalf("host call args len = %d, want 2", len(call.args))
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
		t.Fatalf("Dispatch(top-level return) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(top-level return) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(top-level return) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsReturnInsideClassStaticBlocks(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `function outer() { class Example { static { return 1 } } }; outer()`})
	if err == nil {
		t.Fatalf("Dispatch(return inside class static block) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(return inside class static block) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(return inside class static block) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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
		t.Fatalf("Dispatch(reserved generator function name) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(reserved generator function name) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(reserved generator function name) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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
		t.Fatalf("generator delegation next error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("generator delegation next error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("generator delegation next error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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
		t.Fatalf("Dispatch(arrow function reserved parameter) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(arrow function reserved parameter) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(arrow function reserved parameter) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

func TestDispatchSupportsDynamicImportExpressionsInClassicJS(t *testing.T) {
	host := &echoHost{}
	runtime := NewRuntime(host)

	bindings := map[string]Value{
		"math": ObjectValue([]ObjectEntry{
			{Key: "default", Value: StringValue("seeded")},
			{Key: "value", Value: NumberValue(7)},
		}),
	}

	got, err := runtime.Dispatch(DispatchRequest{Source: `await import("math")`, Bindings: bindings})
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
		t.Fatalf("Dispatch(yield outside generator) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(yield outside generator) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(yield outside generator) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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
		t.Fatalf("Dispatch(spread on scalar) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(spread on scalar) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(spread on scalar) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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
		t.Fatalf("Dispatch(in operator on non-object) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(in operator on non-object) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(in operator on non-object) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsInstanceofOnNonClassObjects(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `({}) instanceof ({})`})
	if err == nil {
		t.Fatalf("Dispatch(instanceof on non-class objects) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(instanceof on non-class objects) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(instanceof on non-class objects) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsReservedTypeofDeclarationNames(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `let typeof = 1`})
	if err == nil {
		t.Fatalf("Dispatch(reserved typeof declaration name) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(reserved typeof declaration name) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(reserved typeof declaration name) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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

func TestDispatchRejectsMemberAccessOnUnsupportedScalarValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo((1).foo)`})
	if err == nil {
		t.Fatalf("Dispatch(member access on unsupported scalar) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(member access on unsupported scalar) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(member access on unsupported scalar) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
}

func TestDispatchRejectsBracketAccessOnUnsupportedScalarValues(t *testing.T) {
	host := &fakeHost{
		values: map[string]Value{
			"echo": UndefinedValue(),
		},
		errs: map[string]error{},
	}
	runtime := NewRuntime(host)

	_, err := runtime.Dispatch(DispatchRequest{Source: `host.echo((1)?.[0])`})
	if err == nil {
		t.Fatalf("Dispatch(bracket access on unsupported scalar) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(bracket access on unsupported scalar) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(bracket access on unsupported scalar) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
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
