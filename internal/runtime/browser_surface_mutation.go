package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func (h *inlineScriptHost) SetHostReference(path string, value script.Value) error {
	store := h.currentStore()
	if h == nil || store == nil {
		return fmt.Errorf("inline script host is unavailable")
	}
	return setBrowserHostReferenceValue(h.session, store, path, value)
}

func (h *inlineScriptHost) DeleteHostReference(path string) error {
	store := h.currentStore()
	if h == nil || store == nil {
		return fmt.Errorf("inline script host is unavailable")
	}
	return deleteBrowserHostReferenceValue(h.session, store, path)
}

func setBrowserHostReferenceValue(session *Session, store *dom.Store, path string, value script.Value) error {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return script.NewError(script.ErrorKindUnsupported, "unsupported browser surface \"\" in this bounded classic-JS slice")
	}

	if strings.HasPrefix(normalized, browserWindowPropertyReferencePrefix) {
		return session.setWindowPropertyReference(normalized, value)
	}

	for _, prefix := range []string{"window.", "self.", "globalThis.", "top.", "parent.", "frames."} {
		if strings.HasPrefix(normalized, prefix) {
			return setBrowserHostReferenceValue(session, store, normalized[len(prefix):], value)
		}
	}

	switch normalized {
	case "window", "self", "globalThis", "top", "parent", "frames":
		return script.NewError(script.ErrorKindUnsupported, "window is read-only in this bounded classic-JS slice")
	case "name":
		if session == nil {
			return script.NewError(script.ErrorKindUnsupported, "window.name is unavailable in this bounded classic-JS slice")
		}
		if err := session.setWindowName(script.ToJSString(value)); err != nil {
			return err
		}
		return nil
	case "Intl":
		if session == nil {
			return script.NewError(script.ErrorKindUnsupported, "Intl is unavailable in this bounded classic-JS slice")
		}
		session.setIntlOverride(value)
		return nil
	}

	if strings.HasPrefix(normalized, "url:") {
		return setURLInstanceReferenceValue(session, normalized, value)
	}

	if strings.HasPrefix(normalized, "element:") {
		return setElementReferenceValue(session, store, normalized, value)
	}

	if isReservedWindowPropertyName(session, store, normalized) {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
	if session != nil {
		session.setWindowProperty(normalized, value)
		return nil
	}

	return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
}

func deleteBrowserHostReferenceValue(session *Session, store *dom.Store, path string) error {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return script.NewError(script.ErrorKindUnsupported, "unsupported browser surface \"\" in this bounded classic-JS slice")
	}

	if strings.HasPrefix(normalized, browserWindowPropertyReferencePrefix) {
		return session.deleteWindowPropertyReference(normalized)
	}

	for _, prefix := range []string{"window.", "self.", "globalThis.", "top.", "parent.", "frames."} {
		if strings.HasPrefix(normalized, prefix) {
			return deleteBrowserHostReferenceValue(session, store, normalized[len(prefix):])
		}
	}

	switch normalized {
	case "window", "self", "globalThis", "top", "parent", "frames":
		return script.NewError(script.ErrorKindUnsupported, "window is read-only in this bounded classic-JS slice")
	case "name":
		return script.NewError(script.ErrorKindUnsupported, "deletion of window.name is unsupported in this bounded classic-JS slice")
	}

	if strings.HasPrefix(normalized, "element:") {
		return deleteElementReferenceValue(session, store, normalized)
	}

	if isReservedWindowPropertyName(session, store, normalized) {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
	if session != nil && session.deleteWindowProperty(normalized) {
		return nil
	}

	return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
}

func isReservedWindowPropertyName(session *Session, store *dom.Store, name string) bool {
	if name == "" {
		return false
	}
	if name == "crypto" {
		return false
	}
	_, reserved := browserGlobalBindings(session, store)[name]
	return reserved
}

func setElementReferenceValue(session *Session, store *dom.Store, path string, value script.Value) error {
	nodeID, rest, err := splitElementReferencePath(path)
	if err != nil {
		return err
	}
	if store == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid element reference %q in this bounded classic-JS slice", path))
	}
	switch {
	case rest == "":
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	case rest == "href":
		if !supportsHyperlinkHref(node.TagName) {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		return store.SetAttribute(nodeID, "href", script.ToJSString(value))
	case rest == "download":
		if !supportsHyperlinkHref(node.TagName) {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		return store.SetAttribute(nodeID, "download", script.ToJSString(value))
	case rest == "className":
		return store.SetAttribute(nodeID, "class", script.ToJSString(value))
	case rest == "textContent" || rest == "innerText" || rest == "outerText":
		return store.SetTextContent(nodeID, script.ToJSString(value))
	case rest == "innerHTML":
		return store.SetInnerHTML(nodeID, script.ToJSString(value))
	case rest == "outerHTML":
		return store.SetOuterHTML(nodeID, script.ToJSString(value))
	case rest == "value":
		if node.TagName == "select" {
			return store.SetSelectValue(nodeID, script.ToJSString(value))
		}
		if node.TagName == "option" {
			return store.SetAttribute(nodeID, "value", script.ToJSString(value))
		}
		return store.SetFormControlValue(nodeID, script.ToJSString(value))
	case rest == "checked":
		if value.Kind != script.ValueKindBool {
			return fmt.Errorf("element.checked expects a boolean in this bounded classic-JS slice")
		}
		return store.SetFormControlChecked(nodeID, value.Bool)
	case rest == "disabled":
		if !supportsDisabledAttribute(node.TagName) {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		if value.Kind != script.ValueKindBool {
			return fmt.Errorf("element.disabled expects a boolean in this bounded classic-JS slice")
		}
		if value.Bool {
			return store.SetAttribute(nodeID, "disabled", "")
		}
		return store.RemoveAttribute(nodeID, "disabled")
	case rest == "open":
		if value.Kind != script.ValueKindBool {
			return fmt.Errorf("element.open expects a boolean in this bounded classic-JS slice")
		}
		if value.Bool {
			return store.SetAttribute(nodeID, "open", "")
		}
		return store.RemoveAttribute(nodeID, "open")
	case rest == "selected":
		if node.TagName != "option" {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		if value.Kind != script.ValueKindBool {
			return fmt.Errorf("element.selected expects a boolean in this bounded classic-JS slice")
		}
		if value.Bool {
			return store.SetAttribute(nodeID, "selected", "")
		}
		return store.RemoveAttribute(nodeID, "selected")
	case rest == "id":
		return store.SetAttribute(nodeID, "id", script.ToJSString(value))
	case rest == "style":
		return setElementStyleText(store, nodeID, script.ToJSString(value))
	case strings.HasPrefix(rest, "style."):
		return setElementStylePropertyValue(store, nodeID, strings.TrimPrefix(rest, "style."), script.ToJSString(value))
	case rest == "dataset":
		return script.NewError(script.ErrorKindUnsupported, "assignment to element.dataset is unsupported in this bounded classic-JS slice")
	case strings.HasPrefix(rest, "dataset."):
		dataset, err := store.Dataset(nodeID)
		if err != nil {
			return err
		}
		if err := dataset.Set(strings.TrimPrefix(rest, "dataset."), script.ToJSString(value)); err != nil {
			return err
		}
		return nil
	default:
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
}

func deleteElementReferenceValue(session *Session, store *dom.Store, path string) error {
	nodeID, rest, err := splitElementReferencePath(path)
	if err != nil {
		return err
	}
	if store == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid element reference %q in this bounded classic-JS slice", path))
	}
	switch {
	case rest == "":
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	case rest == "dataset":
		return script.NewError(script.ErrorKindUnsupported, "deletion of element.dataset is unsupported in this bounded classic-JS slice")
	case strings.HasPrefix(rest, "dataset."):
		dataset, err := store.Dataset(nodeID)
		if err != nil {
			return err
		}
		if err := dataset.Remove(strings.TrimPrefix(rest, "dataset.")); err != nil {
			return err
		}
		return nil
	default:
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
}

func browserConfirm(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "confirm is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("confirm expects at most 1 argument")
	}
	message := ""
	if len(args) == 1 {
		message = script.ToJSString(args[0])
	}
	ok, err := session.Confirm(message)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.BoolValue(ok), nil
}

func browserPrompt(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "prompt is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("prompt expects at most 2 arguments")
	}
	message := ""
	if len(args) >= 1 {
		message = script.ToJSString(args[0])
	}
	value, submitted, err := session.Prompt(message)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if !submitted {
		return script.NullValue(), nil
	}
	return script.StringValue(value), nil
}

func browserWindowAddEventListener(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	return browserRegisterEventListener(session, store, browserHostObjectValue("window"), args, "window.addEventListener")
}

func browserWindowRemoveEventListener(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	return browserRemoveRegisteredEventListener(session, store, browserHostObjectValue("window"), args, "window.removeEventListener")
}

func browserDocumentAddEventListener(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	return browserRegisterEventListener(session, store, browserHostObjectValue("document"), args, "document.addEventListener")
}

func browserDocumentRemoveEventListener(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	return browserRemoveRegisteredEventListener(session, store, browserHostObjectValue("document"), args, "document.removeEventListener")
}

func browserDocumentExecCommand(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document.execCommand is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("document.execCommand expects 1 argument")
	}
	ok, err := session.execCommand(script.ToJSString(args[0]))
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.BoolValue(ok), nil
}

func browserElementAddEventListener(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	return browserRegisterEventListener(session, store, browserElementReferenceValue(nodeID, store), args, "element.addEventListener")
}

func browserElementRemoveEventListener(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	return browserRemoveRegisteredEventListener(session, store, browserElementReferenceValue(nodeID, store), args, "element.removeEventListener")
}

func browserElementSetAttribute(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.setAttribute is unavailable in this bounded classic-JS slice")
	}
	if len(args) < 2 {
		return script.UndefinedValue(), fmt.Errorf("element.setAttribute requires 2 arguments")
	}
	if len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("element.setAttribute accepts at most 2 arguments")
	}
	name, err := scriptStringArg("element.setAttribute", args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if err := store.SetAttribute(nodeID, name, script.ToJSString(args[1])); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserElementRemoveAttribute(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.removeAttribute is unavailable in this bounded classic-JS slice")
	}
	name, err := scriptStringArg("element.removeAttribute", args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("element.removeAttribute accepts at most 1 argument")
	}
	if err := store.RemoveAttribute(nodeID, name); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserElementAppendChild(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.appendChild is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("element.appendChild expects 1 argument")
	}
	childID, err := browserNodeIDFromValue(args[0], "element.appendChild")
	if err != nil {
		return script.UndefinedValue(), err
	}
	if err := store.AppendChild(nodeID, childID); err != nil {
		return script.UndefinedValue(), err
	}
	return browserElementReferenceValue(childID, store), nil
}

func browserElementRemoveChild(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.removeChild is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("element.removeChild expects 1 argument")
	}
	childID, err := browserNodeIDFromValue(args[0], "element.removeChild")
	if err != nil {
		return script.UndefinedValue(), err
	}
	if err := store.RemoveChild(nodeID, childID); err != nil {
		return script.UndefinedValue(), err
	}
	return browserElementReferenceValue(childID, store), nil
}

func browserElementSelect(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.select is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("element.select accepts no arguments")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.select is unavailable in this bounded classic-JS slice")
	}
	session.setSelectedText(store.ValueForNode(nodeID))
	return script.UndefinedValue(), nil
}

func browserElementClick(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.click is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("element.click accepts no arguments")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.click is unavailable in this bounded classic-JS slice")
	}

	prevented, err := session.dispatchEventListeners(store, nodeID, "click")
	if err != nil {
		return script.UndefinedValue(), err
	}
	if session.domStore != nil && session.domStore != store {
		return script.UndefinedValue(), session.drainMicrotasks(session.domStore)
	}
	if prevented {
		return script.UndefinedValue(), session.drainMicrotasks(store)
	}
	if err := session.applyClickDefaultActionForNode(store, nodeID, node); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), session.drainMicrotasks(store)
}

func browserElementFocus(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.focus is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("element.focus accepts no arguments")
	}
	if err := session.focusElementNode(store, nodeID); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserElementBlur(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.blur is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("element.blur accepts no arguments")
	}
	if err := session.blurElementNode(store, nodeID); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserRegisterEventListener(session *Session, store *dom.Store, currentTarget script.Value, args []script.Value, method string) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	if len(args) < 2 {
		return script.UndefinedValue(), fmt.Errorf("%s requires 2 arguments", method)
	}
	if len(args) > 3 {
		return script.UndefinedValue(), fmt.Errorf("%s accepts at most 3 arguments", method)
	}
	eventType, err := scriptStringArg(method, args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	listener := args[1]
	phase, once, err := browserEventListenerOptions(method, args[2:])
	if err != nil {
		return script.UndefinedValue(), err
	}
	nodeID, err := browserEventListenerNodeID(store, currentTarget)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if err := session.registerEventListenerValue(nodeID, currentTarget, eventType, listener, phase, once); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserRemoveRegisteredEventListener(session *Session, store *dom.Store, currentTarget script.Value, args []script.Value, method string) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	if len(args) < 2 {
		return script.UndefinedValue(), fmt.Errorf("%s requires 2 arguments", method)
	}
	if len(args) > 3 {
		return script.UndefinedValue(), fmt.Errorf("%s accepts at most 3 arguments", method)
	}
	eventType, err := scriptStringArg(method, args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	listener := args[1]
	phase, _, err := browserEventListenerOptions(method, args[2:])
	if err != nil {
		return script.UndefinedValue(), err
	}
	nodeID, err := browserEventListenerNodeID(store, currentTarget)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if _, err := session.removeEventListenerValue(nodeID, currentTarget, eventType, listener, phase); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserEventListenerNodeID(store *dom.Store, currentTarget script.Value) (dom.NodeID, error) {
	if store == nil {
		return 0, fmt.Errorf("event listener target must be available")
	}
	switch currentTarget.Kind {
	case script.ValueKindHostReference:
		if currentTarget.HostReferencePath == "document" || currentTarget.HostReferencePath == "window" {
			return store.DocumentID(), nil
		}
		return parseElementReferencePath(currentTarget.HostReferencePath)
	default:
		return 0, fmt.Errorf("event listener target must be a browser host reference")
	}
}

func browserEventListenerOptions(method string, args []script.Value) (string, bool, error) {
	if len(args) == 0 {
		return string(eventPhaseBubble), false, nil
	}
	if len(args) > 1 {
		return "", false, fmt.Errorf("%s accepts at most 3 arguments", method)
	}
	value := args[0]
	switch value.Kind {
	case script.ValueKindBool:
		if value.Bool {
			return string(eventPhaseCapture), false, nil
		}
		return string(eventPhaseBubble), false, nil
	case script.ValueKindNull, script.ValueKindUndefined:
		return string(eventPhaseBubble), false, nil
	case script.ValueKindObject:
		capture := false
		once := false
		if prop, ok := objectProperty(value, "capture"); ok && prop.Kind == script.ValueKindBool {
			capture = prop.Bool
		}
		if prop, ok := objectProperty(value, "once"); ok && prop.Kind == script.ValueKindBool {
			once = prop.Bool
		}
		if capture {
			return string(eventPhaseCapture), once, nil
		}
		return string(eventPhaseBubble), once, nil
	default:
		return "", false, fmt.Errorf("%s listener options must be a boolean or object", method)
	}
}

func browserNodeIDFromValue(value script.Value, method string) (dom.NodeID, error) {
	if value.Kind == script.ValueKindHostReference && strings.HasPrefix(value.HostReferencePath, "element:") {
		return parseElementReferencePath(value.HostReferencePath)
	}
	return 0, fmt.Errorf("%s requires a node reference", method)
}

func browserHostObjectValue(path string) script.Value {
	return script.HostObjectReference(path)
}
