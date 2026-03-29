package runtime

import (
	"fmt"
	"sort"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func (s *Session) loadModuleBindings(store *dom.Store) error {
	if s == nil || store == nil {
		return nil
	}

	moduleSources := make(map[string]string)
	nodes := store.Nodes()
	for _, node := range nodes {
		if node == nil || node.Kind != dom.NodeKindElement || node.TagName != "script" {
			continue
		}
		typeValue, ok, err := store.GetAttribute(node.ID, "type")
		if err != nil {
			return err
		}
		if !ok || !strings.EqualFold(strings.TrimSpace(typeValue), "module") {
			continue
		}
		idValue, ok, err := store.GetAttribute(node.ID, "id")
		if err != nil {
			return err
		}
		idValue = strings.TrimSpace(idValue)
		if !ok || idValue == "" {
			return fmt.Errorf("module scripts require an id in this bounded classic-JS slice")
		}
		if _, exists := moduleSources[idValue]; exists {
			return fmt.Errorf("duplicate module id %q in this bounded classic-JS slice", idValue)
		}
		source, _, err := s.resolveScriptSource(store, node.ID)
		if err != nil {
			return err
		}
		moduleSources[idValue] = source
	}

	if len(moduleSources) == 0 {
		s.moduleBindings = map[string]script.Value{}
		return nil
	}

	loaded := make(map[string]script.Value, len(moduleSources))
	visiting := make(map[string]bool, len(moduleSources))
	loadModule := func(id string) (script.Value, error) {
		return s.loadInlineModule(id, moduleSources, loaded, visiting, store)
	}

	for id := range moduleSources {
		if _, err := loadModule(id); err != nil {
			return err
		}
	}

	s.moduleBindings = loaded
	return nil
}

func (s *Session) loadInlineModule(id string, moduleSources map[string]string, loaded map[string]script.Value, visiting map[string]bool, store *dom.Store) (script.Value, error) {
	if value, ok := loaded[id]; ok {
		return value, nil
	}
	if visiting[id] {
		return script.UndefinedValue(), fmt.Errorf("cyclic module dependency involving %q is not supported in this bounded classic-JS slice", id)
	}
	source, ok := moduleSources[id]
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("module %q is not available in this bounded classic-JS slice", id)
	}

	visiting[id] = true
	defer delete(visiting, id)

	dependencies, err := extractInlineModuleDependencies(source)
	if err != nil {
		return script.UndefinedValue(), err
	}

	bindings := make(map[string]script.Value, len(dependencies))
	for _, dep := range dependencies {
		depValue, err := s.loadInlineModule(dep, moduleSources, loaded, visiting, store)
		if err != nil {
			return script.UndefinedValue(), err
		}
		bindings[dep] = depValue
	}
	bindings[script.ClassicJSModuleMetaURLBindingName] = script.StringValue("inline-module:" + id)

	moduleExports := map[string]script.Value{}
	runtime := script.NewRuntimeWithBindings(&inlineScriptHost{session: s, store: store}, browserGlobalBindings(s, store))
	result, err := runtime.Dispatch(script.DispatchRequest{
		Source:        source,
		Bindings:      bindings,
		ModuleExports: moduleExports,
	})
	if err != nil {
		return script.UndefinedValue(), err
	}

	entries := make([]script.ObjectEntry, 0, len(moduleExports))
	exportNames := make([]string, 0, len(moduleExports))
	for name := range moduleExports {
		exportNames = append(exportNames, name)
	}
	sort.Strings(exportNames)
	for _, name := range exportNames {
		entries = append(entries, script.ObjectEntry{Key: name, Value: moduleExports[name]})
	}
	if len(entries) == 0 {
		// Preserve the module result so modules without exports can still be imported as empty namespaces.
		entries = []script.ObjectEntry{}
	}
	namespace := script.ObjectValue(entries)
	loaded[id] = namespace
	_ = result
	return namespace, nil
}

func extractInlineModuleDependencies(source string) ([]string, error) {
	statements, err := script.SplitScriptStatementsForRuntime(source)
	if err != nil {
		return nil, err
	}

	deps := make([]string, 0, 4)
	seen := map[string]struct{}{}
	for _, statement := range statements {
		specifier, ok, err := extractInlineModuleSpecifier(statement)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if _, exists := seen[specifier]; exists {
			continue
		}
		seen[specifier] = struct{}{}
		deps = append(deps, specifier)
	}
	return deps, nil
}

func extractInlineModuleSpecifier(statement string) (string, bool, error) {
	trimmed := strings.TrimSpace(statement)
	if trimmed == "" {
		return "", false, nil
	}
	if hasModuleKeywordPrefix(trimmed, "import") {
		remainder := strings.TrimSpace(trimmed[len("import"):])
		if remainder == "" {
			return "", false, nil
		}
		if remainder[0] == '\'' || remainder[0] == '"' {
			return parseQuotedModuleSpecifier(remainder)
		}
		if idx := findTopLevelModuleKeyword(remainder, "from"); idx >= 0 {
			return parseQuotedModuleSpecifier(remainder[idx+len("from"):])
		}
		return "", false, nil
	}
	if hasModuleKeywordPrefix(trimmed, "export") {
		remainder := strings.TrimSpace(trimmed[len("export"):])
		if remainder == "" {
			return "", false, nil
		}
		if remainder[0] != '{' && remainder[0] != '*' {
			return "", false, nil
		}
		if idx := findTopLevelModuleKeyword(remainder, "from"); idx >= 0 {
			return parseQuotedModuleSpecifier(remainder[idx+len("from"):])
		}
	}
	return "", false, nil
}

func hasModuleKeywordPrefix(source string, keyword string) bool {
	if len(source) < len(keyword) {
		return false
	}
	if source[:len(keyword)] != keyword {
		return false
	}
	if len(source) > len(keyword) && isModuleIdentifierPart(source[len(keyword)]) {
		return false
	}
	return true
}

func findTopLevelModuleKeyword(source string, keyword string) int {
	depth := 0
	inSingle := false
	inDouble := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(source); i++ {
		ch := source[i]

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && i+1 < len(source) && source[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inSingle {
			if ch == '\\' {
				i++
				continue
			}
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if ch == '\\' {
				i++
				continue
			}
			if ch == '"' {
				inDouble = false
			}
			continue
		}
		if inBacktick {
			if ch == '\\' {
				i++
				continue
			}
			if ch == '`' {
				inBacktick = false
			}
			continue
		}

		if ch == '/' && i+1 < len(source) {
			next := source[i+1]
			if next == '/' {
				inLineComment = true
				i++
				continue
			}
			if next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		switch ch {
		case '\'':
			inSingle = true
			continue
		case '"':
			inDouble = true
			continue
		case '`':
			inBacktick = true
			continue
		case '{', '(', '[':
			depth++
			continue
		case '}', ')', ']':
			if depth > 0 {
				depth--
			}
			continue
		}

		if depth == 0 && i+len(keyword) <= len(source) && source[i:i+len(keyword)] == keyword {
			if (i == 0 || !isModuleIdentifierPart(source[i-1])) && (i+len(keyword) == len(source) || !isModuleIdentifierPart(source[i+len(keyword)])) {
				return i
			}
		}
	}

	return -1
}

func isModuleIdentifierPart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func parseQuotedModuleSpecifier(source string) (string, bool, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", false, fmt.Errorf("expected module specifier string literal in this bounded classic-JS slice")
	}
	if trimmed[0] != '\'' && trimmed[0] != '"' {
		return "", false, fmt.Errorf("expected module specifier string literal in this bounded classic-JS slice")
	}
	quote := trimmed[0]
	i := 1
	var b strings.Builder
	for i < len(trimmed) {
		ch := trimmed[i]
		if ch == quote {
			return b.String(), true, nil
		}
		if ch == '\\' {
			i++
			if i >= len(trimmed) {
				return "", false, fmt.Errorf("unterminated module specifier string literal in this bounded classic-JS slice")
			}
			b.WriteByte(trimmed[i])
			i++
			continue
		}
		b.WriteByte(ch)
		i++
	}
	return "", false, fmt.Errorf("unterminated module specifier string literal in this bounded classic-JS slice")
}
