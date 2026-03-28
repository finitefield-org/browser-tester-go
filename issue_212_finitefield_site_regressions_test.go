package browsertester

import "testing"

func TestIssue212TypeofWindowHistoryReplaceStateMemberReference(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  document.getElementById("out").textContent =
		    typeof window.history.replaceState === "function" ? "ok" : "blocked";
		</script>
	`)

	if err := harness.AssertText("#out", "ok"); err != nil {
		t.Fatalf("AssertText(#out, ok) error = %v", err)
	}
}

func TestIssue212TypeofWindowLocationReplaceMemberReference(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  document.getElementById("out").textContent =
		    typeof window.location.replace === "function" ? "ok" : "blocked";
		</script>
	`)

	if err := harness.AssertText("#out", "ok"); err != nil {
		t.Fatalf("AssertText(#out, ok) error = %v", err)
	}
}
