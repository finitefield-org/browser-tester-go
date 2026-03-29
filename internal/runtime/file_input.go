package runtime

import (
	"fmt"
	"strconv"
	"strings"

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

	selection, ok := latestFileInputSelectionForNode(session, store, nodeID)
	if !ok || len(selection.Files) == 0 {
		return script.ArrayValue(nil), nil
	}

	files := make([]script.Value, 0, len(selection.Files))
	for _, fileName := range selection.Files {
		text, seeded := session.fileInputTextForSelection(selection.Selector, fileName)
		files = append(files, browserFileInputFileValue(fileName, text, seeded))
	}
	return script.ArrayValue(files), nil
}

func clearFileInputSelectionForNode(session *Session, store *dom.Store, nodeID dom.NodeID) {
	if session == nil || store == nil {
		return
	}
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
	matches, err := store.Select(normalized)
	if err != nil {
		return false
	}
	for _, match := range matches {
		if match == nodeID {
			return true
		}
	}
	return false
}

func (s *Session) fileInputTextForSelection(selector, fileName string) (string, bool) {
	if s == nil {
		return "", false
	}
	registry := s.Registry()
	if registry == nil || registry.FileInput() == nil {
		return "", false
	}
	return registry.FileInput().FileText(selector, fileName)
}

func browserFileInputFileValue(fileName, text string, hasText bool) script.Value {
	entries := []script.ObjectEntry{
		{Key: "name", Value: script.StringValue(fileName)},
		{Key: "text", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("File.text accepts no arguments")
			}
			if !hasText {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "file content is unavailable in this bounded classic-JS slice")
			}
			return script.PromiseValue(script.StringValue(text)), nil
		})},
	}
	return script.ObjectValue(entries)
}
