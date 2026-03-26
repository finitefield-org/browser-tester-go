package script

import (
	"fmt"
	"strconv"
	"strings"
)

func parseHostInvocation(source string) (string, []Value, error) {
	text := strings.TrimSpace(source)
	if text == "" {
		return "", nil, fmt.Errorf("host dispatch requires a non-empty method name")
	}

	open := strings.IndexByte(text, '(')
	if open == -1 {
		if strings.ContainsAny(text, " \t\r\n") {
			return "", nil, fmt.Errorf("host dispatch requires a simple method name")
		}
		return text, nil, nil
	}

	method := strings.TrimSpace(text[:open])
	if method == "" {
		return "", nil, fmt.Errorf("host dispatch requires a non-empty method name")
	}
	if !strings.HasSuffix(text, ")") {
		return "", nil, fmt.Errorf("host dispatch has an unterminated argument list")
	}

	args, err := parseHostArgs(text[open+1 : len(text)-1])
	if err != nil {
		return "", nil, err
	}
	return method, args, nil
}

func parseHostArgs(input string) ([]Value, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return []Value{}, nil
	}

	args := make([]Value, 0, 4)
	i := 0
	for {
		i = skipHostSpaces(text, i)
		if i >= len(text) {
			break
		}

		next, err := scanHostArgEnd(text, i)
		if err != nil {
			return nil, err
		}
		value, err := parseHostArg(text[i:next])
		if err != nil {
			return nil, err
		}
		args = append(args, value)

		i = skipHostSpaces(text, next)
		if i >= len(text) {
			break
		}
		if text[i] != ',' {
			return nil, fmt.Errorf("host dispatch arguments must be comma-separated")
		}
		i++
	}
	return args, nil
}

func scanHostArgEnd(input string, start int) (int, error) {
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int

	for i := start; i < len(input); i++ {
		ch := input[i]
		if lineComment {
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && i+1 < len(input) && input[i+1] == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == ',' && parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
			return i, nil
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '/':
			if i+1 < len(input) {
				switch input[i+1] {
				case '/':
					lineComment = true
					i++
				case '*':
					blockComment = true
					i++
				}
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		}
	}

	if quote != 0 {
		return 0, fmt.Errorf("unterminated quoted host argument")
	}
	if blockComment {
		return 0, fmt.Errorf("unterminated block comment in host dispatch argument")
	}
	return len(input), nil
}

func parseHostArg(raw string) (Value, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Value{}, fmt.Errorf("host dispatch argument is missing")
	}

	switch raw[0] {
	case '"', '\'':
		quote := raw[0]
		i := 1
		var b strings.Builder
		for i < len(raw) {
			if raw[i] == quote {
				return StringValue(b.String()), nil
			}
			if raw[i] == '\\' && i+1 < len(raw) {
				i++
				switch raw[i] {
				case '\\', '\'', '"':
					b.WriteByte(raw[i])
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				default:
					b.WriteByte(raw[i])
				}
				i++
				continue
			}
			b.WriteByte(raw[i])
			i++
		}
		return Value{}, fmt.Errorf("unterminated quoted host argument")
	default:
		if strings.HasPrefix(raw, "expr(") && strings.HasSuffix(raw, ")") {
			inner := strings.TrimSpace(raw[len("expr(") : len(raw)-1])
			if inner == "" {
				return Value{}, fmt.Errorf("expression wrapper requires a non-empty source")
			}
			return InvocationValue(inner), nil
		}
		if strings.EqualFold(raw, "true") {
			return BoolValue(true), nil
		}
		if strings.EqualFold(raw, "false") {
			return BoolValue(false), nil
		}
		if number, err := strconv.ParseFloat(raw, 64); err == nil {
			return NumberValue(number), nil
		}
		return StringValue(raw), nil
	}
}

func skipHostSpaces(text string, i int) int {
	for i < len(text) {
		switch text[i] {
		case ' ', '\t', '\n', '\r', '\f':
			i++
		default:
			return i
		}
	}
	return i
}
