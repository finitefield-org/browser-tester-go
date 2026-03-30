package browsertester

import "testing"

func TestIssue223StringNormalizeBootstrapsWithNFKC(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  document.getElementById("out").textContent = String("\uFB01").normalize("NFKC");
		</script>
	`)

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "fi" {
		t.Fatalf("TextContent(#out) = %q, want fi", got)
	}
}
