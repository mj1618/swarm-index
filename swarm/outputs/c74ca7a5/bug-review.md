# Bug Review

## Bug Found and Fixed

### `isCommentLine` missing `/*` prefix for JS/TS (index/context.go:156)

A prior refactoring stage removed `HasPrefix(trimmed, "/**")` and `HasPrefix(trimmed, "*/")` as "redundant" since `HasPrefix(trimmed, "*")` already covers strings starting with `*`. However, this analysis was incorrect for `/**` — the JSDoc opening delimiter starts with `/`, not `*`, so neither `HasPrefix(trimmed, "//")` nor `HasPrefix(trimmed, "*")` would match it.

This meant JSDoc-style `/** ... */` opening lines would not be recognized as comment lines, causing `extractDocComment` to stop collecting too early and miss the first line of multi-line JSDoc comments.

**Fix:** Added `HasPrefix(trimmed, "/*")` to the JS/TS branch, which correctly matches `/**`, `/*`, and any other block comment opening.

## No Other Bugs Found

The remaining changes (main.go context command, README/SKILL.md docs, context.go core logic, tests) were reviewed and found correct.
