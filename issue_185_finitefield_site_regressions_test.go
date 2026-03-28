package browsertester

import "testing"

func TestIssue185InlineObjectLiteralComputedLookupReturnsSelectedValue(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="out"></div>
		<script>
		  const backgroundCss = {
		    checker: "background:#f8fafc;",
		    dark: "background:#0f172a;"
		  }["dark"] || "background:#ffffff;";

		  const zoomScale = {
		    fit: 1,
		    "200": 2
		  }["200"] || 1;

		  document.getElementById("out").textContent =
		    backgroundCss + "|" + String(zoomScale);
		</script>
	`)

	if err := harness.AssertText("#out", "background:#0f172a;|2"); err != nil {
		t.Fatalf("AssertText(#out, background:#0f172a;|2) error = %v", err)
	}
}

func TestIssue185InlineObjectLiteralLookupSurvivesTemplateInterpolation(t *testing.T) {
	harness := mustHarnessFromHTML(t,
		"<pre id=\"out\"></pre><script>"+
			"const srcdoc = "+"`"+`
		      <style>
		        body { overflow: auto; ${
		          {
		            checker: "background:#f8fafc;",
		            dark: "background:#0f172a;"
		          }["dark"] || "background:#ffffff;"
		        } }
		        svg { transform: scale(${
		          {
		            fit: 1,
		            "200": 2
		          }["200"] || 1
		        }); }
		      </style>
		    `+"`"+`;`+
			"document.getElementById(\"out\").textContent = "+
			"srcdoc.includes(\"background:#0f172a;\") && "+
			"srcdoc.includes(\"transform: scale(2);\") ? \"ok\" : srcdoc;"+
			"</script>",
	)

	if err := harness.AssertText("#out", "ok"); err != nil {
		t.Fatalf("AssertText(#out, ok) error = %v", err)
	}
}
