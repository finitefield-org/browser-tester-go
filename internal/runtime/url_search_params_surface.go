package runtime

import (
	"fmt"
	neturl "net/url"
	"strings"

	"browsertester/internal/script"
)

type urlSearchParamPair struct {
	key   string
	value string
}

func browserURLSearchParamsValue(parsed *neturl.URL) script.Value {
	if parsed == nil {
		return browserURLSearchParamsValueFromRaw("")
	}
	return browserURLSearchParamsValueFromRaw(parsed.RawQuery)
}

func browserURLSearchParamsValueFromRaw(rawQuery string) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{
			Key: "keys",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.keys expects no arguments")
				}
				pairs, err := parseURLSearchParamPairs(rawQuery)
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
			Key: "toString",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 0 {
					return script.UndefinedValue(), fmt.Errorf("URLSearchParams.toString expects no arguments")
				}
				return script.StringValue(rawQuery), nil
			}),
		},
	})
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

func browserURLSearchParamsIteratorResult(value script.Value, done bool) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "value", Value: value},
		{Key: "done", Value: script.BoolValue(done)},
	})
}

func parseURLSearchParamPairs(rawQuery string) ([]urlSearchParamPair, error) {
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
