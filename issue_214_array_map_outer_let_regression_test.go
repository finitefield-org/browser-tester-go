package browsertester

import "testing"

func TestIssue214ArrayMapCallbackMutationsUpdateOuterLetBindings(t *testing.T) {
	harness, err := FromHTML(`
		<div id="calculated">0</div>
		<div id="errors">0</div>
		<div id="preview"></div>
		<script>
		  const rows = [
		    { ok: true, label: "valid" },
		    { ok: false, label: "invalid" }
		  ];
		  let calculatedCount = 0;
		  let errorCount = 0;
		  const previewRows = rows.map((row) => {
		    const notes = [];
		    if (!row.ok) {
		      notes.push("bad");
		      errorCount += 1;
		      return { label: row.label, notes };
		    }
		    calculatedCount += 1;
		    return { label: row.label, notes };
		  });
		  document.getElementById("calculated").textContent = String(calculatedCount);
		  document.getElementById("errors").textContent = String(errorCount);
		  document.getElementById("preview").textContent = previewRows
		    .map((row) => row.label + ":" + row.notes.join(";"))
		    .join("|");
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#calculated"); err != nil {
		t.Fatalf("TextContent(#calculated) error = %v", err)
	} else if got != "1" {
		t.Fatalf("TextContent(#calculated) = %q, want 1", got)
	}
	if got, err := harness.TextContent("#errors"); err != nil {
		t.Fatalf("TextContent(#errors) error = %v", err)
	} else if got != "1" {
		t.Fatalf("TextContent(#errors) = %q, want 1", got)
	}
	if got, err := harness.TextContent("#preview"); err != nil {
		t.Fatalf("TextContent(#preview) error = %v", err)
	} else if got != "valid:|invalid:bad" {
		t.Fatalf("TextContent(#preview) = %q, want valid:|invalid:bad", got)
	}
}
