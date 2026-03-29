package runtime

import (
	"fmt"
	neturl "net/url"
	"sort"
	"strings"

	"browsertester/internal/script"
)

func browserURLSearchParamsValue(parsed *neturl.URL) script.Value {
	if parsed == nil {
		return browserURLSearchParamsValueFromRaw("")
	}
	return browserURLSearchParamsValueFromRaw(parsed.RawQuery)
}

func browserURLSearchParamsValueFromRaw(rawQuery string) script.Value {
	state := &browserURLSearchParamsState{rawQuery: normalizeBrowserURLSearchParamsRawQuery(rawQuery)}
	return browserURLSearchParamsValueFromState(state)
}

func browserURLSearchParamsValueFromState(state *browserURLSearchParamsState) script.Value {
	if state == nil {
		state = &browserURLSearchParamsState{}
	}

	var paramsValue script.Value
	paramsValue = script.ObjectValue([]script.ObjectEntry{
		{
			Key: "keys",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.keys expects no arguments")
				}
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				keys := make([]string, 0, len(pairs))
				for _, pair := range pairs {
					keys = append(keys, pair.key)
				}
				return browserURLSearchParamsKeysIteratorValue(keys), nil
			}),
		},
		{
			Key: "entries",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.entries expects no arguments")
				}
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				return browserURLSearchParamsEntriesIteratorValue(pairs), nil
			}),
		},
		{
			Key: "values",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.values expects no arguments")
				}
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				return browserURLSearchParamsValuesIteratorValue(pairs), nil
			}),
		},
		{
			Key: "forEach",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) == 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.forEach expects a callback")
				}
				if len(args) > 2 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.forEach accepts at most 2 arguments")
				}
				callback := args[0]
				thisArg := script.UndefinedValue()
				hasReceiver := false
				if len(args) > 1 {
					thisArg = args[1]
					hasReceiver = true
				}
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				for _, pair := range pairs {
					if _, err := script.InvokeCallableValue(script.CurrentInvokeHost(), callback, []script.Value{
						script.StringValue(pair.value),
						script.StringValue(pair.key),
						paramsValue,
					}, thisArg, hasReceiver); err != nil {
						return script.UndefinedValue(), err
					}
				}
				return script.UndefinedValue(), nil
			}),
		},
		{
			Key: "get",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 1 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.get expects 1 argument")
				}
				key := strings.TrimSpace(script.ToJSString(args[0]))
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				for _, pair := range pairs {
					if pair.key == key {
						return script.StringValue(pair.value), nil
					}
				}
				return script.NullValue(), nil
			}),
		},
		{
			Key: "getAll",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 1 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.getAll expects 1 argument")
				}
				key := strings.TrimSpace(script.ToJSString(args[0]))
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				values := make([]script.Value, 0)
				for _, pair := range pairs {
					if pair.key == key {
						values = append(values, script.StringValue(pair.value))
					}
				}
				return script.ArrayValue(values), nil
			}),
		},
		{
			Key: "has",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 1 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.has expects 1 argument")
				}
				key := strings.TrimSpace(script.ToJSString(args[0]))
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				for _, pair := range pairs {
					if pair.key == key {
						return script.BoolValue(true), nil
					}
				}
				return script.BoolValue(false), nil
			}),
		},
		{
			Key: "set",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 2 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.set expects 2 arguments")
				}
				key := strings.TrimSpace(script.ToJSString(args[0]))
				value := script.ToJSString(args[1])
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				filtered := make([]urlSearchParamPair, 0, len(pairs)+1)
				for _, pair := range pairs {
					if pair.key == key {
						continue
					}
					filtered = append(filtered, pair)
				}
				filtered = append(filtered, urlSearchParamPair{key: key, value: value})
				state.setPairs(filtered)
				return script.UndefinedValue(), nil
			}),
		},
		{
			Key: "append",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 2 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.append expects 2 arguments")
				}
				key := strings.TrimSpace(script.ToJSString(args[0]))
				value := script.ToJSString(args[1])
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				pairs = append(pairs, urlSearchParamPair{key: key, value: value})
				state.setPairs(pairs)
				return script.UndefinedValue(), nil
			}),
		},
		{
			Key: "delete",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 1 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.delete expects 1 argument")
				}
				key := strings.TrimSpace(script.ToJSString(args[0]))
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				filtered := make([]urlSearchParamPair, 0, len(pairs))
				for _, pair := range pairs {
					if pair.key == key {
						continue
					}
					filtered = append(filtered, pair)
				}
				state.setPairs(filtered)
				return script.UndefinedValue(), nil
			}),
		},
		{
			Key: "sort",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.sort expects no arguments")
				}
				pairs, err := state.snapshotPairs()
				if err != nil {
					return script.UndefinedValue(), err
				}
				sort.SliceStable(pairs, func(i, j int) bool {
					return pairs[i].key < pairs[j].key
				})
				state.setPairs(pairs)
				return script.UndefinedValue(), nil
			}),
		},
		{
			Key: "toString",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.toString expects no arguments")
				}
				return script.StringValue(state.rawQueryString()), nil
			}),
		},
	})
	return paramsValue
}

func browserURLSearchParamsConstructor(args []script.Value) (script.Value, error) {
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("URLSearchParams expects at most 1 argument")
	}
	rawQuery := ""
	if len(args) == 1 && args[0].Kind != script.ValueKindUndefined {
		rawQuery = script.ToJSString(args[0])
	}
	return browserURLSearchParamsValueFromRaw(rawQuery), nil
}

func normalizeBrowserURLSearchParamsRawQuery(rawQuery string) string {
	return strings.TrimPrefix(strings.TrimSpace(rawQuery), "?")
}

func browserURLSearchParamsKeysIteratorValue(keys []string) script.Value {
	index := 0
	return script.ObjectValue([]script.ObjectEntry{
		{
			Key: "next",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams iterator next expects no arguments")
				}
				if index >= len(keys) {
					return browserURLSearchParamsIteratorResult(script.UndefinedValue(), true), nil
				}
				current := keys[index]
				index++
				return browserURLSearchParamsIteratorResult(script.StringValue(current), false), nil
			}),
		},
	})
}

func browserURLSearchParamsEntriesIteratorValue(pairs []urlSearchParamPair) script.Value {
	index := 0
	return script.ObjectValue([]script.ObjectEntry{
		{
			Key: "next",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams iterator next expects no arguments")
				}
				if index >= len(pairs) {
					return browserURLSearchParamsIteratorResult(script.UndefinedValue(), true), nil
				}
				current := pairs[index]
				index++
				return browserURLSearchParamsIteratorResult(script.ArrayValue([]script.Value{
					script.StringValue(current.key),
					script.StringValue(current.value),
				}), false), nil
			}),
		},
	})
}

func browserURLSearchParamsValuesIteratorValue(pairs []urlSearchParamPair) script.Value {
	index := 0
	return script.ObjectValue([]script.ObjectEntry{
		{
			Key: "next",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams iterator next expects no arguments")
				}
				if index >= len(pairs) {
					return browserURLSearchParamsIteratorResult(script.UndefinedValue(), true), nil
				}
				current := pairs[index]
				index++
				return browserURLSearchParamsIteratorResult(script.StringValue(current.value), false), nil
			}),
		},
	})
}

func browserURLSearchParamsIteratorResult(value script.Value, done bool) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "value", Value: value},
		{Key: "done", Value: script.BoolValue(done)},
	})
}

func parseURLSearchParamPairs(rawQuery string) ([]urlSearchParamPair, error) {
	rawQuery = normalizeBrowserURLSearchParamsRawQuery(rawQuery)
	if rawQuery == "" {
		return nil, nil
	}
	parts := strings.FieldsFunc(rawQuery, func(r rune) bool {
		return r == '&' || r == ';'
	})
	out := make([]urlSearchParamPair, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		keyRaw, valueRaw, _ := strings.Cut(part, "=")
		key, err := neturl.QueryUnescape(keyRaw)
		if err != nil {
			return nil, fmt.Errorf("URLSearchParams: invalid escape in key %q: %w", keyRaw, err)
		}
		value, err := neturl.QueryUnescape(valueRaw)
		if err != nil {
			return nil, fmt.Errorf("URLSearchParams: invalid escape in value %q: %w", valueRaw, err)
		}
		out = append(out, urlSearchParamPair{key: key, value: value})
	}
	return out, nil
}

func serializeURLSearchParamPairs(pairs []urlSearchParamPair) string {
	if len(pairs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, pair := range pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(neturl.QueryEscape(pair.key))
		b.WriteByte('=')
		b.WriteString(neturl.QueryEscape(pair.value))
	}
	return b.String()
}
