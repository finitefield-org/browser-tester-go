package dom

import "fmt"

func (s *Store) QuerySelectorWithin(rootID NodeID, selector string) (NodeID, bool, error) {
	if s == nil {
		return 0, false, fmt.Errorf("dom store is nil")
	}
	root := s.Node(rootID)
	if root == nil {
		return 0, false, fmt.Errorf("invalid node id: %d", rootID)
	}
	parsed, err := parseSelectorSequence(selector)
	if err != nil {
		return 0, false, err
	}

	scopeNodeID := NodeID(0)
	if root.Kind == NodeKindElement {
		scopeNodeID = rootID
	}

	var matches []NodeID
	if root.Kind == NodeKindElement {
		s.walkElementPreOrder(rootID, func(node *Node) {
			if node == nil {
				return
			}
			if parsed.matchesWithScope(s, node, scopeNodeID) {
				matches = append(matches, node.ID)
			}
		})
	} else {
		for _, childID := range root.Children {
			s.walkElementPreOrder(childID, func(node *Node) {
				if node == nil {
					return
				}
				if parsed.matchesWithScope(s, node, scopeNodeID) {
					matches = append(matches, node.ID)
				}
			})
		}
	}

	if len(matches) == 0 {
		return 0, false, nil
	}
	return matches[0], true, nil
}

func (s *Store) QuerySelectorAllWithin(rootID NodeID, selector string) (NodeList, error) {
	if s == nil {
		return NodeList{}, fmt.Errorf("dom store is nil")
	}
	root := s.Node(rootID)
	if root == nil {
		return NodeList{}, fmt.Errorf("invalid node id: %d", rootID)
	}
	parsed, err := parseSelectorSequence(selector)
	if err != nil {
		return NodeList{}, err
	}

	scopeNodeID := NodeID(0)
	if root.Kind == NodeKindElement {
		scopeNodeID = rootID
	}

	matches := make([]NodeID, 0, 4)
	if root.Kind == NodeKindElement {
		s.walkElementPreOrder(rootID, func(node *Node) {
			if node == nil {
				return
			}
			if parsed.matchesWithScope(s, node, scopeNodeID) {
				matches = append(matches, node.ID)
			}
		})
	} else {
		for _, childID := range root.Children {
			s.walkElementPreOrder(childID, func(node *Node) {
				if node == nil {
					return
				}
				if parsed.matchesWithScope(s, node, scopeNodeID) {
					matches = append(matches, node.ID)
				}
			})
		}
	}

	return newNodeList(matches), nil
}

func (s *Store) GetElementByID(id string) (NodeID, bool, error) {
	if s == nil {
		return 0, false, fmt.Errorf("dom store is nil")
	}
	normalized := id
	if normalized == "" {
		return 0, false, nil
	}
	for _, rootID := range s.documentChildren() {
		var match NodeID
		s.walkElementPreOrder(rootID, func(node *Node) {
			if match != 0 || node == nil {
				return
			}
			if value, ok := attributeValue(node.Attrs, "id"); ok && value == normalized {
				match = node.ID
			}
		})
		if match != 0 {
			return match, true, nil
		}
	}
	return 0, false, nil
}

func (s *Store) QuerySelector(selector string) (NodeID, bool, error) {
	if s == nil {
		return 0, false, fmt.Errorf("dom store is nil")
	}
	nodes, err := s.Select(selector)
	if err != nil {
		return 0, false, err
	}
	if len(nodes) == 0 {
		return 0, false, nil
	}
	return nodes[0], true, nil
}

func (s *Store) QuerySelectorAll(selector string) (NodeList, error) {
	if s == nil {
		return NodeList{}, fmt.Errorf("dom store is nil")
	}
	nodes, err := s.Select(selector)
	if err != nil {
		return NodeList{}, err
	}
	return newNodeList(nodes), nil
}

func (s *Store) Matches(nodeID NodeID, selector string) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("dom store is nil")
	}
	node := s.Node(nodeID)
	if node == nil {
		return false, fmt.Errorf("invalid node id: %d", nodeID)
	}
	parsed, err := parseSelectorSequence(selector)
	if err != nil {
		return false, err
	}
	return parsed.matchesWithScope(s, node, nodeID), nil
}

func (s *Store) Closest(nodeID NodeID, selector string) (NodeID, bool, error) {
	if s == nil {
		return 0, false, fmt.Errorf("dom store is nil")
	}
	parsed, err := parseSelectorSequence(selector)
	if err != nil {
		return 0, false, err
	}

	current := nodeID
	for current != 0 {
		node := s.Node(current)
		if node == nil {
			return 0, false, fmt.Errorf("invalid node id: %d", nodeID)
		}
		if parsed.matchesWithScope(s, node, nodeID) {
			return current, true, nil
		}
		current = node.Parent
	}

	return 0, false, nil
}
