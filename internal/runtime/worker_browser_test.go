package runtime

import "testing"

func TestSessionBootstrapsWorkerMessageRoundTrip(t *testing.T) {
	const rawHTML = `<main><div id="out">pending</div><script>const workerSource = [
  "function normalize(value) { return String(value).toUpperCase(); }",
  "function buildMessage(event) { return { echoed: event.data, normalized: normalize(event.data) }; }",
  "self.onmessage = function(event) { self.postMessage(buildMessage(event)); };"
].join("\n"); const blob = new Blob([workerSource], { type: "text/javascript" }); const worker = new Worker(URL.createObjectURL(blob)); if (!(worker instanceof Worker)) { throw new Error("Worker instanceof failed"); } worker.onmessage = function(event) { document.getElementById("out").textContent = [event.data.echoed, event.data.normalized, String(worker instanceof Worker)].join("|"); }; worker.postMessage("hello");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "hello|HELLO|true" {
		t.Fatalf("TextContent(#out) = %q, want hello|HELLO|true", got)
	}
}
