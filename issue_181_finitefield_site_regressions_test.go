package browsertester

import "testing"

func TestIssue181XMLSerializerIsAvailableForElementNodes(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const node = document.createElement("div");
		  node.setAttribute("data-test", "ok");
		  const serializer = new XMLSerializer();
		  if (!(serializer instanceof XMLSerializer)) {
		    throw new Error("XMLSerializer instanceof failed");
		  }
		  document.getElementById("out").textContent = serializer.serializeToString(node);
		</script>
	`)

	if err := harness.AssertText("#out", `<div data-test="ok"></div>`); err != nil {
		t.Fatalf("AssertText(#out, <div data-test=\"ok\"></div>) error = %v", err)
	}
}

func TestIssue181XMLSerializerSerializesSvgAfterDomParserRoundtrip(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const parsed = new DOMParser().parseFromString(
		    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><script>alert(1)<\/script><circle cx="5" cy="5" r="4" /></svg>',
		    "image/svg+xml"
		  );
		  const safeRoot = parsed.documentElement.cloneNode(true);
		  for (const node of Array.from(safeRoot.querySelectorAll("script"))) {
		    if (node.parentNode) {
		      node.parentNode.removeChild(node);
		    }
		  }
		  const serialized = new XMLSerializer().serializeToString(safeRoot);
		  document.getElementById("out").textContent = [
		    String(serialized.startsWith("<svg")),
		    String(serialized.includes('xmlns="http://www.w3.org/2000/svg"')),
		    String(serialized.includes("<circle")),
		    String(serialized.includes("<script")),
		  ].join("|");
		</script>
	`)

	if err := harness.AssertText("#out", "true|true|true|false"); err != nil {
		t.Fatalf("AssertText(#out, true|true|true|false) error = %v", err)
	}
}
