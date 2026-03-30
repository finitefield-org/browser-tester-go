package dom

import "testing"

func TestInputTypeCanonicalizesInvalidAndMissingValues(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<main><input id="missing"><input id="invalid" type="bogus"><input id="submit" type="submit"></main>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	tests := []struct {
		selector string
		want     string
	}{
		{selector: "#missing", want: "text"},
		{selector: "#invalid", want: "text"},
		{selector: "#submit", want: "submit"},
	}

	for _, tc := range tests {
		id := mustSelectSingle(t, store, tc.selector)
		if got := InputType(store.Node(id)); got != tc.want {
			t.Fatalf("InputType(%s) = %q, want %q", tc.selector, got, tc.want)
		}
	}
}

func TestSelectTypeReflectsMultipleAttribute(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<main><select id="single"><option>one</option></select><select id="multi" multiple><option>one</option></select></main>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	tests := []struct {
		selector string
		want     string
	}{
		{selector: "#single", want: "select-one"},
		{selector: "#multi", want: "select-multiple"},
	}

	for _, tc := range tests {
		id := mustSelectSingle(t, store, tc.selector)
		if got := SelectType(store.Node(id)); got != tc.want {
			t.Fatalf("SelectType(%s) = %q, want %q", tc.selector, got, tc.want)
		}
	}
}

func TestButtonTypeReflectsSubmitButtonState(t *testing.T) {
	store := NewStore()

	formID := store.newNode(Node{
		Kind:    NodeKindElement,
		TagName: "form",
		Attrs: []Attribute{
			{Name: "id", Value: "profile", HasValue: true},
		},
	})
	selectID := store.newNode(Node{
		Kind:    NodeKindElement,
		TagName: "select",
	})
	autoButtonID := store.newNode(Node{
		Kind:    NodeKindElement,
		TagName: "button",
		Attrs: []Attribute{
			{Name: "id", Value: "auto", HasValue: true},
		},
	})
	invalidButtonID := store.newNode(Node{
		Kind:    NodeKindElement,
		TagName: "button",
		Attrs: []Attribute{
			{Name: "id", Value: "invalid", HasValue: true},
			{Name: "type", Value: "menu", HasValue: true},
		},
	})

	store.appendChild(store.DocumentID(), formID)
	store.appendChild(formID, selectID)
	store.appendChild(selectID, autoButtonID)
	store.appendChild(formID, invalidButtonID)

	if got, want := ButtonType(store, store.Node(autoButtonID)), "button"; got != want {
		t.Fatalf("ButtonType(auto button inside select) = %q, want %q", got, want)
	}
	if got := IsSubmitButton(store, store.Node(autoButtonID)); got {
		t.Fatalf("IsSubmitButton(auto button inside select) = true, want false")
	}

	if got, want := ButtonType(store, store.Node(invalidButtonID)), "submit"; got != want {
		t.Fatalf("ButtonType(invalid button) = %q, want %q", got, want)
	}
	if got := IsSubmitButton(store, store.Node(invalidButtonID)); !got {
		t.Fatalf("IsSubmitButton(invalid button) = false, want true")
	}
}

func TestButtonTypeTreatsCommandButtonsAsNonSubmitButtons(t *testing.T) {
	store := NewStore()
	if err := store.BootstrapHTML(`<main><form id="profile"><button id="command" command="show-modal">Open</button><button id="commandfor" commandfor="dialog">Open</button></form></main>`); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	tests := []struct {
		selector string
	}{
		{selector: "#command"},
		{selector: "#commandfor"},
	}

	for _, tc := range tests {
		id := mustSelectSingle(t, store, tc.selector)
		if got, want := ButtonType(store, store.Node(id)), "button"; got != want {
			t.Fatalf("ButtonType(%s) = %q, want %q", tc.selector, got, want)
		}
		if got := IsSubmitButton(store, store.Node(id)); got {
			t.Fatalf("IsSubmitButton(%s) = true, want false", tc.selector)
		}
	}
}
