# Refactoring Review — Unstaged Changes

## Changes Reviewed
- **README.md / SKILL.md**: Documentation for new `tree` command
- **main.go**: New `tree` case, `parseDepth` function, usage updates
- **index/tree.go**: New `BuildTree` / `RenderTree` implementation
- **index/tree_test.go**: Tests for tree functionality

## Refactoring Applied

**Deduplicated `parseDepth` / `parseMax` into `parseIntFlag`** (`main.go`)

`parseDepth` and `parseMax` were identical functions differing only in the flag name and default value. Extracted a single `parseIntFlag(args, flag, defaultVal)` helper and updated both call sites and tests.

Files changed: `main.go`, `main_test.go`

## No Other Issues Found
- `index/tree.go` is clean — correctly reuses `shouldSkipDir` from `index.go`
- `tree_test.go` reuses the existing `mkFile` test helper
- No dead code, unused imports, or naming issues
- All tests pass
