package browsertester

import "testing"

func TestIssue193PostfixIncrementInsideExpressionIsSupported(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  let rowSeq = 1;
		  function createDefaultRow(partial = {}) {
		    return {
		      id: partial.id || "r" + rowSeq++,
		    };
		  }
		  document.getElementById("out").textContent = createDefaultRow({}).id;
		</script>
	`)

	if err := harness.AssertText("#out", "r1"); err != nil {
		t.Fatalf("AssertText(#out, r1) error = %v", err)
	}
}

func TestIssue194ArrayDestructureAssignmentInsideElseIfBranchIsSupported(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const state = {
		    rows: [{ id: "a" }, { id: "b" }, { id: "c" }],
		  };

		  function reorder(action, index) {
		    if (action === "duplicate") {
		      state.rows.splice(index + 1, 0, state.rows[index]);
		    } else if (action === "delete") {
		      state.rows.splice(index, 1);
		    } else if (action === "up" && index > 0) {
		      [state.rows[index - 1], state.rows[index]] = [state.rows[index], state.rows[index - 1]];
		    } else if (action === "down" && index < state.rows.length - 1) {
		      [state.rows[index + 1], state.rows[index]] = [state.rows[index], state.rows[index + 1]];
		    }
		  }

		  reorder("up", 2);
		  document.getElementById("out").textContent = state.rows.map((row) => row.id).join(",");
		</script>
	`)

	if err := harness.AssertText("#out", "a,c,b"); err != nil {
		t.Fatalf("AssertText(#out, a,c,b) error = %v", err)
	}
}
