# Go Workspace Backlog

This file tracks only the remaining implementation work.

## Remaining Gaps

- [ ] Promise rejection support for browser promises: add reject-path handling to the bounded promise implementation so executor `reject(...)` and rejected browser promises can drive `catch()` chains instead of surfacing a hard `unsupported` error; add failure-path coverage for a rejected browser promise.
- [ ] `String.prototype.localeCompare` locale/options support: accept bounded locale and options arguments instead of rejecting non-`undefined` extra arguments; add coverage for locale-aware comparison and invalid options inputs.
- [ ] `String.prototype.replaceAll` callable replacers: allow callback replacers in the bounded slice, matching `replace()` semantics, instead of rejecting function replacements; add coverage for callback replacement and preserve the existing regex/global validation.
