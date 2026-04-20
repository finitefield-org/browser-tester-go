# Synthetic `paste` still loses clipboard payload in tolerance-checker manual table tests

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and choose the smallest layer that can fix the behavior.
- Review `capability-matrix.md` and confirm the missing capability row.
- Review `roadmap.md` if the fix affects sequencing or rollout order.

## Summary

- Short summary: A synthetic `paste` dispatched by the browser harness still does not carry clipboard text into `event.clipboardData`, so the tolerance-checker manual table paste test cannot expand TSV rows.

## Context

- Owning subsystem: `bt-runtime` event dispatch / user-like actions
- Related capability or gap: clipboard-backed `paste` event payloads for delegated paste handlers
- Related docs: `browser-tester-go/doc/capability-matrix.md`, `browser-tester-go/doc/implementation-guide.md`

## Problem

- Current behavior: `finitefield-site/web-go/internal/generate/construction_tolerance_checker_browser_test.go` seeds the clipboard with TSV, dispatches `paste` on the first manual table input, and the table stays at 2 rows instead of expanding to 3.
- Expected behavior: The manual table `paste` handler should receive the seeded clipboard payload through `event.clipboardData.getData("text/plain")` and paste all TSV rows into the table.
- Reproduction steps:
  1. Load the tolerance-checker page in the browser harness.
  2. Open the manual tab.
  3. Seed the clipboard with TSV text containing three rows.
  4. Dispatch `paste` on the first manual input cell.
  5. Observe that the manual table row count remains 2 and no pasted TSV content appears.
- Reproduction code:

```go
harness, _ := browsertester.FromHTMLWithURL("https://finitefield.org/", html)
_ = harness.WriteClipboard("A001\t10.00\t10.01\nA002\t10.02\t10.03\nA003\t10.04\t10.05")
_ = harness.Dispatch("#tolerance-checker-manual-table input[data-manual-row=\"0\"][data-manual-col=\"p\"][data-manual-point=\"0\"]", "paste")
```

```html
<main>
  <table id="tolerance-checker-manual-table">
    <tr>
      <td><input data-manual-row="0" data-manual-col="id"></td>
      <td><input data-manual-row="0" data-manual-col="p" data-manual-point="0"></td>
    </tr>
  </table>
  <script>
    document.getElementById("tolerance-checker-manual-table").addEventListener("paste", (event) => {
      const target = event.target;
      if (!(target instanceof HTMLInputElement)) return;
      const text = event.clipboardData ? event.clipboardData.getData("text/plain") : "";
      if (!text) return;
      document.getElementById("tolerance-checker-manual-table").dataset.pasted = text;
    });
  </script>
</main>
```

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run 'TestConstructionToleranceCheckerBRT011ManualPasteExpandsTSVIntoTable' -count=1
```

- Scope / non-goals: This is about the browser tester runtime event payload plumbing. It is not a request to change the tolerance-checker page logic or to work around the missing clipboard payload in `web-go`.

## Evidence

- Failing test:
  - `TestConstructionToleranceCheckerBRT011ManualPasteExpandsTSVIntoTable`
- Observed failure:

```text
manual table row count after paste = 2, want 3
```

- Why this points to `browser-tester-go`:
  - The page's paste handler already consumes `event.clipboardData` and expands the table when TSV text is present.
  - The harness does seed clipboard state, but the synthetic dispatched event does not surface that payload to the listener, so the page sees an empty paste.

## Acceptance Criteria

- [ ] Synthetic `paste` events can expose seeded clipboard text through `event.clipboardData`.
- [ ] Delegated `paste` listeners on elements like the tolerance-checker manual table can read TSV payloads without a workaround.
- [ ] The tolerance-checker manual paste browser test expands the table from 2 rows to 3 rows when three TSV rows are pasted.

## Test Plan

- Suggested test layer: runtime/session event dispatch test plus a public harness contract test for clipboard-backed paste.
- Regression coverage: verify a listener attached to a container element can read `event.clipboardData.getData("text/plain")` from a synthetic paste action and mutate the DOM accordingly.
- Mock or fixture needs: a minimal HTML page with a delegated `paste` listener and seeded clipboard text.

## Notes

- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
- This issue is separate from the earlier general paste-clipboard report because it was observed in the tolerance-checker manual paste flow after the browser harness had already been updated.
- Reconfirmed after rerunning `TestConstructionToleranceCheckerBRT011ManualPasteExpandsTSVIntoTable` on 2026-03-31; the failure still shows `manual table row count after paste = 2, want 3`.
