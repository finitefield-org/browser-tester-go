package script

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

func currentBindingUpdateContextReplaceObjectBindings(oldValue Value, newValue Value) int {
	if ctx := CurrentBindingUpdateContext(); ctx != nil {
		return ctx.ReplaceObjectBindings(oldValue, newValue)
	}
	return 0
}

func currentBindingUpdateContextReplaceArrayBindings(oldValue Value, newValue Value) int {
	if ctx := CurrentBindingUpdateContext(); ctx != nil {
		return ctx.ReplaceArrayBindings(oldValue, newValue)
	}
	return 0
}

func (p *classicJSStatementParser) resolveArrayPrototypeMethod(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindArray {
		return UndefinedValue(), false, nil
	}
	switch name {
	case "push":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			updated := append([]Value(nil), value.Array...)
			updated = append(updated, args...)
			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return NumberValue(float64(len(updated))), nil
		}), true, nil
	case "includes":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return BoolValue(false), nil
			}
			start := 0
			if len(args) > 1 {
				start = indexFromValue(args[1], 0)
			}
			length := len(value.Array)
			if start < 0 {
				start = length + start
				if start < 0 {
					start = 0
				}
			}
			for i := start; i < length; i++ {
				if sameValueZero(value.Array[i], args[0]) {
					return BoolValue(true), nil
				}
			}
			return BoolValue(false), nil
		}), true, nil
	case "indexOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			search := UndefinedValue()
			if len(args) > 0 {
				search = args[0]
			}
			length := len(value.Array)
			start := indexFromValueOrDefault(args, 1, 0)
			if start < 0 {
				start = length + start
				if start < 0 {
					start = 0
				}
			}
			if start > length {
				start = length
			}
			for i := start; i < length; i++ {
				if classicJSSameValue(value.Array[i], search) {
					return NumberValue(float64(i)), nil
				}
			}
			return NumberValue(-1), nil
		}), true, nil
	case "lastIndexOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			search := UndefinedValue()
			if len(args) > 0 {
				search = args[0]
			}
			length := len(value.Array)
			if length == 0 {
				return NumberValue(-1), nil
			}
			start := indexFromValueOrDefault(args, 1, length-1)
			if start < 0 {
				start = length + start
				if start < 0 {
					return NumberValue(-1), nil
				}
			}
			if start >= length {
				start = length - 1
			}
			for i := start; i >= 0; i-- {
				if classicJSSameValue(value.Array[i], search) {
					return NumberValue(float64(i)), nil
				}
			}
			return NumberValue(-1), nil
		}), true, nil
	case "filter":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.filter expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			var filtered []Value
			for i, element := range value.Array {
				result, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				if jsTruthy(result) {
					filtered = append(filtered, element)
				}
			}
			return ArrayValue(filtered), nil
		}), true, nil
	case "forEach":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.forEach expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			for i, element := range value.Array {
				if _, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver); err != nil {
					return UndefinedValue(), err
				}
			}
			return UndefinedValue(), nil
		}), true, nil
	case "map":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.map expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			mapped := make([]Value, 0, len(value.Array))
			for i, element := range value.Array {
				result, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				mapped = append(mapped, result)
			}
			return ArrayValue(mapped), nil
		}), true, nil
	case "flatMap":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.flatMap expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			flattened := make([]Value, 0, len(value.Array))
			for i, element := range value.Array {
				result, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				if result.Kind == ValueKindArray {
					flattened = append(flattened, result.Array...)
					continue
				}
				flattened = append(flattened, result)
			}
			return ArrayValue(flattened), nil
		}), true, nil
	case "flat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			depth := indexFromValueOrDefault(args, 0, 1)
			if depth < 0 {
				depth = 0
			}
			flattened := make([]Value, 0, len(value.Array))
			flattenArrayValues(value.Array, depth, &flattened)
			return ArrayValue(flattened), nil
		}), true, nil
	case "some":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.some expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			for i, element := range value.Array {
				result, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				if jsTruthy(result) {
					return BoolValue(true), nil
				}
			}
			return BoolValue(false), nil
		}), true, nil
	case "every":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.every expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			for i, element := range value.Array {
				result, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				if !jsTruthy(result) {
					return BoolValue(false), nil
				}
			}
			return BoolValue(true), nil
		}), true, nil
	case "find":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.find expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			for i, element := range value.Array {
				result, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				if jsTruthy(result) {
					return element, nil
				}
			}
			return UndefinedValue(), nil
		}), true, nil
	case "findIndex":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.findIndex expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			for i, element := range value.Array {
				result, err := InvokeCallableValue(p.host, callback, []Value{
					element,
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				if jsTruthy(result) {
					return NumberValue(float64(i)), nil
				}
			}
			return NumberValue(-1), nil
		}), true, nil
	case "splice":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.splice expects at least 1 argument")
			}

			length := len(value.Array)
			start := indexFromValue(args[0], 0)
			if start < 0 {
				start = length + start
			}
			if start < 0 {
				start = 0
			}
			if start > length {
				start = length
			}

			deleteCount := length - start
			if len(args) > 1 && args[1].Kind != ValueKindUndefined {
				deleteCount = indexFromValue(args[1], 0)
				if deleteCount < 0 {
					deleteCount = 0
				}
				if deleteCount > length-start {
					deleteCount = length - start
				}
			}

			removed := append([]Value(nil), value.Array[start:start+deleteCount]...)
			insert := args[2:]

			updated := make([]Value, 0, length-deleteCount+len(insert))
			updated = append(updated, value.Array[:start]...)
			updated = append(updated, insert...)
			updated = append(updated, value.Array[start+deleteCount:]...)

			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return ArrayValue(removed), nil
		}), true, nil
	case "fill":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.fill expects a value")
			}

			length := len(value.Array)
			start := indexFromValueOrDefault(args, 1, 0)
			end := indexFromValueOrDefault(args, 2, length)
			start = clampSliceIndex(start, length)
			end = clampSliceIndex(end, length)
			if end < start {
				end = start
			}

			updated := append([]Value(nil), value.Array...)
			for i := start; i < end; i++ {
				updated[i] = args[0]
			}

			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return updatedValue, nil
		}), true, nil
	case "sort":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			updated := append([]Value(nil), value.Array...)
			compareCallback := UndefinedValue()
			useComparator := false
			if len(args) > 0 && args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
				compareCallback = args[0]
				useComparator = true
			}

			compareValues := func(left, right Value) (int, error) {
				if useComparator {
					result, err := InvokeCallableValue(p.host, compareCallback, []Value{left, right}, UndefinedValue(), false)
					if err != nil {
						return 0, err
					}
					number, ok := classicJSNumberValue(result)
					if !ok {
						return 0, fmt.Errorf("Array.sort comparator must return a number")
					}
					switch {
					case math.IsNaN(number), number == 0:
						return 0, nil
					case number < 0:
						return -1, nil
					default:
						return 1, nil
					}
				}
				leftText := ToJSString(left)
				rightText := ToJSString(right)
				switch {
				case leftText < rightText:
					return -1, nil
				case leftText > rightText:
					return 1, nil
				default:
					return 0, nil
				}
			}

			for i := 1; i < len(updated); i++ {
				for j := i; j > 0; j-- {
					cmp, err := compareValues(updated[j-1], updated[j])
					if err != nil {
						return UndefinedValue(), err
					}
					if cmp <= 0 {
						break
					}
					updated[j-1], updated[j] = updated[j], updated[j-1]
				}
			}

			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return updatedValue, nil
		}), true, nil
	case "unshift":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return NumberValue(float64(len(value.Array))), nil
			}
			updated := make([]Value, 0, len(args)+len(value.Array))
			updated = append(updated, args...)
			updated = append(updated, value.Array...)
			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return NumberValue(float64(len(updated))), nil
		}), true, nil
	case "slice":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			length := len(value.Array)
			start := indexFromValueOrDefault(args, 0, 0)
			end := indexFromValueOrDefault(args, 1, length)
			start = clampSliceIndex(start, length)
			end = clampSliceIndex(end, length)
			if end < start {
				end = start
			}
			return ArrayValue(value.Array[start:end]), nil
		}), true, nil
	case "concat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			result := make([]Value, 0, len(value.Array))
			result = append(result, value.Array...)
			for _, arg := range args {
				if arg.Kind == ValueKindArray {
					result = append(result, arg.Array...)
				} else {
					result = append(result, arg)
				}
			}
			return ArrayValue(result), nil
		}), true, nil
	case "join":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			sep := ","
			if len(args) > 0 && args[0].Kind != ValueKindUndefined {
				sep = ToJSString(args[0])
			}
			var b strings.Builder
			for i, element := range value.Array {
				if i > 0 {
					b.WriteString(sep)
				}
				b.WriteString(arrayElementString(element))
			}
			return StringValue(b.String()), nil
		}), true, nil
	case "toString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			var b strings.Builder
			for i, element := range value.Array {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteString(arrayElementString(element))
			}
			return StringValue(b.String()), nil
		}), true, nil
	case "valueOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return ArrayValue(value.Array), nil
		}), true, nil
	}
	return UndefinedValue(), false, nil
}

func (p *classicJSStatementParser) resolveStringPrototypeMethod(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindString {
		return UndefinedValue(), false, nil
	}
	switch name {
	case "trim":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(strings.TrimSpace(value.String)), nil
		}), true, nil
	case "toLowerCase":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(strings.ToLower(value.String)), nil
		}), true, nil
	case "replace":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) < 2 {
				return UndefinedValue(), fmt.Errorf("String.replace expects 2 arguments")
			}
			if args[1].Kind == ValueKindFunction || args[1].Kind == ValueKindHostReference {
				if compiled, flags, ok, err := classicJSRegExpValue(args[0]); ok || err != nil {
					if err != nil {
						return UndefinedValue(), err
					}
					updated, err := replaceRegexpWithCallback(p.host, compiled, value.String, args[1], strings.Contains(flags, "g"))
					if err != nil {
						return UndefinedValue(), err
					}
					return StringValue(updated), nil
				}
				updated, err := replaceStringWithCallback(p.host, value.String, ToJSString(args[0]), args[1])
				if err != nil {
					return UndefinedValue(), err
				}
				return StringValue(updated), nil
			}
			replacement := ToJSString(args[1])
			if compiled, flags, ok, err := classicJSRegExpValue(args[0]); ok || err != nil {
				if err != nil {
					return UndefinedValue(), err
				}
				if strings.Contains(flags, "g") {
					return StringValue(compiled.ReplaceAllString(value.String, replacement)), nil
				}
				return StringValue(replaceFirstRegexp(compiled, value.String, replacement)), nil
			}
			search := ToJSString(args[0])
			return StringValue(strings.Replace(value.String, search, replacement, 1)), nil
		}), true, nil
	case "charCodeAt":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			index := 0
			if len(args) > 0 {
				index = indexFromValue(args[0], 0)
			}
			runes := []rune(value.String)
			if index < 0 || index >= len(runes) {
				return NumberValue(math.NaN()), nil
			}
			return NumberValue(float64(runes[index])), nil
		}), true, nil
	case "split":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 || args[0].Kind == ValueKindUndefined {
				return ArrayValue([]Value{StringValue(value.String)}), nil
			}
			limit := -1
			if len(args) > 1 && args[1].Kind != ValueKindUndefined {
				limit = indexFromValue(args[1], -1)
				if limit < 0 {
					limit = -1
				}
			}
			if compiled, _, ok, err := classicJSRegExpValue(args[0]); ok || err != nil {
				if err != nil {
					return UndefinedValue(), err
				}
				parts := compiled.Split(value.String, limit)
				out := make([]Value, 0, len(parts))
				for _, part := range parts {
					out = append(out, StringValue(part))
				}
				return ArrayValue(out), nil
			}
			separator := ToJSString(args[0])
			var parts []string
			if limit < 0 {
				parts = strings.Split(value.String, separator)
			} else {
				parts = strings.SplitN(value.String, separator, limit)
			}
			out := make([]Value, 0, len(parts))
			for _, part := range parts {
				out = append(out, StringValue(part))
			}
			return ArrayValue(out), nil
		}), true, nil
	case "match":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.match expects 1 argument")
			}
			if compiled, flags, ok, err := classicJSRegExpValue(args[0]); ok || err != nil {
				if err != nil {
					return UndefinedValue(), err
				}
				if strings.Contains(flags, "g") {
					matches := compiled.FindAllString(value.String, -1)
					if matches == nil {
						return NullValue(), nil
					}
					out := make([]Value, 0, len(matches))
					for _, match := range matches {
						out = append(out, StringValue(match))
					}
					return ArrayValue(out), nil
				}
				matches := compiled.FindStringSubmatch(value.String)
				if matches == nil {
					return NullValue(), nil
				}
				out := make([]Value, 0, len(matches))
				for _, match := range matches {
					out = append(out, StringValue(match))
				}
				return ArrayValue(out), nil
			}
			search := ToJSString(args[0])
			if strings.Index(value.String, search) == -1 {
				return NullValue(), nil
			}
			return ArrayValue([]Value{StringValue(search)}), nil
		}), true, nil
	case "lastIndexOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.lastIndexOf expects 1 argument")
			}
			search := ToJSString(args[0])
			length := len(value.String)
			fromIndex := length
			if len(args) > 1 {
				fromIndex = indexFromValue(args[1], length)
			}
			if fromIndex < 0 {
				return NumberValue(-1), nil
			}
			if fromIndex > length {
				fromIndex = length
			}
			if search == "" {
				return NumberValue(float64(fromIndex)), nil
			}
			limit := fromIndex + len(search)
			if limit > length {
				limit = length
			}
			idx := strings.LastIndex(value.String[:limit], search)
			return NumberValue(float64(idx)), nil
		}), true, nil
	case "indexOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.indexOf expects 1 argument")
			}
			search := ToJSString(args[0])
			length := len(value.String)
			fromIndex := 0
			if len(args) > 1 {
				fromIndex = indexFromValue(args[1], 0)
			}
			if fromIndex < 0 {
				fromIndex = 0
			}
			if fromIndex > length {
				fromIndex = length
			}
			if search == "" {
				return NumberValue(float64(fromIndex)), nil
			}
			idx := strings.Index(value.String[fromIndex:], search)
			if idx == -1 {
				return NumberValue(-1), nil
			}
			return NumberValue(float64(fromIndex + idx)), nil
		}), true, nil
	case "startsWith":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.startsWith expects 1 argument")
			}
			if len(args) > 2 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.startsWith accepts at most 2 arguments")
			}
			search := ToJSString(args[0])
			fromIndex := 0
			if len(args) > 1 {
				fromIndex = indexFromValue(args[1], 0)
			}
			if fromIndex < 0 {
				fromIndex = 0
			}
			if fromIndex > len(value.String) {
				return BoolValue(false), nil
			}
			if search == "" {
				return BoolValue(true), nil
			}
			return BoolValue(strings.HasPrefix(value.String[fromIndex:], search)), nil
		}), true, nil
	case "endsWith":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.endsWith expects 1 argument")
			}
			if len(args) > 2 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.endsWith accepts at most 2 arguments")
			}
			search := ToJSString(args[0])
			end := len(value.String)
			if len(args) > 1 {
				end = indexFromValue(args[1], end)
			}
			if end < 0 {
				end = 0
			}
			if end > len(value.String) {
				end = len(value.String)
			}
			if search == "" {
				return BoolValue(true), nil
			}
			if len(search) > end {
				return BoolValue(false), nil
			}
			return BoolValue(strings.HasSuffix(value.String[:end], search)), nil
		}), true, nil
	case "includes":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.includes expects 1 argument")
			}
			search := ToJSString(args[0])
			start := 0
			if len(args) > 1 {
				start = indexFromValue(args[1], 0)
			}
			if start < 0 {
				start = 0
			}
			if start > len(value.String) {
				return BoolValue(false), nil
			}
			return BoolValue(strings.Contains(value.String[start:], search)), nil
		}), true, nil
	case "padStart":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.padStart expects 1 argument")
			}
			targetLength := indexFromValue(args[0], len(value.String))
			if targetLength <= len(value.String) {
				return StringValue(value.String), nil
			}
			fill := " "
			if len(args) > 1 && args[1].Kind != ValueKindUndefined {
				fill = ToJSString(args[1])
			}
			if fill == "" {
				return StringValue(value.String), nil
			}
			needed := targetLength - len(value.String)
			var prefix strings.Builder
			for prefix.Len() < needed {
				prefix.WriteString(fill)
			}
			return StringValue(prefix.String()[:needed] + value.String), nil
		}), true, nil
	case "slice":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			runes := []rune(value.String)
			length := len(runes)
			start := indexFromValueOrDefault(args, 0, 0)
			end := indexFromValueOrDefault(args, 1, length)
			start = clampSliceIndex(start, length)
			end = clampSliceIndex(end, length)
			if end < start {
				end = start
			}
			return StringValue(string(runes[start:end])), nil
		}), true, nil
	case "toString", "valueOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(value.String), nil
		}), true, nil
	}
	return UndefinedValue(), false, nil
}

func (p *classicJSStatementParser) resolveBoolPrototypeMethod(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindBool {
		return UndefinedValue(), false, nil
	}
	switch name {
	case "toString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if value.Bool {
				return StringValue("true"), nil
			}
			return StringValue("false"), nil
		}), true, nil
	case "valueOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return BoolValue(value.Bool), nil
		}), true, nil
	}
	return UndefinedValue(), false, nil
}

func (p *classicJSStatementParser) resolveSymbolPrototypeMethod(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindSymbol {
		return UndefinedValue(), false, nil
	}
	switch name {
	case "toString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(ToJSString(value)), nil
		}), true, nil
	case "valueOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return value, nil
		}), true, nil
	}
	return UndefinedValue(), false, nil
}

func (p *classicJSStatementParser) resolveNumberPrototypeMethod(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindNumber {
		return UndefinedValue(), false, nil
	}
	switch name {
	case "toString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			radix := 10
			if len(args) > 0 && args[0].Kind != ValueKindUndefined {
				converted, ok := classicJSNumberValue(args[0])
				if !ok || math.IsNaN(converted) {
					return UndefinedValue(), fmt.Errorf("Number.toString radix must be numeric")
				}
				radix = int(math.Trunc(converted))
			}
			text, err := numberToStringRadix(value.Number, radix)
			if err != nil {
				return UndefinedValue(), err
			}
			return StringValue(text), nil
		}), true, nil
	case "toFixed":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			fractionDigits := 0
			if len(args) > 0 && args[0].Kind != ValueKindUndefined {
				converted, ok := classicJSNumberValue(args[0])
				if !ok || math.IsNaN(converted) || math.IsInf(converted, 0) {
					return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toFixed expects a finite numeric fractionDigits")
				}
				fractionDigits = int(math.Trunc(converted))
				if fractionDigits < 0 || fractionDigits > 100 {
					return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toFixed fractionDigits must be between 0 and 100")
				}
			}
			text := numberToFixedString(value.Number, fractionDigits)
			return StringValue(text), nil
		}), true, nil
	case "valueOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(value.Number), nil
		}), true, nil
	case "toPrecision":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 || args[0].Kind == ValueKindUndefined {
				return StringValue(ToJSString(value)), nil
			}
			precision, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(precision) || math.IsInf(precision, 0) {
				return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toPrecision expects a finite numeric precision")
			}
			pInt := int(math.Trunc(precision))
			if pInt < 1 || pInt > 100 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toPrecision precision must be between 1 and 100")
			}
			text, err := numberToPrecisionString(value.Number, pInt)
			if err != nil {
				return UndefinedValue(), err
			}
			return StringValue(text), nil
		}), true, nil
	case "toExponential":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			fractionDigits := -1
			if len(args) > 0 && args[0].Kind != ValueKindUndefined {
				converted, ok := classicJSNumberValue(args[0])
				if !ok || math.IsNaN(converted) || math.IsInf(converted, 0) {
					return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toExponential expects a finite numeric fractionDigits")
				}
				fractionDigits = int(math.Trunc(converted))
				if fractionDigits < 0 || fractionDigits > 100 {
					return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toExponential fractionDigits must be between 0 and 100")
				}
			}
			text := numberToExponentialString(value.Number, fractionDigits)
			return StringValue(text), nil
		}), true, nil
	}
	return UndefinedValue(), false, nil
}

func (p *classicJSStatementParser) resolveDatePrototypeMethod(value Value, name string) (Value, bool, error) {
	ms, ok := BrowserDateTimestamp(value)
	if !ok {
		return UndefinedValue(), false, nil
	}
	switch name {
	case "toISOString", "toJSON", "toString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(BrowserDateISOString(ms)), nil
		}), true, nil
	case "toLocaleDateString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			locale := "en-US"
			if len(args) > 0 && args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
				locale = strings.TrimSpace(ToJSString(args[0]))
			}
			return StringValue(BrowserDateLocaleDateString(ms, locale)), nil
		}), true, nil
	case "valueOf", "getTime":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(ms)), nil
		}), true, nil
	}
	return UndefinedValue(), false, nil
}

func (p *classicJSStatementParser) resolvePromisePrototypeMethod(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindPromise {
		return UndefinedValue(), false, nil
	}

	resolved := unwrapPromiseValue(value)
	pending, isPending := pendingPromiseState(value)

	switch name {
	case "then":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) > 2 {
				return UndefinedValue(), fmt.Errorf("Promise.then accepts at most 2 arguments")
			}
			if !isPending || pending == nil {
				if len(args) == 0 || args[0].Kind == ValueKindUndefined || args[0].Kind == ValueKindNull {
					return PromiseValue(resolved), nil
				}
				result, err := InvokeCallableValue(p.host, args[0], []Value{resolved}, UndefinedValue(), false)
				if err != nil {
					return UndefinedValue(), err
				}
				return PromiseValue(unwrapPromiseValue(result)), nil
			}

			resultPromise := &classicJSPromiseState{}
			fulfill := func(resolvedValue Value) {
				if len(args) == 0 || args[0].Kind == ValueKindUndefined || args[0].Kind == ValueKindNull {
					resultPromise.resolve(unwrapPromiseValue(resolvedValue))
					return
				}
				result, err := InvokeCallableValue(p.host, args[0], []Value{unwrapPromiseValue(resolvedValue)}, UndefinedValue(), false)
				if err != nil {
					resultPromise.resolve(UndefinedValue())
					return
				}
				if chainedPromise, ok := pendingPromiseState(result); ok {
					chainedPromise.addWaiter(func(next Value) {
						resultPromise.resolve(unwrapPromiseValue(next))
					})
					return
				}
				resultPromise.resolve(unwrapPromiseValue(result))
			}
			pending.addWaiter(func(resolvedValue Value) {
				fulfill(resolvedValue)
			})
			return PendingPromiseValue(resultPromise), nil
		}), true, nil
	case "catch":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) > 1 {
				return UndefinedValue(), fmt.Errorf("Promise.catch accepts at most 1 argument")
			}
			if !isPending || pending == nil {
				return PromiseValue(resolved), nil
			}
			resultPromise := &classicJSPromiseState{}
			pending.addWaiter(func(resolvedValue Value) {
				resultPromise.resolve(unwrapPromiseValue(resolvedValue))
			})
			return PendingPromiseValue(resultPromise), nil
		}), true, nil
	}

	return UndefinedValue(), false, nil
}

func callbackReceiver(args []Value) (Value, bool) {
	if len(args) > 1 {
		return args[1], true
	}
	return UndefinedValue(), false
}

func indexFromValue(value Value, fallback int) int {
	number, ok := classicJSNumberValue(value)
	if !ok || math.IsNaN(number) {
		return fallback
	}
	return int(math.Trunc(number))
}

func indexFromValueOrDefault(args []Value, index int, fallback int) int {
	if len(args) <= index {
		return fallback
	}
	if args[index].Kind == ValueKindUndefined {
		return fallback
	}
	return indexFromValue(args[index], fallback)
}

func clampSliceIndex(index int, length int) int {
	if index < 0 {
		index = length + index
	}
	if index < 0 {
		index = 0
	}
	if index > length {
		index = length
	}
	return index
}

func sameValueZero(left Value, right Value) bool {
	if left.Kind == ValueKindNumber && right.Kind == ValueKindNumber {
		if math.IsNaN(left.Number) && math.IsNaN(right.Number) {
			return true
		}
		if left.Number == 0 && right.Number == 0 {
			return true
		}
	}
	return classicJSSameValue(left, right)
}

func arrayElementString(value Value) string {
	switch value.Kind {
	case ValueKindUndefined, ValueKindNull:
		return ""
	default:
		return ToJSString(value)
	}
}

func flattenArrayValues(values []Value, depth int, out *[]Value) {
	for _, element := range values {
		if element.Kind == ValueKindUndefined {
			continue
		}
		if depth > 0 && element.Kind == ValueKindArray {
			flattenArrayValues(element.Array, depth-1, out)
			continue
		}
		*out = append(*out, element)
	}
}

func classicJSRegExpValue(value Value) (*regexp.Regexp, string, bool, error) {
	if value.Kind != ValueKindObject {
		return nil, "", false, nil
	}
	patternValue, ok := lookupObjectProperty(value.Object, classicJSRegExpPatternKey)
	if !ok || patternValue.Kind != ValueKindString {
		return nil, "", false, nil
	}
	flagsValue, ok := lookupObjectProperty(value.Object, classicJSRegExpFlagsKey)
	if !ok || flagsValue.Kind != ValueKindString {
		return nil, "", false, nil
	}
	compiled, err := classicJSCompileRegExpLiteral(patternValue.String, flagsValue.String)
	if err != nil {
		return nil, "", true, err
	}
	return compiled, flagsValue.String, true, nil
}

func replaceFirstRegexp(compiled *regexp.Regexp, input string, replacement string) string {
	loc := compiled.FindStringIndex(input)
	if loc == nil {
		return input
	}
	return input[:loc[0]] + replacement + input[loc[1]:]
}

func replaceStringWithCallback(host HostBindings, input, search string, replacer Value) (string, error) {
	index := strings.Index(input, search)
	if index < 0 {
		return input, nil
	}
	replacement, err := invokeStringReplaceCallback(host, replacer, input, search, index, index+len(search), nil)
	if err != nil {
		return "", err
	}
	return input[:index] + replacement + input[index+len(search):], nil
}

func replaceRegexpWithCallback(host HostBindings, compiled *regexp.Regexp, input string, replacer Value, global bool) (string, error) {
	matches := compiled.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input, nil
	}
	if !global {
		matches = matches[:1]
	}
	var b strings.Builder
	last := 0
	for _, loc := range matches {
		if len(loc) < 2 {
			continue
		}
		start, end := loc[0], loc[1]
		if start < 0 || end < 0 || start < last {
			continue
		}
		b.WriteString(input[last:start])
		replacement, err := invokeStringReplaceCallback(host, replacer, input, input[start:end], start, end, loc)
		if err != nil {
			return "", err
		}
		b.WriteString(replacement)
		last = end
	}
	b.WriteString(input[last:])
	return b.String(), nil
}

func invokeStringReplaceCallback(host HostBindings, replacer Value, input, match string, start, end int, matchLoc []int) (string, error) {
	args := make([]Value, 0, len(matchLoc)/2+3)
	args = append(args, StringValue(match))
	if len(matchLoc) > 0 {
		for i := 2; i+1 < len(matchLoc); i += 2 {
			if matchLoc[i] < 0 || matchLoc[i+1] < 0 {
				args = append(args, UndefinedValue())
				continue
			}
			args = append(args, StringValue(input[matchLoc[i]:matchLoc[i+1]]))
		}
	}
	args = append(args, NumberValue(float64(start)), StringValue(input))
	result, err := InvokeCallableValue(host, replacer, args, UndefinedValue(), false)
	if err != nil {
		return "", err
	}
	return ToJSString(result), nil
}

func numberToStringRadix(number float64, radix int) (string, error) {
	if radix == 0 {
		radix = 10
	}
	if radix < 2 || radix > 36 {
		return "", fmt.Errorf("Number.toString radix must be between 2 and 36")
	}
	if math.IsNaN(number) {
		return "NaN", nil
	}
	if math.IsInf(number, 1) {
		return "Infinity", nil
	}
	if math.IsInf(number, -1) {
		return "-Infinity", nil
	}
	if radix == 10 {
		if number == 0 {
			return "0", nil
		}
		return strconv.FormatFloat(number, 'f', -1, 64), nil
	}
	if number == 0 {
		return "0", nil
	}
	sign := ""
	if number < 0 {
		sign = "-"
		number = -number
	}
	intPart := math.Floor(number)
	fracPart := number - intPart
	intString := radixIntString(int64(intPart), radix)
	if fracPart == 0 {
		return sign + intString, nil
	}
	var fracBuilder strings.Builder
	for i := 0; i < 16 && fracPart > 0; i++ {
		fracPart *= float64(radix)
		digit := int(fracPart)
		fracPart -= float64(digit)
		fracBuilder.WriteByte(radixDigit(digit))
	}
	return sign + intString + "." + fracBuilder.String(), nil
}

func radixIntString(value int64, radix int) string {
	if value == 0 {
		return "0"
	}
	var digits [64]byte
	pos := len(digits)
	for value > 0 {
		digit := int(value % int64(radix))
		pos--
		digits[pos] = radixDigit(digit)
		value /= int64(radix)
	}
	return string(digits[pos:])
}

func radixDigit(value int) byte {
	if value < 10 {
		return byte('0' + value)
	}
	return byte('a' + (value - 10))
}

func numberToExponentialString(number float64, fractionDigits int) string {
	switch {
	case math.IsNaN(number):
		return "NaN"
	case math.IsInf(number, 1):
		return "Infinity"
	case math.IsInf(number, -1):
		return "-Infinity"
	}
	prec := fractionDigits
	if prec < 0 {
		prec = -1
	}
	text := strconv.FormatFloat(number, 'e', prec, 64)
	return normalizeExponentDigits(text)
}

func numberToFixedString(number float64, fractionDigits int) string {
	switch {
	case math.IsNaN(number):
		return "NaN"
	case math.IsInf(number, 1):
		return "Infinity"
	case math.IsInf(number, -1):
		return "-Infinity"
	}
	if number == 0 {
		number = 0
	}
	if math.Abs(number) >= 1e21 {
		return normalizeExponentDigits(strconv.FormatFloat(number, 'g', -1, 64))
	}
	return strconv.FormatFloat(number, 'f', fractionDigits, 64)
}

func numberToPrecisionString(number float64, precision int) (string, error) {
	switch {
	case math.IsNaN(number):
		return "NaN", nil
	case math.IsInf(number, 1):
		return "Infinity", nil
	case math.IsInf(number, -1):
		return "-Infinity", nil
	}
	if precision < 1 || precision > 100 {
		return "", NewError(ErrorKindRuntime, "Number.toPrecision precision must be between 1 and 100")
	}

	scientific := strconv.FormatFloat(number, 'e', precision-1, 64)
	sign := ""
	if strings.HasPrefix(scientific, "-") {
		sign = "-"
		scientific = scientific[1:]
	}
	ePos := strings.IndexByte(scientific, 'e')
	if ePos < 0 {
		return sign + scientific, nil
	}

	mantissa := scientific[:ePos]
	exponentText := scientific[ePos+1:]
	exp, err := strconv.Atoi(exponentText)
	if err != nil {
		return "", fmt.Errorf("Number.toPrecision unexpected exponent %q", exponentText)
	}

	// Mantissa is always 1 digit before '.', followed by precision-1 digits.
	digits := strings.ReplaceAll(mantissa, ".", "")
	if len(digits) != precision {
		// Defensive guard: keep the behavior explicit instead of silently producing a wrong string.
		return "", fmt.Errorf("Number.toPrecision expected %d digits, got %d", precision, len(digits))
	}

	if exp < -6 || exp >= precision {
		return sign + normalizeExponentDigits(mantissa+"e"+exponentText), nil
	}

	decimalPos := exp + 1
	if decimalPos >= precision {
		return sign + digits + strings.Repeat("0", decimalPos-precision), nil
	}
	if decimalPos > 0 {
		return sign + digits[:decimalPos] + "." + digits[decimalPos:], nil
	}
	return sign + "0." + strings.Repeat("0", -decimalPos) + digits, nil
}

func normalizeExponentDigits(text string) string {
	ePos := strings.IndexByte(text, 'e')
	if ePos < 0 || ePos+2 >= len(text) {
		return text
	}
	sign := text[ePos+1]
	if sign != '+' && sign != '-' {
		return text
	}
	expDigits := text[ePos+2:]
	trimmed := strings.TrimLeft(expDigits, "0")
	if trimmed == "" {
		trimmed = "0"
	}
	return text[:ePos+2] + trimmed
}
