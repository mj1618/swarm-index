# Persist Index to Disk (Save & Load)

## Problem

`Scan` builds an in-memory index but never writes it to disk. When the process exits, all scan results are lost. `Lookup` is completely stubbed — it always returns `"no index loaded"`. This means the tool is not usable end-to-end. Every downstream feature (lookup, summary, tree cache, search, refs, etc.) depends on a persisted index.

## Goal

Add `Save` and `Load` methods to `*Index` so that `scan` persists results to `./swarm/index/` and `lookup` (and future commands) can read them back.

## Plan

### 1. Add `Save(dir string) error` to `*Index` in `index/index.go`

- Create `<dir>/swarm/index/` directory if it doesn't exist (`os.MkdirAll`).
- Marshal `idx.Entries` to `swarm/index/index.json` using `json.MarshalIndent` (2-space indent for readability/diffability).
- Write `swarm/index/meta.json` with:
  - `root` — the absolute scan root path
  - `scannedAt` — RFC 3339 timestamp
  - `version` — `"0.1.0"`
  - `fileCount` — from `idx.FileCount()`
  - `packageCount` — from `idx.PackageCount()`

### 2. Add `Load(dir string) (*Index, error)` in `index/index.go`

- Read `<dir>/swarm/index/index.json`.
- Unmarshal into `[]Entry`, construct and return an `*Index`.
- Read `meta.json` to populate `idx.Root`.
- Return a clear error if the index directory or files don't exist.

### 3. Wire up `scan` command in `main.go`

- After `index.Scan(dir)`, call `idx.Save(dir)`.
- Print a summary line including the path to the index directory, e.g.: `"Index saved to ./swarm/index/ (42 files, 5 packages)"`.

### 4. Wire up `lookup` command in `main.go`

- Replace the current stubbed `index.Lookup(query)` call.
- Determine project root: walk up from CWD looking for `swarm/index/meta.json`, or use the directory containing the index.
- Call `index.Load(root)` to get the persisted index.
- Call `idx.Match(query)` and print results.
- The standalone `Lookup` function can either be removed or refactored to use `Load` internally.

### 5. Tests in `index/index_test.go`

- **Round-trip test**: Scan a temp dir → `Save` → `Load` → verify entries match original.
- **Meta test**: After `Save`, read `meta.json` and verify fields (root, version, counts, timestamp format).
- **Directory creation test**: `Save` to a path where `swarm/index/` doesn't exist yet → verify it's created.
- **Load error test**: `Load` from a nonexistent path → verify a clear error is returned.
- **Lookup integration test**: Scan + Save, then Load + Match, verify results.

## Files to Modify

- `index/index.go` — add `Save`, `Load`, meta struct, JSON tags on `Entry`
- `index/index_test.go` — add persistence tests
- `main.go` — wire `Save` into `scan`, replace `Lookup` stub with `Load` + `Match`

## Notes

- Add `json:"..."` struct tags to `Entry` fields (currently missing — needed for clean JSON serialization).
- Keep `encoding/json` as the only new import (stdlib only, per design decisions in PLAN.md).
- The `Lookup` package-level function can be removed once `main.go` uses `Load` + `Match` directly.

## Completion Notes (Agent 52e7c7f8)

All 5 plan steps implemented and verified:

1. **`Save(dir string) error`** on `*Index` — writes `index.json` (entries) and `meta.json` (root, scannedAt, version, fileCount, packageCount) to `<dir>/swarm/index/`, creating the directory if needed.
2. **`Load(dir string) (*Index, error)`** — reads both JSON files back, reconstructs `*Index` with `Root` from meta. Returns clear errors on missing files.
3. **`scan` command** — now calls `idx.Save(dir)` after scanning, prints `"Index saved to <dir>/swarm/index/ (N files, M packages)"`.
4. **`lookup` command** — replaced stub with `findIndexRoot` (walks up from CWD for `swarm/index/meta.json`) → `index.Load` → `idx.Match`. Removed the old `Lookup` package-level function.
5. **5 new tests** — `TestSaveLoad` (round-trip), `TestSaveMeta` (meta fields + RFC3339 timestamp), `TestSaveCreatesDirectory`, `TestLoadNonexistent`, `TestLookupIntegration`. All 8 tests pass.

Only stdlib imports used (`encoding/json`, `time`). JSON struct tags added to `Entry`.
