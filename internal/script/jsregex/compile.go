package jsregex

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

// CompilePattern lowers a pattern and flag set into an immutable compiled
// representation.
func CompilePattern(pattern, flags string) (*CompiledPattern, error) {
	translated, err := expandRegExpUnicodeEscapes(pattern)
	if err != nil {
		return nil, err
	}

	ast, err := parseTranslatedPattern(translated, flags, true)
	if err == nil {
		return &CompiledPattern{
			AST:    ast,
			Source: pattern,
			Flags:  flags,
			Mode:   ast.Flags,
		}, nil
	}

	if !errors.Is(err, ErrNativeUnsupported) {
		return nil, err
	}

	parsedFlags, flagErr := ParseFlags(flags)
	if flagErr != nil {
		return nil, flagErr
	}
	options := buildRegexp2Options(parsedFlags)
	compiled2, extErr := regexp2.Compile(translated, options|regexp2.ECMAScript)
	if extErr != nil {
		return nil, fmt.Errorf("%v; regexp2 fallback failed: %v", err, extErr)
	}
	return &CompiledPattern{
		AST:    ast,
		Source: pattern,
		Flags:  flags,
		Mode:   parsedFlags,
		re2x:   compiled2,
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

func buildRegexp2Options(flags FlagSet) regexp2.RegexOptions {
	var options regexp2.RegexOptions

	if flags.IgnoreCase {
		options |= regexp2.IgnoreCase
	}
	if flags.Multiline {
		options |= regexp2.Multiline
	}
	if flags.DotAll {
		options |= regexp2.Singleline
	}
	if flags.Unicode {
		options |= regexp2.Unicode
	}
	return options
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

func regexpNeedsRegexp2(pattern string) bool {
	inClass := false
	escaped := false
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if escaped {
			escaped = false
			continue
		}
		switch ch {
		case '\\':
			if !inClass && i+2 < len(pattern) {
				next := pattern[i+1]
				switch {
				case next >= '1' && next <= '9':
					return true
				case next == 'k' && pattern[i+2] == '<':
					return true
				}
			}
			escaped = true
		case '[':
			if !inClass {
				inClass = true
			}
		case ']':
			if inClass {
				inClass = false
			}
		case '(':
			if inClass || i+1 >= len(pattern) || pattern[i+1] != '?' || i+2 >= len(pattern) {
				continue
			}
			if pattern[i+2] == '<' && i+3 < len(pattern) {
				switch pattern[i+3] {
				case '=', '!':
					return true
				}
			}
		}
	}
	return false
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
