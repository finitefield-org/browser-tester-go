package script

import (
	"strconv"
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

func TestDispatchSupportsStringSearchMethodsWithUnicode(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["あいう".indexOf("い"), "あいう".indexOf("い", 2), "あいう".indexOf("", 5), "あいう".lastIndexOf("い"), "あいう".lastIndexOf("い", 2), "あいう".startsWith("あ"), "あいう".startsWith("い", 1), "あいう".endsWith("う"), "あいう".endsWith("い", 2), "あいう".includes("い", 1), "あいう".includes("い", 2)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.search methods unicode) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.search methods unicode) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|-1|3|1|1|true|true|true|true|true|false" {
		t.Fatalf("Dispatch(String.search methods unicode) value = %q, want %q", result.Value.String, "1|-1|3|1|1|true|true|true|true|true|false")
	}
}

func TestDispatchRejectsInvalidStringIncludesArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".includes()`})
	if err == nil {
		t.Fatalf("Dispatch(String.includes()) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "String.includes expects 1 argument") {
		t.Fatalf("Dispatch(String.includes()) error = %q, want arity error", got)
	}
}

func TestDispatchSupportsStringSearch(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["abc".search(), "abc".search("b"), "abc".search(/b/), "あいう".search("い"), "あいう".search(/い/)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.search) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.search) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "-1|1|1|1|1" {
		t.Fatalf("Dispatch(String.search) value = %q, want %q", result.Value.String, "-1|1|1|1|1")
	}
}

func TestDispatchSupportsStringCharAt(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["あいう".charAt(-1), "あいう".charAt(1), "あいう".charAt(3)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.charAt) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.charAt) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "|い|" {
		t.Fatalf("Dispatch(String.charAt) value = %q, want %q", result.Value.String, "|い|")
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

func TestDispatchSupportsStringTrimStartEnd(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["  Go  ".trimStart(), "  Go  ".trimEnd()].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.trimStart/trimEnd) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.trimStart/trimEnd) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "Go  |  Go" {
		t.Fatalf("Dispatch(String.trimStart/trimEnd) value = %q, want %q", result.Value.String, "Go  |  Go")
	}
}

func TestDispatchSupportsStringCaseConversion(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["Go".toLowerCase(), "go".toUpperCase()].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.toLowerCase/toUpperCase) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.toLowerCase/toUpperCase) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "go|GO" {
		t.Fatalf("Dispatch(String.toLowerCase/toUpperCase) value = %q, want %q", result.Value.String, "go|GO")
	}
}

func TestDispatchSupportsStringConcat(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["go".concat(), "go".concat("!", 1, null, undefined), "あ".concat("い", "う")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.concat) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.concat) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "go|go!1nullundefined|あいう" {
		t.Fatalf("Dispatch(String.concat) value = %q, want %q", result.Value.String, "go|go!1nullundefined|あいう")
	}
}

func TestDispatchSupportsStringLocaleCompare(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["b".localeCompare("a"), "a".localeCompare("a"), "a".localeCompare("b"), "ä".localeCompare("z", "sv"), "item 2".localeCompare("item 10", undefined, { numeric: true })].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.localeCompare) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.localeCompare) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|0|-1|1|-1" {
		t.Fatalf("Dispatch(String.localeCompare) value = %q, want %q", result.Value.String, "1|0|-1|1|-1")
	}
}

func TestDispatchRejectsInvalidStringLocaleCompareOptions(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"a".localeCompare("b", "en", "true")`})
	if err == nil {
		t.Fatalf("Dispatch(String.localeCompare options) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.localeCompare options) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.localeCompare options) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsStringReplaceCallbackReplacer(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["A1B2".replace(/([A-Z])([0-9])/g, (match, letter, digit, offset, input) => letter.toLowerCase() + digit + ":" + offset), "g".replace("g", (match, offset, input) => match + ":" + offset + ":" + input), "あ1い2".replace(/([0-9])/g, (match, digit, offset, input) => digit + ":" + offset)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.replace callback) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.replace callback) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "a1:0b2:2|g:0:g|あ1:1い2:3" {
		t.Fatalf("Dispatch(String.replace callback) value = %q, want %q", result.Value.String, "a1:0b2:2|g:0:g|あ1:1い2:3")
	}
}

func TestDispatchSupportsStringReplaceRegexpReplacementCaptures(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `"10.0".replace(/\.0+$|(\.\d*?)0+$/, "$1")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.replace regexp replacement captures) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.replace regexp replacement captures) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "10" {
		t.Fatalf("Dispatch(String.replace regexp replacement captures) value = %q, want %q", result.Value.String, "10")
	}
}

func TestDispatchSupportsStringReplaceAll(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["gooo".replaceAll("oo", "b"), "gooo".replaceAll(/o/g, "a"), "gooo".replaceAll("o", (match, offset, input) => match + ":" + offset), "A1B2".replaceAll(/([A-Z])([0-9])/g, (match, letter, digit, offset, input) => letter.toLowerCase() + digit + ":" + offset)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.replaceAll) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.replaceAll) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "gbo|gaaa|go:1o:2o:3|a1:0b2:2" {
		t.Fatalf("Dispatch(String.replaceAll) value = %q, want %q", result.Value.String, "gbo|gaaa|go:1o:2o:3|a1:0b2:2")
	}
}

func TestDispatchRejectsInvalidStringReplaceAllRegexFlags(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"gooo".replaceAll(/o/, "a")`})
	if err == nil {
		t.Fatalf("Dispatch(String.replaceAll(/o/)) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.replaceAll(/o/)) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.replaceAll(/o/)) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsStringReplaceAllCallbackReplacer(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `"gooo".replaceAll("oo", (match, offset, input) => match.toUpperCase() + ":" + offset + ":" + input).concat("|", "A1B2".replaceAll(/([A-Z])([0-9])/g, (match, letter, digit, offset, input) => letter.toLowerCase() + digit + ":" + offset))`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.replaceAll callback) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.replaceAll callback) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "gOO:1:goooo|a1:0b2:2" {
		t.Fatalf("Dispatch(String.replaceAll callback) value = %q, want %q", result.Value.String, "gOO:1:goooo|a1:0b2:2")
	}
}

func TestDispatchSupportsStringMatchAll(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["A1B2".matchAll(/([A-Z])([0-9])/g).map(match => match.join(":")).join("|"), "gooo".matchAll("oo").map(match => match[0]).join(",")].join("~")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.matchAll) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.matchAll) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "A1:A:1|B2:B:2~oo" {
		t.Fatalf("Dispatch(String.matchAll) value = %q, want %q", result.Value.String, "A1:A:1|B2:B:2~oo")
	}
}

func TestDispatchRejectsInvalidStringMatchAllRegexFlags(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"gooo".matchAll(/o/)`})
	if err == nil {
		t.Fatalf("Dispatch(String.matchAll(/o/)) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.matchAll(/o/)) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.matchAll(/o/)) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsArrayAndStringAt(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[ [1, 2, 3].at(0), [1, 2, 3].at(-1), [1, 2, 3].at(3) === undefined, "アイウ".at(1) ].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array/String.at) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array/String.at) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|3|true|イ" {
		t.Fatalf("Dispatch(Array/String.at) value = %q, want %q", result.Value.String, "1|3|true|イ")
	}
}

func TestDispatchSupportsStringCodePointAt(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["あいう".codePointAt(-1) === undefined, "あいう".codePointAt(1), "あいう".codePointAt(3) === undefined].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.codePointAt) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.codePointAt) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "true|12356|true" {
		t.Fatalf("Dispatch(String.codePointAt) value = %q, want %q", result.Value.String, "true|12356|true")
	}
}

func TestDispatchRejectsInvalidStringCharAtArgument(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".charAt({})`})
	if err == nil {
		t.Fatalf("Dispatch(\"go\".charAt({})) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(\"go\".charAt({})) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(\"go\".charAt({})) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsInvalidArrayAtArgument(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[].at({})`})
	if err == nil {
		t.Fatalf("Dispatch([].at({})) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch([].at({})) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch([].at({})) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsInvalidStringAtArgument(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".at({})`})
	if err == nil {
		t.Fatalf("Dispatch(\"go\".at({})) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(\"go\".at({})) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(\"go\".at({})) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsInvalidStringCodePointAtArgument(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".codePointAt({})`})
	if err == nil {
		t.Fatalf("Dispatch(\"go\".codePointAt({})) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(\"go\".codePointAt({})) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(\"go\".codePointAt({})) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsArrayFindLastAndFindLastIndex(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[[1, 2, 3, 2].findLast(v => v === 2), [1, 2, 3, 2].findLastIndex(v => v === 2), [1, 2, 3].findLastIndex(v => v === 4)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.findLast/findLastIndex) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.findLast/findLastIndex) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "2|3|-1" {
		t.Fatalf("Dispatch(Array.findLast/findLastIndex) value = %q, want %q", result.Value.String, "2|3|-1")
	}
}

func TestDispatchRejectsInvalidArrayFindLastArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[].findLast()`})
	if err == nil {
		t.Fatalf("Dispatch([].findLast()) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch([].findLast()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch([].findLast()) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsInvalidArrayFindLastIndexArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[].findLastIndex()`})
	if err == nil {
		t.Fatalf("Dispatch([].findLastIndex()) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch([].findLastIndex()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch([].findLastIndex()) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
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

func TestDispatchRejectsInvalidStringPadStartArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".padStart()`})
	if err == nil {
		t.Fatalf("Dispatch(String.padStart()) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.padStart()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.padStart()) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchSupportsStringPadEnd(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["7".padEnd(3, "0"), "7".padEnd(2, "0"), "7".padEnd(5, "abc")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.padEnd) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.padEnd) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "700|70|7abca" {
		t.Fatalf("Dispatch(String.padEnd) value = %q, want %q", result.Value.String, "700|70|7abca")
	}
}

func TestDispatchSupportsStringPadUnicodeFill(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["go".padStart(3, "あ"), "go".padEnd(3, "あ")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.padStart/padEnd unicode fill) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.padStart/padEnd unicode fill) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "あgo|goあ" {
		t.Fatalf("Dispatch(String.padStart/padEnd unicode fill) value = %q, want %q", result.Value.String, "あgo|goあ")
	}
}

func TestDispatchSupportsStringRepeat(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["go".repeat(3), "go".repeat(2.9), "go".repeat("2"), "あ".repeat(3)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.repeat) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.repeat) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "gogogo|gogo|gogo|あああ" {
		t.Fatalf("Dispatch(String.repeat) value = %q, want %q", result.Value.String, "gogogo|gogo|gogo|あああ")
	}
}

func TestDispatchRejectsInvalidStringPadEndArity(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".padEnd()`})
	if err == nil {
		t.Fatalf("Dispatch(String.padEnd()) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.padEnd()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.padEnd()) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
	}
}

func TestDispatchRejectsInvalidStringRepeatCount(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `"go".repeat(-1)`})
	if err == nil {
		t.Fatalf("Dispatch(String.repeat(-1)) error = nil, want error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(String.repeat(-1)) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindRuntime {
		t.Fatalf("Dispatch(String.repeat(-1)) error kind = %q, want %q", scriptErr.Kind, ErrorKindRuntime)
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

func TestDispatchSupportsStringSubstringWithUnicodeRunes(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `["アイウ".substring(0, 2), "アイウ".substring(2, 0), "アイウ".substring(-1, 2), "アイウ".substring(1)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(String.substring) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(String.substring) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "アイ|アイ|アイ|イウ" {
		t.Fatalf("Dispatch(String.substring) value = %q, want %q", result.Value.String, "アイ|アイ|アイ|イウ")
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

func TestDispatchSupportsArrayPop(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			let a = [1, 2, 3];
			let last = a.pop();
			[last, a.join(",")].join("|")
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.pop) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.pop) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "3|1,2" {
		t.Fatalf("Dispatch(Array.pop) value = %q, want %q", result.Value.String, "3|1,2")
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

func TestDispatchSupportsArrayCopyWithin(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const values = [1, 2, 3, 4, 5]; values.copyWithin(0, 3); const mixed = ["a", "b", "c", "d", "e"]; mixed.copyWithin(-2, 1, 3); [values.join(","), mixed.join(",")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.copyWithin) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.copyWithin) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "4,5,3,4,5|a,b,c,b,c" {
		t.Fatalf("Dispatch(Array.copyWithin) value = %q, want %q", result.Value.String, "4,5,3,4,5|a,b,c,b,c")
	}
}

func TestDispatchSupportsArrayIncludes(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[1, 2, NaN].includes(NaN) + "|" + [1, 2, 3].includes(2, -2) + "|" + [1, 2, 3].includes(1, 1)`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.includes) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.includes) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "true|true|false" {
		t.Fatalf("Dispatch(Array.includes) value = %q, want %q", result.Value.String, "true|true|false")
	}
}

func TestDispatchSupportsArrayReduceAndReduceRight(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[[1, 2, 3, 4].reduce((acc, value) => acc + value), [1, 2, 3, 4].reduce((acc, value) => acc + value, 10), ["a", "b", "c"].reduceRight((acc, value) => acc + value)].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.reduce/reduceRight) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.reduce/reduceRight) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "10|20|cba" {
		t.Fatalf("Dispatch(Array.reduce/reduceRight) value = %q, want %q", result.Value.String, "10|20|cba")
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

func TestDispatchRejectsInvalidArrayReduceOnEmptyArray(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[].reduce((acc, value) => acc + value)`})
	if err == nil {
		t.Fatalf("Dispatch([].reduce()) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Array.reduce requires at least one value") {
		t.Fatalf("Dispatch([].reduce()) error = %q, want empty-array error", got)
	}
}

func TestDispatchRejectsInvalidArrayReduceRightOnEmptyArray(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `[].reduceRight((acc, value) => acc + value)`})
	if err == nil {
		t.Fatalf("Dispatch([].reduceRight()) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Array.reduceRight requires at least one value") {
		t.Fatalf("Dispatch([].reduceRight()) error = %q, want empty-array error", got)
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

func TestDispatchSupportsArraySortComparatorWithObjectSubtraction(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const rows = [{ start: { index: 2 }, end: { index: 4 } }, { start: { index: 1 }, end: { index: 3 } }]; rows.sort((left, right) => left.start - right.start || left.end - right.end); rows.length`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.sort object subtraction) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber {
		t.Fatalf("Dispatch(Array.sort object subtraction) kind = %q, want %q", result.Value.Kind, ValueKindNumber)
	}
	if result.Value.Number != 2 {
		t.Fatalf("Dispatch(Array.sort object subtraction) value = %v, want 2", result.Value.Number)
	}
}

func TestDispatchSupportsArrayReverse(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const values = [1, 2, 3]; const reversed = values.reverse(); [values.join(","), reversed.join(",")].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Array.reverse) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Array.reverse) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "3,2,1|3,2,1" {
		t.Fatalf("Dispatch(Array.reverse) value = %q, want %q", result.Value.String, "3,2,1|3,2,1")
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

func TestDispatchSupportsNumberToLocaleString(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `[(600).toLocaleString("en-US", { minimumFractionDigits: 1, maximumFractionDigits: 1 }), (1200).toLocaleString("ja-JP", { style: "currency", currency: "JPY", minimumFractionDigits: 0, maximumFractionDigits: 0 })].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Number.toLocaleString) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Number.toLocaleString) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "600.0|￥1,200" {
		t.Fatalf("Dispatch(Number.toLocaleString) value = %q, want %q", result.Value.String, "600.0|￥1,200")
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

func TestDispatchSupportsBrowserDateGetFullYear(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2026, time.March, 29, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `date.getFullYear()`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.getFullYear) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber {
		t.Fatalf("Dispatch(Date.getFullYear) kind = %q, want %q", result.Value.Kind, ValueKindNumber)
	}
	if result.Value.Number != 2026 {
		t.Fatalf("Dispatch(Date.getFullYear) value = %v, want %v", result.Value.Number, 2026)
	}
}

func TestDispatchSupportsBrowserDateSetTime(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `const next = date.setTime(1700000004567); [next, date.getTime(), date.toISOString()].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setTime) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setTime) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1700000004567|1700000004567|2023-11-14T22:13:24.567Z" {
		t.Fatalf("Dispatch(Date.setTime) value = %q, want %q", result.Value.String, "1700000004567|1700000004567|2023-11-14T22:13:24.567Z")
	}
}

func TestDispatchSupportsBrowserDateSetMilliseconds(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `const first = [date.setMilliseconds(567), date.getTime(), date.toISOString()].join("|"); const second = [date.setUTCMilliseconds(999), date.getTime(), date.toISOString()].join("|"); [first, second].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setMilliseconds) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setMilliseconds) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	wantFirst := time.UnixMilli(1700000000567).UTC().Format("2006-01-02T15:04:05.000Z")
	wantSecond := time.UnixMilli(1700000000999).UTC().Format("2006-01-02T15:04:05.000Z")
	want := strings.Join([]string{
		"1700000000567",
		"1700000000567",
		wantFirst,
		"1700000000999",
		"1700000000999",
		wantSecond,
	}, "|")
	if result.Value.String != want {
		t.Fatalf("Dispatch(Date.setMilliseconds) value = %q, want %q", result.Value.String, want)
	}
}

func TestDispatchSupportsBrowserDateSetDate(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `const first = [date.setDate(5), date.getTime(), date.toISOString()].join("|"); const second = [date.setUTCDate(31), date.getTime(), date.toISOString()].join("|"); [first, second].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setDate) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setDate) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	wantFirst := time.Date(2023, time.November, 5, 22, 13, 20, 123*int(time.Millisecond), time.UTC).UnixMilli()
	wantSecond := time.Date(2023, time.November, 31, 22, 13, 20, 123*int(time.Millisecond), time.UTC).UnixMilli()
	want := strings.Join([]string{
		strconv.FormatInt(wantFirst, 10),
		strconv.FormatInt(wantFirst, 10),
		time.UnixMilli(wantFirst).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(wantSecond, 10),
		strconv.FormatInt(wantSecond, 10),
		time.UnixMilli(wantSecond).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")
	if result.Value.String != want {
		t.Fatalf("Dispatch(Date.setDate) value = %q, want %q", result.Value.String, want)
	}
}

func TestDispatchSupportsBrowserDateSetMonth(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `const first = [date.setMonth(0), date.getTime(), date.toISOString()].join("|"); const second = [date.setUTCMonth(11, 31), date.getTime(), date.toISOString()].join("|"); [first, second].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setMonth) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setMonth) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	wantFirst := time.Date(2023, time.January, 14, 22, 13, 20, 123*int(time.Millisecond), time.UTC).UnixMilli()
	wantSecond := time.Date(2023, time.December, 31, 22, 13, 20, 123*int(time.Millisecond), time.UTC).UnixMilli()
	want := strings.Join([]string{
		strconv.FormatInt(wantFirst, 10),
		strconv.FormatInt(wantFirst, 10),
		time.UnixMilli(wantFirst).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(wantSecond, 10),
		strconv.FormatInt(wantSecond, 10),
		time.UnixMilli(wantSecond).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")
	if result.Value.String != want {
		t.Fatalf("Dispatch(Date.setMonth) value = %q, want %q", result.Value.String, want)
	}
}

func TestDispatchSupportsBrowserDateSetSeconds(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `const first = [date.setSeconds(5, 7), date.getTime(), date.toISOString()].join("|"); const second = [date.setUTCSeconds(59, 8), date.getTime(), date.toISOString()].join("|"); [first, second].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setSeconds) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setSeconds) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	want := strings.Join([]string{
		"1699999985007",
		"1699999985007",
		time.UnixMilli(1699999985007).UTC().Format("2006-01-02T15:04:05.000Z"),
		"1700000039008",
		"1700000039008",
		time.UnixMilli(1700000039008).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")
	if result.Value.String != want {
		t.Fatalf("Dispatch(Date.setSeconds) value = %q, want %q", result.Value.String, want)
	}
}

func TestDispatchRejectsBrowserDateSetMonthNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `date.setMonth(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setMonth(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setMonth requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setMonth(NaN)) error = %q, want finite timestamp error", got)
	}
}

func TestDispatchRejectsBrowserDateSetDateNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `date.setDate(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setDate(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setDate requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setDate(NaN)) error = %q, want finite timestamp error", got)
	}
}

func TestDispatchSupportsBrowserDateSetFullYear(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `const first = [date.setFullYear(2024), date.getTime(), date.toISOString()].join("|"); const second = [date.setUTCFullYear(2025, 0, 15), date.getTime(), date.toISOString()].join("|"); [first, second].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setFullYear) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setFullYear) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	wantFirst := time.Date(2024, time.November, 14, 22, 13, 20, 123*int(time.Millisecond), time.UTC).UnixMilli()
	wantSecond := time.Date(2025, time.January, 15, 22, 13, 20, 123*int(time.Millisecond), time.UTC).UnixMilli()
	want := strings.Join([]string{
		strconv.FormatInt(wantFirst, 10),
		strconv.FormatInt(wantFirst, 10),
		time.UnixMilli(wantFirst).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(wantSecond, 10),
		strconv.FormatInt(wantSecond, 10),
		time.UnixMilli(wantSecond).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")
	if result.Value.String != want {
		t.Fatalf("Dispatch(Date.setFullYear) value = %q, want %q", result.Value.String, want)
	}
}

func TestDispatchRejectsBrowserDateSetFullYearNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `date.setFullYear(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setFullYear(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setFullYear requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setFullYear(NaN)) error = %q, want finite timestamp error", got)
	}
}

func TestDispatchSupportsBrowserDateSetMinutes(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `const first = [date.setMinutes(4, 5, 6), date.getTime(), date.toISOString()].join("|"); const second = [date.setUTCMinutes(59, 58, 57), date.getTime(), date.toISOString()].join("|"); [first, second].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setMinutes) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setMinutes) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	want := strings.Join([]string{
		strconv.FormatInt(time.Date(2023, time.November, 14, 22, 4, 5, 6*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		strconv.FormatInt(time.Date(2023, time.November, 14, 22, 4, 5, 6*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		time.UnixMilli(time.Date(2023, time.November, 14, 22, 4, 5, 6*int(time.Millisecond), time.UTC).UnixMilli()).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(time.Date(2023, time.November, 14, 22, 59, 58, 57*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		strconv.FormatInt(time.Date(2023, time.November, 14, 22, 59, 58, 57*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		time.UnixMilli(time.Date(2023, time.November, 14, 22, 59, 58, 57*int(time.Millisecond), time.UTC).UnixMilli()).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")
	if result.Value.String != want {
		t.Fatalf("Dispatch(Date.setMinutes) value = %q, want %q", result.Value.String, want)
	}
}

func TestDispatchSupportsBrowserDateSetHours(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `const first = [date.setHours(4, 5, 6, 7), date.getTime(), date.toISOString()].join("|"); const second = [date.setUTCHours(23, 58, 57, 56), date.getTime(), date.toISOString()].join("|"); [first, second].join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(Date.setHours) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(Date.setHours) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	want := strings.Join([]string{
		strconv.FormatInt(time.Date(2023, time.November, 14, 4, 5, 6, 7*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		strconv.FormatInt(time.Date(2023, time.November, 14, 4, 5, 6, 7*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		time.UnixMilli(time.Date(2023, time.November, 14, 4, 5, 6, 7*int(time.Millisecond), time.UTC).UnixMilli()).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(time.Date(2023, time.November, 14, 23, 58, 57, 56*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		strconv.FormatInt(time.Date(2023, time.November, 14, 23, 58, 57, 56*int(time.Millisecond), time.UTC).UnixMilli(), 10),
		time.UnixMilli(time.Date(2023, time.November, 14, 23, 58, 57, 56*int(time.Millisecond), time.UTC).UnixMilli()).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")
	if result.Value.String != want {
		t.Fatalf("Dispatch(Date.setHours) value = %q, want %q", result.Value.String, want)
	}
}

func TestDispatchRejectsBrowserDateSetTimeNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `date.setTime(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setTime(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setTime requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setTime(NaN)) error = %q, want finite timestamp error", got)
	}
}

func TestDispatchRejectsBrowserDateSetMillisecondsNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC).UnixMilli()),
		},
		Source: `date.setMilliseconds(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setMilliseconds(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setMilliseconds requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setMilliseconds(NaN)) error = %q, want finite timestamp error", got)
	}
}

func TestDispatchRejectsBrowserDateSetSecondsNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `date.setSeconds(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setSeconds(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setSeconds requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setSeconds(NaN)) error = %q, want finite timestamp error", got)
	}
}

func TestDispatchRejectsBrowserDateSetMinutesNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `date.setMinutes(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setMinutes(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setMinutes requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setMinutes(NaN)) error = %q, want finite timestamp error", got)
	}
}

func TestDispatchRejectsBrowserDateSetHoursNonFinite(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{
		Bindings: map[string]Value{
			"date": BrowserDateValue(1700000000123),
		},
		Source: `date.setHours(0 / 0)`,
	})
	if err == nil {
		t.Fatalf("Dispatch(Date.setHours(NaN)) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Date.prototype.setHours requires a finite timestamp") {
		t.Fatalf("Dispatch(Date.setHours(NaN)) error = %q, want finite timestamp error", got)
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

func TestDispatchRejectsInvalidNumberToLocaleStringOptions(t *testing.T) {
	runtime := NewRuntime(nil)

	_, err := runtime.Dispatch(DispatchRequest{Source: `(600).toLocaleString("en-US", "bad")`})
	if err == nil {
		t.Fatalf("Dispatch(Number.toLocaleString invalid options) error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "options argument must be an object") {
		t.Fatalf("Dispatch(Number.toLocaleString invalid options) error = %q, want options type error", got)
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
