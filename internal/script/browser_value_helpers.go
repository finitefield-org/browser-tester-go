package script

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const browserDateInternalPrefix = "\x00browser-date:"
const browserDateTimestampKey = browserDateInternalPrefix + "timestamp"
const browserTypedArrayInternalPrefix = "\x00browser-typed-array:"
const BrowserUint8ArrayBytesKey = browserTypedArrayInternalPrefix + "bytes"
const symbolObjectKeyPrefix = "\x00classic-js-symbol:"

var symbolDescriptions sync.Map

func IsInternalObjectKey(name string) bool {
	return classicJSIsInternalObjectKey(name)
}

func registerSymbolDescription(symbolID, description string) {
	if symbolID == "" {
		return
	}
	symbolDescriptions.Store(symbolID, description)
}

func symbolDescriptionForID(symbolID string) string {
	if symbolID == "" {
		return ""
	}
	if description, ok := symbolDescriptions.Load(symbolID); ok {
		if text, ok := description.(string); ok {
			return text
		}
	}
	return ""
}

func IsSymbolObjectKey(name string) bool {
	return strings.HasPrefix(name, symbolObjectKeyPrefix)
}

func SymbolObjectKey(value Value) (string, bool) {
	if value.Kind != ValueKindSymbol || value.SymbolID == "" {
		return "", false
	}
	return symbolObjectKeyPrefix + value.SymbolID, true
}

func SymbolValueFromObjectKey(key string) (Value, bool) {
	if !IsSymbolObjectKey(key) {
		return Value{}, false
	}
	symbolID := strings.TrimPrefix(key, symbolObjectKeyPrefix)
	return Value{
		Kind:              ValueKindSymbol,
		SymbolDescription: symbolDescriptionForID(symbolID),
		SymbolID:          symbolID,
	}, true
}

func propertyKeyString(value Value) string {
	if key, ok := SymbolObjectKey(value); ok {
		return key
	}
	return ToJSString(value)
}

func IsConstructibleFunctionValue(value Value) bool {
	if value.Kind != ValueKindFunction {
		return false
	}
	if value.NativeConstructibleFunction != nil {
		return true
	}
	return value.Function != nil && value.Function.constructible
}

func browserElementReferenceTag(path string) (string, bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "element:") {
		return "", false
	}
	remainder := strings.TrimPrefix(normalized, "element:")
	if remainder == "" {
		return "", false
	}
	if index := strings.IndexByte(remainder, '.'); index >= 0 {
		remainder = remainder[:index]
	}
	if index := strings.IndexByte(remainder, '@'); index >= 0 {
		if index+1 >= len(remainder) {
			return "", false
		}
		return remainder[index+1:], true
	}
	return "", false
}

func browserHTMLConstructorTag(name string) (string, bool) {
	normalized := strings.TrimSpace(name)
	if normalized == "HTMLElement" {
		return "", true
	}
	if !strings.HasPrefix(normalized, "HTML") || !strings.HasSuffix(normalized, "Element") {
		return "", false
	}
	tag := strings.TrimSuffix(strings.TrimPrefix(normalized, "HTML"), "Element")
	if tag == "" {
		return "", false
	}
	return strings.ToLower(tag), true
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

func BrowserDateLocaleDateString(ms int64, locale string) string {
	t := time.UnixMilli(ms).UTC()
	normalized := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(normalized, "ja") {
		return t.Format("2006/01/02")
	}
	return strconv.Itoa(int(t.Month())) + "/" + strconv.Itoa(t.Day()) + "/" + strconv.Itoa(t.Year())
}

func BrowserDateYear(ms int64) int {
	return time.UnixMilli(ms).UTC().Year()
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
