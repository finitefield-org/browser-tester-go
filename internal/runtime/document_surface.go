package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func resolveDocumentTitleValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.title")
	}
	return script.StringValue(documentTitleValue(store)), nil
}

func resolveDocumentReadyStateValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.readyState")
	}
	return script.StringValue(documentReadyStateValue(session)), nil
}

func resolveDocumentActiveElementValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.activeElement")
	}
	nodeID := documentActiveElementNodeID(session, store)
	if nodeID == 0 {
		return script.NullValue(), nil
	}
	return browserElementReferenceValue(nodeID, store), nil
}

func resolveDocumentURLValue(session *Session, store *dom.Store, property string) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document." + property)
	}
	switch property {
	case "baseURI", "URL", "documentURI":
		return script.StringValue(session.URL()), nil
	default:
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document." + property)
	}
}

func resolveDocumentDoctypeValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.doctype")
	}
	return script.NullValue(), nil
}

func resolveDocumentDefaultViewValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.defaultView")
	}
	return script.HostObjectReference("window"), nil
}

func resolveDocumentCompatModeValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.compatMode")
	}
	return script.StringValue("CSS1Compat"), nil
}

func resolveDocumentContentTypeValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.contentType")
	}
	return script.StringValue("text/html"), nil
}

func resolveDocumentDesignModeValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.designMode")
	}
	return script.StringValue("off"), nil
}

func resolveDocumentDirValue(session *Session, store *dom.Store) (script.Value, error) {
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedDocumentSurfaceError("document.dir")
	}
	return script.StringValue(documentDirValue(store)), nil
}

func unsupportedDocumentSurfaceError(surface string) error {
	return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", surface))
}

func documentTitleValue(store *dom.Store) string {
	if store == nil {
		return ""
	}
	nodeID := firstDocumentElementByTag(store, "title")
	if nodeID == 0 {
		return ""
	}
	return strings.TrimSpace(store.TextContentForNode(nodeID))
}

func documentReadyStateValue(session *Session) string {
	if session == nil {
		return "loading"
	}
	if !session.domReady || strings.TrimSpace(session.currentScriptHTML) != "" || session.writingHTML {
		return "loading"
	}
	return "complete"
}

func documentActiveElementNodeID(session *Session, store *dom.Store) dom.NodeID {
	if session == nil || store == nil {
		return 0
	}
	if nodeID := store.FocusedNodeID(); nodeID != 0 {
		return nodeID
	}
	if nodeID := firstDocumentElementByTag(store, "body"); nodeID != 0 {
		return nodeID
	}
	children, err := store.Children(store.DocumentID())
	if err != nil {
		return 0
	}
	nodeID, ok := children.Item(0)
	if !ok {
		return 0
	}
	return nodeID
}

func documentDirValue(store *dom.Store) string {
	if store == nil {
		return ""
	}
	children, err := store.Children(store.DocumentID())
	if err != nil {
		return ""
	}
	nodeID, ok := children.Item(0)
	if !ok {
		return ""
	}
	value, ok := domAttributeValue(store, nodeID, "dir")
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}
