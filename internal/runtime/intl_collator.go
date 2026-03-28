package runtime

import (
	"fmt"
	"strings"
	"unicode"

	"browsertester/internal/script"
)

func browserCollatorConstructor(args []script.Value) (script.Value, error) {
	locale := "en-US"
	var options script.Value
	hasOptions := false

	switch len(args) {
	case 0:
	case 1:
		if args[0].Kind == script.ValueKindObject {
			options = args[0]
			hasOptions = true
		} else if args[0].Kind != script.ValueKindUndefined && args[0].Kind != script.ValueKindNull {
			locale = strings.TrimSpace(script.ToJSString(args[0]))
		}
	default:
		if args[0].Kind != script.ValueKindUndefined && args[0].Kind != script.ValueKindNull {
			locale = strings.TrimSpace(script.ToJSString(args[0]))
		}
		if args[1].Kind != script.ValueKindObject {
			return script.UndefinedValue(), fmt.Errorf("Intl.Collator options argument must be an object")
		}
		options = args[1]
		hasOptions = true
	}

	if locale == "" {
		locale = "en-US"
	}

	numeric := false
	if hasOptions {
		if value, ok := objectProperty(options, "numeric"); ok {
			if value.Kind != script.ValueKindBool {
				return script.UndefinedValue(), fmt.Errorf("Intl.Collator numeric must be a boolean")
			}
			numeric = value.Bool
		}
	}

	compare := script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if len(args) != 2 {
			return script.UndefinedValue(), fmt.Errorf("Intl.Collator#compare expects 2 arguments")
		}
		left := script.ToJSString(args[0])
		right := script.ToJSString(args[1])
		return script.NumberValue(float64(browserCollatorCompare(left, right, locale, numeric))), nil
	})
	resolvedOptions := script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if len(args) != 0 {
			return script.UndefinedValue(), fmt.Errorf("Intl.Collator#resolvedOptions expects no arguments")
		}
		return script.ObjectValue([]script.ObjectEntry{
			{Key: "locale", Value: script.StringValue(locale)},
			{Key: "numeric", Value: script.BoolValue(numeric)},
		}), nil
	})

	return script.ObjectValue([]script.ObjectEntry{
		{Key: "compare", Value: compare},
		{Key: "resolvedOptions", Value: resolvedOptions},
	}), nil
}

func browserCollatorCompare(left, right, locale string, numeric bool) int {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	leftIndex := 0
	rightIndex := 0

	for leftIndex < len(leftRunes) && rightIndex < len(rightRunes) {
		leftRune := leftRunes[leftIndex]
		rightRune := rightRunes[rightIndex]

		if numeric && isASCIIDigit(leftRune) && isASCIIDigit(rightRune) {
			leftDigits, nextLeft := browserCollatorDigitRun(leftRunes, leftIndex)
			rightDigits, nextRight := browserCollatorDigitRun(rightRunes, rightIndex)
			if cmp := browserCollatorCompareDigits(leftDigits, rightDigits); cmp != 0 {
				return cmp
			}
			leftIndex = nextLeft
			rightIndex = nextRight
			continue
		}

		if cmp := browserCollatorCompareRune(leftRune, rightRune, locale); cmp != 0 {
			return cmp
		}
		leftIndex++
		rightIndex++
	}

	switch {
	case leftIndex < len(leftRunes):
		return 1
	case rightIndex < len(rightRunes):
		return -1
	default:
		return 0
	}
}

func browserCollatorDigitRun(runes []rune, start int) (string, int) {
	end := start
	for end < len(runes) && isASCIIDigit(runes[end]) {
		end++
	}
	return string(runes[start:end]), end
}

func browserCollatorCompareDigits(left, right string) int {
	left = strings.TrimLeft(left, "0")
	right = strings.TrimLeft(right, "0")
	if left == "" {
		left = "0"
	}
	if right == "" {
		right = "0"
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func browserCollatorCompareRune(left, right rune, locale string) int {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "sv") {
		leftWeight := browserCollatorSwedishWeight(left)
		rightWeight := browserCollatorSwedishWeight(right)
		if leftWeight < rightWeight {
			return -1
		}
		if leftWeight > rightWeight {
			return 1
		}
		return 0
	}

	leftLower := unicode.ToLower(left)
	rightLower := unicode.ToLower(right)
	if leftLower < rightLower {
		return -1
	}
	if leftLower > rightLower {
		return 1
	}
	return 0
}

func browserCollatorSwedishWeight(r rune) int {
	lower := unicode.ToLower(r)
	switch lower {
	case 'å':
		return 27
	case 'ä':
		return 28
	case 'ö':
		return 29
	}
	if lower >= 'a' && lower <= 'z' {
		return int(lower-'a') + 1
	}
	return 1000 + int(lower)
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
