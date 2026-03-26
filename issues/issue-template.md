# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `doc/subsystem-map.md` and identify the owning subsystem.
- Review `doc/implementation-guide.md` and pick the test layer first.
- Review `doc/capability-matrix.md` and confirm the capability row or gap.
- Review `doc/roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary:

## Context

- Owning subsystem:
- Related capability or gap:
- Related docs:

## Problem

- Current behavior:
- Expected behavior:
- Reproduction steps:
  1.
  2.
  3.
- Reproduction code:

```text
# Paste the smallest code or test case that reproduces the problem.
```

- Original failed command:

```bash
# Paste the exact command that failed, including flags and arguments.
```
- Scope / non-goals:

## Acceptance Criteria

- [ ] Primary behavior is implemented or fixed.
- [ ] Failure paths are explicit and do not silently fall back.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer:
- Regression or failure-path coverage:
- Mock or fixture needs:

## Notes

- Links, screenshots, logs, or other context:
