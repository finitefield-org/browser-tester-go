# HTMLInputElement constructor is unavailable in inline input handlers

## Summary

- Short summary: `TypeText()` on text-like inputs can trigger page `input` handlers that guard with `instanceof HTMLInputElement`, but the bounded classic-JS slice rejects `HTMLInputElement` as an unsupported browser surface.

## Context

- Owning subsystem: `bt-runtime` browser-global bridge / constructor globals
- Related capability or gap: inline-script `instanceof HTMLInputElement` support
- Related docs: `browser-tester-go/README.md`, `browser-tester-go/doc/capability-matrix.md`

## Problem

- Current behavior: When an inline `input` listener evaluates `event.target instanceof HTMLInputElement`, the runtime throws `unsupported browser surface "HTMLInputElement" in this bounded classic-JS slice`.
- Expected behavior: Inline scripts should be able to use `HTMLInputElement` the same way they can use other supported constructor globals such as `HTMLButtonElement` and `HTMLSelectElement`.
- Reproduction steps:
  1. Load a page with an `input` listener that checks `event.target instanceof HTMLInputElement`.
  2. Call `Harness.TypeText("#side-0", "12")` on an `<input type="number">`.
  3. Observe the unsupported browser surface error.
- Reproduction code:

```html
<main>
  <div id="wrap">
    <input id="side-0" type="number" value="">
  </div>
  <div id="out"></div>
  <script>
    document.getElementById("wrap").addEventListener("input", (event) => {
      if (!(event.target instanceof HTMLInputElement)) return;
      document.getElementById("out").textContent = event.target.value;
    });
  </script>
</main>
```

```go
package main

import "browsertester"

func main() error {
	harness, err := browsertester.FromHTML(`<main>...</main>`)
	if err != nil {
		return err
	}
	return harness.TypeText("#side-0", "12")
}
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run ScaffoldingAreaCalculator -count=1
```

- Scope / non-goals: This is about supporting `HTMLInputElement` in browser tests. It is not a request to avoid the input listener by changing page code.

## Acceptance Criteria

- [ ] `HTMLInputElement` is available in the bounded browser surface.
- [ ] `instanceof HTMLInputElement` works in inline event handlers.
- [ ] A regression test covers a text-like input dispatching `input` through `TypeText()`.

## Test Plan

- Suggested test layer: runtime/session test for `TypeText()` plus a public contract test if needed.
- Regression coverage: verify a page-level `input` listener using `instanceof HTMLInputElement` can read the target value without throwing.
- Mock or fixture needs: a minimal HTML page with an `input` listener that writes the target value to the DOM.

## Notes

- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
- This gap blocks exact browser coverage for input-driven polygon rows in `finitefield-site/web-go/internal/generate/construction_scaffolding_area_calculator_browser_test.go`.
