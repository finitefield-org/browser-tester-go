package runtime

import (
	"fmt"
	"math"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func browserGlobalBindings(session *Session, store *dom.Store) map[string]script.Value {
	windowRef := script.HostObjectReference("window")
	documentRef := script.HostObjectReference("document")
	locationRef := script.HostObjectReference("location")
	historyRef := script.HostObjectReference("history")
	navigatorRef := script.HostObjectReference("navigator")
	intlRef := script.HostObjectReference("Intl")
	localStorageRef := script.HostObjectReference("localStorage")
	sessionStorageRef := script.HostObjectReference("sessionStorage")
	consoleRef := script.HostObjectReference("console")
	clipboardRef := script.HostObjectReference("clipboard")

	return map[string]script.Value{
		"Array":                 script.HostFunctionReference("Array"),
		"Object":                script.HostFunctionReference("Object"),
		"JSON":                  script.HostObjectReference("JSON"),
		"Number":                script.HostFunctionReference("Number"),
		"String":                script.HostFunctionReference("String"),
		"Boolean":               script.HostFunctionReference("Boolean"),
		"Math":                  script.HostObjectReference("Math"),
		"Date":                  script.HostFunctionReference("Date"),
		"window":                windowRef,
		"self":                  windowRef,
		"globalThis":            windowRef,
		"top":                   windowRef,
		"parent":                windowRef,
		"frames":                windowRef,
		"document":              documentRef,
		"location":              locationRef,
		"history":               historyRef,
		"navigator":             navigatorRef,
		"name":                  script.HostObjectReference("name"),
		"URL":                   script.HostConstructorReference("URL"),
		"URLSearchParams":       script.HostConstructorReference("URLSearchParams"),
		"Intl":                  intlRef,
		"localStorage":          localStorageRef,
		"sessionStorage":        sessionStorageRef,
		"matchMedia":            script.HostFunctionReference("matchMedia"),
		"addEventListener":      script.HostFunctionReference("addEventListener"),
		"removeEventListener":   script.HostFunctionReference("removeEventListener"),
		"confirm":               script.HostFunctionReference("confirm"),
		"prompt":                script.HostFunctionReference("prompt"),
		"open":                  script.HostFunctionReference("open"),
		"close":                 script.HostFunctionReference("close"),
		"print":                 script.HostFunctionReference("print"),
		"scrollTo":              script.HostFunctionReference("scrollTo"),
		"scrollBy":              script.HostFunctionReference("scrollBy"),
		"setTimeout":            script.HostFunctionReference("setTimeout"),
		"setInterval":           script.HostFunctionReference("setInterval"),
		"clearTimeout":          script.HostFunctionReference("clearTimeout"),
		"clearInterval":         script.HostFunctionReference("clearInterval"),
		"requestAnimationFrame": script.HostFunctionReference("requestAnimationFrame"),
		"cancelAnimationFrame":  script.HostFunctionReference("cancelAnimationFrame"),
		"queueMicrotask":        script.HostFunctionReference("queueMicrotask"),
		"console":               consoleRef,
		"clipboard":             clipboardRef,
	}
}

func (h *inlineScriptHost) ResolveHostReference(path string) (script.Value, error) {
	return resolveBrowserGlobalReference(h.session, h.currentStore(), path)
}

func resolveBrowserGlobalReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "unsupported browser surface \"\" in this bounded classic-JS slice")
	}

	for _, prefix := range []string{"window.", "self.", "globalThis.", "top.", "parent.", "frames."} {
		if strings.HasPrefix(normalized, prefix) {
			return resolveBrowserGlobalReference(session, store, normalized[len(prefix):])
		}
	}

	if value, ok, err := resolveStdlibReference(session, store, normalized); ok || err != nil {
		return value, err
	}

	if strings.HasPrefix(normalized, "url:") {
		return resolveURLInstanceReference(session, normalized)
	}

	switch normalized {
	case "window", "self", "globalThis", "top", "parent", "frames":
		return script.HostObjectReference("window"), nil
	case "name":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "window.name is unavailable in this bounded classic-JS slice")
		}
		return script.StringValue(session.WindowName()), nil
	case "document":
		return script.HostObjectReference("document"), nil
	case "location":
		return script.HostObjectReference("location"), nil
	case "history":
		return script.HostObjectReference("history"), nil
	case "navigator":
		return script.HostObjectReference("navigator"), nil
	case "Intl":
		return script.HostObjectReference("Intl"), nil
	case "lucide":
		return script.ObjectValue([]script.ObjectEntry{
			{
				Key: "createIcons",
				Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
					return script.UndefinedValue(), nil
				}),
			},
		}), nil
	case "URL":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserURLConstructor(session, args)
		}), nil
	case "URLSearchParams":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserURLSearchParamsConstructor(args)
		}), nil
	case "localStorage":
		return script.HostObjectReference("localStorage"), nil
	case "sessionStorage":
		return script.HostObjectReference("sessionStorage"), nil
	case "matchMedia":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserMatchMedia(session, args)
		}), nil
	case "addEventListener":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserWindowAddEventListener(session, store, args)
		}), nil
	case "removeEventListener":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserWindowRemoveEventListener(session, store, args)
		}), nil
	case "confirm":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserConfirm(session, args)
		}), nil
	case "prompt":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserPrompt(session, args)
		}), nil
	case "open":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserOpen(session, args)
		}), nil
	case "close":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserClose(session, args)
		}), nil
	case "print":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserPrint(session, args)
		}), nil
	case "scrollTo":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserScrollTo(session, args)
		}), nil
	case "scrollBy":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserScrollBy(session, args)
		}), nil
	case "setTimeout":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserSetTimeout(session, args)
		}), nil
	case "setInterval":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserSetInterval(session, args)
		}), nil
	case "clearTimeout":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserClearTimeout(session, args)
		}), nil
	case "clearInterval":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserClearInterval(session, args)
		}), nil
	case "requestAnimationFrame":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserRequestAnimationFrame(session, args)
		}), nil
	case "cancelAnimationFrame":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserCancelAnimationFrame(session, args)
		}), nil
	case "queueMicrotask":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserQueueMicrotask(session, args)
		}), nil
	case "console":
		return script.HostObjectReference("console"), nil
	case "clipboard":
		return script.HostObjectReference("clipboard"), nil
	}

	if strings.HasPrefix(normalized, "document.") {
		return resolveDocumentReference(session, store, strings.TrimPrefix(normalized, "document."))
	}
	if strings.HasPrefix(normalized, "location.") {
		return resolveLocationReference(session, strings.TrimPrefix(normalized, "location."))
	}
	if strings.HasPrefix(normalized, "history.") {
		return resolveHistoryReference(session, strings.TrimPrefix(normalized, "history."))
	}
	if strings.HasPrefix(normalized, "navigator.") {
		return resolveNavigatorReference(session, store, strings.TrimPrefix(normalized, "navigator."))
	}
	if strings.HasPrefix(normalized, "Intl.") {
		return resolveIntlReference(session, strings.TrimPrefix(normalized, "Intl."))
	}
	if strings.HasPrefix(normalized, "localStorage.") {
		return resolveStorageReference(session, "local", strings.TrimPrefix(normalized, "localStorage."))
	}
	if strings.HasPrefix(normalized, "sessionStorage.") {
		return resolveStorageReference(session, "session", strings.TrimPrefix(normalized, "sessionStorage."))
	}
	if strings.HasPrefix(normalized, "console.") {
		return resolveConsoleReference(strings.TrimPrefix(normalized, "console."))
	}
	if strings.HasPrefix(normalized, "clipboard.") {
		return resolveClipboardReference(session, strings.TrimPrefix(normalized, "clipboard."))
	}
	if strings.HasPrefix(normalized, "url:") {
		return resolveURLInstanceReference(session, normalized)
	}
	if strings.HasPrefix(normalized, "element:") {
		return resolveElementReference(session, store, normalized)
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
}

func resolveDocumentReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.HostObjectReference("document"), nil
	case "title":
		return resolveDocumentTitleValue(session, store)
	case "readyState":
		return resolveDocumentReadyStateValue(session, store)
	case "activeElement":
		return resolveDocumentActiveElementValue(session, store)
	case "baseURI", "URL", "documentURI":
		return resolveDocumentURLValue(session, store, rest)
	case "doctype":
		return resolveDocumentDoctypeValue(session, store)
	case "defaultView":
		return resolveDocumentDefaultViewValue(session, store)
	case "compatMode":
		return resolveDocumentCompatModeValue(session, store)
	case "contentType":
		return resolveDocumentContentTypeValue(session, store)
	case "designMode":
		return resolveDocumentDesignModeValue(session, store)
	case "dir":
		return resolveDocumentDirValue(session, store)
	case "nodeType", "nodeName", "nodeValue", "ownerDocument", "parentNode", "parentElement", "firstChild", "lastChild", "firstElementChild", "lastElementChild", "nextSibling", "previousSibling", "nextElementSibling", "previousElementSibling", "childElementCount":
		if value, handled, err := resolveNodeTreeNavigationValue(store, store.DocumentID(), "document."+rest, rest); handled || err != nil {
			return value, err
		}
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "document."+rest))
	case "cookie":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document.cookie is unavailable in this bounded classic-JS slice")
		}
		return script.StringValue(session.documentCookie()), nil
	case "children":
		return browserHTMLCollectionValueForDocument(store, func(s *dom.Store) (dom.HTMLCollection, error) {
			return s.Children(s.DocumentID())
		})
	case "childNodes":
		return browserChildNodeListValueForDocument(session, store, func(s *dom.Store) (dom.ChildNodeList, error) {
			return s.ChildNodes(s.DocumentID())
		})
	case "forms":
		return browserHTMLCollectionValueForDocument(store, func(s *dom.Store) (dom.HTMLCollection, error) {
			return s.Forms()
		})
	case "images":
		return browserHTMLCollectionValueForDocument(store, func(s *dom.Store) (dom.HTMLCollection, error) {
			return s.Images()
		})
	case "scripts":
		return browserHTMLCollectionValueForDocument(store, func(s *dom.Store) (dom.HTMLCollection, error) {
			return s.Scripts()
		})
	case "links":
		return browserHTMLCollectionValueForDocument(store, func(s *dom.Store) (dom.HTMLCollection, error) {
			return s.Links()
		})
	case "anchors":
		return browserHTMLCollectionValueForDocument(store, func(s *dom.Store) (dom.HTMLCollection, error) {
			return s.Anchors()
		})
	case "currentScript":
		nodeID := currentInlineScriptNodeID(session, store)
		if nodeID == 0 {
			return script.NullValue(), nil
		}
		return browserElementReferenceValue(nodeID), nil
	case "createElement":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if store == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document.createElement is unavailable in this bounded classic-JS slice")
			}
			tagName, err := scriptStringArg("document.createElement", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if len(args) > 1 {
				return script.UndefinedValue(), fmt.Errorf("document.createElement accepts at most 1 argument")
			}
			nodeID, err := store.CreateElement(tagName)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserElementReferenceValue(nodeID), nil
		}), nil
	case "createTextNode":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if store == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document.createTextNode is unavailable in this bounded classic-JS slice")
			}
			if len(args) == 0 {
				return script.UndefinedValue(), fmt.Errorf("document.createTextNode requires argument %d", 1)
			}
			if len(args) > 1 {
				return script.UndefinedValue(), fmt.Errorf("document.createTextNode accepts at most 1 argument")
			}
			nodeID, err := store.CreateTextNode(script.ToJSString(args[0]))
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserElementReferenceValue(nodeID), nil
		}), nil
	case "addEventListener":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDocumentAddEventListener(session, store, args)
		}), nil
	case "removeEventListener":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDocumentRemoveEventListener(session, store, args)
		}), nil
	case "execCommand":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDocumentExecCommand(session, args)
		}), nil
	case "getElementById":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if store == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document.getElementById is unavailable in this bounded classic-JS slice")
			}
			id, err := scriptStringArg("document.getElementById", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			nodeID, ok, err := store.GetElementByID(id)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if !ok {
				return script.NullValue(), nil
			}
			return browserElementReferenceValue(nodeID), nil
		}), nil
	case "querySelector":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if store == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document.querySelector is unavailable in this bounded classic-JS slice")
			}
			selector, err := scriptStringArg("document.querySelector", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			nodeID, ok, err := store.QuerySelector(selector)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if !ok {
				return script.NullValue(), nil
			}
			return browserElementReferenceValue(nodeID), nil
		}), nil
	case "querySelectorAll":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if store == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document.querySelectorAll is unavailable in this bounded classic-JS slice")
			}
			selector, err := scriptStringArg("document.querySelectorAll", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			nodes, err := store.QuerySelectorAll(selector)
			if err != nil {
				return script.UndefinedValue(), err
			}
			value, err := browserNodeListValue(session, store, nodes.IDs())
			if err != nil {
				return script.UndefinedValue(), err
			}
			return value, nil
		}), nil
	case "documentElement":
		nodeID := firstDocumentElementNodeID(store)
		if nodeID == 0 {
			return script.NullValue(), nil
		}
		return browserElementReferenceValue(nodeID), nil
	case "body":
		nodeID := firstDocumentElementByTag(store, "body")
		if nodeID == 0 {
			return script.NullValue(), nil
		}
		return browserElementReferenceValue(nodeID), nil
	case "head":
		nodeID := firstDocumentElementByTag(store, "head")
		if nodeID == 0 {
			return script.NullValue(), nil
		}
		return browserElementReferenceValue(nodeID), nil
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "document."+path))
}

func resolveLocationReference(session *Session, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.HostObjectReference("location"), nil
	case "href":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "location.href is unavailable in this bounded classic-JS slice")
		}
		return script.StringValue(session.URL()), nil
	case "origin":
		return locationStringValue(session, (*Session).LocationOrigin)
	case "protocol":
		return locationStringValue(session, (*Session).LocationProtocol)
	case "host":
		return locationStringValue(session, (*Session).LocationHost)
	case "hostname":
		return locationStringValue(session, (*Session).LocationHostname)
	case "port":
		return locationStringValue(session, (*Session).LocationPort)
	case "pathname":
		return locationStringValue(session, (*Session).LocationPathname)
	case "search":
		return locationStringValue(session, (*Session).LocationSearch)
	case "hash":
		return locationStringValue(session, (*Session).LocationHash)
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("location.%s accepts no arguments", rest)
			}
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "location is unavailable in this bounded classic-JS slice")
			}
			return script.StringValue(session.URL()), nil
		}), nil
	case "assign":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "location.assign is unavailable in this bounded classic-JS slice")
			}
			url, err := scriptStringArg("location.assign", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if err := session.AssignLocation(url); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "replace":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "location.replace is unavailable in this bounded classic-JS slice")
			}
			url, err := scriptStringArg("location.replace", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if err := session.ReplaceLocation(url); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "reload":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("location.reload accepts no arguments")
			}
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "location.reload is unavailable in this bounded classic-JS slice")
			}
			if err := session.ReloadLocation(); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "location."+rest))
}

func locationStringValue(session *Session, getter func(*Session) (string, error)) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "location is unavailable in this bounded classic-JS slice")
	}
	value, err := getter(session)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.StringValue(value), nil
}

func resolveHistoryReference(session *Session, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.HostObjectReference("history"), nil
	case "length":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.length is unavailable in this bounded classic-JS slice")
		}
		return script.NumberValue(float64(session.windowHistoryLength())), nil
	case "state":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.state is unavailable in this bounded classic-JS slice")
		}
		state, ok := session.windowHistoryState()
		if !ok {
			return script.NullValue(), nil
		}
		return script.StringValue(state), nil
	case "scrollRestoration":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.scrollRestoration is unavailable in this bounded classic-JS slice")
		}
		return script.StringValue(session.windowHistoryScrollRestoration()), nil
	case "pushState":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.pushState is unavailable in this bounded classic-JS slice")
			}
			if len(args) < 2 || len(args) > 3 {
				return script.UndefinedValue(), fmt.Errorf("history.pushState expects 2 or 3 arguments")
			}
			state := script.ToJSString(args[0])
			title := script.ToJSString(args[1])
			url := ""
			if len(args) == 3 {
				url = script.ToJSString(args[2])
			}
			if err := session.windowHistoryPushState(state, title, url); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "replaceState":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.replaceState is unavailable in this bounded classic-JS slice")
			}
			if len(args) < 2 || len(args) > 3 {
				return script.UndefinedValue(), fmt.Errorf("history.replaceState expects 2 or 3 arguments")
			}
			state := script.ToJSString(args[0])
			title := script.ToJSString(args[1])
			url := ""
			if len(args) == 3 {
				url = script.ToJSString(args[2])
			}
			if err := session.windowHistoryReplaceState(state, title, url); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "back":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("history.back accepts no arguments")
			}
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.back is unavailable in this bounded classic-JS slice")
			}
			if err := session.windowHistoryBack(); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "forward":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("history.forward accepts no arguments")
			}
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.forward is unavailable in this bounded classic-JS slice")
			}
			if err := session.windowHistoryForward(); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "go":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "history.go is unavailable in this bounded classic-JS slice")
			}
			delta := int64(0)
			if len(args) > 0 {
				if len(args) > 1 {
					return script.UndefinedValue(), fmt.Errorf("history.go accepts at most 1 argument")
				}
				value, err := browserInt64Value("history.go", args[0])
				if err != nil {
					return script.UndefinedValue(), err
				}
				delta = value
			}
			if err := session.windowHistoryGo(delta); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "history."+rest))
}

func resolveNavigatorReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.HostObjectReference("navigator"), nil
	case "onLine":
		return script.BoolValue(true), nil
	case "language":
		return script.StringValue("en-US"), nil
	case "cookieEnabled":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "navigator.cookieEnabled is unavailable in this bounded classic-JS slice")
		}
		return script.BoolValue(session.navigatorCookieEnabled()), nil
	case "clipboard":
		return script.HostObjectReference("clipboard"), nil
	}

	if strings.HasPrefix(rest, "clipboard.") {
		return resolveClipboardReference(session, strings.TrimPrefix(rest, "clipboard."))
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "navigator."+rest))
}

func resolveIntlReference(session *Session, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.HostObjectReference("Intl"), nil
	case "NumberFormat":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserNumberFormatConstructor(args)
		}), nil
	case "DateTimeFormat":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserDateTimeFormatConstructor(args)
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "Intl."+rest))
}

func resolveStorageReference(session *Session, scope string, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		if scope == "local" {
			return script.HostObjectReference("localStorage"), nil
		}
		return script.HostObjectReference("sessionStorage"), nil
	case "length":
		if session == nil {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "storage length is unavailable in this bounded classic-JS slice")
		}
		if scope == "local" {
			return script.NumberValue(float64(session.localStorageLength())), nil
		}
		return script.NumberValue(float64(session.sessionStorageLength())), nil
	case "getItem":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "storage.getItem is unavailable in this bounded classic-JS slice")
			}
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("storage.getItem expects 1 argument")
			}
			key, err := scriptStringArg("storage.getItem", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			var value string
			var ok bool
			if scope == "local" {
				value, ok = session.localStorageGetItem(key)
			} else {
				value, ok = session.sessionStorageGetItem(key)
			}
			if !ok {
				return script.NullValue(), nil
			}
			return script.StringValue(value), nil
		}), nil
	case "setItem":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "storage.setItem is unavailable in this bounded classic-JS slice")
			}
			if len(args) != 2 {
				return script.UndefinedValue(), fmt.Errorf("storage.setItem expects 2 arguments")
			}
			key, err := scriptStringArg("storage.setItem", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			value, err := scriptStringArg("storage.setItem", args, 1)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if scope == "local" {
				err = session.localStorageSetItem(key, value)
			} else {
				err = session.sessionStorageSetItem(key, value)
			}
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "removeItem":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "storage.removeItem is unavailable in this bounded classic-JS slice")
			}
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("storage.removeItem expects 1 argument")
			}
			key, err := scriptStringArg("storage.removeItem", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if scope == "local" {
				err = session.localStorageRemoveItem(key)
			} else {
				err = session.sessionStorageRemoveItem(key)
			}
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "clear":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("storage.clear accepts no arguments")
			}
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "storage.clear is unavailable in this bounded classic-JS slice")
			}
			var err error
			if scope == "local" {
				err = session.localStorageClear()
			} else {
				err = session.sessionStorageClear()
			}
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		}), nil
	case "key":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "storage.key is unavailable in this bounded classic-JS slice")
			}
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("storage.key expects 1 argument")
			}
			index, err := browserInt64Value("storage.key", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			var key string
			var ok bool
			if scope == "local" {
				key, ok = session.localStorageKey(int(index))
			} else {
				key, ok = session.sessionStorageKey(int(index))
			}
			if !ok {
				return script.NullValue(), nil
			}
			return script.StringValue(key), nil
		}), nil
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", func() string {
		if scope == "local" {
			return "localStorage." + rest
		}
		return "sessionStorage." + rest
	}()))
}

func resolveConsoleReference(path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.HostObjectReference("console"), nil
	case "log", "info", "warn", "error", "debug", "trace", "dir", "table", "group", "groupEnd", "time", "timeEnd":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return script.UndefinedValue(), nil
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "console."+rest))
}

func resolveClipboardReference(session *Session, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.HostObjectReference("clipboard"), nil
	case "readText":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "clipboard.readText is unavailable in this bounded classic-JS slice")
			}
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("clipboard.readText accepts no arguments")
			}
			text, err := session.ReadClipboard()
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.PromiseValue(script.StringValue(text)), nil
		}), nil
	case "writeText":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "clipboard.writeText is unavailable in this bounded classic-JS slice")
			}
			text, err := scriptStringArg("clipboard.writeText", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if err := session.WriteClipboard(text); err != nil {
				return script.UndefinedValue(), err
			}
			return script.PromiseValue(script.UndefinedValue()), nil
		}), nil
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "clipboard."+rest))
}

func resolveElementReference(session *Session, store *dom.Store, path string) (script.Value, error) {
	normalized := strings.TrimSpace(path)
	nodeID, err := parseElementReferencePath(normalized)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if nodeID == 0 {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}

	base := "element:" + strconv.FormatInt(int64(nodeID), 10)
	rest := strings.TrimPrefix(normalized, base)
	rest = strings.TrimPrefix(rest, ".")

	if rest == "" {
		return browserElementReferenceValue(nodeID), nil
	}

	if strings.HasPrefix(rest, "style.") {
		return resolveElementStylePropertyValue(session, store, nodeID, strings.TrimPrefix(rest, "style."))
	}
	if strings.HasPrefix(rest, "attributes.") {
		return resolveElementAttributesPropertyValue(session, store, nodeID, strings.TrimPrefix(rest, "attributes."))
	}
	if strings.HasPrefix(rest, "classList.") {
		return resolveElementClassListPropertyValue(session, store, nodeID, strings.TrimPrefix(rest, "classList."))
	}

	node := nodeFromStore(store, nodeID)
	if node == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid element reference %q in this bounded classic-JS slice", path))
	}

	switch rest {
	case "nodeType", "nodeName", "nodeValue", "ownerDocument", "parentNode", "parentElement",
		"firstChild", "lastChild", "firstElementChild", "lastElementChild", "nextSibling",
		"previousSibling", "nextElementSibling", "previousElementSibling", "childElementCount":
		if value, handled, err := resolveNodeTreeNavigationValue(store, nodeID, "element:"+strconv.FormatInt(int64(nodeID), 10)+"."+rest, rest); handled || err != nil {
			return value, err
		}
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "element:"+strconv.FormatInt(int64(nodeID), 10)+"."+rest))
	case "id":
		value, ok := domAttributeValue(store, nodeID, "id")
		if !ok {
			return script.StringValue(""), nil
		}
		return script.StringValue(value), nil
	case "className":
		return resolveElementClassNameValue(session, store, nodeID)
	case "classList":
		if node.Kind != dom.NodeKindElement {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "element:"+strconv.FormatInt(int64(nodeID), 10)+"."+rest))
		}
		return script.HostObjectReference(base + ".classList"), nil
	case "innerText":
		return resolveElementInnerTextValue(session, store, nodeID)
	case "outerText":
		return resolveElementOuterTextValue(session, store, nodeID)
	case "open":
		return resolveElementOpenValue(session, store, nodeID)
	case "addEventListener":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserElementAddEventListener(session, store, nodeID, args)
		}), nil
	case "removeEventListener":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserElementRemoveEventListener(session, store, nodeID, args)
		}), nil
	case "setAttribute":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserElementSetAttribute(session, store, nodeID, args)
		}), nil
	case "removeAttribute":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserElementRemoveAttribute(session, store, nodeID, args)
		}), nil
	case "appendChild":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserElementAppendChild(session, store, nodeID, args)
		}), nil
	case "removeChild":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserElementRemoveChild(session, store, nodeID, args)
		}), nil
	case "select":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserElementSelect(session, store, nodeID, args)
		}), nil
	case "style":
		if node.Kind != dom.NodeKindElement {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "element:"+strconv.FormatInt(int64(nodeID), 10)+"."+rest))
		}
		return script.HostObjectReference(base + ".style"), nil
	case "attributes":
		if node.Kind != dom.NodeKindElement {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "element:"+strconv.FormatInt(int64(nodeID), 10)+"."+rest))
		}
		return script.HostObjectReference(base + ".attributes"), nil
	case "tagName":
		return script.StringValue(strings.ToUpper(node.TagName)), nil
	case "textContent":
		return script.StringValue(store.TextContentForNode(nodeID)), nil
	case "innerHTML":
		value, err := store.InnerHTMLForNode(nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.StringValue(value), nil
	case "outerHTML":
		value, err := store.OuterHTMLForNode(nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return script.StringValue(value), nil
	case "value":
		return script.StringValue(store.ValueForNode(nodeID)), nil
	case "checked":
		value, ok := store.CheckedForNode(nodeID)
		if !ok {
			return script.BoolValue(false), nil
		}
		return script.BoolValue(value), nil
	case "children":
		coll, err := store.Children(nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return browserHTMLCollectionValue(coll), nil
	case "childNodes":
		nodes, err := store.ChildNodes(nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		value, err := browserChildNodeListValue(session, store, nodes)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return value, nil
	case "querySelector":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			selector, err := scriptStringArg("element.querySelector", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			matched, ok, err := store.QuerySelectorWithin(nodeID, selector)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if !ok {
				return script.NullValue(), nil
			}
			return browserElementReferenceValue(matched), nil
		}), nil
	case "querySelectorAll":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			selector, err := scriptStringArg("element.querySelectorAll", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			nodes, err := store.QuerySelectorAllWithin(nodeID, selector)
			if err != nil {
				return script.UndefinedValue(), err
			}
			value, err := browserNodeListValue(session, store, nodes.IDs())
			if err != nil {
				return script.UndefinedValue(), err
			}
			return value, nil
		}), nil
	case "matches":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			selector, err := scriptStringArg("element.matches", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			matched, err := store.Matches(nodeID, selector)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.BoolValue(matched), nil
		}), nil
	case "closest":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			selector, err := scriptStringArg("element.closest", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			matched, ok, err := store.Closest(nodeID, selector)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if !ok {
				return script.NullValue(), nil
			}
			return browserElementReferenceValue(matched), nil
		}), nil
	case "getAttribute":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			name, err := scriptStringArg("element.getAttribute", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			value, ok, err := store.GetAttribute(nodeID, name)
			if err != nil {
				return script.UndefinedValue(), err
			}
			if !ok {
				return script.NullValue(), nil
			}
			return script.StringValue(value), nil
		}), nil
	case "hasAttribute":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			name, err := scriptStringArg("element.hasAttribute", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			ok, err := store.HasAttribute(nodeID, name)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return script.BoolValue(ok), nil
		}), nil
	case "form":
		if node.TagName == "input" || node.TagName == "select" || node.TagName == "textarea" || node.TagName == "button" || node.TagName == "output" {
			if formID := closestAncestorTag(store, nodeID, "form"); formID != 0 {
				return browserElementReferenceValue(formID), nil
			}
		}
		return script.NullValue(), nil
	case "elements":
		switch node.TagName {
		case "form":
			coll, err := store.FormElements(nodeID)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserHTMLCollectionValue(coll), nil
		case "fieldset":
			coll, err := store.FieldsetElements(nodeID)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserHTMLCollectionValue(coll), nil
		}
		return script.NullValue(), nil
	case "options":
		switch node.TagName {
		case "select":
			coll, err := store.Options(nodeID)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserHTMLCollectionValue(coll), nil
		case "datalist":
			coll, err := store.DatalistOptions(nodeID)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserHTMLCollectionValue(coll), nil
		}
		return script.NullValue(), nil
	case "selectedOptions":
		if node.TagName != "select" {
			return script.NullValue(), nil
		}
		coll, err := store.SelectedOptions(nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return browserHTMLCollectionValue(coll), nil
	case "rows":
		switch node.TagName {
		case "table":
			coll, err := store.Rows(nodeID)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserHTMLCollectionValue(coll), nil
		case "tbody", "thead", "tfoot":
			coll, err := store.Rows(nodeID)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserHTMLCollectionValue(coll), nil
		case "tr":
			coll, err := store.Cells(nodeID)
			if err != nil {
				return script.UndefinedValue(), err
			}
			return browserHTMLCollectionValue(coll), nil
		}
		return script.NullValue(), nil
	case "tBodies":
		if node.TagName != "table" {
			return script.NullValue(), nil
		}
		coll, err := store.TBodies(nodeID)
		if err != nil {
			return script.UndefinedValue(), err
		}
		return browserHTMLCollectionValue(coll), nil
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "element:"+strconv.FormatInt(int64(nodeID), 10)+"."+rest))
}

func resolveURLReference(session *Session, path string) (script.Value, error) {
	rest := strings.TrimPrefix(path, ".")
	switch rest {
	case "":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserURLConstructor(session, args)
		}), nil
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("URL.%s accepts no arguments", rest)
			}
			if session == nil {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "URL is unavailable in this bounded classic-JS slice")
			}
			return script.StringValue(session.URL()), nil
		}), nil
	}
	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "URL."+rest))
}

func protocolFromURL(parsed *neturl.URL) string {
	if parsed == nil || parsed.Scheme == "" {
		return ""
	}
	return parsed.Scheme + ":"
}

func pathnameFromURL(parsed *neturl.URL) string {
	if parsed == nil {
		return ""
	}
	pathname := parsed.EscapedPath()
	if pathname == "" {
		return "/"
	}
	return pathname
}

func searchFromURL(parsed *neturl.URL) string {
	if parsed == nil {
		return ""
	}
	if parsed.RawQuery == "" {
		if parsed.ForceQuery {
			return "?"
		}
		return ""
	}
	return "?" + parsed.RawQuery
}

func hashFromURL(parsed *neturl.URL) string {
	if parsed == nil {
		return ""
	}
	fragment := parsed.EscapedFragment()
	if fragment == "" {
		return ""
	}
	return "#" + fragment
}

func browserNumberFormatConstructor(args []script.Value) (script.Value, error) {
	maxFractionDigits := -1
	if len(args) > 1 {
		if args[1].Kind != script.ValueKindObject {
			return script.UndefinedValue(), fmt.Errorf("Intl.NumberFormat options argument must be an object")
		}
		if value, ok := objectProperty(args[1], "maximumFractionDigits"); ok {
			if value.Kind != script.ValueKindNumber && value.Kind != script.ValueKindBigInt {
				return script.UndefinedValue(), fmt.Errorf("Intl.NumberFormat maximumFractionDigits must be numeric")
			}
			parsed, err := browserInt64Value("Intl.NumberFormat.maximumFractionDigits", value)
			if err != nil {
				return script.UndefinedValue(), err
			}
			maxFractionDigits = int(parsed)
		}
	}

	entries := []script.ObjectEntry{
		{
			Key: "format",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 1 {
					return script.UndefinedValue(), fmt.Errorf("Intl.NumberFormat#format expects 1 argument")
				}
				number, err := browserFloat64Value("Intl.NumberFormat#format", args[0])
				if err != nil {
					return script.UndefinedValue(), err
				}
				return script.StringValue(formatNumber(number, maxFractionDigits)), nil
			}),
		},
	}
	return script.ObjectValue(entries), nil
}

func browserDateTimeFormatConstructor(args []script.Value) (script.Value, error) {
	locale := "en-US"
	var options script.Value
	hasOptions := false

	switch len(args) {
	case 0:
	case 1:
		if args[0].Kind == script.ValueKindObject {
			options = args[0]
			hasOptions = true
		} else if args[0].Kind != script.ValueKindUndefined && args[0].Kind != script.ValueKindNull {
			locale = strings.TrimSpace(script.ToJSString(args[0]))
		}
	default:
		if args[0].Kind != script.ValueKindUndefined && args[0].Kind != script.ValueKindNull {
			locale = strings.TrimSpace(script.ToJSString(args[0]))
		}
		if args[1].Kind != script.ValueKindObject {
			return script.UndefinedValue(), fmt.Errorf("Intl.DateTimeFormat options argument must be an object")
		}
		options = args[1]
		hasOptions = true
	}

	if locale == "" {
		locale = "en-US"
	}

	hour12 := strings.HasPrefix(strings.ToLower(locale), "en")
	if hasOptions {
		if value, ok := objectProperty(options, "hour12"); ok && value.Kind == script.ValueKindBool {
			hour12 = value.Bool
		}
	}

	entries := []script.ObjectEntry{
		{
			Key: "format",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if len(args) != 1 {
					return script.UndefinedValue(), fmt.Errorf("Intl.DateTimeFormat#format expects 1 argument")
				}
				ms, err := browserInt64Value("Intl.DateTimeFormat#format", args[0])
				if err != nil {
					return script.UndefinedValue(), err
				}
				return script.StringValue(formatDateTime(ms, locale, hour12)), nil
			}),
		},
	}
	return script.ObjectValue(entries), nil
}

func formatNumber(value float64, maxFractionDigits int) string {
	if math.IsNaN(value) {
		return "NaN"
	}
	if math.IsInf(value, 1) {
		return "Infinity"
	}
	if math.IsInf(value, -1) {
		return "-Infinity"
	}
	if maxFractionDigits < 0 {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	pow := math.Pow10(maxFractionDigits)
	rounded := math.Round(value*pow) / pow
	text := strconv.FormatFloat(rounded, 'f', maxFractionDigits, 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

func formatDateTime(ms int64, locale string, hour12 bool) string {
	t := time.UnixMilli(ms).UTC()
	normalized := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(normalized, "ja") {
		return t.Format("2006/01/02 15:04")
	}
	if hour12 {
		return t.Format("01/02/2006, 03:04 PM")
	}
	return t.Format("01/02/2006, 15:04")
}

func browserMatchMedia(session *Session, args []script.Value) (script.Value, error) {
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("matchMedia expects 1 argument")
	}
	query, err := scriptStringArg("matchMedia", args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "matchMedia is unavailable in this bounded classic-JS slice")
	}
	matches, err := session.MatchMedia(query)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "matches", Value: script.BoolValue(matches)},
		{Key: "media", Value: script.StringValue(query)},
		{Key: "addListener", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			session.Registry().MatchMedia().RecordListenerCall(query, "addListener")
			return script.UndefinedValue(), nil
		})},
		{Key: "removeListener", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			session.Registry().MatchMedia().RecordListenerCall(query, "removeListener")
			return script.UndefinedValue(), nil
		})},
		{Key: "addEventListener", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			session.Registry().MatchMedia().RecordListenerCall(query, "addEventListener")
			return script.UndefinedValue(), nil
		})},
		{Key: "removeEventListener", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			session.Registry().MatchMedia().RecordListenerCall(query, "removeEventListener")
			return script.UndefinedValue(), nil
		})},
	}), nil
}

func browserOpen(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "open is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("open expects 1 argument")
	}
	url, err := scriptStringArg("open", args, 0)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if err := session.Open(url); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserClose(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "close is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("close expects no arguments")
	}
	if err := session.Close(); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserPrint(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "print is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("print expects no arguments")
	}
	if err := session.Print(); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserScrollTo(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "scrollTo is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 2 {
		return script.UndefinedValue(), fmt.Errorf("scrollTo expects 2 arguments")
	}
	x, err := browserInt64Value("scrollTo", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	y, err := browserInt64Value("scrollTo", args[1])
	if err != nil {
		return script.UndefinedValue(), err
	}
	if err := session.ScrollTo(x, y); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserScrollBy(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "scrollBy is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 2 {
		return script.UndefinedValue(), fmt.Errorf("scrollBy expects 2 arguments")
	}
	x, err := browserInt64Value("scrollBy", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	y, err := browserInt64Value("scrollBy", args[1])
	if err != nil {
		return script.UndefinedValue(), err
	}
	if err := session.ScrollBy(x, y); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserSetTimeout(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "setTimeout is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 || len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("setTimeout expects 1 or 2 arguments")
	}
	source, err := browserTimerSource("setTimeout", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	timeoutMs := int64(0)
	if len(args) == 2 {
		timeoutMs, err = browserInt64Value("setTimeout", args[1])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	id, err := session.scheduleTimeout(source, timeoutMs)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.NumberValue(float64(id)), nil
}

func browserSetInterval(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "setInterval is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 || len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("setInterval expects 1 or 2 arguments")
	}
	source, err := browserTimerSource("setInterval", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	timeoutMs := int64(0)
	if len(args) == 2 {
		timeoutMs, err = browserInt64Value("setInterval", args[1])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	id, err := session.scheduleInterval(source, timeoutMs)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.NumberValue(float64(id)), nil
}

func browserClearTimeout(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "clearTimeout is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("clearTimeout expects at most 1 argument")
	}
	var id int64
	var err error
	if len(args) == 1 {
		id, err = browserInt64Value("clearTimeout", args[0])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	session.clearTimeout(id)
	return script.UndefinedValue(), nil
}

func browserClearInterval(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "clearInterval is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("clearInterval expects at most 1 argument")
	}
	var id int64
	var err error
	if len(args) == 1 {
		id, err = browserInt64Value("clearInterval", args[0])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	session.clearInterval(id)
	return script.UndefinedValue(), nil
}

func browserRequestAnimationFrame(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "requestAnimationFrame is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("requestAnimationFrame expects 1 argument")
	}
	source, err := browserTimerSource("requestAnimationFrame", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	id, err := session.requestAnimationFrame(source)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return script.NumberValue(float64(id)), nil
}

func browserCancelAnimationFrame(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "cancelAnimationFrame is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("cancelAnimationFrame expects at most 1 argument")
	}
	var id int64
	var err error
	if len(args) == 1 {
		id, err = browserInt64Value("cancelAnimationFrame", args[0])
		if err != nil {
			return script.UndefinedValue(), err
		}
	}
	session.cancelAnimationFrame(id)
	return script.UndefinedValue(), nil
}

func browserQueueMicrotask(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "queueMicrotask is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("queueMicrotask expects 1 argument")
	}
	source, err := browserTimerSource("queueMicrotask", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	session.enqueueMicrotask(source)
	return script.UndefinedValue(), nil
}

func browserTimerSource(method string, value script.Value) (string, error) {
	if value.Kind != script.ValueKindString {
		return "", fmt.Errorf("%s callback functions are not supported in this bounded classic-JS slice", method)
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return "", fmt.Errorf("%s source must not be empty", method)
	}
	return trimmed, nil
}

func browserInt64Value(method string, value script.Value) (int64, error) {
	switch value.Kind {
	case script.ValueKindNumber:
		return int64(value.Number), nil
	case script.ValueKindObject:
		if ms, ok := dateObjectMs(value); ok {
			return ms, nil
		}
		return 0, fmt.Errorf("%s argument must be numeric", method)
	case script.ValueKindBigInt:
		bigInt, err := strconv.ParseInt(value.BigInt, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s argument must be numeric", method)
		}
		return bigInt, nil
	case script.ValueKindString:
		trimmed := strings.TrimSpace(value.String)
		if trimmed == "" {
			return 0, fmt.Errorf("%s argument must be numeric", method)
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err == nil {
			return parsed, nil
		}
		floatValue, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, fmt.Errorf("%s argument must be numeric", method)
		}
		return int64(floatValue), nil
	default:
		return 0, fmt.Errorf("%s argument must be numeric", method)
	}
}

func browserFloat64Value(method string, value script.Value) (float64, error) {
	switch value.Kind {
	case script.ValueKindNumber:
		return value.Number, nil
	case script.ValueKindObject:
		if ms, ok := dateObjectMs(value); ok {
			return float64(ms), nil
		}
		return 0, fmt.Errorf("%s argument must be numeric", method)
	case script.ValueKindBigInt:
		parsed, err := strconv.ParseFloat(value.BigInt, 64)
		if err != nil {
			return 0, fmt.Errorf("%s argument must be numeric", method)
		}
		return parsed, nil
	case script.ValueKindString:
		trimmed := strings.TrimSpace(value.String)
		if trimmed == "" {
			return 0, fmt.Errorf("%s argument must be numeric", method)
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, fmt.Errorf("%s argument must be numeric", method)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%s argument must be numeric", method)
	}
}

func browserElementReferenceValue(nodeID dom.NodeID) script.Value {
	if nodeID == 0 {
		return script.NullValue()
	}
	return script.HostObjectReference("element:" + strconv.FormatInt(int64(nodeID), 10))
}

func browserHTMLCollectionValue(coll dom.HTMLCollection) script.Value {
	ids := coll.IDs()
	entries := make([]script.ObjectEntry, 0, len(ids)+3)
	for i, nodeID := range ids {
		entries = append(entries, script.ObjectEntry{
			Key:   strconv.Itoa(i),
			Value: browserElementReferenceValue(nodeID),
		})
	}
	entries = append(entries, script.ObjectEntry{
		Key:   "length",
		Value: script.NumberValue(float64(len(ids))),
	})
	entries = append(entries, script.ObjectEntry{
		Key: "item",
		Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("HTMLCollection.item expects 1 argument")
			}
			index, err := browserInt64Value("HTMLCollection.item", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			nodeID, ok := coll.Item(int(index))
			if !ok {
				return script.NullValue(), nil
			}
			return browserElementReferenceValue(nodeID), nil
		}),
	})
	entries = append(entries, script.ObjectEntry{
		Key: "namedItem",
		Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("HTMLCollection.namedItem expects 1 argument")
			}
			name, err := scriptStringArg("HTMLCollection.namedItem", args, 0)
			if err != nil {
				return script.UndefinedValue(), err
			}
			nodeID, ok := coll.NamedItem(name)
			if !ok {
				return script.NullValue(), nil
			}
			return browserElementReferenceValue(nodeID), nil
		}),
	})
	return script.ObjectValue(entries)
}

func browserHTMLCollectionValueForDocument(store *dom.Store, fn func(*dom.Store) (dom.HTMLCollection, error)) (script.Value, error) {
	if store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document collection is unavailable in this bounded classic-JS slice")
	}
	coll, err := fn(store)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return browserHTMLCollectionValue(coll), nil
}

func browserChildNodeListValueForDocument(session *Session, store *dom.Store, fn func(*dom.Store) (dom.ChildNodeList, error)) (script.Value, error) {
	if store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "document childNodes are unavailable in this bounded classic-JS slice")
	}
	nodes, err := fn(store)
	if err != nil {
		return script.UndefinedValue(), err
	}
	value, err := browserChildNodeListValue(session, store, nodes)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return value, nil
}

func browserHTMLCollectionValueForElement(store *dom.Store, nodeID dom.NodeID, fn func(*dom.Store, dom.NodeID) (dom.HTMLCollection, error)) (script.Value, error) {
	if store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element collection is unavailable in this bounded classic-JS slice")
	}
	coll, err := fn(store, nodeID)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return browserHTMLCollectionValue(coll), nil
}

func browserChildNodeListValueForElement(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	if store == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "element childNodes are unavailable in this bounded classic-JS slice")
	}
	nodes, err := store.ChildNodes(nodeID)
	if err != nil {
		return script.UndefinedValue(), err
	}
	value, err := browserChildNodeListValue(session, store, nodes)
	if err != nil {
		return script.UndefinedValue(), err
	}
	return value, nil
}

func currentInlineScriptNodeID(session *Session, store *dom.Store) dom.NodeID {
	if session == nil || store == nil {
		return 0
	}
	current := strings.TrimSpace(session.currentScriptHTML)
	if current == "" {
		return 0
	}
	nodes, err := store.QuerySelectorAll("script")
	if err != nil {
		return 0
	}
	for _, nodeID := range nodes.IDs() {
		outerHTML, err := store.OuterHTMLForNode(nodeID)
		if err != nil {
			continue
		}
		if outerHTML == current {
			return nodeID
		}
	}
	return 0
}

func firstDocumentElementNodeID(store *dom.Store) dom.NodeID {
	if store == nil {
		return 0
	}
	children, err := store.Children(store.DocumentID())
	if err != nil {
		return 0
	}
	if id, ok := children.Item(0); ok {
		return id
	}
	return 0
}

func firstDocumentElementByTag(store *dom.Store, tag string) dom.NodeID {
	if store == nil {
		return 0
	}
	normalized := strings.ToLower(strings.TrimSpace(tag))
	if normalized == "" {
		return 0
	}
	nodes, err := store.QuerySelectorAll(normalized)
	if err != nil {
		return 0
	}
	if id, ok := nodes.Item(0); ok {
		return id
	}
	return 0
}

func closestAncestorTag(store *dom.Store, nodeID dom.NodeID, tag string) dom.NodeID {
	if store == nil || nodeID == 0 {
		return 0
	}
	normalized := strings.ToLower(strings.TrimSpace(tag))
	if normalized == "" {
		return 0
	}
	current := nodeID
	for current != 0 {
		node := store.Node(current)
		if node == nil {
			return 0
		}
		if node.TagName == normalized {
			return node.ID
		}
		current = node.Parent
	}
	return 0
}

func nodeFromStore(store *dom.Store, nodeID dom.NodeID) *dom.Node {
	if store == nil || nodeID == 0 {
		return nil
	}
	return store.Node(nodeID)
}

func domAttributeValue(store *dom.Store, nodeID dom.NodeID, name string) (string, bool) {
	if store == nil {
		return "", false
	}
	value, ok, err := store.GetAttribute(nodeID, name)
	if err != nil || !ok {
		return "", false
	}
	return value, true
}

func objectProperty(value script.Value, key string) (script.Value, bool) {
	for i := len(value.Object) - 1; i >= 0; i-- {
		if value.Object[i].Key == key {
			return value.Object[i].Value, true
		}
	}
	return script.Value{}, false
}

func parseElementReferencePath(path string) (dom.NodeID, error) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "element:") {
		return 0, fmt.Errorf("invalid element reference %q", path)
	}
	remainder := strings.TrimPrefix(normalized, "element:")
	base := remainder
	if index := strings.IndexByte(remainder, '.'); index >= 0 {
		base = remainder[:index]
	}
	if base == "" {
		return 0, fmt.Errorf("invalid element reference %q", path)
	}
	id, err := strconv.ParseInt(base, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid element reference %q", path)
	}
	return dom.NodeID(id), nil
}
