package runtime

import (
	"strings"
	"testing"

	"browsertester/internal/script"
)

func moduleObjectProperty(value script.Value, key string) (script.Value, bool) {
	for _, entry := range value.Object {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return script.Value{}, false
}

func TestLoadModuleBindingsSupportsInlineModuleImportsAndReExports(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="seed">seed</div><script type="module" id="math">export const value = host?.["textContent"]("#seed"); export default value;</script><script type="module" id="reexport">export { value as alias } from "math";</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	math, ok := session.moduleBindings["math"]
	if !ok {
		t.Fatalf("moduleBindings[\"math\"] is missing")
	}
	if math.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"math\"].Kind = %q, want object", math.Kind)
	}
	if got, ok := moduleObjectProperty(math, "value"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"math\"].value = %#v, want string seed", got)
	}
	if got, ok := moduleObjectProperty(math, "default"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"math\"].default = %#v, want string seed", got)
	}

	reexport, ok := session.moduleBindings["reexport"]
	if !ok {
		t.Fatalf("moduleBindings[\"reexport\"] is missing")
	}
	if reexport.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"reexport\"].Kind = %q, want object", reexport.Kind)
	}
	if got, ok := moduleObjectProperty(reexport, "alias"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"reexport\"].alias = %#v, want string seed", got)
	}
}

func TestLoadModuleBindingsSupportsDefaultClassExports(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="seed">seed</div><script type="module" id="box">export default class Box { static value = host?.["textContent"]("#seed"); };</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	box, ok := session.moduleBindings["box"]
	if !ok {
		t.Fatalf("moduleBindings[\"box\"] is missing")
	}
	if box.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"box\"].Kind = %q, want object", box.Kind)
	}
	defaultExport, ok := moduleObjectProperty(box, "default")
	if !ok || defaultExport.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"box\"].default = %#v, want object", defaultExport)
	}
	if got, ok := moduleObjectProperty(defaultExport, "value"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"box\"].default.value = %#v, want string seed", got)
	}
}

func TestLoadModuleBindingsRejectsCyclicInlineModuleDependencies(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><script type="module" id="a">export { alias } from "b";</script><script type="module" id="b">export { alias } from "a";</script></main>`,
	})

	_, err := session.ensureDOM()
	if err == nil {
		t.Fatalf("ensureDOM() error = nil, want cyclic dependency error")
	}
	if !strings.Contains(err.Error(), "cyclic module dependency") {
		t.Fatalf("ensureDOM() error = %v, want cyclic dependency error", err)
	}
}
