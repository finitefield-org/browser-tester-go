package dom

import (
	"fmt"
	"strings"
)

func (s *Store) FocusedNodeID() NodeID {
	if s == nil {
		return 0
	}
	return s.focusedNodeID
}

func (s *Store) SetFocusedNode(nodeID NodeID) error {
	if s == nil {
		return fmt.Errorf("dom store is nil")
	}
	if nodeID == 0 {
		s.focusedNodeID = 0
		return nil
	}
	node, err := s.elementNode(nodeID)
	if err != nil {
		return err
	}
	s.focusedNodeID = node.ID
	return nil
}

func (s *Store) ClearFocusedNode() {
	if s == nil {
		return
	}
	s.focusedNodeID = 0
}

func (s *Store) clearFocusedNodeIfSubtreeContains(nodeID NodeID, includeSelf bool) {
	if s == nil || s.focusedNodeID == 0 || nodeID == 0 {
		return
	}
	if includeSelf {
		if subtreeContainsNode(s, nodeID, s.focusedNodeID) {
			s.focusedNodeID = 0
		}
		return
	}
	if subtreeContainsDescendant(s, nodeID, s.focusedNodeID) {
		s.focusedNodeID = 0
	}
}

func subtreeContainsNode(s *Store, ancestorID, nodeID NodeID) bool {
	if s == nil || ancestorID == 0 || nodeID == 0 {
		return false
	}
	if ancestorID == nodeID {
		return true
	}
	ancestor := s.Node(ancestorID)
	if ancestor == nil {
		return false
	}
	for _, childID := range ancestor.Children {
		if childID == nodeID || subtreeContainsNode(s, childID, nodeID) {
			return true
		}
	}
	return false
}

func subtreeContainsDescendant(s *Store, ancestorID, nodeID NodeID) bool {
	return nodeID != 0 && ancestorID != nodeID && subtreeContainsNode(s, ancestorID, nodeID)
}

func (s *Store) ContainsNode(ancestorID, nodeID NodeID) bool {
	if s == nil || ancestorID == 0 || nodeID == 0 {
		return false
	}
	if s.Node(ancestorID) == nil || s.Node(nodeID) == nil {
		return false
	}
	return subtreeContainsNode(s, ancestorID, nodeID)
}

func (s *Store) IsConnected(nodeID NodeID) bool {
	if s == nil || nodeID == 0 {
		return false
	}
	if nodeID == s.documentID {
		return true
	}
	node := s.Node(nodeID)
	if node == nil {
		return false
	}
	for node.Parent != 0 {
		if node.Parent == s.documentID {
			return true
		}
		node = s.Node(node.Parent)
		if node == nil {
			return false
		}
	}
	return false
}

func (s *Store) HasChildNodes(nodeID NodeID) bool {
	if s == nil || nodeID == 0 {
		return false
	}
	node := s.Node(nodeID)
	if node == nil {
		return false
	}
	return len(node.Children) > 0
}

func (s *Store) RootNodeID(nodeID NodeID) NodeID {
	if s == nil || nodeID == 0 {
		return 0
	}
	node := s.Node(nodeID)
	if node == nil {
		return 0
	}
	current := node
	for current.Parent != 0 {
		parent := s.Node(current.Parent)
		if parent == nil {
			return 0
		}
		current = parent
	}
	return current.ID
}

const (
	documentPositionDisconnected           uint16 = 1
	documentPositionPreceding              uint16 = 2
	documentPositionFollowing              uint16 = 4
	documentPositionContains               uint16 = 8
	documentPositionContainedBy            uint16 = 16
	documentPositionImplementationSpecific uint16 = 32
)

func (s *Store) CompareDocumentPosition(nodeID, otherID NodeID) uint16 {
	if s == nil || nodeID == 0 || otherID == 0 {
		return documentPositionDisconnected | documentPositionImplementationSpecific
	}
	if nodeID == otherID {
		return 0
	}
	if s.Node(nodeID) == nil || s.Node(otherID) == nil {
		return documentPositionDisconnected | documentPositionImplementationSpecific
	}

	rootID := s.RootNodeID(nodeID)
	otherRootID := s.RootNodeID(otherID)
	if rootID == 0 || otherRootID == 0 {
		return documentPositionDisconnected | documentPositionImplementationSpecific
	}
	if rootID != otherRootID {
		if rootID < otherRootID || (rootID == otherRootID && nodeID < otherID) {
			return documentPositionDisconnected | documentPositionFollowing | documentPositionImplementationSpecific
		}
		return documentPositionDisconnected | documentPositionPreceding | documentPositionImplementationSpecific
	}

	if s.ContainsNode(nodeID, otherID) {
		return documentPositionContains | documentPositionFollowing
	}
	if s.ContainsNode(otherID, nodeID) {
		return documentPositionContainedBy | documentPositionPreceding
	}

	order := make([]NodeID, 0, 8)
	var visit func(NodeID)
	visit = func(currentID NodeID) {
		current := s.Node(currentID)
		if current == nil {
			return
		}
		order = append(order, currentID)
		for _, childID := range current.Children {
			visit(childID)
		}
	}
	visit(rootID)

	nodeIndex := indexOfNodeID(order, nodeID)
	otherIndex := indexOfNodeID(order, otherID)
	if nodeIndex < 0 || otherIndex < 0 {
		return documentPositionDisconnected | documentPositionImplementationSpecific
	}
	if nodeIndex < otherIndex {
		return documentPositionFollowing
	}
	return documentPositionPreceding
}

func subtreeContainsAttribute(s *Store, ancestorID NodeID, attrName string) bool {
	if s == nil || ancestorID == 0 || strings.TrimSpace(attrName) == "" {
		return false
	}
	found := false
	s.walkElementPreOrder(ancestorID, func(node *Node) {
		if found || node == nil || node.Kind != NodeKindElement {
			return
		}
		if _, ok := attributeValue(node.Attrs, attrName); ok {
			found = true
		}
	})
	return found
}
