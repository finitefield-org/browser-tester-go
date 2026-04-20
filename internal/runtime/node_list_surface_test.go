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

func TestSessionInlineScriptsCanUseNodeListForEachWithElementTypeReflection(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><section><input data-bind="sampleMass" type="text" value="10"><input data-bind="showChart" type="checkbox" checked><select data-bind="massBasis"><option value="as_received">As received</option><option value="dry">Dry</option></select><textarea data-bind="sampleNote">seed</textarea></section><div id="probe"></div><script>const controls = document.querySelectorAll("[data-bind]"); let out = ""; controls.forEach((control) => { out += (out === "" ? "" : "|") + control.type; }); host:setTextContent("#probe", expr(out))</script></main>`,
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
	} else if want := "text|checkbox|select-one|textarea"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after NodeList.forEach element.type reflection bridge", got)
	}
}

func TestSessionInlineScriptsCanUseElementQuerySelectorAllWithElementTypeReflection(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="sampleMass" type="text" value="10"><input data-bind="showChart" type="checkbox" checked><select data-bind="massBasis"><option value="as_received">As received</option><option value="dry">Dry</option></select><textarea data-bind="sampleNote">seed</textarea></section><div id="probe"></div><script>const root = document.getElementById("root"); const controls = root.querySelectorAll("[data-bind]"); let out = ""; controls.forEach((control) => { out += (out === "" ? "" : "|") + control.type; }); host:setTextContent("#probe", expr(out))</script></main>`,
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
	} else if want := "text|checkbox|select-one|textarea"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after element.querySelectorAll element.type reflection bridge", got)
	}
}

func TestSessionInlineScriptsCanSyncElementQuerySelectorAllControls(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="sampleName" type="text" value="seed"><input data-bind="sampleDate" type="date" value="2026-04-02"><input data-bind="displayPrecision" type="number" value="1"><input data-bind="showChart" type="checkbox"><select data-bind="massBasis"><option value="as_received">As received</option><option value="dry">Dry</option></select><textarea data-bind="sampleNote">seed</textarea></section><div id="probe"></div><script>const state = { sampleName: "Lot 1", sampleDate: "2026-04-03", displayPrecision: "2", showChart: true, massBasis: "dry", sampleNote: "hello" }; const root = document.getElementById("root"); root.querySelectorAll("[data-bind]").forEach((el) => { const path = el.dataset.bind; const value = state[path]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); host:setTextContent("#probe", expr([root.querySelector("[data-bind='sampleName']").value, root.querySelector("[data-bind='sampleDate']").value, root.querySelector("[data-bind='displayPrecision']").value, String(root.querySelector("[data-bind='showChart']").checked), root.querySelector("[data-bind='massBasis']").value, root.querySelector("[data-bind='sampleNote']").value].join("|")))</script></main>`,
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
	} else if want := "Lot 1|2026-04-03|2|true|dry|hello"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after element.querySelectorAll control sync bridge", got)
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
