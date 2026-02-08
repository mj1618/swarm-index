# `refs <symbol>` Command

## Summary

Add a `refs` command that finds all usages/references of a named symbol across the indexed codebase. This is Phase 8 from PLAN.md and is critical for coding agents that need to understand the impact of changes before modifying code.

## Motivation

When an agent needs to rename a function, change a type signature, or understand how widely a symbol is used, it currently has to manually `search` for the name and mentally filter results. The `refs` command automates this: given a symbol name, it finds the definition and all call sites / usage sites, grouped by file.

## Design

### CLI interface

```
swarm-index refs <symbol> [--root <dir>] [--max N] [--json]
```

- `<symbol>` — the symbol name to search for (e.g. `HandleAuth`, `Entry`, `Scan`)
- `--root <dir>` — project root (auto-detected if omitted)
- `--max N` — max references to return (default 50)
- `--json` — structured JSON output

### Algorithm

1. Load the persisted index from `swarm/index/index.json`.
2. Search index entries for the symbol definition — match entries where `entry.Name == symbol` (exact, case-sensitive). This finds both file entries and any symbol entries if symbol indexing is added later.
3. Grep all indexed files for occurrences of the symbol name using a word-boundary regex (`\b<symbol>\b`), collecting file path, line number, and the matching line content.
4. Classify each match:
   - **definition** — the line where the symbol is defined (matches the index entry's line number and path, or detected via heuristics like `func <symbol>`, `type <symbol>`, `class <symbol>`, `def <symbol>`)
   - **reference** — all other occurrences
5. Return results grouped by file, with the definition listed first.

### Output structures

#### Text output
```
Definition:
  index/index.go:146  func Scan(root string) (*Index, error)

References (12 matches):
  main.go:59           idx, err := index.Scan(dir)
  main_test.go:23      idx, _ := index.Scan(tmpDir)
  main_test.go:45      _, err := index.Scan("/nonexistent")
  index/search.go:27   // uses entries from Scan
  ...
```

#### JSON output
```json
{
  "symbol": "Scan",
  "definition": {
    "path": "index/index.go",
    "line": 146,
    "content": "func Scan(root string) (*Index, error)"
  },
  "references": [
    {"path": "main.go", "line": 59, "content": "idx, err := index.Scan(dir)"},
    ...
  ],
  "totalReferences": 12
}
```

### Implementation plan

#### 1. Add `Refs` method to `*Index` in a new file `index/refs.go`

```go
type RefMatch struct {
    Path       string `json:"path"`
    Line       int    `json:"line"`
    Content    string `json:"content"`
    IsDefinition bool `json:"isDefinition"`
}

type RefsResult struct {
    Symbol      string     `json:"symbol"`
    Definition  *RefMatch  `json:"definition"`
    References  []RefMatch `json:"references"`
    TotalRefs   int        `json:"totalReferences"`
}

func (idx *Index) Refs(symbol string, maxResults int) (*RefsResult, error)
```

- Build a word-boundary regex: `\b` + `regexp.QuoteMeta(symbol)` + `\b`
- Iterate unique file paths from the index
- For each file, scan lines for regex matches
- Heuristically detect definitions (lines matching patterns like `func <symbol>`, `type <symbol> struct`, `var <symbol>`, `const <symbol>`, `class <symbol>`, `def <symbol>`)
- Also check if any index entry has `Name == symbol` and a non-zero `Line` — use that as the authoritative definition location
- Collect all non-definition matches as references, up to `maxResults`

#### 2. Add `refs` case to `main.go` command switch

Wire up the CLI: parse args, call `idx.Refs(symbol, max)`, format output (text or JSON).

#### 3. Add tests in `index/refs_test.go`

- Create a temp directory with multiple Go files containing a function defined in one and called in others
- Test that the definition is correctly identified
- Test that references are found across files
- Test `--max` limiting
- Test that the symbol's own definition line is excluded from references
- Test with a symbol that doesn't exist (should return empty results, not an error)

#### 4. Update README.md

- Move `refs` from roadmap to commands table
- Add usage example

#### 5. Update SKILL.md

- Add `refs` command to the agent skill reference

## Dependencies

- Requires persisted index (Phase 1 — completed)
- Benefits from outline/symbol data but works without it (falls back to regex heuristics for definition detection)
- Reuses `searchFile`-style logic from `index/search.go`

## Files to create/modify

| File | Action |
|---|---|
| `index/refs.go` | Create — `Refs` method and types |
| `index/refs_test.go` | Create — tests |
| `main.go` | Modify — add `refs` command case |
| `README.md` | Modify — document command |
| `SKILL.md` | Modify — add command reference |

## Completion Notes

Implemented by agent dd5c8cd4. All 6 tests pass (TestRefsFindsDefinitionAndReferences, TestRefsMaxResults, TestRefsSymbolNotFound, TestRefsAcrossMultipleFiles, TestRefsDefinitionExcludedFromRefs, TestRefsPythonDef). Full test suite (73 tests) passes. Files created/modified:

- `index/refs.go` — Created with `RefMatch`, `RefsResult` types and `Refs()` method on `*Index`. Uses word-boundary regex matching and heuristic definition detection (supports Go func/type/var/const, Python def, JS function/class/let/const, Java/TS interface/enum).
- `index/refs_test.go` — Created with 6 tests covering definition detection, max results, missing symbols, cross-file references, definition exclusion from refs, and Python def support.
- `main.go` — Added `refs` command case with `--root`, `--max`, and `--json` flag support. Added to usage text.
- `README.md` — Added refs to commands table, quick start examples, project structure, and marked as done in roadmap.
- `SKILL.md` — Added refs command examples.
