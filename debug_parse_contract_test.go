package browsertester

import (
	"strings"
	"testing"
)

func mustDebugParseHarness(t *testing.T, html string) *Harness {
	t.Helper()

	harness, err := FromHTML(html)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}
	return harness
}

func mustDebugParseClick(t *testing.T, harness *Harness, selector string) {
	t.Helper()

	if err := harness.Click(selector); err != nil {
		t.Fatalf("Click(%s) error = %v", selector, err)
	}
}

func mustDebugParseTextContent(t *testing.T, harness *Harness, selector string) string {
	t.Helper()

	got, err := harness.TextContent(selector)
	if err != nil {
		t.Fatalf("TextContent(%s) error = %v", selector, err)
	}
	return got
}

func TestDebugParseSingleReportsBootstrapError(t *testing.T) {
	harness := mustDebugParseHarness(t, `<script>const a = document.getElementById('a'); document.getElementById('btn').addEventListener('click', () => {});</script>`)

	if _, err := harness.TextContent("body"); err == nil {
		t.Fatalf("TextContent(body) error = nil, want DOM bootstrap error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(body) error = %#v, want DOM error", err)
	}
	if got := harness.Debug().DOMReady(); got {
		t.Fatalf("Debug().DOMReady() = %v, want false after bootstrap failure", got)
	}
	if got := harness.Debug().DOMError(); !strings.Contains(got, `cannot access property "addEventListener" on nullish value`) {
		t.Fatalf("Debug().DOMError() = %q, want nullish addEventListener error text", got)
	}
}

func TestDebugParseFocusActiveElementTernary(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<input id='a'>
		<input id='b'>
		<button id='btn'>run</button>
		<p id='result'></p>
		<script>
		  const a = document.getElementById('a');
		  const b = document.getElementById('b');
		  let order = '';

		  a.addEventListener('focus', () => {
		    order += 'aF';
		  });
		  a.addEventListener('blur', () => {
		    order += 'aB';
		  });
		  b.addEventListener('focus', () => {
		    order += 'bF';
		  });
		  b.addEventListener('blur', () => {
		    order += 'bB';
		  });

		  document.getElementById('btn').addEventListener('click', () => {
		    a.focus();
		    b.focus();
		    b.blur();
		    document.getElementById('result').textContent =
		      order + ':' + (document.activeElement === null ? 'none' : 'active');
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#btn")
	if got, want := mustDebugParseTextContent(t, harness, "#result"), "aFaBbFbB:active"; got != want {
		t.Fatalf("TextContent(#result) = %q, want %q", got, want)
	}
}

func TestDebugParseActiveElementTernaryDirect(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<button id='btn'>run</button>
		<p id='result'></p>
		<script>
		  document.getElementById('btn').addEventListener('click', () => {
		    document.getElementById('result').textContent =
		      document.activeElement === null ? 'none' : 'active';
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#btn")
	if got, want := mustDebugParseTextContent(t, harness, "#result"), "active"; got != want {
		t.Fatalf("TextContent(#result) = %q, want %q", got, want)
	}
}

func TestDebugParseConcatAndTernary(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<button id='btn'>run</button>
		<p id='result'></p>
		<p id='concat2'></p>
		<p id='concat3'></p>
		<script>
		  document.getElementById('btn').addEventListener('click', () => {
		    const order = 'aFaBbFbB';
		    document.getElementById('result').textContent =
		      order + ':' + (document.activeElement === null ? 'none' : 'active');
		    document.getElementById('concat2').textContent =
		      order + (document.activeElement === null ? 'none' : 'active');
		    document.getElementById('concat3').textContent =
		      (document.activeElement === null ? 'none' : 'active');
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#btn")
	if got, want := mustDebugParseTextContent(t, harness, "#result"), "aFaBbFbB:active"; got != want {
		t.Fatalf("TextContent(#result) = %q, want %q", got, want)
	}
	if got, want := mustDebugParseTextContent(t, harness, "#concat2"), "aFaBbFbBactive"; got != want {
		t.Fatalf("TextContent(#concat2) = %q, want %q", got, want)
	}
	if got, want := mustDebugParseTextContent(t, harness, "#concat3"), "active"; got != want {
		t.Fatalf("TextContent(#concat3) = %q, want %q", got, want)
	}
}

func TestDebugParseTernaryVariable(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<button id='btn'>run</button>
		<p id='result'></p>
		<script>
		  document.getElementById('btn').addEventListener('click', () => {
		    const suffix = document.activeElement === null ? 'none' : 'active';
		    document.getElementById('result').textContent = 'start:' + suffix;
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#btn")
	if got, want := mustDebugParseTextContent(t, harness, "#result"), "start:active"; got != want {
		t.Fatalf("TextContent(#result) = %q, want %q", got, want)
	}
}

func TestDebugParseWhileLoop(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<button id='btn'>run</button>
		<p id='result'></p>
		<script>
		  document.getElementById('btn').addEventListener('click', () => {
		    let counter = 0;
		    let text = '';
		    while (counter < 3) {
		      text += 'x';
		      counter = counter + 1;
		    };
		    document.getElementById('result').textContent = text + ':' + counter;
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#btn")
	if got, want := mustDebugParseTextContent(t, harness, "#result"), "xxx:3"; got != want {
		t.Fatalf("TextContent(#result) = %q, want %q", got, want)
	}
}

func TestDebugParseForLoop(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<button id='btn'>run</button>
		<p id='result'></p>
		<script>
		  document.getElementById('btn').addEventListener('click', () => {
		    let text = '';
		    for (let i = 0; i < 3; i = i + 1) {
		      text += 'y';
		    };
		    document.getElementById('result').textContent = text;
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#btn")
	if got, want := mustDebugParseTextContent(t, harness, "#result"), "yyy"; got != want {
		t.Fatalf("TextContent(#result) = %q, want %q", got, want)
	}
}

func TestDebugParseIfBlockAndNextStatementWithoutSemicolon(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<button id='btn'>run</button>
		<p id='result'></p>
		<script>
		  document.getElementById('btn').addEventListener('click', () => {
		    let text = '';
		    if (true) {
		      text += 'x';
		    }
		    text += 'y';
		    document.getElementById('result').textContent = text;
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#btn")
	if got, want := mustDebugParseTextContent(t, harness, "#result"), "xy"; got != want {
		t.Fatalf("TextContent(#result) = %q, want %q", got, want)
	}
}

func TestDebugParseParenthesizedFormulaTrace(t *testing.T) {
	harness := mustDebugParseHarness(t, `
		<button id="go" type="button">go</button>
		<div id="out"></div>
		<script>
		  document.getElementById("go").addEventListener("click", () => {
		    document.getElementById("out").textContent = String((2 + 3) * (4 + 1));
		  });
		</script>
	`)

	mustDebugParseClick(t, harness, "#go")
	if got, want := mustDebugParseTextContent(t, harness, "#out"), "25"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}

func TestDebugParseRecursiveClosureStopCharAndIndex(t *testing.T) {
	harness := mustDebugParseHarness(t, `
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
		      const before = String(close);
		      const inner = parseSequence(close);
		      return "before=" + before + "|after=" + (current() || "<eof>") + "|close=" + close + "|index=" + index + "|" + inner;
		    }

		    return parseBracketGroup();
		  }

		  document.getElementById("go").addEventListener("click", () => {
		    document.getElementById("out").textContent = make("(SO4)3");
		  });
		})();
		</script>
	`)

	mustDebugParseClick(t, harness, "#go")
	if got, want := mustDebugParseTextContent(t, harness, "#out"), "before=)|after=)|close=)|index=4|seen=SO4|stop=)|curr=)|index=4"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}
