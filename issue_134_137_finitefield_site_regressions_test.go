package browsertester

import "testing"

func TestIssue134FinitefieldSiteRegressionObjectAssignGlobalIsAvailable(t *testing.T) {
	harness, err := FromHTML(`
		<main>
		  <p id='out'></p>
		  <script>
		    const out = document.getElementById('out');
		    try {
		      const target = { a: 1 };
		      const src = { b: 2 };
		      Object.assign(target, src);
		      out.textContent = String(target.a) + ':' + String(target.b);
		    } catch (err) {
		      out.textContent = 'err:' + String(err && err.message ? err.message : err);
		    }
		  </script>
		</main>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1:2" {
		t.Fatalf("TextContent(#out) = %q, want 1:2", got)
	}
}

func TestIssue135FinitefieldSiteRegressionOptionalChainingListenerOnMemberPathParsesAndRuns(t *testing.T) {
	harness, err := FromHTML(`
		<main>
		  <button id='btn'>run</button>
		  <p id='out'></p>
		  <script>
		    const actionEls = { close: document.getElementById('btn') };
		    actionEls.close?.addEventListener('click', () => {
		      document.getElementById('out').textContent = 'ok';
		    });
		  </script>
		</main>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
}

func TestIssue134FinitefieldSiteRegressionObjectAssignReturnsTargetAndIgnoresNullishSources(t *testing.T) {
	harness, err := FromHTML(`
		<main>
		  <p id='out'></p>
		  <script>
		    const target = { a: 1, b: 1 };
		    const returned = Object.assign(target, null, { b: 4 }, undefined, { c: 5 });
		    document.getElementById('out').textContent = [
		      String(target.a),
		      String(target.b),
		      String(target.c),
		      String(returned === target),
		    ].join('|');
		  </script>
		</main>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|4|5|true" {
		t.Fatalf("TextContent(#out) = %q, want 1|4|5|true", got)
	}
}

func TestIssue134FinitefieldSiteRegressionObjectAssignCopiesSymbolAndStringSourceKeys(t *testing.T) {
	harness, err := FromHTML(`
		<main>
		  <p id='out'></p>
		  <script>
		    const sym = Symbol('token');
		    const copied = Object.assign({}, { [sym]: 'sym', x: 'x' });
		    const fromString = Object.assign({}, 'abc');
		    const symbols = Object.getOwnPropertySymbols(copied);
		    document.getElementById('out').textContent = [
		      String(symbols.length),
		      String(copied[sym]),
		      String(copied.x),
		      Object.keys(fromString).join(','),
		      fromString[0] + fromString[1] + fromString[2],
		    ].join('|');
		  </script>
		</main>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|sym|x|0,1,2|abc" {
		t.Fatalf("TextContent(#out) = %q, want 1|sym|x|0,1,2|abc", got)
	}
}

func TestIssue134FinitefieldSiteRegressionObjectAssignUsesGettersAndSetters(t *testing.T) {
	harness, err := FromHTML(`
		<main>
		  <p id='out'></p>
		  <script>
		    let getCount = 0;
		    let setTotal = 0;
		    const source = {
		      get amount() {
		        getCount += 1;
		        return 7;
		      }
		    };
		    const target = {
		      set amount(value) {
		        setTotal += value;
		      }
		    };
		    Object.assign(target, source);
		    document.getElementById('out').textContent = [
		      String(getCount),
		      String(setTotal),
		    ].join('|');
		  </script>
		</main>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|7" {
		t.Fatalf("TextContent(#out) = %q, want 1|7", got)
	}
}

func TestIssue134FinitefieldSiteRegressionObjectAssignWrapsPrimitiveTargetAndRejectsNullTarget(t *testing.T) {
	harness, err := FromHTML(`
		<main>
		  <p id='out'></p>
		  <script>
		    const wrapped = Object.assign(3, { a: 1 });
		    let threwForNull = false;
		    try {
		      Object.assign(null, { a: 1 });
		    } catch (err) {
		      threwForNull = String(err && err.message ? err.message : err)
		        .includes('Cannot convert undefined or null to object');
		    }
		    document.getElementById('out').textContent = [
		      typeof wrapped,
		      String(wrapped.a),
		      String(threwForNull),
		    ].join('|');
		  </script>
		</main>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "object|1|true" {
		t.Fatalf("TextContent(#out) = %q, want object|1|true", got)
	}
}

func TestIssue136FinitefieldSiteRegressionHtmlButtonElementGlobalSupportsInstanceofChecks(t *testing.T) {
	harness, err := FromHTML(`
		<main>
		  <button id='btn'>run</button>
		  <p id='out'></p>
		  <script>
		    document.getElementById('btn').addEventListener('click', (event) => {
		      const checks = [
		        typeof HTMLButtonElement,
		        String(window.HTMLButtonElement === HTMLButtonElement),
		        String(event.target instanceof HTMLButtonElement),
		        String(event.target instanceof HTMLElement),
		      ];
		      document.getElementById('out').textContent = checks.join('|');
		    });
		  </script>
		</main>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "function|true|true|true" {
		t.Fatalf("TextContent(#out) = %q, want function|true|true|true", got)
	}
}

func TestIssue137FinitefieldSiteRegressionToFixedChainParsesAfterEscapeNormalizationWithUnicode(t *testing.T) {
	html := "<main>" +
		"<p id='out'></p>" +
		"<script>" +
		"const quotePair = \"\\\"\\\"\";" +
		"const label = " + "`ABC-001 (${quotePair.length} 件)`" + ";" +
		"const rect = { w: 4.2 };" +
		"const formatted = Math.max(0, rect.w).toFixed(2);" +
		"document.getElementById('out').textContent = " + "`" + "${label}|${formatted}" + "`" + ";" +
		"</script>" +
		"</main>"

	harness, err := FromHTML(html)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ABC-001 (2 件)|4.20" {
		t.Fatalf("TextContent(#out) = %q, want ABC-001 (2 件)|4.20", got)
	}
}
