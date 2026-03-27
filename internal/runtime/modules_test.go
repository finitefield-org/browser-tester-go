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

func TestLoadModuleBindingsSupportsNamespaceReExports(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="seed">seed</div><script type="module" id="math">export const value = host?.["textContent"]("#seed"); export default value;</script><script type="module" id="reexport">export * as ns from "math";</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	reexport, ok := session.moduleBindings["reexport"]
	if !ok {
		t.Fatalf("moduleBindings[\"reexport\"] is missing")
	}
	if reexport.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"reexport\"].Kind = %q, want object", reexport.Kind)
	}
	ns, ok := moduleObjectProperty(reexport, "ns")
	if !ok || ns.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"reexport\"].ns = %#v, want object", ns)
	}
	if got, ok := moduleObjectProperty(ns, "default"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"reexport\"].ns.default = %#v, want string seed", got)
	}
	if got, ok := moduleObjectProperty(ns, "value"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"reexport\"].ns.value = %#v, want string seed", got)
	}
}

func TestLoadModuleBindingsSupportsDefaultSpecifierAliases(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="seed">seed</div><script type="module" id="math">export const value = host?.["textContent"]("#seed"); export default value;</script><script type="module" id="reexport">export { default as mirror } from "math"; export { value as default } from "math";</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	reexport, ok := session.moduleBindings["reexport"]
	if !ok {
		t.Fatalf("moduleBindings[\"reexport\"] is missing")
	}
	if reexport.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"reexport\"].Kind = %q, want object", reexport.Kind)
	}
	if got, ok := moduleObjectProperty(reexport, "mirror"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"reexport\"].mirror = %#v, want string seed", got)
	}
	if got, ok := moduleObjectProperty(reexport, "default"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"reexport\"].default = %#v, want string seed", got)
	}
}

func TestLoadModuleBindingsSupportsImportAttributes(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="seed">seed</div><script type="module" id="math">export const value = host?.["textContent"]("#seed"); export default value;</script><script type="module" id="consumer">import { default as seeded } from "math" with { type: "json" }; export const viaImport = seeded; export default seeded;</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	consumer, ok := session.moduleBindings["consumer"]
	if !ok {
		t.Fatalf("moduleBindings[\"consumer\"] is missing")
	}
	if consumer.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"consumer\"].Kind = %q, want object", consumer.Kind)
	}
	if got, ok := moduleObjectProperty(consumer, "viaImport"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"consumer\"].viaImport = %#v, want string seed", got)
	}
	if got, ok := moduleObjectProperty(consumer, "default"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"consumer\"].default = %#v, want string seed", got)
	}
}

func TestLoadModuleBindingsSupportsDefaultAndNamespaceImports(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="seed">seed</div><script type="module" id="math">export const value = host?.["textContent"]("#seed"); export default value;</script><script type="module" id="consumer">import seeded, * as ns from "math" with { type: "json" }; export const viaDefault = seeded; export const viaNamespace = ns.value; export default seeded;</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	consumer, ok := session.moduleBindings["consumer"]
	if !ok {
		t.Fatalf("moduleBindings[\"consumer\"] is missing")
	}
	if consumer.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"consumer\"].Kind = %q, want object", consumer.Kind)
	}
	if got, ok := moduleObjectProperty(consumer, "viaDefault"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"consumer\"].viaDefault = %#v, want string seed", got)
	}
	if got, ok := moduleObjectProperty(consumer, "viaNamespace"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"consumer\"].viaNamespace = %#v, want string seed", got)
	}
	if got, ok := moduleObjectProperty(consumer, "default"); !ok || got.Kind != script.ValueKindString || got.String != "seed" {
		t.Fatalf("moduleBindings[\"consumer\"].default = %#v, want string seed", got)
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

func TestLoadModuleBindingsSupportsDefaultGeneratorFunctionExports(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><script type="module" id="spin">export default function* read() { yield "seed"; };</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	spin, ok := session.moduleBindings["spin"]
	if !ok {
		t.Fatalf("moduleBindings[\"spin\"] is missing")
	}
	if spin.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"spin\"].Kind = %q, want object", spin.Kind)
	}
	defaultExport, ok := moduleObjectProperty(spin, "default")
	if !ok || defaultExport.Kind != script.ValueKindFunction {
		t.Fatalf("moduleBindings[\"spin\"].default = %#v, want function", defaultExport)
	}
}

func TestLoadModuleBindingsSupportsAnonymousDefaultAsyncFunctionAndGeneratorExports(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="seed">seed</div><script type="module" id="asyncFn">export default async function() { return await host?.["textContent"]("#seed"); };</script><script type="module" id="asyncGen">export default async function*() { yield await host?.["textContent"]("#seed"); };</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	asyncFn, ok := session.moduleBindings["asyncFn"]
	if !ok {
		t.Fatalf("moduleBindings[\"asyncFn\"] is missing")
	}
	if asyncFn.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"asyncFn\"].Kind = %q, want object", asyncFn.Kind)
	}
	defaultExport, ok := moduleObjectProperty(asyncFn, "default")
	if !ok || defaultExport.Kind != script.ValueKindFunction {
		t.Fatalf("moduleBindings[\"asyncFn\"].default = %#v, want function", defaultExport)
	}

	asyncGen, ok := session.moduleBindings["asyncGen"]
	if !ok {
		t.Fatalf("moduleBindings[\"asyncGen\"] is missing")
	}
	if asyncGen.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"asyncGen\"].Kind = %q, want object", asyncGen.Kind)
	}
	defaultExport, ok = moduleObjectProperty(asyncGen, "default")
	if !ok || defaultExport.Kind != script.ValueKindFunction {
		t.Fatalf("moduleBindings[\"asyncGen\"].default = %#v, want function", defaultExport)
	}
}

func TestLoadModuleBindingsSupportsImportMetaUrl(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><script type="module" id="meta">export const url = import.meta.url; export default url;</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	meta, ok := session.moduleBindings["meta"]
	if !ok {
		t.Fatalf("moduleBindings[\"meta\"] is missing")
	}
	if meta.Kind != script.ValueKindObject {
		t.Fatalf("moduleBindings[\"meta\"].Kind = %q, want object", meta.Kind)
	}
	if got, ok := moduleObjectProperty(meta, "url"); !ok || got.Kind != script.ValueKindString || got.String != "inline-module:meta" {
		t.Fatalf("moduleBindings[\"meta\"].url = %#v, want string inline-module:meta", got)
	}
	if got, ok := moduleObjectProperty(meta, "default"); !ok || got.Kind != script.ValueKindString || got.String != "inline-module:meta" {
		t.Fatalf("moduleBindings[\"meta\"].default = %#v, want string inline-module:meta", got)
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
