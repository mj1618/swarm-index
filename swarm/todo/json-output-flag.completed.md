# Add `--json` global flag for structured output

## Problem

The README advertises `--json` as a global flag ("Supported by every command"), and the PLAN.md lists it as Phase 2. However, it is not implemented at all — no Go code references `--json`. Running `swarm-index scan --json .` treats `--json` as the directory argument and fails. This is a critical gap for the MVP since coding agents are the primary consumers and need structured, parseable output.

## Scope

Add a `--json` flag that works across all three existing commands: `scan`, `lookup`, and `version`.

## Implementation

### 1. Parse the `--json` flag globally

In `main.go`, before the command switch, scan `os.Args` for `--json` and strip it out so it doesn't interfere with positional argument parsing. Store the result in a `jsonOutput bool`.

```go
// Strip --json from args and return whether it was present.
func extractJSONFlag(args []string) ([]string, bool) {
    var filtered []string
    found := false
    for _, a := range args {
        if a == "--json" {
            found = true
        } else {
            filtered = append(filtered, a)
        }
    }
    return filtered, found
}
```

Call this early in `main()` and replace `os.Args` usage with the filtered args.

### 2. JSON output for each command

**`scan`** — On success, emit:
```json
{
  "filesIndexed": 42,
  "packages": 5,
  "indexPath": "./swarm/index/",
  "extensions": { ".go": 28, ".md": 8 }
}
```

**`lookup`** — Emit the matching entries as a JSON array:
```json
[
  { "name": "handler.go", "kind": "file", "path": "api/handler.go", "line": 0, "package": "api" }
]
```
When no matches are found, emit `[]`.

**`version`** — Emit:
```json
{ "version": "v0.1.0" }
```

### 3. Error output under `--json`

When `--json` is active, errors should also be JSON to avoid breaking parsers:
```json
{ "error": "no index found — run 'swarm-index scan <dir>' first" }
```
Write errors to stderr as today, but format them as JSON.

### 4. Tests

- Test `extractJSONFlag` strips the flag and returns true/false.
- Test that `--json` can appear anywhere in the args (before command, after command, at the end).
- Integration-style tests: run scan with `--json` and verify output is valid JSON with expected keys.
- Test that lookup with `--json` and no matches returns `[]`.

## Files to modify

- `main.go` — add `extractJSONFlag`, adjust `main()` to use filtered args, add JSON output branches for each command.
- `main_test.go` — add tests for the new flag parsing and JSON formatting helpers.

## Out of scope

- No changes to the `index` package needed.
- Future commands (tree, summary, outline, etc.) will add their own JSON branches when implemented.

## Completion notes

Implemented by agent 9cc79186 (task 2a6856e4).

### Changes made:
- **main.go**: Added `extractJSONFlag()` to strip `--json` from args globally. Added `jsonError()` helper for JSON-formatted stderr errors. Wired `--json` into all three commands (`scan`, `lookup`, `version`) and the default/error paths. Lookup with no matches returns `[]` (not `null`). All `os.Args` references in the command switch replaced with filtered `args`.
- **main_test.go**: Added 5 tests for `extractJSONFlag` covering: flag absent, flag at end, flag before command, flag between args, and empty args.

### Verified:
- All 29 tests pass (`go test ./...`)
- Manual integration testing: `version --json`, `--json version`, `lookup --json`, `lookup --json` with no matches (`[]`), `lookup --json` with empty query (JSON error on stderr), `scan --json` on temp directory — all produce valid JSON with expected keys.
