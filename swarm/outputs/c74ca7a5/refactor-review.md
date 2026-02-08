# Refactoring Review

## Changes Applied

### 1. Eliminated duplicate import-extraction functions in `index/context.go`

The new `context.go` file contained three functions (`extractGoFileImports`, `extractJSFileImports`, `extractPyFileImports`) that were exact duplicates of existing functions in `related.go` (`extractGoImports`, `extractJSImports`, `extractPyImports`).

**Fix:** Removed the three duplicated functions and updated `extractFileImports` to call the shared functions from `related.go` directly.

### 2. Removed redundant `HasPrefix` checks in `isCommentLine`

In the JS/TS branch, `strings.HasPrefix(trimmed, "/**")` and `strings.HasPrefix(trimmed, "*/")` were redundant since `strings.HasPrefix(trimmed, "*")` already matches any string starting with `*`.

**Fix:** Simplified to just `HasPrefix(trimmed, "//") || HasPrefix(trimmed, "*")`.

## Verification

All tests pass after refactoring (`go test ./...`).
