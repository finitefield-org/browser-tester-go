package runtime

import (
	"strconv"

	"browsertester/internal/script"
)

func browserToStringValue(value script.Value) (string, error) {
	if value.Kind == script.ValueKindSymbol {
		return "", script.NewError(script.ErrorKindRuntime, "Cannot convert a Symbol value to a string")
	}
	return script.ToJSString(value), nil
}

func browserToStringArg(args []script.Value) (string, error) {
	if len(args) == 0 {
		return script.ToJSString(script.UndefinedValue()), nil
	}
	return browserToStringValue(args[0])
}

func browserRequiredStringArg(method string, args []script.Value, index int) (string, error) {
	if index >= len(args) {
		return "", script.NewError(script.ErrorKindRuntime, method+" requires argument "+strconv.Itoa(index+1))
	}
	return browserToStringValue(args[index])
}
