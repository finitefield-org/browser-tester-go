package runtime

import (
	"math"
	"math/bits"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func TestRunScriptSupportsBrowserStdlibSlice(t *testing.T) {
	session := NewSession(SessionConfig{})
	session.SetNowMs(1700000000000)
	wantLocaleDate := time.UnixMilli(1700000000000).UTC().Format("1/2/2006")

	result, err := session.runScriptOnStore(dom.NewStore(), `
		let date = new Date(Date.now());
		let assigned = Object.assign({ first: "a" }, { second: "b" });
		let parsed = JSON.parse("{\"a\":1,\"b\":[2,3]}");
		let items = [1, 2, 3];
		let pushed = [1, 2];
		pushed.push(3);
		let seen = "";
		[1, 2, 3].forEach(v => seen = seen + v);
		[
			Array.from("go").join("|"),
			Array.isArray(items) ? "true" : "false",
			Object.keys(assigned).join(","),
			Object.entries(assigned).map(entry => entry.join("=")).join(","),
			Object.values(assigned).join(","),
			new Intl.DateTimeFormat("en-US-u-nu-latn", { timeZone: "America/Chicago", year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false }).formatToParts(new Date(Date.UTC(2026, 0, 21, 8, 45, 0, 0))).find(part => part.type === "hour").value,
			Object.fromEntries([["left", "l"], ["right", "r"]]).left + "|" + Object.fromEntries(new Map([["first", 1], ["second", 2]])).second,
			Object.keys(date).length + "|" + Object.keys(Object.assign({}, date)).length,
			JSON.stringify(parsed),
			JSON.stringify(date),
			date.toISOString(),
			date.toLocaleDateString("en-US"),
			Math.abs(-4) + "|" + Math.pow(2, 3) + "|" + Math.min(3, 1, 2) + "|" + Math.max(3, 1, 2) + "|" + Math.ceil(1.1) + "|" + Math.ceil(-1.1) + "|" + Math.ceil(-0.1) + "|" + Math.floor(1.9) + "|" + Math.floor(-1.1) + "|" + Math.trunc(1.9) + "|" + Math.trunc(-1.9),
			Number.isFinite(1) + "|" + Number.isFinite(Number.NaN) + "|" + Number.isNaN(Number.NaN),
			encodeURI("https://example.com/A B?x=春&y=1#frag") + "|" + decodeURI("https://example.com/%3F%23%26%20x") + "|" + encodeURIComponent("A&B 春") + "|" + decodeURIComponent("A%26B%20%E6%98%A5"),
			CSS.escape(),
			String(true) + "|" + Boolean(0).valueOf(),
			Number(15).toString(16) + "|" + (15).valueOf(),
			"  Go  ".trim().toLowerCase() + "|" + "go".replace("g", "n") + "|" + "ab".lastIndexOf("b") + "|" + "go".slice(1),
			["alpha", "beta", "gamma", "beta"].indexOf("beta") + "|" + ["alpha", "beta", "gamma", "beta"].indexOf("beta", 2) + "|" + ["alpha", "beta", "gamma", "beta"].indexOf("beta", -2) + "|" + ["alpha", "beta", "gamma", "beta"].lastIndexOf("beta") + "|" + ["alpha", "beta", "gamma", "beta"].lastIndexOf("beta", 2) + "|" + ["alpha", "beta", "gamma", "beta"].lastIndexOf("beta", -3),
			items.slice(1).join(",") + "|" + items.filter(v => v > 1).join(",") + "|" + items.map(v => v * 2).join(",") + "|" + items.some(v => v === 3) + "|" + items.includes(2) + "|" + pushed.join(",") + "|" + items.find(v => v > 1),
			seen
		].join("~");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}

	wantDate := time.UnixMilli(1700000000000).UTC().Format("2006-01-02T15:04:05.000Z")
	want := strings.Join([]string{
		"g|o",
		"true",
		"first,second",
		"first=a,second=b",
		"a,b",
		"02",
		"l|2",
		"0|0",
		`{"a":1,"b":[2,3]}`,
		strconv.Quote(wantDate),
		wantDate,
		wantLocaleDate,
		"4|8|1|3|2|-1|0|1|-2|1|-1",
		"true|false|true",
		"https://example.com/A%20B?x=%E6%98%A5&y=1#frag|https://example.com/%3F%23%26 x|A%26B%20%E6%98%A5|A&B 春",
		"undefined",
		"true|false",
		"f|15",
		"go|no|1|o",
		"1|3|3|3|1|1",
		"2,3|2,3|2,4,6|true|true|1,2,3|2",
		"123",
	}, "~")

	if result.String != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", result.String, want)
	}
}

func TestRunScriptSupportsJSONStringifySpaceArgument(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `JSON.stringify({ b: 1, a: { d: 4, c: 3 }, arr: [{ y: 2, x: 1 }, 3] }, null, 2)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	want := "{\n  \"b\": 1,\n  \"a\": {\n    \"d\": 4,\n    \"c\": 3\n  },\n  \"arr\": [\n    {\n      \"y\": 2,\n      \"x\": 1\n    },\n    3\n  ]\n}"
	if result.String != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", result.String, want)
	}
}

func TestRunScriptSupportsJSONParsePreservesObjectKeyOrder(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `const parsed = JSON.parse("{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}"); [Object.keys(parsed).join(","), Object.keys(parsed.a).join(","), Object.keys(parsed.arr[0]).join(",")].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "b,a,arr|d,c|y,x"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsJSONParseDuplicateKeysUseLastValueAndFirstOrder(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `const parsed = JSON.parse("{\"b\":1,\"a\":2,\"b\":3}"); [Object.keys(parsed).join(","), String(parsed.b)].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "b,a|3"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsObjectKeysSortAndReverse(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `const parsed = JSON.parse("{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}"); const compareKeys = (a, b) => (a < b ? -1 : a > b ? 1 : 0); const ascending = Object.keys(parsed).sort(compareKeys).join(","); const descending = Object.keys(parsed).sort(compareKeys).reverse().join(","); [ascending, descending].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "a,arr,b|b,arr,a"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsNumberParseInt(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `[
		Number.parseInt("42", 10),
		parseInt("0x10"),
		Number.parseInt("  -0x10", 16)
	].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "42|16|-16"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsNumberIsInteger(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `[
		Number.isInteger(42),
		Number.isInteger(1.5),
		Number.isInteger(Number.NaN),
		Number.isInteger("42")
	].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "true|false|false|false"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsNumberIsNaN(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `[
		Number.isNaN(Number.NaN),
		Number.isNaN(1.5),
		Number.isNaN("42"),
		Number.isNaN()
	].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "true|false|false|false"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsMathConstants(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `[
		Math.E,
		Math.LN10,
		Math.LN2,
		Math.LOG10E,
		Math.LOG2E,
		Math.PI,
		Math.SQRT1_2,
		Math.SQRT2
	].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	want := strings.Join([]string{
		strconv.FormatFloat(math.E, 'f', -1, 64),
		strconv.FormatFloat(math.Ln10, 'f', -1, 64),
		strconv.FormatFloat(math.Ln2, 'f', -1, 64),
		strconv.FormatFloat(math.Log10E, 'f', -1, 64),
		strconv.FormatFloat(math.Log2E, 'f', -1, 64),
		strconv.FormatFloat(math.Pi, 'f', -1, 64),
		strconv.FormatFloat(1/math.Sqrt2, 'f', -1, 64),
		strconv.FormatFloat(math.Sqrt2, 'f', -1, 64),
	}, "|")
	if got := result.String; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsMathRemainingMethods(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `[
		String(Math.acos(1)),
		String(Math.acosh(1)),
		String(Math.asin(0)),
		String(Math.asinh(0)),
		String(Math.atan(0)),
		String(Math.atan2(1, 1)),
		String(Math.atanh(0)),
		String(Math.cbrt(27)),
		String(Math.clz32(1)),
		String(Math.cos(0)),
		String(Math.cosh(0)),
		String(Math.exp(1)),
		String(Math.expm1(1)),
		String(Math.fround(16777217)),
		String(Math.hypot(3, 4)),
		String(Math.imul(-1, 2)),
		String(Math.log(1)),
		String(Math.log10(1000)),
		String(Math.log1p(1)),
		String(Math.log2(8)),
		String(Math.sign(-3)),
		String(1 / Math.sign(-0)),
		String(Math.sin(0)),
		String(Math.sinh(0)),
		String(Math.sqrt(9)),
		String(Math.tan(0)),
		String(Math.tanh(0)),
		String(1 / Math.min(0, -0)),
		String(1 / Math.max(-0, 0))
	].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	want := strings.Join([]string{
		strconv.FormatFloat(math.Acos(1), 'f', -1, 64),
		strconv.FormatFloat(math.Acosh(1), 'f', -1, 64),
		strconv.FormatFloat(math.Asin(0), 'f', -1, 64),
		strconv.FormatFloat(math.Asinh(0), 'f', -1, 64),
		strconv.FormatFloat(math.Atan(0), 'f', -1, 64),
		strconv.FormatFloat(math.Atan2(1, 1), 'f', -1, 64),
		strconv.FormatFloat(math.Atanh(0), 'f', -1, 64),
		strconv.FormatFloat(math.Cbrt(27), 'f', -1, 64),
		strconv.FormatFloat(float64(bits.LeadingZeros32(1)), 'f', -1, 64),
		strconv.FormatFloat(math.Cos(0), 'f', -1, 64),
		strconv.FormatFloat(math.Cosh(0), 'f', -1, 64),
		strconv.FormatFloat(math.Exp(1), 'f', -1, 64),
		strconv.FormatFloat(math.Expm1(1), 'f', -1, 64),
		strconv.FormatFloat(float64(float32(16777217)), 'f', -1, 64),
		strconv.FormatFloat(math.Hypot(3, 4), 'f', -1, 64),
		"-2",
		strconv.FormatFloat(math.Log(1), 'f', -1, 64),
		strconv.FormatFloat(math.Log10(1000), 'f', -1, 64),
		strconv.FormatFloat(math.Log1p(1), 'f', -1, 64),
		strconv.FormatFloat(math.Log2(8), 'f', -1, 64),
		"-1",
		"-Infinity",
		strconv.FormatFloat(math.Sin(0), 'f', -1, 64),
		strconv.FormatFloat(math.Sinh(0), 'f', -1, 64),
		strconv.FormatFloat(math.Sqrt(9), 'f', -1, 64),
		strconv.FormatFloat(math.Tan(0), 'f', -1, 64),
		strconv.FormatFloat(math.Tanh(0), 'f', -1, 64),
		"-Infinity",
		"Infinity",
	}, "|")
	if got := result.String; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsURIComponentHelpers(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `[
		encodeURI(),
		decodeURI(),
		encodeURIComponent(),
		decodeURIComponent(),
		encodeURI(true, "ignored"),
		decodeURI(false, "ignored"),
		encodeURIComponent(42, "ignored"),
		decodeURIComponent(42, "ignored"),
		encodeURIComponent("A&B 春", "ignored"),
		decodeURIComponent("A%26B%20%e6%98%a5", "ignored"),
		encodeURIComponent("C++", "ignored"),
		decodeURIComponent("C%2B%2B", "ignored")
	].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "undefined|undefined|undefined|undefined|true|false|42|42|A%26B%20%E6%98%A5|A&B 春|C%2B%2B|C++"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsURIHelpers(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `[
		encodeURI("https://example.com/A B?x=春&y=1#frag", "ignored"),
		decodeURI("https://example.com/%2f%3f%23%26%20x", "ignored"),
		decodeURI("https://example.com/A%20B?x=%E6%98%A5&y=1#frag", "ignored")
	].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "https://example.com/A%20B?x=%E6%98%A5&y=1#frag|https://example.com/%2f%3f%23%26 x|https://example.com/A B?x=春&y=1#frag"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsURIHelpersSymbolInput(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	for _, tc := range []struct {
		name   string
		source string
	}{
		{name: "encodeURIComponent", source: `encodeURIComponent(Symbol("token"))`},
		{name: "decodeURIComponent", source: `decodeURIComponent(Symbol("token"))`},
		{name: "encodeURI", source: `encodeURI(Symbol("token"))`},
		{name: "decodeURI", source: `decodeURI(Symbol("token"))`},
	} {
		if _, err := session.runScriptOnStore(dom.NewStore(), tc.source); err == nil {
			t.Fatalf("runScriptOnStore(%s) error = nil, want Symbol coercion failure", tc.name)
		} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
			t.Fatalf("runScriptOnStore(%s) error = %#v, want runtime script error", tc.name, err)
		} else if !strings.Contains(scriptErr.Message, "Cannot convert a Symbol value to a string") {
			t.Fatalf("runScriptOnStore(%s) error = %v, want Symbol coercion failure", tc.name, err)
		}
	}
}

func TestRunScriptRejectsDecodeURIComponentMalformedSequence(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `decodeURIComponent("%C3%28")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want URI malformed failure")
	}
	if !strings.Contains(err.Error(), "URI malformed") {
		t.Fatalf("runScriptOnStore() error = %v, want URI malformed message", err)
	}
}

func TestRunScriptRejectsDecodeURIMalformedSequence(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `decodeURI("%C3%28")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want URI malformed failure")
	}
	if !strings.Contains(err.Error(), "URI malformed") {
		t.Fatalf("runScriptOnStore() error = %v, want URI malformed message", err)
	}
}

func TestRunScriptSupportsObjectPrototypeHasOwnPropertyCall(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `const sym = Symbol("token"); const object = { alpha: 1, [sym]: 2 }; const array = [1, 2]; const fn = function Base() {}; [Object.prototype.hasOwnProperty.call(object, "alpha"), Object.prototype.hasOwnProperty.call(object, "beta"), Object.prototype.hasOwnProperty.call(array, "0"), Object.prototype.hasOwnProperty.call(array, "length"), Object.prototype.hasOwnProperty.call(array, "2"), Object.prototype.hasOwnProperty.call(object, sym), Object.prototype.hasOwnProperty.call(object, Symbol("token")), Object.prototype.hasOwnProperty.call(fn, "prototype")].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "true|false|true|true|false|true|false|true"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsObjectHasOwn(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `const sym = Symbol("token"); const object = { alpha: 1, [sym]: 2 }; const array = [1, 2]; const text = "go"; const fn = function Base() {}; [Object.hasOwn(object, "alpha"), Object.hasOwn(object, "beta"), Object.hasOwn(array, "0"), Object.hasOwn(array, "length"), Object.hasOwn(array, "2"), Object.hasOwn(text, "0"), Object.hasOwn(text, "length"), Object.hasOwn(object, sym), Object.hasOwn(object, Symbol("token")), Object.hasOwn(fn, "prototype")].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "true|false|true|true|false|true|true|true|false|true"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsObjectGetOwnPropertyNames(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `const sym = Symbol("token"); const object = { alpha: 1, [sym]: 2 }; const array = [1, 2]; const text = "go"; const fn = function Base() {}; [Object.getOwnPropertyNames(object).join(","), Object.getOwnPropertyNames(array).join(","), Object.getOwnPropertyNames(text).join(","), Object.getOwnPropertyNames(fn).join(",")].join("|")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "alpha|0,1,length|0,1,length|prototype"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsObjectHasOwnOnNullishReceiver(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := session.runScriptOnStore(dom.NewStore(), `Object.hasOwn(null, "alpha")`); err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "Object.hasOwn requires an object receiver") {
		t.Fatalf("runScriptOnStore() error = %v, want nullish receiver error", err)
	}
}

func TestRunScriptRejectsObjectHasOwnWrongArity(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := session.runScriptOnStore(dom.NewStore(), `Object.hasOwn({ alpha: 1 })`); err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "Object.hasOwn expects 2 arguments") {
		t.Fatalf("runScriptOnStore() error = %v, want arity error", err)
	}
}

func TestRunScriptRejectsObjectGetOwnPropertyNamesOnNullishReceiver(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := session.runScriptOnStore(dom.NewStore(), `Object.getOwnPropertyNames(null)`); err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "Cannot convert undefined or null to object") {
		t.Fatalf("runScriptOnStore() error = %v, want nullish conversion error", err)
	}
}

func TestRunScriptRejectsObjectGetOwnPropertyNamesWrongArity(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := session.runScriptOnStore(dom.NewStore(), `Object.getOwnPropertyNames()`); err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "Object.getOwnPropertyNames expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want arity error", err)
	}
}

func TestRunScriptSupportsIntlNumberFormatGrouping(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `new Intl.NumberFormat("en", { maximumFractionDigits: 0 }).format(1198.88)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "1,199"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsIntlNumberFormatCurrencyStyle(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY", maximumFractionDigits: 0 }).format(1200)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "￥1,200"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsIntlNumberFormatCurrencyStyleWithExplicitZeroDigits(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY", minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(1200)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "￥1,200"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsIntlCollatorNumericAndSwedishSorting(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const values = ["item 10", "item 2", "item 1"];
		const collator = new Intl.Collator("en", {
			usage: "sort",
			numeric: true,
			sensitivity: "variant",
		});
		const asc = values.slice().sort(collator.compare).join(",");
		const desc = values.slice().sort((left, right) => collator.compare(right, left)).join(",");
		const zeroPadded = collator.compare("item 02", "item 2");
		const numeric = String(collator.resolvedOptions().numeric);
		const sv = new Intl.Collator("sv", { usage: "sort", sensitivity: "variant" });
		const swedish = ["Öga", "Zebra", "Äpple", "Ål"].slice().sort(sv.compare).join(",");
		asc + "|" + desc + "|" + zeroPadded + "|" + numeric + "|" + swedish
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "item 1,item 2,item 10|item 10,item 2,item 1|0|true|Zebra,Ål,Äpple,Öga"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsUint8ArrayConstruction(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Array.from(new Uint8Array([65, 66, 67])).join(",")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "65,66,67"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsUint8ArrayFromMapFunction(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const binary = "AZ";
		const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
		Array.from(bytes).join(",");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "65,90"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsPromiseResolve(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Promise.resolve(new Uint8Array([1, 2, 3]).buffer)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindPromise {
		t.Fatalf("runScriptOnStore() kind = %q, want promise", result.Kind)
	}
	if result.Promise == nil {
		t.Fatalf("runScriptOnStore() promise payload = nil, want object buffer")
	}
	if result.Promise.Kind != script.ValueKindObject {
		t.Fatalf("runScriptOnStore() promise payload kind = %q, want object", result.Promise.Kind)
	}
	if got, ok := objectProperty(*result.Promise, "byteLength"); !ok || got.Kind != script.ValueKindNumber || got.Number != 3 {
		t.Fatalf("runScriptOnStore() promise payload byteLength = %#v, want 3", got)
	}
}

func TestRunScriptRejectsUint8ArrayFromNonArrayLikeValue(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := session.runScriptOnStore(dom.NewStore(), `new Uint8Array({})`); err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want unsupported array-like input")
	} else if !strings.Contains(err.Error(), "array-like") {
		t.Fatalf("runScriptOnStore() error = %v, want array-like validation failure", err)
	}
}

func TestRunScriptSupportsArrayFromOnHostArrayLikeAttributeReferences(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const parsed = new DOMParser().parseFromString(
			'<svg xmlns="http://www.w3.org/2000/svg"><image href="https://example.com/p.png" width="20" height="20" /></svg>',
			"image/svg+xml"
		);
		const safeRoot = parsed.documentElement.cloneNode(true);
		const image = safeRoot.querySelector("image");
		const attrCount = image ? String(image.attributes.length) : "missing";
		const href = image ? String(image.getAttribute("href")) : "missing";
		let snapshot = [];
		if (image) {
			snapshot = Array.from(image.attributes);
		}
		const snapshotLength = String(snapshot.length);
		const attrs = image
			? snapshot
				.map((attr) => attr.name + "=" + attr.value)
				.sort()
				.join(",")
			: "missing";
		const firstAttr = snapshot[0] ? snapshot[0].name + "=" + snapshot[0].value : "missing";
		if (image) {
			image.removeAttribute("href");
		}
		const hrefAfterRemoval = image ? String(image.getAttribute("href")) : "missing";
		[
			String(!!image),
			attrCount,
			href,
			snapshotLength,
			firstAttr,
			attrs,
			hrefAfterRemoval
		].join("|")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "true|3|https://example.com/p.png|3|height=20|height=20,href=https://example.com/p.png,width=20|null"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptParsesDimensionSuffixesToMillimeters(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		function safeString(value) { return String(value == null ? "" : value); }
		function normalizeDigits(value) {
			return safeString(value)
				.replace(/[\uFF10-\uFF19]/g, (s) => String.fromCharCode(s.charCodeAt(0) - 65248))
				.replace(/[\uFF0E\u3002]/g, ".")
				.replace(/[\uFF0C\u3001]/g, ",")
				.replace(/[\uFF0B]/g, "+")
				.replace(/[\u30FC\uFF0D\u2015]/g, "-")
				.trim();
		}
		function parseFlexibleNumber(value) {
			const normalized = normalizeDigits(value).replace(/[\s_\u00A0]/g, "");
			if (!normalized) return null;
			const sign = normalized.startsWith("-") ? -1 : 1;
			const unsigned = normalized.replace(/^[+-]/, "");
			if (!unsigned) return null;
			const commaCount = (unsigned.match(/,/g) || []).length;
			const dotCount = (unsigned.match(/\./g) || []).length;
			let candidate = unsigned;
			if (commaCount > 0 && dotCount > 0) {
				const lastComma = unsigned.lastIndexOf(",");
				const lastDot = unsigned.lastIndexOf(".");
				const decimalIndex = Math.max(lastComma, lastDot);
				const intPart = unsigned.slice(0, decimalIndex).replace(/[.,]/g, "");
				const fracPart = unsigned.slice(decimalIndex + 1).replace(/[.,]/g, "");
				candidate = fracPart ? intPart + "." + fracPart : intPart;
			} else if (commaCount === 1) {
				const parts = unsigned.split(",");
				candidate = parts[1].length === 3 ? parts.join("") : parts[0] + "." + parts[1];
			} else if (dotCount === 1) {
				const parts = unsigned.split(".");
				candidate = parts[1].length === 3 ? parts.join("") : parts[0] + "." + parts[1];
			}
			const parsed = Number(candidate);
			if (!Number.isFinite(parsed)) return null;
			return parsed * sign;
		}
		function parseDimensionToMm(value, fallbackUnit) {
			const raw = normalizeDigits(value);
			if (!raw) return { mm: null, unit: fallbackUnit, error: null };
			const compact = raw.replace(/[\s_\u00A0]/g, "");
			const match = compact.match(/^([+-]?[0-9.,]+)(mm|cm|m|in|inch|ft|["'])?$/i);
			if (!match) return { mm: null, unit: fallbackUnit, error: "format" };
			const numeric = parseFlexibleNumber(match[1]);
			if (numeric == null) return { mm: null, unit: fallbackUnit, error: "format" };
			const unit = (match[2] || fallbackUnit).toLowerCase();
			const factor =
				unit === "in" || unit === "inch" ? 25.4 :
				unit === "cm" ? 10 :
				unit === "m" ? 1000 :
				unit === "ft" ? 304.8 :
				1;
			const mm = numeric * factor;
			if (!Number.isFinite(mm) || mm <= 0) return { mm: null, unit, error: "positive" };
			return { mm, unit, error: null };
		}
		const first = parseDimensionToMm("47.2in", "mm");
		const second = parseDimensionToMm("35.4in", "mm");
		[String(first.mm), first.unit, String(second.mm), second.unit].join("|");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "1198.88|in|899.16|in"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsStringFromCharCode(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `String.fromCharCode(0x41, 0x42, 0x43, 0x20, 0x30)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "ABC 0"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsBrowserObjectAssignSymbolsAndToFixed(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const symA = Symbol("token");
		const symB = Symbol("token");
		const assigned = Object.assign({ plain: "a" }, "go", null, undefined, { extra: "b" }, { [symA]: "symbol" });
		const symbols = Object.getOwnPropertySymbols(assigned);
		[
			symA === symB,
			assigned.plain,
			assigned[0],
			assigned[1],
			assigned.extra,
			symbols.length,
			symbols[0].toString(),
			assigned[symbols[0]],
			(1.2).toFixed(2)
		].join("|");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "false|a|g|o|b|1|Symbol(token)|symbol|1.20"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsNullishObjectHelpers(t *testing.T) {
	t.Run("assign target", func(t *testing.T) {
		session := NewSession(DefaultSessionConfig())

		_, err := session.runScriptOnStore(dom.NewStore(), `Object.assign(null, { a: 1 })`)
		if err == nil {
			t.Fatalf("runScriptOnStore() error = nil, want Object.assign target failure")
		}
		if !strings.Contains(err.Error(), "Cannot convert undefined or null to object") {
			t.Fatalf("runScriptOnStore() error = %v, want nullish conversion failure", err)
		}
	})

	t.Run("getOwnPropertySymbols target", func(t *testing.T) {
		session := NewSession(DefaultSessionConfig())

		_, err := session.runScriptOnStore(dom.NewStore(), `Object.getOwnPropertySymbols(null)`)
		if err == nil {
			t.Fatalf("runScriptOnStore() error = nil, want Object.getOwnPropertySymbols failure")
		}
		if !strings.Contains(err.Error(), "Cannot convert undefined or null to object") {
			t.Fatalf("runScriptOnStore() error = %v, want nullish conversion failure", err)
		}
	})
}

func TestRunScriptUsesSeededMathRandom(t *testing.T) {
	sessionA := NewSession(SessionConfig{
		RandomSeed:    7,
		HasRandomSeed: true,
	})
	sessionB := NewSession(SessionConfig{
		RandomSeed:    7,
		HasRandomSeed: true,
	})
	source := `Math.random() + "|" + Math.random()`

	resultA, err := sessionA.runScriptOnStore(dom.NewStore(), source)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	resultB, err := sessionB.runScriptOnStore(dom.NewStore(), source)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}

	rng := rand.New(rand.NewSource(7))
	want := strconv.FormatFloat(rng.Float64(), 'f', -1, 64) + "|" + strconv.FormatFloat(rng.Float64(), 'f', -1, 64)

	if resultA.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", resultA.Kind)
	}
	if resultB.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", resultB.Kind)
	}
	if resultA.String != want {
		t.Fatalf("runScriptOnStore() seed 7 value = %q, want %q", resultA.String, want)
	}
	if resultB.String != want {
		t.Fatalf("runScriptOnStore() seed 7 repeat value = %q, want %q", resultB.String, want)
	}
}

func TestRunScriptSupportsCSSEscapeOnGlobalCSSObject(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `CSS.escape() + "|" + CSS.escape("0", "ignored") + "|" + CSS.escape("alpha-beta", "ignored")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, `undefined|\30 |alpha-beta`; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsCSSEscapeSymbolInput(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `CSS.escape(Symbol("token"))`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Symbol coercion failure")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
		t.Fatalf("runScriptOnStore() error = %#v, want runtime script error", err)
	} else if !strings.Contains(scriptErr.Message, "Cannot convert a Symbol value to a string") {
		t.Fatalf("runScriptOnStore() error = %v, want Symbol coercion failure message", err)
	}
}

func TestRunScriptRejectsUnsupportedCSSReference(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `CSS.supports("color", "red")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want unsupported CSS reference error")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported || !strings.Contains(scriptErr.Message, "CSS.supports") {
		t.Fatalf("runScriptOnStore() error = %#v, want unsupported CSS reference error", err)
	}
}

func TestRunScriptRejectsInvalidJSONParse(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `JSON.parse("{")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want parse error")
	}
}

func TestRunScriptRejectsInvalidObjectEntriesArity(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Object.entries()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want object entries error")
	}
	if !strings.Contains(err.Error(), "Object.entries expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want object entries message", err)
	}
}

func TestRunScriptSupportsObjectFromEntries(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const table = Object.fromEntries([
			["full", "アイ"],
			["half", "ｱｲ"]
		]);
		const aliasTable = Object.fromEntries(new Map([
			["zenkaku", table.full],
			["hankaku", table.half]
		]));
		aliasTable.zenkaku + "|" + aliasTable.hankaku + "|" + Object.keys(aliasTable).join(",")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "アイ|ｱｲ|zenkaku,hankaku"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsSetConstructorAndMethods(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const seen = new Set(["alpha", "alpha", "beta"]);
		seen.add("gamma");
		seen.delete("alpha");
		[String(seen.size), String(seen.has("alpha")), String(seen.has("gamma"))].join("|")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "2|false|true"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsSetConstructorIterableInputs(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const empty = new Set();
		const copied = new Set(new Set(["alpha", "alpha", "beta"]));
		const params = new URLSearchParams("u=metric&h=3.2&s=4.0");
		const fromParams = new Set(params.keys());
		[String(empty.size), Array.from(copied).join(","), Array.from(fromParams).join(",")].join("|")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "0|alpha,beta|u,h,s"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsArrayFromSet(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		Array.from(new Set(["alpha", "alpha", "beta"])).join(",")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "alpha,beta"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsIntlDateTimeFormatTimeZoneAndFormatToParts(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		function zonedText(instantMs, zone) {
			const formatter = new Intl.DateTimeFormat("en-US-u-nu-latn", {
				timeZone: zone,
				year: "numeric",
				month: "2-digit",
				day: "2-digit",
				hour: "2-digit",
				minute: "2-digit",
				second: "2-digit",
				hour12: false,
			});
			const parts = formatter.formatToParts(new Date(instantMs));
			const get = (type) => parts.find((part) => part.type === type)?.value || "?";
			return get("year") + "-" + get("month") + "-" + get("day") + " " + get("hour") + ":" + get("minute") + ":" + get("second");
		}
		const arrivalInstant = Date.UTC(2026, 0, 21, 8, 45, 0, 0);
		zonedText(arrivalInstant, "America/Chicago") + "|" + zonedText(arrivalInstant, "America/New_York")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "2026-01-21 02:45:00|2026-01-21 03:45:00"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsInvalidIntlDateTimeFormatTimeZone(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `new Intl.DateTimeFormat("en-US", { timeZone: "Mars/Phobos" }).formatToParts(new Date(0))`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want invalid timeZone failure")
	}
	if !strings.Contains(err.Error(), "timeZone") {
		t.Fatalf("runScriptOnStore() error = %v, want timeZone failure message", err)
	}
}

func TestRunScriptRejectsIntlCollatorNumericTypeMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `new Intl.Collator("en-US", { numeric: "true" })`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want collator numeric type failure")
	}
	if !strings.Contains(err.Error(), "Intl.Collator numeric must be a boolean") {
		t.Fatalf("runScriptOnStore() error = %v, want collator numeric failure message", err)
	}
}

func TestRunScriptRejectsSetCallWithoutNew(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Set()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Set call failure")
	}
	if !strings.Contains(err.Error(), "Set constructor must be called with `new`") {
		t.Fatalf("runScriptOnStore() error = %v, want Set constructor call message", err)
	}
}

func TestRunScriptSupportsDateUTC(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Date.UTC(2026, 0, 21, 8, 45, 0, 0)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindNumber {
		t.Fatalf("runScriptOnStore() kind = %q, want number", result.Kind)
	}
	want := time.Date(2026, time.January, 21, 8, 45, 0, 0, time.UTC).UnixMilli()
	if got := int64(result.Number); got != want {
		t.Fatalf("runScriptOnStore() value = %d, want %d", got, want)
	}
}

func TestRunScriptRejectsSetConstructorNullIterable(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `new Set(null)`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want null iterable failure")
	}
	if !strings.Contains(err.Error(), "Set constructor cannot iterate over null") {
		t.Fatalf("runScriptOnStore() error = %v, want null iterable failure message", err)
	}
}

func TestRunScriptRejectsInvalidObjectFromEntriesPair(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Object.fromEntries([["full"]])`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want object.fromEntries pair failure")
	}
	if !strings.Contains(err.Error(), "Object.fromEntries pair 0 must be a two-item array") {
		t.Fatalf("runScriptOnStore() error = %v, want object.fromEntries pair message", err)
	}
}

func TestRunScriptRejectsInvalidObjectFromEntriesArity(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Object.fromEntries()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want object.fromEntries arity failure")
	}
	if !strings.Contains(err.Error(), "Object.fromEntries expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want object.fromEntries arity message", err)
	}
}

func TestRunScriptRejectsObjectAssignGetterOnlyTarget(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `
		Object.assign({
			get a() { return 1; }
		}, {
			a: 2
		})
	`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want getter-only target error")
	}
	if !strings.Contains(err.Error(), "getter-only property") {
		t.Fatalf("runScriptOnStore() error = %v, want getter-only target message", err)
	}
}

func TestRunScriptRejectsMathRoundArityMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.round()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.round arity failure")
	}
	if !strings.Contains(err.Error(), "Math.round expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.round arity message", err)
	}
}

func TestRunScriptSupportsMathFloor(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Math.floor(1.9) + "|" + Math.floor(-1.1)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "1|-2"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsMathCeil(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `String(Math.ceil(1.1)) + "|" + String(Math.ceil(-1.1)) + "|" + String(1 / Math.ceil(-0.1)) + "|" + String(Math.ceil(2)) + "|" + String(Math.ceil(Number.NaN)) + "|" + String(Math.trunc(1.9)) + "|" + String(Math.trunc(-1.9))`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "2|-1|-Infinity|2|NaN|1|-1"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsMathPow(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `String(Math.pow(2, 3)) + "|" + String(Math.pow(9, 0.5)) + "|" + String(Math.pow(2, -3)) + "|" + String(Math.pow(-2, 0.5))`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "8|3|0.125|NaN"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsMathFloorArityMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.floor()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.floor arity failure")
	}
	if !strings.Contains(err.Error(), "Math.floor expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.floor arity message", err)
	}
}

func TestRunScriptRejectsMathCeilArityMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.ceil()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.ceil arity failure")
	}
	if !strings.Contains(err.Error(), "Math.ceil expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.ceil arity message", err)
	}
}

func TestRunScriptRejectsMathPowArityMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.pow(2)`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.pow arity failure")
	}
	if !strings.Contains(err.Error(), "Math.pow expects 2 arguments") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.pow arity message", err)
	}
}

func TestRunScriptRejectsMathRemainingMethodsOnObjectInput(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.log({})`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.log type failure")
	}
	if !strings.Contains(err.Error(), "argument must be a primitive number") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.log type message", err)
	}
}

func TestRunScriptRejectsMathTruncArityMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.trunc()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.trunc arity failure")
	}
	if !strings.Contains(err.Error(), "Math.trunc expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.trunc arity message", err)
	}
}
