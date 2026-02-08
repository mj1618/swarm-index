# `related <file>` Command

## Summary

Add a `related` command that shows files connected to a given file. When an agent is working on a file, this command instantly answers: "What does this file depend on? What depends on it? Where are its tests?" — saving significant exploration time.

## Motivation

Agents frequently need to understand the dependency neighborhood of a file they're editing. Today they must manually grep for imports and test files. The `related` command automates this into a single call, returning three groups: imports (files the target depends on), importers (files that depend on the target), and associated test files.

## Design (from PLAN.md Phase 10)

### Input

```
swarm-index related <file> [--root <dir>] [--json]
```

- `<file>` — path to the target file (relative or absolute)
- `--root <dir>` — project root override (default: auto-detect via `swarm/index/meta.json`)
- `--json` — structured JSON output

### Output Structure

Return three groups:

```go
type RelatedResult struct {
    File      string   `json:"file"`      // the target file path
    Imports   []string `json:"imports"`   // files this file imports/requires
    Importers []string `json:"importers"` // files that import/require this file
    TestFiles []string `json:"testFiles"` // associated test files
}
```

### Implementation Steps

#### 1. Add `related.go` in the `index` package

Create `index/related.go` with a `Related(filePath string) (*RelatedResult, error)` method on `*Index`.

**Import detection** — parse the target file's content to extract import paths:
- **Go**: Parse `import (...)` blocks and single `import "..."` statements. Match import paths against the index to find local files (ignore stdlib/external packages). Use `go/parser` or simple regex like `import\s+"([^"]+)"` and `"([^"]+)"` within import blocks.
- **JS/TS**: Match `import ... from '...'` and `require('...')` patterns. Resolve relative paths (starting with `./` or `../`) against the file's directory and match to index entries.
- **Python**: Match `from X import ...` and `import X` patterns. Convert dotted module paths to file paths and match against the index.

**Importer detection** — search all indexed files for import statements that reference the target file:
- For each file in the index, read its content and check if any of its imports resolve to the target file path.
- To keep this fast, only scan files with extensions that have known import syntax (`.go`, `.js`, `.ts`, `.jsx`, `.tsx`, `.py`).

**Test file detection** — use naming conventions:
- Go: `<name>_test.go` in the same directory
- JS/TS: `<name>.test.{js,ts,jsx,tsx}`, `<name>.spec.{js,ts,jsx,tsx}`, or files in a `__tests__/` directory with matching names
- Python: `test_<name>.py` or `<name>_test.py` in the same directory, or under a `tests/` directory

#### 2. Add `FormatRelated` function

Create a text formatter in `index/related.go`:

```
Related files for main.go:

Imports (3):
  index/index.go
  parsers/parsers.go
  parsers/goparser.go

Imported by (1):
  main_test.go

Test files (1):
  main_test.go
```

#### 3. Wire up in `main.go`

Add a `case "related":` block in the switch statement:
- Parse `<file>` argument (required)
- Resolve root via `--root` or auto-detect
- Load index
- Call `idx.Related(filePath)`
- Output text or JSON based on `--json` flag

#### 4. Update `printUsage()` in `main.go`

Add the related command to the usage text.

#### 5. Add tests in `index/related_test.go`

- Test import extraction for Go, JS/TS, and Python files
- Test importer detection (create temp files that import each other)
- Test test-file detection by naming convention
- Test with a file that has no relations (empty result)

#### 6. Update README.md

- Add `related` to the commands table
- Add usage example to Quick Start
- Mark `related` as completed in the Roadmap

#### 7. Update SKILL.md

Add the `related` command to the agent instructions.

## Dependencies

- Requires a persisted index (Phase 1 — already completed)
- Uses `openTextFile` from `index/index.go` for reading file contents
- Leverages `FilePaths()` for iterating indexed files

## Edge Cases

- Binary files should be skipped when scanning for importers
- External/stdlib imports should be filtered out (only show local project files)
- If the target file is not in the index, return an error
- Handle files with no imports, no importers, and no tests gracefully (empty arrays, not nil)

## Completion Notes

Implemented by agent a1af0e6b (task d0be0049).

### Files created/modified:
- **index/related.go** — Core implementation with `Related()` method, import extraction (Go/JS/TS/Python), importer detection, test file detection, and `FormatRelated()` text formatter
- **index/related_test.go** — 15 tests covering Go/JS/TS/Python imports, importers, test file detection, edge cases (non-existent files, non-importable files, no relations), and format output
- **main.go** — Wired up `case "related":` command with `--root` and `--json` flag support, added to `printUsage()`
- **README.md** — Added to commands table, quick start example, project structure, and marked as completed in roadmap
- **SKILL.md** — Added related command usage examples

### All tests pass (go test ./...)
