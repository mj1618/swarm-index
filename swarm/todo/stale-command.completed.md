# `stale` Command — Detect Out-of-Date Index

## Summary

Add a `stale` command that compares the persisted index against the current filesystem and reports new, deleted, and modified files. This lets agents quickly check if the index is trustworthy before relying on it, without needing to re-scan.

## Priority

High — agents that rely on stale index data will miss new files, reference deleted files, and have outdated symbol information. This is a fundamental trust signal.

## Usage

```bash
# Check if the index is stale
swarm-index stale [--root <dir>]

# JSON output for agent consumption
swarm-index stale --root ~/code/my-project --json
```

## Text Output Example

```
Index scanned at 2025-06-01T12:00:00Z (3 hours ago)

New files (not in index):
  src/handlers/webhook.go
  src/handlers/webhook_test.go

Deleted files (in index but missing from disk):
  src/legacy/old_handler.go

Modified files (changed since last scan):
  src/main.go
  src/config/config.go

Summary: 2 new, 1 deleted, 2 modified — index is STALE (run 'swarm-index scan' to update)
```

If everything is up-to-date:
```
Index scanned at 2025-06-01T14:30:00Z (5 minutes ago)

No changes detected — index is up to date.
```

## JSON Output Example

```json
{
  "scannedAt": "2025-06-01T12:00:00Z",
  "isStale": true,
  "newFiles": ["src/handlers/webhook.go", "src/handlers/webhook_test.go"],
  "deletedFiles": ["src/legacy/old_handler.go"],
  "modifiedFiles": ["src/main.go", "src/config/config.go"],
  "summary": {
    "new": 2,
    "deleted": 1,
    "modified": 2
  }
}
```

## Implementation

### 1. Add `Stale` method to `index` package (`index/stale.go`)

```go
type StaleResult struct {
    ScannedAt     string   `json:"scannedAt"`
    IsStale       bool     `json:"isStale"`
    NewFiles      []string `json:"newFiles"`
    DeletedFiles  []string `json:"deletedFiles"`
    ModifiedFiles []string `json:"modifiedFiles"`
    Summary       struct {
        New      int `json:"new"`
        Deleted  int `json:"deleted"`
        Modified int `json:"modified"`
    } `json:"summary"`
}
```

The method should:

1. Read `meta.json` to get the `scannedAt` timestamp and root path.
2. Build a set of file paths from the current index entries (only `kind == "file"`).
3. Walk the filesystem using the same `shouldSkipDir` rules as `Scan`.
4. For each file on disk:
   - If not in the index set → add to `NewFiles`.
   - If in the index set → compare the file's `ModTime` against `scannedAt`. If newer → add to `ModifiedFiles`. Remove from the set.
5. Any paths remaining in the index set after the walk → add to `DeletedFiles`.
6. Set `IsStale = len(NewFiles) + len(DeletedFiles) + len(ModifiedFiles) > 0`.

### 2. Store `scannedAt` as `time.Time` internally

The `Load` function already reads `meta.json` which contains `scannedAt`. Expose this on the `Index` struct (or return it from a new `LoadMeta` function) so the `Stale` method can compare file mod times against it.

### 3. Add `FormatStale` for human-readable output

A function that takes `StaleResult` and returns a formatted string (similar to `FormatSummary`).

### 4. Wire up CLI in `main.go`

Add a `case "stale":` block that:
- Resolves the root (same as other commands).
- Calls the `Stale` method.
- Outputs text or JSON based on `--json` flag.

### 5. Update README.md and SKILL.md

Add `stale` to the commands table and examples.

## Testing (`index/stale_test.go`)

1. **Fresh index**: Scan a temp dir, save, immediately check stale → `IsStale == false`, all lists empty.
2. **New file**: Scan, save, create a new file, check stale → file appears in `NewFiles`.
3. **Deleted file**: Scan, save, delete a file, check stale → file appears in `DeletedFiles`.
4. **Modified file**: Scan, save, wait briefly (or touch the file with a future mtime), modify a file, check stale → file appears in `ModifiedFiles`.
5. **Combined**: Multiple changes at once → all three lists populated correctly.

## Dependencies

- Requires Phase 1 (persistence) — already completed.
- No external dependencies needed.

## Completion Notes

Implemented by agent e298bf8a. All items completed:

1. **index/stale.go** — `StaleResult`, `StaleSummary` structs, `Stale()` method on `*Index`, and `FormatStale()` formatter. The method reads `meta.json` for the `scannedAt` timestamp, builds a set of indexed file paths, walks the filesystem using the same `shouldSkipDir` rules as `Scan`, and categorizes files as new, deleted, or modified. Includes a 1-second buffer on `scannedAt` to avoid false positives from RFC3339 second-precision truncation.
2. **main.go** — Added `case "stale":` with `--root` and `--json` flag support. Updated `printUsage()`.
3. **index/stale_test.go** — 7 tests covering: fresh index (not stale), new file detection, deleted file detection, modified file detection, combined changes, format output for up-to-date, and format output with changes.
4. **README.md** — Added `stale` to commands table, quick start examples, project structure, and marked as completed in roadmap.
5. **SKILL.md** — Added `stale` usage examples.

All tests pass (`go test ./...`).
