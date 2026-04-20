# Enter key dispatch is not configurable

- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site`
- Blocked regression: `finitefield-site/web-go/internal/generate/construction_diagonal_calculator_browser_test.go` `BRT-012`
- Runtime reference: `browser-tester-go/internal/runtime/dispatch.go`

The finitefield-site `diagonal-calculator` page adds a measurement row when the last measurement input receives `Enter`:

- `el.measureTbody.addEventListener("keydown", ...)`
- the handler checks `event.key === "Enter"`

`browser-tester-go` currently exposes `DispatchKeyboard(selector)` only, and it hardcodes `Escape` in `internal/runtime/dispatch.go`.

Because of that, the Enter-key regression for the diagonal calculator cannot be written without a workaround.

Expected:

- a public API to dispatch an arbitrary keyboard key, or
- a keyboard-event helper that can set `event.key` to `Enter`

Actual:

- only `Escape` can be dispatched
