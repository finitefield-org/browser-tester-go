package jsregex

import (
	"strings"
	"testing"
)

func TestCompileLiteralExecAndReplace(t *testing.T) {
	re, err := CompileLiteral("a(?<word>b)c", "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}

	matched, err := re.MatchString("zabcw")
	if err != nil {
		t.Fatalf("MatchString returned error: %v", err)
	}
	if !matched {
		t.Fatalf("MatchString returned false, want true")
	}

	result, err := re.Exec("zabcw")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("Exec returned nil result")
	}
	if result.Full != "abc" {
		t.Fatalf("Exec Full = %q, want %q", result.Full, "abc")
	}
	if result.Index != 1 {
		t.Fatalf("Exec Index = %d, want 1", result.Index)
	}
	if len(result.Captures) != 2 || result.Captures[0] != "abc" || result.Captures[1] != "b" {
		t.Fatalf("Exec Captures = %#v, want [abc b]", result.Captures)
	}
	if result.NamedCaptures == nil || result.NamedCaptures["word"] != "b" {
		t.Fatalf("Exec NamedCaptures = %#v, want word=b", result.NamedCaptures)
	}

	updated, err := re.ReplaceString("zabcw", "[$$][$&][$`][$'][$1][$<word>]")
	if err != nil {
		t.Fatalf("ReplaceString returned error: %v", err)
	}
	if updated != "z[$][abc][z][w][b][b]w" {
		t.Fatalf("ReplaceString = %q, want %q", updated, "z[$][abc][z][w][b][b]w")
	}

	all, err := re.ReplaceAllString("zabcabcw", "<$<word>>")
	if err != nil {
		t.Fatalf("ReplaceAllString returned error: %v", err)
	}
	if all != "z<b><b>w" {
		t.Fatalf("ReplaceAllString = %q, want %q", all, "z<b><b>w")
	}
}

func TestCompileLiteralRejectsDuplicateFlags(t *testing.T) {
	_, err := CompileLiteral("a", "gg")
	if err == nil {
		t.Fatalf("CompileLiteral succeeded, want duplicate-flag error")
	}
	if !strings.Contains(err.Error(), "duplicate regular expression flag") {
		t.Fatalf("CompileLiteral error = %v, want duplicate flag message", err)
	}
}

func TestCompileLiteralMatchesUnitSuffixPattern(t *testing.T) {
	re, err := CompileLiteral(`^([+-]?[0-9.,]+)(mm|cm|m|in|inch|ft|["'])?$`, "i")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if re.Pattern == nil || re.Pattern.AST == nil || re.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for unit suffix pattern: %#v", re.Pattern)
	}

	matches, err := re.FindStringSubmatch("47.2in")
	if err != nil {
		t.Fatalf("FindStringSubmatch returned error: %v", err)
	}
	if matches == nil {
		t.Fatalf("FindStringSubmatch returned nil, want match")
	}
	if len(matches) != 3 || matches[1] != "47.2" || matches[2] != "in" {
		t.Fatalf("FindStringSubmatch = %#v, want [47.2 in]", matches)
	}
}

func TestCompileLiteralMatchesNumericBackreference(t *testing.T) {
	re, err := CompileLiteral(`^(a)\1$`, "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if re.Pattern == nil || re.Pattern.AST == nil || re.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for numeric backreference: %#v", re.Pattern)
	}

	matched, err := re.MatchString("aa")
	if err != nil {
		t.Fatalf("MatchString returned error: %v", err)
	}
	if !matched {
		t.Fatalf("MatchString returned false, want true")
	}

	result, err := re.Exec("aa")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("Exec returned nil result")
	}
	if result.Full != "aa" {
		t.Fatalf("Exec Full = %q, want %q", result.Full, "aa")
	}
	if len(result.Captures) != 2 || result.Captures[0] != "aa" || result.Captures[1] != "a" {
		t.Fatalf("Exec Captures = %#v, want [aa a]", result.Captures)
	}
}

func TestCompileLiteralMatchesNamedBackreference(t *testing.T) {
	re, err := CompileLiteral(`^(?<word>a)b\k<word>$`, "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if re.Pattern == nil || re.Pattern.AST == nil || re.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for named backreference: %#v", re.Pattern)
	}

	matched, err := re.MatchString("aba")
	if err != nil {
		t.Fatalf("MatchString returned error: %v", err)
	}
	if !matched {
		t.Fatalf("MatchString returned false, want true")
	}

	result, err := re.Exec("aba")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("Exec returned nil result")
	}
	if result.Full != "aba" {
		t.Fatalf("Exec Full = %q, want %q", result.Full, "aba")
	}
	if result.NamedCaptures == nil || result.NamedCaptures["word"] != "a" {
		t.Fatalf("Exec NamedCaptures = %#v, want word=a", result.NamedCaptures)
	}
	if len(result.Captures) != 2 || result.Captures[0] != "aba" || result.Captures[1] != "a" {
		t.Fatalf("Exec Captures = %#v, want [aba a]", result.Captures)
	}
}

func TestCompileLiteralMatchesLookbehindBackreference(t *testing.T) {
	re, err := CompileLiteral(`(?<=(a))\1`, "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if re.Pattern == nil || re.Pattern.AST == nil || re.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for lookbehind backreference: %#v", re.Pattern)
	}

	matched, err := re.MatchString("aa")
	if err != nil {
		t.Fatalf("MatchString returned error: %v", err)
	}
	if !matched {
		t.Fatalf("MatchString returned false, want true")
	}

	result, err := re.Exec("aa")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("Exec returned nil result")
	}
	if result.Full != "a" {
		t.Fatalf("Exec Full = %q, want %q", result.Full, "a")
	}
	if result.Index != 1 {
		t.Fatalf("Exec Index = %d, want 1", result.Index)
	}
	if len(result.Captures) != 2 || result.Captures[0] != "a" || result.Captures[1] != "a" {
		t.Fatalf("Exec Captures = %#v, want [a a]", result.Captures)
	}
}

func TestCompileLiteralRejectsInvalidBackreference(t *testing.T) {
	_, err := CompileLiteral(`(a)\2`, "")
	if err == nil {
		t.Fatalf("CompileLiteral succeeded, want invalid-backreference error")
	}
	if !strings.Contains(err.Error(), "invalid backreference") {
		t.Fatalf("CompileLiteral error = %v, want invalid backreference message", err)
	}
}

func TestCompileLiteralReplacesZeroWidthLookaheadPattern(t *testing.T) {
	re, err := CompileLiteral(`\B(?=(\d{3})+(?!\d))`, "g")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if re.Pattern == nil || re.Pattern.AST == nil || re.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for lookahead pattern: %#v", re.Pattern)
	}

	updated, err := re.ReplaceAllString("1234", ",")
	if err != nil {
		t.Fatalf("ReplaceAllString returned error: %v", err)
	}
	if updated != "1,234" {
		t.Fatalf("ReplaceAllString = %q, want %q", updated, "1,234")
	}
}

func TestCompileLiteralMatchesLookbehindPattern(t *testing.T) {
	re, err := CompileLiteral(`(?<=a)b(c)`, "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if re.Pattern == nil || re.Pattern.AST == nil || re.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for lookbehind pattern: %#v", re.Pattern)
	}

	matched, err := re.MatchString("abc")
	if err != nil {
		t.Fatalf("MatchString returned error: %v", err)
	}
	if !matched {
		t.Fatalf("MatchString returned false, want true")
	}

	result, err := re.Exec("abc")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("Exec returned nil result")
	}
	if result.Full != "bc" {
		t.Fatalf("Exec Full = %q, want %q", result.Full, "bc")
	}
	if result.Index != 1 {
		t.Fatalf("Exec Index = %d, want 1", result.Index)
	}
	if len(result.Captures) != 2 || result.Captures[0] != "bc" || result.Captures[1] != "c" {
		t.Fatalf("Exec Captures = %#v, want [bc c]", result.Captures)
	}
}

func TestCompileLiteralMatchesNegativeLookbehindPattern(t *testing.T) {
	re, err := CompileLiteral(`(?<!a)b(c)`, "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if re.Pattern == nil || re.Pattern.AST == nil || re.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for negative lookbehind pattern: %#v", re.Pattern)
	}

	matched, err := re.MatchString("cbc")
	if err != nil {
		t.Fatalf("MatchString returned error: %v", err)
	}
	if !matched {
		t.Fatalf("MatchString returned false, want true")
	}

	result, err := re.Exec("cbc")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("Exec returned nil result")
	}
	if result.Full != "bc" {
		t.Fatalf("Exec Full = %q, want %q", result.Full, "bc")
	}
	if result.Index != 1 {
		t.Fatalf("Exec Index = %d, want 1", result.Index)
	}
	if len(result.Captures) != 2 || result.Captures[0] != "bc" || result.Captures[1] != "c" {
		t.Fatalf("Exec Captures = %#v, want [bc c]", result.Captures)
	}
}

func TestCompileLiteralPreservesLookaroundCaptures(t *testing.T) {
	lookahead, err := CompileLiteral(`(?=(a))a(b)`, "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if lookahead.Pattern == nil || lookahead.Pattern.AST == nil || lookahead.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for lookahead capture pattern: %#v", lookahead.Pattern)
	}

	lookaheadResult, err := lookahead.Exec("ab")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if lookaheadResult == nil {
		t.Fatalf("Exec returned nil result")
	}
	if len(lookaheadResult.Captures) != 3 || lookaheadResult.Captures[0] != "ab" || lookaheadResult.Captures[1] != "a" || lookaheadResult.Captures[2] != "b" {
		t.Fatalf("Exec Captures = %#v, want [ab a b]", lookaheadResult.Captures)
	}

	lookbehind, err := CompileLiteral(`(?<=(a))b(c)`, "")
	if err != nil {
		t.Fatalf("CompileLiteral returned error: %v", err)
	}
	if lookbehind.Pattern == nil || lookbehind.Pattern.AST == nil || lookbehind.Pattern.re2x != nil {
		t.Fatalf("CompileLiteral used fallback for lookbehind capture pattern: %#v", lookbehind.Pattern)
	}

	lookbehindResult, err := lookbehind.Exec("abc")
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if lookbehindResult == nil {
		t.Fatalf("Exec returned nil result")
	}
	if len(lookbehindResult.Captures) != 3 || lookbehindResult.Captures[0] != "bc" || lookbehindResult.Captures[1] != "a" || lookbehindResult.Captures[2] != "c" {
		t.Fatalf("Exec Captures = %#v, want [bc a c]", lookbehindResult.Captures)
	}
}

func TestUTF16Helpers(t *testing.T) {
	const text = "a😀b"

	if got := UTF16Length(text); got != 4 {
		t.Fatalf("UTF16Length(%q) = %d, want 4", text, got)
	}
	if got := UTF16IndexToByteOffset(text, 0); got != 0 {
		t.Fatalf("UTF16IndexToByteOffset(%q, 0) = %d, want 0", text, got)
	}
	if got := UTF16IndexToByteOffset(text, 1); got != 1 {
		t.Fatalf("UTF16IndexToByteOffset(%q, 1) = %d, want 1", text, got)
	}
	if got := UTF16IndexToByteOffset(text, 2); got != 1 {
		t.Fatalf("UTF16IndexToByteOffset(%q, 2) = %d, want 1", text, got)
	}
	if got := ByteOffsetToUTF16Index(text, 1); got != 1 {
		t.Fatalf("ByteOffsetToUTF16Index(%q, 1) = %d, want 1", text, got)
	}
}
