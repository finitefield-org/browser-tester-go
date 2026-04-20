package runtime

import "browsertester/internal/script"

func (s *Session) errorConstructorValue() script.Value {
	if s == nil {
		return buildBrowserErrorValue()
	}
	if s.hasErrorConstructor {
		return s.errorConstructor
	}
	s.errorConstructor = buildBrowserErrorValue()
	s.hasErrorConstructor = true
	return s.errorConstructor
}

func browserErrorValue(session *Session) script.Value {
	if session == nil {
		return buildBrowserErrorValue()
	}
	return session.errorConstructorValue()
}

func buildBrowserErrorValue() script.Value {
	var value script.Value
	value = script.NativeConstructibleNamedFunctionValue("Error",
		func(args []script.Value) (script.Value, error) {
			return browserErrorInstanceValue(value, args)
		},
		func(args []script.Value) (script.Value, error) {
			return browserErrorInstanceValue(value, args)
		},
	)
	return value
}

func browserErrorInstanceValue(constructor script.Value, args []script.Value) (script.Value, error) {
	marker, ok := script.ConstructibleFunctionMarker(constructor)
	if !ok || marker == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindRuntime, "Error constructor is unavailable")
	}

	message := ""
	if len(args) > 0 && args[0].Kind != script.ValueKindUndefined {
		message = script.ToJSString(args[0])
	}

	return script.ObjectValue([]script.ObjectEntry{
		{Key: "message", Value: script.StringValue(message)},
		{Key: "name", Value: script.StringValue("Error")},
		{Key: script.ConstructibleInstanceMarkerKey(marker), Value: script.BoolValue(true)},
	}), nil
}
