# Refactoring Review

## Changes Reviewed
- `README.md` — Documentation updates for Python parser support
- `SKILL.md` — Documentation updates for Python parser support
- `parsers/pyparser.go` — New Python heuristic parser (untracked)
- `parsers/pyparser_test.go` — Tests for Python parser (untracked)

## Refactoring Applied

**Extracted `symbolsByName` test helper** (`parsers/goparser_test.go`):
- Both `goparser_test.go` and `pyparser_test.go` repeated a 4-line `byName` map construction pattern (8 times total across both files).
- Extracted into a shared `symbolsByName(symbols []Symbol) map[string]Symbol` helper in `goparser_test.go` (same package, accessible to both test files).
- Replaced all 8 occurrences across both test files.

## Verified
- `go build ./...` — compiles cleanly
- `go test ./parsers/ -v` — all 17 tests pass
