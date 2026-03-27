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
