package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
	"browsertester/internal/script/jsregex"
)

func resolveElementClassNameValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError("element.className")
	}
	value, ok := domAttributeValue(store, nodeID, "class")
	if !ok {
		return script.StringValue(""), nil
	}
	return script.StringValue(value), nil
}

func resolveElementInnerTextValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError("element.innerText")
	}
	return script.StringValue(store.TextContentForNode(nodeID)), nil
}

func resolveElementOuterTextValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError("element.outerText")
	}
	return script.StringValue(store.TextContentForNode(nodeID)), nil
}

func resolveElementOpenValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError("element.open")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError("element.open")
	}
	_, ok := domAttributeValue(store, nodeID, "open")
	return script.BoolValue(ok), nil
}

func resolveElementHrefValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".href"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || !supportsHyperlinkHref(node.TagName) {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	href, ok := domAttributeValue(store, nodeID, "href")
	if !ok {
		return script.StringValue(""), nil
	}
	return script.StringValue(resolveHyperlinkURL(session.URL(), href)), nil
}

func resolveElementDownloadValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".download"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || !supportsHyperlinkDownload(node.TagName) {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	value, ok := domAttributeValue(store, nodeID, "download")
	if !ok {
		return script.StringValue(""), nil
	}
	return script.StringValue(value), nil
}

func resolveElementTabIndexValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".tabIndex"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || !supportsTabIndexAttribute(node.TagName) {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	value, ok := domAttributeValue(store, nodeID, "tabindex")
	if !ok {
		return script.NumberValue(0), nil
	}
	index, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return script.NumberValue(0), nil
	}
	return script.NumberValue(float64(index)), nil
}

func resolveElementPlaceholderValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".placeholder"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || !supportsPlaceholderAttribute(node) {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	value, ok := domAttributeValue(store, nodeID, "placeholder")
	if !ok {
		return script.StringValue(""), nil
	}
	return script.StringValue(value), nil
}

func resolveElementTypeValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".type"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	switch node.TagName {
	case "button":
		return script.StringValue(dom.ButtonType(store, node)), nil
	case "input":
		return script.StringValue(dom.InputType(node)), nil
	case "select":
		return script.StringValue(dom.SelectType(node)), nil
	default:
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
}

func resolveElementLangValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".lang"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	value, ok := domAttributeValue(store, nodeID, "lang")
	if !ok {
		return script.StringValue(""), nil
	}
	return script.StringValue(value), nil
}

func resolveElementDirValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".dir"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	value, ok := domAttributeValue(store, nodeID, "dir")
	if !ok {
		return script.StringValue(""), nil
	}
	return script.StringValue(strings.ToLower(strings.TrimSpace(value))), nil
}

func resolveElementDisabledValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".disabled"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || !supportsDisabledAttribute(node.TagName) {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	_, ok := domAttributeValue(store, nodeID, "disabled")
	return script.BoolValue(ok), nil
}

func resolveElementSelectedValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".selected"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || node.TagName != "option" {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	_, ok := domAttributeValue(store, nodeID, "selected")
	return script.BoolValue(ok), nil
}

func resolveElementSelectedIndexValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".selectedIndex"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || node.TagName != "select" {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	return script.NumberValue(float64(store.SelectedIndexForNode(nodeID))), nil
}

func resolveElementSelectionStartValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".selectionStart"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if !supportsTextSelectionNode(node) {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	return script.NumberValue(float64(jsregex.UTF16Length(store.ValueForNode(nodeID)))), nil
}

func resolveElementSelectionEndValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".selectionEnd"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if !supportsTextSelectionNode(node) {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	return script.NumberValue(float64(jsregex.UTF16Length(store.ValueForNode(nodeID)))), nil
}

func resolveElementStylePropertyValue(session *Session, store *dom.Store, nodeID dom.NodeID, property string) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".style"
	if property != "" {
		surface += "." + property
	}
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}

	switch property {
	case "cssText":
		return script.StringValue(elementStyleText(store, nodeID)), nil
	case "length":
		return script.NumberValue(float64(len(elementStyleDeclarations(store, nodeID)))), nil
	case "item":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("element.style.item expects 1 argument")
			}
			index, err := browserInt64Value("element.style.item", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			declarations := elementStyleDeclarations(store, nodeID)
			if index < 0 || int(index) >= len(declarations) {
				return script.NullValue(), nil
			}
			return script.StringValue(declarations[index].name), nil
		}), nil
	case "getPropertyValue":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			name, err := scriptStringArg("element.style.getPropertyValue", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.StringValue(elementStylePropertyValue(store, nodeID, name)), nil
		}), nil
	case "setProperty":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) < 2 {
				return script.UndefinedValue(), fmt.Errorf("element.style.setProperty requires 2 or 3 arguments")
			}
			if len(args) > 3 {
				return script.UndefinedValue(), fmt.Errorf("element.style.setProperty accepts at most 3 arguments")
			}
			name, err := scriptStringArg("element.style.setProperty", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if normalizeStylePropertyName(name) == "csstext" {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.style.setProperty does not support cssText")
			}
			priority := ""
			if len(args) == 3 {
				priority = strings.ToLower(strings.TrimSpace(script.ToJSString(args[2])))
				if priority != "" && priority != "important" {
					return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.style.setProperty priority must be empty or \"important\"")
				}
			}
			if err := setElementStylePropertyValueWithPriority(store, nodeID, name, script.ToJSString(args[1]), priority); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "removeProperty":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			name, err := scriptStringArg("element.style.removeProperty", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if len(args) > 1 {
				return script.UndefinedValue(), fmt.Errorf("element.style.removeProperty accepts at most 1 argument")
			}
			if normalizeStylePropertyName(name) == "csstext" {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.style.removeProperty does not support cssText")
			}
			value, err := removeElementStylePropertyValue(store, nodeID, name)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.StringValue(value), nil
		}), nil
	case "getPropertyPriority":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			name, err := scriptStringArg("element.style.getPropertyPriority", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if len(args) > 1 {
				return script.UndefinedValue(), fmt.Errorf("element.style.getPropertyPriority accepts at most 1 argument")
			}
			if normalizeStylePropertyName(name) == "csstext" {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.style.getPropertyPriority does not support cssText")
			}
			return script.StringValue(elementStylePropertyPriority(store, nodeID, name)), nil
		}), nil
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("element.style.%s accepts no arguments", property)
			}
			return script.StringValue(elementStyleText(store, nodeID)), nil
		}), nil
	default:
		return script.StringValue(elementStylePropertyValue(store, nodeID, property)), nil
	}
}

func resolveElementAttributesPropertyValue(session *Session, store *dom.Store, nodeID dom.NodeID, property string) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".attributes"
	if property != "" {
		surface += "." + property
	}
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}

	switch property {
	case "length":
		return script.NumberValue(float64(len(node.Attrs))), nil
	case "item":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("element.attributes.item expects 1 argument")
			}
			index, err := browserInt64Value("element.attributes.item", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			return elementAttributeByIndexValue(nodeID, node.Attrs, int(index)), nil
		}), nil
	case "namedItem", "getNamedItem":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			name, err := scriptStringArg("element.attributes."+property, args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return elementAttributeByNameValue(nodeID, node.Attrs, name), nil
		}), nil
	case "toString":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("element.attributes.toString accepts no arguments")
			}
			return script.StringValue("[object NamedNodeMap]"), nil
		}), nil
	case "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("element.attributes.valueOf accepts no arguments")
			}
			return script.StringValue("[object NamedNodeMap]"), nil
		}), nil
	default:
		if index, err := strconv.Atoi(property); err == nil {
			return elementAttributeByIndexValue(nodeID, node.Attrs, index), nil
		}
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
}

func resolveElementClassListPropertyValue(session *Session, store *dom.Store, nodeID dom.NodeID, property string) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".classList"
	if property != "" {
		surface += "." + property
	}
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	classList, err := store.ClassList(nodeID)
	if err != nil {
		return script.UndefinedValue(), err
	}

	switch property {
	case "":
		return script.HostObjectReference("element:" + strconv.FormatInt(int64(nodeID), 10) + ".classList"), nil
	case "length":
		return script.NumberValue(float64(len(classList.Values()))), nil
	case "item":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("element.classList.item expects 1 argument")
			}
			index, err := browserInt64Value("element.classList.item", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			token, ok := classList.Item(int(index))
			if !ok {
				return script.NullValue(), nil
			}
			return script.StringValue(token), nil
		}), nil
	case "contains":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			token, err := scriptStringArg(surface+".contains", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.BoolValue(classList.Contains(token)), nil
		}), nil
	case "add":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.UndefinedValue(), nil
			}
			tokens := make([]string, 0, len(args))
			for _, arg := range args {
				tokens = append(tokens, script.ToJSString(arg))
			}
			if err := classList.Add(tokens...); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "remove":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.UndefinedValue(), nil
			}
			tokens := make([]string, 0, len(args))
			for _, arg := range args {
				tokens = append(tokens, script.ToJSString(arg))
			}
			if err := classList.Remove(tokens...); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "toggle":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			token, err := scriptStringArg(surface+".toggle", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			enabled := classList.Contains(token)
			if len(args) > 1 {
				force := jsTruthyValue(args[1])
				if force {
					if err := classList.Add(token); err != nil {
						return script.UndefinedValue(), err
					}
					return script.BoolValue(true), nil
				}
				if err := classList.Remove(token); err != nil {
					return script.UndefinedValue(), err
				}
				return script.BoolValue(false), nil
			}
			if enabled {
				if err := classList.Remove(token); err != nil {
					return script.UndefinedValue(), err
				}
				return script.BoolValue(false), nil
			}
			if err := classList.Add(token); err != nil {
				return script.UndefinedValue(), err
			}
			return script.BoolValue(true), nil
		}), nil
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("element.classList.%s accepts no arguments", property)
			}
			return script.StringValue(strings.Join(classList.Values(), " ")), nil
		}), nil
	default:
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
}

func resolveElementDatasetPropertyValue(session *Session, store *dom.Store, nodeID dom.NodeID, property string) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".dataset"
	if property != "" {
		surface += "." + property
	}
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	if node := nodeFromStore(store, nodeID); node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	if property == "" {
		return script.HostObjectReference("element:" + strconv.FormatInt(int64(nodeID), 10) + ".dataset"), nil
	}
	dataset, err := store.Dataset(nodeID)
	if err != nil {
		return script.UndefinedValue(), err
	}
	value, ok := dataset.Get(property)
	if !ok {
		return script.UndefinedValue(), nil
	}
	return script.StringValue(value), nil
}

func supportsTextSelectionNode(node *dom.Node) bool {
	if node == nil || node.Kind != dom.NodeKindElement {
		return false
	}
	switch node.TagName {
	case "textarea":
		return true
	case "input":
		return isTextInputType(inputType(node))
	default:
		return false
	}
}

func supportsDisabledAttribute(tagName string) bool {
	switch tagName {
	case "button", "fieldset", "input", "optgroup", "option", "select", "textarea":
		return true
	default:
		return false
	}
}

func supportsHyperlinkHref(tagName string) bool {
	switch tagName {
	case "a", "area", "link":
		return true
	default:
		return false
	}
}

func supportsHyperlinkDownload(tagName string) bool {
	switch tagName {
	case "a", "area":
		return true
	default:
		return false
	}
}

func supportsTabIndexAttribute(tagName string) bool {
	switch tagName {
	case "button", "input", "select", "textarea":
		return true
	default:
		return false
	}
}

func unsupportedElementSurfaceError(surface string) error {
	return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", surface))
}

type styleDeclaration struct {
	name     string
	value    string
	priority string
}

func elementStyleText(store *dom.Store, nodeID dom.NodeID) string {
	if store == nil {
		return ""
	}
	value, ok := domAttributeValue(store, nodeID, "style")
	if !ok {
		return ""
	}
	return value
}

func elementStyleDeclarations(store *dom.Store, nodeID dom.NodeID) []styleDeclaration {
	text := elementStyleText(store, nodeID)
	if text == "" {
		return nil
	}
	parts := splitStyleDeclarations(text)
	if len(parts) == 0 {
		return nil
	}
	declarations := make([]styleDeclaration, 0, len(parts))
	for _, part := range parts {
		declaration, ok := parseStyleDeclaration(part)
		if !ok {
			continue
		}
		declarations = append(declarations, declaration)
	}
	return declarations
}

func parseStyleDeclaration(part string) (styleDeclaration, bool) {
	colon := strings.IndexByte(part, ':')
	if colon <= 0 {
		return styleDeclaration{}, false
	}
	name := strings.ToLower(strings.TrimSpace(part[:colon]))
	value, priority := splitStyleDeclarationValueAndPriority(strings.TrimSpace(part[colon+1:]))
	if name == "" {
		return styleDeclaration{}, false
	}
	return styleDeclaration{name: name, value: value, priority: priority}, true
}

func splitStyleDeclarationValueAndPriority(value string) (string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ""
	}
	const importantSuffix = "!important"
	if len(trimmed) < len(importantSuffix) {
		return trimmed, ""
	}
	if !strings.HasSuffix(strings.ToLower(trimmed), importantSuffix) {
		return trimmed, ""
	}
	if !styleTextBalanced(trimmed[:len(trimmed)-len(importantSuffix)]) {
		return trimmed, ""
	}
	prefix := strings.TrimSpace(trimmed[:len(trimmed)-len(importantSuffix)])
	if prefix == "" {
		return trimmed, ""
	}
	return prefix, "important"
}

func styleTextBalanced(text string) bool {
	var quote byte
	var escape bool
	var parenDepth int
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if quote != 0 {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			quote = ch
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		}
	}
	return quote == 0 && parenDepth == 0 && !escape
}

func splitStyleDeclarations(input string) []string {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil
	}

	parts := make([]string, 0, 4)
	start := 0
	var quote byte
	var escape bool
	var parenDepth int
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if quote != 0 {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			quote = ch
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case ';':
			if parenDepth == 0 {
				part := strings.TrimSpace(text[start:i])
				if part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
	}

	if tail := strings.TrimSpace(text[start:]); tail != "" {
		parts = append(parts, tail)
	}
	return parts
}

func elementStylePropertyValue(store *dom.Store, nodeID dom.NodeID, property string) string {
	normalized := normalizeStylePropertyName(property)
	if normalized == "" {
		return ""
	}
	declarations := elementStyleDeclarations(store, nodeID)
	for i := len(declarations) - 1; i >= 0; i-- {
		if declarations[i].name == normalized {
			return declarations[i].value
		}
	}
	return ""
}

func elementStylePropertyPriority(store *dom.Store, nodeID dom.NodeID, property string) string {
	normalized := normalizeStylePropertyName(property)
	if normalized == "" {
		return ""
	}
	declarations := elementStyleDeclarations(store, nodeID)
	for i := len(declarations) - 1; i >= 0; i-- {
		if declarations[i].name == normalized {
			return declarations[i].priority
		}
	}
	return ""
}

func setElementStylePropertyValue(store *dom.Store, nodeID dom.NodeID, property, value string) error {
	return setElementStylePropertyValueWithPriority(store, nodeID, property, value, "")
}

func setElementStylePropertyValueWithPriority(store *dom.Store, nodeID dom.NodeID, property, value, priority string) error {
	normalized := normalizeStylePropertyName(property)
	if normalized == "" {
		return fmt.Errorf("style property name must not be empty")
	}
	if store == nil {
		return fmt.Errorf("dom store is nil")
	}
	node := store.Node(nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return fmt.Errorf("node %d is not an element", nodeID)
	}
	if normalized == "csstext" {
		return store.SetAttribute(nodeID, "style", value)
	}
	declarations := elementStyleDeclarations(store, nodeID)
	next := make([]styleDeclaration, 0, len(declarations)+1)
	for _, declaration := range declarations {
		if declaration.name == normalized {
			continue
		}
		next = append(next, declaration)
	}
	next = append(next, styleDeclaration{name: normalized, value: value, priority: normalizeStylePriority(priority)})
	return store.SetAttribute(nodeID, "style", serializeStyleDeclarations(next))
}

func removeElementStylePropertyValue(store *dom.Store, nodeID dom.NodeID, property string) (string, error) {
	normalized := normalizeStylePropertyName(property)
	if normalized == "" {
		return "", fmt.Errorf("style property name must not be empty")
	}
	if store == nil {
		return "", fmt.Errorf("dom store is nil")
	}
	node := store.Node(nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return "", fmt.Errorf("node %d is not an element", nodeID)
	}
	if normalized == "csstext" {
		return "", fmt.Errorf("style property name must not be cssText")
	}
	declarations := elementStyleDeclarations(store, nodeID)
	next := make([]styleDeclaration, 0, len(declarations))
	removed := ""
	found := false
	for _, declaration := range declarations {
		if declaration.name == normalized {
			removed = declaration.value
			found = true
			continue
		}
		next = append(next, declaration)
	}
	if !found {
		return "", nil
	}
	if err := store.SetAttribute(nodeID, "style", serializeStyleDeclarations(next)); err != nil {
		return "", err
	}
	return removed, nil
}

func setElementStyleText(store *dom.Store, nodeID dom.NodeID, text string) error {
	if store == nil {
		return fmt.Errorf("dom store is nil")
	}
	node := store.Node(nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return fmt.Errorf("node %d is not an element", nodeID)
	}
	return store.SetAttribute(nodeID, "style", text)
}

func normalizeStylePropertyName(name string) string {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "--") {
		return strings.ToLower(normalized)
	}

	var b strings.Builder
	b.Grow(len(normalized) + 4)
	for i := 0; i < len(normalized); i++ {
		ch := normalized[i]
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 && normalized[i-1] != '-' {
				b.WriteByte('-')
			}
			b.WriteByte(ch + ('a' - 'A'))
			continue
		}
		b.WriteByte(ch)
	}
	return strings.ToLower(b.String())
}

func normalizeStylePriority(priority string) string {
	normalized := strings.ToLower(strings.TrimSpace(priority))
	switch normalized {
	case "", "important":
		return normalized
	default:
		return normalized
	}
}

func serializeStyleDeclarations(declarations []styleDeclaration) string {
	if len(declarations) == 0 {
		return ""
	}
	parts := make([]string, 0, len(declarations))
	for _, declaration := range declarations {
		name := strings.TrimSpace(declaration.name)
		if name == "" {
			continue
		}
		value := strings.TrimSpace(declaration.value)
		if declaration.priority != "" {
			parts = append(parts, name+": "+value+" !"+strings.TrimSpace(declaration.priority))
			continue
		}
		parts = append(parts, name+": "+value)
	}
	return strings.Join(parts, "; ")
}

func browserAttributeObjectValue(nodeID dom.NodeID, attr dom.Attribute) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "name", Value: script.StringValue(attr.Name)},
		{Key: "nodeName", Value: script.StringValue(attr.Name)},
		{Key: "localName", Value: script.StringValue(attr.Name)},
		{Key: "value", Value: script.StringValue(attr.Value)},
		{Key: "specified", Value: script.BoolValue(true)},
		{Key: "ownerElement", Value: browserElementReferenceValue(nodeID)},
		{Key: "toString", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("Attr.toString accepts no arguments")
			}
			return script.StringValue(attr.Value), nil
		})},
		{Key: "valueOf", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("Attr.valueOf accepts no arguments")
			}
			return script.StringValue(attr.Value), nil
		})},
	})
}

func elementAttributeByIndexValue(nodeID dom.NodeID, attrs []dom.Attribute, index int) script.Value {
	if index < 0 || index >= len(attrs) {
		return script.NullValue()
	}
	return browserAttributeObjectValue(nodeID, attrs[index])
}

func elementAttributeByNameValue(nodeID dom.NodeID, attrs []dom.Attribute, name string) script.Value {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return script.NullValue()
	}
	for _, attr := range attrs {
		if attr.Name == normalized {
			return browserAttributeObjectValue(nodeID, attr)
		}
	}
	return script.NullValue()
}
