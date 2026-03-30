package script

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"browsertester/internal/script/jsregex"
)

const browserDateInternalPrefix = "\x00browser-date:"
const browserDateTimestampKey = browserDateInternalPrefix + "timestamp"
const browserTypedArrayInternalPrefix = "\x00browser-typed-array:"
const BrowserUint8ArrayBytesKey = browserTypedArrayInternalPrefix + "bytes"
const symbolObjectKeyPrefix = "\x00classic-js-symbol:"

var symbolDescriptions sync.Map
var wellKnownSymbolValues sync.Map
var wellKnownSymbolIterator = WellKnownSymbolValue("Symbol.iterator")

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

func WellKnownSymbolValue(name string) Value {
	if value, ok := wellKnownSymbolValues.Load(name); ok {
		if symbol, ok := value.(Value); ok {
			return symbol
		}
	}
	symbol := SymbolValue(name)
	if value, loaded := wellKnownSymbolValues.LoadOrStore(name, symbol); loaded {
		if cached, ok := value.(Value); ok {
			return cached
		}
	}
	return symbol
}

func isWellKnownSymbolIterator(value Value) bool {
	return value.Kind == ValueKindSymbol && value.SymbolID == wellKnownSymbolIterator.SymbolID
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
	if normalized == "HTMLImageElement" {
		return "img", true
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

func BrowserDateSetTimestamp(value *Value, ms int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	for i := len(value.Object) - 1; i >= 0; i-- {
		if value.Object[i].Key != browserDateTimestampKey {
			continue
		}
		value.Object[i].Value = NumberValue(float64(ms))
		return true
	}
	return false
}

func BrowserDateSetDayOfMonth(value *Value, day int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	currentMs, ok := BrowserDateTimestamp(*value)
	if !ok {
		return false
	}
	current := time.UnixMilli(currentMs).UTC()
	next := time.Date(
		current.Year(),
		current.Month(),
		int(day),
		current.Hour(),
		current.Minute(),
		current.Second(),
		current.Nanosecond(),
		time.UTC,
	).UnixMilli()
	return BrowserDateSetTimestamp(value, next)
}

func BrowserDateSetMonth(value *Value, month int64, day *int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	currentMs, ok := BrowserDateTimestamp(*value)
	if !ok {
		return false
	}
	current := time.UnixMilli(currentMs).UTC()
	dom := int64(current.Day())
	if day != nil {
		dom = *day
	}
	next := time.Date(
		current.Year(),
		time.Month(month+1),
		int(dom),
		current.Hour(),
		current.Minute(),
		current.Second(),
		current.Nanosecond(),
		time.UTC,
	).UnixMilli()
	return BrowserDateSetTimestamp(value, next)
}

func BrowserDateSetFullYear(value *Value, year int64, month *int64, day *int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	currentMs, ok := BrowserDateTimestamp(*value)
	if !ok {
		return false
	}
	current := time.UnixMilli(currentMs).UTC()
	mon := current.Month()
	if month != nil {
		mon = time.Month(*month + 1)
	}
	dom := int64(current.Day())
	if day != nil {
		dom = *day
	}
	next := time.Date(
		int(year),
		mon,
		int(dom),
		current.Hour(),
		current.Minute(),
		current.Second(),
		current.Nanosecond(),
		time.UTC,
	).UnixMilli()
	return BrowserDateSetTimestamp(value, next)
}

func BrowserDateSetMilliseconds(value *Value, milliseconds int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	currentMs, ok := BrowserDateTimestamp(*value)
	if !ok {
		return false
	}
	current := time.UnixMilli(currentMs).UTC()
	next := time.Date(
		current.Year(),
		current.Month(),
		current.Day(),
		current.Hour(),
		current.Minute(),
		current.Second(),
		int(milliseconds)*int(time.Millisecond),
		time.UTC,
	).UnixMilli()
	return BrowserDateSetTimestamp(value, next)
}

func BrowserDateSetSeconds(value *Value, seconds int64, milliseconds *int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	currentMs, ok := BrowserDateTimestamp(*value)
	if !ok {
		return false
	}
	current := time.UnixMilli(currentMs).UTC()
	ms := int64(current.Nanosecond() / int(time.Millisecond))
	if milliseconds != nil {
		ms = *milliseconds
	}
	next := time.Date(
		current.Year(),
		current.Month(),
		current.Day(),
		current.Hour(),
		current.Minute(),
		int(seconds),
		int(ms)*int(time.Millisecond),
		time.UTC,
	).UnixMilli()
	return BrowserDateSetTimestamp(value, next)
}

func BrowserDateSetMinutes(value *Value, minutes int64, seconds *int64, milliseconds *int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	currentMs, ok := BrowserDateTimestamp(*value)
	if !ok {
		return false
	}
	current := time.UnixMilli(currentMs).UTC()
	sec := int64(current.Second())
	if seconds != nil {
		sec = *seconds
	}
	ms := int64(current.Nanosecond() / int(time.Millisecond))
	if milliseconds != nil {
		ms = *milliseconds
	}
	next := time.Date(
		current.Year(),
		current.Month(),
		current.Day(),
		current.Hour(),
		int(minutes),
		int(sec),
		int(ms)*int(time.Millisecond),
		time.UTC,
	).UnixMilli()
	return BrowserDateSetTimestamp(value, next)
}

func BrowserDateSetHours(value *Value, hours int64, minutes *int64, seconds *int64, milliseconds *int64) bool {
	if value == nil || value.Kind != ValueKindObject {
		return false
	}
	currentMs, ok := BrowserDateTimestamp(*value)
	if !ok {
		return false
	}
	current := time.UnixMilli(currentMs).UTC()
	min := int64(current.Minute())
	if minutes != nil {
		min = *minutes
	}
	sec := int64(current.Second())
	if seconds != nil {
		sec = *seconds
	}
	ms := int64(current.Nanosecond() / int(time.Millisecond))
	if milliseconds != nil {
		ms = *milliseconds
	}
	next := time.Date(
		current.Year(),
		current.Month(),
		current.Day(),
		int(hours),
		int(min),
		int(sec),
		int(ms)*int(time.Millisecond),
		time.UTC,
	).UnixMilli()
	return BrowserDateSetTimestamp(value, next)
}

func BrowserDateISOString(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
}

func BrowserDateUTCString(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

func BrowserDateDateString(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("Mon Jan _2 2006")
}

func BrowserDateTimeString(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("15:04:05 GMT")
}

func BrowserDateParse(text string) (int64, bool) {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return 0, false
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, normalized); err == nil {
			return t.UnixMilli(), true
		}
	}
	return 0, false
}

func BrowserDateLocaleDateString(ms int64, locale string) string {
	t := time.UnixMilli(ms).UTC()
	normalized := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(normalized, "ja") {
		return t.Format("2006/01/02")
	}
	return strconv.Itoa(int(t.Month())) + "/" + strconv.Itoa(t.Day()) + "/" + strconv.Itoa(t.Year())
}

func BrowserDateLocaleString(ms int64, locale string) string {
	t := time.UnixMilli(ms).UTC()
	normalized := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(normalized, "ja") {
		return t.Format("2006/01/02 15:04:05")
	}
	return t.Format("1/2/2006, 3:04:05 PM")
}

func BrowserDateLocaleTimeString(ms int64, locale string) string {
	t := time.UnixMilli(ms).UTC()
	normalized := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(normalized, "ja") {
		return t.Format("15:04:05")
	}
	return t.Format("3:04:05 PM")
}

func BrowserDateYear(ms int64) int {
	return time.UnixMilli(ms).UTC().Year()
}

func BrowserDateMonth(ms int64) int {
	return int(time.UnixMilli(ms).UTC().Month()) - 1
}

func BrowserDateUTCMonth(ms int64) int {
	return int(time.UnixMilli(ms).UTC().Month()) - 1
}

func BrowserDateDayOfMonth(ms int64) int {
	return time.UnixMilli(ms).UTC().Day()
}

func BrowserDateDayOfWeek(ms int64) int {
	return int(time.UnixMilli(ms).UTC().Weekday())
}

func BrowserDateHour(ms int64) int {
	return time.UnixMilli(ms).UTC().Hour()
}

func BrowserDateMinute(ms int64) int {
	return time.UnixMilli(ms).UTC().Minute()
}

func BrowserDateSecond(ms int64) int {
	return time.UnixMilli(ms).UTC().Second()
}

func BrowserDateMillisecond(ms int64) int {
	return time.UnixMilli(ms).UTC().Nanosecond() / int(time.Millisecond)
}

func BrowserDateTimezoneOffset(ms int64) int {
	return 0
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

func CompileRegExpLiteral(pattern, flags string) (*jsregex.RegexpState, error) {
	return jsregex.CompileLiteral(pattern, flags)
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
