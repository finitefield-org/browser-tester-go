package script

import "testing"

func TestDispatchSupportsStringIndexOf(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["go".indexOf("o"), "go".indexOf("o", 2), "go".indexOf("", 5)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.indexOf) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.indexOf) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|-1|2" {
		t.Fatalf("Dispatch(String.indexOf) value = %q, want %q", result.Value.String, "1|-1|2")
	}
}

func TestDispatchSupportsStringStartsWith(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["go".startsWith("g"), "go".startsWith("o"), "go".startsWith("o", 1), "go".startsWith("", 2)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.startsWith) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.startsWith) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "true|false|true|true" {
		t.Fatalf("Dispatch(String.startsWith) value = %q, want %q", result.Value.String, "true|false|true|true")
	}
}

func TestDispatchRejectsInvalidStringStartsWithArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".startsWith()`})
	if err == nil {
		t.Fatalf("Dispatch(String.startsWith()) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.startsWith()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.startsWith()) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsStringEndsWith(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["go".endsWith("o"), "go".endsWith("g"), "go".endsWith("g", 1), "go".endsWith("", 1)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.endsWith) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.endsWith) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "true|false|true|true" {
		t.Fatalf("Dispatch(String.endsWith) value = %q, want %q", result.Value.String, "true|false|true|true")
	}
}

func TestDispatchRejectsInvalidStringEndsWithArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".endsWith()`})
	if err == nil {
		t.Fatalf("Dispatch(String.endsWith()) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.endsWith()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.endsWith()) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsArrayFindIndexSpliceAndUnshift(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			let a = [1, 2, 3];
			let idx = a.findIndex(v => v === 2);
			let removed = a.splice(1, 1, 9, 8);
			let len = a.unshift(0);
			[idx, removed.join(","), a.join(","), len].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.findIndex/splice/unshift) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.findIndex/splice/unshift) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|2|0,1,9,8,3|5" {
		t.Fatalf("Dispatch(Array.findIndex/splice/unshift) value = %q, want %q", result.Value.String, "1|2|0,1,9,8,3|5")
	}
}

func TestDispatchSupportsNumberToPrecisionAndToExponential(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[(0.0001).toExponential(2), (1.2).toPrecision(3), (1234).toPrecision(2)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Number.toPrecision/toExponential) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Number.toPrecision/toExponential) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1.00e-4|1.20|1.2e+3" {
		t.Fatalf("Dispatch(Number.toPrecision/toExponential) value = %q, want %q", result.Value.String, "1.00e-4|1.20|1.2e+3")
	}
}

func TestDispatchRejectsInvalidNumberToPrecision(t *testing.T) {
	runtime := NewRuntime(&echoHost{})

	_, err := runtime.Dispatch(DispatchRequest{Source: `(1.2).toPrecision(0)`})
	if err == nil {
		t.Fatalf("Dispatch(Number.toPrecision(0)) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(Number.toPrecision(0)) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(Number.toPrecision(0)) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchUpdatesNestedArrayBindings(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"state": ObjectValue([]ObjectEntry{
				{Key: "history", Value: ArrayValue([]Value{StringValue("a"), StringValue("b")})},
				{Key: "favorites", Value: ArrayValue([]Value{NumberValue(1), NumberValue(2), NumberValue(3)})},
			}),
		},
		Source: `state.history.push("c"); state.favorites.splice(1, 1, 9, 8); state.favorites.unshift(0); [state.history.join(","), state.favorites.join(",")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(nested array bindings) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(nested array bindings) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "a,b,c|0,1,9,8,3" {
		t.Fatalf("Dispatch(nested array bindings) value = %q, want %q", result.Value.String, "a,b,c|0,1,9,8,3")
	}
}
