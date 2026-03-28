package browsertester

import (
	"fmt"
	"strings"
	"testing"
)

func TestParserGeneratedStatementBlocksDoNotPanic(t *testing.T) {
	cases := []string{
		`
const seed = 1;
const wrapped = [seed, 2];
const first = wrapped[0];
const fallback = first ? first : wrapped[1];
Object.prototype.hasOwnProperty.call({}, String(fallback));
return;
`,
		`
let [first, , third] = [1, 2, 3];
return;
`,
		`
let {kind: label} = {kind: "box"};
return;
`,
		`
let payload = {title: "ready", nested: {value: "changed"}, items: [1, 2, 3]};
return;
`,
		`
if (true) {
  const thenValue = 1;
} else {
  const elseValue = 2;
}
return;
`,
		`
for (let i = 0; i < 3; i = i + 1) {
  const count = i;
}
return;
`,
		`
while (false) {
  break;
}
return;
`,
		`
function helper(arg) {
  return arg;
}
helper(1);
return;
`,
		`
try {
  const done = true;
} catch (err) {
  const fallback = err;
} finally {
  const cleanup = 0;
}
return;
`,
		`
switch ("b") {
  case "a":
    break;
  case "b":
    break;
  default:
    break;
}
return;
`,
		`
class Example {
  static foo() {}
}
return;
`,
		`
const regex = /foo(?=bar)/;
const next = regex.test("foobar");
return;
`,
	}

	for i, body := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			_ = mustHarnessFromHTML(t, parserCallbackHTML(body))
		})
	}
}

func TestParserGeneratedExpressionCombinationsDoNotPanic(t *testing.T) {
	expressions := []string{
		`1`,
		`null`,
		`true`,
		`false`,
		`undefined`,
		`'x'`,
		`'日本語'`,
		`"double"`,
		"`template`",
		`/a/`,
		`/\d+/`,
		`/^\w+$/`,
		`/foo(?=bar)/`,
		`/\/(x|y)/`,
		`/[a-z]{1,3}/gi`,
		`[1, 2, 3]`,
		`{ left: 1, right: 2 }`,
		`1 ? 2 : 3`,
		`!true`,
		`+(1)`,
		`-(2)`,
		`([1, 2, 3])[0]`,
		`({ left: 1, right: 2 }).left`,
	}

	for i, expr := range expressions {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			body := fmt.Sprintf(`
const seed = %s;
const wrapped = [seed, %s];
const first = wrapped[0];
const fallback = first ? first : wrapped[1];
Object.prototype.hasOwnProperty.call({}, String(fallback));
return;
`, expr, expr)
			_ = mustHarnessFromHTML(t, parserCallbackHTML(body))
		})
	}
}

func TestParserScriptBoundaryCombinationsDoNotReportUnclosedScript(t *testing.T) {
	cases := [][]string{
		{`const marker = '<\/script>';`},
		{`const marker = "<\/SCRIPT>";`},
		{"const marker = `x${String('<\\/script>')}y`;"},
		{`const rx = /<\/script>/i;`},
		{`const rxHit = /<\/script>/i.test('<\/script>');`},
		{`const n = 1; // marker: <\/script>`},
		{`/* marker: <\/script> */ const block = 2;`},
		{`const marker = '<\/script>';`, `const rx = /<\/script>/i;`},
		{`const marker = "<\/SCRIPT>";`, `const rxHit = /<\/script>/i.test('<\/script>');`},
	}

	for i, fragments := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			body := strings.Join(fragments, "\n") + `
document.getElementById("run").textContent = "ok";
`
			harness := mustHarnessFromHTML(t, parserBoundaryHTML(body))
			if err := harness.AssertText("#run", "ok"); err != nil {
				t.Fatalf("AssertText(#run, ok) error = %v", err)
			}
		})
	}
}

func parserCallbackHTML(body string) string {
	return fmt.Sprintf(`
<button id="run">run</button>
<script>
document.getElementById("run").addEventListener("click", () => {
%s
});
</script>
`, strings.TrimSpace(body))
}

func parserBoundaryHTML(body string) string {
	return fmt.Sprintf(`
<div id="run">seed</div>
<script>
%s
</script>
`, strings.TrimSpace(body))
}
