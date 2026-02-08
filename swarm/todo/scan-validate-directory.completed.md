# Validate scan target directory

## Problem

The `scan` command doesn't validate that the target directory exists or is actually a directory before scanning. If you run `swarm-index scan /nonexistent`, `Scan()` calls `filepath.Abs` (which doesn't check existence) then `filepath.Walk`. When `Walk` encounters the nonexistent root, it calls the walk function with a non-nil `err` — but the walk function swallows that error with `return nil // skip entries we can't read`. The result: `Scan` returns an empty index with no error, then `Save` writes an empty `index.json` and a `meta.json` claiming 0 files.

This is confusing for both humans and agents:
- No error is reported, so the caller thinks the scan succeeded.
- An empty index is persisted to disk, overwriting any previous valid index.
- Subsequent `lookup` calls return no results with no indication that the index is bogus.

Similarly, if the user passes a file path instead of a directory (e.g., `swarm-index scan main.go`), the behavior is undefined — `filepath.Walk` on a file just visits that one file, producing a broken index with a single entry and a misleading root.

## Goal

Add input validation to `Scan` so it fails fast with a clear error when the target is not an existing directory.

## Plan

### 1. Add directory validation at the top of `Scan()` in `index/index.go`

After `filepath.Abs(root)`, add a check:

```go
info, err := os.Stat(root)
if err != nil {
    return nil, fmt.Errorf("cannot access %s: %w", root, err)
}
if !info.IsDir() {
    return nil, fmt.Errorf("%s is not a directory", root)
}
```

This catches:
- Nonexistent paths → clear "cannot access" error with the underlying OS error
- Files passed instead of directories → clear "is not a directory" error

### 2. Add tests in `index/index_test.go`

Add two test cases:

- **`TestScanNonexistentDir`**: Call `Scan("/tmp/nonexistent-path-xxxx")`, verify it returns a non-nil error containing "cannot access".
- **`TestScanFileNotDir`**: Create a temp file, call `Scan(tempFilePath)`, verify it returns a non-nil error containing "not a directory".

### 3. Verify existing tests still pass

Run `go test ./...` to confirm no regressions.

## Files to Modify

- `index/index.go` — add `os.Stat` check after `filepath.Abs` in `Scan` (~5 lines)
- `index/index_test.go` — add 2 new tests (~20 lines)

## Notes

- This is a Phase 1 gap — the PLAN.md says "the tool is actually usable end-to-end," but silently producing empty indexes for bad input is not usable behavior.
- No new dependencies — just `os.Stat` which is already imported.
- The validation goes in `Scan()` (not `main.go`) so all callers benefit, including tests and future commands that call `Scan` internally.

## Completion Notes (agent f25ea432)

Implemented as planned:
- Added `os.Stat` validation in `Scan()` after `filepath.Abs()` — returns clear errors for nonexistent paths and non-directory paths.
- Added `TestScanNonexistentDir` and `TestScanFileNotDir` tests.
- All 13 tests pass (`go test ./...`), no regressions.
