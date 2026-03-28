package runtime

import (
	"strings"
	"testing"
)

func TestSessionInlineScriptsCanConstructAndReorderNodes(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main></main><script>const root = host:querySelector("main"); const span = host:createElement("span"); const em = host:createElement("em"); const strong = host:createElement("strong"); host:appendChild(expr(root), expr(span)); host:appendChild(expr(root), expr(em)); host:appendChild(expr(root), expr(strong)); host:insertBefore(expr(root), expr(span), expr(strong)); const removed = host:removeChild(expr(root), expr(span)); host:appendChild(expr(root), expr(removed))</script>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main><em></em><strong></strong><span></span></main><script>const root = host:querySelector("main"); const span = host:createElement("span"); const em = host:createElement("em"); const strong = host:createElement("strong"); host:appendChild(expr(root), expr(span)); host:appendChild(expr(root), expr(em)); host:appendChild(expr(root), expr(strong)); host:insertBefore(expr(root), expr(span), expr(strong)); const removed = host:removeChild(expr(root), expr(span)); host:appendChild(expr(root), expr(removed))</script>`; got != want {
		t.Fatalf("DumpDOM() after node construction helpers = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsSupportStandardNodeConstructionHelpers(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="root"></main><script>const root = document.querySelector("#root"); const span = document.createElement("span"); const text = document.createTextNode("seed"); span.setAttribute("id", "child"); span.appendChild(text); root.appendChild(span); root.removeChild(span); root.appendChild(span)</script>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main id="root"><span id="child">seed</span></main><script>const root = document.querySelector("#root"); const span = document.createElement("span"); const text = document.createTextNode("seed"); span.setAttribute("id", "child"); span.appendChild(text); root.appendChild(span); root.removeChild(span); root.appendChild(span)</script>`; got != want {
		t.Fatalf("DumpDOM() after standard node construction helpers = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanUpdateTextNodeNodeValue(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="root"><div id="out"></div><script>const out = document.querySelector("#out"); const text = document.createTextNode("seed"); out.appendChild(text); text.nodeValue = "updated"</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main id="root"><div id="out">updated</div><script>const out = document.querySelector("#out"); const text = document.createTextNode("seed"); out.appendChild(text); text.nodeValue = "updated"</script></main>`; got != want {
		t.Fatalf("DumpDOM() after text node nodeValue update = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanUpdateTextNodeData(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="root"><div id="out"></div><div id="mirror"></div><script>const out = document.querySelector("#out"); const mirror = document.querySelector("#mirror"); const text = document.createTextNode("seed"); mirror.appendChild(text); const before = text.data; text.data = "updated"; out.textContent = before + "|" + text.nodeValue</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main id="root"><div id="out">seed|updated</div><div id="mirror">updated</div><script>const out = document.querySelector("#out"); const mirror = document.querySelector("#mirror"); const text = document.createTextNode("seed"); mirror.appendChild(text); const before = text.data; text.data = "updated"; out.textContent = before + "|" + text.nodeValue</script></main>`; got != want {
		t.Fatalf("DumpDOM() after text node data update = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsRejectNodeValueAssignmentOnElements(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").nodeValue = "updated"</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want unsupported nodeValue assignment on element")
	}
	if !strings.Contains(err.Error(), "assignment to") || !strings.Contains(err.Error(), "nodeValue") {
		t.Fatalf("WriteHTML() error = %q, want unsupported nodeValue assignment error", err)
	}
}

func TestSessionInlineScriptsRejectDataAssignmentOnElements(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").data = "updated"</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want unsupported data assignment on element")
	}
	if !strings.Contains(err.Error(), "assignment to") || !strings.Contains(err.Error(), "data") {
		t.Fatalf("WriteHTML() error = %q, want unsupported data assignment error", err)
	}
}

func TestSessionInlineScriptsRejectWholeTextAssignmentOnElements(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").wholeText = "updated"</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want unsupported wholeText assignment on element")
	}
	if !strings.Contains(err.Error(), "assignment to") || !strings.Contains(err.Error(), "wholeText") {
		t.Fatalf("WriteHTML() error = %q, want unsupported wholeText assignment error", err)
	}
}

func TestSessionInlineScriptsRejectSplitTextOnElements(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").splitText(1)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want unsupported splitText on element")
	}
	if !strings.Contains(err.Error(), "splitText") {
		t.Fatalf("WriteHTML() error = %q, want unsupported splitText error", err)
	}
}

func TestSessionInlineScriptsRejectNormalizeWithArguments(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").normalize(1)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want normalize arity error")
	}
	if !strings.Contains(err.Error(), "normalize") || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("WriteHTML() error = %q, want normalize arity error", err)
	}
}

func TestSessionInlineScriptsRejectInvalidNodeConstructionTargets(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main></main><script>host:appendChild("#missing", host:createElement("span"))</script>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want missing-target error")
	}
	if !strings.Contains(err.Error(), "did not match any element") {
		t.Fatalf("WriteHTML() error = %q, want missing-target selector error", err)
	}
}

func TestSessionInlineScriptsCanCreateTextNodesReplaceChildrenAndInsertAdjacentNodes(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="wrap"><div id="target"><span id="keep">keep</span></div><p id="tail">tail</p></main><script>const target = host:querySelector("#target"); const keep = host:querySelector("#keep"); const seed = host:createTextNode("seed"); host:replaceChild(expr(target), expr(seed), expr(keep)); const em = host:createElement("em"); const strong = host:createElement("strong"); host:insertAdjacentElement(expr(target), "afterbegin", expr(em)); host:insertAdjacentText(expr(target), "beforeend", " tail"); host:insertAdjacentElement(expr(target), "beforebegin", expr(strong)); host:insertAdjacentText(expr(target), "afterend", "!")</script>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main id="wrap"><strong></strong><div id="target"><em></em>seed tail</div>!<p id="tail">tail</p></main><script>const target = host:querySelector("#target"); const keep = host:querySelector("#keep"); const seed = host:createTextNode("seed"); host:replaceChild(expr(target), expr(seed), expr(keep)); const em = host:createElement("em"); const strong = host:createElement("strong"); host:insertAdjacentElement(expr(target), "afterbegin", expr(em)); host:insertAdjacentText(expr(target), "beforeend", " tail"); host:insertAdjacentElement(expr(target), "beforebegin", expr(strong)); host:insertAdjacentText(expr(target), "afterend", "!")</script>`; got != want {
		t.Fatalf("DumpDOM() after node construction helpers = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanUseNodeBeforeAfterAndReplaceWith(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="root"><span id="a">A</span><span id="b">B</span><span id="c">C</span></main><div id="probe"></div><script>const root = document.querySelector("#root"); const a = document.querySelector("#a"); const b = document.querySelector("#b"); const c = document.querySelector("#c"); a.before("x", document.createElement("em")); b.after(document.createTextNode("y")); c.replaceWith("r", document.createElement("strong")); document.querySelector("#probe").textContent = root.innerHTML</script>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, err := s.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if want := `x<em></em><span id="a">A</span><span id="b">B</span>yr<strong></strong>`; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanUseNodeReplaceChildren(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="root"><div id="target"><span id="keep">keep</span><b id="drop"><i id="gone">gone</i></b></div><div id="source"><em id="moved">move</em></div><div id="probe"></div><script>const root = document.querySelector("#root"); const target = document.querySelector("#target"); const source = document.querySelector("#source"); const keep = document.querySelector("#keep"); target.replaceChildren("pre-", source, keep, document.createElement("em")); document.querySelector("#probe").textContent = String(root.children.length) + "|" + target.innerHTML</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, err := s.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if want := `3|pre-<div id="source"><em id="moved">move</em></div><span id="keep">keep</span><em></em>`; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanUseDocumentReplaceChildren(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="root"><script>document.replaceChildren(document.createElement("section"), document.createElement("aside"))</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := s.DumpDOM(), `<section></section><aside></aside>`; got != want {
		t.Fatalf("DumpDOM() after document.replaceChildren = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsRejectDocumentReplaceChildrenSelfInsertion(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><script>document.replaceChildren(document)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want self-insertion error")
	}
	if !strings.Contains(err.Error(), "itself") {
		t.Fatalf("WriteHTML() error = %q, want self-insertion error", err)
	}
}

func TestSessionInlineScriptsCanUseNodeRemove(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main id="root"><div id="outer"><span id="keep">keep</span></div><div id="probe"></div><script>const orphan = document.createElement("em"); orphan.remove(); const keep = document.querySelector("#keep"); const outer = document.querySelector("#outer"); keep.remove(); outer.remove(); document.querySelector("#probe").textContent = String(document.querySelector("#keep") === null) + ":" + String(document.querySelector("#outer") === null) + ":" + String(orphan.parentNode === null)</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main id="root"><div id="probe">true:true:true</div><script>const orphan = document.createElement("em"); orphan.remove(); const keep = document.querySelector("#keep"); const outer = document.querySelector("#outer"); keep.remove(); outer.remove(); document.querySelector("#probe").textContent = String(document.querySelector("#keep") === null) + ":" + String(document.querySelector("#outer") === null) + ":" + String(orphan.parentNode === null)</script></main>`; got != want {
		t.Fatalf("DumpDOM() after node.remove = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsRejectInvalidLowLevelNodeConstructionPositions(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="wrap"><div id="target"><span id="keep"></span></div></main><script>const target = host:querySelector("#target"); const em = host:createElement("em"); host:insertAdjacentElement(expr(target), "sideways", expr(em))</script>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want invalid-position error")
	}
	if !strings.Contains(err.Error(), "invalid insertAdjacentElement position") {
		t.Fatalf("WriteHTML() error = %q, want invalid-position selector error", err)
	}
}

func TestSessionInlineScriptsRejectNodeReplaceChildrenSelfInsertion(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="target"><span id="keep">keep</span></div><script>const target = document.querySelector("#target"); target.replaceChildren(target)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want self-insertion error")
	}
	if !strings.Contains(err.Error(), "itself") {
		t.Fatalf("WriteHTML() error = %q, want self-insertion error", err)
	}
}

func TestSessionInlineScriptsRejectNodeRemoveWithArguments(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").remove(1)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want node.remove arity error")
	}
	if !strings.Contains(err.Error(), "remove") || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("WriteHTML() error = %q, want node.remove arity error", err)
	}
}

func TestSessionInlineScriptsRejectDocumentRemove(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.remove()</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want unsupported document.remove")
	}
	if !strings.Contains(err.Error(), "document.remove") || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("WriteHTML() error = %q, want unsupported document.remove error", err)
	}
}

func TestSessionInlineScriptsRejectDocumentContainsWithWrongArity(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.contains()</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want document.contains arity error")
	}
	if !strings.Contains(err.Error(), "contains") || !strings.Contains(err.Error(), "1 argument") {
		t.Fatalf("WriteHTML() error = %q, want document.contains arity error", err)
	}
}

func TestSessionInlineScriptsRejectDocumentGetRootNodeWithWrongArity(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.getRootNode(1)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want document.getRootNode arity error")
	}
	if !strings.Contains(err.Error(), "getRootNode") || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("WriteHTML() error = %q, want document.getRootNode arity error", err)
	}
}

func TestSessionInlineScriptsRejectElementGetRootNodeWithWrongArity(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").getRootNode(1)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want element.getRootNode arity error")
	}
	if !strings.Contains(err.Error(), "getRootNode") || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("WriteHTML() error = %q, want element.getRootNode arity error", err)
	}
}

func TestSessionInlineScriptsRejectDocumentCompareDocumentPositionWithWrongArity(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.compareDocumentPosition()</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want compareDocumentPosition arity error")
	}
	if !strings.Contains(err.Error(), "compareDocumentPosition") || !strings.Contains(err.Error(), "1 argument") {
		t.Fatalf("WriteHTML() error = %q, want compareDocumentPosition arity error", err)
	}
}

func TestSessionInlineScriptsRejectElementCompareDocumentPositionWithWrongArity(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").compareDocumentPosition()</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want element.compareDocumentPosition arity error")
	}
	if !strings.Contains(err.Error(), "compareDocumentPosition") || !strings.Contains(err.Error(), "1 argument") {
		t.Fatalf("WriteHTML() error = %q, want element.compareDocumentPosition arity error", err)
	}
}

func TestSessionInlineScriptsRejectDocumentHasChildNodesWithWrongArity(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.hasChildNodes(1)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want document.hasChildNodes arity error")
	}
	if !strings.Contains(err.Error(), "hasChildNodes") || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("WriteHTML() error = %q, want document.hasChildNodes arity error", err)
	}
}

func TestSessionInlineScriptsRejectElementHasChildNodesWithWrongArity(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>document.querySelector("#out").hasChildNodes(1)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want element.hasChildNodes arity error")
	}
	if !strings.Contains(err.Error(), "hasChildNodes") || !strings.Contains(err.Error(), "no arguments") {
		t.Fatalf("WriteHTML() error = %q, want element.hasChildNodes arity error", err)
	}
}

func TestSessionInlineScriptsRejectNodeBeforeSelfInsertion(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	err := s.WriteHTML(`<main id="root"><div id="out">seed</div><script>const out = document.querySelector("#out"); out.before(out)</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want self-insertion error")
	}
	if !strings.Contains(err.Error(), "itself") {
		t.Fatalf("WriteHTML() error = %q, want self-insertion error", err)
	}
}
