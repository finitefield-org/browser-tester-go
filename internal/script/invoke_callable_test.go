package script

import (
	"fmt"
	"testing"
)

type callableCaptureHost struct {
	stored   Value
	captured string
}

func (h *callableCaptureHost) Call(method string, args []Value) (Value, error) {
	switch method {
	case "store":
		if len(args) != 1 {
			return UndefinedValue(), fmt.Errorf("store expects 1 argument")
		}
		h.stored = args[0]
		return UndefinedValue(), nil
	case "capture":
		if len(args) != 1 {
			return UndefinedValue(), fmt.Errorf("capture expects 1 argument")
		}
		h.captured = ToJSString(args[0])
		return UndefinedValue(), nil
	default:
		return UndefinedValue(), fmt.Errorf("host method %q is not configured", method)
	}
}

func TestInvokeCallableValuePreservesLexicalStateAcrossDeferredInvocation(t *testing.T) {
	host := &callableCaptureHost{}
	runtime := NewRuntimeWithBindings(host, nil)

	if _, err := runtime.Dispatch(DispatchRequest{Source: `let order = ""; host.store(() => { order += "x"; host.capture(order); });`}); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if host.stored.Kind != ValueKindFunction {
		t.Fatalf("stored callback kind = %q, want function", host.stored.Kind)
	}

	if _, err := InvokeCallableValue(host, host.stored, nil, HostObjectReference("element:1"), true); err != nil {
		t.Fatalf("InvokeCallableValue() error = %v", err)
	}
	if got, want := host.captured, "x"; got != want {
		t.Fatalf("captured after deferred callback = %q, want %q", got, want)
	}

	if _, err := InvokeCallableValue(host, host.stored, nil, HostObjectReference("element:1"), true); err != nil {
		t.Fatalf("InvokeCallableValue() second error = %v", err)
	}
	if got, want := host.captured, "xx"; got != want {
		t.Fatalf("captured after second deferred callback = %q, want %q", got, want)
	}
}
