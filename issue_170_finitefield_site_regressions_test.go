package browsertester

import "testing"

func TestIssue170DOMParserSupportsSvgMime(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const parser = new DOMParser();
		  if (!(parser instanceof DOMParser)) {
		    throw new Error("DOMParser instanceof failed");
		  }
		  const doc = parser.parseFromString(
		    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4" /></svg>',
		    "image/svg+xml"
		  );
		  const rootName = doc && doc.documentElement ? String(doc.documentElement.nodeName || "") : "missing";
		  const namespaceUri = doc && doc.documentElement ? String(doc.documentElement.namespaceURI || "") : "missing";
		  const contentType = doc ? String(doc.contentType || "") : "missing";
		  document.getElementById("out").textContent = rootName + "|" + namespaceUri + "|" + contentType;
		</script>
	`)

	if err := harness.AssertText("#out", "svg|http://www.w3.org/2000/svg|image/svg+xml"); err != nil {
		t.Fatalf("AssertText(#out, svg|http://www.w3.org/2000/svg|image/svg+xml) error = %v", err)
	}
}
