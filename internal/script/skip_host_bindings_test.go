package script

import (
	"fmt"
	"testing"
)

type skipHostBindingsReproHost struct {
	calls         []browserCall
	resolvedPaths []string
}

func (h *skipHostBindingsReproHost) Call(method string, args []Value) (Value, error) {
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

func (h *skipHostBindingsReproHost) ResolveHostReference(path string) (Value, error) {
	h.resolvedPaths = append(h.resolvedPaths, path)

	switch path {
	case "String":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return StringValue(""), nil
			}
			return StringValue(ToJSString(args[0])), nil
		}), nil
	case "Intl.NumberFormat":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return ObjectValue([]ObjectEntry{
				{
					Key: "format",
					Value: NativeFunctionValue(func(args []Value) (Value, error) {
						if len(args) != 1 {
							return UndefinedValue(), fmt.Errorf("Intl.NumberFormat#format expects 1 argument")
						}
						return StringValue("1.23"), nil
					}),
				},
			}), nil
		}), nil
	default:
		return UndefinedValue(), fmt.Errorf("host reference %q is not configured", path)
	}
}

func TestSkipHostBindingsPreservePureHostConstructorsInShortCircuit(t *testing.T) {
	cases := []struct {
		name         string
		source       string
		want         string
		wantResolved string
	}{
		{
			name:         "string-trim",
			source:       `host.echo("yes" || String(null).trim())`,
			want:         "yes",
			wantResolved: "String",
		},
		{
			name:         "intl-number-format",
			source:       `host.echo("yes" || new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23))`,
			want:         "yes",
			wantResolved: "Intl.NumberFormat",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := &skipHostBindingsReproHost{}
			runtime := NewRuntimeWithBindings(host, map[string]Value{
				"String": HostFunctionReference("String"),
				"Intl":   HostObjectReference("Intl"),
			})

			result, err := runtime.Dispatch(DispatchRequest{Source: tc.source})
			if err != nil {
				t.Fatalf("Dispatch(%s) error = %v", tc.name, err)
			}
			if result.Value.Kind != ValueKindString || result.Value.String != tc.want {
				t.Fatalf("Dispatch(%s) value = %#v, want string %q", tc.name, result.Value, tc.want)
			}
			if len(host.calls) != 1 {
				t.Fatalf("Dispatch(%s) host calls = %#v, want one echo call", tc.name, host.calls)
			}
			if host.calls[0].method != "echo" {
				t.Fatalf("Dispatch(%s) host.calls[0].method = %q, want echo", tc.name, host.calls[0].method)
			}
			if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != tc.want {
				t.Fatalf("Dispatch(%s) host.calls[0].args = %#v, want one %q string", tc.name, host.calls[0].args, tc.want)
			}
			found := false
			for _, path := range host.resolvedPaths {
				if path == tc.wantResolved {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Dispatch(%s) resolved paths = %#v, want %q to be resolved in skip mode", tc.name, host.resolvedPaths, tc.wantResolved)
			}
		})
	}
}
