package script

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

type browserCall struct {
	method string
	args   []Value
}

type browserBootstrapHost struct {
	calls            []browserCall
	resolvedPaths    []string
	documentLookups  []string
	documentElements map[string]Value
	localStorage     map[string]string
	sessionStorage   map[string]string
	clipboardWrites  []string
	matchMediaCalls  []string
	timerSources     []string
	microtaskSources []string
	historyURL       string
}

type promiseCaptureHost struct {
	capturedResolve Value
	echoes          []string
}

func (h *browserBootstrapHost) Call(method string, args []Value) (Value, error) {
	copiedArgs := make([]Value, len(args))
	copy(copiedArgs, args)
	h.calls = append(h.calls, browserCall{method: method, args: copiedArgs})

	switch method {
	case "echo":
		if len(args) != 1 {
			return UndefinedValue(), fmt.Errorf("echo expects 1 argument")
		}
		return args[0], nil
	default:
		return UndefinedValue(), fmt.Errorf("host method %q is not configured", method)
	}
}

func (h *promiseCaptureHost) Call(method string, args []Value) (Value, error) {
	switch method {
	case "captureResolve":
		if len(args) != 1 {
			return UndefinedValue(), fmt.Errorf("captureResolve expects 1 argument")
		}
		h.capturedResolve = args[0]
		return UndefinedValue(), nil
	case "echo":
		if len(args) != 1 {
			return UndefinedValue(), fmt.Errorf("echo expects 1 argument")
		}
		h.echoes = append(h.echoes, ToJSString(args[0]))
		return args[0], nil
	default:
		return UndefinedValue(), fmt.Errorf("host method %q is not configured", method)
	}
}

func (h *promiseCaptureHost) ResolveHostReference(path string) (Value, error) {
	switch path {
	case "host.echo":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("echo expects 1 argument")
			}
			h.echoes = append(h.echoes, ToJSString(args[0]))
			return args[0], nil
		}), nil
	default:
		return UndefinedValue(), fmt.Errorf("host reference %q is not configured", path)
	}
}

func (h *browserBootstrapHost) ResolveHostReference(path string) (Value, error) {
	h.resolvedPaths = append(h.resolvedPaths, path)

	switch path {
	case "document.getElementById":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("document.getElementById expects 1 argument")
			}
			id := ToJSString(args[0])
			h.documentLookups = append(h.documentLookups, id)
			if id != "agri-unit-converter-root" && id != "out" {
				return NullValue(), nil
			}
			if h.documentElements == nil {
				h.documentElements = map[string]Value{}
			}
			if element, ok := h.documentElements[id]; ok {
				return element, nil
			}
			textContent := ""
			if id == "agri-unit-converter-root" {
				textContent = "root"
			}
			element := ObjectValue([]ObjectEntry{
				{Key: "textContent", Value: StringValue(textContent)},
			})
			h.documentElements[id] = element
			return element, nil
		}), nil
	case "window.location":
		return ObjectValue([]ObjectEntry{
			{Key: "href", Value: StringValue("https://example.test/app?mode=initial")},
			{Key: "search", Value: StringValue("?mode=initial")},
		}), nil
	case "window.history":
		return ObjectValue([]ObjectEntry{
			{Key: "replaceState", Value: NativeFunctionValue(func(args []Value) (Value, error) {
				if len(args) != 3 {
					return UndefinedValue(), fmt.Errorf("history.replaceState expects 3 arguments")
				}
				h.historyURL = ToJSString(args[2])
				return UndefinedValue(), nil
			})},
		}), nil
	case "navigator.onLine":
		return BoolValue(true), nil
	case "URL":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("URL expects 1 argument")
			}
			href := ToJSString(args[0])
			return ObjectValue([]ObjectEntry{
				{Key: "href", Value: StringValue(href)},
				{Key: "toString", Value: NativeFunctionValue(func(args []Value) (Value, error) {
					return StringValue(href), nil
				})},
			}), nil
		}), nil
	case "Intl.NumberFormat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return ObjectValue([]ObjectEntry{
				{Key: "format", Value: NativeFunctionValue(func(args []Value) (Value, error) {
					if len(args) != 1 {
						return UndefinedValue(), fmt.Errorf("Intl.NumberFormat#format expects 1 argument")
					}
					return StringValue("1.23"), nil
				})},
			}), nil
		}), nil
	case "Number.EPSILON":
		return NumberValue(2.220446049250313e-16), nil
	case "Number.MAX_VALUE":
		return NumberValue(1.7976931348623157e308), nil
	case "Number.MIN_VALUE":
		return NumberValue(5e-324), nil
	case "Number.MAX_SAFE_INTEGER":
		return NumberValue(9007199254740991), nil
	case "Number.MIN_SAFE_INTEGER":
		return NumberValue(-9007199254740991), nil
	case "Number.isSafeInteger":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("Number.isSafeInteger expects 1 argument")
			}
			if args[0].Kind != ValueKindNumber {
				return BoolValue(false), nil
			}
			value := args[0].Number
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return BoolValue(false), nil
			}
			return BoolValue(math.Trunc(value) == value && math.Abs(value) <= 9007199254740991), nil
		}), nil
	case "localStorage.setItem":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 2 {
				return UndefinedValue(), fmt.Errorf("localStorage.setItem expects 2 arguments")
			}
			if h.localStorage == nil {
				h.localStorage = map[string]string{}
			}
			h.localStorage[ToJSString(args[0])] = ToJSString(args[1])
			return UndefinedValue(), nil
		}), nil
	case "sessionStorage.setItem":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 2 {
				return UndefinedValue(), fmt.Errorf("sessionStorage.setItem expects 2 arguments")
			}
			if h.sessionStorage == nil {
				h.sessionStorage = map[string]string{}
			}
			h.sessionStorage[ToJSString(args[0])] = ToJSString(args[1])
			return UndefinedValue(), nil
		}), nil
	case "matchMedia":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("matchMedia expects 1 argument")
			}
			query := ToJSString(args[0])
			h.matchMediaCalls = append(h.matchMediaCalls, query)
			return ObjectValue([]ObjectEntry{
				{Key: "matches", Value: BoolValue(true)},
				{Key: "media", Value: StringValue(query)},
			}), nil
		}), nil
	case "clipboard.writeText":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("clipboard.writeText expects 1 argument")
			}
			h.clipboardWrites = append(h.clipboardWrites, ToJSString(args[0]))
			return PromiseValue(UndefinedValue()), nil
		}), nil
	case "setTimeout":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 2 {
				return UndefinedValue(), fmt.Errorf("setTimeout expects 2 arguments")
			}
			h.timerSources = append(h.timerSources, ToJSString(args[0]))
			return NumberValue(1), nil
		}), nil
	case "queueMicrotask":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("queueMicrotask expects 1 argument")
			}
			h.microtaskSources = append(h.microtaskSources, ToJSString(args[0]))
			return UndefinedValue(), nil
		}), nil
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
}

func TestDispatchSkipsUnsupportedBrowserSurfaceInUntakenBranch(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"window":         HostObjectReference("window"),
		"document":       HostObjectReference("document"),
		"navigator":      HostObjectReference("navigator"),
		"localStorage":   HostObjectReference("localStorage"),
		"sessionStorage": HostObjectReference("sessionStorage"),
		"matchMedia":     HostFunctionReference("matchMedia"),
		"clipboard":      HostObjectReference("clipboard"),
		"URL":            HostConstructorReference("URL"),
		"Intl":           HostObjectReference("Intl"),
		"setTimeout":     HostFunctionReference("setTimeout"),
		"queueMicrotask": HostFunctionReference("queueMicrotask"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const root = document.getElementById("agri-unit-converter-root");
			host.echo(root.textContent.length)
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(raw browser globals) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 4 {
		t.Fatalf("Dispatch(raw browser globals) value = %#v, want number 4", result.Value)
	}
	if len(host.documentLookups) != 1 || host.documentLookups[0] != "agri-unit-converter-root" {
		t.Fatalf("document.getElementById calls = %#v, want the agri root lookup", host.documentLookups)
	}
}

func TestDispatchSupportsAssignmentThroughDocumentGetElementByIdCall(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"document": HostObjectReference("document"),
	})

	result, err := runtime.Dispatch(DispatchRequest{Source: `document.getElementById("out").textContent = "assigned"`})
	if err != nil {
		t.Fatalf("Dispatch(document.getElementById().textContent assignment) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "assigned" {
		t.Fatalf("Dispatch(document.getElementById().textContent assignment) value = %#v, want string assigned", result.Value)
	}
	if len(host.documentLookups) != 1 || host.documentLookups[0] != "out" {
		t.Fatalf("document.getElementById calls = %#v, want the out lookup", host.documentLookups)
	}
	element, ok := host.documentElements["out"]
	if !ok {
		t.Fatalf("documentElements[out] missing after assignment")
	}
	if len(element.Object) != 1 || element.Object[0].Key != "textContent" || element.Object[0].Value.Kind != ValueKindString || element.Object[0].Value.String != "assigned" {
		t.Fatalf("documentElements[out] = %#v, want textContent assigned", element)
	}
}

func TestDispatchRejectsAssignmentThroughDocumentGetElementByIdCallOnMissingElement(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"document": HostObjectReference("document"),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `document.getElementById("missing").textContent = "assigned"`})
	if err == nil {
		t.Fatalf("Dispatch(document.getElementById().textContent assignment on missing element) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(document.getElementById().textContent assignment on missing element) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(document.getElementById().textContent assignment on missing element) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if len(host.documentLookups) != 1 || host.documentLookups[0] != "missing" {
		t.Fatalf("document.getElementById calls = %#v, want the missing lookup", host.documentLookups)
	}
	if _, ok := host.documentElements["missing"]; ok {
		t.Fatalf("documentElements[missing] unexpectedly created after failed assignment")
	}
}

func TestDispatchSupportsBuiltinMapSlice(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const pickMap = new Map();
			pickMap.set("sku-1", 12);
			pickMap.set("sku-2", 5);
			const deleted = pickMap.delete("sku-1", "extra");
			const missing = pickMap.delete("missing", "extra");
			[
				"" + deleted,
				"" + missing,
				"" + pickMap.size,
				"" + pickMap.get("sku-2"),
				typeof pickMap.get,
			].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Map slice) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Map slice) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "true|false|1|5|function" {
		t.Fatalf("Dispatch(Map slice) value = %q, want true|false|1|5|function", result.Value.String)
	}
}

func TestDispatchSupportsBuiltinMapForEach(t *testing.T) {
	host := &promiseCaptureHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"Map": BuiltinMapValue(),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const pickMap = new Map([["sku-1", 12], ["sku-2", 5]]);
			const context = { prefix: "ctx" };
			let seen = "";
			pickMap.forEach(function (value, key, mapObject) {
				seen = seen + (seen === "" ? "" : "|") + host.echo(this.prefix + ":" + key + ":" + value + ":" + mapObject.get(key));
			}, context);
			seen
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Map.forEach) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Map.forEach) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "ctx:sku-1:12:12|ctx:sku-2:5:5" {
		t.Fatalf("Dispatch(Map.forEach) value = %q, want ctx:sku-1:12:12|ctx:sku-2:5:5", result.Value.String)
	}
	if got, want := strings.Join(host.echoes, "|"), "ctx:sku-1:12:12|ctx:sku-2:5:5"; got != want {
		t.Fatalf("Dispatch(Map.forEach) host echoes = %q, want %q", got, want)
	}
}

func TestDispatchSupportsBuiltinMapGetArrayMutationAliases(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const grouped = new Map();
			grouped.set("bucket", []);
			const bucket = grouped.get("bucket");
			bucket.push("alpha");
			const firstRead = grouped.get("bucket").length;
			grouped.get("bucket").push("beta");
			[firstRead, bucket.length, grouped.get("bucket").length, bucket.join(",")].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Map.get array mutation aliases) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Map.get array mutation aliases) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|2|2|alpha,beta" {
		t.Fatalf("Dispatch(Map.get array mutation aliases) value = %q, want 1|2|2|alpha,beta", result.Value.String)
	}
}

func TestDispatchSupportsBuiltinMapAndSetIterators(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
		"Set": BuiltinSetValue(),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const map = new Map([["left", 1], ["right", 2]]);
			const set = new Set(["alpha", "beta"]);
			const mapValues = map.values();
			const mapKeys = map.keys();
			const mapEntries = map.entries();
			const setValues = set.values();
			const setKeys = set.keys();
			const setEntries = set.entries();
			[
				"" + mapValues.next().value,
				"" + mapValues.next().value,
				"" + mapKeys.next().value,
				"" + mapKeys.next().done,
				mapEntries.next().value.join("="),
				mapEntries.next().value.join("="),
				"" + mapEntries.next().done,
				"" + setValues.next().value,
				"" + setKeys.next().value,
				setEntries.next().value.join("="),
				setEntries.next().value.join("="),
				"" + setEntries.next().done,
			].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Map and Set iterators) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Map and Set iterators) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != `1|2|left|false|left=1|right=2|true|alpha|alpha|alpha=alpha|beta=beta|true` {
		t.Fatalf("Dispatch(Map and Set iterators) value = %q, want Map and Set iterator parity", result.Value.String)
	}
}

func TestDispatchSupportsNumberSafeIntegerSurface(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"Number": HostObjectReference("Number"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			[
				Number.EPSILON === 2.220446049250313e-16,
				Number.MAX_VALUE === 1.7976931348623157e308,
				Number.MIN_VALUE === 5e-324,
				Number.MAX_SAFE_INTEGER === 9007199254740991,
				Number.MIN_SAFE_INTEGER === -9007199254740991,
				Number.isSafeInteger(Number.MAX_SAFE_INTEGER),
				Number.isSafeInteger(Number.MAX_SAFE_INTEGER + 1),
				Number.isSafeInteger(1.5),
				Number.isSafeInteger("42"),
			].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Number safe-integer surface) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Number safe-integer surface) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "true|true|true|true|true|true|false|false|false" {
		t.Fatalf("Dispatch(Number safe-integer surface) value = %q, want safe-integer surface parity", result.Value.String)
	}
}

func TestDispatchRejectsNumberIsSafeIntegerArityMismatch(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"Number": HostObjectReference("Number"),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `Number.isSafeInteger(1, 2)`})
	if err == nil {
		t.Fatalf("Dispatch(Number.isSafeInteger(1, 2)) error = nil, want arity failure")
	}
	if !strings.Contains(err.Error(), "Number.isSafeInteger expects 1 argument") {
		t.Fatalf("Dispatch(Number.isSafeInteger(1, 2)) error = %v, want Number.isSafeInteger arity failure", err)
	}
}

func TestDispatchSupportsBuiltinMapClear(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const map = new Map([["left", 1], ["right", 2]]);
			const cleared = map.clear();
			[map.size, map.has("left"), map.has("right"), typeof cleared].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Map.clear) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Map.clear) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "0|false|false|undefined" {
		t.Fatalf("Dispatch(Map.clear) value = %q, want %q", result.Value.String, "0|false|false|undefined")
	}
}

func TestDispatchSupportsBuiltinSetClear(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Set": BuiltinSetValue(),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const set = new Set(["left", "right"]);
			const cleared = set.clear();
			[set.size, set.has("left"), set.has("right"), typeof cleared].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Set.clear) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Set.clear) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "0|false|false|undefined" {
		t.Fatalf("Dispatch(Set.clear) value = %q, want %q", result.Value.String, "0|false|false|undefined")
	}
}

func TestDispatchRejectsBuiltinMapAndSetIteratorsWithArguments(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
		"Set": BuiltinSetValue(),
	})

	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "map-values",
			source: `new Map().values(1)`,
			want:   "Map.values expects no arguments",
		},
		{
			name:   "set-entries",
			source: `new Set().entries(1)`,
			want:   "Set.entries expects no arguments",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err == nil {
				t.Fatalf("Dispatch(%s) error = nil, want arity failure", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Dispatch(%s) error = %v, want message containing %q", tc.name, err, tc.want)
			}
		})
	}
}

func TestDispatchRejectsBuiltinMapClearWithArguments(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `new Map().clear(1)`})
	if err == nil {
		t.Fatalf("Dispatch(new Map().clear(1)) error = nil, want arity failure")
	}
	if !strings.Contains(err.Error(), "Map.clear expects no arguments") {
		t.Fatalf("Dispatch(new Map().clear(1)) error = %v, want clear arity failure", err)
	}
}

func TestDispatchRejectsBuiltinSetClearWithArguments(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Set": BuiltinSetValue(),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `new Set().clear(1)`})
	if err == nil {
		t.Fatalf("Dispatch(new Set().clear(1)) error = nil, want arity failure")
	}
	if !strings.Contains(err.Error(), "Set.clear expects no arguments") {
		t.Fatalf("Dispatch(new Set().clear(1)) error = %v, want clear arity failure", err)
	}
}

func TestDispatchRejectsBuiltinMapForEachWithoutCallback(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `new Map().forEach()`})
	if err == nil {
		t.Fatalf("Dispatch(new Map().forEach()) error = nil, want callback error")
	}
	if !strings.Contains(err.Error(), "Map.forEach expects a callback") {
		t.Fatalf("Dispatch(new Map().forEach()) error = %v, want callback error", err)
	}
}

func TestDispatchRejectsMapCallWithoutNew(t *testing.T) {
	runtime := NewRuntimeWithBindings(nil, map[string]Value{
		"Map": BuiltinMapValue(),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `Map()`})
	if err == nil {
		t.Fatalf("Dispatch(Map()) error = nil, want constructor error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(Map()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(Map()) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
	if !strings.Contains(scriptErr.Message, "called with `new`") {
		t.Fatalf("Dispatch(Map()) error message = %q, want constructor error", scriptErr.Message)
	}
}

func TestDispatchSupportsNewBrowserConstructorsWithMemberChains(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"window": HostObjectReference("window"),
		"URL":    HostConstructorReference("URL"),
		"Intl":   HostObjectReference("Intl"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const current = new URL(window.location.href); const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23); host.echo(current.href); host.echo(formatted)`,
	})
	if err != nil {
		t.Fatalf("Dispatch(new browser constructors) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "1.23" {
		t.Fatalf("Dispatch(new browser constructors) value = %#v, want formatted string", result.Value)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two echo calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "https://example.test/app?mode=initial" {
		t.Fatalf("host.calls[0].args = %#v, want initial URL string", host.calls[0].args)
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "1.23" {
		t.Fatalf("host.calls[1].args = %#v, want formatted number", host.calls[1].args)
	}
}

func TestDispatchSupportsReassignedIntlBinding(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"Intl": HostObjectReference("Intl"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `Intl = { NumberFormat: function () { return { format: function () { return "ok"; } }; } }; new Intl.NumberFormat("en-US", {}).format(1)`,
	})
	if err != nil {
		t.Fatalf("Dispatch(reassigned Intl binding) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "ok" {
		t.Fatalf("Dispatch(reassigned Intl binding) value = %#v, want string ok", result.Value)
	}
}

func TestDispatchReportsUnsupportedBrowserSurfaceDirectly(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"window": HostObjectReference("window"),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `window.crypto`})
	if err == nil {
		t.Fatalf("Dispatch(window.crypto) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(window.crypto) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(window.crypto) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if !strings.Contains(scriptErr.Message, "window.crypto") {
		t.Fatalf("Dispatch(window.crypto) error = %q, want browser-surface path", scriptErr.Message)
	}
}

func TestDispatchReportsUnsupportedDocumentSurfaceDirectly(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"document": HostObjectReference("document"),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `document.title`})
	if err == nil {
		t.Fatalf("Dispatch(document.title) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(document.title) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(document.title) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if !strings.Contains(scriptErr.Message, "document.title") {
		t.Fatalf("Dispatch(document.title) error = %q, want browser-surface path", scriptErr.Message)
	}
}

func TestDispatchTreatsMissingHostReferencePropertiesAsAbsentInOperator(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"navigator": HostObjectReference("navigator"),
		"window":    HostObjectReference("window"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `host.echo(("missingFeature" in navigator) + "|" + ("missingFeature" in window))`,
	})
	if err != nil {
		t.Fatalf("Dispatch(host-reference in operator) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "false|false" {
		t.Fatalf("Dispatch(host-reference in operator) value = %#v, want string false|false", result.Value)
	}
}

func TestDispatchSupportsBrowserPromiseThenCatch(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"clipboard": HostObjectReference("clipboard"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `clipboard.writeText("copied").then(function () { host.echo("done") }).catch(function () { host.echo("caught") }); "done"`,
	})
	if err != nil {
		t.Fatalf("Dispatch(browser promise chain) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "done" {
		t.Fatalf("Dispatch(browser promise chain) value = %#v, want string done", result.Value)
	}
	if len(host.clipboardWrites) != 1 || host.clipboardWrites[0] != "copied" {
		t.Fatalf("clipboard writes = %#v, want one copied write", host.clipboardWrites)
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one echo call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "done" {
		t.Fatalf("host.calls[0].args = %#v, want string done", host.calls[0].args)
	}
}

func TestDispatchSupportsRejectedBrowserPromiseCatchChain(t *testing.T) {
	host := &promiseCaptureHost{}
	promise, _, rejectPromise := NewPendingPromiseWithReject()
	rejectPromise(StringValue("boom"))
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"host":    HostObjectReference("host"),
		"promise": promise,
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `promise.catch(function (reason) { host.echo(reason); return "handled:" + reason; }).then(function (value) { host.echo(value); }); "done"`,
	})
	if err != nil {
		t.Fatalf("Dispatch(rejected browser promise chain) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "done" {
		t.Fatalf("Dispatch(rejected browser promise chain) value = %#v, want string done", result.Value)
	}
	if got, want := strings.Join(host.echoes, "|"), "boom|handled:boom"; got != want {
		t.Fatalf("host echoes = %q, want %q", got, want)
	}
}

func TestDispatchPropagatesPromiseThenCallbackErrors(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"clipboard": HostObjectReference("clipboard"),
	})

	_, err := runtime.Dispatch(DispatchRequest{
		Source: `clipboard.writeText("copied").then(function () { throw "boom" })`,
	})
	if err == nil {
		t.Fatalf("Dispatch(browser promise then callback error) error = nil, want runtime error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Dispatch(browser promise then callback error) error = %q, want thrown value", err)
	}
	if len(host.clipboardWrites) != 1 || host.clipboardWrites[0] != "copied" {
		t.Fatalf("clipboard writes = %#v, want one copied write", host.clipboardWrites)
	}
}

func TestDispatchPreservesHostBindingsForPromiseThenContinuationResolvedFromGo(t *testing.T) {
	host := &promiseCaptureHost{}
	promise, resolvePromise := NewPendingPromise()
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"promise": promise,
	})

	if _, err := runtime.Dispatch(DispatchRequest{
		Source: `promise.then(function (value) { host.echo(value); return value; }); "queued"`,
	}); err != nil {
		t.Fatalf("Dispatch(promise.then) error = %v", err)
	}

	resolvePromise(StringValue("ready"))

	if got, want := strings.Join(host.echoes, "|"), "ready"; got != want {
		t.Fatalf("promise then host echoes = %q, want %q", got, want)
	}
}

func TestDispatchResumesAwaitOnManuallyResolvedPendingPromise(t *testing.T) {
	host := &promiseCaptureHost{}
	promise, resolvePromise := NewPendingPromise()
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"host":    HostObjectReference("host"),
		"promise": promise,
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `async function run() { await promise; host.echo("done"); } run()`,
	})
	if err != nil {
		t.Fatalf("Dispatch(pending promise await) error = %v", err)
	}
	if result.Value.Kind != ValueKindPromise {
		t.Fatalf("Dispatch(pending promise await) kind = %q, want promise", result.Value.Kind)
	}
	if result.Value.PromiseState == nil || result.Value.PromiseState.resolved {
		t.Fatalf("Dispatch(pending promise await) state = %#v, want pending promise state", result.Value)
	}
	if len(host.echoes) != 0 {
		t.Fatalf("host echoes before resolve = %#v, want none", host.echoes)
	}

	resolvePromise(StringValue("ready"))

	if len(host.echoes) != 1 || host.echoes[0] != "done" {
		t.Fatalf("host echoes after resolve = %#v, want one done echo", host.echoes)
	}
}

func TestPendingPromiseAwaitContinuationStateResumesManually(t *testing.T) {
	host := &promiseCaptureHost{}
	promise, resolvePromise := NewPendingPromise()
	env := newClassicJSEnvironment()
	if err := env.declare("host", scalarJSValue(HostObjectReference("host")), true); err != nil {
		t.Fatalf("declare(host) error = %v", err)
	}
	if err := env.declare("promise", scalarJSValue(promise), true); err != nil {
		t.Fatalf("declare(promise) error = %v", err)
	}

	parser := &classicJSStatementParser{
		host:        host,
		env:         env,
		allowAwait:  true,
		allowReturn: true,
	}

	_, err := evalClassicJSProgramWithAllowAwaitAndYieldAndExports(`await promise; host.echo("done")`, host, env, DefaultRuntimeConfig().StepLimit, true, false, false, nil, UndefinedValue(), false, nil, nil)
	if err == nil {
		t.Fatalf("evalClassicJSProgramWithAllowAwaitAndYieldAndExports error = nil, want await suspension")
	}
	awaitedPromise, resumeState, ok := classicJSAwaitSignalDetails(err)
	if !ok {
		t.Fatalf("await signal details = false, want await suspension")
	}
	if awaitedPromise == nil {
		t.Fatalf("awaitedPromise = nil, want pending promise state")
	}
	if resumeState == nil {
		t.Fatalf("resumeState = nil, want continuation state")
	}
	t.Logf("resumeState type = %T", resumeState)
	if block, ok := resumeState.(*classicJSBlockState); ok {
		t.Logf("resumeState.index=%d len=%d child=%T owner=%T statements=%#v", block.index, len(block.statements), block.child, block.owner, block.statements)
	}
	if len(host.echoes) != 0 {
		t.Fatalf("host echoes before resolve = %#v, want none", host.echoes)
	}

	resolvePromise(StringValue("ready"))
	parser.resumeState = resumeState
	value, nextState, err := parser.resumeClassicJSState(resumeState)
	if err != nil {
		t.Fatalf("resumeClassicJSState() error = %v", err)
	}
	if nextState != nil {
		t.Fatalf("resumeClassicJSState() nextState = %#v, want nil", nextState)
	}
	if value.Kind != ValueKindString || value.String != "done" {
		t.Fatalf("resumeClassicJSState() value = %#v, want \"done\"", value)
	}
	if len(host.echoes) != 1 || host.echoes[0] != "done" {
		t.Fatalf("host echoes after manual resume = %#v, want one done echo", host.echoes)
	}
}

func TestPendingPromiseAwaitDeclarationContinuationResumesFollowingStatement(t *testing.T) {
	host := &promiseCaptureHost{}
	promise, resolvePromise := NewPendingPromise()
	env := newClassicJSEnvironment()
	if err := env.declare("host", scalarJSValue(HostObjectReference("host")), true); err != nil {
		t.Fatalf("declare(host) error = %v", err)
	}
	if err := env.declare("promise", scalarJSValue(promise), true); err != nil {
		t.Fatalf("declare(promise) error = %v", err)
	}

	parser := &classicJSStatementParser{
		host:        host,
		env:         env,
		allowAwait:  true,
		allowReturn: true,
	}

	_, err := evalClassicJSProgramWithAllowAwaitAndYieldAndExports(`const value = await promise; host.echo(value)`, host, env, DefaultRuntimeConfig().StepLimit, true, false, false, nil, UndefinedValue(), false, nil, nil)
	if err == nil {
		t.Fatalf("evalClassicJSProgramWithAllowAwaitAndYieldAndExports error = nil, want await suspension")
	}
	awaitedPromise, resumeState, ok := classicJSAwaitSignalDetails(err)
	if !ok {
		t.Fatalf("await signal details = false, want await suspension")
	}
	if awaitedPromise == nil {
		t.Fatalf("awaitedPromise = nil, want pending promise state")
	}
	if resumeState == nil {
		t.Fatalf("resumeState = nil, want continuation state")
	}
	resolvePromise(StringValue("ready"))
	parser.resumeState = resumeState
	parser.generatorNextValue = StringValue("ready")
	parser.hasGeneratorNextValue = true
	_, nextState, err := parser.resumeClassicJSState(resumeState)
	if err != nil {
		t.Fatalf("resumeClassicJSState() error = %v", err)
	}
	if nextState != nil {
		t.Fatalf("resumeClassicJSState() nextState = %#v, want nil", nextState)
	}
	if len(host.echoes) != 1 || host.echoes[0] != "ready" {
		t.Fatalf("host echoes after manual resume = %#v, want one ready echo", host.echoes)
	}
}
