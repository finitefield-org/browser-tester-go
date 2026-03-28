package dom

import "testing"

func TestInnerHTMLForNodeAndSetInnerHTML(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><p>Hello</p><span>world</span></div></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	inner, err := store.InnerHTMLForNode(targetID)
	if err != nil {
		t.Fatalf("InnerHTMLForNode(#target) error = %v", err)
	}
	if got, want := inner, `<p>Hello</p><span>world</span>`; got != want {
		t.Fatalf("InnerHTMLForNode(#target) = %q, want %q", got, want)
	}

	if err := store.SetInnerHTML(targetID, `<em id="next">updated</em>tail`); err != nil {
		t.Fatalf("SetInnerHTML(#target) error = %v", err)
	}
	if got, want := store.DumpDOM(), `<section id="wrap"><div id="target"><em id="next">updated</em>tail</div></section>`; got != want {
		t.Fatalf("DumpDOM() after SetInnerHTML = %q, want %q", got, want)
	}
	children, err := store.Children(targetID)
	if err != nil {
		t.Fatalf("Children(#target) after SetInnerHTML error = %v", err)
	}
	if got, want := children.Length(), 1; got != want {
		t.Fatalf("Children(#target).Length() after SetInnerHTML = %d, want %d", got, want)
	}
	if got, ok := children.NamedItem("next"); !ok {
		t.Fatalf("Children(#target).NamedItem(next) = (%d, %v), want inserted child", got, ok)
	}
	if ids, err := store.Select("p"); err != nil || len(ids) != 0 {
		t.Fatalf("Select(p) after SetInnerHTML = (%v, %v), want no matches", ids, err)
	}
}

func TestTextContentForNodeAndSetTextContent(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><p>Hello</p><span>world</span></div><p id="tail">tail</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	if got, want := store.TextContentForNode(targetID), "Helloworld"; got != want {
		t.Fatalf("TextContentForNode(#target) = %q, want %q", got, want)
	}

	if err := store.SetTextContent(targetID, `plain <text> & more`); err != nil {
		t.Fatalf("SetTextContent(#target) error = %v", err)
	}
	if got, want := store.DumpDOM(), `<section id="wrap"><div id="target">plain &lt;text&gt; &amp; more</div><p id="tail">tail</p></section>`; got != want {
		t.Fatalf("DumpDOM() after SetTextContent = %q, want %q", got, want)
	}
	if got, want := store.TextContentForNode(targetID), `plain <text> & more`; got != want {
		t.Fatalf("TextContentForNode(#target) after SetTextContent = %q, want %q", got, want)
	}
	if ids, err := store.Select("span"); err != nil || len(ids) != 0 {
		t.Fatalf("Select(span) after SetTextContent = (%v, %v), want no matches", ids, err)
	}
}

func TestContainsNodeReportsInclusiveDescendants(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="child">x</span></div><p id="sibling">y</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	docID := store.DocumentID()
	wrapID := mustSelectSingle(t, store, "#wrap")
	targetID := mustSelectSingle(t, store, "#target")
	childID := mustSelectSingle(t, store, "#child")
	siblingID := mustSelectSingle(t, store, "#sibling")

	if !store.ContainsNode(docID, childID) {
		t.Fatalf("ContainsNode(document, child) = false, want true")
	}
	if !store.ContainsNode(wrapID, childID) {
		t.Fatalf("ContainsNode(#wrap, #child) = false, want true")
	}
	if !store.ContainsNode(targetID, targetID) {
		t.Fatalf("ContainsNode(#target, #target) = false, want true")
	}
	if !store.ContainsNode(childID, childID) {
		t.Fatalf("ContainsNode(#child, #child) = false, want true")
	}
	if store.ContainsNode(childID, targetID) {
		t.Fatalf("ContainsNode(#child, #target) = true, want false")
	}
	if store.ContainsNode(targetID, siblingID) {
		t.Fatalf("ContainsNode(#target, #sibling) = true, want false")
	}
}

func TestIsConnectedTracksConnectionState(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="child">x</span></div></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	docID := store.DocumentID()
	wrapID := mustSelectSingle(t, store, "#wrap")
	targetID := mustSelectSingle(t, store, "#target")
	childID := mustSelectSingle(t, store, "#child")
	orphanID, err := store.CreateElement("em")
	if err != nil {
		t.Fatalf("CreateElement(em) error = %v", err)
	}

	if !store.IsConnected(docID) {
		t.Fatalf("IsConnected(document) = false, want true")
	}
	if !store.IsConnected(wrapID) {
		t.Fatalf("IsConnected(#wrap) = false, want true")
	}
	if !store.IsConnected(targetID) {
		t.Fatalf("IsConnected(#target) = false, want true")
	}
	if !store.IsConnected(childID) {
		t.Fatalf("IsConnected(#child) = false, want true")
	}
	if store.IsConnected(orphanID) {
		t.Fatalf("IsConnected(orphan) = true, want false")
	}

	if err := store.AppendChild(wrapID, orphanID); err != nil {
		t.Fatalf("AppendChild(#wrap, orphan) error = %v", err)
	}
	if !store.IsConnected(orphanID) {
		t.Fatalf("IsConnected(orphan) after AppendChild = false, want true")
	}

	if err := store.RemoveNode(orphanID); err != nil {
		t.Fatalf("RemoveNode(orphan) error = %v", err)
	}
	if store.IsConnected(orphanID) {
		t.Fatalf("IsConnected(orphan) after RemoveNode = true, want false")
	}
}

func TestRootNodeIDReturnsTreeRoot(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="child">x</span></div></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	docID := store.DocumentID()
	wrapID := mustSelectSingle(t, store, "#wrap")
	targetID := mustSelectSingle(t, store, "#target")
	childID := mustSelectSingle(t, store, "#child")
	orphanID, err := store.CreateElement("em")
	if err != nil {
		t.Fatalf("CreateElement(em) error = %v", err)
	}

	if got := store.RootNodeID(docID); got != docID {
		t.Fatalf("RootNodeID(document) = %d, want %d", got, docID)
	}
	if got := store.RootNodeID(wrapID); got != docID {
		t.Fatalf("RootNodeID(#wrap) = %d, want %d", got, docID)
	}
	if got := store.RootNodeID(targetID); got != docID {
		t.Fatalf("RootNodeID(#target) = %d, want %d", got, docID)
	}
	if got := store.RootNodeID(childID); got != docID {
		t.Fatalf("RootNodeID(#child) = %d, want %d", got, docID)
	}
	if got := store.RootNodeID(orphanID); got != orphanID {
		t.Fatalf("RootNodeID(orphan) = %d, want %d", got, orphanID)
	}
	if got := store.RootNodeID(0); got != 0 {
		t.Fatalf("RootNodeID(0) = %d, want 0", got)
	}
}

func TestCompareDocumentPositionReportsTreeOrder(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="child">x</span></div><p id="sibling">y</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	docID := store.DocumentID()
	wrapID := mustSelectSingle(t, store, "#wrap")
	targetID := mustSelectSingle(t, store, "#target")
	childID := mustSelectSingle(t, store, "#child")
	siblingID := mustSelectSingle(t, store, "#sibling")
	orphanID, err := store.CreateElement("em")
	if err != nil {
		t.Fatalf("CreateElement(em) error = %v", err)
	}

	if got := store.CompareDocumentPosition(docID, docID); got != 0 {
		t.Fatalf("CompareDocumentPosition(document, document) = %d, want 0", got)
	}
	if got := store.CompareDocumentPosition(docID, childID); got != 12 {
		t.Fatalf("CompareDocumentPosition(document, child) = %d, want 12", got)
	}
	if got := store.CompareDocumentPosition(childID, docID); got != 18 {
		t.Fatalf("CompareDocumentPosition(child, document) = %d, want 18", got)
	}
	if got := store.CompareDocumentPosition(wrapID, childID); got != 12 {
		t.Fatalf("CompareDocumentPosition(#wrap, #child) = %d, want 12", got)
	}
	if got := store.CompareDocumentPosition(childID, siblingID); got != 4 {
		t.Fatalf("CompareDocumentPosition(#child, #sibling) = %d, want 4", got)
	}
	if got := store.CompareDocumentPosition(orphanID, docID); got != 35 {
		t.Fatalf("CompareDocumentPosition(orphan, document) = %d, want 35", got)
	}
	if got := store.CompareDocumentPosition(docID, orphanID); got != 37 {
		t.Fatalf("CompareDocumentPosition(document, orphan) = %d, want 37", got)
	}
	if got := store.CompareDocumentPosition(targetID, targetID); got != 0 {
		t.Fatalf("CompareDocumentPosition(#target, #target) = %d, want 0", got)
	}
}

func TestNormalizeMergesAdjacentTextNodesAndRemovesEmptyTextNodes(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="main"><div id="inner">abc</div></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	mainID := mustSelectSingle(t, store, "#main")
	innerID := mustSelectSingle(t, store, "#inner")
	innerChildren, err := store.ChildNodes(innerID)
	if err != nil {
		t.Fatalf("ChildNodes(#inner) error = %v", err)
	}
	innerTextID, ok := innerChildren.Item(0)
	if !ok {
		t.Fatalf("ChildNodes(#inner).Item(0) = no node, want text node")
	}
	if _, err := store.SplitText(innerTextID, 1); err != nil {
		t.Fatalf("SplitText(#inner text, 1) error = %v", err)
	}

	mergedID, err := store.CreateTextNode("x")
	if err != nil {
		t.Fatalf("CreateTextNode(x) error = %v", err)
	}
	if err := store.AppendChild(mainID, mergedID); err != nil {
		t.Fatalf("AppendChild(#main, merged) error = %v", err)
	}
	tailID, err := store.CreateTextNode("y")
	if err != nil {
		t.Fatalf("CreateTextNode(y) error = %v", err)
	}
	if err := store.AppendChild(mainID, tailID); err != nil {
		t.Fatalf("AppendChild(#main, tail) error = %v", err)
	}
	emptyID, err := store.CreateTextNode("")
	if err != nil {
		t.Fatalf("CreateTextNode(\"\") error = %v", err)
	}
	if err := store.AppendChild(mainID, emptyID); err != nil {
		t.Fatalf("AppendChild(#main, empty) error = %v", err)
	}

	if got, want := store.TextContentForNode(mainID), "abcxy"; got != want {
		t.Fatalf("TextContentForNode(#main) before normalize = %q, want %q", got, want)
	}
	if err := store.Normalize(mainID); err != nil {
		t.Fatalf("Normalize(#main) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<section id="main"><div id="inner">abc</div>xy</section>`; got != want {
		t.Fatalf("DumpDOM() after Normalize = %q, want %q", got, want)
	}
	innerChildren, err = store.ChildNodes(innerID)
	if err != nil {
		t.Fatalf("ChildNodes(#inner) after normalize error = %v", err)
	}
	if got, want := innerChildren.Length(), 1; got != want {
		t.Fatalf("ChildNodes(#inner).Length() after normalize = %d, want %d", got, want)
	}
	if got, want := store.Node(innerTextID).Text, "abc"; got != want {
		t.Fatalf("Normalize merged inner text = %q, want %q", got, want)
	}
	if got, want := store.Node(mergedID).Text, "xy"; got != want {
		t.Fatalf("Normalize merged sibling text = %q, want %q", got, want)
	}
	if store.Node(tailID) != nil {
		t.Fatalf("Normalize kept merged text node %d, want it removed", tailID)
	}
	if store.Node(emptyID) != nil {
		t.Fatalf("Normalize kept empty text node %d, want it removed", emptyID)
	}
}

func TestSetOuterHTMLReplacesNodeAndPreservesSiblings(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><b>x</b></div><p id="tail">tail</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	if err := store.SetOuterHTML(targetID, `<article id="next">n</article><aside id="extra"></aside>`); err != nil {
		t.Fatalf("SetOuterHTML(#target) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<section id="wrap"><article id="next">n</article><aside id="extra"></aside><p id="tail">tail</p></section>`; got != want {
		t.Fatalf("DumpDOM() after SetOuterHTML = %q, want %q", got, want)
	}
	if ids, err := store.Select("#target"); err != nil || len(ids) != 0 {
		t.Fatalf("Select(#target) after replacement = (%v, %v), want no matches", ids, err)
	}
}

func TestInsertAdjacentHTMLPositions(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="inside">x</span></div><p id="tail">t</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	if err := store.InsertAdjacentHTML(targetID, "beforebegin", `<a id="bb"></a>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(beforebegin) error = %v", err)
	}
	if err := store.InsertAdjacentHTML(targetID, "afterbegin", `<i id="ab">a</i>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(afterbegin) error = %v", err)
	}
	if err := store.InsertAdjacentHTML(targetID, "beforeend", `<i id="be">b</i>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(beforeend) error = %v", err)
	}
	if err := store.InsertAdjacentHTML(targetID, "afterend", `<a id="ae"></a>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(afterend) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<section id="wrap"><a id="bb"></a><div id="target"><i id="ab">a</i><span id="inside">x</span><i id="be">b</i></div><a id="ae"></a><p id="tail">t</p></section>`; got != want {
		t.Fatalf("DumpDOM() after InsertAdjacentHTML = %q, want %q", got, want)
	}
	wrapID := mustSelectSingle(t, store, "#wrap")
	children, err := store.Children(wrapID)
	if err != nil {
		t.Fatalf("Children(#wrap) after InsertAdjacentHTML error = %v", err)
	}
	if got, want := children.Length(), 4; got != want {
		t.Fatalf("Children(#wrap).Length() after InsertAdjacentHTML = %d, want %d", got, want)
	}
	wantIDs := []string{"bb", "target", "ae", "tail"}
	for i, wantID := range wantIDs {
		id, ok := children.Item(i)
		if !ok {
			t.Fatalf("Children(#wrap).Item(%d) = (0, false), want %q", i, wantID)
		}
		node := store.Node(id)
		if node == nil {
			t.Fatalf("Children(#wrap).Item(%d) node = nil", i)
		}
		gotID, ok := attributeValue(node.Attrs, "id")
		if !ok || gotID != wantID {
			t.Fatalf("Children(#wrap).Item(%d) id = (%q, %v), want %q", i, gotID, ok, wantID)
		}
	}
}

func TestReplaceChildrenWithNodeIDsMovesChildrenAndDeletesRemovedSubtrees(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="keep">keep</span><b id="drop"><i id="gone">gone</i></b></div><p id="source"><em id="moved">move</em></p><div id="probe"></div></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	keepID := mustSelectSingle(t, store, "#keep")
	sourceID := mustSelectSingle(t, store, "#source")
	movedID := mustSelectSingle(t, store, "#moved")
	dropID := mustSelectSingle(t, store, "#drop")
	goneID := mustSelectSingle(t, store, "#gone")

	if err := store.ReplaceChildrenWithNodeIDs(targetID, []NodeID{sourceID, keepID}); err != nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs(#target, source, keep) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<section id="wrap"><div id="target"><p id="source"><em id="moved">move</em></p><span id="keep">keep</span></div><div id="probe"></div></section>`; got != want {
		t.Fatalf("DumpDOM() after ReplaceChildrenWithNodeIDs = %q, want %q", got, want)
	}
	if store.Node(keepID) == nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs removed retained node %d", keepID)
	}
	if got := store.Node(sourceID); got == nil || got.Parent != targetID {
		t.Fatalf("ReplaceChildrenWithNodeIDs(source) parent = %#v, want parent %d", got, targetID)
	}
	if store.Node(movedID) == nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs removed moved descendant %d", movedID)
	}
	if store.Node(dropID) != nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs kept removed node %d", dropID)
	}
	if store.Node(goneID) != nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs kept removed descendant %d", goneID)
	}
}

func TestReplaceChildrenWithNodeIDsPreservesFocusedAndTargetedRetainedNodes(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="keep">keep</span><b id="drop">drop</b></div><p id="source">move</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	keepID := mustSelectSingle(t, store, "#keep")
	sourceID := mustSelectSingle(t, store, "#source")

	if err := store.SetFocusedNode(keepID); err != nil {
		t.Fatalf("SetFocusedNode(#keep) error = %v", err)
	}
	store.SyncTargetFromURL("https://example.test/page#keep")
	if got := store.TargetNodeID(); got != keepID {
		t.Fatalf("TargetNodeID() before ReplaceChildrenWithNodeIDs = %d, want %d", got, keepID)
	}

	if err := store.ReplaceChildrenWithNodeIDs(targetID, []NodeID{sourceID, keepID}); err != nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs(#target, source, keep) error = %v", err)
	}

	if got := store.FocusedNodeID(); got != keepID {
		t.Fatalf("FocusedNodeID() after ReplaceChildrenWithNodeIDs = %d, want %d", got, keepID)
	}
	if got := store.TargetNodeID(); got != keepID {
		t.Fatalf("TargetNodeID() after ReplaceChildrenWithNodeIDs = %d, want %d", got, keepID)
	}
}

func TestReplaceChildrenWithNodeIDsSupportsDocumentNode(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<main id="root"></main>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	docID := store.DocumentID()
	sectionID, err := store.CreateElement("section")
	if err != nil {
		t.Fatalf("CreateElement(section) error = %v", err)
	}
	if err := store.SetAttribute(sectionID, "id", "first"); err != nil {
		t.Fatalf("SetAttribute(section, id) error = %v", err)
	}
	emID, err := store.CreateElement("em")
	if err != nil {
		t.Fatalf("CreateElement(em) error = %v", err)
	}
	textID, err := store.CreateTextNode("x")
	if err != nil {
		t.Fatalf("CreateTextNode(x) error = %v", err)
	}
	if err := store.AppendChild(emID, textID); err != nil {
		t.Fatalf("AppendChild(em, text) error = %v", err)
	}
	if err := store.AppendChild(sectionID, emID); err != nil {
		t.Fatalf("AppendChild(section, em) error = %v", err)
	}
	probeID, err := store.CreateElement("div")
	if err != nil {
		t.Fatalf("CreateElement(div) error = %v", err)
	}
	if err := store.SetAttribute(probeID, "id", "second"); err != nil {
		t.Fatalf("SetAttribute(div, id) error = %v", err)
	}

	if err := store.ReplaceChildrenWithNodeIDs(docID, []NodeID{sectionID, probeID}); err != nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs(document, section, probe) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<section id="first"><em>x</em></section><div id="second"></div>`; got != want {
		t.Fatalf("DumpDOM() after ReplaceChildrenWithNodeIDs(document) = %q, want %q", got, want)
	}
	if ids, err := store.Select("#root"); err != nil || len(ids) != 0 {
		t.Fatalf("Select(#root) after ReplaceChildrenWithNodeIDs(document) = (%v, %v), want no matches", ids, err)
	}
}

func TestReplaceChildrenWithNodeIDsRejectsSelfInsertion(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="keep">keep</span></div></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	if err := store.ReplaceChildrenWithNodeIDs(targetID, []NodeID{targetID}); err == nil {
		t.Fatalf("ReplaceChildrenWithNodeIDs(#target, self) error = nil, want self-insertion error")
	}
}

func TestRemoveNodeRemovesSubtree(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="remove"><span id="child">x</span></div><p id="keep">k</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	removeID := mustSelectSingle(t, store, "#remove")
	if err := store.RemoveNode(removeID); err != nil {
		t.Fatalf("RemoveNode(#remove) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<section id="wrap"><p id="keep">k</p></section>`; got != want {
		t.Fatalf("DumpDOM() after RemoveNode = %q, want %q", got, want)
	}
	if ids, err := store.Select("#child"); err != nil || len(ids) != 0 {
		t.Fatalf("Select(#child) after RemoveNode = (%v, %v), want no matches", ids, err)
	}
}

func TestMutationHelpersUpdateFocusedNodeState(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="child">x</span></div><p id="keep">k</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	childID := mustSelectSingle(t, store, "#child")
	keepID := mustSelectSingle(t, store, "#keep")

	if err := store.SetFocusedNode(childID); err != nil {
		t.Fatalf("SetFocusedNode(#child) error = %v", err)
	}
	if err := store.SetInnerHTML(targetID, `<em id="next">updated</em>`); err != nil {
		t.Fatalf("SetInnerHTML(#target) error = %v", err)
	}
	if got := store.FocusedNodeID(); got != 0 {
		t.Fatalf("FocusedNodeID() after removing focused descendant = %d, want 0", got)
	}

	if err := store.SetFocusedNode(targetID); err != nil {
		t.Fatalf("SetFocusedNode(#target) error = %v", err)
	}
	if err := store.SetInnerHTML(targetID, `<em id="next">updated</em>`); err != nil {
		t.Fatalf("SetInnerHTML(#target) preserve focus error = %v", err)
	}
	if got := store.FocusedNodeID(); got != targetID {
		t.Fatalf("FocusedNodeID() after SetInnerHTML on focused node = %d, want %d", got, targetID)
	}
	if err := store.SetTextContent(targetID, `plain & more`); err != nil {
		t.Fatalf("SetTextContent(#target) preserve focus error = %v", err)
	}
	if got := store.FocusedNodeID(); got != targetID {
		t.Fatalf("FocusedNodeID() after SetTextContent on focused node = %d, want %d", got, targetID)
	}

	if err := store.SetFocusedNode(targetID); err != nil {
		t.Fatalf("SetFocusedNode(#target) before SetOuterHTML error = %v", err)
	}
	if err := store.SetOuterHTML(targetID, `<article id="next">n</article>`); err != nil {
		t.Fatalf("SetOuterHTML(#target) error = %v", err)
	}
	if got := store.FocusedNodeID(); got != 0 {
		t.Fatalf("FocusedNodeID() after SetOuterHTML = %d, want 0", got)
	}

	if err := store.SetFocusedNode(keepID); err != nil {
		t.Fatalf("SetFocusedNode(#keep) error = %v", err)
	}
	if err := store.RemoveNode(keepID); err != nil {
		t.Fatalf("RemoveNode(#keep) error = %v", err)
	}
	if got := store.FocusedNodeID(); got != 0 {
		t.Fatalf("FocusedNodeID() after RemoveNode = %d, want 0", got)
	}
}

func TestMutationHelpersUpdateTargetNodeState(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="child">x</span></div><p id="keep">k</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	childID := mustSelectSingle(t, store, "#child")
	keepID := mustSelectSingle(t, store, "#keep")

	store.SyncTargetFromURL("https://example.test/page#child")
	if got := store.TargetNodeID(); got != childID {
		t.Fatalf("TargetNodeID() after #child = %d, want %d", got, childID)
	}
	if err := store.SetTextContent(targetID, `plain & more`); err != nil {
		t.Fatalf("SetTextContent(#target) error = %v", err)
	}
	if got := store.TargetNodeID(); got != 0 {
		t.Fatalf("TargetNodeID() after SetTextContent = %d, want 0", got)
	}
	if got, want := store.TextContentForNode(targetID), `plain & more`; got != want {
		t.Fatalf("TextContentForNode(#target) after SetTextContent = %q, want %q", got, want)
	}

	store.SyncTargetFromURL("https://example.test/page#target")
	if got := store.TargetNodeID(); got != targetID {
		t.Fatalf("TargetNodeID() after re-targeting #target = %d, want %d", got, targetID)
	}
	if err := store.SetInnerHTML(targetID, `<em id="next">updated</em>`); err != nil {
		t.Fatalf("SetInnerHTML(#target) error = %v", err)
	}
	if got := store.TargetNodeID(); got != targetID {
		t.Fatalf("TargetNodeID() after SetInnerHTML on targeted node = %d, want %d", got, targetID)
	}

	store.SyncTargetFromURL("https://example.test/page#target")
	if got := store.TargetNodeID(); got != targetID {
		t.Fatalf("TargetNodeID() after #target = %d, want %d", got, targetID)
	}
	if err := store.SetOuterHTML(targetID, `<article id="next">n</article>`); err != nil {
		t.Fatalf("SetOuterHTML(#target) error = %v", err)
	}
	if got := store.TargetNodeID(); got != 0 {
		t.Fatalf("TargetNodeID() after SetOuterHTML = %d, want 0", got)
	}

	store.SyncTargetFromURL("https://example.test/page#keep")
	if got := store.TargetNodeID(); got != keepID {
		t.Fatalf("TargetNodeID() after #keep = %d, want %d", got, keepID)
	}
	if err := store.RemoveNode(keepID); err != nil {
		t.Fatalf("RemoveNode(#keep) error = %v", err)
	}
	if got := store.TargetNodeID(); got != 0 {
		t.Fatalf("TargetNodeID() after RemoveNode = %d, want 0", got)
	}
}

func TestCloneNodeDeepAndShallow(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root"><p id="p" class="copy"><span>text</span></p></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")
	pID := mustSelectSingle(t, store, "#p")

	deepCloneID, err := store.CloneNode(pID, true)
	if err != nil {
		t.Fatalf("CloneNode(deep) error = %v", err)
	}
	if deepCloneID == pID {
		t.Fatalf("CloneNode(deep) returned source node id")
	}
	store.appendChild(rootID, deepCloneID)

	shallowCloneID, err := store.CloneNode(pID, false)
	if err != nil {
		t.Fatalf("CloneNode(shallow) error = %v", err)
	}
	store.appendChild(rootID, shallowCloneID)

	deepOuter, err := store.OuterHTMLForNode(deepCloneID)
	if err != nil {
		t.Fatalf("OuterHTMLForNode(deepCloneID) error = %v", err)
	}
	if got, want := deepOuter, `<p id="p" class="copy"><span>text</span></p>`; got != want {
		t.Fatalf("OuterHTMLForNode(deepCloneID) = %q, want %q", got, want)
	}

	shallowOuter, err := store.OuterHTMLForNode(shallowCloneID)
	if err != nil {
		t.Fatalf("OuterHTMLForNode(shallowCloneID) error = %v", err)
	}
	if got, want := shallowOuter, `<p id="p" class="copy"></p>`; got != want {
		t.Fatalf("OuterHTMLForNode(shallowCloneID) = %q, want %q", got, want)
	}
}

func TestCloneNodePreservesUserValidity(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<form id="profile"><input id="name" type="text" required value="Ada"></form>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	nameID := mustSelectSingle(t, store, "#name")
	if err := store.SetUserValidity(nameID, true); err != nil {
		t.Fatalf("SetUserValidity(#name) error = %v", err)
	}

	deepCloneID, err := store.CloneNode(nameID, true)
	if err != nil {
		t.Fatalf("CloneNode(deep) error = %v", err)
	}
	if node := store.Node(deepCloneID); node == nil || !node.UserValidity {
		t.Fatalf("CloneNode(deep) UserValidity = %v, want true", node)
	}

	shallowCloneID, err := store.CloneNode(nameID, false)
	if err != nil {
		t.Fatalf("CloneNode(shallow) error = %v", err)
	}
	if node := store.Node(shallowCloneID); node == nil || !node.UserValidity {
		t.Fatalf("CloneNode(shallow) UserValidity = %v, want true", node)
	}
}

func TestCloneNodeAfterInsertsCloneAfterSource(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<main><div id="source"><span id="child">text</span></div><p id="tail">tail</p></main>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	sourceID := mustSelectSingle(t, store, "#source")
	cloneID, err := store.CloneNodeAfter(sourceID, true)
	if err != nil {
		t.Fatalf("CloneNodeAfter(#source) error = %v", err)
	}
	if cloneID == sourceID {
		t.Fatalf("CloneNodeAfter(#source) returned source node id")
	}

	if got, want := store.DumpDOM(), `<main><div id="source"><span id="child">text</span></div><div id="source"><span id="child">text</span></div><p id="tail">tail</p></main>`; got != want {
		t.Fatalf("DumpDOM() after CloneNodeAfter = %q, want %q", got, want)
	}
	if got, err := store.OuterHTMLForNode(cloneID); err != nil {
		t.Fatalf("OuterHTMLForNode(cloneID) error = %v", err)
	} else if want := `<div id="source"><span id="child">text</span></div>`; got != want {
		t.Fatalf("OuterHTMLForNode(cloneID) = %q, want %q", got, want)
	}
	if ids, err := store.Select("main > div + div"); err != nil || len(ids) != 1 || ids[0] != cloneID {
		t.Fatalf("Select(main > div + div) = (%v, %v), want one clone match", ids, err)
	}
}

func TestAppendChildInsertBeforeAndRemoveChild(t *testing.T) {
	store := NewStore()

	mainID, err := store.CreateElement("main")
	if err != nil {
		t.Fatalf("CreateElement(main) error = %v", err)
	}
	spanID, err := store.CreateElement("span")
	if err != nil {
		t.Fatalf("CreateElement(span) error = %v", err)
	}
	emID, err := store.CreateElement("em")
	if err != nil {
		t.Fatalf("CreateElement(em) error = %v", err)
	}
	strongID, err := store.CreateElement("strong")
	if err != nil {
		t.Fatalf("CreateElement(strong) error = %v", err)
	}

	if err := store.AppendChild(store.DocumentID(), mainID); err != nil {
		t.Fatalf("AppendChild(document, main) error = %v", err)
	}
	if err := store.AppendChild(mainID, spanID); err != nil {
		t.Fatalf("AppendChild(main, span) error = %v", err)
	}
	if err := store.AppendChild(mainID, emID); err != nil {
		t.Fatalf("AppendChild(main, em) error = %v", err)
	}
	if err := store.AppendChild(mainID, strongID); err != nil {
		t.Fatalf("AppendChild(main, strong) error = %v", err)
	}
	if err := store.InsertBefore(mainID, spanID, strongID); err != nil {
		t.Fatalf("InsertBefore(main, span, strong) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<main><em></em><span></span><strong></strong></main>`; got != want {
		t.Fatalf("DumpDOM() after InsertBefore = %q, want %q", got, want)
	}

	if err := store.SetFocusedNode(spanID); err != nil {
		t.Fatalf("SetFocusedNode(span) error = %v", err)
	}
	if err := store.SetTargetNode(spanID); err != nil {
		t.Fatalf("SetTargetNode(span) error = %v", err)
	}
	if err := store.RemoveChild(mainID, spanID); err != nil {
		t.Fatalf("RemoveChild(main, span) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<main><em></em><strong></strong></main>`; got != want {
		t.Fatalf("DumpDOM() after RemoveChild = %q, want %q", got, want)
	}
	if got := store.FocusedNodeID(); got != 0 {
		t.Fatalf("FocusedNodeID() after RemoveChild = %d, want 0", got)
	}
	if got := store.TargetNodeID(); got != 0 {
		t.Fatalf("TargetNodeID() after RemoveChild = %d, want 0", got)
	}
	if node := store.Node(spanID); node == nil || node.Parent != 0 {
		t.Fatalf("Node(span) after RemoveChild = %#v, want detached node", node)
	}

	if err := store.AppendChild(mainID, spanID); err != nil {
		t.Fatalf("AppendChild(main, detached span) error = %v", err)
	}
	if got, want := store.DumpDOM(), `<main><em></em><strong></strong><span></span></main>`; got != want {
		t.Fatalf("DumpDOM() after re-AppendChild = %q, want %q", got, want)
	}
	if err := store.AppendChild(spanID, mainID); err == nil {
		t.Fatalf("AppendChild(span, main) error = nil, want cycle error")
	}
}

func TestCreateTextNodeReplaceChildAndInsertAdjacentNodeHelpers(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="wrap"><div id="target"><span id="keep">keep</span></div><p id="tail">tail</p></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	wrapID := mustSelectSingle(t, store, "#wrap")
	targetID := mustSelectSingle(t, store, "#target")
	keepID := mustSelectSingle(t, store, "#keep")
	tailID := mustSelectSingle(t, store, "#tail")

	if err := store.SetFocusedNode(keepID); err != nil {
		t.Fatalf("SetFocusedNode(#keep) error = %v", err)
	}
	if err := store.SetTargetNode(keepID); err != nil {
		t.Fatalf("SetTargetNode(#keep) error = %v", err)
	}

	textID, err := store.CreateTextNode("seed")
	if err != nil {
		t.Fatalf("CreateTextNode(seed) error = %v", err)
	}
	if node := store.Node(textID); node == nil || node.Kind != NodeKindText || node.Text != "seed" {
		t.Fatalf("CreateTextNode(seed) node = %#v, want text node", node)
	}

	if err := store.ReplaceChild(targetID, textID, keepID); err != nil {
		t.Fatalf("ReplaceChild(target, seed, keep) error = %v", err)
	}
	if got, want := store.DumpDOM(), `<section id="wrap"><div id="target">seed</div><p id="tail">tail</p></section>`; got != want {
		t.Fatalf("DumpDOM() after ReplaceChild = %q, want %q", got, want)
	}
	if got := store.FocusedNodeID(); got != 0 {
		t.Fatalf("FocusedNodeID() after ReplaceChild = %d, want 0", got)
	}
	if got := store.TargetNodeID(); got != 0 {
		t.Fatalf("TargetNodeID() after ReplaceChild = %d, want 0", got)
	}
	if ids, err := store.Select("#keep"); err != nil || len(ids) != 0 {
		t.Fatalf("Select(#keep) after ReplaceChild = (%v, %v), want no matches", ids, err)
	}

	emID, err := store.CreateElement("em")
	if err != nil {
		t.Fatalf("CreateElement(em) error = %v", err)
	}
	if err := store.InsertAdjacentElement(targetID, "afterbegin", emID); err != nil {
		t.Fatalf("InsertAdjacentElement(afterbegin) error = %v", err)
	}
	if textID, err := store.InsertAdjacentText(targetID, "beforeend", " tail"); err != nil {
		t.Fatalf("InsertAdjacentText(beforeend) error = %v", err)
	} else if node := store.Node(textID); node == nil || node.Kind != NodeKindText || node.Text != " tail" {
		t.Fatalf("InsertAdjacentText(beforeend) node = %#v, want text node", node)
	}
	strongID, err := store.CreateElement("strong")
	if err != nil {
		t.Fatalf("CreateElement(strong) error = %v", err)
	}
	if err := store.InsertAdjacentElement(targetID, "beforebegin", strongID); err != nil {
		t.Fatalf("InsertAdjacentElement(beforebegin) error = %v", err)
	}
	if bangID, err := store.InsertAdjacentText(targetID, "afterend", "!"); err != nil {
		t.Fatalf("InsertAdjacentText(afterend) error = %v", err)
	} else if node := store.Node(bangID); node == nil || node.Kind != NodeKindText || node.Text != "!" {
		t.Fatalf("InsertAdjacentText(afterend) node = %#v, want text node", node)
	}

	if got, want := store.DumpDOM(), `<section id="wrap"><strong></strong><div id="target"><em></em>seed tail</div>!<p id="tail">tail</p></section>`; got != want {
		t.Fatalf("DumpDOM() after node construction helpers = %q, want %q", got, want)
	}
	if got, err := store.OuterHTMLForNode(wrapID); err != nil {
		t.Fatalf("OuterHTMLForNode(#wrap) error = %v", err)
	} else if want := `<section id="wrap"><strong></strong><div id="target"><em></em>seed tail</div>!<p id="tail">tail</p></section>`; got != want {
		t.Fatalf("OuterHTMLForNode(#wrap) = %q, want %q", got, want)
	}
	if node := store.Node(tailID); node == nil || node.Parent != wrapID {
		t.Fatalf("Node(#tail) after insertAdjacent helpers = %#v, want still attached to wrap", node)
	}
}

func TestInsertNodeListBeforeAfterAndReplaceNodeWithChildren(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<section id="root"><span id="a">A</span><span id="b">B</span><span id="c">C</span></section>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")
	aID := mustSelectSingle(t, store, "#a")
	bID := mustSelectSingle(t, store, "#b")
	cID := mustSelectSingle(t, store, "#c")

	beforeTextID, err := store.CreateTextNode("x")
	if err != nil {
		t.Fatalf("CreateTextNode(x) error = %v", err)
	}
	if err := store.InsertNodeListBefore(bID, []NodeID{beforeTextID, cID}); err != nil {
		t.Fatalf("InsertNodeListBefore(#b, x, #c) error = %v", err)
	}
	afterTextID, err := store.CreateTextNode("y")
	if err != nil {
		t.Fatalf("CreateTextNode(y) error = %v", err)
	}
	if err := store.InsertNodeListAfter(aID, []NodeID{afterTextID}); err != nil {
		t.Fatalf("InsertNodeListAfter(#a, y) error = %v", err)
	}
	if err := store.ReplaceNodeWithChildren(bID, nil); err != nil {
		t.Fatalf("ReplaceNodeWithChildren(#b, nil) error = %v", err)
	}

	if got, want := store.DumpDOM(), `<section id="root"><span id="a">A</span>yx<span id="c">C</span></section>`; got != want {
		t.Fatalf("DumpDOM() after node list insertion helpers = %q, want %q", got, want)
	}
	if node := store.Node(beforeTextID); node == nil || node.Parent != rootID {
		t.Fatalf("Node(x) after InsertNodeListBefore = %#v, want attached to root", node)
	}
	if node := store.Node(afterTextID); node == nil || node.Parent != rootID {
		t.Fatalf("Node(y) after InsertNodeListAfter = %#v, want attached to root", node)
	}
	if node := store.Node(cID); node == nil || node.Parent != rootID {
		t.Fatalf("Node(#c) after moving before #b = %#v, want attached to root", node)
	}
}

func TestMutationHelpersRejectInvalidInputs(t *testing.T) {
	var nilStore *Store
	if _, err := nilStore.InnerHTMLForNode(1); err == nil {
		t.Fatalf("nil InnerHTMLForNode() error = nil, want dom store error")
	}
	if err := nilStore.SetInnerHTML(1, "<p>x</p>"); err == nil {
		t.Fatalf("nil SetInnerHTML() error = nil, want dom store error")
	}
	if err := nilStore.SetTextContent(1, "x"); err == nil {
		t.Fatalf("nil SetTextContent() error = nil, want dom store error")
	}
	if err := nilStore.SetOuterHTML(1, "<p>x</p>"); err == nil {
		t.Fatalf("nil SetOuterHTML() error = nil, want dom store error")
	}
	if err := nilStore.InsertAdjacentHTML(1, "beforeend", "<p>x</p>"); err == nil {
		t.Fatalf("nil InsertAdjacentHTML() error = nil, want dom store error")
	}
	if _, err := nilStore.CreateTextNode("x"); err == nil {
		t.Fatalf("nil CreateTextNode() error = nil, want dom store error")
	}
	if err := nilStore.ReplaceChild(1, 2, 3); err == nil {
		t.Fatalf("nil ReplaceChild() error = nil, want dom store error")
	}
	if err := nilStore.InsertAdjacentElement(1, "beforeend", 2); err == nil {
		t.Fatalf("nil InsertAdjacentElement() error = nil, want dom store error")
	}
	if _, err := nilStore.InsertAdjacentText(1, "beforeend", "x"); err == nil {
		t.Fatalf("nil InsertAdjacentText() error = nil, want dom store error")
	}
	if err := nilStore.RemoveNode(1); err == nil {
		t.Fatalf("nil RemoveNode() error = nil, want dom store error")
	}
	if _, err := nilStore.CloneNode(1, true); err == nil {
		t.Fatalf("nil CloneNode() error = nil, want dom store error")
	}
	if _, err := nilStore.CloneNodeAfter(1, true); err == nil {
		t.Fatalf("nil CloneNodeAfter() error = nil, want dom store error")
	}
	if err := nilStore.DeleteNode(1); err == nil {
		t.Fatalf("nil DeleteNode() error = nil, want dom store error")
	}
	if err := nilStore.InsertNodeListBefore(1, []NodeID{2}); err == nil {
		t.Fatalf("nil InsertNodeListBefore() error = nil, want dom store error")
	}
	if err := nilStore.InsertNodeListAfter(1, []NodeID{2}); err == nil {
		t.Fatalf("nil InsertNodeListAfter() error = nil, want dom store error")
	}
	if err := nilStore.ReplaceNodeWithChildren(1, []NodeID{2}); err == nil {
		t.Fatalf("nil ReplaceNodeWithChildren() error = nil, want dom store error")
	}

	store := NewStore()
	if err := store.BootstrapHTML(`<div id="target">text</div><p id="sibling">tail</p>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	targetID := mustSelectSingle(t, store, "#target")
	textNode := store.Node(targetID).Children[0]

	if _, err := store.InnerHTMLForNode(999); err == nil {
		t.Fatalf("InnerHTMLForNode(invalid) error = nil, want invalid node error")
	}
	if got := store.TextContentForNode(999); got != "" {
		t.Fatalf("TextContentForNode(invalid) = %q, want empty string", got)
	}
	if err := store.SetInnerHTML(999, "<p>x</p>"); err == nil {
		t.Fatalf("SetInnerHTML(invalid) error = nil, want invalid node error")
	}
	if err := store.SetTextContent(999, "x"); err == nil {
		t.Fatalf("SetTextContent(invalid) error = nil, want invalid node error")
	}
	if err := store.SetOuterHTML(999, "<p>x</p>"); err == nil {
		t.Fatalf("SetOuterHTML(invalid) error = nil, want invalid node error")
	}
	if err := store.InsertAdjacentHTML(999, "beforeend", "<p>x</p>"); err == nil {
		t.Fatalf("InsertAdjacentHTML(invalid) error = nil, want invalid node error")
	}
	if err := store.RemoveNode(999); err == nil {
		t.Fatalf("RemoveNode(invalid) error = nil, want invalid node error")
	}
	if err := store.RemoveNode(store.DocumentID()); err == nil {
		t.Fatalf("RemoveNode(document) error = nil, want document removal error")
	}
	if _, err := store.CloneNode(999, true); err == nil {
		t.Fatalf("CloneNode(invalid) error = nil, want invalid node error")
	}
	if _, err := store.CloneNodeAfter(999, true); err == nil {
		t.Fatalf("CloneNodeAfter(invalid) error = nil, want invalid node error")
	}

	if _, err := store.InnerHTMLForNode(textNode); err == nil {
		t.Fatalf("InnerHTMLForNode(text) error = nil, want non-element error")
	}
	if err := store.SetTextContent(textNode, "x"); err != nil {
		t.Fatalf("SetTextContent(text) error = %v", err)
	}
	if got, want := store.TextContentForNode(targetID), "x"; got != want {
		t.Fatalf("TextContentForNode(#target) after SetTextContent(text) = %q, want %q", got, want)
	}
	if err := store.SetInnerHTML(textNode, "<p>x</p>"); err == nil {
		t.Fatalf("SetInnerHTML(text) error = nil, want non-element error")
	}
	if err := store.SetOuterHTML(textNode, "<p>x</p>"); err == nil {
		t.Fatalf("SetOuterHTML(text) error = nil, want non-element error")
	}
	if err := store.InsertAdjacentHTML(textNode, "beforeend", "<p>x</p>"); err == nil {
		t.Fatalf("InsertAdjacentHTML(text) error = nil, want non-element error")
	}
	if err := store.InsertAdjacentElement(textNode, "beforeend", targetID); err == nil {
		t.Fatalf("InsertAdjacentElement(text) error = nil, want non-element error")
	}
	if _, err := store.InsertAdjacentText(textNode, "beforeend", "x"); err == nil {
		t.Fatalf("InsertAdjacentText(text) error = nil, want non-element error")
	}

	beforeCount := store.NodeCount()
	if err := store.InsertAdjacentHTML(targetID, "sideways", "<p>x</p>"); err == nil {
		t.Fatalf("InsertAdjacentHTML(invalid position) error = nil, want invalid position error")
	}
	if got, want := store.NodeCount(), beforeCount; got != want {
		t.Fatalf("NodeCount() after invalid InsertAdjacentHTML = %d, want %d", got, want)
	}
	if _, err := store.InsertAdjacentText(targetID, "sideways", "x"); err == nil {
		t.Fatalf("InsertAdjacentText(invalid position) error = nil, want invalid position error")
	}
	if got, want := store.NodeCount(), beforeCount; got != want {
		t.Fatalf("NodeCount() after invalid InsertAdjacentText = %d, want %d", got, want)
	}
	if err := store.InsertNodeListBefore(targetID, []NodeID{targetID}); err == nil {
		t.Fatalf("InsertNodeListBefore(self) error = nil, want self-insertion error")
	}
	if err := store.InsertNodeListAfter(targetID, []NodeID{targetID}); err == nil {
		t.Fatalf("InsertNodeListAfter(self) error = nil, want self-insertion error")
	}

	if err := store.SetOuterHTML(targetID, `<section id="new"></section>`); err == nil {
		t.Fatalf("SetOuterHTML(document child) error = nil, want document-parent error")
	}
	if err := store.InsertAdjacentHTML(targetID, "beforebegin", `<a id="bb"></a>`); err == nil {
		t.Fatalf("InsertAdjacentHTML(beforebegin on document child) error = nil, want document-parent error")
	}
	if err := store.InsertAdjacentHTML(targetID, "afterend", `<a id="ae"></a>`); err == nil {
		t.Fatalf("InsertAdjacentHTML(afterend on document child) error = nil, want document-parent error")
	}
	if err := store.ReplaceChild(targetID, store.DocumentID(), textNode); err == nil {
		t.Fatalf("ReplaceChild(target, document, text) error = nil, want cycle error")
	}
	if err := store.InsertAdjacentElement(targetID, "sideways", store.DocumentID()); err == nil {
		t.Fatalf("InsertAdjacentElement(invalid position) error = nil, want invalid position error")
	}
	if _, err := store.CloneNodeAfter(store.DocumentID(), true); err == nil {
		t.Fatalf("CloneNodeAfter(document) error = nil, want document clone error")
	}
}
