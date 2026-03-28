package browsertester

import "testing"

func mustHarnessFromHTML(t *testing.T, html string) *Harness {
	t.Helper()
	harness, err := FromHTML(html)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}
	return harness
}

func TestClickTogglesButtonInsideOpenDialog(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="open">Open</button>
		<div id="dialog" class="hidden" role="dialog" aria-modal="true">
		  <button id="settings-toggle" type="button" aria-expanded="false">Settings</button>
		  <div id="settings-panel" class="hidden">Panel</div>
		</div>
		<p id="status"></p>
		<p id="trace"></p>
		<script>
		  (() => {
		    const el = {
		      open: document.getElementById("open"),
		      dialog: document.getElementById("dialog"),
		      settingsToggle: document.getElementById("settings-toggle"),
		      settingsPanel: document.getElementById("settings-panel"),
		      status: document.getElementById("status"),
		      trace: document.getElementById("trace"),
		    };

		    let settingsOpen = false;

		    function setHiddenClass(node, hidden) {
		      node.classList.toggle("hidden", hidden);
		    }

		    function syncStatus() {
		      el.status.textContent = [
		        String(settingsOpen),
		        el.settingsToggle.getAttribute("aria-expanded"),
		        String(el.settingsPanel.classList.contains("hidden")),
		      ].join("|");
		    }

		    function render() {
		      el.trace.textContent += "render>";
		      setHiddenClass(el.settingsPanel, !settingsOpen);
		      el.settingsToggle.setAttribute("aria-expanded", settingsOpen ? "true" : "false");
		      syncStatus();
		    }

		    el.open.addEventListener("click", () => {
		      el.trace.textContent += "open>";
		      el.dialog.classList.remove("hidden");
		      syncStatus();
		    });

		    el.settingsToggle.addEventListener("click", () => {
		      el.trace.textContent += "toggle>";
		      settingsOpen = !settingsOpen;
		      render();
		    });

		    syncStatus();
		  })();
		</script>
	`)

	if err := harness.Click("#open"); err != nil {
		t.Fatalf("Click(#open) error = %v", err)
	}
	if err := harness.Click("#settings-toggle"); err != nil {
		t.Fatalf("Click(#settings-toggle) error = %v", err)
	}
	if err := harness.AssertText("#trace", "open>toggle>render>"); err != nil {
		t.Fatalf("AssertText(#trace, open>toggle>render>) error = %v", err)
	}
	if err := harness.AssertText("#status", "true|true|false"); err != nil {
		t.Fatalf("AssertText(#status, true|true|false) error = %v", err)
	}
}

func TestWindowOpenReturnsPopupStubForPrintFlows(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go">go</button>
		<div id="out"></div>
		<script>
		  document.getElementById("go").addEventListener("click", () => {
		    const win = window.open("", "_blank", "noopener,noreferrer");
		    if (!win) {
		      document.getElementById("out").textContent = "null";
		      return;
		    }
		    win.document.open();
		    win.document.write("<p>print view</p>");
		    win.document.close();
		    win.focus();
		    win.print();
		    document.getElementById("out").textContent = [
		      String(win.closed),
		      String(win.opener === null),
		      String(win.document.readyState),
		    ].join("|");
		  });
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "false|true|complete"); err != nil {
		t.Fatalf("AssertText(#out, false|true|complete) error = %v", err)
	}
	if got := harness.Debug().PrintCalls(); len(got) != 1 {
		t.Fatalf("Debug().PrintCalls() = %#v, want one print call", got)
	}
}

func TestClickPreservesPreRequestAnimationFrameProcessingState(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="run" type="button">Run</button>
		<div id="processing" class="hidden">Processing</div>
		<p id="status"></p>
		<script>
		  (() => {
		    const el = {
		      run: document.getElementById("run"),
		      processing: document.getElementById("processing"),
		      status: document.getElementById("status"),
		    };

		    function setProcessing(processing) {
		      el.processing.classList.toggle("hidden", !processing);
		      el.run.disabled = processing;
		      el.status.textContent = [
		        String(el.run.disabled),
		        String(el.processing.classList.contains("hidden")),
		      ].join("|");
		    }

		    function nextFrame() {
		      return new Promise((resolve) => {
		        window.requestAnimationFrame(() => resolve());
		      });
		    }

		    async function runTask() {
		      setProcessing(true);
		      await nextFrame();
		      setProcessing(false);
		    }

		    el.run.addEventListener("click", runTask);
		    setProcessing(false);
		  })();
		</script>
	`)

	if err := harness.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}
	if err := harness.AssertText("#status", "true|false"); err != nil {
		t.Fatalf("AssertText(#status, true|false) error = %v", err)
	}
	if err := harness.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}
	if err := harness.AssertText("#status", "false|true"); err != nil {
		t.Fatalf("AssertText(#status, false|true) after AdvanceTime(0) error = %v", err)
	}
}

func TestDispatchKeyboardCompletesAsyncKeydownHandlersWaitingForAnimationFrame(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<textarea id="input"></textarea>
		<textarea id="result"></textarea>
		<script>
		  (() => {
		    const input = document.getElementById("input");
		    const result = document.getElementById("result");

		    window.addEventListener("keydown", (event) => {
		      if (event.key !== "Escape") {
		        return;
		      }
		      event.preventDefault();
		      window.requestAnimationFrame(() => {
		        const seen = new Set();
		        const lines = [];
		        for (const rawLine of input.value.split(/\r?\n/)) {
		          if (rawLine !== "" && !seen.has(rawLine)) {
		            seen.add(rawLine);
		            lines.push(rawLine);
		          }
		        }
		        result.value = lines.join("\n");
		      });
		    });
		  })();
		</script>
	`)

	if err := harness.TypeText("#input", "A\nA\nB"); err != nil {
		t.Fatalf("TypeText(#input) error = %v", err)
	}
	if err := harness.DispatchKeyboard("#input"); err != nil {
		t.Fatalf("DispatchKeyboard(#input) error = %v", err)
	}
	if err := harness.AssertValue("#result", ""); err != nil {
		t.Fatalf("AssertValue(#result, empty) error = %v", err)
	}
	if err := harness.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}
	if err := harness.AssertValue("#result", "A\nB"); err != nil {
		t.Fatalf("AssertValue(#result, A\\nB) error = %v", err)
	}
}

func TestIIFEHelperListenerReadsLiveOuterLetAfterSiblingRender(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="open">Open</button>
		<button id="copy">Copy</button>
		<p id="result"></p>
		<script>
		  (() => {
		    let lastComputation = null;

		    function bindActions() {
		      document.getElementById("open").addEventListener("click", () => {
		        document.body.setAttribute("data-opened", "yes");
		      });
		      document.getElementById("copy").addEventListener("click", () => {
		        document.getElementById("result").textContent =
		          lastComputation ? lastComputation.value : "null";
		      });
		    }

		    function render() {
		      lastComputation = { value: "1.23 mg/L", canCopy: true };
		    }

		    bindActions();
		    render();
		  })();
		</script>
	`)

	if err := harness.Click("#open"); err != nil {
		t.Fatalf("Click(#open) error = %v", err)
	}
	if err := harness.Click("#copy"); err != nil {
		t.Fatalf("Click(#copy) error = %v", err)
	}
	if err := harness.AssertText("#result", "1.23 mg/L"); err != nil {
		t.Fatalf("AssertText(#result, 1.23 mg/L) error = %v", err)
	}
}

func TestIIFEFunctionKeepsLaterSiblingFunctionDeclarationCallable(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<p id="result"></p>
		<script>
		  (() => {
		    const state = createDefaultState();

		    function createDefaultState() {
		      return {
		        fieldRules: defaultFieldRules(),
		      };
		    }

		    function defaultFieldRules() {
		      return ["ok"];
		    }

		    document.getElementById("result").textContent = state.fieldRules[0];
		  })();
		</script>
	`)

	if err := harness.AssertText("#result", "ok"); err != nil {
		t.Fatalf("AssertText(#result, ok) error = %v", err)
	}
}

func TestListenerReferenceKeepsPendingFunctionDeclOuterCapture(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    let lastResult = null;

		    function attachEvents() {
		      document.getElementById("go").addEventListener("click", openPrintView);
		    }

		    function openPrintView() {
		      document.getElementById("out").textContent = lastResult ? lastResult.value : "null";
		    }

		    function recompute() {
		      lastResult = { value: "ready" };
		    }

		    attachEvents();
		    recompute();
		  })();
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "ready"); err != nil {
		t.Fatalf("AssertText(#out, ready) error = %v", err)
	}
}

func TestSiblingClosureCallsDoNotPruneScopeCaptureEnv(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="open">open</button>
		<button id="print">print</button>
		<div id="out"></div>
		<script>
		  (() => {
		    const el = {
		      out: document.getElementById("out"),
		      open: document.getElementById("open"),
		      print: document.getElementById("print"),
		    };
		    let state = { mode: "ready" };
		    let lastResult = { value: "ok" };

		    function openDialog() {
		      el.out.dataset.dialog = "open";
		    }

		    function openPrintView() {
		      el.out.textContent = lastResult.value + ":" + state.mode;
		    }

		    el.open.addEventListener("click", openDialog);
		    el.print.addEventListener("click", openPrintView);
		  })();
		</script>
	`)

	if err := harness.Click("#open"); err != nil {
		t.Fatalf("Click(#open) error = %v", err)
	}
	if err := harness.Click("#print"); err != nil {
		t.Fatalf("Click(#print) error = %v", err)
	}
	if err := harness.AssertText("#out", "ok:ready"); err != nil {
		t.Fatalf("AssertText(#out, ok:ready) error = %v", err)
	}
}

func TestNestedCallPreservesCallerLocalCloseBinding(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    function make(source) {
		      let index = 0;

		      function current() {
		        return source[index] || "";
		      }

		      function consume() {
		        const char = source[index] || "";
		        index += 1;
		        return char;
		      }

		      function parseSequence(stopChar) {
		        let seen = "";
		        while (index < source.length && current() !== stopChar) {
		          seen += consume();
		        }
		        return "seen=" + seen + "|stop=" + stopChar + "|curr=" + (current() || "<eof>") + "|index=" + index;
		      }

		      function parseBracketGroup() {
		        const open = consume();
		        const close = open === "(" ? ")" : "]";
		        const inner = parseSequence(close);
		        return "after=" + (current() || "<eof>") + "|close=" + close + "|index=" + index + "|" + inner;
		      }

		      return parseBracketGroup();
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      document.getElementById("out").textContent = make("(SO4)3");
		    });
		  })();
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "after=)|close=)|index=4|seen=SO4|stop=)|curr=)|index=4"); err != nil {
		t.Fatalf("AssertText(#out, nested call binding) error = %v", err)
	}
}

func TestNestedCallKeepsCallerLocalBindingBeforeFollowUpCalls(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    function make(source) {
		      let index = 0;

		      function consume() {
		        const char = source[index] || "";
		        index += 1;
		        return char;
		      }

		      function parseSequence(stopChar) {
		        let seen = "";
		        while (index < source.length && source[index] !== stopChar) {
		          seen += consume();
		        }
		        return seen;
		      }

		      function parseBracketGroup() {
		        const open = consume();
		        const close = open === "(" ? ")" : "]";
		        const inner = parseSequence(close);
		        return "close=" + close + "|inner=" + inner + "|index=" + index;
		      }

		      return parseBracketGroup();
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      document.getElementById("out").textContent = make("(SO4)3");
		    });
		  })();
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "close=)|inner=SO4|index=4"); err != nil {
		t.Fatalf("AssertText(#out, close=)|inner=SO4|index=4) error = %v", err)
	}
}

func TestNestedCallKeepsCallerLocalBindingAfterSiblingCall(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    function make(source) {
		      let index = 0;

		      function current() {
		        return source[index] || "";
		      }

		      function consume() {
		        const char = source[index] || "";
		        index += 1;
		        return char;
		      }

		      function parseSequence(stopChar) {
		        let seen = "";
		        while (index < source.length && current() !== stopChar) {
		          seen += consume();
		        }
		        return seen;
		      }

		      function parseBracketGroup() {
		        const open = consume();
		        const close = open === "(" ? ")" : "]";
		        const inner = parseSequence(close);
		        const after = current();
		        return "close=" + close + "|after=" + after + "|inner=" + inner + "|index=" + index;
		      }

		      return parseBracketGroup();
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      document.getElementById("out").textContent = make("(SO4)3");
		    });
		  })();
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "close=)|after=)|inner=SO4|index=4"); err != nil {
		t.Fatalf("AssertText(#out, close=)|after=)|inner=SO4|index=4) error = %v", err)
	}
}

func TestTrivialNestedCallDoesNotReplaceLocalCloseWithWindowClose(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    function make() {
		      function noop() {
		        return "ok";
		      }

		      function parseBracketGroup() {
		        const close = ")";
		        const inner = noop();
		        return "close=" + close + "|inner=" + inner;
		      }

		      return parseBracketGroup();
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      document.getElementById("out").textContent = make();
		    });
		  })();
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "close=)|inner=ok"); err != nil {
		t.Fatalf("AssertText(#out, close=)|inner=ok) error = %v", err)
	}
}

func TestNestedCallKeepsCapturedIndexVisibleToBareReads(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    function make(source) {
		      let index = 0;

		      function current() {
		        return source[index] || "";
		      }

		      function consume() {
		        const char = source[index] || "";
		        index += 1;
		        return char;
		      }

		      function parseDigits() {
		        const start = index;
		        while (/[0-9]/.test(current())) {
		          consume();
		        }
		        return "start=" + start + "|index=" + index + "|curr=" + current() + "|raw=" + source.slice(start, index);
		      }

		      consume();
		      consume();
		      return parseDigits();
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      document.getElementById("out").textContent = make("Al2(SO4)3");
		    });
		  })();
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "start=2|index=3|curr=(|raw=2"); err != nil {
		t.Fatalf("AssertText(#out, start=2|index=3|curr=(|raw=2) error = %v", err)
	}
}

func TestNestedParseNumberKeepsOuterProgressVisible(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  (() => {
		    function createParser(source) {
		      let index = 0;

		      function current() {
		        return source[index] || "";
		      }

		      function consume() {
		        const char = source[index] || "";
		        index += 1;
		        return char;
		      }

		      function isDigit(char) {
		        return /[0-9]/.test(char);
		      }

		      function isUpper(char) {
		        return /[A-Z]/.test(char);
		      }

		      function isLower(char) {
		        return /[a-z]/.test(char);
		      }

		      function parseNumber() {
		        const start = index;
		        while (isDigit(current())) {
		          consume();
		        }
		        return source.slice(start, index);
		      }

		      function parseOptionalMultiplier() {
		        if (isDigit(current())) return parseNumber();
		        return "";
		      }

		      function parseElementSymbol() {
		        const first = current();
		        if (!isUpper(first)) {
		          throw new Error("invalid symbol");
		        }
		        let symbol = consume();
		        if (isLower(current())) symbol += consume();
		        return symbol;
		      }

		      function parseElementGroup() {
		        const symbol = parseElementSymbol();
		        const count = parseOptionalMultiplier();
		        return symbol + count + "|index=" + index + "|curr=" + current();
		      }

		      return parseElementGroup();
		    }

		    document.getElementById("go").addEventListener("click", () => {
		      try {
		        document.getElementById("out").textContent = createParser("Al2(SO4)3");
		      } catch (error) {
		        document.getElementById("out").textContent =
		          error && error.message ? error.message : "unknown";
		      }
		    });
		  })();
		</script>
	`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", "Al2|index=3|curr=("); err != nil {
		t.Fatalf("AssertText(#out, Al2|index=3|curr=() error = %v", err)
	}
}

func TestPlainFormulaParserAcceptsParenthesizedGroups(t *testing.T) {
	harness := mustHarnessFromHTML(t,
		`<div><input id="formula" value="Al2(SO4)3" /><button id="go" type="button">go</button><div id="out"></div></div>`+
			`<script>`+
			`(() => {`+
			`const weights = { Al: true, S: true, O: true };`+
			`const input = document.getElementById("formula");`+
			`const out = document.getElementById("out");`+
			`function parserError(message) { return { message }; }`+
			`function multiplyCounts(map, factor) { const out = {}; Object.keys(map).forEach((key) => { out[key] = map[key] * factor; }); return out; }`+
			`function createParser(source) {`+
			`  let index = 0;`+
			`  function current() { return source[index] || ""; }`+
			`  function consume() { const char = source[index] || ""; index += 1; return char; }`+
			`  function isDigit(char) { return /[0-9]/.test(char); }`+
			`  function isUpper(char) { return /[A-Z]/.test(char); }`+
			`  function isLower(char) { return /[a-z]/.test(char); }`+
			`  function parseNumber() { const start = index; let sawDigit = false; while (isDigit(current())) { sawDigit = true; consume(); } const raw = source.slice(start, index); if (!sawDigit) { throw parserError("invalid number"); } return { raw, value: Number(raw) }; }`+
			`  function parseOptionalMultiplier() { if (isDigit(current())) return parseNumber(); return { raw: "", value: 1 }; }`+
			`  function parseElementSymbol() { const first = current(); if (!isUpper(first)) { throw parserError("invalid symbol"); } let symbol = consume(); if (isLower(current())) symbol += consume(); if (!weights[symbol]) { throw parserError("unknown element"); } return symbol; }`+
			`  function parseBracketGroup() { const open = consume(); const close = open === "(" ? ")" : "]"; const inner = parseSequence(close, 1); if (current() !== close) { throw parserError("Bracket mismatch detected."); } consume(); const multiplier = parseOptionalMultiplier(); return { counts: multiplyCounts(inner.counts, multiplier.value), order: inner.order.slice(), normalized: open + inner.normalized + close + multiplier.raw }; }`+
			`  function parseElementGroup() { const symbol = parseElementSymbol(); const count = parseOptionalMultiplier(); return { counts: { [symbol]: count.value }, order: [symbol], normalized: symbol + count.raw }; }`+
			`  function parseGroup(nesting) { const char = current(); if (char === "(" || char === "[") { return parseBracketGroup(); } return parseElementGroup(); }`+
			`  function parseSequence(stopChar, nesting) { const counts = {}; const order = []; let normalized = ""; while (index < source.length && current() !== stopChar) { if (current() === ")" || current() === "]") { throw parserError("unexpected close"); } const group = parseGroup(nesting); Object.keys(group.counts).forEach((key) => { counts[key] = (counts[key] || 0) + group.counts[key]; }); group.order.forEach((item) => { if (!order.includes(item)) order.push(item); }); normalized += group.normalized; } return { counts, order, normalized }; }`+
			`  function parseFragment() { const body = parseSequence("", 1); if (index !== source.length) { throw parserError("invalid tail"); } return { counts: body.counts, order: body.order.slice(), normalized: body.normalized }; }`+
			`  return { parseFragment };`+
			`}`+
			`document.getElementById("go").addEventListener("click", () => { try { const parsed = createParser(input.value).parseFragment(); out.textContent = parsed.normalized + "|" + JSON.stringify(parsed.counts); } catch (error) { out.textContent = error && error.message ? error.message : "unknown"; } });`+
			`})();`+
			`</script>`)

	if err := harness.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := harness.AssertText("#out", `Al2(SO4)3|{"Al":2,"S":3,"O":12}`); err != nil {
		t.Fatalf("AssertText(#out, parsed parenthesized formula output) error = %v", err)
	}
}

func TestForeachAttachedClickHandlerReassignsOuterState(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="reset-a" type="button">Reset A</button>
		<button id="reset-b" type="button">Reset B</button>
		<input id="status" value="" />
		<script>
		  let state = { value: "1.2" };
		  function createDefaultState() {
		    return { value: "" };
		  }
		  function renderControls() {
		    document.getElementById("status").value = state.value;
		  }
		  const els = {
		    resetButtons: [
		      document.getElementById("reset-a"),
		      document.getElementById("reset-b")
		    ]
		  };
		  renderControls();
		  els.resetButtons.forEach((button) => button?.addEventListener("click", () => {
		    state = createDefaultState();
		    renderControls();
		  }));
		</script>
	`)

	if err := harness.Click("#reset-a"); err != nil {
		t.Fatalf("Click(#reset-a) error = %v", err)
	}
	if err := harness.AssertValue("#status", ""); err != nil {
		t.Fatalf("AssertValue(#status, empty) error = %v", err)
	}
}

func TestForeachAttachedClickHandlerReassignsOuterStateInIIFEPageFlow(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="open" type="button">Open</button>
		<button id="reset" type="button">Reset</button>
		<input id="burn" value="" />
		<input id="price" value="" />
		<input id="idle" value="" />
		<p id="status"></p>
		<script>
		  (() => {
		    const messages = { reset: "Inputs reset." };
		    const els = {
		      openButton: document.getElementById("open"),
		      resetButtons: [document.getElementById("reset")],
		      burn: document.getElementById("burn"),
		      price: document.getElementById("price"),
		      idle: document.getElementById("idle"),
		      status: document.getElementById("status"),
		    };

		    function createDefaultState() {
		      return {
		        presetKind: "small",
		        fuelBurnValue: "20",
		        priceValue: "",
		        idleMinutes: "",
		      };
		    }

		    let state = createDefaultState();

		    function setStatus(text) {
		      els.status.textContent = text || "";
		    }

		    function renderAll() {
		      els.burn.value = state.fuelBurnValue;
		      els.price.value = state.priceValue;
		      els.idle.value = state.idleMinutes;
		    }

		    function attachEvents() {
		      els.openButton.addEventListener("click", () => {
		        state.priceValue = "155";
		        state.idleMinutes = "300";
		        renderAll();
		      });
		      els.resetButtons.forEach((button) => button?.addEventListener("click", () => {
		        state = createDefaultState();
		        renderAll();
		        setStatus(messages.reset);
		      }));
		    }

		    renderAll();
		    attachEvents();
		  })();
		</script>
	`)

	if err := harness.Click("#open"); err != nil {
		t.Fatalf("Click(#open) error = %v", err)
	}
	if err := harness.AssertValue("#price", "155"); err != nil {
		t.Fatalf("AssertValue(#price, 155) error = %v", err)
	}
	if err := harness.AssertValue("#idle", "300"); err != nil {
		t.Fatalf("AssertValue(#idle, 300) error = %v", err)
	}
	if err := harness.Click("#reset"); err != nil {
		t.Fatalf("Click(#reset) error = %v", err)
	}
	if err := harness.AssertValue("#burn", "20"); err != nil {
		t.Fatalf("AssertValue(#burn, 20) error = %v", err)
	}
	if err := harness.AssertValue("#price", ""); err != nil {
		t.Fatalf("AssertValue(#price, empty) error = %v", err)
	}
	if err := harness.AssertValue("#idle", ""); err != nil {
		t.Fatalf("AssertValue(#idle, empty) error = %v", err)
	}
	if err := harness.AssertText("#status", "Inputs reset."); err != nil {
		t.Fatalf("AssertText(#status, Inputs reset.) error = %v", err)
	}
}

func TestForOfLoopSupportsArrayDestructuringBinding(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const entries = Object.entries({ mode: "mooring" });
		  for (const [key, value] of entries) {
		    document.getElementById("out").textContent = key + ":" + value;
		  }
		</script>
	`)

	if err := harness.AssertText("#out", "mode:mooring"); err != nil {
		t.Fatalf("AssertText(#out, mode:mooring) error = %v", err)
	}
}

func TestAppendChildSyncsSelectValueForPreselectedOption(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<select id="s"></select>
		<p id="out"></p>
		<script>
		  const s = document.getElementById("s");
		  ["g", "kg", "ml"].forEach((value) => {
		    const option = document.createElement("option");
		    option.value = value;
		    option.textContent = value;
		    if (value === "ml") {
		      option.selected = true;
		    }
		    s.appendChild(option);
		  });
		  document.getElementById("out").textContent = "value:" + s.value;
		</script>
	`)

	if err := harness.AssertText("#out", "value:ml"); err != nil {
		t.Fatalf("AssertText(#out, value:ml) error = %v", err)
	}
	if err := harness.AssertValue("#s", "ml"); err != nil {
		t.Fatalf("AssertValue(#s, ml) error = %v", err)
	}
}

func TestNestedHelperFunctionRetainsTransitiveOuterCapture(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<button id="run" type="button">Run</button>
		<p id="out"></p>
		<script>
		  (() => {
		    const prefix = "outer";

		    function makeHandler() {
		      function formatValue() {
		        return prefix + "-value";
		      }

		      return () => {
		        document.getElementById("out").textContent = formatValue();
		      };
		    }

		    const handler = makeHandler();
		    document.getElementById("run").addEventListener("click", handler);
		  })();
		</script>
	`)

	if err := harness.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}
	if err := harness.AssertText("#out", "outer-value"); err != nil {
		t.Fatalf("AssertText(#out, outer-value) error = %v", err)
	}
}

func TestClassFieldInitializerKeepsOuterCaptureThroughFactory(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<p id="out"></p>
		<script>
		  (() => {
		    const prefix = "captured";

		    function buildWidget() {
		      class Widget {
		        label = prefix + "-field";
		      }

		      return Widget;
		    }

		    const Widget = buildWidget();
		    document.getElementById("out").textContent = new Widget().label;
		  })();
		</script>
	`)

	if err := harness.AssertText("#out", "captured-field"); err != nil {
		t.Fatalf("AssertText(#out, captured-field) error = %v", err)
	}
}
