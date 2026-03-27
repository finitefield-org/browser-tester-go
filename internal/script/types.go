package script

import (
	"math"
	"strconv"
	"strings"
)

type ValueKind string

const (
	ValueKindUndefined     ValueKind = "undefined"
	ValueKindNull          ValueKind = "null"
	ValueKindString        ValueKind = "string"
	ValueKindBool          ValueKind = "bool"
	ValueKindNumber        ValueKind = "number"
	ValueKindBigInt        ValueKind = "bigint"
	ValueKindArray         ValueKind = "array"
	ValueKindObject        ValueKind = "object"
	ValueKindPrivateName   ValueKind = "private-name"
	ValueKindFunction      ValueKind = "function"
	ValueKindHostReference ValueKind = "host-reference"
	ValueKindPromise       ValueKind = "promise"
	ValueKindInvocation    ValueKind = "invocation"
)

type NativeFunction func(args []Value) (Value, error)

type HostReferenceKind string

const (
	HostReferenceKindObject      HostReferenceKind = "object"
	HostReferenceKindFunction    HostReferenceKind = "function"
	HostReferenceKindConstructor HostReferenceKind = "constructor"
)

type ObjectEntry struct {
	Key   string
	Value Value
}

type Value struct {
	Kind              ValueKind
	String            string
	Bool              bool
	Number            float64
	BigInt            string
	Array             []Value
	Object            []ObjectEntry
	PrivateName       string
	ClassKey          string
	ClassDefinition   *classicJSClassDefinition
	Function          *classicJSArrowFunction
	NativeFunction    NativeFunction
	HostReferencePath string
	HostReferenceKind HostReferenceKind
	Promise           *Value
	Invocation        string
}

func UndefinedValue() Value {
	return Value{Kind: ValueKindUndefined}
}

func NullValue() Value {
	return Value{Kind: ValueKindNull}
}

func StringValue(value string) Value {
	return Value{
		Kind:   ValueKindString,
		String: value,
	}
}

func BoolValue(value bool) Value {
	return Value{
		Kind: ValueKindBool,
		Bool: value,
	}
}

func NumberValue(value float64) Value {
	return Value{
		Kind:   ValueKindNumber,
		Number: value,
	}
}

func BigIntValue(value string) Value {
	return Value{
		Kind:   ValueKindBigInt,
		BigInt: value,
	}
}

func ArrayValue(values []Value) Value {
	copied := make([]Value, len(values))
	copy(copied, values)
	return Value{
		Kind:  ValueKindArray,
		Array: copied,
	}
}

func ObjectValue(entries []ObjectEntry) Value {
	copied := make([]ObjectEntry, len(entries))
	copy(copied, entries)
	return Value{
		Kind:   ValueKindObject,
		Object: copied,
	}
}

func PrivateNameValue(name string) Value {
	return Value{
		Kind:        ValueKindPrivateName,
		PrivateName: name,
	}
}

func FunctionValue(fn *classicJSArrowFunction) Value {
	return Value{
		Kind:     ValueKindFunction,
		Function: fn,
	}
}

func NativeFunctionValue(fn NativeFunction) Value {
	return Value{
		Kind:           ValueKindFunction,
		NativeFunction: fn,
	}
}

func HostReferenceValue(path string, kind HostReferenceKind) Value {
	if kind == "" {
		kind = HostReferenceKindObject
	}
	return Value{
		Kind:              ValueKindHostReference,
		HostReferencePath: path,
		HostReferenceKind: kind,
	}
}

func HostObjectReference(path string) Value {
	return HostReferenceValue(path, HostReferenceKindObject)
}

func HostFunctionReference(path string) Value {
	return HostReferenceValue(path, HostReferenceKindFunction)
}

func HostConstructorReference(path string) Value {
	return HostReferenceValue(path, HostReferenceKindConstructor)
}

func PromiseValue(value Value) Value {
	copied := value
	return Value{
		Kind:    ValueKindPromise,
		Promise: &copied,
	}
}

func InvocationValue(source string) Value {
	return Value{
		Kind:       ValueKindInvocation,
		Invocation: source,
	}
}

func ToJSString(value Value) string {
	switch value.Kind {
	case ValueKindUndefined:
		return "undefined"
	case ValueKindNull:
		return "null"
	case ValueKindString:
		return value.String
	case ValueKindBool:
		if value.Bool {
			return "true"
		}
		return "false"
	case ValueKindNumber:
		switch {
		case math.IsNaN(value.Number):
			return "NaN"
		case math.IsInf(value.Number, 1):
			return "Infinity"
		case math.IsInf(value.Number, -1):
			return "-Infinity"
		case value.Number == 0:
			return "0"
		default:
			return strconv.FormatFloat(value.Number, 'f', -1, 64)
		}
	case ValueKindBigInt:
		return value.BigInt
	case ValueKindArray:
		if len(value.Array) == 0 {
			return ""
		}
		var b strings.Builder
		for i, element := range value.Array {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(toJSArrayElementString(element))
		}
		return b.String()
	case ValueKindObject:
		if ms, ok := BrowserDateTimestamp(value); ok {
			return BrowserDateISOString(ms)
		}
		if literal, ok := classicJSRegExpLiteralString(value); ok {
			return literal
		}
		return "[object Object]"
	case ValueKindPrivateName:
		return "#" + value.PrivateName
	case ValueKindFunction:
		return "[Function]"
	case ValueKindHostReference:
		switch value.HostReferenceKind {
		case HostReferenceKindFunction, HostReferenceKindConstructor:
			return "[Function]"
		default:
			return "[object Object]"
		}
	case ValueKindPromise:
		return "[object Promise]"
	default:
		return ""
	}
}

func toJSArrayElementString(value Value) string {
	switch value.Kind {
	case ValueKindUndefined, ValueKindNull:
		return ""
	case ValueKindArray:
		return ToJSString(value)
	case ValueKindObject:
		return "[object Object]"
	case ValueKindPromise:
		return ToJSString(value)
	default:
		return ToJSString(value)
	}
}

func unwrapPromiseValue(value Value) Value {
	for value.Kind == ValueKindPromise {
		if value.Promise == nil {
			return UndefinedValue()
		}
		value = *value.Promise
	}
	return value
}
