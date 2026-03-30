package browsertester

import "testing"

func TestIssue221NestedConditionalRenderBootstrapsAndReRenders(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<input id="trees" />
		<div id="out"></div>
	<script>
		  const state = { kind: "tree", treeDensityPerHa: NaN };

		  function render() {
		    const hint = state.kind === "tree"
		      ? Number.isFinite(state.treeDensityPerHa)
		        ? String(state.treeDensityPerHa) + " trees/ha"
		        : "Primary work units"
		      : "Primary work units";
		    document.getElementById("out").textContent = hint;
		  }

		  document.getElementById("trees").addEventListener("input", () => {
		    state.treeDensityPerHa = 400;
		    render();
		  });

		  render();
		</script>
	`)

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "Primary work units" {
		t.Fatalf("TextContent(#out) = %q, want Primary work units", got)
	}

	if err := harness.TypeText("#trees", "4"); err != nil {
		t.Fatalf("TypeText(#trees) error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after TypeText error = %v", err)
	} else if got != "400 trees/ha" {
		t.Fatalf("TextContent(#out) after TypeText = %q, want 400 trees/ha", got)
	}
}
