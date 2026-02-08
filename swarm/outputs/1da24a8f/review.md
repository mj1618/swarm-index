# Refactoring Review — Unstaged Changes

## Summary
No refactoring applied. The unstaged changes are clean and consistent.

## What was reviewed
- `index/hotspots.go` — new hotspots command implementation
- `index/hotspots_test.go` — 10 test cases covering the feature
- `main.go` — CLI wiring for the hotspots command
- `README.md` — documentation additions
- `SKILL.md` — skill documentation additions

## Analysis
- **Pattern consistency**: The hotspots code follows the same conventions as `history.go` (struct types, formatting, error handling, test patterns).
- **Git error handling**: Both `hotspots.go` and `history.go` share a similar `exec.ExitError` pattern. Only two call sites — not worth extracting a helper.
- **`getLastModified` N+1 calls**: Each hotspot entry triggers a separate `git log -1`. Acceptable for the default max of 20 entries and consistent with the simple patterns used elsewhere.
- **No dead code, unused imports, or naming issues found.**
- **Tests pass, code compiles.**
