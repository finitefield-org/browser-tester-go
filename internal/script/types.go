package script

import (
	"math"
	"strconv"
	"strings"
	"sync/atomic"
)

type ValueKind string

const (
	ValueKindUndefined     ValueKind = "undefined"
	ValueKindNull          ValueKind = "null"
	ValueKindString        ValueKind = "string"
	ValueKindBool          ValueKind = "bool"
	ValueKindNumber        ValueKind = "number"
	ValueKindBigInt        ValueKind = "bigint"
	ValueKindSymbol        ValueKind = "symbol"
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

var nextSymbolID uint64

type Value struct {
	Kind                        ValueKind
	String                      string
	Bool                        bool
	Number                      float64
	BigInt                      string
	SymbolDescription           string
	SymbolID                    string
	Array                       []Value
	Object                      []ObjectEntry
	PrivateName                 string
	ClassKey                    string
	ClassDefinition             *classicJSClassDefinition
	Function                    *classicJSArrowFunction
	NativeFunction              NativeFunction
	NativeConstructibleFunction NativeFunction
	HostReferencePath           string
	HostReferenceKind           HostReferenceKind
	Promise                     *Value
	PromiseState                *classicJSPromiseState
	Invocation                  string
	MapState                    *classicJSMapState
	SetState                    *classicJSSetState
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

func SymbolValue(description string) Value {
	symbolID := strconv.FormatUint(atomic.AddUint64(&nextSymbolID, 1), 10)
	registerSymbolDescription(symbolID, description)
	return Value{
		Kind:              ValueKindSymbol,
		SymbolDescription: description,
		SymbolID:          symbolID,
	}
}

func ArrayValue(values []Value) Value {
	capacity := len(values)
	if capacity == 0 {
		capacity = 1
	}
	copied := make([]Value, len(values), capacity)
	copy(copied, values)
	return Value{
		Kind:  ValueKindArray,
		Array: copied,
	}
}

func ObjectValue(entries []ObjectEntry) Value {
	capacity := len(entries)
	if capacity == 0 {
		capacity = 1
	}
	copied := make([]ObjectEntry, len(entries), capacity)
	copy(copied, entries)
	return Value{
		Kind:   ValueKindObject,
		Object: copied,
	}
}

func objectValueWithMetadata(base Value, entries []ObjectEntry) Value {
	cloned := ObjectValue(entries)
	cloned.ClassKey = base.ClassKey
	cloned.ClassDefinition = base.ClassDefinition
	cloned.MapState = base.MapState
	cloned.SetState = base.SetState
	return cloned
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

func NativeConstructibleFunctionValue(callFn, constructFn NativeFunction) Value {
	return Value{
		Kind:                        ValueKindFunction,
		NativeFunction:              callFn,
		NativeConstructibleFunction: constructFn,
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

func RejectedPromiseValue(reason Value) Value {
	copied := reason
	return Value{
		Kind: ValueKindPromise,
		PromiseState: &classicJSPromiseState{
			resolved: true,
			rejected: true,
			value:    copied,
		},
	}
}

func PendingPromiseValue(state *classicJSPromiseState) Value {
	return Value{
		Kind:         ValueKindPromise,
		PromiseState: state,
	}
}

func NewPendingPromise() (Value, func(Value)) {
	state := &classicJSPromiseState{}
	return PendingPromiseValue(state), state.resolve
}

func NewPendingPromiseWithReject() (Value, func(Value), func(Value)) {
	state := &classicJSPromiseState{}
	return PendingPromiseValue(state), state.resolve, state.reject
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
	case ValueKindSymbol:
		if value.SymbolDescription == "" {
			return "Symbol()"
		}
		return "Symbol(" + value.SymbolDescription + ")"
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
		if value.PromiseState != nil {
			if value.PromiseState.resolved {
				value = value.PromiseState.value
				continue
			}
			return UndefinedValue()
		}
		if value.Promise == nil {
			return UndefinedValue()
		}
		value = *value.Promise
	}
	return value
}

func promiseHandlerIsAbsent(value Value) bool {
	return value.Kind == ValueKindUndefined || value.Kind == ValueKindNull
}

func promiseSettlement(value Value) (settled bool, rejected bool, settlement Value) {
	if value.Kind != ValueKindPromise {
		return false, false, UndefinedValue()
	}
	if value.PromiseState != nil {
		if !value.PromiseState.resolved {
			return false, false, UndefinedValue()
		}
		return true, value.PromiseState.rejected, value.PromiseState.value
	}
	if value.Promise == nil {
		return false, false, UndefinedValue()
	}
	return true, false, *value.Promise
}

func promiseValueFromState(state *classicJSPromiseState) Value {
	if state == nil || !state.resolved {
		return PendingPromiseValue(state)
	}
	if state.rejected {
		return RejectedPromiseValue(state.value)
	}
	return PromiseValue(state.value)
}

func settlePromiseFromResult(target *classicJSPromiseState, result Value) bool {
	if target == nil {
		return false
	}
	if pendingPromise, ok := pendingPromiseState(result); ok {
		pendingPromise.addWaiter(func(next Value, rejected bool) {
			if rejected {
				target.reject(next)
				return
			}
			target.resolve(unwrapPromiseValue(next))
		})
		return true
	}
	if settled, rejected, settlement := promiseSettlement(result); settled {
		if rejected {
			target.reject(settlement)
		} else {
			target.resolve(unwrapPromiseValue(settlement))
		}
		return false
	}
	target.resolve(unwrapPromiseValue(result))
	return false
}

func pendingPromiseState(value Value) (*classicJSPromiseState, bool) {
	if value.Kind != ValueKindPromise || value.PromiseState == nil || value.PromiseState.resolved {
		return nil, false
	}
	return value.PromiseState, true
}
