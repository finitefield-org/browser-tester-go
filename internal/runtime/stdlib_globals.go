package runtime

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

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
	case strings.HasPrefix(path, "Object."):
		value, err := resolveObjectReference(strings.TrimPrefix(path, "Object."))
		return value, true, err
	case path == "JSON":
		return script.HostObjectReference("JSON"), true, nil
	case strings.HasPrefix(path, "JSON."):
		value, err := resolveJSONReference(strings.TrimPrefix(path, "JSON."))
		return value, true, err
	case path == "Number":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserNumberConstructor(args)
		}), true, nil
	case strings.HasPrefix(path, "Number."):
		value, err := resolveNumberReference(strings.TrimPrefix(path, "Number."))
		return value, true, err
	case path == "String":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserStringConstructor(args)
		}), true, nil
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

func resolveObjectReference(path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "assign":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectAssign(args)
		}), nil
	case "entries":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserObjectEntries(args)
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
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Number."+path))
}

func resolveMathReference(session *Session, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
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
	case "min":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.NumberValue(math.Inf(1)), nil
			}
			result := math.Inf(1)
			for _, arg := range args {
				value, err := coerceNumber(arg)
				if err != nil {
					return script.UndefinedValue(), err
				}
				if math.IsNaN(value) {
					return script.NumberValue(math.NaN()), nil
				}
				if value < result {
					result = value
				}
			}
			return script.NumberValue(result), nil
		}), nil
	case "max":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.NumberValue(math.Inf(-1)), nil
			}
			result := math.Inf(-1)
			for _, arg := range args {
				value, err := coerceNumber(arg)
				if err != nil {
					return script.UndefinedValue(), err
				}
				if math.IsNaN(value) {
					return script.NumberValue(math.NaN()), nil
				}
				if value > result {
					result = value
				}
			}
			return script.NumberValue(result), nil
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

func browserObjectAssign(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.UndefinedValue(), fmt.Errorf("Object.assign expects at least 1 argument")
	}
	if args[0].Kind != script.ValueKindObject {
		return script.UndefinedValue(), fmt.Errorf("Object.assign target must be an object")
	}
	target := args[0]
	entries := target.Object
	for _, source := range args[1:] {
		if source.Kind != script.ValueKindObject {
			return script.UndefinedValue(), fmt.Errorf("Object.assign source must be an object")
		}
		for _, entry := range source.Object {
			if script.IsInternalObjectKey(entry.Key) {
				continue
			}
			index := findObjectEntryIndex(entries, entry.Key)
			if index >= 0 {
				entries[index].Value = entry.Value
			} else {
				entries = append(entries, script.ObjectEntry{Key: entry.Key, Value: entry.Value})
			}
		}
	}
	target.Object = entries
	return target, nil
}

func browserObjectKeys(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.keys expects 1 argument")
	}
	if args[0].Kind != script.ValueKindObject {
		return script.UndefinedValue(), fmt.Errorf("Object.keys expects an object")
	}
	keys := uniqueObjectKeys(args[0].Object)
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
	if args[0].Kind != script.ValueKindObject {
		return script.UndefinedValue(), fmt.Errorf("Object.entries expects an object")
	}
	keys := uniqueObjectKeys(args[0].Object)
	entries := make([]script.Value, 0, len(keys))
	for _, key := range keys {
		value, _ := objectProperty(args[0], key)
		entries = append(entries, script.ArrayValue([]script.Value{
			script.StringValue(key),
			value,
		}))
	}
	return script.ArrayValue(entries), nil
}

func browserObjectValues(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("Object.values expects 1 argument")
	}
	if args[0].Kind != script.ValueKindObject {
		return script.UndefinedValue(), fmt.Errorf("Object.values expects an object")
	}
	keys := uniqueObjectKeys(args[0].Object)
	entries := make([]script.Value, 0, len(keys))
	for _, key := range keys {
		value, _ := objectProperty(args[0], key)
		entries = append(entries, value)
	}
	return script.ArrayValue(entries), nil
}

func browserJSONParse(args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("JSON.parse expects 1 argument")
	}
	input := script.ToJSString(args[0])
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return script.UndefinedValue(), err
	}
	return jsonValueToScript(data)
}

func browserJSONStringify(args []script.Value) (script.Value, error) {
	if len(args) == 0 {
		return script.UndefinedValue(), nil
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("JSON.stringify expects 1 argument in this bounded slice")
	}
	text, err := jsonStringifyValue(args[0])
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
		value, err := coerceNumber(args[0])
		if err != nil {
			return script.UndefinedValue(), err
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return script.UndefinedValue(), fmt.Errorf("Date constructor requires a finite timestamp")
		}
		ms = int64(value)
	}
	return browserDateValue(ms), nil
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

func jsonValueToScript(value interface{}) (script.Value, error) {
	switch typed := value.(type) {
	case nil:
		return script.NullValue(), nil
	case bool:
		return script.BoolValue(typed), nil
	case float64:
		return script.NumberValue(typed), nil
	case string:
		return script.StringValue(typed), nil
	case []interface{}:
		values := make([]script.Value, 0, len(typed))
		for _, entry := range typed {
			converted, err := jsonValueToScript(entry)
			if err != nil {
				return script.UndefinedValue(), err
			}
			values = append(values, converted)
		}
		return script.ArrayValue(values), nil
	case map[string]interface{}:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		entries := make([]script.ObjectEntry, 0, len(keys))
		for _, key := range keys {
			converted, err := jsonValueToScript(typed[key])
			if err != nil {
				return script.UndefinedValue(), err
			}
			entries = append(entries, script.ObjectEntry{Key: key, Value: converted})
		}
		return script.ObjectValue(entries), nil
	default:
		return script.UndefinedValue(), fmt.Errorf("unsupported JSON value type %T", value)
	}
}

func jsonStringifyValue(value script.Value) (string, error) {
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
			encoded, err := jsonStringifyValue(entry)
			if err != nil {
				return "", err
			}
			parts = append(parts, encoded)
		}
		return "[" + strings.Join(parts, ",") + "]", nil
	case script.ValueKindObject:
		keys := uniqueObjectKeys(value.Object)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return "", err
			}
			encodedValue, err := jsonStringifyValue(objectValueByKey(value.Object, key))
			if err != nil {
				return "", err
			}
			parts = append(parts, string(encodedKey)+":"+encodedValue)
		}
		return "{" + strings.Join(parts, ",") + "}", nil
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
		if script.IsInternalObjectKey(entry.Key) {
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
