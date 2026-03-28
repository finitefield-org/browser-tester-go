package browsertester

import "testing"

func TestIssue218BulkMappingAndSummaryCallbacksKeepOuterBindingsIsolatedAndAccumulating(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="mapping"></div>
      <div id="summary"></div>
      <div id="preview"></div>
      <script>
        (() => {
          function inferMappings(headers) {
            const roles = ["name", "cost", "price", "extra", "target"];
            const mappings = new Array(headers.length).fill("unused");
            const normalizedHeaders = headers.map((value) => String(value || "").toLowerCase());

            roles.forEach((role) => {
              const index = normalizedHeaders.indexOf(role);
              if (index >= 0) {
                mappings[index] = role;
              }
            });

            return mappings;
          }

          function computeBulkResult(rows) {
            let calculated = 0;
            const resultRows = rows.map((row, rowIndex) => {
              const status = row.price > 0 ? "calculated" : "missing";
              if (status === "calculated") {
                calculated += 1;
              }
              return {
                label: row.name || "row-" + String(rowIndex + 1),
                status,
              };
            });

            return {
              summary: "total=" + String(resultRows.length) + ";calculated=" + String(calculated),
              preview: resultRows.map((row) => row.label + ":" + row.status).join("|"),
            };
          }

          const mapping = inferMappings(["name", "cost", "price", "extra", "target"]);
          const result = computeBulkResult([
            { name: "SKU-A", cost: 1200, price: 2200 },
            { name: "SKU-B", cost: 900, price: 1680 },
            { name: "SKU-C", cost: 600, price: 990 }
          ]);

          document.getElementById("mapping").textContent = mapping.join("|");
          document.getElementById("summary").textContent = result.summary;
          document.getElementById("preview").textContent = result.preview;
        })();
      </script>
    `)

	if err := harness.AssertText("#mapping", "name|cost|price|extra|target"); err != nil {
		t.Fatalf("AssertText(#mapping, name|cost|price|extra|target) error = %v", err)
	}
	if err := harness.AssertText("#summary", "total=3;calculated=3"); err != nil {
		t.Fatalf("AssertText(#summary, total=3;calculated=3) error = %v", err)
	}
	if err := harness.AssertText("#preview", "SKU-A:calculated|SKU-B:calculated|SKU-C:calculated"); err != nil {
		t.Fatalf("AssertText(#preview, SKU-A:calculated|SKU-B:calculated|SKU-C:calculated) error = %v", err)
	}
}

func TestIssue218BulkResultMapAccumulatesMixedPriceAndTargetRows(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="summary"></div>
      <div id="preview"></div>
      <script>
        (() => {
          const page = {
            results: {
              noValue: "-",
            },
            bulk: {
              statusCalculated: "calculated",
              statusNotCalculated: "not-calculated",
              summary: "total={total};calculated={calculated}",
            },
          };

          function formatMessage(template, values) {
            return String(template || "").replace(/\{(\w+)\}/g, (_, key) => {
              return key in values ? values[key] : "";
            });
          }

          function valueOrZero(value) {
            const numeric = Number(value);
            return Number.isFinite(numeric) ? numeric : 0;
          }

          function parseLooseNumber(value) {
            if (value == null || value === "") return null;
            const numeric = Number(value);
            return Number.isFinite(numeric) ? numeric : null;
          }

          function calculateMetrics(price, baseCost, feeRate) {
            const grossProfit = price * (1 - feeRate) - baseCost;
            const margin = price > 0 ? grossProfit / price : Number.NaN;
            const markup = baseCost > 0 ? grossProfit / baseCost : Number.NaN;
            return { grossProfit, margin, markup };
          }

          function formatMoney(value) {
            if (value == null || !Number.isFinite(value)) return "-";
            return value.toFixed(0);
          }

          function formatPlain(value, digits) {
            const numeric = Number(value);
            if (!Number.isFinite(numeric)) return "";
            const factor = digits === 0 ? 1 : 10;
            const rounded = Math.round(numeric * factor) / factor;
            const text = String(rounded);
            if (digits > 0 && text.indexOf(".") < 0) {
              return text + ".0";
            }
            return text;
          }

          function formatPercent(value) {
            if (value == null || !Number.isFinite(value)) return "-";
            return String((value * 100).toFixed(1)) + "%";
          }

          function computeBulkResult(rows) {
            const feeRate = valueOrZero("3.6") / 100;
            const feeFixed = valueOrZero("10");
            const defaultTarget = parseLooseNumber("40");
            let calculated = 0;
            const resultRows = rows.map((row, rowIndex) => {
              const name = row.name || "row-" + String(rowIndex + 1);
              const cost = parseLooseNumber(row.cost);
              const price = parseLooseNumber(row.price);
              const extra = valueOrZero(row.extra);
              const parsedTarget = parseLooseNumber(row.target);
              const target = parsedTarget == null ? defaultTarget : parsedTarget;

              const baseCost = cost + extra + feeFixed;
              let recommendedPrice = Number.NaN;
              let grossProfit = Number.NaN;
              let margin = Number.NaN;
              let markup = Number.NaN;
              let status = page.bulk.statusNotCalculated;

              if (price != null && Number.isFinite(price) && price > 0) {
                const metrics = calculateMetrics(price, baseCost, feeRate);
                grossProfit = metrics.grossProfit;
                margin = metrics.margin;
                markup = metrics.markup;
                status = page.bulk.statusCalculated;
                calculated += 1;
              }

              if (target != null && Number.isFinite(target) && target >= 0 && target < 100 && feeRate + target / 100 < 1) {
                recommendedPrice = baseCost / (1 - feeRate - target / 100);
                if (status !== page.bulk.statusCalculated) {
                  status = page.bulk.statusCalculated;
                  calculated += 1;
                }
              }

              let priceText = page.results.noValue;
              if (price != null && Number.isFinite(price) && price > 0) {
                priceText = formatMoney(price);
              }

              let targetText = page.results.noValue;
              if (target != null) {
                targetText = String(formatPlain(target, 1)) + "%";
              }

              let recommendedPriceText = page.results.noValue;
              if (Number.isFinite(recommendedPrice)) {
                recommendedPriceText = formatMoney(recommendedPrice);
              }

              let grossProfitText = page.results.noValue;
              if (Number.isFinite(grossProfit)) {
                grossProfitText = formatMoney(grossProfit);
              }

              let marginText = page.results.noValue;
              if (Number.isFinite(margin)) {
                marginText = formatPercent(margin);
              }

              let markupText = page.results.noValue;
              if (Number.isFinite(markup)) {
                markupText = formatPercent(markup);
              }

              return {
                name,
                cost: formatMoney(cost),
                price: priceText,
                extra: formatMoney(extra),
                target: targetText,
                recommendedPrice: recommendedPriceText,
                grossProfit: grossProfitText,
                margin: marginText,
                markup: markupText,
                status,
              };
            });

            return {
              summary: formatMessage(page.bulk.summary, {
                total: String(resultRows.length),
                calculated: String(calculated),
              }),
              preview: resultRows.map((row) => row.name + ":" + row.status + ":" + row.recommendedPrice).join("|"),
            };
          }

          const result = computeBulkResult([
            { name: "定番A", cost: "1200", price: "1980", extra: "80", target: "40" },
            { name: "セットB", cost: "2400", price: "", extra: "230", target: "45" },
            { name: "SKU-C", cost: "600", price: "990", extra: "50", target: "" }
          ]);

          document.getElementById("summary").textContent = result.summary;
          document.getElementById("preview").textContent = result.preview;
        })();
      </script>
    `)

	if err := harness.AssertText("#summary", "total=3;calculated=3"); err != nil {
		t.Fatalf("AssertText(#summary, total=3;calculated=3) error = %v", err)
	}
	if err := harness.AssertText("#preview", "定番A:calculated:2287|セットB:calculated:5136|SKU-C:calculated:1170"); err != nil {
		t.Fatalf("AssertText(#preview, bulk rows) error = %v", err)
	}
}

func TestIssue218NestedBulkParserHelpersKeepOuterRowsAndCellsLive(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function parseDelimitedTable(text) {
            const rows = [];
            let row = [];
            let cell = "";

            const pushCell = () => {
              row.push(cell);
              cell = "";
            };

            const pushRow = () => {
              if (row.length || cell) {
                pushCell();
              }
              const normalized = row.map((value) => value.trim());
              const hasAny = normalized.some((value) => value !== "");
              if (hasAny) rows.push(normalized);
              row = [];
            };

            String(text || "").split(/\r?\n/).forEach((line) => {
              if (line === "") return;
              line.split("\t").forEach((part, partIndex) => {
                if (partIndex > 0) {
                  pushCell();
                }
                cell += part;
              });
              pushRow();
            });
            return rows;
          }

          const rows = parseDelimitedTable("商品名\t原価\t売価\n定番A\t1200\t1980\nSKU-C\t600\t990");
          document.getElementById("out").textContent = rows.map((row) => row.join("|")).join(" / ");
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "商品名|原価|売価 / 定番A|1200|1980 / SKU-C|600|990"); err != nil {
		t.Fatalf("AssertText(#out, parsed rows) error = %v", err)
	}
}

func TestIssue218NestedHelperUpdatesStayVisibleAfterDirectHelperCall(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function buildRows() {
            const rows = [];
            let row = [];
            let cell = "A";

            const pushCell = () => {
              row.push(cell);
              cell = "";
            };

            const pushRow = () => {
              pushCell();
              rows.push(row.slice());
              row = [];
            };

            pushRow();
            return String(rows.length) + ":" + (rows[0] ? rows[0].join("|") : "-") + ":" + cell + ":" + String(row.length);
          }

          document.getElementById("out").textContent = buildRows();
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "1:A::0"); err != nil {
		t.Fatalf("AssertText(#out, 1:A::0) error = %v", err)
	}
}

func TestIssue218ArrayCallbackCounterSurvivesPlainHelperCalls(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function formatPlain(value) {
            return String(value);
          }

          function compute() {
            let calculated = 0;
            const labels = [1, 2, 3].map(() => {
              calculated += 1;
              return formatPlain(calculated);
            });
            return String(calculated) + ":" + labels.join("|");
          }

          document.getElementById("out").textContent = compute();
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "3:1|2|3"); err != nil {
		t.Fatalf("AssertText(#out, 3:1|2|3) error = %v", err)
	}
}

func TestIssue218ArrayCallbackUpdatesRemainVisibleToLaterPlainHelperCalls(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function formatPlain(value) {
            return String(value);
          }

          function compute() {
            let calculated = 0;
            const rows = [1, 2, 3].map(() => {
              calculated += 1;
              return "ok";
            });
            const summary = formatPlain(calculated);
            return String(calculated) + ":" + summary + ":" + String(rows.length);
          }

          document.getElementById("out").textContent = compute();
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "3:3:3"); err != nil {
		t.Fatalf("AssertText(#out, 3:3:3) error = %v", err)
	}
}

func TestIssue218HelperUpdatesAreVisibleToLaterArrayMapInSameFunction(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function buildRow() {
            let row = [];
            let cell = " A ";

            const pushCell = () => {
              row.push(cell);
              cell = "";
            };

            if (row.length === 0) {
              pushCell();
            }

            const normalized = row.map((value) => value.trim());
            return normalized.join("|") + ":" + cell;
          }

          document.getElementById("out").textContent = buildRow();
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "A:"); err != nil {
		t.Fatalf("AssertText(#out, A:) error = %v", err)
	}
}

func TestIssue218RepeatedPushCellUpdatesFeedParserStylePushRow(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function buildRows() {
            const rows = [];
            let row = [];
            let cell = "";

            const pushCell = () => {
              row.push(cell);
              cell = "";
            };

            const pushRow = () => {
              if (row.length || cell) {
                pushCell();
              }
              const normalized = row.map((value) => value.trim());
              const hasAny = normalized.some((value) => value !== "");
              if (hasAny) rows.push(normalized);
              row = [];
            };

            cell = " 商品名 ";
            pushCell();
            cell = " 原価 ";
            pushCell();
            cell = " 売価 ";
            pushRow();

            return rows.map((entry) => entry.join("|")).join(" / ");
          }

          document.getElementById("out").textContent = buildRows();
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "商品名|原価|売価"); err != nil {
		t.Fatalf("AssertText(#out, 商品名|原価|売価) error = %v", err)
	}
}

func TestIssue218SimpleLoopParserKeepsPushCellAndPushRowUpdates(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function parseSimple(text) {
            const rows = [];
            let row = [];
            let cell = "";

            const pushCell = () => {
              row.push(cell);
              cell = "";
            };

            const pushRow = () => {
              if (row.length || cell) {
                pushCell();
              }
              const normalized = row.map((value) => value.trim());
              const hasAny = normalized.some((value) => value !== "");
              if (hasAny) rows.push(normalized);
              row = [];
            };

            String(text || "").split("\n").forEach((line) => {
              if (line === "") return;
              line.split("\t").forEach((part, partIndex) => {
                if (partIndex > 0) {
                  pushCell();
                }
                cell += part;
              });
              pushRow();
            });
            return rows;
          }

          const rows = parseSimple("商品名\t原価\t売価\n定番A\t1200\t1980");
          document.getElementById("out").textContent = rows.map((entry) => entry.join("|")).join(" / ");
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "商品名|原価|売価 / 定番A|1200|1980"); err != nil {
		t.Fatalf("AssertText(#out, parsed simple rows) error = %v", err)
	}
}

func TestIssue218SimpleLoopPushCellUpdatesSurviveAcrossContinueIterations(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
      <div id="out"></div>
      <script>
        (() => {
          function parseCells(text) {
            let row = [];
            let cell = "";

            const pushCell = () => {
              row.push(cell);
              cell = "";
            };

            String(text || "").split("\t").forEach((part, partIndex) => {
              if (partIndex > 0) {
                pushCell();
              }
              cell += part;
            });
            pushCell();
            return row.join("|");
          }

          document.getElementById("out").textContent = parseCells("商品名\t原価\t売価");
        })();
      </script>
    `)

	if err := harness.AssertText("#out", "商品名|原価|売価"); err != nil {
		t.Fatalf("AssertText(#out, 商品名|原価|売価) error = %v", err)
	}
}
