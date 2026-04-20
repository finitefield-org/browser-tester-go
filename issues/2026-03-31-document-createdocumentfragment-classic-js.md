# `document.createDocumentFragment()` is unavailable in bounded classic-JS slices

## Summary
The `document.createDocumentFragment()` browser surface is missing in the bounded classic-JS runtime path used by `web-go`.

## Repro
Working directory:
`/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`

Command:
```bash
go test ./internal/generate -run TestConstructionPipeDimensionsChart -count=1
```

## Result
`SetSelectValue()` and `Click()` paths fail with:
`unsupported browser surface "document.createDocumentFragment" in this bounded classic-JS slice`

## Notes
This is a browser-tester-go issue, not a `web-go` issue. A workaround in the page code would only mask the missing runtime support.
