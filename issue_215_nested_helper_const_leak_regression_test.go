package browsertester

import "testing"

func TestIssue215NestedHelperLocalIndexDoesNotPoisonLaterConstIndex(t *testing.T) {
	harness, err := FromHTML(`
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    const state = {
		      stack: {
		        steps: [
		          { id: "step-1", config: { type: "percent", percent: { rateRaw: "10" } } },
		          { id: "step-2", config: { type: "fixed", fixed: { amountRaw: "10" } } }
		        ]
		      }
		    };

		    function setDeepValue(obj, path, value) {
		      if (!obj || !path) return;
		      const parts = path.split(".");
		      let current = obj;
		      for (let index = 0; index < parts.length - 1; index += 1) {
		        const part = parts[index];
		        if (!current[part] || typeof current[part] !== "object") {
		          current[part] = {};
		        }
		        current = current[part];
		      }
		      current[parts[parts.length - 1]] = value;
		    }

		    function describeStep(step) {
		      if (step.config.type === "percent") {
		        return step.id + ":" + step.config.percent.rateRaw;
		      }
		      return step.id + ":" + step.config.fixed.amountRaw;
		    }

		    function render() {
		      document.getElementById("out").textContent = state.stack.steps
		        .map((step) => describeStep(step))
		        .join("|");
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      const edited = state.stack.steps.find((item) => item.id === "step-1");
		      if (!edited) return;
		      setDeepValue(edited.config, "percent.rateRaw", "20");

		      const index = state.stack.steps.findIndex((item) => item.id === "step-2");
		      const moved = state.stack.steps.splice(index, 1)[0];
		      state.stack.steps.splice(index - 1, 0, moved);
		      render();
		    });

		    render();
		  })();
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "step-2:10|step-1:20" {
		t.Fatalf("TextContent(#out) = %q, want step-2:10|step-1:20", got)
	}
}

func TestIssue215NestedHelperLocalIndexDoesNotPoisonPlainConstDeclaration(t *testing.T) {
	harness, err := FromHTML(`
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    const state = { nested: {} };

		    function setDeepValue(obj, path, value) {
		      const parts = path.split(".");
		      let current = obj;
		      for (let index = 0; index < parts.length - 1; index += 1) {
		        const part = parts[index];
		        if (!current[part] || typeof current[part] !== "object") {
		          current[part] = {};
		        }
		        current = current[part];
		      }
		      current[parts[parts.length - 1]] = value;
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      setDeepValue(state.nested, "percent.rateRaw", "20");
		      const index = 1;
		      document.getElementById("out").textContent =
		        "" + index + ":" + state.nested.percent.rateRaw;
		    });
		  })();
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1:20" {
		t.Fatalf("TextContent(#out) = %q, want 1:20", got)
	}
}
