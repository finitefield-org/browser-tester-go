package browsertester

import "testing"

func TestIssue173SwedishCollationOrdersARingBeforeAUmlaut(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<pre id="out"></pre>
		<script>
		  const collator = new Intl.Collator("sv", {
		    usage: "sort",
		    sensitivity: "variant",
		  });
		  const values = ["Öga", "Zebra", "Äpple", "Ål"];
		  values.sort(collator.compare);
		  document.getElementById("out").textContent =
		    values.join(",") + "|" + String(collator.compare("Ål", "Äpple") < 0);
		</script>
	`)

	if err := harness.AssertText("#out", "Zebra,Ål,Äpple,Öga|true"); err != nil {
		t.Fatalf("AssertText(#out, Zebra,Ål,Äpple,Öga|true) error = %v", err)
	}
}
