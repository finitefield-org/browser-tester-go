package script

import (
	"fmt"
	"math"
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
	case "Array":
		return NativeConstructibleNamedFunctionValue("Array", func(args []Value) (Value, error) {
			return skipHostBindingsArrayConstructor(args)
		}, func(args []Value) (Value, error) {
			return skipHostBindingsArrayConstructor(args)
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

func skipHostBindingsArrayConstructor(args []Value) (Value, error) {
	if len(args) == 1 && args[0].Kind == ValueKindNumber {
		if math.IsNaN(args[0].Number) || math.IsInf(args[0].Number, 0) {
			return UndefinedValue(), fmt.Errorf("Array length must be a finite number")
		}
		if math.Trunc(args[0].Number) != args[0].Number {
			return UndefinedValue(), fmt.Errorf("Array length must be an integer")
		}
		if args[0].Number < 0 {
			return UndefinedValue(), fmt.Errorf("Array length must be non-negative")
		}
		return ArrayValue(make([]Value, int(args[0].Number))), nil
	}
	return ArrayValue(args), nil
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
			name:         "array-constructor",
			source:       `host.echo("yes" || Array(1, 2).length)`,
			want:         "yes",
			wantResolved: "Array",
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

func TestSkipHostBindingsDoNotMutateSharedBindingsInShortCircuit(t *testing.T) {
	env := newClassicJSEnvironment()
	shared := ObjectValue([]ObjectEntry{{Key: "flag", Value: StringValue("safe")}})
	if err := env.declare("shared", scalarJSValue(shared), true); err != nil {
		t.Fatalf("declare(shared) error = %v", err)
	}

	host := &skipHostBindingsReproHost{}
	result, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(
		`host.echo("keep" || (shared.flag = "boom"))`,
		host,
		env,
		DefaultRuntimeConfig().StepLimit,
		false,
		false,
		false,
		nil,
		UndefinedValue(),
		false,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports() error = %v", err)
	}
	if result.Kind != ValueKindString || result.String != "keep" {
		t.Fatalf("result = %#v, want string %q", result, "keep")
	}
	if len(host.calls) != 1 || host.calls[0].method != "echo" {
		t.Fatalf("host calls = %#v, want one echo call", host.calls)
	}

	binding, ok := env.lookup("shared")
	if !ok || binding.kind != jsValueScalar || binding.value.Kind != ValueKindObject {
		t.Fatalf("shared binding = %#v, want object", binding)
	}
	flag, ok := lookupObjectProperty(binding.value.Object, "flag")
	if !ok || flag.Kind != ValueKindString || flag.String != "safe" {
		t.Fatalf("shared.flag = %#v, want %q", flag, "safe")
	}
}

func TestSkipHostBindingsDoNotExecuteSkippedArrowFunctionBodies(t *testing.T) {
	host := &skipHostBindingsReproHost{}
	runtime := NewRuntimeWithBindings(host, nil)

	result, err := runtime.Dispatch(DispatchRequest{Source: `host.echo("keep" || (() => host.echo("boom"))())`})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.Value.Kind != ValueKindString || result.Value.String != "keep" {
		t.Fatalf("Dispatch() value = %#v, want string %q", result.Value, "keep")
	}
	if len(host.calls) != 1 {
		t.Fatalf("host calls = %#v, want one echo call", host.calls)
	}
	if host.calls[0].method != "echo" {
		t.Fatalf("host.calls[0].method = %q, want echo", host.calls[0].method)
	}
	if len(host.calls[0].args) != 1 || host.calls[0].args[0].Kind != ValueKindString || host.calls[0].args[0].String != "keep" {
		t.Fatalf("host.calls[0].args = %#v, want one %q string", host.calls[0].args, "keep")
	}
}
