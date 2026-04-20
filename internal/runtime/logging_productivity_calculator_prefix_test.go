package runtime

import (
	"os"
	"strings"
	"testing"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func TestLoggingProductivityCalculatorScriptsParseWithoutBoot(t *testing.T) {
	const htmlPath = "/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/build/en/tools/forestry/logging-productivity-calculator/index.html"

	html, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", htmlPath, err)
	}

	store := dom.NewStore()
	if err := store.BootstrapHTML(string(html)); err != nil {
		t.Fatalf("BootstrapHTML() error = %v", err)
	}

	session := NewSession(SessionConfig{
		HTML: string(html),
		URL:  "https://finitefield.org/tools/forestry/logging-productivity-calculator/",
	})
	session.Registry().ExternalJS().RespondSource("https://unpkg.com/lucide@latest", `var lucide = globalThis.lucide = window.lucide = { createIcons: function () {} };`)

	var source string
	nodes := store.Nodes()
	for _, node := range nodes {
		if node == nil || node.Kind != dom.NodeKindElement || node.TagName != "script" {
			continue
		}
		typeAttr, ok, err := store.GetAttribute(node.ID, "type")
		if err != nil {
			t.Fatalf("GetAttribute(type) for script %d error = %v", node.ID, err)
		}
		if ok && !isClassicInlineScriptType(typeAttr) {
			continue
		}
		scriptSource, _, err := session.resolveScriptSource(store, node.ID)
		if err != nil {
			t.Fatalf("resolveScriptSource(%d) error = %v", node.ID, err)
		}
		if strings.TrimSpace(scriptSource) == "" {
			continue
		}
		if len(scriptSource) > len(source) {
			source = scriptSource
		}
	}

	if source == "" {
		t.Fatal("could not locate the logging-productivity-calculator inline script")
	}

	trimmed := strings.TrimSpace(source)
	trimmed = strings.TrimPrefix(trimmed, "(() => {")
	trimmed = strings.TrimSuffix(trimmed, "})();")
	body := strings.TrimSpace(trimmed)
	if body == "" {
		t.Fatal("extracted logging-productivity-calculator script body is empty")
	}

	statements, err := script.SplitScriptStatementsForRuntime(body)
	if err != nil {
		t.Fatalf("SplitScriptStatementsForRuntime() error = %v", err)
	}
	if len(statements) == 0 {
		t.Fatal("could not split logging-productivity-calculator script body into statements")
	}

	prefix := strings.Join(statements[:len(statements)-1], ";\n") + ";"
	testHTML := strings.Replace(string(html), source, prefix, 1)
	s := NewSession(SessionConfig{HTML: testHTML})
	s.Registry().ExternalJS().RespondSource("https://unpkg.com/lucide@latest", `var lucide = globalThis.lucide = window.lucide = { createIcons: function () {} };`)
	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() without boot error = %v", err)
	}
}
