package browsertester

import (
	"runtime/debug"
	"testing"
)

func TestIssue219MarginMarkupPageLoadsAndRunsBulkFlowOnSmallTestThread(t *testing.T) {
	prevMaxStack := debug.SetMaxStack(2 << 20)
	t.Cleanup(func() {
		debug.SetMaxStack(prevMaxStack)
	})

	harness := mustHarnessFromHTML(t, `
		<div id="margin-markup-calculator-root" data-lang="ja">
		  <button type="button" id="margin-markup-calculator-open-button">ツールを開く</button>
		  <div id="margin-markup-calculator-fullscreen-dialog" class="dialog hidden">
		    <details id="margin-markup-calculator-settings" class="disclosure">
		      <summary id="margin-markup-calculator-settings-summary">詳細オプション</summary>
		      <div>Settings</div>
		    </details>
		    <div class="mode-tabs">
		      <button type="button" data-mode-tab="cost_to_price">原価→売価</button>
		      <button type="button" data-mode-tab="bulk">一括計算</button>
		    </div>
		    <section id="single-mode-pane">
		      <input id="field-cost" type="text" inputmode="decimal" />
		      <input id="field-extra-cost" type="text" inputmode="decimal" />
		      <input id="field-target-margin" type="text" inputmode="decimal" />
		    </section>
		    <section id="bulk-mode-pane" class="hidden">
		      <textarea id="bulk-paste"></textarea>
		      <input id="bulk-default-target" type="text" />
		      <button type="button" id="bulk-insert-sample">サンプルを挿入</button>
		      <div id="bulk-summary"></div>
		    </section>
		  </div>
		</div>
		<script>
		  (() => {
		    const state = {
		      mode: "cost_to_price",
		      cost: "",
		      extraCost: "",
		      targetMargin: "",
		    };

		    const root = document.getElementById("margin-markup-calculator-root");
		    const dialog = document.getElementById("margin-markup-calculator-fullscreen-dialog");
		    const openButton = document.getElementById("margin-markup-calculator-open-button");
		    const modeTabs = Array.from(document.querySelectorAll("[data-mode-tab]"));
		    const singlePane = document.getElementById("single-mode-pane");
		    const bulkPane = document.getElementById("bulk-mode-pane");
		    const cost = document.getElementById("field-cost");
		    const extraCost = document.getElementById("field-extra-cost");
		    const targetMargin = document.getElementById("field-target-margin");
		    const bulkInsertSample = document.getElementById("bulk-insert-sample");
		    const bulkSummary = document.getElementById("bulk-summary");

		    function setMode(mode) {
		      state.mode = mode;
		      const bulkActive = mode === "bulk";
		      bulkPane.classList.toggle("hidden", !bulkActive);
		      singlePane.classList.toggle("hidden", bulkActive);
		      modeTabs.forEach((button) => {
		        button.classList.toggle("active", button.dataset.modeTab === mode);
		      });
		      root.dataset.mode = mode;
		    }

		    openButton.addEventListener("click", () => {
		      dialog.classList.remove("hidden");
		    });

		    cost.addEventListener("input", () => {
		      state.cost = cost.value;
		    });
		    extraCost.addEventListener("input", () => {
		      state.extraCost = extraCost.value;
		    });
		    targetMargin.addEventListener("input", () => {
		      state.targetMargin = targetMargin.value;
		    });

		    modeTabs.forEach((button) => {
		      button.addEventListener("click", () => setMode(button.dataset.modeTab));
		    });

		    bulkInsertSample.addEventListener("click", () => {
		      const rows = [
		        { name: "定番A", cost: state.cost || "1200", extra: state.extraCost || "80", target: state.targetMargin || "40" },
		        { name: "セットB", cost: "2400", extra: "230", target: "45" },
		        { name: "SKU-C", cost: "600", extra: "50", target: "42" },
		      ];
		      const calculated = rows.filter((row) => row.cost !== "" && row.target !== "").length;
		      bulkSummary.textContent = "全 " + String(rows.length) + " 行中 " + String(calculated) + " 行を計算しました。";
		    });

		    setMode(state.mode);
		  })();
		</script>
	`)

	if err := harness.Click("#margin-markup-calculator-open-button"); err != nil {
		t.Fatalf("Click(#margin-markup-calculator-open-button) error = %v", err)
	}
	if err := harness.Click("#margin-markup-calculator-settings summary"); err != nil {
		t.Fatalf("Click(#margin-markup-calculator-settings summary) error = %v", err)
	}
	if ok, err := harness.HasAttribute("#margin-markup-calculator-settings", "open"); err != nil {
		t.Fatalf("HasAttribute(#margin-markup-calculator-settings, open) error = %v", err)
	} else if !ok {
		t.Fatalf("HasAttribute(#margin-markup-calculator-settings, open) = false, want true")
	}
	if err := harness.TypeText("#field-cost", "1200"); err != nil {
		t.Fatalf("TypeText(#field-cost) error = %v", err)
	}
	if err := harness.TypeText("#field-extra-cost", "230"); err != nil {
		t.Fatalf("TypeText(#field-extra-cost) error = %v", err)
	}
	if err := harness.TypeText("#field-target-margin", "40"); err != nil {
		t.Fatalf("TypeText(#field-target-margin) error = %v", err)
	}
	if err := harness.Click("[data-mode-tab='bulk']"); err != nil {
		t.Fatalf("Click([data-mode-tab='bulk']) error = %v", err)
	}
	if err := harness.Click("#bulk-insert-sample"); err != nil {
		t.Fatalf("Click(#bulk-insert-sample) error = %v", err)
	}
	if err := harness.AssertText("#bulk-summary", "全 3 行中 3 行を計算しました。"); err != nil {
		t.Fatalf("AssertText(#bulk-summary, 全 3 行中 3 行を計算しました。) error = %v", err)
	}
}
