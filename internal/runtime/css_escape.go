package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"browsertester/internal/script"
)

func resolveCSSReference(path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "":
		return script.HostObjectReference("CSS"), nil
	case "escape":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			input, err := browserToStringArg(args)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.StringValue(cssEscapeIdentifier(input)), nil
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "CSS."+path))
}

func cssEscapeIdentifier(input string) string {
	if input == "" {
		return ""
	}

	runes := []rune(input)
	var b strings.Builder
	for i, r := range runes {
		switch {
		case r == 0:
			b.WriteRune('\uFFFD')
		case r >= 0x1 && r <= 0x1f || r == 0x7f:
			b.WriteString(cssEscapeCodePoint(r))
		case i == 0 && r >= '0' && r <= '9':
			b.WriteString(cssEscapeCodePoint(r))
		case i == 1 && runes[0] == '-' && r >= '0' && r <= '9':
			b.WriteString(cssEscapeCodePoint(r))
		case i == 0 && r == '-' && len(runes) == 1:
			b.WriteString(`\`)
			b.WriteRune(r)
		case r >= 0x80 || r == '-' || r == '_' ||
			(r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			b.WriteRune(r)
		default:
			b.WriteRune('\\')
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cssEscapeCodePoint(r rune) string {
	return `\` + strconv.FormatInt(int64(r), 16) + " "
}
