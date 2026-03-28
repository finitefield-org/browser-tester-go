package browsertester

import (
	"strings"
	"testing"
)

func TestIssue211DumpDOMPreservesAdjustedSVGAttributeCasing(t *testing.T) {
	harness := mustHarnessFromHTML(t, `
		<div id="probe">
		  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">
		    <defs>
		      <marker
		        id="arrow"
		        viewBox="0 0 4 4"
		        markerWidth="4"
		        markerHeight="4"
		        refX="2"
		        refY="2"
		      >
		        <path d="M0,0 L4,2 L0,4 z"></path>
		      </marker>
		    </defs>
		  </svg>
		</div>
	`)

	snippet := harness.Debug().DumpDOM()

	if !containsAll(snippet, []string{
		`viewBox="0 0 10 10"`,
		`viewBox="0 0 4 4"`,
		`markerWidth="4"`,
		`markerHeight="4"`,
		`refX="2"`,
		`refY="2"`,
	}) {
		t.Fatalf("Debug().DumpDOM() = %q, want preserved SVG attribute casing", snippet)
	}
	if containsAny(snippet, []string{
		`viewbox=`,
		`markerwidth=`,
		`markerheight=`,
		`refx=`,
		`refy=`,
	}) {
		t.Fatalf("Debug().DumpDOM() = %q, want no lowercased adjusted SVG attributes", snippet)
	}
}

func containsAll(haystack string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

func TestIssue211DumpDOMDoesNotRecaseHTMLAttributesOutsideSVG(t *testing.T) {
	harness := mustHarnessFromHTML(t, `<div id="probe" viewbox="kept-lowercase"></div>`)

	snippet := harness.Debug().DumpDOM()

	if !containsAll(snippet, []string{`viewbox="kept-lowercase"`}) {
		t.Fatalf("Debug().DumpDOM() = %q, want HTML attribute casing preserved", snippet)
	}
	if containsAny(snippet, []string{`viewBox="kept-lowercase"`}) {
		t.Fatalf("Debug().DumpDOM() = %q, did not want SVG casing on HTML attribute", snippet)
	}
}
