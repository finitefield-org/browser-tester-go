package runtime

import (
	"fmt"
	"strconv"

	"browsertester/internal/script"
)

const browserRegExpPatternKey = "\x00classic-js-regexp:pattern"
const browserRegExpFlagsKey = "\x00classic-js-regexp:flags"

func browserRegExpConstructor(args []script.Value) (script.Value, error) {
	if len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("RegExp expects at most 2 arguments")
	}

	pattern := ""
	flags := ""
	if len(args) >= 1 {
		if literalPattern, literalFlags, ok := script.RegExpLiteralParts(args[0]); ok {
			pattern = literalPattern
			flags = literalFlags
		} else {
			pattern = script.ToJSString(args[0])
		}
	}
	if len(args) == 2 {
		flags = script.ToJSString(args[1])
	}

	compiled, err := script.CompileRegExpLiteral(pattern, flags)
	if err != nil {
		return script.UndefinedValue(), err
	}

	literal := "/" + pattern + "/" + flags
	test := func(args []script.Value) (script.Value, error) {
		input := script.UndefinedValue()
		if len(args) > 0 {
			input = args[0]
		}
		matched, err := compiled.MatchString(script.ToJSString(input))
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.BoolValue(matched), nil
	}
	exec := func(args []script.Value) (script.Value, error) {
		input := script.UndefinedValue()
		if len(args) > 0 {
			input = args[0]
		}
		text := script.ToJSString(input)
		result, err := compiled.Exec(text)
		if err != nil {
			return script.UndefinedValue(), err
		}
		if result == nil {
			return script.NullValue(), nil
		}
		entries := make([]script.ObjectEntry, 0, len(result.Captures)+3)
		for i, match := range result.Captures {
			entries = append(entries, script.ObjectEntry{Key: strconv.Itoa(i), Value: script.StringValue(match)})
		}
		entries = append(entries,
			script.ObjectEntry{Key: "length", Value: script.NumberValue(float64(len(result.Captures)))},
			script.ObjectEntry{Key: "index", Value: script.NumberValue(float64(result.Index))},
			script.ObjectEntry{Key: "input", Value: script.StringValue(result.Input)},
		)
		return script.ObjectValue(entries), nil
	}

	return script.ObjectValue([]script.ObjectEntry{
		{Key: browserRegExpPatternKey, Value: script.StringValue(pattern)},
		{Key: browserRegExpFlagsKey, Value: script.StringValue(flags)},
		{Key: "source", Value: script.StringValue(pattern)},
		{Key: "flags", Value: script.StringValue(flags)},
		{Key: "test", Value: script.NativeFunctionValue(test)},
		{Key: "exec", Value: script.NativeFunctionValue(exec)},
		{Key: "toString", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("RegExp.toString accepts no arguments")
			}
			return script.StringValue(literal), nil
		})},
	}), nil
}
