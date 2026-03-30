package jsregex

import (
	"fmt"
	"strconv"
	"strings"
)

// CompilePattern lowers a pattern and flag set into an immutable compiled
// representation.
func CompilePattern(pattern, flags string) (*CompiledPattern, error) {
	translated, err := expandRegExpUnicodeEscapes(pattern)
	if err != nil {
		return nil, err
	}

	ast, err := parsePattern(translated, flags, true)
	if err != nil {
		return nil, err
	}
	return &CompiledPattern{
		AST:    ast,
		Source: pattern,
		Flags:  flags,
		Mode:   ast.Flags,
	}, nil
}

// CompileLiteral returns a mutable regex instance for a literal or constructor
// call.
func CompileLiteral(pattern, flags string) (*RegexpState, error) {
	compiled, err := CompilePattern(pattern, flags)
	if err != nil {
		return nil, err
	}
	return compiled.NewState(), nil
}

func expandRegExpUnicodeEscapes(pattern string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(pattern); {
		if pattern[i] != '\\' || i+1 >= len(pattern) {
			b.WriteByte(pattern[i])
			i++
			continue
		}
		if pattern[i+1] != 'u' {
			b.WriteByte(pattern[i])
			b.WriteByte(pattern[i+1])
			i += 2
			continue
		}
		if i+5 >= len(pattern) || !isHexDigit(pattern[i+2]) || !isHexDigit(pattern[i+3]) || !isHexDigit(pattern[i+4]) || !isHexDigit(pattern[i+5]) {
			b.WriteByte(pattern[i])
			b.WriteByte(pattern[i+1])
			i += 2
			continue
		}
		code, err := strconv.ParseInt(pattern[i+2:i+6], 16, 32)
		if err != nil {
			return "", fmt.Errorf("invalid unicode escape %q", pattern[i:i+6])
		}
		if code > 0x10FFFF || (code >= 0xD800 && code <= 0xDFFF) {
			return "", fmt.Errorf("unsupported unicode escape %q", pattern[i:i+6])
		}
		b.WriteRune(rune(code))
		i += 6
	}
	return b.String(), nil
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
