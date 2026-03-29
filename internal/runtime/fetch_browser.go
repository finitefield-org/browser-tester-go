package runtime

import (
	"fmt"

	"browsertester/internal/script"
)

func resolveFetchReference(session *Session) (script.Value, error) {
	return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "fetch is unavailable in this bounded classic-JS slice")
		}
		url, err := browserRequiredStringArg("fetch", args, 0)
		if err != nil {
			return script.UndefinedValue(), err
		}
		normalizedURL, status, body, err := session.Fetch(url)
		if err != nil {
			return script.RejectedPromiseValue(script.StringValue(fmt.Sprintf("fetch(%q) failed: %v", url, err))), nil
		}
		return browserFetchResponseValue(normalizedURL, status, body), nil
	}), nil
}

func browserFetchResponseValue(url string, status int, body string) script.Value {
	ok := status >= 200 && status <= 299

	return script.PromiseValue(script.ObjectValue([]script.ObjectEntry{
		{Key: "url", Value: script.StringValue(url)},
		{Key: "status", Value: script.NumberValue(float64(status))},
		{Key: "ok", Value: script.BoolValue(ok)},
		{Key: "text", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("Response.text accepts no arguments")
			}
			return script.PromiseValue(script.StringValue(body)), nil
		})},
		{Key: "json", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("Response.json accepts no arguments")
			}
			parsed, err := browserJSONParse([]script.Value{script.StringValue(body)})
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.PromiseValue(parsed), nil
		})},
	}))
}
