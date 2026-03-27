package script

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const browserDateInternalPrefix = "\x00browser-date:"
const browserDateTimestampKey = browserDateInternalPrefix + "timestamp"

func IsInternalObjectKey(name string) bool {
	return classicJSIsInternalObjectKey(name)
}

func BrowserDateValue(ms int64) Value {
	return ObjectValue([]ObjectEntry{
		{
			Key:   browserDateTimestampKey,
			Value: NumberValue(float64(ms)),
		},
	})
}

func BrowserDateTimestamp(value Value) (int64, bool) {
	if value.Kind != ValueKindObject {
		return 0, false
	}
	for i := len(value.Object) - 1; i >= 0; i-- {
		if value.Object[i].Key != browserDateTimestampKey {
			continue
		}
		switch timestamp := value.Object[i].Value; timestamp.Kind {
		case ValueKindNumber:
			return int64(timestamp.Number), true
		case ValueKindBigInt:
			parsed, err := strconv.ParseInt(timestamp.BigInt, 10, 64)
			if err != nil {
				return 0, false
			}
			return parsed, true
		}
		return 0, false
	}
	return 0, false
}

func BrowserDateISOString(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
}

func RegExpLiteralParts(value Value) (pattern string, flags string, ok bool) {
	if value.Kind != ValueKindObject {
		return "", "", false
	}
	patternValue, ok := lookupObjectProperty(value.Object, classicJSRegExpPatternKey)
	if !ok || patternValue.Kind != ValueKindString {
		return "", "", false
	}
	flagsValue, ok := lookupObjectProperty(value.Object, classicJSRegExpFlagsKey)
	if !ok || flagsValue.Kind != ValueKindString {
		return "", "", false
	}
	return patternValue.String, flagsValue.String, true
}

func CompileRegExpLiteral(pattern, flags string) (*regexp.Regexp, error) {
	return classicJSCompileRegExpLiteral(pattern, flags)
}

func BuiltinFunctionValue(name string, params []string, restName string, body string, bodyIsBlock bool) Value {
	functionParams := make([]classicJSFunctionParameter, len(params))
	for i, param := range params {
		functionParams[i] = classicJSFunctionParameter{name: param}
	}
	return FunctionValue(&classicJSArrowFunction{
		name:        name,
		params:      functionParams,
		restName:    restName,
		body:        body,
		bodyIsBlock: bodyIsBlock,
		allowReturn: true,
		env:         newClassicJSEnvironment(),
	})
}

func browserDateTimeString(ms int64) string {
	return BrowserDateISOString(ms)
}

func browserNumberToString(value float64, radix int) string {
	if math.IsNaN(value) {
		return "NaN"
	}
	if math.IsInf(value, 1) {
		return "Infinity"
	}
	if math.IsInf(value, -1) {
		return "-Infinity"
	}
	if radix == 10 {
		return ToJSString(NumberValue(value))
	}
	if radix < 2 || radix > 36 {
		radix = 10
	}

	negative := value < 0
	if negative {
		value = -value
	}
	intPart, fracPart := math.Modf(value)
	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	intText := strconv.FormatInt(int64(intPart), radix)
	if intPart == 0 && fracPart > 0 {
		intText = "0"
	}
	if fracPart == 0 {
		if negative {
			return "-" + intText
		}
		return intText
	}

	var fraction strings.Builder
	for i := 0; i < 16 && fracPart > 0; i++ {
		fracPart *= float64(radix)
		digit := int(fracPart)
		if digit < 0 {
			digit = 0
		}
		if digit >= radix {
			digit = radix - 1
		}
		fraction.WriteByte(digits[digit])
		fracPart -= float64(digit)
	}

	if negative {
		return "-" + intText + "." + fraction.String()
	}
	return intText + "." + fraction.String()
}
