package runtime

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func browserImageConstructor(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	return browserConstructImageLikeElement(session, store, "img", "Image", args)
}

func browserCanvasConstructor(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	return browserConstructImageLikeElement(session, store, "canvas", "HTMLCanvasElement", args)
}

func browserConstructImageLikeElement(session *Session, store *dom.Store, tagName string, method string, args []script.Value) (script.Value, error) {
	if store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("%s is unavailable in this bounded classic-JS slice", method))
	}
	if len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("%s expects at most 2 arguments", method)
	}
	nodeID, err := store.CreateElement(tagName)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if len(args) >= 1 {
		width, err := scriptInt64Arg(method, args, 0)
		if err != nil {
			return script.UndefinedValue(), err
		}
		if err := store.SetAttribute(nodeID, "width", strconv.FormatInt(width, 10)); err != nil {
			return script.UndefinedValue(), err
		}
	}
	if len(args) >= 2 {
		height, err := scriptInt64Arg(method, args, 1)
		if err != nil {
			return script.UndefinedValue(), err
		}
		if err := store.SetAttribute(nodeID, "height", strconv.FormatInt(height, 10)); err != nil {
			return script.UndefinedValue(), err
		}
	}
	return browserElementReferenceValue(nodeID, store), nil
}

func browserCanvasContextValue(session *Session, store *dom.Store, nodeID dom.NodeID) script.Value {
	entries := []script.ObjectEntry{
		{
			Key:   "canvas",
			Value: browserElementReferenceValue(nodeID, store),
		},
		{
			Key: "drawImage",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				switch len(args) {
				case 3, 5, 9:
					return script.UndefinedValue(), nil
				default:
					return script.UndefinedValue(), fmt.Errorf("canvas.drawImage expects 3, 5, or 9 arguments")
				}
			}),
		},
	}
	for _, name := range []string{
		"beginPath",
		"closePath",
		"clearRect",
		"fillRect",
		"strokeRect",
		"save",
		"restore",
		"fill",
		"stroke",
		"clip",
		"moveTo",
		"lineTo",
		"translate",
		"scale",
		"rotate",
		"setTransform",
		"resetTransform",
		"putImageData",
	} {
		entries = append(entries, script.ObjectEntry{
			Key: name,
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				return script.UndefinedValue(), nil
			}),
		})
	}
	_ = session
	return script.ObjectValue(entries)
}

func browserCanvasGetContextValue(session *Session, store *dom.Store, nodeID dom.NodeID) script.Value {
	return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if store == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "canvas.getContext is unavailable in this bounded classic-JS slice")
		}
		if len(args) == 0 {
			return script.UndefinedValue(), fmt.Errorf("canvas.getContext requires argument 1")
		}
		if len(args) > 2 {
			return script.UndefinedValue(), fmt.Errorf("canvas.getContext accepts at most 2 arguments")
		}
		contextType := strings.ToLower(strings.TrimSpace(script.ToJSString(args[0])))
		switch contextType {
		case "2d":
			return browserCanvasContextValue(session, store, nodeID), nil
		default:
			return script.NullValue(), nil
		}
	})
}

func browserCanvasToBlobValue(session *Session, store *dom.Store, nodeID dom.NodeID) script.Value {
	return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if session == nil || store == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "canvas.toBlob is unavailable in this bounded classic-JS slice")
		}
		if len(args) == 0 {
			return script.UndefinedValue(), fmt.Errorf("canvas.toBlob requires a callback")
		}
		if len(args) > 3 {
			return script.UndefinedValue(), fmt.Errorf("canvas.toBlob accepts at most 3 arguments")
		}
		if args[0].Kind != script.ValueKindFunction || (args[0].NativeFunction == nil && args[0].Function == nil) {
			return script.UndefinedValue(), fmt.Errorf("canvas.toBlob callback must be callable")
		}
		mimeType := "image/png"
		if len(args) >= 2 {
			mimeType = strings.ToLower(strings.TrimSpace(script.ToJSString(args[1])))
			if mimeType == "" {
				mimeType = "image/png"
			}
		}
		if mimeType != "image/png" {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "canvas.toBlob supports image/png only in this bounded classic-JS slice")
		}
		pngBytes, err := browserCanvasPNGBytes(store, nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		blobID := session.allocateBrowserBlobState(pngBytes, "image/png")
		if strings.TrimSpace(blobID) == "" {
			return script.UndefinedValue(), fmt.Errorf("canvas.toBlob could not allocate blob state")
		}
		blobValue := browserBlobReferenceValue(blobID)
		session.enqueueCallableMicrotask(args[0], []script.Value{blobValue}, browserElementReferenceValue(nodeID, store), true)
		return script.UndefinedValue(), nil
	})
}

func browserCanvasToDataURLValue(session *Session, store *dom.Store, nodeID dom.NodeID) script.Value {
	return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if store == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "canvas.toDataURL is unavailable in this bounded classic-JS slice")
		}
		if len(args) > 2 {
			return script.UndefinedValue(), fmt.Errorf("canvas.toDataURL accepts at most 2 arguments")
		}
		mimeType := "image/png"
		if len(args) >= 1 {
			mimeType = strings.ToLower(strings.TrimSpace(script.ToJSString(args[0])))
			if mimeType == "" {
				mimeType = "image/png"
			}
		}
		if mimeType != "image/png" {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "canvas.toDataURL supports image/png only in this bounded classic-JS slice")
		}
		pngBytes, err := browserCanvasPNGBytes(store, nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.StringValue("data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)), nil
	})
}

func browserCanvasPNGBytes(store *dom.Store, nodeID dom.NodeID) ([]byte, error) {
	width, height := browserCanvasPNGDimensions(store, nodeID)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func browserCanvasPNGDimensions(store *dom.Store, nodeID dom.NodeID) (int, int) {
	width := browserElementDimensionValue(store, nodeID, "width", 300)
	height := browserElementDimensionValue(store, nodeID, "height", 150)
	return width, height
}

func browserElementDimensionValue(store *dom.Store, nodeID dom.NodeID, name string, fallback int) int {
	value, ok := domAttributeValue(store, nodeID, name)
	if !ok {
		return fallback
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return fallback
	}
	return int(parsed)
}
