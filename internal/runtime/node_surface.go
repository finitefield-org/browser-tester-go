package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func resolveNodeTreeNavigationValue(store *dom.Store, nodeID dom.NodeID, surface, property string) (script.Value, bool, error) {
	if store == nil {
		return script.UndefinedValue(), true, script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", surface))
	}
	node := store.Node(nodeID)
	if node == nil {
		return script.UndefinedValue(), true, fmt.Errorf("invalid element reference %q in this bounded classic-JS slice", surface)
	}

	switch property {
	case "nodeType":
		return script.NumberValue(float64(nodeTypeForNode(node))), true, nil
	case "nodeName":
		return script.StringValue(nodeNameForNode(node)), true, nil
	case "namespaceURI":
		if node.Kind == dom.NodeKindDocument {
			return script.NullValue(), true, nil
		}
		return script.StringValue(node.NamespaceURI), true, nil
	case "nodeValue":
		if node.Kind == dom.NodeKindText {
			return script.StringValue(node.Text), true, nil
		}
		return script.NullValue(), true, nil
	case "ownerDocument":
		if node.Kind == dom.NodeKindDocument {
			return script.NullValue(), true, nil
		}
		return script.HostObjectReference("document"), true, nil
	case "parentNode":
		if node.Parent == 0 {
			return script.NullValue(), true, nil
		}
		if parent := store.Node(node.Parent); parent != nil && parent.Kind == dom.NodeKindDocument {
			return script.HostObjectReference("document"), true, nil
		}
		return browserElementReferenceValue(node.Parent, store), true, nil
	case "parentElement":
		if parent := store.Node(node.Parent); parent != nil && parent.Kind == dom.NodeKindElement {
			return browserElementReferenceValue(parent.ID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "firstChild":
		if childID := firstChildNodeID(node); childID != 0 {
			return browserElementReferenceValue(childID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "lastChild":
		if childID := lastChildNodeID(node); childID != 0 {
			return browserElementReferenceValue(childID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "firstElementChild":
		if childID := firstElementChildNodeID(store, node); childID != 0 {
			return browserElementReferenceValue(childID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "lastElementChild":
		if childID := lastElementChildNodeID(store, node); childID != 0 {
			return browserElementReferenceValue(childID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "nextSibling":
		if siblingID := siblingNodeID(store, node, true, false); siblingID != 0 {
			return browserElementReferenceValue(siblingID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "previousSibling":
		if siblingID := siblingNodeID(store, node, false, false); siblingID != 0 {
			return browserElementReferenceValue(siblingID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "nextElementSibling":
		if siblingID := siblingNodeID(store, node, true, true); siblingID != 0 {
			return browserElementReferenceValue(siblingID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "previousElementSibling":
		if siblingID := siblingNodeID(store, node, false, true); siblingID != 0 {
			return browserElementReferenceValue(siblingID, store), true, nil
		}
		return script.NullValue(), true, nil
	case "childElementCount":
		return script.NumberValue(float64(childElementCount(store, node))), true, nil
	}

	return script.UndefinedValue(), false, nil
}

func nodeTypeForNode(node *dom.Node) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case dom.NodeKindElement:
		return 1
	case dom.NodeKindText:
		return 3
	case dom.NodeKindDocument:
		return 9
	default:
		return 0
	}
}

func nodeNameForNode(node *dom.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind {
	case dom.NodeKindDocument:
		return "#document"
	case dom.NodeKindText:
		return "#text"
	case dom.NodeKindElement:
		if node.NamespaceURI != "" {
			return node.TagName
		}
		return strings.ToUpper(node.TagName)
	default:
		return ""
	}
}

func firstChildNodeID(node *dom.Node) dom.NodeID {
	if node == nil || len(node.Children) == 0 {
		return 0
	}
	return node.Children[0]
}

func lastChildNodeID(node *dom.Node) dom.NodeID {
	if node == nil || len(node.Children) == 0 {
		return 0
	}
	return node.Children[len(node.Children)-1]
}

func firstElementChildNodeID(store *dom.Store, node *dom.Node) dom.NodeID {
	if store == nil || node == nil {
		return 0
	}
	for _, childID := range node.Children {
		child := store.Node(childID)
		if child != nil && child.Kind == dom.NodeKindElement {
			return childID
		}
	}
	return 0
}

func lastElementChildNodeID(store *dom.Store, node *dom.Node) dom.NodeID {
	if store == nil || node == nil {
		return 0
	}
	for i := len(node.Children) - 1; i >= 0; i-- {
		childID := node.Children[i]
		child := store.Node(childID)
		if child != nil && child.Kind == dom.NodeKindElement {
			return childID
		}
	}
	return 0
}

func siblingNodeID(store *dom.Store, node *dom.Node, forward bool, elementOnly bool) dom.NodeID {
	if store == nil || node == nil || node.Parent == 0 {
		return 0
	}
	parent := store.Node(node.Parent)
	if parent == nil || len(parent.Children) == 0 {
		return 0
	}
	index := -1
	for i, siblingID := range parent.Children {
		if siblingID == node.ID {
			index = i
			break
		}
	}
	if index < 0 {
		return 0
	}

	if forward {
		for _, siblingID := range parent.Children[index+1:] {
			sibling := store.Node(siblingID)
			if sibling == nil {
				continue
			}
			if !elementOnly || sibling.Kind == dom.NodeKindElement {
				return siblingID
			}
		}
		return 0
	}

	for i := index - 1; i >= 0; i-- {
		siblingID := parent.Children[i]
		sibling := store.Node(siblingID)
		if sibling == nil {
			continue
		}
		if !elementOnly || sibling.Kind == dom.NodeKindElement {
			return siblingID
		}
	}
	return 0
}

func childElementCount(store *dom.Store, node *dom.Node) int {
	if store == nil || node == nil {
		return 0
	}
	count := 0
	for _, childID := range node.Children {
		child := store.Node(childID)
		if child == nil || child.Kind != dom.NodeKindElement {
			continue
		}
		count++
	}
	return count
}
