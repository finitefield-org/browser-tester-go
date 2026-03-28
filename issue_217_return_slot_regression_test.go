package browsertester

import (
	"strings"
	"testing"
)

func TestIssue217NestedHelperCallInReturnExpressionKeepsOuterReturnValue(t *testing.T) {
	harness, err := FromHTML(`
		<div id="out"></div>
		<script>
		  (() => {
		    function renderLabel(label) {
		      return ` + "`" + `<div class="field">${escapeHtml(label)}</div>` + "`" + `;
		    }

		    function escapeHtml(value) {
		      return String(value || "")
		        .replace(/&/g, "&amp;")
		        .replace(/</g, "&lt;")
		        .replace(/>/g, "&gt;")
		        .replace(/"/g, "&quot;")
		        .replace(/'/g, "&#39;");
		    }

		    document.getElementById("out").textContent = renderLabel("Holding rate");
		  })();
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `<div class="field">Holding rate</div>` {
		t.Fatalf("TextContent(#out) = %q, want %q", got, `<div class="field">Holding rate</div>`)
	}
}

func TestIssue217BatchMappingGridKeepsSelectMarkupWithLateHelperDeclaration(t *testing.T) {
	harness, err := FromHTML(`
		<div id="grid"></div>
		<script>
		  (() => {
		    function renderBatchMappingGrid() {
		      const labels = [
		        "#1 annual_demand",
		        "#2 order_cost",
		        "#3 alt_rate",
		        "#4 unit_cost",
		      ];
		      const mapping = {
		        annualDemand: 0,
		        orderCost: 1,
		        holdingRate: -1,
		        unitCost: 3,
		      };
		      const mappingFields = [
		        ["annualDemand", "Annual demand"],
		        ["orderCost", "Order cost"],
		        ["holdingRate", "Holding rate"],
		        ["unitCost", "Unit cost"],
		      ];

		      document.getElementById("grid").innerHTML = mappingFields.map(([key, label]) => {
		        const options = [` + "`" + `<option value="-1">${escapeHtml("Unused")}</option>` + "`" + `]
		          .concat(labels.map((header, index) => ` + "`" + `<option value="${index}" ${mapping[key] === index ? "selected" : ""}>${escapeHtml(header)}</option>` + "`" + `))
		          .join("");
		        return ` + "`" + `<div class="field">
		          <label class="field-label" for="eoq-calculator-map-${key}">${escapeHtml(label)}</label>
		          <select id="eoq-calculator-map-${key}" data-map-key="${key}">${options}</select>
		        </div>` + "`" + `;
		      }).join("");
		    }

		    function escapeHtml(value) {
		      return String(value || "")
		        .replace(/&/g, "&amp;")
		        .replace(/</g, "&lt;")
		        .replace(/>/g, "&gt;")
		        .replace(/"/g, "&quot;")
		        .replace(/'/g, "&#39;");
		    }

		    renderBatchMappingGrid();
		  })();
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("#eoq-calculator-map-holdingRate"); err != nil {
		t.Fatalf("AssertExists(#eoq-calculator-map-holdingRate) error = %v", err)
	}
	if err := harness.AssertValue("#eoq-calculator-map-orderCost", "1"); err != nil {
		t.Fatalf("AssertValue(#eoq-calculator-map-orderCost) error = %v", err)
	}

	snippet := harness.Debug().DumpDOM()
	if !strings.Contains(snippet, "<select") ||
		!strings.Contains(snippet, `id="eoq-calculator-map-holdingRate"`) ||
		!strings.Contains(snippet, `<option value="2">#3 alt_rate</option>`) {
		t.Fatalf("Debug().DumpDOM() = %q, want rendered select markup", snippet)
	}
}
