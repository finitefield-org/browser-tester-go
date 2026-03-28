package browsertester

import (
	"strings"
	"testing"
)

func issue165CSVLine(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]byte, 0, len(values)*8)
	for i, value := range values {
		if i > 0 {
			out = append(out, ',')
		}
		needsQuotes := false
		for _, ch := range value {
			if ch == ',' || ch == '"' || ch == '\n' {
				needsQuotes = true
				break
			}
		}
		if needsQuotes {
			out = append(out, '"')
			for i := 0; i < len(value); i++ {
				if value[i] == '"' {
					out = append(out, '"', '"')
					continue
				}
				out = append(out, value[i])
			}
			out = append(out, '"')
			continue
		}
		out = append(out, value...)
	}
	return string(out)
}

func issue165CSVDownload() string {
	lines := [][]string{
		{"field_name", "field_group", "crop_name", "start_ym", "end_ym", "caution_tag", "status", "memo"},
		{"Field 1", "North Block", "Cabbage", "2026-02", "2026-05", "Brassicaceae", "fixed", "Spring crop plan"},
		{"Field 2", "North Block", "Tomato", "2026-03", "2026-08", "Solanaceae", "plan", "Summer-autumn crop"},
	}
	out := make([]byte, 0, 256)
	for i, line := range lines {
		if i > 0 {
			out = append(out, '\n')
		}
		out = append(out, issue165CSVLine(line)...)
	}
	return string(out)
}

func TestIssue165QuotedNewlineSeparatorInJoinIsSupported(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  document.getElementById("out").textContent = ["a", "b"].join("\n");
		</script>
	`)

	if err := harness.AssertText("#out", "a\nb"); err != nil {
		t.Fatalf("AssertText(#out, a\\nb) error = %v", err)
	}
}

func TestIssue165BuildCsvKeepsRowBreaksBeforeDownload(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  function csvLine(values) {
		    return values.map((value) => {
		      const text = String(value === undefined || value === null ? "" : value);
		      if (/[",\n]/.test(text)) return "\"" + text.replace(/"/g, "\"\"") + "\"";
		      return text;
		    }).join(",");
		  }
		  function buildCsv() {
		    const lines = [
		      ["field_name", "field_group"],
		      ["Field 1", "North Block"],
		      ["Field 2", "South Block"]
		    ];
		    return lines.map(csvLine).join("\n");
		  }
		  document.getElementById("out").textContent = buildCsv();
		</script>
	`)

	if err := harness.AssertText("#out", "field_name,field_group\nField 1,North Block\nField 2,South Block"); err != nil {
		t.Fatalf("AssertText(#out, csv rows) error = %v", err)
	}
}

func TestIssue165ChainedMapJoinKeepsExplicitSeparator(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  document.getElementById("out").textContent = ["a", "b"].map((value) => value).join("\n");
		</script>
	`)

	if err := harness.AssertText("#out", "a\nb"); err != nil {
		t.Fatalf("AssertText(#out, a\\nb) error = %v", err)
	}
}

func TestIssue165JoinAfterStoringMapResultKeepsExplicitSeparator(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const mapped = ["a", "b"].map((value) => value);
		  document.getElementById("out").textContent = mapped.join("\n");
		</script>
	`)

	if err := harness.AssertText("#out", "a\nb"); err != nil {
		t.Fatalf("AssertText(#out, a\\nb) error = %v", err)
	}
}

func TestIssue165NamedMapCallbackFollowedByJoinKeepsExplicitSeparator(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  function identity(value) {
		    return value;
		  }
		  function build() {
		    return ["a", "b"].map(identity).join("\n");
		  }
		  document.getElementById("out").textContent = build();
		</script>
	`)

	if err := harness.AssertText("#out", "a\nb"); err != nil {
		t.Fatalf("AssertText(#out, a\\nb) error = %v", err)
	}
}

func TestIssue165NestedArrayRowsMapToStringsThenJoinWithNewlines(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  function rowToLine(row) {
		    return row.join(",");
		  }
		  function build() {
		    const rows = [
		      ["a", "b"],
		      ["c", "d"]
		    ];
		    return rows.map(rowToLine).join("\n");
		  }
		  document.getElementById("out").textContent = build();
		</script>
	`)

	if err := harness.AssertText("#out", "a,b\nc,d"); err != nil {
		t.Fatalf("AssertText(#out, a,b\\nc,d) error = %v", err)
	}
}

func TestIssue166MultilineCSVBlobDownloadKeepsRowBreaks(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<body>
		  <button id="download">Download</button>
		  <div id="out"></div>
		  <script>
		    function csvLine(values) {
		      return values.map((value) => {
		        const text = String(value === undefined || value === null ? "" : value);
		        if (/[",\n]/.test(text)) return "\"" + text.replace(/"/g, "\"\"") + "\"";
		        return text;
		      }).join(",");
		    }
		    function buildCsv() {
		      const lines = [
		        ["field_name", "field_group", "crop_name", "start_ym", "end_ym", "caution_tag", "status", "memo"],
		        ["Field 1", "North Block", "Cabbage", "2026-02", "2026-05", "Brassicaceae", "fixed", "Spring crop plan"],
		        ["Field 2", "North Block", "Tomato", "2026-03", "2026-08", "Solanaceae", "plan", "Summer-autumn crop"]
		      ];
		      return lines.map(csvLine).join("\n");
		    }
		    document.getElementById("download").addEventListener("click", () => {
		      const blob = new Blob([buildCsv()], { type: "text/csv" });
		      const url = URL.createObjectURL(blob);
		      const link = document.createElement("a");
		      link.href = url;
		      link.download = "sample.csv";
		      document.body.appendChild(link);
		      link.click();
		      document.body.removeChild(link);
		      URL.revokeObjectURL(url);
		      document.getElementById("out").textContent = url;
		    });
		  </script>
		</body>
	`)

	if err := harness.Click("#download"); err != nil {
		t.Fatalf("Click(#download) error = %v", err)
	}
	downloads := harness.Mocks().Downloads().Artifacts()
	if len(downloads) != 1 {
		t.Fatalf("Downloads().Artifacts() = %#v, want one captured download", downloads)
	}
	if downloads[0].FileName != "sample.csv" {
		t.Fatalf("Downloads()[0].FileName = %q, want sample.csv", downloads[0].FileName)
	}
	if got, want := string(downloads[0].Bytes), issue165CSVDownload(); got != want {
		t.Fatalf("Downloads()[0].Bytes = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if !strings.HasPrefix(got, "blob:") {
		t.Fatalf("TextContent(#out) = %q, want blob URL", got)
	}
}

func TestIssue166InvalidQueryFallbackKeepsDefaultAreaResult(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="result"></div>
      <div id="status"></div>
      <div id="from"></div>
      <div id="to"></div>
      <script>
        const UNIT_GROUPS = {
          area: ["ha", "acre"],
          crop: ["bushel_acre", "kg_ha"]
        };

        const DEFAULT_PAIRS = {
          area: { fromUnit: "ha", toUnit: "acre" },
          crop: { fromUnit: "bushel_acre", toUnit: "kg_ha" }
        };

        const DEFAULTS = {
          category: "area",
          inputValue: "1",
          fromUnit: "ha",
          toUnit: "acre",
          localeMode: "auto",
          roundMode: "sigfig",
          significantDigits: 4,
          fixedDecimals: "auto",
          gallonType: "us",
          cropPreset: "corn",
          testWeightLbPerBushel: "56",
          showJpCustomUnits: false
        };

        const state = {
          category: DEFAULTS.category,
          inputValue: DEFAULTS.inputValue,
          fromUnit: DEFAULTS.fromUnit,
          toUnit: DEFAULTS.toUnit,
          localeMode: DEFAULTS.localeMode,
          roundMode: DEFAULTS.roundMode,
          significantDigits: DEFAULTS.significantDigits,
          fixedDecimals: DEFAULTS.fixedDecimals,
          gallonType: DEFAULTS.gallonType,
          cropPreset: DEFAULTS.cropPreset,
          testWeightLbPerBushel: DEFAULTS.testWeightLbPerBushel,
          showJpCustomUnits: DEFAULTS.showJpCustomUnits
        };

        function getDefaultPair(category) {
          return DEFAULT_PAIRS[category] || DEFAULT_PAIRS.area;
        }

        function applyCategory(category) {
          const nextCategory = UNIT_GROUPS[category] ? category : DEFAULTS.category;
          state.category = nextCategory;
          const pair = getDefaultPair(nextCategory);
          state.fromUnit = pair.fromUnit;
          state.toUnit = pair.toUnit;
        }

        applyCategory("unknown");
        document.getElementById("result").textContent = state.category;
        document.getElementById("status").textContent = UNIT_GROUPS[state.category].join(",");
        document.getElementById("from").textContent = state.fromUnit;
        document.getElementById("to").textContent = state.toUnit;
      </script>
    `)

	if err := harness.AssertText("#result", "area"); err != nil {
		t.Fatalf("AssertText(#result, area) error = %v", err)
	}
	if err := harness.AssertText("#status", "ha,acre"); err != nil {
		t.Fatalf("AssertText(#status, ha,acre) error = %v", err)
	}
	if err := harness.AssertText("#from", "ha"); err != nil {
		t.Fatalf("AssertText(#from, ha) error = %v", err)
	}
	if err := harness.AssertText("#to", "acre"); err != nil {
		t.Fatalf("AssertText(#to, acre) error = %v", err)
	}
}
