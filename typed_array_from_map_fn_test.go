package browsertester

import "testing"

func TestTypedArrayFromSupportsMapFunctionArgument(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const binary = "AZ";
		  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
		  document.getElementById("out").textContent = Array.from(bytes).join(",");
		</script>
	`)

	if err := harness.AssertText("#out", "65,90"); err != nil {
		t.Fatalf("AssertText(#out, 65,90) error = %v", err)
	}
}
