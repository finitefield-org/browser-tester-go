package script

import (
	"strings"
	"testing"
	"time"
)

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

func TestDispatchSupportsStringSplit(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["a.b".split(".").join("|"), "a,b".split(",").join("|"), "line1\r\nline2".split(/\r?\n/).join("|")].join("~")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.split) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.split) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "a|b~a|b~line1|line2" {
		t.Fatalf("Dispatch(String.split) value = %q, want %q", result.Value.String, "a|b~a|b~line1|line2")
	}
}

func TestDispatchSupportsStringReplaceCallbackReplacer(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["A1B2".replace(/([A-Z])([0-9])/g, (match, letter, digit, offset, input) => letter.toLowerCase() + digit + ":" + offset), "g".replace("g", (match, offset, input) => match + ":" + offset + ":" + input)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.replace callback) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.replace callback) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "a1:0b2:2|g:0:g" {
		t.Fatalf("Dispatch(String.replace callback) value = %q, want %q", result.Value.String, "a1:0b2:2|g:0:g")
	}
}

func TestDispatchSupportsStringMatchWithUnitSuffixCapture(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const match = "47.2in".match(/^([+-]?[0-9.,]+)(mm|cm|m|in|inch|ft|["'])?$/i); match ? match.slice(1).join("|") + "|" + match.length : "nil"`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.match capture) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.match capture) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "47.2|in|3" {
		t.Fatalf("Dispatch(String.match capture) value = %q, want %q", result.Value.String, "47.2|in|3")
	}
}

func TestDispatchSupportsStringCharCodeAt(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["A".charCodeAt(0), "１２３".charCodeAt(0), "A".charCodeAt(2)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.charCodeAt) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.charCodeAt) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "65|65297|NaN" {
		t.Fatalf("Dispatch(String.charCodeAt) value = %q, want %q", result.Value.String, "65|65297|NaN")
	}
}

func TestDispatchSupportsStringPadStart(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["7".padStart(3, "0"), "7".padStart(2, "0"), "7".padStart(5, "abc")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.padStart) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.padStart) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "007|07|abca7" {
		t.Fatalf("Dispatch(String.padStart) value = %q, want %q", result.Value.String, "007|07|abca7")
	}
}

func TestDispatchSupportsStringSliceWithUnicodeRunes(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `"アイウ".slice(0, 2)`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.slice) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.slice) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "アイ" {
		t.Fatalf("Dispatch(String.slice) value = %q, want %q", result.Value.String, "アイ")
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

func TestDispatchSupportsArrayFlatMap(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["north", "south"].flatMap((value) => [value]).join(",")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.flatMap) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.flatMap) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "north,south" {
		t.Fatalf("Dispatch(Array.flatMap) value = %q, want %q", result.Value.String, "north,south")
	}
}

func TestDispatchSupportsArrayFlat(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[["north"], ["south"]].flat().join(",")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.flat) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.flat) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "north,south" {
		t.Fatalf("Dispatch(Array.flat) value = %q, want %q", result.Value.String, "north,south")
	}
}

func TestDispatchSupportsArrayFlatDepthAndSparseSlots(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `let nested = []; nested[0] = 1; nested[1] = [2, [3]]; nested[2] = "skip"; delete nested[2]; nested[3] = [4]; nested.flat(2).join(",")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.flat depth/sparse slots) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.flat depth/sparse slots) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1,2,3,4" {
		t.Fatalf("Dispatch(Array.flat depth/sparse slots) value = %q, want %q", result.Value.String, "1,2,3,4")
	}
}

func TestDispatchSupportsArrayFill(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const values = [1, 2, 3]; const filled = values.fill(0, 1); [values.join(","), filled.join(","), [4, 5, 6].fill("x", 0, 2).join(",")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.fill) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.fill) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1,0,0|1,0,0|x,x,6" {
		t.Fatalf("Dispatch(Array.fill) value = %q, want %q", result.Value.String, "1,0,0|1,0,0|x,x,6")
	}
}

func TestDispatchRejectsInvalidArrayFillArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[1, 2, 3].fill()`})
	if err == nil {
		t.Fatalf("Dispatch(Array.fill()) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Array.fill expects a value") {
		t.Fatalf("Dispatch(Array.fill()) error = %q, want value error", got)
	}
}

func TestDispatchSupportsArraySortWithComparator(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const values = [3, 1, 2]; const sorted = values.sort((left, right) => left - right); [values.join(","), sorted.join(",")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.sort) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.sort) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1,2,3|1,2,3" {
		t.Fatalf("Dispatch(Array.sort) value = %q, want %q", result.Value.String, "1,2,3|1,2,3")
	}
}

func TestDispatchRejectsInvalidArraySortComparator(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[1, 2, 3].sort("oops")`})
	if err == nil {
		t.Fatalf("Dispatch(Array.sort()) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "non-callable value") {
		t.Fatalf("Dispatch(Array.sort()) error = %q, want callable value error", got)
	}
}

func TestDispatchSupportsArrayEvery(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[ [1, 2, 3].every(v => v > 0), [1, 2, 3].every(v => v > 1), [].every(v => false) ].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.every) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.every) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "true|false|true" {
		t.Fatalf("Dispatch(Array.every) value = %q, want %q", result.Value.String, "true|false|true")
	}
}

func TestDispatchRejectsInvalidArrayFlatMapArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[].flatMap()`})
	if err == nil {
		t.Fatalf("Dispatch(Array.flatMap()) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Array.flatMap expects a callback") {
		t.Fatalf("Dispatch(Array.flatMap()) error = %q, want callback error", got)
	}
}

func TestDispatchRejectsInvalidArrayEveryArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[].every()`})
	if err == nil {
		t.Fatalf("Dispatch(Array.every()) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Array.every expects a callback") {
		t.Fatalf("Dispatch(Array.every()) error = %q, want callback error", got)
	}
}

func TestDispatchSupportsArrayIndexOfAndLastIndexOf(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const values = ["alpha", "beta", "gamma", "beta"]; ["" + values.indexOf("beta"), "" + values.indexOf("beta", 2), "" + values.indexOf("beta", -2), "" + values.lastIndexOf("beta"), "" + values.lastIndexOf("beta", 2), "" + values.lastIndexOf("beta", -3)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.indexOf/lastIndexOf) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.indexOf/lastIndexOf) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|3|3|3|1|1" {
		t.Fatalf("Dispatch(Array.indexOf/lastIndexOf) value = %q, want %q", result.Value.String, "1|3|3|3|1|1")
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

func TestDispatchSupportsBrowserDateToLocaleDateString(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `date.toLocaleDateString("en-US")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.toLocaleDateString) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.toLocaleDateString) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "2/3/2024" {
		t.Fatalf("Dispatch(Date.toLocaleDateString) value = %q, want %q", result.Value.String, "2/3/2024")
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
