package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
)

func (s *Session) DumpDOM() string {
	if s == nil {
		return ""
	}
	store, err := s.ensureDOM()
	if err != nil || store == nil {
		return ""
	}
	return store.DumpDOM()
}

func (s *Session) SetValue(selector, value string) (err error) {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	store, nodeID, node, normalized, err := s.resolveActionTarget(selector)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			s.discardMicrotasks()
		}
	}()
	if node == nil || node.Kind != dom.NodeKindElement {
		return fmt.Errorf("selector `%s` does not reference a supported form control", normalized)
	}

	switch node.TagName {
	case "select":
		return store.SetSelectValue(nodeID, value)
	case "textarea":
		return store.SetFormControlValue(nodeID, value)
	case "input":
		if inputType(node) == "file" {
			if value != "" {
				return fmt.Errorf("set_value is only supported on text-like inputs, textareas, and selects, not <input type=%q>", inputType(node))
			}
			if err := store.SetFormControlValue(nodeID, value); err != nil {
				return err
			}
			clearFileInputSelectionForNode(s, store, nodeID)
			return nil
		}
		return store.SetFormControlValue(nodeID, value)
	default:
		return fmt.Errorf("selector `%s` does not reference a supported form control", normalized)
	}
}

func (s *Session) TypeText(selector, text string) (err error) {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	store, nodeID, _, normalized, err := s.resolveActionTarget(selector)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			s.discardMicrotasks()
		}
	}()
	if store.FocusedNodeID() != nodeID {
		if err := s.blurFocusedNodeIfNeeded(store, nodeID); err != nil {
			return err
		}
		if s.domStore != nil && s.domStore != store {
			return s.drainMicrotasks(s.domStore)
		}
		// Blur handlers may replace the target subtree, so re-resolve before writing.
		store, nodeID, _, normalized, err = s.resolveActionTarget(selector)
		if err != nil {
			return err
		}
		if err := s.focusNode(store, nodeID, normalized, false); err != nil {
			return err
		}
	}
	if err := store.SetFormControlValue(nodeID, text); err != nil {
		return err
	}
	if err := store.SetUserValidity(nodeID, true); err != nil {
		return err
	}
	s.recordInteraction(InteractionKindTypeText, normalized)
	if _, err := s.dispatchEventListeners(store, nodeID, "input"); err != nil {
		return err
	}
	return s.drainMicrotasks(store)
}

func (s *Session) SetChecked(selector string, checked bool) (err error) {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	store, nodeID, _, normalized, err := s.resolveActionTarget(selector)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			s.discardMicrotasks()
		}
	}()
	if err := store.SetFormControlChecked(nodeID, checked); err != nil {
		return err
	}
	if err := store.SetUserValidity(nodeID, true); err != nil {
		return err
	}
	s.recordInteraction(InteractionKindSetChecked, normalized)
	if _, err := s.dispatchEventListeners(store, nodeID, "input"); err != nil {
		return err
	}
	if _, err := s.dispatchEventListeners(store, nodeID, "change"); err != nil {
		return err
	}
	return s.drainMicrotasks(store)
}

func (s *Session) SetSelectValue(selector, value string) (err error) {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	store, nodeID, _, normalized, err := s.resolveActionTarget(selector)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			s.discardMicrotasks()
		}
	}()
	if err := store.SetSelectValue(nodeID, value); err != nil {
		return err
	}
	if err := store.SetUserValidity(nodeID, true); err != nil {
		return err
	}
	s.recordInteraction(InteractionKindSetSelectValue, normalized)
	if _, err := s.dispatchEventListeners(store, nodeID, "input"); err != nil {
		return err
	}
	if _, err := s.dispatchEventListeners(store, nodeID, "change"); err != nil {
		return err
	}
	return s.drainMicrotasks(store)
}

func (s *Session) Submit(selector string) (err error) {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	store, nodeID, node, normalized, err := s.resolveActionTarget(selector)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			s.discardMicrotasks()
		}
	}()
	formID, ok := submitTarget(store, nodeID, node)
	if !ok {
		return fmt.Errorf("selector `%s` does not reference a form or submit control with an owning form", normalized)
	}
	s.recordInteraction(InteractionKindSubmit, normalized)
	if _, err := s.dispatchEventListeners(store, formID, "submit"); err != nil {
		return err
	}
	return s.drainMicrotasks(store)
}

func (s *Session) applyClickDefaultAction(selector string) error {
	store, nodeID, node, normalized, err := s.resolveActionTarget(selector)
	if err != nil {
		return err
	}
	if node == nil || node.Kind != dom.NodeKindElement {
		return nil
	}

	switch node.TagName {
	case "input":
		switch inputType(node) {
		case "checkbox":
			checked := hasAttribute(node.Attrs, "checked")
			if err := store.SetFormControlChecked(nodeID, !checked); err != nil {
				return err
			}
			if err := store.SetUserValidity(nodeID, true); err != nil {
				return err
			}
			if _, err := s.dispatchEventListeners(store, nodeID, "input"); err != nil {
				return err
			}
			_, err := s.dispatchEventListeners(store, nodeID, "change")
			return err
		case "radio":
			if err := store.SetFormControlChecked(nodeID, true); err != nil {
				return err
			}
			if err := store.SetUserValidity(nodeID, true); err != nil {
				return err
			}
			if _, err := s.dispatchEventListeners(store, nodeID, "input"); err != nil {
				return err
			}
			_, err := s.dispatchEventListeners(store, nodeID, "change")
			return err
		case "submit", "image":
			if _, ok := submitTarget(store, nodeID, node); ok {
				return s.Submit(normalized)
			}
		case "reset":
			if formID, ok := resetTarget(store, nodeID, node); ok {
				prevented, err := s.dispatchTargetEventListeners(store, formID, "reset")
				if err != nil {
					return err
				}
				if prevented {
					return nil
				}
				return store.ResetFormControls(formID)
			}
		}
	case "button":
		if isSubmitControl(store, node) {
			if _, ok := submitTarget(store, nodeID, node); ok {
				return s.Submit(normalized)
			}
		} else if isResetControl(node) {
			if formID, ok := resetTarget(store, nodeID, node); ok {
				prevented, err := s.dispatchTargetEventListeners(store, formID, "reset")
				if err != nil {
					return err
				}
				if prevented {
					return nil
				}
				return store.ResetFormControls(formID)
			}
		}
	case "a", "area":
		return s.applyHyperlinkDefaultAction(node)
	case "summary":
		detailsID, err := s.applyDetailsSummaryDefaultAction(store, nodeID, node)
		if err != nil {
			return err
		}
		return s.dispatchDetailsToggleEvent(store, detailsID)
	}

	return nil
}

func (s *Session) applyClickDefaultActionForNode(store *dom.Store, nodeID dom.NodeID, node *dom.Node) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return fmt.Errorf("session is unavailable")
	}
	if node == nil || node.Kind != dom.NodeKindElement {
		return nil
	}

	switch node.TagName {
	case "input":
		switch inputType(node) {
		case "checkbox":
			checked := hasAttribute(node.Attrs, "checked")
			if err := store.SetFormControlChecked(nodeID, !checked); err != nil {
				return err
			}
			if err := store.SetUserValidity(nodeID, true); err != nil {
				return err
			}
			if _, err := s.dispatchEventListeners(store, nodeID, "input"); err != nil {
				return err
			}
			_, err := s.dispatchEventListeners(store, nodeID, "change")
			return err
		case "radio":
			if err := store.SetFormControlChecked(nodeID, true); err != nil {
				return err
			}
			if err := store.SetUserValidity(nodeID, true); err != nil {
				return err
			}
			if _, err := s.dispatchEventListeners(store, nodeID, "input"); err != nil {
				return err
			}
			_, err := s.dispatchEventListeners(store, nodeID, "change")
			return err
		case "submit", "image":
			if formID, ok := submitTarget(store, nodeID, node); ok {
				if _, err := s.dispatchEventListeners(store, formID, "submit"); err != nil {
					return err
				}
				return nil
			}
		case "reset":
			if formID, ok := resetTarget(store, nodeID, node); ok {
				prevented, err := s.dispatchTargetEventListeners(store, formID, "reset")
				if err != nil {
					return err
				}
				if prevented {
					return nil
				}
				return store.ResetFormControls(formID)
			}
		}
	case "button":
		if isSubmitControl(store, node) {
			if formID, ok := submitTarget(store, nodeID, node); ok {
				if _, err := s.dispatchEventListeners(store, formID, "submit"); err != nil {
					return err
				}
				return nil
			}
		} else if isResetControl(node) {
			if formID, ok := resetTarget(store, nodeID, node); ok {
				prevented, err := s.dispatchTargetEventListeners(store, formID, "reset")
				if err != nil {
					return err
				}
				if prevented {
					return nil
				}
				return store.ResetFormControls(formID)
			}
		}
	case "a", "area":
		return s.applyHyperlinkDefaultAction(node)
	case "summary":
		detailsID, err := s.applyDetailsSummaryDefaultAction(store, nodeID, node)
		if err != nil {
			return err
		}
		return s.dispatchDetailsToggleEvent(store, detailsID)
	}

	return nil
}

func (s *Session) applyDetailsSummaryDefaultAction(store *dom.Store, summaryNodeID dom.NodeID, node *dom.Node) (dom.NodeID, error) {
	if s == nil {
		return 0, fmt.Errorf("session is unavailable")
	}
	if store == nil || node == nil || node.Kind != dom.NodeKindElement || node.TagName != "summary" {
		return 0, nil
	}

	detailsID := summaryDetailsAncestorID(store, summaryNodeID)
	if detailsID == 0 {
		return 0, nil
	}
	if firstSummaryChildID(store, detailsID) != summaryNodeID {
		return 0, nil
	}

	details := store.Node(detailsID)
	if details == nil {
		return 0, nil
	}
	changed, err := s.setDetailsOpenState(store, detailsID, !hasAttribute(details.Attrs, "open"))
	if err != nil {
		return 0, err
	}
	if !changed {
		return 0, nil
	}
	return detailsID, nil
}

func (s *Session) setDetailsOpenState(store *dom.Store, detailsID dom.NodeID, open bool) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("session is unavailable")
	}
	if store == nil || detailsID == 0 {
		return false, nil
	}
	details := store.Node(detailsID)
	if details == nil || details.Kind != dom.NodeKindElement || details.TagName != "details" {
		return false, nil
	}
	present := hasAttribute(details.Attrs, "open")
	if present == open {
		return false, nil
	}
	if open {
		if err := store.SetAttribute(detailsID, "open", ""); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := store.RemoveAttribute(detailsID, "open"); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Session) dispatchDetailsToggleEvent(store *dom.Store, detailsID dom.NodeID) error {
	if detailsID == 0 {
		return nil
	}
	_, err := s.dispatchTargetEventListeners(store, detailsID, "toggle")
	return err
}

func summaryDetailsAncestorID(store *dom.Store, nodeID dom.NodeID) dom.NodeID {
	if store == nil || nodeID == 0 {
		return 0
	}
	current := nodeID
	for current != 0 {
		node := store.Node(current)
		if node == nil {
			return 0
		}
		if node.Kind == dom.NodeKindElement && node.TagName == "details" {
			return current
		}
		current = node.Parent
	}
	return 0
}

func firstSummaryChildID(store *dom.Store, detailsID dom.NodeID) dom.NodeID {
	if store == nil || detailsID == 0 {
		return 0
	}
	details := store.Node(detailsID)
	if details == nil {
		return 0
	}
	for _, childID := range details.Children {
		child := store.Node(childID)
		if child == nil || child.Kind != dom.NodeKindElement {
			continue
		}
		if child.TagName == "summary" {
			return childID
		}
	}
	return 0
}

func (s *Session) recordInteraction(kind InteractionKind, selector string) {
	s.interactions = append(s.interactions, Interaction{
		Kind:     kind,
		Selector: selector,
	})
}

func (s *Session) resolveActionTarget(selector string) (*dom.Store, dom.NodeID, *dom.Node, string, error) {
	normalized := strings.TrimSpace(selector)
	if normalized == "" {
		return nil, 0, nil, "", fmt.Errorf("selector must not be empty")
	}

	store, err := s.ensureDOM()
	if err != nil {
		return nil, 0, nil, "", err
	}

	ids, err := store.Select(normalized)
	if err != nil {
		return nil, 0, nil, "", err
	}
	if len(ids) == 0 {
		return nil, 0, nil, "", fmt.Errorf("selector `%s` did not match any element", normalized)
	}

	nodeID := ids[0]
	node := store.Node(nodeID)
	if node == nil {
		return nil, 0, nil, "", fmt.Errorf("selector `%s` did not match any element", normalized)
	}
	return store, nodeID, node, normalized, nil
}

func submitTarget(store *dom.Store, nodeID dom.NodeID, node *dom.Node) (dom.NodeID, bool) {
	if store == nil || node == nil || node.Kind != dom.NodeKindElement {
		return 0, false
	}

	if node.TagName == "form" {
		return nodeID, true
	}
	if !isSubmitControl(store, node) {
		return 0, false
	}

	current := node.Parent
	for current != 0 {
		parent := store.Node(current)
		if parent == nil {
			return 0, false
		}
		if parent.Kind == dom.NodeKindElement && parent.TagName == "form" {
			return current, true
		}
		current = parent.Parent
	}

	return 0, false
}

func inputType(node *dom.Node) string {
	return dom.InputType(node)
}

func isTextInputType(typeName string) bool {
	switch strings.ToLower(strings.TrimSpace(typeName)) {
	case "", "text", "search", "url", "tel", "email", "password", "number", "date", "datetime-local", "time", "month", "week", "color":
		return true
	default:
		return false
	}
}

func hasAttribute(attrs []dom.Attribute, name string) bool {
	for _, attr := range attrs {
		if attr.Name == name {
			return true
		}
	}
	return false
}

func isSubmitControl(store *dom.Store, node *dom.Node) bool {
	if node == nil || node.Kind != dom.NodeKindElement {
		return false
	}

	switch node.TagName {
	case "button":
		return dom.IsSubmitButton(store, node)
	case "input":
		switch inputType(node) {
		case "submit", "image":
			return true
		}
	}

	return false
}

func isResetControl(node *dom.Node) bool {
	if node == nil || node.Kind != dom.NodeKindElement {
		return false
	}

	switch node.TagName {
	case "button":
		typeName, _ := attributeValue(node.Attrs, "type")
		return strings.ToLower(strings.TrimSpace(typeName)) == "reset"
	case "input":
		return inputType(node) == "reset"
	}

	return false
}

func resetTarget(store *dom.Store, nodeID dom.NodeID, node *dom.Node) (dom.NodeID, bool) {
	if store == nil || node == nil || node.Kind != dom.NodeKindElement {
		return 0, false
	}

	if node.TagName == "form" {
		return nodeID, true
	}
	if !isResetControl(node) {
		return 0, false
	}

	current := node.Parent
	for current != 0 {
		parent := store.Node(current)
		if parent == nil {
			return 0, false
		}
		if parent.Kind == dom.NodeKindElement && parent.TagName == "form" {
			return current, true
		}
		current = parent.Parent
	}

	return 0, false
}

func attributeValue(attrs []dom.Attribute, name string) (string, bool) {
	for _, attr := range attrs {
		if attr.Name == name {
			return attr.Value, true
		}
	}
	return "", false
}
