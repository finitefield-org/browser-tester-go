# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: Classic JS slice lacks the rasterization primitives needed for SVG-to-PNG export tests.

## Context

- Owning subsystem: `internal/runtime`
- Related capability or gap: `Image`, `canvas.getContext("2d")`, `drawImage`, and `toBlob` support in page scripts
- Related docs: `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: The PNG export path in `finitefield-site` cannot complete because the browser runtime does not provide the canvas/image APIs needed to turn SVG markup into a PNG blob.
- Expected behavior: Page scripts should be able to create an image, draw it onto a canvas, and extract a PNG blob so download assertions can inspect the artifact bytes.
- Reproduction steps:
  1. Run the finitefield PNG export regression below.
  2. Click the PNG export button with sheet scope selected.
  3. No PNG download is captured.
- Reproduction code:

```text
<main><div id="out"></div><script>
  const img = new Image();
  const canvas = document.createElement("canvas");
  canvas.width = 20;
  canvas.height = 20;
  const ctx = canvas.getContext("2d");
  ctx.drawImage(img, 0, 0);
  canvas.toBlob((blob) => {
    document.getElementById("out").textContent = blob ? "ok" : "missing";
  }, "image/png");
</script></main>
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run TestAgriLotTraceabilityLabelsBRT016 -count=1
```
- Scope / non-goals: Do not work around the missing APIs in `finitefield-site`; add the runtime support needed for PNG export tests to observe real PNG bytes.

## Acceptance Criteria

- [ ] Primary behavior is implemented or fixed.
- [ ] Failure paths are explicit and do not silently fall back.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: `internal/runtime`
- Regression or failure-path coverage: Add a runtime regression test that exercises SVG-to-PNG export support.
- Mock or fixture needs: A minimal SVG or image fixture, if the runtime needs one.

## Notes

- Links, screenshots, logs, or other context: `finitefield-site/web-go/internal/generate/agri_lot_traceability_labels_browser_test.go` fails in `TestAgriLotTraceabilityLabelsBRT016PNGScopeSwitchesToSheetDownload` with an empty download list.
