package runtime

import "testing"

func TestSessionInlineScriptsCanUseNodeListLengthAndItem(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><ul><li id="a"></li><li id="b"></li></ul><div id="probe"></div><script>const nodes = document.querySelectorAll("li"); host:setTextContent("#probe", expr(nodes.length + ":" + nodes.item(0).id + ":" + nodes.item(1).id))</script></main>`,
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if store == nil {
		t.Fatalf("ensureDOM() store = nil, want DOM store")
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if want := "2:a:b"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanUseNodeListForEach(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><ul><li id="a"></li><li id="b"></li></ul><div id="probe"></div><script>const nodes = document.querySelectorAll("li"); let out = ""; nodes.forEach((node, index, list) => { out += (out === "" ? "" : "|") + index + ":" + node.id + ":" + list.length; }); host:setTextContent("#probe", expr(out))</script></main>`,
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if store == nil {
		t.Fatalf("ensureDOM() store = nil, want DOM store")
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if want := "0:a:2|1:b:2"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanUseNodeListIterators(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><ul><li id="a"></li><li id="b"></li></ul><div id="probe"></div><script>const nodes = document.querySelectorAll("li"); let keysOut = ""; for (let key of nodes.keys()) { keysOut += (keysOut === "" ? "" : "|") + key; }; let valuesOut = ""; for (let node of nodes.values()) { valuesOut += (valuesOut === "" ? "" : "|") + node.id; }; let entriesOut = ""; for (let entry of nodes.entries()) { entriesOut += (entriesOut === "" ? "" : "|") + entry[0] + ":" + entry[1].id; }; host:setTextContent("#probe", expr(keysOut + ";" + valuesOut + ";" + entriesOut))</script></main>`,
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if store == nil {
		t.Fatalf("ensureDOM() store = nil, want DOM store")
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if want := "0|1;a|b;0:a|1:b"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsRejectNodeListForEachWithoutCallbackExplicitly(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><ul><li id="a"></li></ul><div id="probe"></div><script>const nodes = document.querySelectorAll("li"); let caught = false; try { nodes.forEach() } catch (error) { caught = true; }; host:setTextContent("#probe", expr(caught ? "caught" : "missed"))</script></main>`,
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if store == nil {
		t.Fatalf("ensureDOM() store = nil, want DOM store")
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if want := "caught"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}
