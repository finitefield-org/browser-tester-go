package script

import (
	"fmt"
	"strings"
	"testing"
)

type browserCall struct {
	method string
	args   []Value
}

type browserBootstrapHost struct {
	calls            []browserCall
	resolvedPaths    []string
	documentLookups  []string
	localStorage     map[string]string
	sessionStorage   map[string]string
	clipboardWrites  []string
	matchMediaCalls  []string
	timerSources     []string
	microtaskSources []string
	historyURL       string
}

func (h *browserBootstrapHost) Call(method string, args []Value) (Value, error) {
	copiedArgs := make([]Value, len(args))
	copy(copiedArgs, args)
	h.calls = append(h.calls, browserCall{method: method, args: copiedArgs})

	switch method {
	case "echo":
		if len(args) != 1 {
			return UndefinedValue(), fmt.Errorf("echo expects 1 argument")
		}
		return args[0], nil
	default:
		return UndefinedValue(), fmt.Errorf("host method %q is not configured", method)
	}
}

func (h *browserBootstrapHost) ResolveHostReference(path string) (Value, error) {
	h.resolvedPaths = append(h.resolvedPaths, path)

	switch path {
	case "document.getElementById":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("document.getElementById expects 1 argument")
			}
			id := ToJSString(args[0])
			h.documentLookups = append(h.documentLookups, id)
			if id != "agri-unit-converter-root" {
				return NullValue(), nil
			}
			return ObjectValue([]ObjectEntry{
				{Key: "textContent", Value: StringValue("root")},
			}), nil
		}), nil
	case "window.location":
		return ObjectValue([]ObjectEntry{
			{Key: "href", Value: StringValue("https://example.test/app?mode=initial")},
			{Key: "search", Value: StringValue("?mode=initial")},
		}), nil
	case "window.history":
		return ObjectValue([]ObjectEntry{
			{Key: "replaceState", Value: NativeFunctionValue(func(args []Value) (Value, error) {
				if len(args) != 3 {
					return UndefinedValue(), fmt.Errorf("history.replaceState expects 3 arguments")
				}
				h.historyURL = ToJSString(args[2])
				return UndefinedValue(), nil
			})},
		}), nil
	case "navigator.onLine":
		return BoolValue(true), nil
	case "URL":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("URL expects 1 argument")
			}
			href := ToJSString(args[0])
			return ObjectValue([]ObjectEntry{
				{Key: "href", Value: StringValue(href)},
				{Key: "toString", Value: NativeFunctionValue(func(args []Value) (Value, error) {
					return StringValue(href), nil
				})},
			}), nil
		}), nil
	case "Intl.NumberFormat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return ObjectValue([]ObjectEntry{
				{Key: "format", Value: NativeFunctionValue(func(args []Value) (Value, error) {
					if len(args) != 1 {
						return UndefinedValue(), fmt.Errorf("Intl.NumberFormat#format expects 1 argument")
					}
					return StringValue("1.23"), nil
				})},
			}), nil
		}), nil
	case "localStorage.setItem":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 2 {
				return UndefinedValue(), fmt.Errorf("localStorage.setItem expects 2 arguments")
			}
			if h.localStorage == nil {
				h.localStorage = map[string]string{}
			}
			h.localStorage[ToJSString(args[0])] = ToJSString(args[1])
			return UndefinedValue(), nil
		}), nil
	case "sessionStorage.setItem":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 2 {
				return UndefinedValue(), fmt.Errorf("sessionStorage.setItem expects 2 arguments")
			}
			if h.sessionStorage == nil {
				h.sessionStorage = map[string]string{}
			}
			h.sessionStorage[ToJSString(args[0])] = ToJSString(args[1])
			return UndefinedValue(), nil
		}), nil
	case "matchMedia":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("matchMedia expects 1 argument")
			}
			query := ToJSString(args[0])
			h.matchMediaCalls = append(h.matchMediaCalls, query)
			return ObjectValue([]ObjectEntry{
				{Key: "matches", Value: BoolValue(true)},
				{Key: "media", Value: StringValue(query)},
			}), nil
		}), nil
	case "clipboard.writeText":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("clipboard.writeText expects 1 argument")
			}
			h.clipboardWrites = append(h.clipboardWrites, ToJSString(args[0]))
			return PromiseValue(UndefinedValue()), nil
		}), nil
	case "setTimeout":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 2 {
				return UndefinedValue(), fmt.Errorf("setTimeout expects 2 arguments")
			}
			h.timerSources = append(h.timerSources, ToJSString(args[0]))
			return NumberValue(1), nil
		}), nil
	case "queueMicrotask":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return UndefinedValue(), fmt.Errorf("queueMicrotask expects 1 argument")
			}
			h.microtaskSources = append(h.microtaskSources, ToJSString(args[0]))
			return UndefinedValue(), nil
		}), nil
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
}

func TestDispatchSkipsUnsupportedBrowserSurfaceInUntakenBranch(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"window":         HostObjectReference("window"),
		"document":       HostObjectReference("document"),
		"navigator":      HostObjectReference("navigator"),
		"localStorage":   HostObjectReference("localStorage"),
		"sessionStorage": HostObjectReference("sessionStorage"),
		"matchMedia":     HostFunctionReference("matchMedia"),
		"clipboard":      HostObjectReference("clipboard"),
		"URL":            HostConstructorReference("URL"),
		"Intl":           HostObjectReference("Intl"),
		"setTimeout":     HostFunctionReference("setTimeout"),
		"queueMicrotask": HostFunctionReference("queueMicrotask"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `
			const root = document.getElementById("agri-unit-converter-root");
			host.echo(root.textContent.length)
		`,
	})
	if err != nil {
		t.Fatalf("Dispatch(raw browser globals) error = %v", err)
	}
	if result.Value.Kind != ValueKindNumber || result.Value.Number != 4 {
		t.Fatalf("Dispatch(raw browser globals) value = %#v, want number 4", result.Value)
	}
	if len(host.documentLookups) != 1 || host.documentLookups[0] != "agri-unit-converter-root" {
		t.Fatalf("document.getElementById calls = %#v, want the agri root lookup", host.documentLookups)
	}
}

func TestDispatchSupportsNewBrowserConstructorsWithMemberChains(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"window": HostObjectReference("window"),
		"URL":    HostConstructorReference("URL"),
		"Intl":   HostObjectReference("Intl"),
	})

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const current = new URL(window.location.href); const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23); host.echo(current.href); host.echo(formatted)`,
	})
	if err != nil {
		t.Fatalf("Dispatch(new browser constructors) error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "1.23" {
		t.Fatalf("Dispatch(new browser constructors) value = %#v, want formatted string", result.Value)
	}
	if len(host.calls) != 2 {
		t.Fatalf("host calls = %#v, want two echo calls", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "https://example.test/app?mode=initial" {
		t.Fatalf("host.calls[0].args = %#v, want initial URL string", host.calls[0].args)
	}
	if host.calls[1].method != "echo" {
		t.Fatalf("host.calls[1].method = %q, want echo", host.calls[1].method)
	}
	if len(host.calls[1].args) != 1 || host.calls[1].args[0].Kind != ValueKindString || host.calls[1].args[0].String != "1.23" {
		t.Fatalf("host.calls[1].args = %#v, want formatted number", host.calls[1].args)
	}
}

func TestDispatchReportsUnsupportedBrowserSurfaceDirectly(t *testing.T) {
	host := &browserBootstrapHost{}
	runtime := NewRuntimeWithBindings(host, map[string]Value{
		"window": HostObjectReference("window"),
	})

	_, err := runtime.Dispatch(DispatchRequest{Source: `window.crypto.randomUUID()`})
	if err == nil {
		t.Fatalf("Dispatch(window.crypto.randomUUID()) error = nil, want unsupported error")
	}
	scriptErr, ok := err.(Error)
	if !ok {
		t.Fatalf("Dispatch(window.crypto.randomUUID()) error type = %T, want script.Error", err)
	}
	if scriptErr.Kind != ErrorKindUnsupported {
		t.Fatalf("Dispatch(window.crypto.randomUUID()) error kind = %q, want %q", scriptErr.Kind, ErrorKindUnsupported)
	}
	if !strings.Contains(scriptErr.Message, "window.crypto") {
		t.Fatalf("Dispatch(window.crypto.randomUUID()) error = %q, want browser-surface path", scriptErr.Message)
	}
}
