# `scope` Command — Directory/Package-level Summary

## Problem

Agents frequently need to understand a specific directory or package before modifying code in it. Currently:
- `summary` gives a whole-project overview (too broad)
- `outline` shows symbols for a single file (too narrow)
- `tree` shows structure but no content details

There's no single command to get a focused view of one directory: what files it contains, what symbols they define, how many lines of code, what it imports, and what it exports.

## Solution

Add a `scope <directory>` command that produces a focused summary of a single directory/package.

### Output includes:
1. **File list** — all files in the directory (non-recursive by default, `--recursive` to include subdirs)
2. **Symbol summary** — count of functions, types, classes, methods, etc. across all files
3. **LOC** — total lines of code in the directory
4. **Exports** — exported/public symbols defined in this directory
5. **Internal symbols** — non-exported symbols (count only)
6. **Dependencies** — what other directories/packages files in this scope import
7. **Dependents** — which other directories/packages import from this scope

### CLI interface:
```
swarm-index scope <directory> [--root <dir>] [--recursive] [--json]
```

### Flags:
- `--root <dir>` — project root (auto-detected if omitted)
- `--recursive` — include files in subdirectories
- `--json` — structured JSON output

### Example text output:
```
Scope: index/
  12 files, 1847 LOC

  Symbols:
    func     34 (28 exported, 6 internal)
    type     15 (12 exported, 3 internal)
    method   22 (18 exported, 4 internal)

  Dependencies (imports from):
    parsers/
    fmt, os, path/filepath (stdlib)

  Depended on by:
    (root)  [main.go]
```

### Example JSON output:
```json
{
  "directory": "index/",
  "files": ["blame.go", "complexity.go", "..."],
  "fileCount": 12,
  "loc": 1847,
  "symbols": {
    "func": {"exported": 28, "internal": 6},
    "type": {"exported": 12, "internal": 3},
    "method": {"exported": 18, "internal": 4}
  },
  "dependencies": ["parsers/"],
  "dependents": ["(root)"]
}
```

## Implementation Plan

### 1. Add `ScopeResult` type and `Scope()` method to `index/` package

Create `index/scope.go`:

- `ScopeResult` struct with fields for directory, files, fileCount, LOC, symbol counts, dependencies, dependents
- `(idx *Index) Scope(dir string, recursive bool) (*ScopeResult, error)` method that:
  1. Filters `idx.Entries` to entries whose path is within the given directory
  2. If not recursive, only include entries at the immediate level (no subdirectories)
  3. For each file, parse symbols using the appropriate parser from `parsers.ForExtension()`
  4. Count exported vs internal symbols by kind
  5. Count LOC by reading file contents
  6. Extract imports from each file and resolve to directory-level dependencies
  7. Scan all other files to find which directories import files from this scope
- `FormatScope(result *ScopeResult) string` for human-readable output

### 2. Wire up CLI in `main.go`

Add `case "scope":` that:
- Requires `<directory>` argument
- Parses `--recursive` boolean flag
- Loads index, calls `idx.Scope(dir, recursive)`
- Outputs text or JSON based on `--json`

### 3. Update usage text

Add `scope` to the `printUsage()` help text.

### 4. Add tests

Create `index/scope_test.go`:
- Test filtering entries to a specific directory
- Test recursive vs non-recursive mode
- Test symbol counting (exported vs internal)
- Test dependency and dependent detection

### 5. Update README.md and SKILL.md

Add `scope` command documentation and examples.

## Dependencies

- Requires a prior `scan` (loads index from disk)
- Uses `parsers.ForExtension()` for symbol extraction
- Reuses import extraction logic from `related.go`

## Effort

Medium — mostly combining existing functionality (entry filtering, symbol parsing, import extraction) into a new aggregated view.

## Completion Notes

Implemented by agent 0db6b93f (task 5cc84114).

### Files created:
- `index/scope.go` — `ScopeResult` type, `Scope()` method, `FormatScope()` formatter, `inScope()` and `sortedKeys()` helpers
- `index/scope_test.go` — 8 tests covering basic directory scoping, recursive/non-recursive modes, empty directories, trailing slash normalization, dependency detection, format output, and `inScope()` logic

### Files modified:
- `main.go` — Added `case "scope":` CLI handler with `--recursive` flag support, added `hasBoolFlag()` helper, updated `printUsage()` with scope command
- `README.md` — Added scope command examples, command table entry, and project structure entry
- `SKILL.md` — Added scope command usage examples

### All tests pass (`go test ./...`)
