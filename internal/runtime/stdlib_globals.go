package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func resolveStdlibReference(session *Session, store *dom.Store, path string) (script.Value, bool, error) {
	switch {
	case path == "Array":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserArrayConstructor(args)
		}), true, nil
	case strings.HasPrefix(path, "Array."):
		value, err := resolveArrayReference(session, store, strings.TrimPrefix(path, "Array."))
		return value, true, err
	case path == "Object":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectConstructor(args)
		}), true, nil
	case path == "Symbol":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserSymbolConstructor(args)
		}), true, nil
	case strings.HasPrefix(path, "Object."):
		value, err := resolveObjectReference(session, store, strings.TrimPrefix(path, "Object."))
		return value, true, err
	case path == "JSON":
		return script.HostObjectReference("JSON"), true, nil
	case strings.HasPrefix(path, "JSON."):
		value, err := resolveJSONReference(strings.TrimPrefix(path, "JSON."))
		return value, true, err
	case path == "encodeURIComponent":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserEncodeURIComponent(args)
		}), true, nil
	case path == "decodeURIComponent":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDecodeURIComponent(args)
		}), true, nil
	case path == "encodeURI":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserEncodeURI(args)
		}), true, nil
	case path == "decodeURI":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDecodeURI(args)
		}), true, nil
	case path == "Map":
		return script.BuiltinMapValue(), true, nil
	case path == "Set":
		return script.BuiltinSetValue(), true, nil
	case path == "Promise":
		return script.NativeConstructibleFunctionValue(func(args []script.Value) (script.Value, error) {
			return script.UndefinedValue(), fmt.Errorf("Promise constructor must be called with `new` in this bounded classic-JS slice")
		}, func(args []script.Value) (script.Value, error) {
			return browserPromiseConstructor(session, store, args)
		}), true, nil
	case path == "Promise.resolve":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserPromiseResolve(args)
		}), true, nil
	case path == "NaN":
		return script.NumberValue(math.NaN()), true, nil
	case path == "Infinity":
		return script.NumberValue(math.Inf(1)), true, nil
	case path == "CSS":
		return script.HostObjectReference("CSS"), true, nil
	case strings.HasPrefix(path, "CSS."):
		value, err := resolveCSSReference(strings.TrimPrefix(path, "CSS."))
		return value, true, err
	case path == "Uint8Array":
		return script.NativeConstructibleFunctionValue(func(args []script.Value) (script.Value, error) {
			return script.UndefinedValue(), fmt.Errorf("Uint8Array constructor must be called with `new` in this bounded classic-JS slice")
		}, func(args []script.Value) (script.Value, error) {
			return browserUint8ArrayConstructor(args)
		}), true, nil
	case strings.HasPrefix(path, "Uint8Array."):
		value, err := resolveUint8ArrayReference(session, store, strings.TrimPrefix(path, "Uint8Array."))
		return value, true, err
	case path == "Number":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserNumberConstructor(args)
		}), true, nil
	case path == "parseFloat":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserParseFloat(args)
		}), true, nil
	case path == "parseInt":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserParseInt(args)
		}), true, nil
	case strings.HasPrefix(path, "Number."):
		value, err := resolveNumberReference(strings.TrimPrefix(path, "Number."))
		return value, true, err
	case path == "String":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserStringConstructor(args)
		}), true, nil
	case strings.HasPrefix(path, "String."):
		value, err := resolveStringReference(strings.TrimPrefix(path, "String."))
		return value, true, err
	case path == "Boolean":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserBooleanConstructor(args)
		}), true, nil
	case path == "Math":
		return script.HostObjectReference("Math"), true, nil
	case strings.HasPrefix(path, "Math."):
		value, err := resolveMathReference(session, strings.TrimPrefix(path, "Math."))
		return value, true, err
	case path == "Date":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDateConstructor(session, args)
		}), true, nil
	case strings.HasPrefix(path, "Date."):
		value, err := resolveDateReference(session, strings.TrimPrefix(path, "Date."))
		return value, true, err
	}

	return script.UndefinedValue(), false, nil
}

func resolveArrayReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "from":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserArrayFrom(session, store, args)
		}), nil
	case "isArray":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("Array.isArray expects 1 argument")
			}
			return script.BoolValue(args[0].Kind == script.ValueKindArray), nil
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Array."+path))
}

func resolveObjectReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "prototype":
		return script.HostObjectReference("Object.prototype"), nil
	case "prototype.hasOwnProperty":
		return script.HostObjectReference("Object.prototype.hasOwnProperty"), nil
	case "prototype.hasOwnProperty.call":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectPrototypeHasOwnPropertyCall(args)
		}), nil
	case "hasOwn":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectHasOwn(args)
		}), nil
	case "assign":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectAssign(session, store, args)
		}), nil
	case "fromEntries":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectFromEntries(args)
		}), nil
	case "entries":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectEntries(args)
		}), nil
	case "getOwnPropertySymbols":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectGetOwnPropertySymbols(args)
		}), nil
	case "getOwnPropertyNames":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectGetOwnPropertyNames(args)
		}), nil
	case "keys":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectKeys(args)
		}), nil
	case "values":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectValues(args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Object."+path))
}

func browserObjectPrototypeHasOwnPropertyCall(args []script.Value) (script.Value, error) {
	if len(args) != 2 {
		return script.UndefinedValue(), fmt.Errorf("Object.prototype.hasOwnProperty.call expects 2 arguments")
	}

	key := browserPropertyKeyString(args[1])
	if args[0].Kind == script.ValueKindNull || args[0].Kind == script.ValueKindUndefined {
		return script.UndefinedValue(), fmt.Errorf("Object.prototype.hasOwnProperty.call requires an object receiver")
	}
	return script.BoolValue(browserObjectHasOwnValue(args[0], key)), nil
}

func browserObjectHasOwn(args []script.Value) (script.Value, error) {
	if len(args) != 2 {
		return script.UndefinedValue(), fmt.Errorf("Object.hasOwn expects 2 arguments")
	}
	if args[0].Kind == script.ValueKindNull || args[0].Kind == script.ValueKindUndefined {
		return script.UndefinedValue(), fmt.Errorf("Object.hasOwn requires an object receiver")
	}
	return script.BoolValue(browserObjectHasOwnValue(args[0], browserPropertyKeyString(args[1]))), nil
}

func browserObjectHasOwnValue(value script.Value, key string) bool {
	switch value.Kind {
	case script.ValueKindObject:
		return browserObjectHasOwnKey(value.Object, key)
	case script.ValueKindArray:
		if key == "length" {
			return true
		}
		if index, ok := browserArrayIndexFromString(key); ok {
			return index >= 0 && index < len(value.Array)
		}
		return false
	case script.ValueKindString:
		if key == "length" {
			return true
		}
		if index, ok := browserArrayIndexFromString(key); ok {
			return index >= 0 && index < len([]rune(value.String))
		}
		return false
	case script.ValueKindFunction:
		return key == "prototype" && script.IsConstructibleFunctionValue(value)
	default:
		return false
	}
}

func browserObjectHasOwnKey(entries []script.ObjectEntry, key string) bool {
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if script.IsInternalObjectKey(entry.Key) {
			continue
		}
		if script.IsSymbolObjectKey(key) {
			if entry.Key == key {
				return true
			}
			continue
		}
		if script.IsSymbolObjectKey(entry.Key) {
			continue
		}
		if entry.Key == key {
			return true
		}
	}
	return false
}

func browserArrayIndexFromString(key string) (int, bool) {
	if key == "" {
		return 0, false
	}
	if len(key) > 1 && key[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(key); i++ {
		if key[i] < '0' || key[i] > '9' {
			return 0, false
		}
	}
	index, err := strconv.Atoi(key)
	if err != nil || index < 0 {
		return 0, false
	}
	return index, true
}

func resolveJSONReference(path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "parse":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserJSONParse(args)
		}), nil
	case "stringify":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserJSONStringify(args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "JSON."+path))
}

func resolveNumberReference(path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "parseInt":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserParseInt(args)
		}), nil
	case "parseFloat":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserParseFloat(args)
		}), nil
	case "isInteger":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserIsInteger(args)
		}), nil
	case "isNaN":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserIsNaN(args)
		}), nil
	case "isFinite":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("Number.isFinite expects 1 argument")
			}
			if args[0].Kind != script.ValueKindNumber {
				return script.BoolValue(false), nil
			}
			return script.BoolValue(!math.IsNaN(args[0].Number) && !math.IsInf(args[0].Number, 0)), nil
		}), nil
	case "NaN":
		return script.NumberValue(math.NaN()), nil
	case "POSITIVE_INFINITY":
		return script.NumberValue(math.Inf(1)), nil
	case "NEGATIVE_INFINITY":
		return script.NumberValue(math.Inf(-1)), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Number."+path))
}

func browserParseInt(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.NumberValue(math.NaN()), nil
	}

	source := strings.TrimSpace(script.ToJSString(args[0]))
	radix := 0
	if len(args) > 1 && args[1].Kind != script.ValueKindUndefined && args[1].Kind != script.ValueKindNull {
		number, err := coerceNumber(args[1])
		if err != nil {
			return script.UndefinedValue(), err
		}
		if !math.IsNaN(number) && !math.IsInf(number, 0) {
			radix = int(math.Trunc(number))
		}
	}

	sign := 1
	if strings.HasPrefix(source, "+") {
		source = source[1:]
	} else if strings.HasPrefix(source, "-") {
		sign = -1
		source = source[1:]
	}

	if radix == 0 {
		if len(source) >= 2 && source[0] == '0' && (source[1] == 'x' || source[1] == 'X') {
			radix = 16
			source = source[2:]
		} else {
			radix = 10
		}
	}

	if radix < 2 || radix > 36 {
		return script.NumberValue(math.NaN()), nil
	}
	if radix == 16 && len(source) >= 2 && source[0] == '0' && (source[1] == 'x' || source[1] == 'X') {
		source = source[2:]
	}

	digits := browserParseIntDigits(source, radix)
	if digits == "" {
		return script.NumberValue(math.NaN()), nil
	}

	parsed := new(big.Int)
	if _, ok := parsed.SetString(digits, radix); !ok {
		return script.NumberValue(math.NaN()), nil
	}
	if sign < 0 {
		parsed.Neg(parsed)
	}
	number, _ := new(big.Float).SetInt(parsed).Float64()
	return script.NumberValue(number), nil
}

func browserParseFloat(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.NumberValue(math.NaN()), nil
	}

	source := strings.TrimSpace(script.ToJSString(args[0]))
	if source == "" {
		return script.NumberValue(math.NaN()), nil
	}

	number, err := strconv.ParseFloat(source, 64)
	if err != nil {
		return script.NumberValue(math.NaN()), nil
	}
	return script.NumberValue(number), nil
}

func browserIsInteger(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.BoolValue(false), nil
	}
	if args[0].Kind != script.ValueKindNumber {
		return script.BoolValue(false), nil
	}
	value := args[0].Number
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return script.BoolValue(false), nil
	}
	return script.BoolValue(math.Trunc(value) == value), nil
}

func browserIsNaN(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.BoolValue(false), nil
	}
	if args[0].Kind != script.ValueKindNumber {
		return script.BoolValue(false), nil
	}
	return script.BoolValue(math.IsNaN(args[0].Number)), nil
}

func browserEncodeURIComponent(args []script.Value) (script.Value, error) {
	return browserEncodeURIWithOptions(args, "")
}

func browserEncodeURI(args []script.Value) (script.Value, error) {
	return browserEncodeURIWithOptions(args, browserURIReservedCharacters)
}

func browserEncodeURIWithOptions(args []script.Value, extraUnescaped string) (script.Value, error) {
	source, err := browserToStringArg(args)
	if err != nil {
		return script.UndefinedValue(), err
	}
	var b strings.Builder
	b.Grow(len(source) * 3)
	for i := 0; i < len(source); i++ {
		ch := source[i]
		if browserIsURIUnescaped(ch, extraUnescaped) {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(browserURIComponentHex[ch>>4])
		b.WriteByte(browserURIComponentHex[ch&0x0F])
	}
	return script.StringValue(b.String()), nil
}

func browserDecodeURIComponent(args []script.Value) (script.Value, error) {
	return browserDecodeURIWithPreserve(args, "")
}

func browserDecodeURI(args []script.Value) (script.Value, error) {
	return browserDecodeURIWithPreserve(args, browserURIReservedCharacters)
}

func browserDecodeURIWithPreserve(args []script.Value, preserveEscapeSet string) (script.Value, error) {
	input, err := browserToStringArg(args)
	if err != nil {
		return script.UndefinedValue(), err
	}
	decoded, err := browserDecodeURIString(input, preserveEscapeSet)
	if err != nil {
		return script.UndefinedValue(), fmt.Errorf("URI malformed")
	}
	return script.StringValue(decoded), nil
}

func browserIsURIComponentUnescaped(ch byte) bool {
	switch {
	case ch >= 'a' && ch <= 'z':
		return true
	case ch >= 'A' && ch <= 'Z':
		return true
	case ch >= '0' && ch <= '9':
		return true
	case ch == '-' || ch == '_' || ch == '.' || ch == '!' || ch == '~' || ch == '*' || ch == '\'' || ch == '(' || ch == ')':
		return true
	default:
		return false
	}
}

func browserIsURIUnescaped(ch byte, extraUnescaped string) bool {
	if browserIsURIComponentUnescaped(ch) {
		return true
	}
	return strings.IndexByte(extraUnescaped, ch) >= 0
}

const browserURIComponentHex = "0123456789ABCDEF"
const browserURIReservedCharacters = ";/?:@&=+$,#"

func browserDecodeURIString(input string, preserveEscapeSet string) (string, error) {
	var b strings.Builder
	b.Grow(len(input))
	for i := 0; i < len(input); {
		if input[i] != '%' {
			b.WriteByte(input[i])
			i++
			continue
		}

		decoded, next, preserve, err := browserParseURIEscapedSequence(input, i, preserveEscapeSet)
		if err != nil {
			return "", err
		}
		if preserve {
			b.WriteString(input[i:next])
		} else {
			b.WriteString(string(decoded))
		}
		i = next
	}
	return b.String(), nil
}

func browserParseURIEscapedSequence(input string, start int, preserveEscapeSet string) ([]byte, int, bool, error) {
	if start+3 > len(input) || input[start] != '%' {
		return nil, start, false, fmt.Errorf("URI malformed")
	}

	first, ok := browserURIHexOctet(input[start+1 : start+3])
	if !ok {
		return nil, start, false, fmt.Errorf("URI malformed")
	}

	length := browserURIEscapedUTF8Length(first)
	if length == 0 {
		preserve := strings.IndexByte(preserveEscapeSet, first) >= 0
		return []byte{first}, start + 3, preserve, nil
	}
	if length < 0 {
		return nil, start, false, fmt.Errorf("URI malformed")
	}

	decoded := make([]byte, 1, length)
	decoded[0] = first
	next := start + 3
	for len(decoded) < length {
		if next+3 > len(input) || input[next] != '%' {
			return nil, start, false, fmt.Errorf("URI malformed")
		}
		octet, ok := browserURIHexOctet(input[next+1 : next+3])
		if !ok || octet&0xC0 != 0x80 {
			return nil, start, false, fmt.Errorf("URI malformed")
		}
		decoded = append(decoded, octet)
		next += 3
	}
	if !utf8.Valid(decoded) {
		return nil, start, false, fmt.Errorf("URI malformed")
	}

	preserve := len(decoded) == 1 && decoded[0] < utf8.RuneSelf && strings.IndexByte(preserveEscapeSet, decoded[0]) >= 0
	return decoded, next, preserve, nil
}

func browserURIEscapedUTF8Length(first byte) int {
	switch {
	case first>>7 == 0:
		return 0
	case first>>5 == 0b110:
		return 2
	case first>>4 == 0b1110:
		return 3
	case first>>3 == 0b11110:
		return 4
	default:
		return -1
	}
}

func browserURIHexOctet(twoHexDigits string) (byte, bool) {
	if len(twoHexDigits) != 2 {
		return 0, false
	}
	hi := browserParseIntDigitValue(twoHexDigits[0])
	lo := browserParseIntDigitValue(twoHexDigits[1])
	if hi < 0 || hi > 15 || lo < 0 || lo > 15 {
		return 0, false
	}
	return byte(hi<<4 | lo), true
}

func browserParseIntDigits(source string, radix int) string {
	end := 0
	for end < len(source) {
		digit := browserParseIntDigitValue(source[end])
		if digit < 0 || digit >= radix {
			break
		}
		end++
	}
	return source[:end]
}

func browserParseIntDigitValue(ch byte) int {
	switch {
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'a' && ch <= 'z':
		return int(ch-'a') + 10
	case ch >= 'A' && ch <= 'Z':
		return int(ch-'A') + 10
	default:
		return -1
	}
}

func resolveMathReference(session *Session, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "E":
		return script.NumberValue(math.E), nil
	case "LN10":
		return script.NumberValue(math.Ln10), nil
	case "LN2":
		return script.NumberValue(math.Ln2), nil
	case "LOG10E":
		return script.NumberValue(math.Log10E), nil
	case "LOG2E":
		return script.NumberValue(math.Log2E), nil
	case "PI":
		return script.NumberValue(math.Pi), nil
	case "SQRT1_2":
		return script.NumberValue(1 / math.Sqrt2), nil
	case "SQRT2":
		return script.NumberValue(math.Sqrt2), nil
	case "abs":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("Math.abs expects 1 argument")
			}
			number, err := coerceNumber(args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(math.Abs(number)), nil
		}), nil
	case "pow":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 2 {
				return script.UndefinedValue(), fmt.Errorf("Math.pow expects 2 arguments")
			}
			base, err := coerceNumber(args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			exponent, err := coerceNumber(args[1])
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(math.Pow(base, exponent)), nil
		}), nil
	case "acos":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Acos)
		}), nil
	case "acosh":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Acosh)
		}), nil
	case "asin":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Asin)
		}), nil
	case "asinh":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Asinh)
		}), nil
	case "atan":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Atan)
		}), nil
	case "atan2":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathBinary(args, math.Atan2)
		}), nil
	case "atanh":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Atanh)
		}), nil
	case "cbrt":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Cbrt)
		}), nil
	case "clz32":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			number, err := browserMathNumberArg(args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(float64(bits.LeadingZeros32(browserMathToUint32(number)))), nil
		}), nil
	case "cos":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Cos)
		}), nil
	case "cosh":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Cosh)
		}), nil
	case "exp":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Exp)
		}), nil
	case "expm1":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Expm1)
		}), nil
	case "fround":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			number, err := browserMathNumberArg(args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(float64(float32(number))), nil
		}), nil
	case "hypot":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.NumberValue(0), nil
			}
			result := 0.0
			for _, arg := range args {
				number, err := coerceNumber(arg)
				if err != nil {
					return script.UndefinedValue(), err
				}
				if math.IsInf(number, 0) {
					return script.NumberValue(math.Inf(1)), nil
				}
				if math.IsNaN(number) {
					result = math.NaN()
					continue
				}
				result = math.Hypot(result, number)
			}
			return script.NumberValue(result), nil
		}), nil
	case "imul":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			left, err := browserMathNumberArg(args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			right, err := browserMathNumberArg(args, 1)
			if err != nil {
				return script.UndefinedValue(), err
			}
			leftInt := browserMathToInt32(left)
			rightInt := browserMathToInt32(right)
			return script.NumberValue(float64(int32(uint32(leftInt) * uint32(rightInt)))), nil
		}), nil
	case "log":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Log)
		}), nil
	case "log10":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Log10)
		}), nil
	case "log1p":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Log1p)
		}), nil
	case "log2":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Log2)
		}), nil
	case "sign":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			number, err := browserMathNumberArg(args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			switch {
			case math.IsNaN(number):
				return script.NumberValue(math.NaN()), nil
			case number == 0:
				if math.Signbit(number) {
					return script.NumberValue(math.Copysign(0, -1)), nil
				}
				return script.NumberValue(0), nil
			case number > 0:
				return script.NumberValue(1), nil
			default:
				return script.NumberValue(-1), nil
			}
		}), nil
	case "sin":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Sin)
		}), nil
	case "sinh":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Sinh)
		}), nil
	case "sqrt":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Sqrt)
		}), nil
	case "tan":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Tan)
		}), nil
	case "tanh":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMathUnary(args, math.Tanh)
		}), nil
	case "ceil":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("Math.ceil expects 1 argument")
			}
			number, err := coerceNumber(args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(math.Ceil(number)), nil
		}), nil
	case "min":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.NumberValue(math.Inf(1)), nil
			}
			result := math.Inf(1)
			hasZero := false
			hasNegativeZero := false
			for _, arg := range args {
				value, err := coerceNumber(arg)
				if err != nil {
					return script.UndefinedValue(), err
				}
				if math.IsNaN(value) {
					return script.NumberValue(math.NaN()), nil
				}
				if value == 0 {
					hasZero = true
					if math.Signbit(value) {
						hasNegativeZero = true
					}
				}
				if value < result {
					result = value
				}
			}
			if result == 0 && hasZero && hasNegativeZero {
				return script.NumberValue(math.Copysign(0, -1)), nil
			}
			return script.NumberValue(result), nil
		}), nil
	case "max":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.NumberValue(math.Inf(-1)), nil
			}
			result := math.Inf(-1)
			hasZero := false
			hasPositiveZero := false
			for _, arg := range args {
				value, err := coerceNumber(arg)
				if err != nil {
					return script.UndefinedValue(), err
				}
				if math.IsNaN(value) {
					return script.NumberValue(math.NaN()), nil
				}
				if value == 0 {
					hasZero = true
					if !math.Signbit(value) {
						hasPositiveZero = true
					}
				}
				if value > result {
					result = value
				}
			}
			if result == 0 && hasZero && hasPositiveZero {
				return script.NumberValue(0), nil
			}
			return script.NumberValue(result), nil
		}), nil
	case "round":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("Math.round expects 1 argument")
			}
			number, err := coerceNumber(args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(roundTowardPositiveInfinity(number)), nil
		}), nil
	case "floor":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("Math.floor expects 1 argument")
			}
			number, err := coerceNumber(args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(math.Floor(number)), nil
		}), nil
	case "trunc":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("Math.trunc expects 1 argument")
			}
			number, err := coerceNumber(args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.NumberValue(math.Trunc(number)), nil
		}), nil
	case "random":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 0 {
				return script.UndefinedValue(), fmt.Errorf("Math.random expects no arguments")
			}
			if session == nil {
				return script.NumberValue(0), nil
			}
			return script.NumberValue(session.randomFloat64()), nil
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Math."+path))
}

func roundTowardPositiveInfinity(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value == 0 {
		return value
	}
	rounded := math.Floor(value + 0.5)
	if rounded == 0 && math.Signbit(value) {
		return math.Copysign(0, -1)
	}
	return rounded
}

func resolveDateReference(session *Session, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "now":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 0 {
				return script.UndefinedValue(), fmt.Errorf("Date.now expects no arguments")
			}
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Date.now is unavailable in this bounded classic-JS slice")
			}
			return script.NumberValue(float64(session.NowMs())), nil
		}), nil
	case "parse":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			text := script.ToJSString(script.UndefinedValue())
			if len(args) > 0 {
				text = script.ToJSString(args[0])
			}
			if ms, ok := script.BrowserDateParse(text); ok {
				return script.NumberValue(float64(ms)), nil
			}
			return script.NumberValue(math.NaN()), nil
		}), nil
	case "UTC":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDateUTC(args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Date."+path))
}

func browserArrayConstructor(args []script.Value) (script.Value, error) {
	if len(args) == 1 {
		switch args[0].Kind {
		case script.ValueKindNumber:
			if math.IsNaN(args[0].Number) || math.IsInf(args[0].Number, 0) {
				return script.UndefinedValue(), fmt.Errorf("Array length must be a finite number")
			}
			if math.Trunc(args[0].Number) != args[0].Number {
				return script.UndefinedValue(), fmt.Errorf("Array length must be an integer")
			}
			if args[0].Number < 0 {
				return script.UndefinedValue(), fmt.Errorf("Array length must be non-negative")
			}
			return script.ArrayValue(make([]script.Value, int(args[0].Number))), nil
		case script.ValueKindBigInt:
			length, err := browserInt64Value("Array", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			if length < 0 {
				return script.UndefinedValue(), fmt.Errorf("Array length must be non-negative")
			}
			return script.ArrayValue(make([]script.Value, int(length))), nil
		}
	}
	return script.ArrayValue(args), nil
}

func browserArrayFrom(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.UndefinedValue(), fmt.Errorf("Array.from expects at least 1 argument")
	}
	source := args[0]
	var elements []script.Value
	switch source.Kind {
	case script.ValueKindArray:
		elements = append([]script.Value(nil), source.Array...)
	case script.ValueKindString:
		for _, ch := range source.String {
			elements = append(elements, script.StringValue(string(ch)))
		}
	case script.ValueKindObject:
		if setEntries, ok := script.SetEntries(source); ok {
			elements = append([]script.Value(nil), setEntries...)
			break
		}
		if mapEntries, ok := script.MapEntries(source); ok {
			elements = make([]script.Value, 0, len(mapEntries))
			for _, entry := range mapEntries {
				elements = append(elements, script.ArrayValue([]script.Value{entry.Key, entry.Value}))
			}
			break
		}
		if iterElements, ok, err := browserArrayFromIteratorValues(session, store, source, func(host *inlineScriptHost) (script.Value, bool) {
			nextValue, ok := objectProperty(source, "next")
			if !ok {
				return script.Value{}, false
			}
			return nextValue, true
		}); err != nil {
			return script.UndefinedValue(), err
		} else if ok {
			elements = iterElements
			break
		}
		lengthValue, ok := objectProperty(source, "length")
		if !ok {
			return script.UndefinedValue(), fmt.Errorf("Array.from expects array-like object with length")
		}
		length, err := browserInt64Value("Array.from", lengthValue)
		if err != nil {
			return script.UndefinedValue(), err
		}
		if length < 0 {
			return script.UndefinedValue(), fmt.Errorf("Array.from length must be non-negative")
		}
		for i := int64(0); i < length; i++ {
			key := strconv.FormatInt(i, 10)
			if value, ok := objectProperty(source, key); ok {
				elements = append(elements, value)
			} else {
				elements = append(elements, script.UndefinedValue())
			}
		}
	case script.ValueKindHostReference:
		host := &inlineScriptHost{session: session, store: store}
		lengthValue, err := host.ResolveHostReference(browserJoinHostReferencePath(source.HostReferencePath, "length"))
		if err == nil {
			length, err := browserInt64Value("Array.from", lengthValue)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if length < 0 {
				return script.UndefinedValue(), fmt.Errorf("Array.from length must be non-negative")
			}
			elements = make([]script.Value, 0, int(length))
			for i := int64(0); i < length; i++ {
				value, err := host.ResolveHostReference(browserJoinHostReferencePath(source.HostReferencePath, strconv.FormatInt(i, 10)))
				if err != nil {
					return script.UndefinedValue(), err
				}
				elements = append(elements, value)
			}
			break
		}
		if iterElements, ok, err := browserArrayFromIteratorValues(session, store, source, func(host *inlineScriptHost) (script.Value, bool) {
			nextValue, err := host.ResolveHostReference(browserJoinHostReferencePath(source.HostReferencePath, "next"))
			if err != nil {
				return script.Value{}, false
			}
			return nextValue, true
		}); err != nil {
			return script.UndefinedValue(), err
		} else if ok {
			elements = iterElements
			break
		}
		return script.UndefinedValue(), fmt.Errorf("Array.from expects array-like host object with length")
	default:
		return script.UndefinedValue(), fmt.Errorf("Array.from expects array, string, or array-like object")
	}

	if len(args) > 1 && args[1].Kind != script.ValueKindUndefined {
		mapper := args[1]
		thisArg := script.UndefinedValue()
		hasReceiver := false
		if len(args) > 2 {
			thisArg = args[2]
			hasReceiver = true
		}
		host := &inlineScriptHost{session: session, store: store}
		mapped := make([]script.Value, 0, len(elements))
		for i, element := range elements {
			value, err := script.InvokeCallableValue(host, mapper, []script.Value{
				element,
				script.NumberValue(float64(i)),
				script.ArrayValue(elements),
			}, thisArg, hasReceiver)
			if err != nil {
				return script.UndefinedValue(), err
			}
			mapped = append(mapped, value)
		}
		elements = mapped
	}

	return script.ArrayValue(elements), nil
}

func browserArrayFromIteratorValues(
	session *Session,
	store *dom.Store,
	source script.Value,
	resolveNext func(*inlineScriptHost) (script.Value, bool),
) ([]script.Value, bool, error) {
	host := &inlineScriptHost{session: session, store: store}
	nextValue, ok := resolveNext(host)
	if !ok {
		return nil, false, nil
	}
	if nextValue.Kind != script.ValueKindFunction && nextValue.Kind != script.ValueKindHostReference {
		return nil, false, nil
	}

	values := make([]script.Value, 0)
	for {
		result, err := script.InvokeCallableValue(host, nextValue, nil, source, true)
		if err != nil {
			return nil, false, err
		}
		if result.Kind != script.ValueKindObject {
			return nil, false, fmt.Errorf("Array.from iterator must return an object in this bounded classic-JS slice")
		}
		doneValue, ok := objectProperty(result, "done")
		if !ok || doneValue.Kind != script.ValueKindBool {
			return nil, false, fmt.Errorf("Array.from iterator result must include a boolean `done` property in this bounded classic-JS slice")
		}
		if doneValue.Bool {
			break
		}
		value, ok := objectProperty(result, "value")
		if !ok {
			value = script.UndefinedValue()
		}
		values = append(values, value)
	}

	return values, true, nil
}

func resolveUint8ArrayReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "from":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserUint8ArrayFrom(session, store, args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Uint8Array."+path))
}

func browserUint8ArrayFrom(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	arrayValue, err := browserArrayFrom(session, store, args)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return browserUint8ArrayConstructor([]script.Value{arrayValue})
}

func browserJoinHostReferencePath(base, name string) string {
	base = strings.TrimSpace(base)
	name = strings.TrimSpace(strings.TrimPrefix(name, "."))
	if base == "" {
		return name
	}
	if name == "" {
		return base
	}
	return base + "." + name
}

func browserObjectConstructor(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.ObjectValue(nil), nil
	}
	switch args[0].Kind {
	case script.ValueKindObject:
		return args[0], nil
	}
	return script.ObjectValue(nil), nil
}

func browserPromiseConstructor(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.UndefinedValue(), fmt.Errorf("Promise constructor expects an executor")
	}
	executor := args[0]
	if executor.Kind != script.ValueKindFunction {
		return script.UndefinedValue(), fmt.Errorf("Promise executor must be callable")
	}

	promiseValue, resolvePromise, rejectPromise := script.NewPendingPromiseWithReject()
	resolveFn := script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		value := script.UndefinedValue()
		if len(args) > 0 {
			value = args[0]
		}
		resolvePromise(value)
		return script.UndefinedValue(), nil
	})
	rejectFn := script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		value := script.UndefinedValue()
		if len(args) > 0 {
			value = args[0]
		}
		rejectPromise(value)
		return script.UndefinedValue(), nil
	})
	host := &inlineScriptHost{session: session, store: store}
	if _, err := script.InvokeCallableValue(host, executor, []script.Value{resolveFn, rejectFn}, script.UndefinedValue(), false); err != nil {
		if throwValue, ok := script.ThrowValueFromError(err); ok {
			rejectPromise(throwValue)
		} else {
			rejectPromise(script.StringValue(err.Error()))
		}
	}
	return promiseValue, nil
}

func browserPromiseResolve(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.PromiseValue(script.UndefinedValue()), nil
	}
	if args[0].Kind == script.ValueKindPromise {
		return args[0], nil
	}
	return script.PromiseValue(args[0]), nil
}

func browserSymbolConstructor(args []script.Value) (script.Value, error) {
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("Symbol expects at most 1 argument")
	}
	description := ""
	if len(args) == 1 {
		description = script.ToJSString(args[0])
	}
	return script.SymbolValue(description), nil
}

func browserObjectAssign(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.UndefinedValue(), fmt.Errorf("Object.assign expects at least 1 argument")
	}
	host := &inlineScriptHost{session: session, store: store}
	return script.ObjectAssign(host, args[0], args[1:]...)
}

func browserObjectAssignSourceValue(value script.Value) (script.Value, bool, error) {
	switch value.Kind {
	case script.ValueKindUndefined, script.ValueKindNull:
		return script.UndefinedValue(), false, nil
	case script.ValueKindString:
		entries := make([]script.ObjectEntry, 0, len(value.String))
		for i, ch := range value.String {
			entries = append(entries, script.ObjectEntry{
				Key:   strconv.Itoa(i),
				Value: script.StringValue(string(ch)),
			})
		}
		return script.ObjectValue(entries), true, nil
	case script.ValueKindArray:
		entries := make([]script.ObjectEntry, 0, len(value.Array))
		for i, element := range value.Array {
			entries = append(entries, script.ObjectEntry{
				Key:   strconv.Itoa(i),
				Value: element,
			})
		}
		return script.ObjectValue(entries), true, nil
	default:
		return value, true, nil
	}
}

func browserObjectKeys(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.keys expects 1 argument")
	}
	value, ok, err := browserObjectAssignSourceValue(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("Cannot convert undefined or null to object")
	}
	keys := uniqueObjectKeys(value.Object)
	entries := make([]script.Value, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, script.StringValue(key))
	}
	return script.ArrayValue(entries), nil
}

func browserObjectEntries(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.entries expects 1 argument")
	}
	objectValue, ok, err := browserObjectAssignSourceValue(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("Cannot convert undefined or null to object")
	}
	keys := uniqueObjectKeys(objectValue.Object)
	entries := make([]script.Value, 0, len(keys))
	for _, key := range keys {
		entryValue, _ := objectProperty(objectValue, key)
		entries = append(entries, script.ArrayValue([]script.Value{
			script.StringValue(key),
			entryValue,
		}))
	}
	return script.ArrayValue(entries), nil
}

func browserObjectFromEntries(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.fromEntries expects 1 argument")
	}
	entries, ok, err := browserObjectFromEntriesSource(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("Object.fromEntries expects an array of key/value pairs or a Map")
	}
	return script.ObjectValue(entries), nil
}

func browserObjectFromEntriesSource(value script.Value) ([]script.ObjectEntry, bool, error) {
	switch value.Kind {
	case script.ValueKindArray:
		entries := make([]script.ObjectEntry, 0, len(value.Array))
		for i, pair := range value.Array {
			if pair.Kind != script.ValueKindArray || len(pair.Array) < 2 {
				return nil, false, fmt.Errorf("Object.fromEntries pair %d must be a two-item array in this bounded slice", i)
			}
			entries = append(entries, script.ObjectEntry{
				Key:   browserPropertyKeyString(pair.Array[0]),
				Value: pair.Array[1],
			})
		}
		return entries, true, nil
	case script.ValueKindObject:
		if value.MapState == nil {
			return nil, false, nil
		}
		mapEntries, ok := script.MapEntries(value)
		if !ok {
			return nil, false, nil
		}
		entries := make([]script.ObjectEntry, 0, len(mapEntries))
		for _, entry := range mapEntries {
			entries = append(entries, script.ObjectEntry{
				Key:   browserPropertyKeyString(entry.Key),
				Value: entry.Value,
			})
		}
		return entries, true, nil
	case script.ValueKindUndefined, script.ValueKindNull:
		return nil, false, nil
	default:
		return nil, false, nil
	}
}

func browserObjectValues(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.values expects 1 argument")
	}
	objectValue, ok, err := browserObjectAssignSourceValue(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("Cannot convert undefined or null to object")
	}
	keys := uniqueObjectKeys(objectValue.Object)
	entries := make([]script.Value, 0, len(keys))
	for _, key := range keys {
		entryValue, _ := objectProperty(objectValue, key)
		entries = append(entries, entryValue)
	}
	return script.ArrayValue(entries), nil
}

func browserPropertyKeyString(value script.Value) string {
	if key, ok := script.SymbolObjectKey(value); ok {
		return key
	}
	return script.ToJSString(value)
}

func browserObjectGetOwnPropertySymbols(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.getOwnPropertySymbols expects 1 argument")
	}
	value, ok, err := browserObjectAssignSourceValue(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("Cannot convert undefined or null to object")
	}
	symbols := make([]script.Value, 0)
	seen := make(map[string]struct{})
	for _, entry := range value.Object {
		if !script.IsSymbolObjectKey(entry.Key) {
			continue
		}
		symbol, ok := script.SymbolValueFromObjectKey(entry.Key)
		if !ok {
			continue
		}
		if _, ok := seen[symbol.SymbolID]; ok {
			continue
		}
		seen[symbol.SymbolID] = struct{}{}
		symbols = append(symbols, symbol)
	}
	return script.ArrayValue(symbols), nil
}

func browserObjectGetOwnPropertyNames(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.getOwnPropertyNames expects 1 argument")
	}
	source := args[0]
	_, ok, err := browserObjectAssignSourceValue(source)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("Cannot convert undefined or null to object")
	}
	names := make([]script.Value, 0)
	seen := make(map[string]struct{})
	appendName := func(name string) {
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, script.StringValue(name))
	}
	switch source.Kind {
	case script.ValueKindArray:
		for i := range source.Array {
			appendName(strconv.Itoa(i))
		}
		appendName("length")
		for _, key := range uniqueObjectKeys(source.Object) {
			appendName(key)
		}
	case script.ValueKindString:
		for i := range []rune(source.String) {
			appendName(strconv.Itoa(i))
		}
		appendName("length")
		for _, key := range uniqueObjectKeys(source.Object) {
			appendName(key)
		}
	case script.ValueKindFunction:
		for _, key := range uniqueObjectKeys(source.Object) {
			appendName(key)
		}
		if script.IsConstructibleFunctionValue(source) {
			appendName("prototype")
		}
	default:
		for _, key := range uniqueObjectKeys(source.Object) {
			appendName(key)
		}
	}
	return script.ArrayValue(names), nil
}

func browserJSONParse(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("JSON.parse expects 1 argument")
	}
	input := script.ToJSString(args[0])
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	value, err := browserJSONParseValue(decoder)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.UndefinedValue(), fmt.Errorf("unexpected trailing JSON token %v", token)
	}
	return value, nil
}

func browserJSONStringify(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.UndefinedValue(), nil
	}
	if len(args) > 3 {
		return script.UndefinedValue(), fmt.Errorf("JSON.stringify expects at most 3 arguments in this bounded slice")
	}
	if len(args) >= 2 && args[1].Kind != script.ValueKindUndefined && args[1].Kind != script.ValueKindNull {
		return script.UndefinedValue(), fmt.Errorf("JSON.stringify replacer is unavailable in this bounded slice")
	}
	indent, err := browserJSONStringifySpace(args)
	if err != nil {
		return script.UndefinedValue(), err
	}
	text, err := jsonStringifyValueWithIndent(args[0], indent, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.StringValue(text), nil
}

func browserNumberConstructor(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.NumberValue(0), nil
	}
	number, err := coerceNumber(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.NumberValue(number), nil
}

func browserStringConstructor(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.StringValue(""), nil
	}
	return script.StringValue(script.ToJSString(args[0])), nil
}

func resolveStringReference(path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "fromCharCode":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserStringFromCharCode(args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "String."+path))
}

func browserStringFromCharCode(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.StringValue(""), nil
	}
	var b strings.Builder
	for _, arg := range args {
		unit, err := browserUint16Value(arg)
		if err != nil {
			return script.UndefinedValue(), err
		}
		b.WriteRune(rune(unit))
	}
	return script.StringValue(b.String()), nil
}

func browserBooleanConstructor(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.BoolValue(false), nil
	}
	return script.BoolValue(jsTruthyValue(args[0])), nil
}

func browserDateConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Date is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("Date constructor expects 0 or 1 argument")
	}
	ms := session.NowMs()
	if len(args) == 1 {
		if args[0].Kind == script.ValueKindString {
			parsed, ok := script.BrowserDateParse(args[0].String)
			if !ok {
				return script.UndefinedValue(), fmt.Errorf("Date constructor requires a parsable date string")
			}
			ms = parsed
		} else {
			value, err := coerceNumber(args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return script.UndefinedValue(), fmt.Errorf("Date constructor requires a finite timestamp")
			}
			ms = int64(value)
		}
	}
	return browserDateValue(ms), nil
}

func browserDateUTC(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.NumberValue(math.NaN()), nil
	}

	year, err := browserInt64Value("Date.UTC", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	month := int64(0)
	if len(args) > 1 {
		month, err = browserInt64Value("Date.UTC", args[1])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	day := int64(1)
	if len(args) > 2 {
		day, err = browserInt64Value("Date.UTC", args[2])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	hour := int64(0)
	if len(args) > 3 {
		hour, err = browserInt64Value("Date.UTC", args[3])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	minute := int64(0)
	if len(args) > 4 {
		minute, err = browserInt64Value("Date.UTC", args[4])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	second := int64(0)
	if len(args) > 5 {
		second, err = browserInt64Value("Date.UTC", args[5])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	millisecond := int64(0)
	if len(args) > 6 {
		millisecond, err = browserInt64Value("Date.UTC", args[6])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}

	if 0 <= year && year <= 99 {
		year += 1900
	}

	t := time.Date(
		int(year),
		time.Month(month)+1,
		int(day),
		int(hour),
		int(minute),
		int(second),
		int(millisecond)*int(time.Millisecond),
		time.UTC,
	)
	return script.NumberValue(float64(t.UnixMilli())), nil
}

func browserDateValue(ms int64) script.Value {
	return script.BrowserDateValue(ms)
}

func dateObjectMs(value script.Value) (int64, bool) {
	return script.BrowserDateTimestamp(value)
}

func coerceNumber(value script.Value) (float64, error) {
	switch value.Kind {
	case script.ValueKindUndefined:
		return math.NaN(), nil
	case script.ValueKindNull:
		return 0, nil
	case script.ValueKindBool:
		if value.Bool {
			return 1, nil
		}
		return 0, nil
	case script.ValueKindNumber:
		return value.Number, nil
	case script.ValueKindBigInt:
		parsed, err := strconv.ParseFloat(value.BigInt, 64)
		if err != nil {
			return math.NaN(), nil
		}
		return parsed, nil
	case script.ValueKindString:
		trimmed := strings.TrimSpace(value.String)
		if trimmed == "" {
			return 0, nil
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return math.NaN(), nil
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("argument must be a primitive number in this bounded slice")
	}
}

func browserMathNumberArg(args []script.Value, index int) (float64, error) {
	if index < len(args) {
		return coerceNumber(args[index])
	}
	return coerceNumber(script.UndefinedValue())
}

func browserMathUnary(args []script.Value, fn func(float64) float64) (script.Value, error) {
	number, err := browserMathNumberArg(args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.NumberValue(fn(number)), nil
}

func browserMathBinary(args []script.Value, fn func(float64, float64) float64) (script.Value, error) {
	left, err := browserMathNumberArg(args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	right, err := browserMathNumberArg(args, 1)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.NumberValue(fn(left, right)), nil
}

func browserMathToUint32(value float64) uint32 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value == 0 {
		return 0
	}
	truncated := math.Trunc(value)
	truncated = math.Mod(truncated, 4294967296)
	if truncated < 0 {
		truncated += 4294967296
	}
	return uint32(truncated)
}

func browserMathToInt32(value float64) int32 {
	return int32(browserMathToUint32(value))
}

func browserUint16Value(value script.Value) (uint16, error) {
	number, err := coerceNumber(value)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number == 0 {
		return 0, nil
	}
	truncated := math.Trunc(number)
	truncated = math.Mod(truncated, 65536)
	if truncated < 0 {
		truncated += 65536
	}
	return uint16(truncated), nil
}

func jsTruthyValue(value script.Value) bool {
	switch value.Kind {
	case script.ValueKindUndefined, script.ValueKindNull:
		return false
	case script.ValueKindBool:
		return value.Bool
	case script.ValueKindNumber:
		return value.Number != 0 && !math.IsNaN(value.Number)
	case script.ValueKindBigInt:
		return value.BigInt != "0"
	case script.ValueKindString:
		return value.String != ""
	default:
		return true
	}
}

func browserJSONParseValue(decoder *json.Decoder) (script.Value, error) {
	token, err := decoder.Token()
	if err != nil {
		return script.UndefinedValue(), err
	}

	switch typed := token.(type) {
	case json.Delim:
		switch typed {
		case '{':
			return browserJSONParseObject(decoder)
		case '[':
			return browserJSONParseArray(decoder)
		default:
			return script.UndefinedValue(), fmt.Errorf("unexpected JSON delimiter %q", typed)
		}
	case nil:
		return script.NullValue(), nil
	case bool:
		return script.BoolValue(typed), nil
	case string:
		return script.StringValue(typed), nil
	case json.Number:
		number, err := typed.Float64()
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.NumberValue(number), nil
	default:
		return script.UndefinedValue(), fmt.Errorf("unsupported JSON token type %T", token)
	}
}

func browserJSONParseObject(decoder *json.Decoder) (script.Value, error) {
	entries := make([]script.ObjectEntry, 0)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return script.UndefinedValue(), err
		}
		key, ok := keyToken.(string)
		if !ok {
			return script.UndefinedValue(), fmt.Errorf("JSON object key must be a string")
		}
		value, err := browserJSONParseValue(decoder)
		if err != nil {
			return script.UndefinedValue(), err
		}
		entries = append(entries, script.ObjectEntry{Key: key, Value: value})
	}

	endToken, err := decoder.Token()
	if err != nil {
		return script.UndefinedValue(), err
	}
	if delim, ok := endToken.(json.Delim); !ok || delim != '}' {
		return script.UndefinedValue(), fmt.Errorf("unexpected JSON object terminator %v", endToken)
	}
	return script.ObjectValue(entries), nil
}

func browserJSONParseArray(decoder *json.Decoder) (script.Value, error) {
	values := make([]script.Value, 0)
	for decoder.More() {
		value, err := browserJSONParseValue(decoder)
		if err != nil {
			return script.UndefinedValue(), err
		}
		values = append(values, value)
	}

	endToken, err := decoder.Token()
	if err != nil {
		return script.UndefinedValue(), err
	}
	if delim, ok := endToken.(json.Delim); !ok || delim != ']' {
		return script.UndefinedValue(), fmt.Errorf("unexpected JSON array terminator %v", endToken)
	}
	return script.ArrayValue(values), nil
}

func browserJSONStringifySpace(args []script.Value) (string, error) {
	if len(args) < 3 {
		return "", nil
	}
	space := args[2]
	switch space.Kind {
	case script.ValueKindUndefined, script.ValueKindNull:
		return "", nil
	case script.ValueKindNumber:
		count := 0
		switch {
		case math.IsNaN(space.Number), math.IsInf(space.Number, 0):
			count = 0
		default:
			count = int(space.Number)
		}
		if count < 0 {
			count = 0
		}
		if count > 10 {
			count = 10
		}
		return strings.Repeat(" ", count), nil
	case script.ValueKindString:
		runes := []rune(space.String)
		if len(runes) > 10 {
			runes = runes[:10]
		}
		return string(runes), nil
	default:
		return "", fmt.Errorf("JSON.stringify space argument must be a number or string in this bounded slice")
	}
}

func jsonStringifyValue(value script.Value) (string, error) {
	return jsonStringifyValueWithIndent(value, "", 0)
}

func jsonStringifyValueWithIndent(value script.Value, indentUnit string, depth int) (string, error) {
	if ms, ok := script.BrowserDateTimestamp(value); ok {
		encoded, err := json.Marshal(script.BrowserDateISOString(ms))
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
	switch value.Kind {
	case script.ValueKindNull:
		return "null", nil
	case script.ValueKindBool:
		if value.Bool {
			return "true", nil
		}
		return "false", nil
	case script.ValueKindNumber:
		if math.IsNaN(value.Number) || math.IsInf(value.Number, 0) {
			return "null", nil
		}
		return strconv.FormatFloat(value.Number, 'f', -1, 64), nil
	case script.ValueKindString:
		encoded, err := json.Marshal(value.String)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	case script.ValueKindArray:
		parts := make([]string, 0, len(value.Array))
		for _, entry := range value.Array {
			encoded, err := jsonStringifyValueWithIndent(entry, indentUnit, depth+1)
			if err != nil {
				return "", err
			}
			parts = append(parts, encoded)
		}
		if indentUnit == "" {
			return "[" + strings.Join(parts, ",") + "]", nil
		}
		if len(parts) == 0 {
			return "[]", nil
		}
		childIndent := strings.Repeat(indentUnit, depth+1)
		currentIndent := strings.Repeat(indentUnit, depth)
		return "[\n" + childIndent + strings.Join(parts, ",\n"+childIndent) + "\n" + currentIndent + "]", nil
	case script.ValueKindObject:
		keys := uniqueObjectKeys(value.Object)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return "", err
			}
			encodedValue, err := jsonStringifyValueWithIndent(objectValueByKey(value.Object, key), indentUnit, depth+1)
			if err != nil {
				return "", err
			}
			if indentUnit == "" {
				parts = append(parts, string(encodedKey)+":"+encodedValue)
			} else {
				parts = append(parts, string(encodedKey)+": "+encodedValue)
			}
		}
		if indentUnit == "" {
			return "{" + strings.Join(parts, ",") + "}", nil
		}
		if len(parts) == 0 {
			return "{}", nil
		}
		childIndent := strings.Repeat(indentUnit, depth+1)
		currentIndent := strings.Repeat(indentUnit, depth)
		return "{\n" + childIndent + strings.Join(parts, ",\n"+childIndent) + "\n" + currentIndent + "}", nil
	case script.ValueKindUndefined:
		return "", fmt.Errorf("JSON.stringify does not support undefined in this bounded slice")
	case script.ValueKindFunction, script.ValueKindHostReference, script.ValueKindPromise, script.ValueKindInvocation, script.ValueKindPrivateName, script.ValueKindBigInt:
		return "", fmt.Errorf("JSON.stringify does not support %s values in this bounded slice", value.Kind)
	default:
		return "", fmt.Errorf("JSON.stringify does not support %s values in this bounded slice", value.Kind)
	}
}

func uniqueObjectKeys(entries []script.ObjectEntry) []string {
	seen := make(map[string]struct{}, len(entries))
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		if script.IsInternalObjectKey(entry.Key) || script.IsSymbolObjectKey(entry.Key) {
			continue
		}
		if _, ok := seen[entry.Key]; ok {
			continue
		}
		seen[entry.Key] = struct{}{}
		keys = append(keys, entry.Key)
	}
	return keys
}

func objectValueByKey(entries []script.ObjectEntry, key string) script.Value {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Key == key {
			return entries[i].Value
		}
	}
	return script.UndefinedValue()
}

func findObjectEntryIndex(entries []script.ObjectEntry, key string) int {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Key == key {
			return i
		}
	}
	return -1
}
