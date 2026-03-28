package browsertester

import "testing"

func TestIssue171IntlCollatorNumericOptionOrdersDigitRunsNaturally(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<pre id="out"></pre>
		<script>
		  const values = ["item 10", "item 2", "item 1"];
		  const collator = new Intl.Collator("en", {
		    usage: "sort",
		    numeric: true,
		    sensitivity: "variant",
		  });

		  const asc = values.slice().sort(collator.compare).join(",");
		  const desc = values.slice().sort((left, right) => collator.compare(right, left)).join(",");
		  const zeroPadded = collator.compare("item 02", "item 2");
		  const numeric = String(collator.resolvedOptions().numeric);

		  document.getElementById("out").textContent =
		    asc + "|" + desc + "|" + zeroPadded + "|" + numeric;
		</script>
	`)

	if err := harness.AssertText("#out", "item 1,item 2,item 10|item 10,item 2,item 1|0|true"); err != nil {
		t.Fatalf("AssertText(#out, item 1,item 2,item 10|item 10,item 2,item 1|0|true) error = %v", err)
	}
}
