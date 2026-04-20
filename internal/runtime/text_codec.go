package runtime

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"browsertester/internal/script"
)

const browserTextEncoderInstancePath = "textencoder"
const browserTextDecoderInstancePath = "textdecoder"

var (
	browserTextEncoderOnce sync.Once
	browserTextEncoderCtor script.Value
	browserTextDecoderOnce sync.Once
	browserTextDecoderCtor script.Value
)

func browserTextEncoderValue() script.Value {
	browserTextEncoderOnce.Do(func() {
		value := script.NativeConstructibleNamedFunctionValue("TextEncoder",
			func(args []script.Value) (script.Value, error) {
				return script.UndefinedValue(), fmt.Errorf("TextEncoder constructor must be called with `new` in this bounded classic-JS slice")
			},
			func(args []script.Value) (script.Value, error) {
				if len(args) > 0 {
					return script.UndefinedValue(), fmt.Errorf("TextEncoder constructor accepts no arguments in this bounded classic-JS slice")
				}
				return script.HostObjectReference(browserTextEncoderInstancePath), nil
			},
		)
		prototype := script.ObjectValue([]script.ObjectEntry{
			{Key: "constructor", Value: value},
			{Key: "encoding", Value: script.StringValue("utf-8")},
			{
				Key: "encode",
				Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
					return browserTextEncoderEncode(args)
				}),
			},
		})
		script.SetFunctionOwnProperty(value, "prototype", prototype)
		browserTextEncoderCtor = value
	})
	return browserTextEncoderCtor
}

func browserTextDecoderValue() script.Value {
	browserTextDecoderOnce.Do(func() {
		value := script.NativeConstructibleNamedFunctionValue("TextDecoder",
			func(args []script.Value) (script.Value, error) {
				return script.UndefinedValue(), fmt.Errorf("TextDecoder constructor must be called with `new` in this bounded classic-JS slice")
			},
			func(args []script.Value) (script.Value, error) {
				if len(args) > 2 {
					return script.UndefinedValue(), fmt.Errorf("TextDecoder constructor accepts at most 2 arguments in this bounded classic-JS slice")
				}
				label := "utf-8"
				if len(args) >= 1 {
					normalized, ok := canonicalTextDecoderLabel(script.ToJSString(args[0]))
					if !ok {
						return script.UndefinedValue(), fmt.Errorf("TextDecoder constructor only supports utf-8 or shift_jis in this bounded classic-JS slice")
					}
					label = normalized
				}
				return browserTextDecoderReferenceValue(label), nil
			},
		)
		prototype := script.ObjectValue([]script.ObjectEntry{
			{Key: "constructor", Value: value},
			{Key: "encoding", Value: script.StringValue("utf-8")},
			{Key: "fatal", Value: script.BoolValue(false)},
			{Key: "ignoreBOM", Value: script.BoolValue(false)},
			{
				Key: "decode",
				Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
					return browserTextDecoderDecode("utf-8", args)
				}),
			},
		})
		script.SetFunctionOwnProperty(value, "prototype", prototype)
		browserTextDecoderCtor = value
	})
	return browserTextDecoderCtor
}

func browserTextEncoderEncode(args []script.Value) (script.Value, error) {
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("TextEncoder.encode accepts at most 1 argument")
	}
	text := ""
	if len(args) == 1 && args[0].Kind != script.ValueKindUndefined && args[0].Kind != script.ValueKindNull {
		text = script.ToJSString(args[0])
	}
	return browserUint8ArrayValue([]byte(text)), nil
}

func browserTextDecoderReferenceValue(label string) script.Value {
	return script.HostObjectReference(browserTextDecoderInstancePath + "." + label)
}

func canonicalTextDecoderLabel(label string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(label))
	switch normalized {
	case "", "utf8", "utf-8":
		return "utf-8", true
	case "shift_jis", "shift-jis", "shiftjis":
		return "shift_jis", true
	default:
		return "", false
	}
}

func browserTextDecoderDecode(label string, args []script.Value) (script.Value, error) {
	if len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("TextDecoder.decode accepts at most 2 arguments")
	}
	if len(args) == 0 || args[0].Kind == script.ValueKindUndefined || args[0].Kind == script.ValueKindNull {
		return script.StringValue(""), nil
	}
	bytesValue, err := browserTextDecoderBytesFromValue(args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	switch label {
	case "shift_jis":
		decoder := transform.NewReader(bytes.NewReader(bytesValue), japanese.ShiftJIS.NewDecoder())
		decoded, err := io.ReadAll(decoder)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.StringValue(string(decoded)), nil
	default:
		return script.StringValue(string(bytes.ToValidUTF8(bytesValue, []byte("\uFFFD")))), nil
	}
}

func browserTextDecoderBytesFromValue(value script.Value) ([]byte, error) {
	switch value.Kind {
	case script.ValueKindArray, script.ValueKindObject:
		return browserUint8ArrayBytesFromValue(value)
	case script.ValueKindString:
		return []byte(value.String), nil
	default:
		return nil, fmt.Errorf("TextDecoder.decode expects a Uint8Array or array-like value")
	}
}

func resolveTextEncoderInstanceReference(path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "encode":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserTextEncoderEncode(args)
		}), nil
	case "encoding":
		return script.StringValue("utf-8"), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "textencoder"+path))
}

func resolveTextDecoderInstanceReference(path string) (script.Value, error) {
	labelPart := strings.TrimPrefix(strings.TrimSpace(path), ".")
	if labelPart == "" {
		labelPart = "utf-8"
	}
	label, suffix, found := strings.Cut(labelPart, ".")
	if !found {
		suffix = ""
	}
	normalized, ok := canonicalTextDecoderLabel(label)
	if !ok {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", browserTextDecoderInstancePath+path))
	}

	switch suffix {
	case "":
		return browserTextDecoderReferenceValue(normalized), nil
	case "decode":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserTextDecoderDecode(normalized, args)
		}), nil
	case "encoding":
		return script.StringValue(normalized), nil
	case "fatal":
		return script.BoolValue(false), nil
	case "ignoreBOM":
		return script.BoolValue(false), nil
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("TextDecoder.%s accepts no arguments", suffix)
			}
			return script.StringValue("[object TextDecoder]"), nil
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", browserTextDecoderInstancePath+path))
}
