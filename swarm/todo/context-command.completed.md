# Feature: `context <symbol> <file>` Command

## Problem

Agents frequently need to understand a symbol's full definition — not just its signature (from `outline`) or the raw file (from `show`), but the complete context: doc comments, the full function/type body, and the file's imports. Today an agent must chain `outline` + `show --lines` + manual parsing to assemble this. A single `context` command eliminates multiple round-trips.

## Behavior

```
swarm-index context <symbol> <file> [--root <dir>]
```

Given a symbol name and a file path, extract:

1. **Imports** — all import statements from the file (Go `import (...)`, Python `import`/`from`, JS/TS `import`).
2. **Doc comments** — contiguous comment lines immediately preceding the symbol's definition line.
3. **Definition body** — the full source from the symbol's `Line` to `EndLine` (inclusive), as reported by the parser.

### Output (text)

```
File: index/index.go
Imports:
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "strings"
  "time"

// Save writes the index to disk under <dir>/swarm/index/.
func (idx *Index) Save(dir string) error {
    indexDir := filepath.Join(dir, "swarm", "index")
    ...
}
```

### Output (--json)

```json
{
  "file": "index/index.go",
  "symbol": "Save",
  "kind": "method",
  "line": 97,
  "endLine": 116,
  "signature": "func (idx *Index) Save(dir string) error",
  "imports": ["encoding/json", "fmt", "os", "path/filepath", "strings", "time"],
  "docComment": "// Save writes the index to disk under <dir>/swarm/index/.",
  "body": "func (idx *Index) Save(dir string) error {\n\tindexDir := ..."
}
```

## Implementation Plan

### 1. Add `ContextResult` type and `Context()` function in `index/context.go`

```go
type ContextResult struct {
    File       string   `json:"file"`
    Symbol     string   `json:"symbol"`
    Kind       string   `json:"kind"`
    Line       int      `json:"line"`
    EndLine    int      `json:"endLine"`
    Signature  string   `json:"signature"`
    Imports    []string `json:"imports"`
    DocComment string   `json:"docComment"`
    Body       string   `json:"body"`
}
```

Function `Context(filePath string, symbolName string) (*ContextResult, error)`:

- Read the file.
- Get the parser via `parsers.ForExtension`.
- Parse symbols, find the one matching `symbolName` (exact, case-sensitive).
- If multiple matches (e.g. overloaded or same-named in different scopes), return the first top-level match.
- Read lines `Line` to `EndLine` from the file for the body.
- Walk backwards from `Line-1` collecting contiguous comment lines for the doc comment.
- Extract imports using a simple language-aware regex scan:
  - Go: `import "..."` and `import (...)` blocks
  - Python: `import ...` and `from ... import ...` lines
  - JS/TS: `import ... from "..."` lines
- Assemble and return `ContextResult`.

### 2. Add `FormatContext()` in `index/context.go`

Text formatter that prints the result in the human-readable format shown above.

### 3. Wire up CLI in `main.go`

- New `case "context":` in the switch.
- Parse args: `<symbol>` (required), `<file>` (required), optional `--root`.
- Call `index.Context(filePath, symbol)`.
- Output via JSON or text.

### 4. Add `index/context_test.go`

- Test with a Go file: verify imports, doc comment, and body extraction.
- Test with a Python file: verify Python import extraction.
- Test with unknown symbol: verify error.
- Test with file that has no parser: verify graceful error.

### 5. Update README.md, SKILL.md, and `printUsage()`

- Add `context` to the commands table.
- Add usage examples.
- Mark `context` as completed in the roadmap checklist.

## Flags

| Flag | Description |
|---|---|
| `--root <dir>` | Project root (for resolving relative paths) |
| `--json` | Structured JSON output |

## Dependencies

- Requires the existing `parsers` package (already implemented for Go, Python, JS/TS).
- Requires `parsers.Symbol.Line` and `EndLine` to be populated (already done by all parsers).
- No new external dependencies.

## Completion Notes

Implemented by agent a61de39c (task 1808c8b4).

- Created `index/context.go` with `ContextResult` type, `Context()` function, `FormatContext()` formatter, and language-aware import/doc-comment extraction helpers.
- `Context()` is a standalone function (not an `*Index` method) since it operates directly on a file path — no index required.
- Reuses the existing import regex patterns from `related.go` (`goImportSingle`, `goImportBlock`, etc.) to avoid duplication.
- Added `case "context":` in `main.go` with `<symbol> <file>` positional args and optional `--root` flag.
- Added `index/context_test.go` with 10 tests: Go/Python/JS function context, unknown symbol, unsupported extension, no doc comment, Go single import, file not found, and two FormatContext tests.
- Updated README.md (commands table, quick start examples, project structure, roadmap checkbox) and SKILL.md (usage examples).
- All tests pass (`go test ./...`).
