package jsregex

import (
	"fmt"
	"strings"
)

const backreferencePlaceholderPrefix = "__jsregex_backref_"

func replaceBackreferences(pattern string) (string, []BackreferenceSpec, error) {
	if !strings.Contains(pattern, "\\") {
		return pattern, nil, nil
	}

	var out strings.Builder
	specs := make([]BackreferenceSpec, 0, 2)
	inClass := false

	for i := 0; i < len(pattern); {
		ch := pattern[i]
		if ch == '[' {
			if !inClass {
				inClass = true
			}
			out.WriteByte(ch)
			i++
			continue
		}
		if ch == ']' && inClass {
			inClass = false
			out.WriteByte(ch)
			i++
			continue
		}
		if ch != '\\' || i+1 >= len(pattern) {
			out.WriteByte(ch)
			i++
			continue
		}

		next := pattern[i+1]
		if inClass {
			out.WriteByte(ch)
			out.WriteByte(next)
			i += 2
			continue
		}

		if next == 'k' && i+2 < len(pattern) && pattern[i+2] == '<' {
			end := strings.IndexByte(pattern[i+3:], '>')
			if end < 0 {
				return "", nil, fmt.Errorf("unterminated named backreference in %q", pattern)
			}
			name := pattern[i+3 : i+3+end]
			if name == "" {
				return "", nil, fmt.Errorf("empty named backreference in %q", pattern)
			}
			id := len(specs)
			specs = append(specs, BackreferenceSpec{
				TargetName: name,
			})
			out.WriteString("(?P<")
			out.WriteString(backreferencePlaceholderName(id))
			out.WriteString(">)")
			i += end + 4
			continue
		}

		if next >= '1' && next <= '9' {
			j := i + 2
			for j < len(pattern) && pattern[j] >= '0' && pattern[j] <= '9' {
				j++
			}
			ref, err := parseDecimalIndex(pattern[i+1 : j])
			if err != nil {
				return "", nil, err
			}
			id := len(specs)
			specs = append(specs, BackreferenceSpec{
				TargetNumber: ref,
			})
			out.WriteString("(?P<")
			out.WriteString(backreferencePlaceholderName(id))
			out.WriteString(">)")
			i = j
			continue
		}

		out.WriteByte(ch)
		out.WriteByte(next)
		i += 2
	}

	return out.String(), specs, nil
}

func backreferencePlaceholderName(id int) string {
	return fmt.Sprintf("%s%d", backreferencePlaceholderPrefix, id)
}

func isBackreferencePlaceholderName(name string) bool {
	return strings.HasPrefix(name, backreferencePlaceholderPrefix)
}

func backreferencePlaceholderIndex(name string) (int, bool) {
	if !isBackreferencePlaceholderName(name) {
		return 0, false
	}
	idText := strings.TrimPrefix(name, backreferencePlaceholderPrefix)
	if idText == "" {
		return 0, false
	}
	id, err := parseDecimalIndex(idText)
	if err != nil {
		return 0, false
	}
	return id, true
}
