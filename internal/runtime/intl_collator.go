package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/collation"
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
		return script.NumberValue(float64(collation.Compare(left, right, locale, numeric))), nil
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
