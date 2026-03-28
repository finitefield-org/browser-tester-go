package browsertester

import "testing"

func TestIssue153DynamicIndexCompoundAssignmentIsSupported(t *testing.T) {
	harness, err := FromHTML(`
		<div id="out"></div>
		<script>
		  const values = [1, 2];
		  const index = 1;
		  values[index] += 3;
		  document.getElementById("out").textContent = String(values[1]);
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "5"); err != nil {
		t.Fatalf("AssertText(#out, 5) error = %v", err)
	}
}

func TestIssue154FunctionListenerBindsThisToCurrentTarget(t *testing.T) {
	harness, err := FromHTML(`
		<button id="button" data-value="ok">go</button>
		<div id="out"></div>
		<script>
		  const button = document.getElementById("button");
		  const out = document.getElementById("out");
		  button.addEventListener("click", function () {
		    out.textContent = this.getAttribute("data-value");
		  });
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#button"); err != nil {
		t.Fatalf("Click(#button) error = %v", err)
	}
	if err := harness.AssertText("#out", "ok"); err != nil {
		t.Fatalf("AssertText(#out, ok) error = %v", err)
	}
}

func TestIssue156RequestAnimationFrameIgnoresExtraArguments(t *testing.T) {
	harness, err := FromHTML(`
		<div id="out"></div>
		<script>
		  const out = document.getElementById("out");
		  window.requestAnimationFrame(function () {
		    out.textContent = "done";
		  }, 0);
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", ""); err != nil {
		t.Fatalf("AssertText(#out, empty) error = %v", err)
	}
	if err := harness.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}
	if err := harness.AssertText("#out", "done"); err != nil {
		t.Fatalf("AssertText(#out, done) error = %v", err)
	}
}

func TestIssue157DateToLocaleDateStringIsAvailable(t *testing.T) {
	harness, err := FromHTML(`
		<div id="out"></div>
		<script>
		  const date = new Date(1706918400000);
		  document.getElementById("out").textContent = date.toLocaleDateString("en-US");
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "2/3/2024"); err != nil {
		t.Fatalf("AssertText(#out, 2/3/2024) error = %v", err)
	}
}

func TestIssue159AssignmentThroughCallResultIsSupported(t *testing.T) {
	harness, err := FromHTML(`
		<div id="out"></div>
		<script>
		  const warnings = new Map([["a", { overlap: false }]]);
		  warnings.get("a").overlap = true;
		  document.getElementById("out").textContent = String(warnings.get("a").overlap);
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "true"); err != nil {
		t.Fatalf("AssertText(#out, true) error = %v", err)
	}
}

func TestIssue160ArrayFlatMapIsSupported(t *testing.T) {
	harness, err := FromHTML(`
		<div id="out"></div>
		<script>
		  const values = ["north", "south"];
		  const result = values.flatMap((value) => [value]);
		  document.getElementById("out").textContent = result.join(",");
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "north,south"); err != nil {
		t.Fatalf("AssertText(#out, north,south) error = %v", err)
	}
}
