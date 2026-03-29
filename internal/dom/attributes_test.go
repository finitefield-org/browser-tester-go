package dom

import (
	"strings"
	"testing"
)

func TestAttributeReflectionRoundTrip(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root" data-x="1"></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")

	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || !ok || got != "1" {
		t.Fatalf("GetAttribute(data-x) = (%q, %v, %v), want (\"1\", true, nil)", got, ok, err)
	}

	if ok, err := store.HasAttribute(rootID, "data-x"); err != nil || !ok {
		t.Fatalf("HasAttribute(data-x) = (%v, %v), want (true, nil)", ok, err)
	}

	if err := store.SetAttribute(rootID, "data-x", "2"); err != nil {
		t.Fatalf("SetAttribute(data-x) error = %v", err)
	}
	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || !ok || got != "2" {
		t.Fatalf("GetAttribute(data-x) after SetAttribute = (%q, %v, %v), want (\"2\", true, nil)", got, ok, err)
	}

	if err := store.RemoveAttribute(rootID, "data-x"); err != nil {
		t.Fatalf("RemoveAttribute(data-x) error = %v", err)
	}
	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || ok || got != "" {
		t.Fatalf("GetAttribute(data-x) after RemoveAttribute = (%q, %v, %v), want (\"\", false, nil)", got, ok, err)
	}
	if ok, err := store.HasAttribute(rootID, "data-x"); err != nil || ok {
		t.Fatalf("HasAttribute(data-x) after RemoveAttribute = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestAttributeReflectionHasAttributes(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root" data-x="1"></div><div></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")
	emptyID := mustSelectSingle(t, store, "div:last-of-type")

	if got, err := store.HasAttributes(rootID); err != nil || !got {
		t.Fatalf("HasAttributes(root) = (%v, %v), want (true, nil)", got, err)
	}
	if got, err := store.HasAttributes(emptyID); err != nil || got {
		t.Fatalf("HasAttributes(empty) = (%v, %v), want (false, nil)", got, err)
	}
}

func TestAttributeReflectionGetAttributeNames(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root" data-b="2" data-a="1"></div><div></div><p id="text">seed</p>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")
	emptyID := mustSelectSingle(t, store, "div:last-of-type")

	if got, err := store.GetAttributeNames(rootID); err != nil || strings.Join(got, "|") != "id|data-b|data-a" {
		t.Fatalf("GetAttributeNames(root) = (%v, %v), want ([id data-b data-a], nil)", got, err)
	}

	if err := store.SetAttribute(rootID, "data-c", "3"); err != nil {
		t.Fatalf("SetAttribute(data-c) error = %v", err)
	}
	if got, err := store.GetAttributeNames(rootID); err != nil || strings.Join(got, "|") != "id|data-b|data-a|data-c" {
		t.Fatalf("GetAttributeNames(root) after SetAttribute = (%v, %v), want ([id data-b data-a data-c], nil)", got, err)
	}

	if got, err := store.GetAttributeNames(emptyID); err != nil || len(got) != 0 {
		t.Fatalf("GetAttributeNames(empty) = (%v, %v), want ([], nil)", got, err)
	}
}

func TestAttributeReflectionGetAttributeNode(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root" data-b="2" data-a="1"></div><div></div><p id="text">seed</p>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")
	emptyID := mustSelectSingle(t, store, "div:last-of-type")

	if got, ok, err := store.GetAttributeNode(rootID, "data-a"); err != nil || !ok || got.Name != "data-a" || got.Value != "1" || !got.HasValue {
		t.Fatalf("GetAttributeNode(root, data-a) = (%#v, %v, %v), want ({Name:data-a Value:1 HasValue:true}, true, nil)", got, ok, err)
	}
	if got, ok, err := store.GetAttributeNode(rootID, "missing"); err != nil || ok || got.Name != "" || got.Value != "" || got.HasValue {
		t.Fatalf("GetAttributeNode(root, missing) = (%#v, %v, %v), want ({Name: Value: HasValue:false}, false, nil)", got, ok, err)
	}
	if got, ok, err := store.GetAttributeNode(emptyID, "data-a"); err != nil || ok || got.Name != "" || got.Value != "" || got.HasValue {
		t.Fatalf("GetAttributeNode(empty, data-a) = (%#v, %v, %v), want ({Name: Value: HasValue:false}, false, nil)", got, ok, err)
	}
}

func TestAttributeReflectionToggleRoundTrip(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root"></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")

	if got, err := store.ToggleAttribute(rootID, "data-x", false, false); err != nil || !got {
		t.Fatalf("ToggleAttribute(data-x) = (%v, %v), want (true, nil)", got, err)
	}
	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || !ok || got != "" {
		t.Fatalf("GetAttribute(data-x) after toggle-on = (%q, %v, %v), want (\"\", true, nil)", got, ok, err)
	}

	if got, err := store.ToggleAttribute(rootID, "data-x", false, false); err != nil || got {
		t.Fatalf("ToggleAttribute(data-x) second toggle = (%v, %v), want (false, nil)", got, err)
	}
	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || ok || got != "" {
		t.Fatalf("GetAttribute(data-x) after toggle-off = (%q, %v, %v), want (\"\", false, nil)", got, ok, err)
	}

	if got, err := store.ToggleAttribute(rootID, "data-x", true, true); err != nil || !got {
		t.Fatalf("ToggleAttribute(data-x, true) = (%v, %v), want (true, nil)", got, err)
	}
	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || !ok || got != "" {
		t.Fatalf("GetAttribute(data-x) after forced toggle-on = (%q, %v, %v), want (\"\", true, nil)", got, ok, err)
	}

	if got, err := store.ToggleAttribute(rootID, "data-x", false, true); err != nil || got {
		t.Fatalf("ToggleAttribute(data-x, false) = (%v, %v), want (false, nil)", got, err)
	}
	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || ok || got != "" {
		t.Fatalf("GetAttribute(data-x) after forced toggle-off = (%q, %v, %v), want (\"\", false, nil)", got, ok, err)
	}
}

func TestAttributeReflectionHandlesMissingAttributesAndNormalization(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root"></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "#root")

	if got, ok, err := store.GetAttribute(rootID, "missing"); err != nil || ok || got != "" {
		t.Fatalf("GetAttribute(missing) = (%q, %v, %v), want (\"\", false, nil)", got, ok, err)
	}

	// Exercise trimming + lowercase normalization.
	if err := store.SetAttribute(rootID, " DATA-X ", "1"); err != nil {
		t.Fatalf("SetAttribute(DATA-X) error = %v", err)
	}
	if got, ok, err := store.GetAttribute(rootID, "data-x"); err != nil || !ok || got != "1" {
		t.Fatalf("GetAttribute(data-x) after normalized set = (%q, %v, %v), want (\"1\", true, nil)", got, ok, err)
	}
}

func TestAttributeReflectionMutatesDOMSerializationViaSelectorLookup(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<main><div id="root"></div></main>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	rootID := mustSelectSingle(t, store, "main > div#root")

	if err := store.SetAttribute(rootID, "data-x", "1"); err != nil {
		t.Fatalf("SetAttribute(data-x) error = %v", err)
	}
	if got, want := store.DumpDOM(), `<main><div id="root" data-x="1"></div></main>`; got != want {
		t.Fatalf("DumpDOM() after SetAttribute = %q, want %q", got, want)
	}

	if err := store.RemoveAttribute(rootID, "data-x"); err != nil {
		t.Fatalf("RemoveAttribute(data-x) error = %v", err)
	}
	if got, want := store.DumpDOM(), `<main><div id="root"></div></main>`; got != want {
		t.Fatalf("DumpDOM() after RemoveAttribute = %q, want %q", got, want)
	}
}

func TestAttributeReflectionRejectsNilStoreInvalidNodeAndEmptyName(t *testing.T) {
	var nilStore *Store
	if _, _, err := nilStore.GetAttribute(1, "id"); err == nil {
		t.Fatalf("nil GetAttribute() error = nil, want dom store error")
	}
	if _, err := nilStore.HasAttribute(1, "id"); err == nil {
		t.Fatalf("nil HasAttribute() error = nil, want dom store error")
	}
	if err := nilStore.SetAttribute(1, "id", "x"); err == nil {
		t.Fatalf("nil SetAttribute() error = nil, want dom store error")
	}
	if err := nilStore.RemoveAttribute(1, "id"); err == nil {
		t.Fatalf("nil RemoveAttribute() error = nil, want dom store error")
	}
	if _, err := nilStore.ToggleAttribute(1, "id", false, false); err == nil {
		t.Fatalf("nil ToggleAttribute() error = nil, want dom store error")
	}
	if _, err := nilStore.HasAttributes(1); err == nil {
		t.Fatalf("nil HasAttributes() error = nil, want dom store error")
	}
	if _, err := nilStore.GetAttributeNames(1); err == nil {
		t.Fatalf("nil GetAttributeNames() error = nil, want dom store error")
	}
	if _, _, err := nilStore.GetAttributeNode(1, "id"); err == nil {
		t.Fatalf("nil GetAttributeNode() error = nil, want dom store error")
	}

	store := NewStore()
	if err := store.BootstrapHTML(`<div id="root"></div>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	if _, _, err := store.GetAttribute(999, "id"); err == nil {
		t.Fatalf("GetAttribute(invalid node) error = nil, want invalid node error")
	}
	if _, err := store.HasAttribute(999, "id"); err == nil {
		t.Fatalf("HasAttribute(invalid node) error = nil, want invalid node error")
	}
	if err := store.SetAttribute(999, "id", "x"); err == nil {
		t.Fatalf("SetAttribute(invalid node) error = nil, want invalid node error")
	}
	if err := store.RemoveAttribute(999, "id"); err == nil {
		t.Fatalf("RemoveAttribute(invalid node) error = nil, want invalid node error")
	}
	if _, err := store.ToggleAttribute(999, "id", false, false); err == nil {
		t.Fatalf("ToggleAttribute(invalid node) error = nil, want invalid node error")
	}
	if _, err := store.HasAttributes(999); err == nil {
		t.Fatalf("HasAttributes(invalid node) error = nil, want invalid node error")
	}
	if _, err := store.GetAttributeNames(999); err == nil {
		t.Fatalf("GetAttributeNames(invalid node) error = nil, want invalid node error")
	}
	if _, _, err := store.GetAttributeNode(999, "id"); err == nil {
		t.Fatalf("GetAttributeNode(invalid node) error = nil, want invalid node error")
	}

	rootID := mustSelectSingle(t, store, "#root")
	if _, _, err := store.GetAttribute(rootID, " "); err == nil {
		t.Fatalf("GetAttribute(empty name) error = nil, want validation error")
	}
	if _, err := store.HasAttribute(rootID, " "); err == nil {
		t.Fatalf("HasAttribute(empty name) error = nil, want validation error")
	}
	if err := store.SetAttribute(rootID, " ", "x"); err == nil {
		t.Fatalf("SetAttribute(empty name) error = nil, want validation error")
	}
	if err := store.RemoveAttribute(rootID, " "); err == nil {
		t.Fatalf("RemoveAttribute(empty name) error = nil, want validation error")
	}
	if _, err := store.ToggleAttribute(rootID, " ", false, false); err == nil {
		t.Fatalf("ToggleAttribute(empty name) error = nil, want validation error")
	}
}

func TestAttributeReflectionRejectsNonElementNodes(t *testing.T) {
	store := NewStore()

	textID := store.newNode(Node{
		Kind: NodeKindText,
		Text: "x",
	})
	store.appendChild(store.DocumentID(), textID)

	if _, _, err := store.GetAttribute(textID, "id"); err == nil {
		t.Fatalf("GetAttribute(text node) error = nil, want non-element error")
	}
	if _, err := store.HasAttribute(textID, "id"); err == nil {
		t.Fatalf("HasAttribute(text node) error = nil, want non-element error")
	}
	if err := store.SetAttribute(textID, "id", "x"); err == nil {
		t.Fatalf("SetAttribute(text node) error = nil, want non-element error")
	}
	if err := store.RemoveAttribute(textID, "id"); err == nil {
		t.Fatalf("RemoveAttribute(text node) error = nil, want non-element error")
	}
	if _, err := store.ToggleAttribute(textID, "id", false, false); err == nil {
		t.Fatalf("ToggleAttribute(text node) error = nil, want non-element error")
	}
	if _, err := store.HasAttributes(textID); err == nil {
		t.Fatalf("HasAttributes(text node) error = nil, want non-element error")
	}
	if _, err := store.GetAttributeNames(textID); err == nil {
		t.Fatalf("GetAttributeNames(text node) error = nil, want non-element error")
	}
	if _, _, err := store.GetAttributeNode(textID, "id"); err == nil {
		t.Fatalf("GetAttributeNode(text node) error = nil, want non-element error")
	}
}
