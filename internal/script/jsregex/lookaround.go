package jsregex

import (
	"fmt"
	"regexp/syntax"
	"strings"
)

const lookaroundPlaceholderPrefix = "__jsregex_lookaround_"

func replaceLookarounds(pattern, flags string, allowBackreferences bool) (string, []LookaroundSpec, error) {
	if !strings.Contains(pattern, "(?") {
		return pattern, nil, nil
	}

	var out strings.Builder
	specs := make([]LookaroundSpec, 0, 2)

	for i := 0; i < len(pattern); {
		if pattern[i] != '(' || i+2 >= len(pattern) || pattern[i+1] != '?' {
			switch pattern[i] {
			case '\\':
				if i+1 < len(pattern) {
					out.WriteByte(pattern[i])
					out.WriteByte(pattern[i+1])
					i += 2
					continue
				}
			case '[':
				end := scanCharClass(pattern, i+1)
				if end < 0 {
					return "", nil, fmt.Errorf("unterminated character class in %q", pattern)
				}
				out.WriteString(pattern[i : end+1])
				i = end + 1
				continue
			}
			out.WriteByte(pattern[i])
			i++
			continue
		}

		switch pattern[i+2] {
		case '=', '!':
			end, ok := findGroupClose(pattern, i+3)
			if !ok {
				return "", nil, fmt.Errorf("unterminated lookahead in %q", pattern)
			}
			body := pattern[i+3 : end]
			bodyAST, err := parseTranslatedPattern(body, flags, false)
			if err != nil {
				return "", nil, err
			}
			id := len(specs)
			specs = append(specs, LookaroundSpec{
				Positive:   pattern[i+2] == '=',
				Lookbehind: false,
				Width:      0,
				AST:        bodyAST,
			})
			out.WriteString("(?P<")
			out.WriteString(lookaroundPlaceholderName(id))
			out.WriteString(">)")
			i = end + 1
		case '<':
			if i+3 >= len(pattern) || (pattern[i+3] != '=' && pattern[i+3] != '!') {
				out.WriteByte(pattern[i])
				out.WriteByte(pattern[i+1])
				i += 2
				continue
			}
			end, ok := findGroupClose(pattern, i+4)
			if !ok {
				return "", nil, fmt.Errorf("unterminated lookbehind in %q", pattern)
			}
			body := pattern[i+4 : end]
			bodyAST, err := parseTranslatedPattern(body, flags, false)
			if err != nil {
				return "", nil, err
			}
			width, ok := lookaroundExactWidth(bodyAST.Root)
			if !ok {
				return "", nil, ErrNativeUnsupported
			}
			id := len(specs)
			specs = append(specs, LookaroundSpec{
				Positive:   pattern[i+3] == '=',
				Lookbehind: true,
				Width:      width,
				AST:        bodyAST,
			})
			out.WriteString("(?P<")
			out.WriteString(lookaroundPlaceholderName(id))
			out.WriteString(">)")
			i = end + 1
		default:
			if pattern[i+2] == '<' && i+3 < len(pattern) && (pattern[i+3] == '=' || pattern[i+3] == '!') {
				return "", nil, ErrNativeUnsupported
			}
			out.WriteByte(pattern[i])
			out.WriteByte(pattern[i+1])
			i += 2
		}
	}

	return out.String(), specs, nil
}

func lookaroundPlaceholderName(id int) string {
	return fmt.Sprintf("%s%d", lookaroundPlaceholderPrefix, id)
}

func isLookaroundPlaceholderName(name string) bool {
	return strings.HasPrefix(name, lookaroundPlaceholderPrefix)
}

func lookaroundPlaceholderIndex(name string) (int, bool) {
	if !isLookaroundPlaceholderName(name) {
		return 0, false
	}
	idText := strings.TrimPrefix(name, lookaroundPlaceholderPrefix)
	if idText == "" {
		return 0, false
	}
	id, err := parseDecimalIndex(idText)
	if err != nil {
		return 0, false
	}
	return id, true
}

func parseDecimalIndex(text string) (int, error) {
	n := 0
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid decimal index %q", text)
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

func findGroupClose(pattern string, start int) (int, bool) {
	depth := 1
	inClass := false
	escaped := false
	for i := start; i < len(pattern); i++ {
		ch := pattern[i]
		if escaped {
			escaped = false
			continue
		}
		switch ch {
		case '\\':
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
			if !inClass {
				depth++
			}
		case ')':
			if inClass {
				continue
			}
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func scanCharClass(pattern string, start int) int {
	escaped := false
	for i := start; i < len(pattern); i++ {
		ch := pattern[i]
		if escaped {
			escaped = false
			continue
		}
		switch ch {
		case '\\':
			escaped = true
		case ']':
			return i
		}
	}
	return -1
}

func lookaroundExactWidth(re *syntax.Regexp) (int, bool) {
	if re == nil {
		return 0, true
	}
	switch re.Op {
	case syntax.OpNoMatch, syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpBeginText, syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return 0, true
	case syntax.OpLiteral:
		return len(re.Rune), true
	case syntax.OpCharClass, syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		return 1, true
	case syntax.OpCapture:
		if len(re.Sub) != 1 {
			return 0, false
		}
		return lookaroundExactWidth(re.Sub[0])
	case syntax.OpConcat:
		total := 0
		for _, sub := range re.Sub {
			width, ok := lookaroundExactWidth(sub)
			if !ok {
				return 0, false
			}
			total += width
		}
		return total, true
	case syntax.OpAlternate:
		if len(re.Sub) == 0 {
			return 0, true
		}
		width, ok := lookaroundExactWidth(re.Sub[0])
		if !ok {
			return 0, false
		}
		for _, sub := range re.Sub[1:] {
			nextWidth, ok := lookaroundExactWidth(sub)
			if !ok || nextWidth != width {
				return 0, false
			}
		}
		return width, true
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		if len(re.Sub) != 1 {
			return 0, false
		}
		childWidth, ok := lookaroundExactWidth(re.Sub[0])
		if !ok {
			return 0, false
		}
		if re.Max < 0 {
			return 0, false
		}
		if childWidth == 0 {
			return 0, true
		}
		if re.Min != re.Max {
			return 0, false
		}
		return childWidth * re.Min, true
	default:
		return 0, false
	}
}
