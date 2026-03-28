package browsertester

import "testing"

func TestIssue167ReassignedIntlNumberFormatIsUsedByPageCode(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<pre id="out"></pre>
		<script>
		  Intl = {
		    NumberFormat: function () {
		      throw new Error("forced Intl failure");
		    }
		  };
		  window.Intl = Intl;
		  Intl.NumberFormat = function () {
		    throw new Error("forced Intl failure");
		  };

		  function formatIndex(value, lang, minimumIntegerDigits) {
		    const safeValue = Math.max(0, Number(value) || 0);
		    try {
		      return new Intl.NumberFormat(lang, {
		        useGrouping: false,
		        minimumIntegerDigits,
		        maximumFractionDigits: 0
		      }).format(safeValue);
		    } catch (error) {
		      const digits = String(Math.trunc(safeValue));
		      return digits.padStart(minimumIntegerDigits, "0");
		    }
		  }

		  const lines = ["A", "B"].map((label, index) => {
		    return "[" + formatIndex(index + 1, "ar-EG", 1) + "] " + label;
		  });
		  document.getElementById("out").textContent = lines.join("\n");
		</script>
	`)

	if err := harness.AssertText("#out", "[1] A\n[2] B"); err != nil {
		t.Fatalf("AssertText(#out, [1] A\\n[2] B) error = %v", err)
	}
}
