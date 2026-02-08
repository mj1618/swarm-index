# Refactoring Review

## Changes Reviewed
Unstaged diff adding `diff-summary` command: `index/diffsummary.go`, `index/diffsummary_test.go`, `main.go`, `README.md`, `SKILL.md`.

## Refactoring Applied

### Removed dead code in `index/diffsummary.go`
The `indexed` map (previously lines 45-49) was built from `idx.Entries` but never referenced anywhere in the function. This was likely copy-pasted from `stale.go` where the same pattern IS used. Removed the unused variable and its construction loop.

## No Other Issues Found
- Format function follows established `strings.Builder` pattern used by all other Format* functions
- main.go command case follows the standard parse-args/resolve-root/load-index/call/format pattern
- `extractSymbols` is not duplicating existing code — exports.go and outline return full structs, while this returns just names
- Naming is clear and consistent
- No dead imports or loose typing
- Tests are comprehensive
