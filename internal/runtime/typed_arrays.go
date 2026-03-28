package runtime

import (
	"fmt"
	"math"
	"strconv"

	"browsertester/internal/script"
)

func browserUint8ArrayConstructor(args []script.Value) (script.Value, error) {
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("Uint8Array constructor accepts at most 1 argument")
	}
	if len(args) == 0 {
		return browserUint8ArrayValue(nil), nil
	}
	bytes, err := browserUint8ArrayBytesFromValue(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	return browserUint8ArrayValue(bytes), nil
}

func browserUint8ArrayValue(bytes []byte) script.Value {
	elements := make([]script.ObjectEntry, 0, len(bytes)+3)
	for i, b := range bytes {
		elements = append(elements, script.ObjectEntry{
			Key:   strconv.Itoa(i),
			Value: script.NumberValue(float64(b)),
		})
	}
	elements = append(elements,
		script.ObjectEntry{Key: "length", Value: script.NumberValue(float64(len(bytes)))},
		script.ObjectEntry{Key: "byteLength", Value: script.NumberValue(float64(len(bytes)))},
		script.ObjectEntry{Key: "buffer", Value: browserUint8ArrayBufferValue(bytes)},
		script.ObjectEntry{Key: script.BrowserUint8ArrayBytesKey, Value: script.ArrayValue(browserUint8ArrayBytesToValues(bytes))},
	)
	return script.ObjectValue(elements)
}

func browserUint8ArrayBufferValue(bytes []byte) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "byteLength", Value: script.NumberValue(float64(len(bytes)))},
		{Key: script.BrowserUint8ArrayBytesKey, Value: script.ArrayValue(browserUint8ArrayBytesToValues(bytes))},
	})
}

func browserUint8ArrayBytesToValues(bytes []byte) []script.Value {
	out := make([]script.Value, len(bytes))
	for i, b := range bytes {
		out[i] = script.NumberValue(float64(b))
	}
	return out
}

func browserUint8ArrayBytesFromValue(value script.Value) ([]byte, error) {
	switch value.Kind {
	case script.ValueKindArray:
		out := make([]byte, len(value.Array))
		for i, element := range value.Array {
			byteValue, err := browserUint8ArrayByteFromValue(element)
			if err != nil {
				return nil, err
			}
			out[i] = byteValue
		}
		return out, nil
	case script.ValueKindObject:
		if bytesValue, ok := objectProperty(value, script.BrowserUint8ArrayBytesKey); ok {
			return browserUint8ArrayBytesFromValue(bytesValue)
		}
		lengthValue, ok := objectProperty(value, "length")
		if !ok {
			return nil, fmt.Errorf("Uint8Array expects array-like input or buffer object")
		}
		length, err := browserInt64Value("Uint8Array", lengthValue)
		if err != nil {
			return nil, err
		}
		if length < 0 {
			return nil, fmt.Errorf("Uint8Array length must be non-negative")
		}
		out := make([]byte, length)
		for i := int64(0); i < length; i++ {
			key := strconv.FormatInt(i, 10)
			element, ok := objectProperty(value, key)
			if !ok {
				continue
			}
			byteValue, err := browserUint8ArrayByteFromValue(element)
			if err != nil {
				return nil, err
			}
			out[i] = byteValue
		}
		return out, nil
	case script.ValueKindHostReference:
		return nil, fmt.Errorf("Uint8Array expects array-like input or buffer object")
	case script.ValueKindString:
		out := make([]byte, 0, len(value.String))
		for _, ch := range value.String {
			if ch > math.MaxUint8 {
				return nil, fmt.Errorf("Uint8Array string input contains code points outside byte range")
			}
			out = append(out, byte(ch))
		}
		return out, nil
	default:
		length, err := browserInt64Value("Uint8Array", value)
		if err != nil {
			return nil, fmt.Errorf("Uint8Array expects array-like input or buffer object")
		}
		if length < 0 {
			return nil, fmt.Errorf("Uint8Array length must be non-negative")
		}
		return make([]byte, length), nil
	}
}

func browserUint8ArrayByteFromValue(value script.Value) (byte, error) {
	switch value.Kind {
	case script.ValueKindNumber:
		if math.IsNaN(value.Number) || math.IsInf(value.Number, 0) {
			return 0, fmt.Errorf("Uint8Array byte values must be finite numbers")
		}
		return byte(uint8(int64(value.Number))), nil
	case script.ValueKindBigInt:
		n, err := browserInt64Value("Uint8Array", value)
		if err != nil {
			return 0, err
		}
		return byte(uint8(n)), nil
	case script.ValueKindBool:
		if value.Bool {
			return 1, nil
		}
		return 0, nil
	case script.ValueKindUndefined, script.ValueKindNull:
		return 0, nil
	default:
		return 0, fmt.Errorf("Uint8Array byte values must be numbers")
	}
}
