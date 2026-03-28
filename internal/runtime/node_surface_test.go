package runtime

import (
	"strconv"
	"strings"
	"testing"

	"browsertester/internal/script"
)

func TestNodeTreeNavigationBridgeDuringAndAfterBootstrap(t *testing.T) {
	session := NewSession(SessionConfig{
		URL:  "https://example.test/app?mode=tree#nav",
		HTML: `<html><head><title>Tree Navigation</title></head><body><main id="root"><span id="first"></span>gap<b id="second"></b></main><div id="probe"></div></body></html>`,
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if store == nil {
		t.Fatalf("ensureDOM() store = nil, want DOM store")
	}
	if got := session.DOMReady(); !got {
		t.Fatalf("DOMReady() = %v, want true after bootstrap", got)
	}

	safeNodeType := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : " + expr + ".nodeType)"
	}
	safeNodeName := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : " + expr + ".nodeName)"
	}
	safeNodeValue := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : " + expr + ".nodeValue)"
	}
	safeDataValue := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : " + expr + ".data)"
	}
	safeOwnerDocumentType := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : (" + expr + ".ownerDocument == null ? \"null\" : " + expr + ".ownerDocument.nodeType))"
	}
	safeChildNodeName := func(baseExpr, childProp string) string {
		return "(" + baseExpr + " == null ? \"null\" : (" + baseExpr + "." + childProp + " == null ? \"null\" : " + baseExpr + "." + childProp + ".nodeName))"
	}
	safeChildNodeType := func(baseExpr, childProp string) string {
		return "(" + baseExpr + " == null ? \"null\" : (" + baseExpr + "." + childProp + " == null ? \"null\" : " + baseExpr + "." + childProp + ".nodeType))"
	}
	safeChildNodeValue := func(baseExpr, childProp string) string {
		return "(" + baseExpr + " == null ? \"null\" : (" + baseExpr + "." + childProp + " == null ? \"null\" : " + baseExpr + "." + childProp + ".nodeValue))"
	}
	safeChildNodeID := func(baseExpr, childProp string) string {
		return "(" + baseExpr + " == null ? \"null\" : (" + baseExpr + "." + childProp + " == null ? \"null\" : " + baseExpr + "." + childProp + ".id))"
	}
	safeChildCount := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : " + expr + ".childElementCount)"
	}
	safeContains := func(baseExpr, otherExpr string) string {
		return "(" + baseExpr + " == null ? \"null\" : " + baseExpr + ".contains(" + otherExpr + "))"
	}
	safeParentNodeName := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : (" + expr + ".parentNode == null ? \"null\" : " + expr + ".parentNode.nodeName))"
	}
	safeParentElementName := func(expr string) string {
		return "(" + expr + " == null ? \"null\" : (" + expr + ".parentElement == null ? \"null\" : " + expr + ".parentElement.nodeName))"
	}
	runProbe := func(name, expr, want string) {
		scriptSource := `var docEl = document.documentElement; var root = document.querySelector("#root"); var first = document.getElementById("first"); var second = document.getElementById("second"); host:setTextContent("#probe", expr(` + expr + `))`
		if _, err := session.runScriptOnStore(store, scriptSource); err != nil {
			t.Fatalf("runScriptOnStore(%s) error = %v", name, err)
		}
		if got, err := session.TextContent("#probe"); err != nil {
			t.Fatalf("TextContent(#probe) after %s error = %v", name, err)
		} else if got != want {
			t.Fatalf("TextContent(#probe) after %s = %q, want %q", name, got, want)
		}
	}

	probes := []struct {
		name string
		expr string
		want string
	}{
		{name: "document.nodeType", expr: safeNodeType("document"), want: "9"},
		{name: "document.nodeName", expr: safeNodeName("document"), want: "#document"},
		{name: "document.nodeValue", expr: safeNodeValue("document"), want: "null"},
		{name: "document.parentNode", expr: `document.parentNode == null`, want: "true"},
		{name: "document.parentElement", expr: `document.parentElement == null`, want: "true"},
		{name: "document.firstChild", expr: safeChildNodeName("document", "firstChild"), want: "HTML"},
		{name: "document.lastChild", expr: safeChildNodeName("document", "lastChild"), want: "HTML"},
		{name: "document.firstElementChild", expr: safeChildNodeName("document", "firstElementChild"), want: "HTML"},
		{name: "document.lastElementChild", expr: safeChildNodeName("document", "lastElementChild"), want: "HTML"},
		{name: "document.childElementCount", expr: "document.childElementCount", want: "1"},
		{name: "document.documentElement.parentNode", expr: `docEl == null ? "null" : (docEl.parentNode == null ? "null" : docEl.parentNode.nodeName)`, want: "#document"},
		{name: "document.documentElement.parentElement", expr: `docEl == null ? "null" : docEl.parentElement == null`, want: "true"},
		{name: "document.documentElement.firstElementChild", expr: `docEl == null ? "null" : (docEl.firstElementChild == null ? "null" : docEl.firstElementChild.nodeName)`, want: "HEAD"},
		{name: "document.documentElement.lastElementChild", expr: `docEl == null ? "null" : (docEl.lastElementChild == null ? "null" : docEl.lastElementChild.nodeName)`, want: "BODY"},
		{name: "document.documentElement.childElementCount", expr: `docEl == null ? "null" : docEl.childElementCount`, want: "2"},
		{name: "document.contains(document)", expr: `document.contains(document)`, want: "true"},
		{name: "document.contains(root)", expr: safeContains("document", "root"), want: "true"},
		{name: "document.contains(null)", expr: `document.contains(null)`, want: "false"},
		{name: "root.nodeType", expr: safeNodeType("root"), want: "1"},
		{name: "root.nodeName", expr: safeNodeName("root"), want: "MAIN"},
		{name: "root.nodeValue", expr: safeNodeValue("root"), want: "null"},
		{name: "root.ownerDocument", expr: safeOwnerDocumentType("root"), want: "9"},
		{name: "root.parentNode", expr: safeParentNodeName("root"), want: "BODY"},
		{name: "root.parentElement", expr: safeParentElementName("root"), want: "BODY"},
		{name: "root.firstChild", expr: safeChildNodeName("root", "firstChild"), want: "SPAN"},
		{name: "root.lastChild", expr: safeChildNodeName("root", "lastChild"), want: "B"},
		{name: "root.firstElementChild", expr: safeChildNodeName("root", "firstElementChild"), want: "SPAN"},
		{name: "root.lastElementChild", expr: safeChildNodeName("root", "lastElementChild"), want: "B"},
		{name: "root.childElementCount", expr: safeChildCount("root"), want: "2"},
		{name: "first.nodeType", expr: safeNodeType("first"), want: "1"},
		{name: "first.nodeName", expr: safeNodeName("first"), want: "SPAN"},
		{name: "first.nodeValue", expr: safeNodeValue("first"), want: "null"},
		{name: "first.ownerDocument", expr: safeOwnerDocumentType("first"), want: "9"},
		{name: "first.contains(first)", expr: safeContains("first", "first"), want: "true"},
		{name: "first.contains(root)", expr: safeContains("first", "root"), want: "false"},
		{name: "first.contains(null)", expr: safeContains("first", "null"), want: "false"},
		{name: "first.parentNode", expr: safeParentNodeName("first"), want: "MAIN"},
		{name: "first.parentElement", expr: safeParentElementName("first"), want: "MAIN"},
		{name: "first.firstChild", expr: `first.firstChild == null`, want: "true"},
		{name: "first.lastChild", expr: `first.lastChild == null`, want: "true"},
		{name: "first.nextSibling.type", expr: safeChildNodeType("first", "nextSibling"), want: "3"},
		{name: "first.nextSibling.value", expr: safeChildNodeValue("first", "nextSibling"), want: "gap"},
		{name: "first.nextSibling.data", expr: safeDataValue("first.nextSibling"), want: "gap"},
		{name: "first.nextSibling.contains(self)", expr: safeContains("first.nextSibling", "first.nextSibling"), want: "true"},
		{name: "first.nextElementSibling", expr: safeChildNodeName("first", "nextElementSibling"), want: "B"},
		{name: "second.previousSibling.value", expr: safeChildNodeValue("second", "previousSibling"), want: "gap"},
		{name: "second.previousElementSibling.id", expr: safeChildNodeID("second", "previousElementSibling"), want: "first"},
	}

	for _, probe := range probes {
		runProbe(probe.name, probe.expr, probe.want)
	}
}

func TestNodeTreeNavigationBridgeReportsUnsupportedSurfacesExplicitly(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main id="root"><span id="first"></span></main>`})
	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	rootID, ok, err := store.QuerySelector("#root")
	if err != nil {
		t.Fatalf("QuerySelector(#root) error = %v", err)
	}
	if !ok {
		t.Fatalf("QuerySelector(#root) = no match, want root node")
	}

	path := "element:" + strconv.FormatInt(int64(rootID), 10) + ".unknown"
	if _, err := resolveBrowserGlobalReference(session, store, path); err == nil {
		t.Fatalf("resolveBrowserGlobalReference(%s) error = nil, want unsupported error", path)
	} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported || !strings.Contains(scriptErr.Message, path) {
		t.Fatalf("resolveBrowserGlobalReference(%s) error = %#v, want unsupported script error", path, err)
	}
}

func TestNodeTreeNavigationBridgeReportsIsConnectedStatus(t *testing.T) {
	session := NewSession(DefaultSessionConfig())
	if err := session.WriteHTML(`<main><div id="probe"></div><script>const orphan = document.createElement("em"); document.querySelector("#probe").textContent = String(document.isConnected) + ":" + String(document.documentElement.isConnected) + ":" + String(orphan.isConnected)</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	got, err := session.TextContent("#probe")
	if err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	}
	if want := "true:true:false"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestNodeTreeNavigationBridgeReportsRootNodeStatus(t *testing.T) {
	session := NewSession(DefaultSessionConfig())
	if err := session.WriteHTML(`<main><div id="probe"></div><script>const orphan = document.createElement("em"); document.querySelector("#probe").textContent = String(document.getRootNode() === document) + ":" + String(document.documentElement.getRootNode() === document) + ":" + String(orphan.getRootNode() === orphan)</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	got, err := session.TextContent("#probe")
	if err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	}
	if want := "true:true:true"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestTextNodeSplitTextBridgeReadsAdjacentTextNodes(t *testing.T) {
	session := NewSession(DefaultSessionConfig())
	if err := session.WriteHTML(`<main><div id="mirror">hello</div><div id="probe"></div><script>const mirror = document.querySelector("#mirror"); const right = mirror.firstChild.splitText(2); document.querySelector("#probe").textContent = String(mirror.childNodes.length) + ":" + mirror.firstChild.data + ":" + right.data + ":" + mirror.firstChild.wholeText + ":" + right.wholeText</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	got, err := session.TextContent("#probe")
	if err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	}
	if want := "2:he:llo:hello:hello"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
	got, err = session.TextContent("#mirror")
	if err != nil {
		t.Fatalf("TextContent(#mirror) error = %v", err)
	}
	if want := "hello"; got != want {
		t.Fatalf("TextContent(#mirror) = %q, want %q", got, want)
	}
}

func TestTextNodeSplitTextBridgeSupportsDocumentLevelTextNodes(t *testing.T) {
	session := NewSession(DefaultSessionConfig())
	if err := session.WriteHTML(`hello<div id="probe"></div><script>const text = document.firstChild; const right = text.splitText(2); document.querySelector("#probe").textContent = String(document.childNodes.length) + ":" + text.data + ":" + right.data + ":" + text.wholeText + ":" + right.wholeText</script>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	got, err := session.TextContent("#probe")
	if err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	}
	if want := "4:he:llo:hello:hello"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestTextNodeNormalizeBridgeIsNoOp(t *testing.T) {
	session := NewSession(DefaultSessionConfig())
	if err := session.WriteHTML(`hello<div id="probe"></div><script>const text = document.firstChild; text.normalize(); document.querySelector("#probe").textContent = String(text.data) + ":" + text.wholeText</script>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	got, err := session.TextContent("#probe")
	if err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	}
	if want := "hello:hello"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestDocumentNormalizeBridgeMergesAdjacentTextNodes(t *testing.T) {
	session := NewSession(DefaultSessionConfig())
	if err := session.WriteHTML(`hello<div id="inner">abc</div><div id="probe"></div><script>const text = document.firstChild; text.splitText(2); const inner = document.querySelector("#inner").firstChild; inner.splitText(1); const beforeDoc = document.childNodes.length; const beforeInner = inner.parentNode.childNodes.length; document.normalize(); document.querySelector("#probe").textContent = String(beforeDoc) + ":" + String(document.childNodes.length) + ":" + String(beforeInner) + ":" + String(document.querySelector("#inner").childNodes.length) + ":" + text.data + ":" + inner.data + ":" + text.wholeText + ":" + inner.wholeText</script>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	got, err := session.TextContent("#probe")
	if err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	}
	if want := "5:4:2:1:hello:abc:hello:abc"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}
