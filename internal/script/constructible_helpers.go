package script

// ConstructibleFunctionMarker returns the hidden marker for a constructible function value.
func ConstructibleFunctionMarker(value Value) (string, bool) {
	if value.Kind != ValueKindFunction || value.Function == nil || !value.Function.constructible {
		return "", false
	}
	return classicJSConstructibleFunctionMarker(value.Function)
}

// ConstructibleInstanceMarkerKey returns the hidden property key used by instanceof checks.
func ConstructibleInstanceMarkerKey(marker string) string {
	return classicJSInstanceMarkerKey(marker)
}
