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
		text, err := browserToStringValue(value)
		if err != nil {
			return err
		}
		if err := session.setWindowName(text); err != nil {
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
	if name == "scrollX" || name == "scrollY" {
		return true
	}
	if name == "crypto" {
		return false
	}
	_, reserved := browserGlobalBindings(session, store)[name]
	return reserved
}

func supportsPlaceholderAttribute(node *dom.Node) bool {
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
	case rest == "data":
		if node.Kind != dom.NodeKindText {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		return store.SetTextContent(nodeID, script.ToJSString(value))
	case rest == "nodeValue":
		if node.Kind != dom.NodeKindText {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		return store.SetTextContent(nodeID, script.ToJSString(value))
	case rest == "innerHTML":
		return store.SetInnerHTML(nodeID, script.ToJSString(value))
	case rest == "outerHTML":
		return store.SetOuterHTML(nodeID, script.ToJSString(value))
	case rest == "value":
		if node.TagName == "input" && inputType(node) == "file" {
			if script.ToJSString(value) != "" {
				return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("set_value is only supported on text-like inputs and textareas, not <input type=%q>", inputType(node)))
			}
			clearFileInputSelectionForNode(session, store, nodeID)
			return nil
		}
		if node.TagName == "select" {
			return store.SetSelectValue(nodeID, script.ToJSString(value))
		}
		if node.TagName == "option" {
			return store.SetAttribute(nodeID, "value", script.ToJSString(value))
		}
		return store.SetFormControlValue(nodeID, script.ToJSString(value))
	case rest == "placeholder":
		if !supportsPlaceholderAttribute(node) {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		return store.SetAttribute(nodeID, "placeholder", script.ToJSString(value))
	case rest == "files":
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
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
	case rest == "selectedIndex":
		if node.TagName != "select" {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		index, err := browserInt64Value("element.selectedIndex", value)
		if err != nil {
			return err
		}
		return store.SetSelectIndex(nodeID, int(index))
	case rest == "id":
		return store.SetAttribute(nodeID, "id", script.ToJSString(value))
	case rest == "style":
		return setElementStyleText(store, nodeID, script.ToJSString(value))
	case rest == "lang":
		return store.SetAttribute(nodeID, "lang", script.ToJSString(value))
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
	case rest == "files":
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
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

func browserElementHasAttributes(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.hasAttributes is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("element.hasAttributes accepts no arguments")
	}
	ok, err := store.HasAttributes(nodeID)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.BoolValue(ok), nil
}

func browserElementGetAttributeNames(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.getAttributeNames is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("element.getAttributeNames accepts no arguments")
	}
	names, err := store.GetAttributeNames(nodeID)
	if err != nil {
		return script.UndefinedValue(), err
	}
	values := make([]script.Value, 0, len(names))
	for _, name := range names {
		values = append(values, script.StringValue(name))
	}
	return script.ArrayValue(values), nil
}

func browserElementToggleAttribute(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element.toggleAttribute is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 {
		return script.UndefinedValue(), fmt.Errorf("element.toggleAttribute requires argument 1")
	}
	if len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("element.toggleAttribute accepts at most 2 arguments")
	}
	name, err := scriptStringArg("element.toggleAttribute", args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	force := false
	hasForce := false
	if len(args) == 2 {
		force = jsTruthyValue(args[1])
		hasForce = true
	}
	ok, err := store.ToggleAttribute(nodeID, name, force, hasForce)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.BoolValue(ok), nil
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

func browserNodeBefore(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.before is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 {
		return script.UndefinedValue(), nil
	}
	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.before is unavailable in this bounded classic-JS slice")
	}
	if node.Parent == 0 {
		return script.UndefinedValue(), nil
	}

	childIDs, createdIDs, err := browserNodeIDsFromValues(store, "node.before", args)
	if err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	if err := store.InsertNodeListBefore(nodeID, childIDs); err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserNodeAfter(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.after is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 {
		return script.UndefinedValue(), nil
	}
	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.after is unavailable in this bounded classic-JS slice")
	}
	if node.Parent == 0 {
		return script.UndefinedValue(), nil
	}

	childIDs, createdIDs, err := browserNodeIDsFromValues(store, "node.after", args)
	if err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	if err := store.InsertNodeListAfter(nodeID, childIDs); err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserNodeReplaceWith(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.replaceWith is unavailable in this bounded classic-JS slice")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.replaceWith is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 {
		if err := store.ReplaceNodeWithChildren(nodeID, nil); err != nil {
			return script.UndefinedValue(), err
		}
		return script.UndefinedValue(), nil
	}
	if node.Parent == 0 {
		return script.UndefinedValue(), nil
	}

	childIDs, createdIDs, err := browserNodeIDsFromValues(store, "node.replaceWith", args)
	if err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	if err := store.ReplaceNodeWithChildren(nodeID, childIDs); err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserNodeReplaceChildren(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "replaceChildren is unavailable in this bounded classic-JS slice")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || (node.Kind != dom.NodeKindElement && node.Kind != dom.NodeKindDocument) {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "replaceChildren is unavailable in this bounded classic-JS slice")
	}
	method := "element.replaceChildren"
	if node.Kind == dom.NodeKindDocument {
		method = "document.replaceChildren"
	}

	childIDs, createdIDs, err := browserNodeIDsFromValues(store, method, args)
	if err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	if err := store.ReplaceChildrenWithNodeIDs(nodeID, childIDs); err != nil {
		browserDeleteNodeIDs(store, createdIDs)
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserNodeContains(session *Session, store *dom.Store, nodeID dom.NodeID, method string, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("%s expects 1 argument", method)
	}
	if args[0].Kind == script.ValueKindNull || args[0].Kind == script.ValueKindUndefined {
		return script.BoolValue(false), nil
	}
	otherID, err := browserNodeIDFromContainsValue(store, args[0], method)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.BoolValue(store.ContainsNode(nodeID, otherID)), nil
}

func browserNodeCompareDocumentPosition(session *Session, store *dom.Store, nodeID dom.NodeID, method string, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("%s expects 1 argument", method)
	}
	otherID, err := browserNodeIDFromContainsValue(store, args[0], method)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.NumberValue(float64(store.CompareDocumentPosition(nodeID, otherID))), nil
}

func browserNodeHasChildNodes(session *Session, store *dom.Store, nodeID dom.NodeID, method string, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("%s accepts no arguments", method)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid node reference %q in this bounded classic-JS slice", method))
	}
	return script.BoolValue(len(node.Children) > 0), nil
}

func browserNodeGetRootNode(session *Session, store *dom.Store, nodeID dom.NodeID, method string, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("%s accepts no arguments", method)
	}
	rootID := store.RootNodeID(nodeID)
	if rootID == 0 {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	if rootID == store.DocumentID() {
		return script.HostObjectReference("document"), nil
	}
	return browserElementReferenceValue(rootID, store), nil
}

func browserNodeRemove(session *Session, store *dom.Store, nodeID dom.NodeID, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.remove is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("node.remove accepts no arguments")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "node.remove is unavailable in this bounded classic-JS slice")
	}
	if node.Parent == 0 {
		return script.UndefinedValue(), nil
	}
	if err := store.RemoveNode(nodeID); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
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
	if err := session.blurFocusedNodeIfNeeded(store, nodeID); err != nil {
		return script.UndefinedValue(), err
	}
	if session.domStore != nil && session.domStore != store {
		return script.UndefinedValue(), session.drainMicrotasks(session.domStore)
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

func browserNodeIDFromContainsValue(store *dom.Store, value script.Value, method string) (dom.NodeID, error) {
	if store == nil {
		return 0, script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("%s is unavailable in this bounded classic-JS slice", method))
	}
	if value.Kind == script.ValueKindHostReference {
		switch {
		case value.HostReferencePath == "document":
			return store.DocumentID(), nil
		case strings.HasPrefix(value.HostReferencePath, "element:"):
			return parseElementReferencePath(value.HostReferencePath)
		}
	}
	return 0, fmt.Errorf("%s requires a node reference", method)
}

func browserNodeIDsFromValues(store *dom.Store, method string, args []script.Value) ([]dom.NodeID, []dom.NodeID, error) {
	childIDs := make([]dom.NodeID, 0, len(args))
	createdIDs := make([]dom.NodeID, 0, len(args))
	for _, arg := range args {
		if arg.Kind == script.ValueKindHostReference {
			switch {
			case arg.HostReferencePath == "document":
				childIDs = append(childIDs, store.DocumentID())
				continue
			case strings.HasPrefix(arg.HostReferencePath, "element:"):
				nodeID, err := parseElementReferencePath(arg.HostReferencePath)
				if err != nil {
					browserDeleteNodeIDs(store, createdIDs)
					return nil, nil, err
				}
				childIDs = append(childIDs, nodeID)
				continue
			}
		}

		textID, err := store.CreateTextNode(script.ToJSString(arg))
		if err != nil {
			browserDeleteNodeIDs(store, createdIDs)
			return nil, nil, fmt.Errorf("%s failed to create text node: %w", method, err)
		}
		createdIDs = append(createdIDs, textID)
		childIDs = append(childIDs, textID)
	}
	return childIDs, createdIDs, nil
}

func browserNodeChildIDFromValue(store *dom.Store, method string, value script.Value) (dom.NodeID, bool, error) {
	if store == nil {
		return 0, false, script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("%s is unavailable in this bounded classic-JS slice", method))
	}
	if value.Kind == script.ValueKindHostReference {
		switch {
		case value.HostReferencePath == "document":
			return 0, false, script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("%s does not accept document nodes", method))
		case strings.HasPrefix(value.HostReferencePath, "element:"):
			nodeID, err := parseElementReferencePath(value.HostReferencePath)
			if err != nil {
				return 0, false, err
			}
			return nodeID, false, nil
		}
	}
	textID, err := store.CreateTextNode(script.ToJSString(value))
	if err != nil {
		return 0, false, fmt.Errorf("%s failed to create text node: %w", method, err)
	}
	return textID, true, nil
}

func browserNodeAppendValue(store *dom.Store, parentID dom.NodeID, method string, value script.Value, referenceChildID dom.NodeID) error {
	childID, created, err := browserNodeChildIDFromValue(store, method, value)
	if err != nil {
		return err
	}
	if referenceChildID == 0 {
		if err := store.AppendChild(parentID, childID); err != nil {
			if created {
				_ = store.DeleteNode(childID)
			}
			return err
		}
		return nil
	}
	if err := store.InsertBefore(parentID, childID, referenceChildID); err != nil {
		if created {
			_ = store.DeleteNode(childID)
		}
		return err
	}
	return nil
}

func browserNodeAppend(session *Session, store *dom.Store, nodeID dom.NodeID, method string, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || (node.Kind != dom.NodeKindElement && node.Kind != dom.NodeKindDocument) {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	for _, arg := range args {
		if err := browserNodeAppendValue(store, nodeID, method, arg, 0); err != nil {
			return script.UndefinedValue(), err
		}
	}
	return script.UndefinedValue(), nil
}

func browserNodePrepend(session *Session, store *dom.Store, nodeID dom.NodeID, method string, args []script.Value) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || (node.Kind != dom.NodeKindElement && node.Kind != dom.NodeKindDocument) {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, method+" is unavailable in this bounded classic-JS slice")
	}
	for i := len(args) - 1; i >= 0; i-- {
		referenceChildID := dom.NodeID(0)
		if len(node.Children) > 0 {
			referenceChildID = node.Children[0]
		}
		if err := browserNodeAppendValue(store, nodeID, method, args[i], referenceChildID); err != nil {
			return script.UndefinedValue(), err
		}
	}
	return script.UndefinedValue(), nil
}

func browserDeleteNodeIDs(store *dom.Store, nodeIDs []dom.NodeID) {
	if store == nil {
		return
	}
	for _, nodeID := range nodeIDs {
		_ = store.DeleteNode(nodeID)
	}
}

func browserHostObjectValue(path string) script.Value {
	return script.HostObjectReference(path)
}
