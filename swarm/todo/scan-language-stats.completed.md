# Add language/extension stats to scan output and meta.json

## Problem

When `scan` finishes, it only reports the total file count and package count:

```
Index saved to ./swarm/index/ (42 files, 5 packages)
```

And `meta.json` only stores:

```json
{
  "root": "/path/to/project",
  "scannedAt": "2025-01-01T00:00:00Z",
  "version": "0.1.0",
  "fileCount": 42,
  "packageCount": 5
}
```

The PLAN.md (Phase 4 / `meta.json` description) explicitly calls for a **language breakdown** in the metadata. Without it, an agent loading the index has no quick way to know what kind of project it's looking at (is it Go? TypeScript? Python? A mix?) without iterating over every entry and counting extensions itself.

This is a gap in the existing `scan` feature, not a new command. The scan already walks every file — it just doesn't aggregate the extension data it already has.

## Solution

### 1. Add `Extensions` field to `indexMeta`

Add a `map[string]int` field to `indexMeta` that counts files by extension:

```go
type indexMeta struct {
    Root         string         `json:"root"`
    ScannedAt    string         `json:"scannedAt"`
    Version      string         `json:"version"`
    FileCount    int            `json:"fileCount"`
    PackageCount int            `json:"packageCount"`
    Extensions   map[string]int `json:"extensions"`
}
```

### 2. Add `ExtensionCounts()` method to `*Index`

Iterate over entries, extract `filepath.Ext(e.Name)`, and return a `map[string]int`. Files with no extension are counted under `"(none)"`.

### 3. Wire into `Save`

Call `idx.ExtensionCounts()` and assign to `meta.Extensions` before writing `meta.json`.

### 4. Improve scan summary output

After saving, print a one-line breakdown of the top extensions. For example:

```
Index saved to ./swarm/index/ (42 files, 5 packages)
  .go: 28, .md: 8, .json: 4, .mod: 1, .sum: 1
```

### 5. Tests

- Test `ExtensionCounts()` returns correct counts for a mixed-extension temp directory.
- Test that `meta.json` includes the `extensions` field after save.
- Test round-trip: save with extensions → load → verify meta (requires reading meta.json directly or extending Load to expose metadata).

## Files to change

- `index/index.go` — add `ExtensionCounts()` method, add `Extensions` field to `indexMeta`, update `Save`.
- `index/index_test.go` — add tests for extension counting and meta persistence.
- `main.go` — update scan output to print extension summary.

## Scope

Small. ~30 lines of new logic, ~30 lines of tests. No new dependencies. No new commands.

## Completion Notes (agent 54a7e6b0)

All five steps implemented:

1. **`Extensions` field** added to `indexMeta` as `map[string]int` with `json:"extensions"` tag.
2. **`ExtensionCounts()` method** added to `*Index` — iterates unique file paths, extracts `filepath.Ext()`, counts under `"(none)"` for extensionless files.
3. **`Save` wired** — `meta.Extensions` populated via `idx.ExtensionCounts()` before writing `meta.json`.
4. **Scan summary updated** in `main.go` — prints a second indented line with extensions sorted by count descending (e.g., `.go: 4, .md: 3, .mod: 1`).
5. **Three tests added** to `index/index_test.go`:
   - `TestExtensionCounts` — verifies correct counts for mixed extensions including `(none)`.
   - `TestSaveMetaExtensions` — verifies `meta.json` includes the `extensions` field after save.
   - `TestExtensionCountsRoundTrip` — verifies extensions survive save → read-back from `meta.json`.

All existing and new tests pass (`go test ./...`).
