package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

const browserXMLSerializerHostPath = "xmlserializer"

func browserXMLSerializerConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "XMLSerializer is unavailable in this bounded classic-JS slice")
	}
	return script.HostObjectReference(browserXMLSerializerHostPath), nil
}

func resolveXMLSerializerReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "":
		return script.HostObjectReference(browserXMLSerializerHostPath), nil
	case "serializeToString":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserXMLSerializerSerializeToString(session, store, args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "XMLSerializer."+path))
}

func browserXMLSerializerSerializeToString(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("XMLSerializer.serializeToString expects 1 argument")
	}
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "XMLSerializer.serializeToString is unavailable in this bounded classic-JS slice")
	}
	nodeID, err := browserNodeIDFromValue(args[0], "XMLSerializer.serializeToString")
	if err != nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, err.Error())
	}
	node := store.Node(nodeID)
	if node == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid node reference %q in this bounded classic-JS slice", script.ToJSString(args[0])))
	}
	switch node.Kind {
	case dom.NodeKindElement, dom.NodeKindText:
		value, err := store.OuterHTMLForNode(nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.StringValue(value), nil
	default:
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("XMLSerializer.serializeToString does not support node kind %d in this bounded classic-JS slice", node.Kind))
	}
}
