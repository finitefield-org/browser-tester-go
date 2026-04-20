package runtime

import "testing"

func TestSessionBootstrapsFileConstructorDataTransferAndInputFilesAssignment(t *testing.T) {
	const rawHTML = `<main><input id="upload" type="file"><div id="out"></div><script>const file = new File(["A,B"], "sample.csv", { type: "text/csv", lastModified: 123 }); const dt = new DataTransfer(); dt.items.add(file); const input = document.getElementById("upload"); input.files = dt.files; const selected = input.files[0]; const blobURL = URL.createObjectURL(selected); selected.text().then((text) => { document.getElementById("out").textContent = [selected instanceof File, selected instanceof Blob, selected.name, selected.size, selected.type, selected.lastModified, dt.files.length, dt.items.length, text, blobURL.slice(0, 5) === "blob:"].join("|"); URL.revokeObjectURL(blobURL); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|true|sample.csv|3|text/csv|123|1|1|A,B|true" {
		t.Fatalf("TextContent(#out) = %q, want file/DataTransfer round-trip", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after File/DataTransfer bootstrap", got)
	}
}

func TestSessionBootstrapsRegExpConstructorAndInstanceOf(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const re = new RegExp("([A-Z]+)[,_-]+([A-Z]+)[,_-]+([A-Z]+)"); const exec = re.exec("A_B-C"); document.getElementById("out").textContent = [re instanceof RegExp, re.test("A_B-C"), exec[0], exec[1], exec[2], exec[3], re.toString()].join("|");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `true|true|A_B-C|A|B|C|/([A-Z]+)[,_-]+([A-Z]+)[,_-]+([A-Z]+)/` {
		t.Fatalf("TextContent(#out) = %q, want RegExp constructor behavior", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after RegExp bootstrap", got)
	}
}

func TestSessionBootstrapsDragEventsExposeDataTransfer(t *testing.T) {
	const rawHTML = `<main><div id="box"></div><div id="out"></div><script>document.getElementById("box").addEventListener("dragover", (event) => { event.preventDefault(); event.dataTransfer.dropEffect = "move"; document.getElementById("out").textContent = event.dataTransfer.dropEffect; });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Dispatch("#box", "dragover"); err != nil {
		t.Fatalf("Dispatch(#box, dragover) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "move" {
		t.Fatalf("TextContent(#out) = %q, want dragover dataTransfer support", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after dragover bootstrap", got)
	}
}
