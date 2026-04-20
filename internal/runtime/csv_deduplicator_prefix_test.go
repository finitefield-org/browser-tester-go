package runtime

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func stripHTMLScriptBlocksAfter(html string, keep int) string {
	var out strings.Builder
	cursor := 0
	index := 0

	for {
		start := strings.Index(html[cursor:], "<script")
		if start < 0 {
			out.WriteString(html[cursor:])
			break
		}
		start += cursor
		out.WriteString(html[cursor:start])

		openEnd := strings.Index(html[start:], ">")
		if openEnd < 0 {
			out.WriteString(html[start:])
			break
		}
		openEnd += start + 1

		closeStart := strings.Index(html[openEnd:], "</script>")
		if closeStart < 0 {
			out.WriteString(html[start:])
			break
		}
		closeStart += openEnd
		closeEnd := closeStart + len("</script>")

		if index < keep {
			out.WriteString(html[start:closeEnd])
		}

		cursor = closeEnd
		index++
	}

	return out.String()
}

func TestCSVDeduplicatorBigScriptPrefixesParse(t *testing.T) {
	const htmlPath = "/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/build/en/tools/data/csv-deduplicator/index.html"
	html, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", htmlPath, err)
	}

	targetSource := func() string {
		session := NewSession(SessionConfig{HTML: string(html)})
		session.Registry().ExternalJS().RespondSource("https://unpkg.com/lucide@latest", `var lucide = globalThis.lucide = window.lucide = { createIcons: function () {} };`)

		store := dom.NewStore()
		if err := store.BootstrapHTML(string(html)); err != nil {
			t.Fatalf("BootstrapHTML() error = %v", err)
		}

		nodes := store.Nodes()
		for _, node := range nodes {
			if node == nil || node.Kind != dom.NodeKindElement || node.TagName != "script" {
				continue
			}
			source, _, err := session.resolveScriptSource(store, node.ID)
			if err != nil {
				t.Fatalf("resolveScriptSource(%d) error = %v", node.ID, err)
			}
			typeAttr, ok, err := store.GetAttribute(node.ID, "type")
			if err != nil {
				t.Fatalf("GetAttribute(type) for script %d error = %v", node.ID, err)
			}
			if ok && !isClassicInlineScriptType(typeAttr) {
				continue
			}
			if store.Node(node.ID) == nil {
				continue
			}
			if strings.TrimSpace(source) == "" {
				continue
			}
			if strings.Contains(source, "const pageRaw =") {
				return source
			}
		}
		t.Fatal("could not locate the big deduplicator script")
		return ""
	}

	source := targetSource()
	if source == "" {
		t.Fatal("targetSource() returned empty source")
	}

	trimmed := strings.TrimSpace(source)
	trimmed = strings.TrimPrefix(trimmed, "(() => {")
	trimmed = strings.TrimSuffix(trimmed, "})();")
	body := strings.TrimSpace(trimmed)
	if body == "" {
		t.Fatal("extracted big script body is empty")
	}

	statements, err := script.SplitScriptStatementsForRuntime(body)
	if err != nil {
		t.Fatalf("SplitScriptStatementsForRuntime() error = %v", err)
	}
	if len(statements) == 0 {
		t.Fatal("could not split big deduplicator script body into statements")
	}
	if len(statements) > 30 {
		tailStatements, err := script.SplitScriptStatementsForRuntime(statements[30])
		if err != nil {
			t.Fatalf("SplitScriptStatementsForRuntime(stmt 31 tail) error = %v", err)
		}
		t.Logf("stmt 31 tail split count = %d", len(tailStatements))
		for j := 0; j < len(tailStatements) && j < 3; j++ {
			t.Logf("tail stmt %d: %s", j+1, strings.TrimSpace(tailStatements[j]))
		}
		tail := statements[30]
		for _, marker := range []string{
			"function findMismatch(",
			"function parseInputText(",
			"function restoreSelectedKeysAfterParse(",
			"function renderInputPreview(",
			"function renderKeyList(",
			"function renderKeyChips(",
			"function deduplicateRows(",
			"function renderResult(",
			"function renderTopKeys(",
			"function renderConflicts(",
			"function syncTabButtons(",
			"function renderResultTable(",
			"function renderTable(",
			"function toCsv(",
			"function decodeFileToText(",
			"function bindEvents(",
			"function init(",
		} {
			idx := strings.Index(tail, marker)
			if idx < 0 {
				t.Logf("tail marker missing: %s", marker)
				continue
			}
			prefix := strings.TrimSpace(tail[:idx])
			prefixStatements, err := script.SplitScriptStatementsForRuntime(prefix)
			if err != nil {
				t.Fatalf("SplitScriptStatementsForRuntime(prefix before %s) error = %v", marker, err)
			}
			t.Logf("tail prefix before %s => %d statements", marker, len(prefixStatements))
		}
	}

	checkPrefix := func(label string, script string) {
		t.Helper()
		session := NewSession(SessionConfig{HTML: `<main><div id="out"></div><script>(() => {` + script + `})();</script></main>`})
		session.Registry().ExternalJS().RespondSource("https://unpkg.com/lucide@latest", `var lucide = globalThis.lucide = window.lucide = { createIcons: function () {} };`)
		if _, err := session.ensureDOM(); err != nil {
			t.Fatalf("%s: ensureDOM() error = %v", label, err)
		}
	}

	for i := 0; i < len(statements); i++ {
		if i >= 28 && i <= 31 {
			t.Logf("stmt %d: %s", i+1, strings.TrimSpace(statements[i]))
		}
		prefix := strings.Join(statements[:i+1], ";\n") + ";"
		label := strings.TrimSpace(statements[i])
		if len(label) > 60 {
			label = label[:60]
		}
		checkPrefix(fmt.Sprintf("through stmt %d: %s", i+1, label), prefix)
	}
}
