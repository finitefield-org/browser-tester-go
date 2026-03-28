package browsertester

import "testing"

func TestIssue155ClosestAcceptsSelectorVariableInIfCondition(t *testing.T) {
	harness, err := FromHTML(`
		<div class="btn-wrap">
		  <span id="child">child</span>
		</div>
		<p id="out"></p>
		<script>
		  const child = document.getElementById("child");
		  const buttonWrapSelector = ".btn-wrap, .button-block";
		  if (child.closest(buttonWrapSelector)) {
		    document.getElementById("out").textContent = "matched";
		  }
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "matched"); err != nil {
		t.Fatalf("AssertText(#out, matched) error = %v", err)
	}
}

func TestIssue158ClosestAcceptsSelectorVariableInExpressionPosition(t *testing.T) {
	harness, err := FromHTML(`
		<section class="card">
		  <button id="child">open</button>
		</section>
		<p id="out"></p>
		<script>
		  const child = document.getElementById("child");
		  const selector = ".card";
		  const matched = child.closest(selector);
		  document.getElementById("out").textContent = matched ? matched.tagName : "none";
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "SECTION"); err != nil {
		t.Fatalf("AssertText(#out, SECTION) error = %v", err)
	}
}
