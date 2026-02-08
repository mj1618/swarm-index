# `exports` Command

## Summary

Add an `exports <file|directory>` command that lists the public API surface of a file or package/directory. This is Phase 9 from the PLAN.

## Motivation

Agents frequently need to understand what a file or package exposes publicly — exported functions, types, constants, and variables. Currently they must use `outline` on individual files and manually filter for exported symbols. The `exports` command surfaces this in a single call, making it trivial to answer questions like "what does this package export?" or "what's the public API of this module?"

## Implementation

### 1. Add `Exports` method to `*Index` in `index/exports.go`

```go
type ExportsResult struct {
    Scope   string   `json:"scope"`   // file path or directory queried
    Symbols []Entry  `json:"symbols"` // exported symbols
    Count   int      `json:"count"`
}
```

- Accept a `scope` argument (file path or directory path).
- If `scope` is a file: filter `idx.Entries` to entries where `Path == scope` and `Kind != "file"` and the symbol name starts with an uppercase letter (Go convention) or the entry was marked exported by the parser.
- If `scope` is a directory/package: filter entries where the directory portion of `Path` matches the scope.
- The parsers already set `Exported` on symbols, but this info is not currently persisted in `Entry`. Two approaches:
  - **Option A (preferred):** Add an `Exported bool` field to `Entry` and populate it during scan from `Symbol.Exported`. Then filter on `e.Exported == true`.
  - **Option B (simpler, no schema change):** Re-parse the file at query time using the parser and filter `Symbol.Exported`. This avoids changing the index format but is slower.

Recommend Option A since the `Entry` struct already contains `Kind`, `Line`, etc. Adding `Exported` is a natural extension.

### 2. Add `Exported` field to `Entry` struct

In `index/index.go`, add:
```go
Exported bool `json:"exported,omitempty"`
```

### 3. Populate `Exported` during scan

In the `Scan` function (or wherever symbols are added to the index during scan), set `Exported` from `Symbol.Exported` when creating symbol entries. Currently, `Scan` only creates file-level entries — symbol entries come from `outline`. If symbols aren't stored in the index during scan, the `exports` command should:
1. First check if symbol entries exist in the index for the given scope.
2. If not, parse the files on the fly using `parsers.ForExtension()`.

### 4. Add `FormatExports` function

Text output format:
```
Exports for index/index.go:
  func  Scan(root string) (*Index, error)          :157
  func  Load(dir string) (*Index, error)            :130
  type  Entry                                        :14
  type  Index                                        :30

4 exported symbols
```

### 5. Wire up CLI in `main.go`

```
case "exports":
    if len(args) < 3 {
        fatal(jsonOutput, "usage: swarm-index exports <file|directory> [--root <dir>]")
    }
    scope := args[2]
    // ... resolve root, load index, call Exports, format output
```

- Support `--json` flag (already global).
- Support `--root` flag for specifying index root.

### 6. Update `printUsage()` in `main.go`

Add the exports command to the usage text.

### 7. Update README.md and SKILL.md

Add `exports` command to documentation.

### 8. Tests

In `index/exports_test.go`:
- Test filtering by single file scope.
- Test filtering by directory/package scope.
- Test that only exported symbols are returned.
- Test with file that has no exports.
- Test JSON output format.

## Dependencies

- Parsers package (already exists with `Exported` field on `Symbol`).
- Index persistence (already implemented).

## Scope

- Supported languages: Go, JavaScript/TypeScript, Python (all have parsers already).
- Go: uppercase names are exported.
- JS/TS: `export` keyword marks exports.
- Python: names not starting with `_` are conventionally public.

## Completion Notes

Implemented by agent fabdecba. All items completed:

1. **Added `Exported bool` field to `Entry` struct** in `index/index.go` (Option A from the plan).
2. **Created `index/exports.go`** with:
   - `ExportedSymbol` struct for individual exported symbols
   - `ExportsResult` struct for the query result
   - `Exports(scope)` method on `*Index` that parses files on-the-fly using existing parsers, filtering for `Symbol.Exported == true`. Supports both file and directory scopes.
   - `FormatExports()` function for human-readable text output, with multi-file grouping for directory scopes.
3. **Wired up CLI** in `main.go` with `case "exports":` supporting `--root` and `--json` flags.
4. **Updated `printUsage()`** to include the exports command.
5. **Created `index/exports_test.go`** with 12 tests covering:
   - Go file exports (exported vs unexported)
   - Directory/package scope
   - No exports case
   - Non-existent scope
   - Unsupported file extension
   - Symbol field verification
   - JavaScript/TypeScript exports
   - Python exports (public vs `_private`)
   - FormatExports output (empty, single-file, multi-file)
   - JSON output structure
6. **Updated README.md**: quick start examples, commands table, project structure, and roadmap checkbox.
7. **Updated SKILL.md**: added exports usage examples.
8. **All tests pass**: `go test ./...` succeeds across all packages.
