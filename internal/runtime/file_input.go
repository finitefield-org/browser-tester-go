package runtime

import (
	"strconv"
	"strings"
	"time"

	"browsertester/internal/dom"
	"browsertester/internal/mocks"
	"browsertester/internal/script"
)

func resolveElementFilesValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".files"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || node.TagName != "input" {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	if inputType(node) != "file" {
		return script.NullValue(), nil
	}

	if selection, ok := session.browserFileInputSelectionValues(nodeID); ok {
		return script.ArrayValue(selection), nil
	}

	selection, ok := latestFileInputSelectionForNode(session, store, nodeID)
	if !ok || len(selection.Files) == 0 {
		return script.ArrayValue(nil), nil
	}

	files := make([]script.Value, 0, len(selection.Files))
	for _, fileName := range selection.Files {
		data, seeded := session.fileInputDataForSelection(selection.Selector, fileName)
		files = append(files, browserFileInputFileValue(session, fileName, data, seeded))
	}
	return script.ArrayValue(files), nil
}

func resolveFileInputNodeForSelector(store *dom.Store, selector string) (dom.NodeID, bool) {
	if store == nil {
		return 0, false
	}
	normalized := strings.TrimSpace(selector)
	if normalized == "" {
		return 0, false
	}

	if matches, err := store.Select(normalized); err == nil {
		for _, nodeID := range matches {
			if isFileInputNode(store.Node(nodeID)) {
				return nodeID, true
			}
		}
	}

	nodes := store.Nodes()
	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if !isFileInputNode(node) {
			continue
		}
		matched, err := store.Matches(node.ID, normalized)
		if err != nil || !matched {
			continue
		}
		return node.ID, true
	}
	return 0, false
}

func isFileInputNode(node *dom.Node) bool {
	return node != nil && node.Kind == dom.NodeKindElement && node.TagName == "input" && inputType(node) == "file"
}

func clearFileInputSelectionForNode(session *Session, store *dom.Store, nodeID dom.NodeID) {
	if session == nil || store == nil {
		return
	}
	session.clearBrowserFileInputSelection(nodeID)
	selection, ok := latestFileInputSelectionForNode(session, store, nodeID)
	if !ok {
		return
	}
	registry := session.Registry()
	if registry == nil || registry.FileInput() == nil {
		return
	}
	registry.FileInput().ClearFiles(selection.Selector)
}

func latestFileInputSelectionForNode(session *Session, store *dom.Store, nodeID dom.NodeID) (mocks.FileInputSelection, bool) {
	if session == nil || store == nil {
		return mocks.FileInputSelection{}, false
	}
	registry := session.Registry()
	if registry == nil || registry.FileInput() == nil {
		return mocks.FileInputSelection{}, false
	}
	selections := registry.FileInput().Selections()
	for i := len(selections) - 1; i >= 0; i-- {
		item := selections[i]
		if !fileInputSelectionMatchesNode(store, item.Selector, nodeID) {
			continue
		}
		return item, true
	}
	return mocks.FileInputSelection{}, false
}

func fileInputSelectionMatchesNode(store *dom.Store, selector string, nodeID dom.NodeID) bool {
	if store == nil {
		return false
	}
	normalized := strings.TrimSpace(selector)
	if normalized == "" {
		return false
	}
	matched, err := store.Matches(nodeID, normalized)
	if err != nil {
		return false
	}
	return matched
}

func (s *Session) fileInputDataForSelection(selector, fileName string) (mocks.FileInputFileData, bool) {
	if s == nil {
		return mocks.FileInputFileData{}, false
	}
	registry := s.Registry()
	if registry == nil || registry.FileInput() == nil {
		return mocks.FileInputFileData{}, false
	}
	return registry.FileInput().FileData(selector, fileName)
}

func (s *Session) setBrowserFileInputSelection(nodeID dom.NodeID, files []script.Value) {
	if s == nil {
		return
	}
	if s.fileInputSelections == nil {
		s.fileInputSelections = map[dom.NodeID][]script.Value{}
	}
	copied := append([]script.Value(nil), files...)
	s.fileInputSelections[nodeID] = copied
}

func (s *Session) clearBrowserFileInputSelection(nodeID dom.NodeID) {
	if s == nil || len(s.fileInputSelections) == 0 {
		return
	}
	delete(s.fileInputSelections, nodeID)
}

func (s *Session) browserFileInputSelectionValues(nodeID dom.NodeID) ([]script.Value, bool) {
	if s == nil || len(s.fileInputSelections) == 0 {
		return nil, false
	}
	files, ok := s.fileInputSelections[nodeID]
	if !ok {
		return nil, false
	}
	copied := append([]script.Value(nil), files...)
	return copied, true
}

func cloneBrowserFileInputSelectionMap(selections map[dom.NodeID][]script.Value) map[dom.NodeID][]script.Value {
	if len(selections) == 0 {
		return nil
	}
	out := make(map[dom.NodeID][]script.Value, len(selections))
	for nodeID, files := range selections {
		out[nodeID] = append([]script.Value(nil), files...)
	}
	return out
}

func browserFileInputFileValue(session *Session, fileName string, data mocks.FileInputFileData, seeded bool) script.Value {
	copyBytes := func() []byte {
		if data.HasBytes {
			out := make([]byte, len(data.Bytes))
			copy(out, data.Bytes)
			return out
		}
		if data.HasText {
			return []byte(data.Text)
		}
		return nil
	}

	lastModified := int64(0)
	if seeded {
		lastModified = time.Now().UnixMilli()
	}
	id := session.allocateBrowserFileState(copyBytes(), fileName, data.Type, lastModified, seeded)
	if strings.TrimSpace(id) == "" {
		return script.NullValue()
	}
	return browserFileReferenceValue(id)
}
