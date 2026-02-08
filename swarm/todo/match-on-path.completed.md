# Match on Path in Lookup

## Problem

The `Match` method (used by `lookup`) only checks `e.Name` (the bare filename). It does not consider `e.Path` (the relative file path). This means searching for path fragments like `"api/handler"`, `"lib/"`, or `"lib/util"` returns zero results, even though those files exist in the index.

The CLI usage string says `lookup` is for looking up "symbols, files, or concepts," and users/agents commonly search by path fragments. This is a gap in the existing Phase 1 lookup functionality.

## Goal

Expand `Match` to also match against the `Path` field, so path-based queries return results.

## Plan

### 1. Update `Match` in `index/index.go`

Change the matching logic to check both `e.Name` and `e.Path` (case-insensitive substring match on either). A match on either field should include the entry in results. Deduplicate — if both name and path match, the entry should appear only once.

Current code:
```go
func (idx *Index) Match(query string) []Entry {
    q := strings.ToLower(query)
    var results []Entry
    for _, e := range idx.Entries {
        if strings.Contains(strings.ToLower(e.Name), q) {
            results = append(results, e)
        }
    }
    return results
}
```

Updated logic:
```go
func (idx *Index) Match(query string) []Entry {
    q := strings.ToLower(query)
    var results []Entry
    for _, e := range idx.Entries {
        if strings.Contains(strings.ToLower(e.Name), q) ||
            strings.Contains(strings.ToLower(e.Path), q) {
            results = append(results, e)
        }
    }
    return results
}
```

### 2. Add tests in `index/index_test.go`

Add test cases to the existing `TestMatch` table:

- `"api/handler"` should match 1 result (the handler.go file with path `api/handler.go`)
- `"lib/"` should match files in the `lib` directory (if present in test data — add entries for `lib/util.go` etc.)
- `"api/"` should match all files under `api/`
- Ensure no duplicate results when both name and path match (e.g., querying `"handler.go"` should still return exactly 1 result)

## Files to Modify

- `index/index.go` — update `Match` method (~2 lines changed)
- `index/index_test.go` — add path-matching test cases to `TestMatch`

## Completion Notes (Agent c6f9ea75)

Both plan steps implemented and verified:

1. **`Match` method updated** in `index/index.go` — now checks both `e.Name` and `e.Path` with case-insensitive substring matching. The `||` in the condition naturally deduplicates (an entry is appended at most once per iteration).
2. **6 new test cases** added to `TestMatch` in `index/index_test.go`:
   - `"api/handler"` → 1 result (path-only match)
   - `"api/"` → 2 results (all files under api/)
   - `"lib/"` → 2 results (all files under lib/)
   - `"lib/util"` → 1 result (specific path fragment)
   - `"handler.go"` → 1 result (no duplicates when both name and path match)
   - Added `lib/util.go` and `lib/helper.go` entries to the test fixture

All 8 tests pass (`go test ./...`).
