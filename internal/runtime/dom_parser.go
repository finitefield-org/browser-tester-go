package runtime

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

const browserDOMParserSVGNamespaceURI = "http://www.w3.org/2000/svg"
const browserDOMParserParserErrorNamespaceURI = "http://www.mozilla.org/newlayout/xml/parsererror.xml"

func browserDOMParserConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "DOMParser is unavailable in this bounded classic-JS slice")
	}
	return script.HostObjectReference("domparser"), nil
}

func resolveDOMParserReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	switch strings.TrimPrefix(path, ".") {
	case "":
		return script.HostObjectReference("domparser"), nil
	case "parseFromString":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDOMParserParseFromString(session, store, args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "DOMParser."+path))
}

func browserDOMParserParseFromString(session *Session, store *dom.Store, args []script.Value) (script.Value, error) {
	if len(args) != 2 {
		return script.UndefinedValue(), fmt.Errorf("DOMParser.parseFromString expects 2 arguments")
	}
	if session == nil || store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "DOMParser.parseFromString is unavailable in this bounded classic-JS slice")
	}
	source := script.ToJSString(args[0])
	mimeType := strings.ToLower(strings.TrimSpace(script.ToJSString(args[1])))

	switch mimeType {
	case "image/svg+xml":
		rootID, err := browserDOMParserParseSVGDocument(store, source)
		if err != nil {
			return browserDOMParserParserErrorDocument(session, store, mimeType, err), nil
		}
		return browserDOMParserDocumentValue(session, rootID, store, mimeType), nil
	default:
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("DOMParser.parseFromString unsupported mime type %q in this bounded classic-JS slice", mimeType))
	}
}

func browserDOMParserParseSVGDocument(store *dom.Store, source string) (dom.NodeID, error) {
	if store == nil {
		return 0, fmt.Errorf("dom store is nil")
	}
	decoder := xml.NewDecoder(strings.NewReader(source))
	var rootID dom.NodeID
	var stack []dom.NodeID
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				if rootID == 0 {
					return 0, fmt.Errorf("missing root element")
				}
				if len(stack) != 0 {
					return 0, fmt.Errorf("unclosed SVG element")
				}
				return rootID, nil
			}
			return 0, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			name := strings.TrimSpace(token.Name.Local)
			if name == "" {
				return 0, fmt.Errorf("missing root element name")
			}
			nodeID, err := store.CreateElement(name)
			if err != nil {
				return 0, err
			}
			if node := store.Node(nodeID); node != nil {
				node.NamespaceURI = browserDOMParserSVGNamespaceURI
			}
			for _, attr := range token.Attr {
				attrName := strings.TrimSpace(attr.Name.Local)
				if attrName == "" {
					continue
				}
				if err := store.SetAttribute(nodeID, attrName, attr.Value); err != nil {
					return 0, err
				}
			}
			if node := store.Node(nodeID); node != nil && len(node.Attrs) > 1 {
				sort.SliceStable(node.Attrs, func(i, j int) bool {
					return node.Attrs[i].Name < node.Attrs[j].Name
				})
			}
			if len(stack) > 0 {
				if err := store.AppendChild(stack[len(stack)-1], nodeID); err != nil {
					return 0, err
				}
			} else {
				if rootID != 0 {
					return 0, fmt.Errorf("multiple root elements")
				}
				rootID = nodeID
			}
			stack = append(stack, nodeID)
		case xml.EndElement:
			if len(stack) == 0 {
				return 0, fmt.Errorf("unexpected closing tag </%s>", token.Name.Local)
			}
			nodeID := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			node := store.Node(nodeID)
			if node == nil {
				return 0, fmt.Errorf("invalid SVG tree state")
			}
			if node.TagName != strings.ToLower(strings.TrimSpace(token.Name.Local)) {
				return 0, fmt.Errorf("unexpected closing tag </%s>", token.Name.Local)
			}
		case xml.CharData:
			text := string(token)
			if len(stack) == 0 {
				if strings.TrimSpace(text) == "" {
					continue
				}
				return 0, fmt.Errorf("text outside root element")
			}
			if text == "" {
				continue
			}
			textID, err := store.CreateTextNode(text)
			if err != nil {
				return 0, err
			}
			if err := store.AppendChild(stack[len(stack)-1], textID); err != nil {
				return 0, err
			}
		default:
			continue
		}
	}
}

func browserDOMParserParserErrorDocument(session *Session, store *dom.Store, contentType string, parseErr error) script.Value {
	rootID, err := store.CreateElement("parsererror")
	if err != nil {
		return script.UndefinedValue()
	}
	if node := store.Node(rootID); node != nil {
		node.NamespaceURI = browserDOMParserParserErrorNamespaceURI
	}
	return browserDOMParserDocumentValue(session, rootID, store, contentType)
}

func browserDOMParserDocumentValue(session *Session, rootID dom.NodeID, store *dom.Store, contentType string) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{
			Key:   "documentElement",
			Value: browserElementReferenceValue(rootID, store),
		},
		{Key: "contentType", Value: script.StringValue(contentType)},
		{
			Key: "getElementsByTagName",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 1 {
					return script.UndefinedValue(), fmt.Errorf("Document.getElementsByTagName expects 1 argument")
				}
				if session == nil || store == nil {
					return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Document.getElementsByTagName is unavailable in this bounded classic-JS slice")
				}
				tagName, err := scriptStringArg("Document.getElementsByTagName", args, 0)
				if err != nil {
					return script.UndefinedValue(), err
				}
				return browserNodeListValue(session, store, browserDOMParserElementsByTagName(store, rootID, tagName))
			}),
		},
	})
}

func browserDOMParserElementsByTagName(store *dom.Store, rootID dom.NodeID, tagName string) []dom.NodeID {
	if store == nil {
		return []dom.NodeID{}
	}
	normalized := strings.ToLower(strings.TrimSpace(tagName))
	if normalized == "" {
		return []dom.NodeID{}
	}

	out := make([]dom.NodeID, 0, 4)
	var visit func(dom.NodeID)
	visit = func(nodeID dom.NodeID) {
		node := store.Node(nodeID)
		if node == nil {
			return
		}
		if node.Kind == dom.NodeKindElement && (normalized == "*" || node.TagName == normalized) {
			out = append(out, nodeID)
		}
		for _, childID := range node.Children {
			visit(childID)
		}
	}
	visit(rootID)
	return out
}
