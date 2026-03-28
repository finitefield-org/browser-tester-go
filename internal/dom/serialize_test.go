package dom

import "testing"

func TestDumpDOMPreservesAdjustedSVGAttributeCasing(t *testing.T) {
	store := NewStore()
	input := `<div id="probe"><svg xmlns="http://www.w3.org/2000/svg" viewbox="0 0 10 10"><defs><marker id="arrow" viewbox="0 0 4 4" markerwidth="4" markerheight="4" refx="2" refy="2"><path d="M0,0 L4,2 L0,4 z"></path></marker></defs></svg></div>`
	if err := store.BootstrapHTML(input); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	if got, want := store.DumpDOM(), `<div id="probe"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><defs><marker id="arrow" viewBox="0 0 4 4" markerWidth="4" markerHeight="4" refX="2" refY="2"><path d="M0,0 L4,2 L0,4 z"></path></marker></defs></svg></div>`; got != want {
		t.Fatalf("DumpDOM() = %q, want %q", got, want)
	}
}

func TestDumpDOMDoesNotRecaseHTMLAttributesOutsideSVG(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="probe" viewbox="kept-lowercase"></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	if got, want := store.DumpDOM(), `<div id="probe" viewbox="kept-lowercase"></div>`; got != want {
		t.Fatalf("DumpDOM() = %q, want %q", got, want)
	}
}

func TestOuterHTMLForNodePreservesAdjustedSVGAttributeCasing(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="probe"><svg xmlns="http://www.w3.org/2000/svg" viewbox="0 0 10 10"><marker id="arrow" markerwidth="4" markerheight="4" refx="2" refy="2"></marker></svg></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	markerID := mustSelectSingle(t, store, "marker")
	got, err := store.OuterHTMLForNode(markerID)
	if err != nil {
		t.Fatalf("OuterHTMLForNode() error = %v", err)
	}
	if want := `<marker id="arrow" markerWidth="4" markerHeight="4" refX="2" refY="2"></marker>`; got != want {
		t.Fatalf("OuterHTMLForNode() = %q, want %q", got, want)
	}
}
