# Canvas 2D export needs real image pixels for drawImage, toBlob, and ImageBitmap flows

## Summary

- Short summary: The bounded browser runtime needs a real 2D canvas surface that preserves pixels across `drawImage()`, `fillRect()`, `clearRect()`, `toBlob()`, `toDataURL()`, and `convertToBlob()`, plus image snapshots for loaded `<img>` elements and `ImageBitmap` objects.

## Context

- Owning subsystem: Runtime / canvas and image decoding
- Related capability or gap: canvas 2D context host references, pixel-backed canvas state, and image snapshot caching
- Related docs:
  - `internal/runtime/canvas_browser.go`
  - `internal/runtime/canvas_state.go`
  - `internal/runtime/image_bitmap.go`
  - `internal/runtime/image_events.go`
  - `internal/runtime/image_metadata.go`
- Affected finitefield-site coverage:
  - `../finitefield-site/web-go/internal/generate/image_image_resizer_compressor_browser_test.go`
  - `../finitefield-site/web-go/internal/generate/image_image_converter_browser_test.go`

## Problem

- Current behavior: The runtime has a partial canvas implementation, but canvas context access and image snapshot storage are not fully wired for production-style image processing tests. When browser code loads images, draws them onto a canvas, and exports JPEG or PNG, the runtime can lose the original pixel data or return a stubbed canvas surface.
- Expected behavior: Browser tests should be able to create a canvas, set `fillStyle`, paint a white background, draw loaded images or `ImageBitmap` instances, and export bytes that reflect the rendered pixels and JPEG quality settings.
- Reproduction steps:
  1. Load an image through `<img>` or `createImageBitmap()`.
  2. Draw it to a canvas after the original blob URL has been revoked.
  3. Export the canvas as JPEG or PNG and inspect the bytes or preview.
- Reproduction code:

```text
const img = await new Promise((resolve, reject) => {
  const node = new Image();
  node.onload = () => resolve(node);
  node.onerror = reject;
  node.src = URL.createObjectURL(file);
});
URL.revokeObjectURL(url);
const canvas = document.createElement("canvas");
const ctx = canvas.getContext("2d");
ctx.fillStyle = "#ffffff";
ctx.fillRect(0, 0, canvas.width, canvas.height);
ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
const blob = await canvas.convertToBlob({ type: "image/jpeg", quality: 0.7 });
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run '^TestImageResizerCompressor'
```

- Scope / non-goals: Keep the fix focused on browser-tester-go canvas/image runtime behavior. Do not work around the gap inside finitefield-site tests.

## Acceptance Criteria

- [ ] `canvas.getContext("2d")` returns a host-backed context surface that can mutate canvas pixels.
- [ ] `fillStyle`, `fillRect()`, `clearRect()`, and `drawImage()` update canvas pixels used by `toBlob()`, `convertToBlob()`, and `toDataURL()`.
- [ ] Loaded `<img>` elements keep a decoded pixel snapshot usable after blob URLs are revoked.
- [ ] `ImageBitmap` keeps decoded pixel data and can be drawn after the source object is closed.
- [ ] JPEG export reflects rendered pixels and quality changes.
- [ ] Regression tests cover transparent PNG to white-background JPEG export and JPEG quality size differences.

## Test Plan

- Suggested test layer: runtime/session tests for canvas behavior, plus browser-level smoke tests from finitefield-site.
- Regression or failure-path coverage: verify drawImage after URL revocation, transparent image flattening, and JPEG size differences across quality values.
- Mock or fixture needs: patterned PNG fixtures and transparent PNG fixtures.

## Notes

- This gap blocks the image resizer/compressor browser tests.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
