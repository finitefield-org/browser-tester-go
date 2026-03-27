package dom

import (
	"fmt"
	"strings"
)

func (s *Store) InnerHTMLForNode(nodeID NodeID) (string, error) {
	if s == nil {
		return "", fmt.Errorf("dom store is nil")
	}
	node, err := s.elementNode(nodeID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, childID := range node.Children {
		s.serializeNode(&b, childID)
	}
	return b.String(), nil
}

func (s *Store) SetInnerHTML(nodeID NodeID, markup string) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	node, err := s.elementNode(nodeID)
	if err != nil {
		return err
	}

	fragmentIDs, err := s.parseFragmentNodes(markup)
	if err != nil {
		return err
	}

	s.clearFocusedNodeIfSubtreeContains(nodeID, false)
	s.clearTargetNodeIfSubtreeContains(nodeID, false)
	oldChildren := append([]NodeID(nil), node.Children...)
	node.Children = node.Children[:0]
	for _, childID := range oldChildren {
		s.deleteSubtree(childID)
	}
	for _, childID := range fragmentIDs {
		s.appendChild(nodeID, childID)
	}
	s.syncTextareaDefaultsForSubtree(nodeID)
	return nil
}

func (s *Store) ReplaceChildren(nodeID NodeID, markup string) error {
	return s.SetInnerHTML(nodeID, markup)
}

func (s *Store) ReplaceChild(parentID, newChildID, oldChildID NodeID) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	parent := s.Node(parentID)
	if parent == nil {
		return fmt.Errorf("invalid parent node id: %d", parentID)
	}
	oldChild := s.Node(oldChildID)
	if oldChild == nil {
		return fmt.Errorf("invalid old child node id: %d", oldChildID)
	}
	if oldChild.Parent != parentID {
		return fmt.Errorf("node %d is not a child of parent %d", oldChildID, parentID)
	}
	newChild := s.Node(newChildID)
	if newChild == nil {
		return fmt.Errorf("invalid new child node id: %d", newChildID)
	}
	if newChildID == oldChildID {
		return nil
	}
	if parent.Kind == NodeKindDocument && newChild.Kind != NodeKindElement {
		return fmt.Errorf("document node can only contain element children")
	}
	if subtreeContainsNode(s, newChildID, parentID) {
		return fmt.Errorf("node %d cannot be replaced into its own descendant %d", newChildID, parentID)
	}

	oldIndex := indexOfNodeID(parent.Children, oldChildID)
	if oldIndex < 0 {
		return fmt.Errorf("node %d is not attached to its parent", oldChildID)
	}
	newOldParentID := newChild.Parent
	newOldIndex := -1
	if newOldParentID != 0 {
		if newOldParent := s.Node(newOldParentID); newOldParent != nil {
			newOldIndex = indexOfNodeID(newOldParent.Children, newChildID)
			newOldParent.Children = removeNodeID(newOldParent.Children, newChildID)
		}
	}
	if newOldParentID == parentID && newOldIndex >= 0 && newOldIndex < oldIndex {
		oldIndex--
	}

	s.clearFocusedNodeIfSubtreeContains(oldChildID, true)
	s.clearTargetNodeIfSubtreeContains(oldChildID, true)
	parent.Children = spliceNodeIDs(parent.Children, oldIndex, 1, []NodeID{newChildID})
	newChild.Parent = parentID
	oldChild.Parent = 0
	s.deleteSubtree(oldChildID)
	s.syncTextareaDefaultsForSubtree(parentID)
	return nil
}

func (s *Store) SetOuterHTML(nodeID NodeID, markup string) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	node, err := s.elementNode(nodeID)
	if err != nil {
		return err
	}

	parentID := node.Parent
	if parentID == 0 {
		return nil
	}
	parent := s.Node(parentID)
	if parent == nil {
		return fmt.Errorf("invalid parent node id: %d", parentID)
	}
	if parent.Kind == NodeKindDocument {
		return fmt.Errorf("node %d cannot be replaced within a document", nodeID)
	}

	fragmentIDs, err := s.parseFragmentNodes(markup)
	if err != nil {
		return err
	}

	s.clearFocusedNodeIfSubtreeContains(nodeID, true)
	s.clearTargetNodeIfSubtreeContains(nodeID, true)
	index := indexOfNodeID(parent.Children, nodeID)
	if index < 0 {
		return fmt.Errorf("node %d is not attached to its parent", nodeID)
	}

	parent.Children = spliceNodeIDs(parent.Children, index, 1, fragmentIDs)
	for _, childID := range fragmentIDs {
		child := s.Node(childID)
		if child != nil {
			child.Parent = parentID
		}
	}

	s.deleteSubtree(nodeID)
	s.syncTextareaDefaultsForSubtree(parentID)
	return nil
}

func (s *Store) InsertAdjacentHTML(nodeID NodeID, position, markup string) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	node, err := s.elementNode(nodeID)
	if err != nil {
		return err
	}

	normalized := strings.ToLower(strings.TrimSpace(position))
	switch normalized {
	case "beforebegin":
		parentID := node.Parent
		if parentID == 0 {
			return fmt.Errorf("node %d has no parent for beforebegin", nodeID)
		}
		parent := s.Node(parentID)
		if parent == nil {
			return fmt.Errorf("invalid parent node id: %d", parentID)
		}
		if parent.Kind == NodeKindDocument {
			return fmt.Errorf("node %d cannot insert beforebegin within a document", nodeID)
		}
		index := indexOfNodeID(parent.Children, nodeID)
		if index < 0 {
			return fmt.Errorf("node %d is not attached to its parent", nodeID)
		}
		fragmentIDs, err := s.parseFragmentNodes(markup)
		if err != nil {
			return err
		}
		parent.Children = spliceNodeIDs(parent.Children, index, 0, fragmentIDs)
		for _, childID := range fragmentIDs {
			if child := s.Node(childID); child != nil {
				child.Parent = parentID
			}
		}
		s.syncTextareaDefaultsForSubtree(parentID)
	case "afterbegin":
		fragmentIDs, err := s.parseFragmentNodes(markup)
		if err != nil {
			return err
		}
		node.Children = spliceNodeIDs(node.Children, 0, 0, fragmentIDs)
		for _, childID := range fragmentIDs {
			if child := s.Node(childID); child != nil {
				child.Parent = nodeID
			}
		}
		s.syncTextareaDefaultsForSubtree(nodeID)
	case "beforeend":
		fragmentIDs, err := s.parseFragmentNodes(markup)
		if err != nil {
			return err
		}
		node.Children = spliceNodeIDs(node.Children, len(node.Children), 0, fragmentIDs)
		for _, childID := range fragmentIDs {
			if child := s.Node(childID); child != nil {
				child.Parent = nodeID
			}
		}
		s.syncTextareaDefaultsForSubtree(nodeID)
	case "afterend":
		parentID := node.Parent
		if parentID == 0 {
			return fmt.Errorf("node %d has no parent for afterend", nodeID)
		}
		parent := s.Node(parentID)
		if parent == nil {
			return fmt.Errorf("invalid parent node id: %d", parentID)
		}
		if parent.Kind == NodeKindDocument {
			return fmt.Errorf("node %d cannot insert afterend within a document", nodeID)
		}
		index := indexOfNodeID(parent.Children, nodeID)
		if index < 0 {
			return fmt.Errorf("node %d is not attached to its parent", nodeID)
		}
		fragmentIDs, err := s.parseFragmentNodes(markup)
		if err != nil {
			return err
		}
		parent.Children = spliceNodeIDs(parent.Children, index+1, 0, fragmentIDs)
		for _, childID := range fragmentIDs {
			if child := s.Node(childID); child != nil {
				child.Parent = parentID
			}
		}
		s.syncTextareaDefaultsForSubtree(parentID)
	default:
		return fmt.Errorf("invalid insertAdjacentHTML position %q", position)
	}

	return nil
}

func (s *Store) InsertAdjacentElement(nodeID NodeID, position string, childID NodeID) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	node, err := s.elementNode(nodeID)
	if err != nil {
		return err
	}
	child := s.Node(childID)
	if child == nil {
		return fmt.Errorf("invalid child node id: %d", childID)
	}
	if childID == nodeID {
		return fmt.Errorf("node %d cannot be inserted adjacent to itself", childID)
	}

	normalized := strings.ToLower(strings.TrimSpace(position))
	switch normalized {
	case "beforebegin":
		parentID := node.Parent
		if parentID == 0 {
			return fmt.Errorf("node %d has no parent for beforebegin", nodeID)
		}
		parent := s.Node(parentID)
		if parent == nil {
			return fmt.Errorf("invalid parent node id: %d", parentID)
		}
		if parent.Kind == NodeKindDocument && child.Kind != NodeKindElement {
			return fmt.Errorf("document node can only contain element children")
		}
		return s.InsertBefore(parentID, childID, nodeID)
	case "afterbegin":
		if len(node.Children) == 0 {
			return s.AppendChild(nodeID, childID)
		}
		return s.InsertBefore(nodeID, childID, node.Children[0])
	case "beforeend":
		return s.AppendChild(nodeID, childID)
	case "afterend":
		parentID := node.Parent
		if parentID == 0 {
			return fmt.Errorf("node %d has no parent for afterend", nodeID)
		}
		parent := s.Node(parentID)
		if parent == nil {
			return fmt.Errorf("invalid parent node id: %d", parentID)
		}
		if parent.Kind == NodeKindDocument && child.Kind != NodeKindElement {
			return fmt.Errorf("document node can only contain element children")
		}
		index := indexOfNodeID(parent.Children, nodeID)
		if index < 0 {
			return fmt.Errorf("node %d is not attached to its parent", nodeID)
		}
		if index+1 < len(parent.Children) {
			return s.InsertBefore(parentID, childID, parent.Children[index+1])
		}
		return s.AppendChild(parentID, childID)
	default:
		return fmt.Errorf("invalid insertAdjacentElement position %q", position)
	}
}

func (s *Store) InsertAdjacentText(nodeID NodeID, position, text string) (NodeID, error) {
	if s == nil {
		return 0, fmt.Errorf("dom store is nil")
	}
	textID, err := s.CreateTextNode(text)
	if err != nil {
		return 0, err
	}
	if err := s.InsertAdjacentElement(nodeID, position, textID); err != nil {
		s.deleteSubtree(textID)
		return 0, err
	}
	return textID, nil
}

func (s *Store) RemoveNode(nodeID NodeID) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	node := s.Node(nodeID)
	if node == nil {
		return fmt.Errorf("invalid node id: %d", nodeID)
	}
	if node.Kind == NodeKindDocument {
		return fmt.Errorf("document node cannot be removed")
	}
	if node.Parent == 0 {
		return nil
	}

	s.clearFocusedNodeIfSubtreeContains(nodeID, true)
	s.clearTargetNodeIfSubtreeContains(nodeID, true)
	parent := s.Node(node.Parent)
	if parent != nil {
		parent.Children = removeNodeID(parent.Children, nodeID)
	}
	s.deleteSubtree(nodeID)
	s.syncTextareaDefaultsForSubtree(node.Parent)
	return nil
}

func (s *Store) CloneNode(nodeID NodeID, deep bool) (NodeID, error) {
	if s == nil {
		return 0, fmt.Errorf("dom store is nil")
	}
	if s.Node(nodeID) == nil {
		return 0, fmt.Errorf("invalid node id: %d", nodeID)
	}
	return s.cloneNodeRecursive(nodeID, deep), nil
}

func (s *Store) CloneNodeAfter(nodeID NodeID, deep bool) (NodeID, error) {
	if s == nil {
		return 0, fmt.Errorf("dom store is nil")
	}
	node, err := s.elementNode(nodeID)
	if err != nil {
		return 0, err
	}
	if node.Kind == NodeKindDocument {
		return 0, fmt.Errorf("document node cannot be cloned")
	}
	parentID := node.Parent
	if parentID == 0 {
		return 0, fmt.Errorf("node %d has no parent for clone", nodeID)
	}
	parent := s.Node(parentID)
	if parent == nil {
		return 0, fmt.Errorf("invalid parent node id: %d", parentID)
	}
	index := indexOfNodeID(parent.Children, nodeID)
	if index < 0 {
		return 0, fmt.Errorf("node %d is not attached to its parent", nodeID)
	}

	cloneID, err := s.CloneNode(nodeID, deep)
	if err != nil {
		return 0, err
	}

	parent.Children = spliceNodeIDs(parent.Children, index+1, 0, []NodeID{cloneID})
	if cloned := s.Node(cloneID); cloned != nil {
		cloned.Parent = parentID
	}
	s.syncTextareaDefaultsForSubtree(parentID)
	return cloneID, nil
}

func (s *Store) AppendChild(parentID, childID NodeID) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	parent := s.Node(parentID)
	if parent == nil {
		return fmt.Errorf("invalid parent node id: %d", parentID)
	}
	child := s.Node(childID)
	if child == nil {
		return fmt.Errorf("invalid child node id: %d", childID)
	}
	if parentID == childID {
		return fmt.Errorf("node %d cannot be appended to itself", childID)
	}
	if subtreeContainsNode(s, childID, parentID) {
		return fmt.Errorf("node %d cannot be appended into its own descendant %d", childID, parentID)
	}
	if parent.Kind == NodeKindDocument && child.Kind != NodeKindElement {
		return fmt.Errorf("document node can only contain element children")
	}
	if child.Parent != 0 {
		if oldParent := s.Node(child.Parent); oldParent != nil {
			oldParent.Children = removeNodeID(oldParent.Children, childID)
		}
	}
	child.Parent = parentID
	parent.Children = append(parent.Children, childID)
	s.syncTextareaDefaultsForSubtree(parentID)
	return nil
}

func (s *Store) InsertBefore(parentID, childID, referenceChildID NodeID) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	if referenceChildID == 0 {
		return s.AppendChild(parentID, childID)
	}
	parent := s.Node(parentID)
	if parent == nil {
		return fmt.Errorf("invalid parent node id: %d", parentID)
	}
	reference := s.Node(referenceChildID)
	if reference == nil {
		return fmt.Errorf("invalid reference node id: %d", referenceChildID)
	}
	if reference.Parent != parentID {
		return fmt.Errorf("reference node %d is not a child of parent %d", referenceChildID, parentID)
	}
	child := s.Node(childID)
	if child == nil {
		return fmt.Errorf("invalid child node id: %d", childID)
	}
	if childID == referenceChildID {
		return nil
	}
	if subtreeContainsNode(s, childID, parentID) {
		return fmt.Errorf("node %d cannot be inserted into its own descendant %d", childID, parentID)
	}
	oldParentID := child.Parent
	oldIndex := -1
	if parent.Kind == NodeKindDocument && child.Kind != NodeKindElement {
		return fmt.Errorf("document node can only contain element children")
	}
	index := indexOfNodeID(parent.Children, referenceChildID)
	if index < 0 {
		return fmt.Errorf("reference node %d is not attached to its parent", referenceChildID)
	}
	if child.Parent != 0 {
		if oldParent := s.Node(child.Parent); oldParent != nil {
			if oldParent.ID == parentID {
				oldIndex = indexOfNodeID(oldParent.Children, childID)
			}
			oldParent.Children = removeNodeID(oldParent.Children, childID)
		}
	}
	if oldParentID == parentID && oldIndex >= 0 && oldIndex < index {
		index--
	}
	parent.Children = spliceNodeIDs(parent.Children, index, 0, []NodeID{childID})
	child.Parent = parentID
	s.syncTextareaDefaultsForSubtree(parentID)
	return nil
}

func (s *Store) RemoveChild(parentID, childID NodeID) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	parent := s.Node(parentID)
	if parent == nil {
		return fmt.Errorf("invalid parent node id: %d", parentID)
	}
	child := s.Node(childID)
	if child == nil {
		return fmt.Errorf("invalid child node id: %d", childID)
	}
	if child.Parent != parentID {
		return fmt.Errorf("node %d is not a child of parent %d", childID, parentID)
	}
	s.clearFocusedNodeIfSubtreeContains(childID, true)
	s.clearTargetNodeIfSubtreeContains(childID, true)
	parent.Children = removeNodeID(parent.Children, childID)
	child.Parent = 0
	s.syncTextareaDefaultsForSubtree(parentID)
	return nil
}

func (s *Store) parseFragmentNodes(markup string) ([]NodeID, error) {
	temp := NewStore()
	if err := temp.BootstrapHTML(markup); err != nil {
		return nil, err
	}

	rootChildren := temp.documentChildren()
	if len(rootChildren) == 0 {
		return []NodeID{}, nil
	}

	out := make([]NodeID, 0, len(rootChildren))
	for _, childID := range rootChildren {
		cloned := s.cloneNodeFrom(temp, childID, true)
		if cloned != 0 {
			out = append(out, cloned)
		}
	}
	return out, nil
}

func (s *Store) cloneNodeRecursive(nodeID NodeID, deep bool) NodeID {
	node := s.Node(nodeID)
	if node == nil {
		return 0
	}

	clonedID := s.newNode(Node{
		Kind:         node.Kind,
		TagName:      node.TagName,
		Attrs:        cloneAttributes(node.Attrs),
		Text:         node.Text,
		DefaultAttrs: cloneAttributes(node.DefaultAttrs),
		DefaultText:  node.DefaultText,
		UserValidity: node.UserValidity,
	})
	if !deep {
		return clonedID
	}

	for _, childID := range node.Children {
		clonedChildID := s.cloneNodeRecursive(childID, true)
		if clonedChildID != 0 {
			s.appendChild(clonedID, clonedChildID)
		}
	}
	return clonedID
}

func (s *Store) cloneNodeFrom(src *Store, nodeID NodeID, deep bool) NodeID {
	if s == nil || src == nil {
		return 0
	}
	node := src.Node(nodeID)
	if node == nil {
		return 0
	}

	clonedID := s.newNode(Node{
		Kind:         node.Kind,
		TagName:      node.TagName,
		Attrs:        cloneAttributes(node.Attrs),
		Text:         node.Text,
		DefaultAttrs: cloneAttributes(node.DefaultAttrs),
		DefaultText:  node.DefaultText,
		UserValidity: node.UserValidity,
	})
	if !deep {
		return clonedID
	}

	for _, childID := range node.Children {
		clonedChildID := s.cloneNodeFrom(src, childID, true)
		if clonedChildID != 0 {
			s.appendChild(clonedID, clonedChildID)
		}
	}
	return clonedID
}

func cloneAttributes(attrs []Attribute) []Attribute {
	if len(attrs) == 0 {
		return []Attribute{}
	}
	out := make([]Attribute, len(attrs))
	copy(out, attrs)
	return out
}

func deleteSubtree(s *Store, nodeID NodeID) {
	if s == nil {
		return
	}
	node := s.nodes[nodeID]
	if node == nil {
		return
	}

	children := append([]NodeID(nil), node.Children...)
	for _, childID := range children {
		deleteSubtree(s, childID)
	}

	if parent := s.nodes[node.Parent]; parent != nil {
		parent.Children = removeNodeID(parent.Children, nodeID)
	}
	delete(s.nodes, nodeID)
}

func (s *Store) deleteSubtree(nodeID NodeID) {
	deleteSubtree(s, nodeID)
}

func indexOfNodeID(ids []NodeID, target NodeID) int {
	for i, id := range ids {
		if id == target {
			return i
		}
	}
	return -1
}

func spliceNodeIDs(ids []NodeID, index, deleteCount int, insert []NodeID) []NodeID {
	if index < 0 {
		index = 0
	}
	if index > len(ids) {
		index = len(ids)
	}
	if deleteCount < 0 {
		deleteCount = 0
	}
	end := index + deleteCount
	if end > len(ids) {
		end = len(ids)
	}

	out := make([]NodeID, 0, len(ids)-(end-index)+len(insert))
	out = append(out, ids[:index]...)
	out = append(out, insert...)
	out = append(out, ids[end:]...)
	return out
}

func removeNodeID(ids []NodeID, target NodeID) []NodeID {
	index := indexOfNodeID(ids, target)
	if index < 0 {
		return ids
	}
	return append(ids[:index], ids[index+1:]...)
}
