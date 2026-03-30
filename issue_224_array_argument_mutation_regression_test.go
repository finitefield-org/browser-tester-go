package browsertester

import "testing"

func TestIssue224ArrayArgumentMutationsPropagateThroughNestedHelpers(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<input id="field" />
		<div id="out"></div>
		<script>
		  (() => {
		    function addError(errors) {
		      errors.push("too wide");
		    }

		    function render() {
		      const errors = [];
		      addError(errors);
		      document.getElementById("out").textContent = errors.length + "|" + errors[0];
		    }

		    document.getElementById("field").addEventListener("input", render);
		    render();
		  })();
		</script>
	`)

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|too wide" {
		t.Fatalf("TextContent(#out) = %q, want 1|too wide", got)
	}

	if err := harness.TypeText("#field", "4"); err != nil {
		t.Fatalf("TypeText(#field) error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after TypeText error = %v", err)
	} else if got != "1|too wide" {
		t.Fatalf("TextContent(#out) after TypeText = %q, want 1|too wide", got)
	}
}
