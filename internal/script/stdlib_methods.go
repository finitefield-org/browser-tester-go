package script

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"browsertester/internal/collation"
	"golang.org/x/text/unicode/norm"
)

func (p *classicJSStatementParser) invokeCallableValue(callee Value, args []Value, receiver Value, hasReceiver bool) (Value, error) {
	if p == nil {
		return InvokeCallableValue(nil, callee, args, receiver, hasReceiver)
	}
	invoker := *p
	invoker.env = newClassicJSEnvironment()
	invoker.resumeState = nil
	invoker.generatorNextValue = UndefinedValue()
	invoker.hasGeneratorNextValue = false
	invoker.bindingUpdateParent = p
	callable := scalarJSValue(callee)
	if hasReceiver {
		callable.receiver = receiver
		callable.hasReceiver = true
	}
	result, err := invoker.invoke(callable, args)
	if err != nil {
		return UndefinedValue(), err
	}
	if result.kind != jsValueScalar {
		return UndefinedValue(), NewError(ErrorKindRuntime, "callable did not return a scalar value in this bounded classic-JS slice")
	}
	return result.value, nil
}

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
	case "pop":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(value.Array) == 0 {
				return UndefinedValue(), nil
			}
			updated := append([]Value(nil), value.Array[:len(value.Array)-1]...)
			removed := value.Array[len(value.Array)-1]
			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return removed, nil
		}), true, nil
	case "shift":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(value.Array) == 0 {
				return UndefinedValue(), nil
			}
			removed := value.Array[0]
			updated := append([]Value(nil), value.Array[1:]...)
			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return removed, nil
		}), true, nil
	case "includes":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return BoolValue(false), nil
			}
			elements := append([]Value(nil), value.Array...)
			start := 0
			if len(args) > 1 {
				start = indexFromValue(args[1], 0)
			}
			length := len(elements)
			if start < 0 {
				start = length + start
				if start < 0 {
					start = 0
				}
			}
			for i := start; i < length; i++ {
				if sameValueZero(elements[i], args[0]) {
					return BoolValue(true), nil
				}
			}
			return BoolValue(false), nil
		}), true, nil
	case "at":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			index, ok, err := browserAtIndex(len(value.Array), args, true)
			if err != nil {
				return UndefinedValue(), err
			}
			if !ok {
				return UndefinedValue(), nil
			}
			return value.Array[index], nil
		}), true, nil
	case "indexOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			search := UndefinedValue()
			if len(args) > 0 {
				search = args[0]
			}
			elements := append([]Value(nil), value.Array...)
			length := len(elements)
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
				if classicJSSameValue(elements[i], search) {
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
			elements := append([]Value(nil), value.Array...)
			length := len(elements)
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
				if classicJSSameValue(elements[i], search) {
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
			elements := append([]Value(nil), value.Array...)
			var filtered []Value
			for i, element := range elements {
				result, err := p.invokeCallableValue(callback, []Value{
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
			elements := append([]Value(nil), value.Array...)
			for i, element := range elements {
				if _, err := p.invokeCallableValue(callback, []Value{
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
			elements := append([]Value(nil), value.Array...)
			mapped := make([]Value, 0, len(elements))
			for i, element := range elements {
				result, err := p.invokeCallableValue(callback, []Value{
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
			elements := append([]Value(nil), value.Array...)
			flattened := make([]Value, 0, len(elements))
			for i, element := range elements {
				result, err := p.invokeCallableValue(callback, []Value{
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
	case "toLocaleString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			text, err := arrayLocaleStringText(p, value.Array, args)
			if err != nil {
				return UndefinedValue(), err
			}
			return StringValue(text), nil
		}), true, nil
	case "flat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			depth := indexFromValueOrDefault(args, 0, 1)
			if depth < 0 {
				depth = 0
			}
			elements := append([]Value(nil), value.Array...)
			flattened := make([]Value, 0, len(elements))
			flattenArrayValues(elements, depth, &flattened)
			return ArrayValue(flattened), nil
		}), true, nil
	case "some":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.some expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			elements := append([]Value(nil), value.Array...)
			for i, element := range elements {
				result, err := p.invokeCallableValue(callback, []Value{
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
			elements := append([]Value(nil), value.Array...)
			for i, element := range elements {
				result, err := p.invokeCallableValue(callback, []Value{
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
	case "reduce":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.reduce expects a callback")
			}
			initial, hasInitial := UndefinedValue(), false
			if len(args) > 1 {
				initial = args[1]
				hasInitial = true
			}
			elements := append([]Value(nil), value.Array...)
			return browserArrayReduce(p, elements, args[0], initial, hasInitial, value, false, "Array.reduce")
		}), true, nil
	case "reduceRight":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.reduceRight expects a callback")
			}
			initial, hasInitial := UndefinedValue(), false
			if len(args) > 1 {
				initial = args[1]
				hasInitial = true
			}
			elements := append([]Value(nil), value.Array...)
			return browserArrayReduce(p, elements, args[0], initial, hasInitial, value, true, "Array.reduceRight")
		}), true, nil
	case "find":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Array.find expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			elements := append([]Value(nil), value.Array...)
			for i, element := range elements {
				result, err := p.invokeCallableValue(callback, []Value{
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
			elements := append([]Value(nil), value.Array...)
			for i, element := range elements {
				result, err := p.invokeCallableValue(callback, []Value{
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
	case "findLast":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "Array.findLast expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			elements := append([]Value(nil), value.Array...)
			for i := len(elements) - 1; i >= 0; i-- {
				result, err := p.invokeCallableValue(callback, []Value{
					elements[i],
					NumberValue(float64(i)),
					value,
				}, thisArg, hasReceiver)
				if err != nil {
					return UndefinedValue(), err
				}
				if jsTruthy(result) {
					return elements[i], nil
				}
			}
			return UndefinedValue(), nil
		}), true, nil
	case "findLastIndex":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "Array.findLastIndex expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			elements := append([]Value(nil), value.Array...)
			for i := len(elements) - 1; i >= 0; i-- {
				result, err := p.invokeCallableValue(callback, []Value{
					elements[i],
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
			var insert []Value
			if len(args) > 2 {
				insert = args[2:]
			}

			updated := make([]Value, 0, length-deleteCount+len(insert))
			updated = append(updated, value.Array[:start]...)
			updated = append(updated, insert...)
			updated = append(updated, value.Array[start+deleteCount:]...)

			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return ArrayValue(removed), nil
		}), true, nil
	case "reverse":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			// Reverse a copy, then update any bindings that point at the original array.
			updated := append([]Value(nil), value.Array...)
			for i, j := 0, len(updated)-1; i < j; i, j = i+1, j-1 {
				updated[i], updated[j] = updated[j], updated[i]
			}
			updatedValue := ArrayValue(updated)
			currentBindingUpdateContextReplaceArrayBindings(value, updatedValue)
			return updatedValue, nil
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
	case "copyWithin":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			length := len(value.Array)
			target := indexFromValueOrDefault(args, 0, 0)
			start := indexFromValueOrDefault(args, 1, 0)
			end := indexFromValueOrDefault(args, 2, length)
			target = clampSliceIndex(target, length)
			start = clampSliceIndex(start, length)
			end = clampSliceIndex(end, length)
			if start > end {
				start = end
			}
			count := end - start
			if remaining := length - target; count > remaining {
				count = remaining
			}
			updated := append([]Value(nil), value.Array...)
			if count > 0 {
				copy(updated[target:target+count], updated[start:end])
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
					result, err := p.invokeCallableValue(compareCallback, []Value{left, right}, UndefinedValue(), false)
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
	case "charAt":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			runes := []rune(value.String)
			index, ok, err := browserAtIndex(len(runes), args, false)
			if err != nil {
				return UndefinedValue(), err
			}
			if !ok {
				return StringValue(""), nil
			}
			return StringValue(string(runes[index])), nil
		}), true, nil
	case "trimStart":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(strings.TrimLeftFunc(value.String, unicode.IsSpace)), nil
		}), true, nil
	case "trimEnd":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(strings.TrimRightFunc(value.String, unicode.IsSpace)), nil
		}), true, nil
	case "normalize":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			normalized, err := browserStringNormalize(value.String, args)
			if err != nil {
				return UndefinedValue(), err
			}
			return StringValue(normalized), nil
		}), true, nil
	case "toLowerCase":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(strings.ToLower(value.String)), nil
		}), true, nil
	case "toUpperCase":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(strings.ToUpper(value.String)), nil
		}), true, nil
	case "concat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			var b strings.Builder
			b.Grow(len(value.String))
			b.WriteString(value.String)
			for _, arg := range args {
				b.WriteString(ToJSString(arg))
			}
			return StringValue(b.String()), nil
		}), true, nil
	case "localeCompare":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			target := "undefined"
			if len(args) > 0 {
				target = ToJSString(args[0])
			}
			locale := "en-US"
			if len(args) > 1 && args[1].Kind != ValueKindUndefined && args[1].Kind != ValueKindNull {
				locale = strings.TrimSpace(ToJSString(args[1]))
			}
			numeric := false
			if len(args) > 2 && args[2].Kind != ValueKindUndefined && args[2].Kind != ValueKindNull {
				if args[2].Kind != ValueKindObject {
					return UndefinedValue(), NewError(ErrorKindRuntime, "String.localeCompare options argument must be an object")
				}
				if option, ok := lookupObjectProperty(args[2].Object, "numeric"); ok {
					if option.Kind != ValueKindBool {
						return UndefinedValue(), NewError(ErrorKindRuntime, "String.localeCompare numeric must be a boolean")
					}
					numeric = option.Bool
				}
			}
			return NumberValue(float64(collation.Compare(value.String, target, locale, numeric))), nil
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
	case "replaceAll":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) < 2 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.replaceAll expects 2 arguments")
			}
			if args[1].Kind == ValueKindFunction || args[1].Kind == ValueKindHostReference {
				if compiled, flags, ok, err := classicJSRegExpValue(args[0]); ok || err != nil {
					if err != nil {
						return UndefinedValue(), err
					}
					if !strings.Contains(flags, "g") {
						return UndefinedValue(), NewError(ErrorKindRuntime, "String.replaceAll requires a global regular expression")
					}
					updated, err := replaceRegexpWithCallback(p.host, compiled, value.String, args[1], true)
					if err != nil {
						return UndefinedValue(), err
					}
					return StringValue(updated), nil
				}
				search := ToJSString(args[0])
				updated, err := replaceStringAllWithCallback(p.host, value.String, search, args[1])
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
				if !strings.Contains(flags, "g") {
					return UndefinedValue(), NewError(ErrorKindRuntime, "String.replaceAll requires a global regular expression")
				}
				return StringValue(compiled.ReplaceAllString(value.String, replacement)), nil
			}
			search := ToJSString(args[0])
			return StringValue(strings.ReplaceAll(value.String, search, replacement)), nil
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
	case "matchAll":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.matchAll expects 1 argument")
			}
			if compiled, flags, ok, err := classicJSRegExpValue(args[0]); ok || err != nil {
				if err != nil {
					return UndefinedValue(), err
				}
				if !strings.Contains(flags, "g") {
					return UndefinedValue(), NewError(ErrorKindRuntime, "String.matchAll requires a global regular expression")
				}
				return browserStringMatchAllMatches(value.String, compiled), nil
			}
			quoted := regexp.MustCompile(regexp.QuoteMeta(ToJSString(args[0])))
			return browserStringMatchAllMatches(value.String, quoted), nil
		}), true, nil
	case "search":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			searchValue := UndefinedValue()
			if len(args) > 0 {
				searchValue = args[0]
			}
			if compiled, _, ok, err := classicJSRegExpValue(searchValue); ok || err != nil {
				if err != nil {
					return UndefinedValue(), err
				}
				match := compiled.FindStringIndex(value.String)
				if match == nil {
					return NumberValue(-1), nil
				}
				return NumberValue(float64(browserRuneOffset(value.String, match[0]))), nil
			}
			search := ToJSString(searchValue)
			idx := strings.Index(value.String, search)
			if idx < 0 {
				return NumberValue(-1), nil
			}
			return NumberValue(float64(browserRuneOffset(value.String, idx))), nil
		}), true, nil
	case "lastIndexOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.lastIndexOf expects 1 argument")
			}
			search := ToJSString(args[0])
			runes := []rune(value.String)
			length := len(runes)
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
			searchRunes := []rune(search)
			limit := fromIndex + len(searchRunes)
			if limit > length {
				limit = length
			}
			suffix := string(runes[:limit])
			idx := strings.LastIndex(suffix, search)
			if idx < 0 {
				return NumberValue(-1), nil
			}
			return NumberValue(float64(utf8.RuneCountInString(suffix[:idx]))), nil
		}), true, nil
	case "indexOf":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.indexOf expects 1 argument")
			}
			search := ToJSString(args[0])
			runes := []rune(value.String)
			length := len(runes)
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
			suffix := string(runes[fromIndex:])
			idx := strings.Index(suffix, search)
			if idx == -1 {
				return NumberValue(-1), nil
			}
			return NumberValue(float64(fromIndex + utf8.RuneCountInString(suffix[:idx]))), nil
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
			runes := []rune(value.String)
			length := len(runes)
			fromIndex := 0
			if len(args) > 1 {
				fromIndex = indexFromValue(args[1], 0)
			}
			if fromIndex < 0 {
				fromIndex = 0
			}
			if fromIndex > length {
				return BoolValue(false), nil
			}
			if search == "" {
				return BoolValue(true), nil
			}
			return BoolValue(strings.HasPrefix(string(runes[fromIndex:]), search)), nil
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
			runes := []rune(value.String)
			length := len(runes)
			end := length
			if len(args) > 1 {
				end = indexFromValue(args[1], end)
			}
			if end < 0 {
				end = 0
			}
			if end > length {
				end = length
			}
			if search == "" {
				return BoolValue(true), nil
			}
			if len([]rune(search)) > end {
				return BoolValue(false), nil
			}
			return BoolValue(strings.HasSuffix(string(runes[:end]), search)), nil
		}), true, nil
	case "includes":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("String.includes expects 1 argument")
			}
			search := ToJSString(args[0])
			runes := []rune(value.String)
			length := len(runes)
			start := 0
			if len(args) > 1 {
				start = indexFromValue(args[1], 0)
			}
			if start < 0 {
				start = 0
			}
			if start > length {
				return BoolValue(false), nil
			}
			return BoolValue(strings.Contains(string(runes[start:]), search)), nil
		}), true, nil
	case "at":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			runes := []rune(value.String)
			index, ok, err := browserAtIndex(len(runes), args, true)
			if err != nil {
				return UndefinedValue(), err
			}
			if !ok {
				return UndefinedValue(), nil
			}
			return StringValue(string(runes[index])), nil
		}), true, nil
	case "codePointAt":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			runes := []rune(value.String)
			index, ok, err := browserAtIndex(len(runes), args, false)
			if err != nil {
				return UndefinedValue(), err
			}
			if !ok {
				return UndefinedValue(), nil
			}
			return NumberValue(float64(runes[index])), nil
		}), true, nil
	case "padStart":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.padStart expects 1 argument")
			}
			valueLength := utf8.RuneCountInString(value.String)
			targetLength := indexFromValue(args[0], valueLength)
			if targetLength <= valueLength {
				return StringValue(value.String), nil
			}
			fill := " "
			if len(args) > 1 && args[1].Kind != ValueKindUndefined {
				fill = ToJSString(args[1])
			}
			if fill == "" {
				return StringValue(value.String), nil
			}
			needed := targetLength - valueLength
			return StringValue(repeatStringToRuneLength(fill, needed) + value.String), nil
		}), true, nil
	case "padEnd":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.padEnd expects 1 argument")
			}
			valueLength := utf8.RuneCountInString(value.String)
			targetLength := indexFromValue(args[0], valueLength)
			if targetLength <= valueLength {
				return StringValue(value.String), nil
			}
			fill := " "
			if len(args) > 1 && args[1].Kind != ValueKindUndefined {
				fill = ToJSString(args[1])
			}
			if fill == "" {
				return StringValue(value.String), nil
			}
			needed := targetLength - valueLength
			return StringValue(value.String + repeatStringToRuneLength(fill, needed)), nil
		}), true, nil
	case "repeat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			countArg := UndefinedValue()
			if len(args) > 0 {
				countArg = args[0]
			}
			count, err := browserStringRepeatCount(countArg)
			if err != nil {
				return UndefinedValue(), err
			}
			if count == 0 || value.String == "" {
				return StringValue(""), nil
			}
			maxInt := int(^uint(0) >> 1)
			if len(value.String) > maxInt/count {
				return UndefinedValue(), NewError(ErrorKindRuntime, "String.repeat result is too large in this bounded classic-JS slice")
			}
			return StringValue(strings.Repeat(value.String, count)), nil
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
	case "substring":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			runes := []rune(value.String)
			length := len(runes)
			start := indexFromValueOrDefault(args, 0, 0)
			end := indexFromValueOrDefault(args, 1, length)
			if start < 0 {
				start = 0
			}
			if end < 0 {
				end = 0
			}
			if start > length {
				start = length
			}
			if end > length {
				end = length
			}
			if start > end {
				start, end = end, start
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

func browserStringNormalize(text string, args []Value) (string, error) {
	form := "NFC"
	if len(args) > 0 && args[0].Kind != ValueKindUndefined {
		if args[0].Kind == ValueKindSymbol {
			return "", NewError(ErrorKindRuntime, "String.normalize form cannot be a Symbol")
		}
		form = ToJSString(args[0])
	}

	switch form {
	case "NFC":
		return norm.NFC.String(text), nil
	case "NFD":
		return norm.NFD.String(text), nil
	case "NFKC":
		return norm.NFKC.String(text), nil
	case "NFKD":
		return norm.NFKD.String(text), nil
	default:
		return "", NewError(ErrorKindRuntime, "String.normalize form must be NFC, NFD, NFKC, or NFKD")
	}
}

func repeatStringToRuneLength(fill string, targetLength int) string {
	if fill == "" || targetLength <= 0 {
		return ""
	}
	fillRunes := []rune(fill)
	if len(fillRunes) == 0 {
		return ""
	}
	repeated := make([]rune, 0, targetLength)
	for len(repeated) < targetLength {
		repeated = append(repeated, fillRunes...)
	}
	if len(repeated) > targetLength {
		repeated = repeated[:targetLength]
	}
	return string(repeated)
}

func browserStringMatchAllMatches(text string, compiled *regexp.Regexp) Value {
	matches := compiled.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return ArrayValue(nil)
	}
	out := make([]Value, 0, len(matches))
	for _, match := range matches {
		parts := make([]Value, 0, len(match))
		for _, part := range match {
			parts = append(parts, StringValue(part))
		}
		out = append(out, ArrayValue(parts))
	}
	return ArrayValue(out)
}

func browserStringRepeatCount(value Value) (int, error) {
	number, ok := classicJSUnaryNumberValue(value)
	if !ok {
		return 0, NewError(ErrorKindRuntime, "String.repeat count must be a scalar value in this bounded classic-JS slice")
	}
	if math.IsNaN(number) {
		return 0, nil
	}
	if math.IsInf(number, 0) {
		return 0, NewError(ErrorKindRuntime, "String.repeat count must be finite")
	}
	truncated := math.Trunc(number)
	if truncated < 0 {
		return 0, NewError(ErrorKindRuntime, "String.repeat count must be non-negative")
	}
	maxInt := float64(int(^uint(0) >> 1))
	if truncated > maxInt {
		return 0, NewError(ErrorKindRuntime, "String.repeat count is too large in this bounded classic-JS slice")
	}
	return int(truncated), nil
}

func browserArrayReduce(p *classicJSStatementParser, elements []Value, callback Value, initial Value, hasInitial bool, source Value, reverse bool, methodName string) (Value, error) {
	if len(elements) == 0 {
		if hasInitial {
			return initial, nil
		}
		return UndefinedValue(), fmt.Errorf("%s requires at least one value", methodName)
	}
	start, end, step := 0, len(elements), 1
	if reverse {
		start, end, step = len(elements)-1, -1, -1
	}
	var accumulator Value
	if hasInitial {
		accumulator = initial
	} else {
		accumulator = elements[start]
		start += step
	}
	for i := start; i != end; i += step {
		result, err := p.invokeCallableValue(callback, []Value{
			accumulator,
			elements[i],
			NumberValue(float64(i)),
			source,
		}, UndefinedValue(), false)
		if err != nil {
			return UndefinedValue(), err
		}
		accumulator = result
	}
	return accumulator, nil
}

func browserAtIndex(length int, args []Value, wrapNegative bool) (int, bool, error) {
	indexValue := UndefinedValue()
	if len(args) > 0 && args[0].Kind != ValueKindUndefined {
		indexValue = args[0]
	}
	number, ok := classicJSNumberValue(indexValue)
	if !ok {
		return 0, false, NewError(ErrorKindRuntime, "at index must be a scalar value in this bounded classic-JS slice")
	}
	if math.IsNaN(number) {
		return 0, true, nil
	}
	if math.IsInf(number, 0) {
		return 0, false, nil
	}
	truncated := math.Trunc(number)
	index := int(truncated)
	if truncated < 0 {
		if !wrapNegative {
			return 0, false, nil
		}
		index = length + index
	}
	if index < 0 || index >= length {
		return 0, false, nil
	}
	return index, true, nil
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
	case "toLocaleString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			locale := "en-US"
			var options Value
			hasOptions := false

			switch len(args) {
			case 0:
			case 1:
				if args[0].Kind == ValueKindObject {
					options = args[0]
					hasOptions = true
				} else if args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
					locale = strings.TrimSpace(ToJSString(args[0]))
				}
			default:
				if args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
					locale = strings.TrimSpace(ToJSString(args[0]))
				}
				if args[1].Kind != ValueKindObject {
					return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toLocaleString options argument must be an object")
				}
				options = args[1]
				hasOptions = true
			}

			if locale == "" {
				locale = "en-US"
			}

			minFractionDigits := 0
			maxFractionDigits := -1
			maxSignificantDigits := -1
			style := ""
			currency := ""
			if hasOptions {
				if value, ok := lookupObjectProperty(options.Object, "style"); ok {
					style = strings.ToLower(strings.TrimSpace(ToJSString(value)))
				}
				if value, ok := lookupObjectProperty(options.Object, "currency"); ok {
					currency = strings.ToUpper(strings.TrimSpace(ToJSString(value)))
				}
				if value, ok := lookupObjectProperty(options.Object, "minimumFractionDigits"); ok {
					digits, err := classicJSNumberLocaleDigits("Number.toLocaleString minimumFractionDigits", value, 0, 100)
					if err != nil {
						return UndefinedValue(), err
					}
					minFractionDigits = digits
				}
				if value, ok := lookupObjectProperty(options.Object, "maximumFractionDigits"); ok {
					digits, err := classicJSNumberLocaleDigits("Number.toLocaleString maximumFractionDigits", value, 0, 100)
					if err != nil {
						return UndefinedValue(), err
					}
					maxFractionDigits = digits
				}
				if maxFractionDigits >= 0 && minFractionDigits > maxFractionDigits {
					return UndefinedValue(), NewError(ErrorKindRuntime, "Number.toLocaleString minimumFractionDigits cannot exceed maximumFractionDigits")
				}
				if value, ok := lookupObjectProperty(options.Object, "maximumSignificantDigits"); ok {
					digits, err := classicJSNumberLocaleDigits("Number.toLocaleString maximumSignificantDigits", value, 1, 100)
					if err != nil {
						return UndefinedValue(), err
					}
					maxSignificantDigits = digits
				}
			}

			return StringValue(numberToLocaleStringText(value.Number, locale, minFractionDigits, maxFractionDigits, maxSignificantDigits, style, currency)), nil
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
	case "toDateString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(BrowserDateDateString(ms)), nil
		}), true, nil
	case "toTimeString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(BrowserDateTimeString(ms)), nil
		}), true, nil
	case "toISOString", "toJSON", "toString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(BrowserDateISOString(ms)), nil
		}), true, nil
	case "toUTCString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(BrowserDateUTCString(ms)), nil
		}), true, nil
	case "toLocaleString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			locale := "en-US"
			if len(args) > 0 && args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
				locale = strings.TrimSpace(ToJSString(args[0]))
			}
			return StringValue(BrowserDateLocaleString(ms, locale)), nil
		}), true, nil
	case "toLocaleTimeString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			locale := "en-US"
			if len(args) > 0 && args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
				locale = strings.TrimSpace(ToJSString(args[0]))
			}
			return StringValue(BrowserDateLocaleTimeString(ms, locale)), nil
		}), true, nil
	case "toLocaleDateString":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			locale := "en-US"
			if len(args) > 0 && args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
				locale = strings.TrimSpace(ToJSString(args[0]))
			}
			return StringValue(BrowserDateLocaleDateString(ms, locale)), nil
		}), true, nil
	case "setTime":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.setTime expects 1 argument")
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.setTime requires a finite timestamp")
			}
			if !BrowserDateSetTimestamp(&value, int64(number)) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.setTime requires a date receiver")
			}
			return NumberValue(float64(int64(number))), nil
		}), true, nil
	case "setDate", "setUTCDate":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s expects 1 argument", name)
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
			}
			if !BrowserDateSetDayOfMonth(&value, int64(number)) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a date receiver", name)
			}
			ms, _ := BrowserDateTimestamp(value)
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "setMonth", "setUTCMonth":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) < 1 || len(args) > 2 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s expects 1 or 2 arguments", name)
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
			}
			var day *int64
			if len(args) > 1 {
				dom, ok := classicJSNumberValue(args[1])
				if !ok || math.IsNaN(dom) || math.IsInf(dom, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				day = new(int64)
				*day = int64(dom)
			}
			if !BrowserDateSetMonth(&value, int64(number), day) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a date receiver", name)
			}
			ms, _ := BrowserDateTimestamp(value)
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "setFullYear", "setUTCFullYear":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) < 1 || len(args) > 3 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s expects 1 to 3 arguments", name)
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
			}
			var month *int64
			if len(args) > 1 {
				mon, ok := classicJSNumberValue(args[1])
				if !ok || math.IsNaN(mon) || math.IsInf(mon, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				month = new(int64)
				*month = int64(mon)
			}
			var day *int64
			if len(args) > 2 {
				dom, ok := classicJSNumberValue(args[2])
				if !ok || math.IsNaN(dom) || math.IsInf(dom, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				day = new(int64)
				*day = int64(dom)
			}
			if !BrowserDateSetFullYear(&value, int64(number), month, day) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a date receiver", name)
			}
			ms, _ := BrowserDateTimestamp(value)
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "setMilliseconds", "setUTCMilliseconds":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s expects 1 argument", name)
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
			}
			if !BrowserDateSetMilliseconds(&value, int64(number)) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a date receiver", name)
			}
			ms, _ := BrowserDateTimestamp(value)
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "setSeconds", "setUTCSeconds":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) < 1 || len(args) > 2 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s expects 1 or 2 arguments", name)
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
			}
			var milliseconds *int64
			if len(args) > 1 {
				ms, ok := classicJSNumberValue(args[1])
				if !ok || math.IsNaN(ms) || math.IsInf(ms, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				milliseconds = new(int64)
				*milliseconds = int64(ms)
			}
			if !BrowserDateSetSeconds(&value, int64(number), milliseconds) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a date receiver", name)
			}
			ms, _ := BrowserDateTimestamp(value)
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "setMinutes", "setUTCMinutes":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) < 1 || len(args) > 3 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s expects 1 to 3 arguments", name)
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
			}
			var seconds *int64
			if len(args) > 1 {
				sec, ok := classicJSNumberValue(args[1])
				if !ok || math.IsNaN(sec) || math.IsInf(sec, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				seconds = new(int64)
				*seconds = int64(sec)
			}
			var milliseconds *int64
			if len(args) > 2 {
				ms, ok := classicJSNumberValue(args[2])
				if !ok || math.IsNaN(ms) || math.IsInf(ms, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				milliseconds = new(int64)
				*milliseconds = int64(ms)
			}
			if !BrowserDateSetMinutes(&value, int64(number), seconds, milliseconds) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a date receiver", name)
			}
			ms, _ := BrowserDateTimestamp(value)
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "setHours", "setUTCHours":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) < 1 || len(args) > 4 {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s expects 1 to 4 arguments", name)
			}
			number, ok := classicJSNumberValue(args[0])
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
			}
			var minutes *int64
			if len(args) > 1 {
				min, ok := classicJSNumberValue(args[1])
				if !ok || math.IsNaN(min) || math.IsInf(min, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				minutes = new(int64)
				*minutes = int64(min)
			}
			var seconds *int64
			if len(args) > 2 {
				sec, ok := classicJSNumberValue(args[2])
				if !ok || math.IsNaN(sec) || math.IsInf(sec, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				seconds = new(int64)
				*seconds = int64(sec)
			}
			var milliseconds *int64
			if len(args) > 3 {
				ms, ok := classicJSNumberValue(args[3])
				if !ok || math.IsNaN(ms) || math.IsInf(ms, 0) {
					return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a finite timestamp", name)
				}
				milliseconds = new(int64)
				*milliseconds = int64(ms)
			}
			if !BrowserDateSetHours(&value, int64(number), minutes, seconds, milliseconds) {
				return UndefinedValue(), fmt.Errorf("Date.prototype.%s requires a date receiver", name)
			}
			ms, _ := BrowserDateTimestamp(value)
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "valueOf", "getTime":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(ms)), nil
		}), true, nil
	case "getFullYear":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateYear(ms))), nil
		}), true, nil
	case "getUTCFullYear":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateYear(ms))), nil
		}), true, nil
	case "getMonth":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateMonth(ms))), nil
		}), true, nil
	case "getUTCMonth":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateUTCMonth(ms))), nil
		}), true, nil
	case "getDate", "getUTCDate":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateDayOfMonth(ms))), nil
		}), true, nil
	case "getDay", "getUTCDay":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateDayOfWeek(ms))), nil
		}), true, nil
	case "getHours", "getUTCHours":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateHour(ms))), nil
		}), true, nil
	case "getMinutes", "getUTCMinutes":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateMinute(ms))), nil
		}), true, nil
	case "getSeconds", "getUTCSeconds":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateSecond(ms))), nil
		}), true, nil
	case "getMilliseconds", "getUTCMilliseconds":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateMillisecond(ms))), nil
		}), true, nil
	case "getTimezoneOffset":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return NumberValue(float64(BrowserDateTimezoneOffset(ms))), nil
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
	_, rejected, settledValue := promiseSettlement(value)

	switch name {
	case "then":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) > 2 {
				return UndefinedValue(), fmt.Errorf("Promise.then accepts at most 2 arguments")
			}
			handlerAbsent := func(index int) bool {
				return len(args) <= index || promiseHandlerIsAbsent(args[index])
			}

			chainHandlerResult := func(result Value) (Value, error) {
				resultPromise := &classicJSPromiseState{}
				if settlePromiseFromResult(resultPromise, result) {
					return PendingPromiseValue(resultPromise), nil
				}
				return promiseValueFromState(resultPromise), nil
			}

			if !isPending || pending == nil {
				if rejected {
					if handlerAbsent(1) {
						return RejectedPromiseValue(settledValue), nil
					}
					result, err := p.invokeCallableValue(args[1], []Value{settledValue}, UndefinedValue(), false)
					if err != nil {
						return UndefinedValue(), err
					}
					return chainHandlerResult(result)
				}
				if handlerAbsent(0) {
					return PromiseValue(resolved), nil
				}
				result, err := p.invokeCallableValue(args[0], []Value{resolved}, UndefinedValue(), false)
				if err != nil {
					return UndefinedValue(), err
				}
				return chainHandlerResult(result)
			}

			resultPromise := &classicJSPromiseState{}
			pending.addWaiter(func(resolvedValue Value, rejected bool) {
				if rejected {
					if handlerAbsent(1) {
						resultPromise.reject(resolvedValue)
						return
					}
					result, err := p.invokeCallableValue(args[1], []Value{resolvedValue}, UndefinedValue(), false)
					if err != nil {
						resultPromise.reject(rejectionReasonFromError(err))
						return
					}
					settlePromiseFromResult(resultPromise, result)
					return
				}
				if handlerAbsent(0) {
					resultPromise.resolve(unwrapPromiseValue(resolvedValue))
					return
				}
				result, err := p.invokeCallableValue(args[0], []Value{unwrapPromiseValue(resolvedValue)}, UndefinedValue(), false)
				if err != nil {
					resultPromise.reject(rejectionReasonFromError(err))
					return
				}
				settlePromiseFromResult(resultPromise, result)
			})
			return promiseValueFromState(resultPromise), nil
		}), true, nil
	case "catch":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) > 1 {
				return UndefinedValue(), fmt.Errorf("Promise.catch accepts at most 1 argument")
			}
			handlerAbsent := func(index int) bool {
				return len(args) <= index || promiseHandlerIsAbsent(args[index])
			}
			if !isPending || pending == nil {
				if !rejected {
					return PromiseValue(resolved), nil
				}
				if handlerAbsent(0) {
					return RejectedPromiseValue(settledValue), nil
				}
				result, err := p.invokeCallableValue(args[0], []Value{settledValue}, UndefinedValue(), false)
				if err != nil {
					return UndefinedValue(), err
				}
				resultPromise := &classicJSPromiseState{}
				if settlePromiseFromResult(resultPromise, result) {
					return PendingPromiseValue(resultPromise), nil
				}
				return promiseValueFromState(resultPromise), nil
			}
			resultPromise := &classicJSPromiseState{}
			pending.addWaiter(func(resolvedValue Value, rejected bool) {
				if !rejected {
					resultPromise.resolve(unwrapPromiseValue(resolvedValue))
					return
				}
				if handlerAbsent(0) {
					resultPromise.reject(resolvedValue)
					return
				}
				result, err := p.invokeCallableValue(args[0], []Value{resolvedValue}, UndefinedValue(), false)
				if err != nil {
					resultPromise.reject(rejectionReasonFromError(err))
					return
				}
				settlePromiseFromResult(resultPromise, result)
			})
			return promiseValueFromState(resultPromise), nil
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

func arrayLocaleStringText(p *classicJSStatementParser, values []Value, args []Value) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, element := range values {
		if i > 0 {
			b.WriteByte(',')
		}
		text, err := arrayElementLocaleString(p, element, args)
		if err != nil {
			return "", err
		}
		b.WriteString(text)
	}
	return b.String(), nil
}

func arrayElementLocaleString(p *classicJSStatementParser, value Value, args []Value) (string, error) {
	switch value.Kind {
	case ValueKindUndefined, ValueKindNull:
		return "", nil
	case ValueKindString:
		return value.String, nil
	case ValueKindBool, ValueKindBigInt, ValueKindSymbol:
		return ToJSString(value), nil
	case ValueKindArray:
		return arrayLocaleStringText(p, value.Array, args)
	case ValueKindNumber:
		if p != nil {
			if method, ok, err := p.resolveNumberPrototypeMethod(value, "toLocaleString"); ok || err != nil {
				if err != nil {
					return "", err
				}
				result, err := p.invokeCallableValue(method, args, value, true)
				if err != nil {
					return "", err
				}
				return ToJSString(result), nil
			}
		}
		return ToJSString(value), nil
	case ValueKindObject:
		if ms, ok := BrowserDateTimestamp(value); ok {
			if p != nil {
				if method, ok, err := p.resolveDatePrototypeMethod(value, "toLocaleString"); ok || err != nil {
					if err != nil {
						return "", err
					}
					result, err := p.invokeCallableValue(method, args, value, true)
					if err != nil {
						return "", err
					}
					return ToJSString(result), nil
				}
			}
			locale := "en-US"
			if len(args) > 0 && args[0].Kind != ValueKindUndefined && args[0].Kind != ValueKindNull {
				locale = strings.TrimSpace(ToJSString(args[0]))
				if locale == "" {
					locale = "en-US"
				}
			}
			return BrowserDateLocaleString(ms, locale), nil
		}
		if resolved, ok := lookupObjectProperty(value.Object, "toLocaleString"); ok {
			if resolved.Kind == ValueKindFunction && resolved.Function != nil {
				if p == nil {
					return ToJSString(value), nil
				}
				result, err := p.invokeCallableValue(resolved, args, value, true)
				if err != nil {
					return "", err
				}
				return ToJSString(result), nil
			}
			return ToJSString(resolved), nil
		}
		return ToJSString(value), nil
	default:
		return ToJSString(value), nil
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
	loc := compiled.FindStringSubmatchIndex(input)
	if loc == nil {
		return input
	}
	expanded := compiled.ExpandString(nil, replacement, input, loc)
	return input[:loc[0]] + string(expanded) + input[loc[1]:]
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

func replaceStringAllWithCallback(host HostBindings, input, search string, replacer Value) (string, error) {
	if search == "" {
		var b strings.Builder
		replacement, err := invokeStringReplaceCallback(host, replacer, input, search, 0, 0, nil)
		if err != nil {
			return "", err
		}
		b.Grow(len(input) + len(replacement))
		b.WriteString(replacement)
		for offset := 0; offset < len(input); {
			_, size := utf8.DecodeRuneInString(input[offset:])
			b.WriteString(input[offset : offset+size])
			offset += size
			replacement, err := invokeStringReplaceCallback(host, replacer, input, search, offset, offset, nil)
			if err != nil {
				return "", err
			}
			b.WriteString(replacement)
		}
		return b.String(), nil
	}

	var b strings.Builder
	last := 0
	for {
		index := strings.Index(input[last:], search)
		if index < 0 {
			break
		}
		index += last
		b.WriteString(input[last:index])
		replacement, err := invokeStringReplaceCallback(host, replacer, input, search, index, index+len(search), nil)
		if err != nil {
			return "", err
		}
		b.WriteString(replacement)
		last = index + len(search)
	}
	b.WriteString(input[last:])
	return b.String(), nil
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
	args = append(args, NumberValue(float64(browserRuneOffset(input, start))), StringValue(input))
	result, err := InvokeCallableValue(host, replacer, args, UndefinedValue(), false)
	if err != nil {
		return "", err
	}
	return ToJSString(result), nil
}

func browserRuneOffset(input string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(input) {
		return utf8.RuneCountInString(input)
	}
	return utf8.RuneCountInString(input[:byteOffset])
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

func classicJSNumberLocaleDigits(method string, value Value, min, max int) (int, error) {
	if value.Kind != ValueKindNumber && value.Kind != ValueKindBigInt {
		return 0, NewError(ErrorKindRuntime, method+" must be numeric")
	}
	number, ok := classicJSNumberValue(value)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, NewError(ErrorKindRuntime, method+" must be numeric")
	}
	digits := int(math.Trunc(number))
	if digits < min || digits > max {
		return 0, NewError(ErrorKindRuntime, fmt.Sprintf("%s must be between %d and %d", method, min, max))
	}
	return digits, nil
}

func numberToLocaleStringText(number float64, locale string, minFractionDigits, maxFractionDigits, maxSignificantDigits int, style, currency string) string {
	switch {
	case math.IsNaN(number):
		return "NaN"
	case math.IsInf(number, 1):
		return "Infinity"
	case math.IsInf(number, -1):
		return "-Infinity"
	}
	if maxSignificantDigits > 0 {
		formatted := numberToLocaleStringWithSignificantDigits(number, maxSignificantDigits)
		if style == "currency" {
			formatted = numberToLocaleCurrencyFormat(formatted, currency, locale)
		}
		return formatted
	}

	var formatted string
	if maxFractionDigits < 0 {
		formatted = numberToLocaleGroupDecimalIntegerPart(numberToLocaleStringWithMinimumFractionDigits(strconv.FormatFloat(number, 'f', -1, 64), minFractionDigits))
	} else {
		pow := math.Pow10(maxFractionDigits)
		rounded := math.Round(number*pow) / pow
		text := strconv.FormatFloat(rounded, 'f', maxFractionDigits, 64)
		if strings.Contains(text, ".") {
			text = strings.TrimRight(text, "0")
			text = strings.TrimRight(text, ".")
		}
		if text == "" || text == "-0" {
			text = "0"
		}
		formatted = numberToLocaleGroupDecimalIntegerPart(numberToLocaleStringWithMinimumFractionDigits(text, minFractionDigits))
	}
	if style == "currency" {
		formatted = numberToLocaleCurrencyFormat(formatted, currency, locale)
	}
	return formatted
}

func numberToLocaleStringWithMinimumFractionDigits(text string, minFractionDigits int) string {
	if minFractionDigits <= 0 || text == "" {
		return text
	}
	sign := ""
	if text[0] == '+' || text[0] == '-' {
		sign = text[:1]
		text = text[1:]
	}
	if text == "" || strings.ContainsAny(text, "eE") {
		return sign + text
	}
	parts := strings.SplitN(text, ".", 2)
	if len(parts) == 1 {
		return sign + parts[0] + "." + strings.Repeat("0", minFractionDigits)
	}
	if len(parts[1]) < minFractionDigits {
		parts[1] += strings.Repeat("0", minFractionDigits-len(parts[1]))
	}
	return sign + parts[0] + "." + parts[1]
}

func numberToLocaleStringWithSignificantDigits(value float64, maxSignificantDigits int) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	}
	if value == 0 {
		return "0"
	}
	if maxSignificantDigits <= 0 {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}

	absValue := math.Abs(value)
	if absValue == 0 {
		return "0"
	}
	exponent := math.Floor(math.Log10(absValue))
	decimalPlaces := maxSignificantDigits - 1 - int(exponent)
	if decimalPlaces >= 0 {
		pow := math.Pow10(decimalPlaces)
		rounded := math.Round(value*pow) / pow
		text := strconv.FormatFloat(rounded, 'f', decimalPlaces, 64)
		if strings.Contains(text, ".") {
			text = strings.TrimRight(text, "0")
			text = strings.TrimRight(text, ".")
		}
		if text == "" || text == "-0" {
			return "0"
		}
		return text
	}

	pow := math.Pow10(-decimalPlaces)
	rounded := math.Round(value/pow) * pow
	text := strconv.FormatFloat(rounded, 'f', 0, 64)
	if text == "" || text == "-0" {
		return "0"
	}
	return numberToLocaleGroupDecimalIntegerPart(text)
}

func numberToLocaleGroupDecimalIntegerPart(text string) string {
	if text == "" {
		return text
	}
	sign := ""
	if text[0] == '+' || text[0] == '-' {
		sign = text[:1]
		text = text[1:]
	}
	if text == "" || strings.ContainsAny(text, "eE") {
		return sign + text
	}
	parts := strings.SplitN(text, ".", 2)
	integer := parts[0]
	if len(integer) <= 3 {
		if len(parts) == 2 {
			return sign + integer + "." + parts[1]
		}
		return sign + integer
	}
	var b strings.Builder
	for i, r := range integer {
		if i > 0 && (len(integer)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if len(parts) == 2 {
		return sign + b.String() + "." + parts[1]
	}
	return sign + b.String()
}

func numberToLocaleCurrencyFormat(formatted, currency, locale string) string {
	symbol := numberToLocaleCurrencySymbol(currency, locale)
	if symbol == "" {
		symbol = currency
	}
	if symbol == "" {
		return formatted
	}
	if strings.HasPrefix(formatted, "-") {
		return "-" + symbol + strings.TrimPrefix(formatted, "-")
	}
	return symbol + formatted
}

func numberToLocaleCurrencySymbol(currency, locale string) string {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "JPY":
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "ja") {
			return "￥"
		}
		return "¥"
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	}
	return ""
}
