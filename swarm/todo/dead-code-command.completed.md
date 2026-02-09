# Feature: `dead-code` command — detect potentially unused exports and functions

## Problem

When an agent is refactoring, cleaning up, or trying to understand a codebase, one of the most common questions is: "Is this function/type/export actually used anywhere?" Currently, an agent would need to manually run `refs` on every symbol to check — which is tedious and doesn't scale. A dedicated `dead-code` command surfaces potentially unused symbols in one call, helping agents:

- Identify safe-to-remove code during cleanup
- Understand which public API surface is actually consumed
- Find orphaned helpers and utilities after refactors
- Reduce code complexity by removing dead paths

## Design

### CLI interface

```
swarm-index dead-code [--root <dir>] [--max N] [--kind KIND] [--path PREFIX] [--json]
```

- `--root <dir>`: Project root (auto-detected if omitted)
- `--max N`: Limit results (default 50)
- `--kind KIND`: Filter by symbol kind (func, type, const, var, class, interface)
- `--path PREFIX`: Only analyze symbols in files matching this prefix (e.g. `src/utils`)
- `--json`: Structured output

### How it works

1. **Load the index** (requires prior `scan`)
2. **Collect all exported/defined symbols** by parsing every parseable file in the index using the existing `parsers` package
3. **For each symbol, search for references** across all indexed files (reuse the word-boundary matching logic from `refs`)
4. **Filter out self-references** (the definition line itself doesn't count as a usage)
5. **Report symbols with zero external references** as potentially dead code

### Output

#### Text output
```
Dead code candidates (12 found):

  index/index.go:
    func FormatOldSummary          :145   (0 references)
    type LegacyEntry               :28    (0 references)

  parsers/jsparser.go:
    func parseOldStyle             :200   (0 references)

  ...
```

#### JSON output
```json
{
  "totalCandidates": 12,
  "candidates": [
    {
      "name": "FormatOldSummary",
      "kind": "func",
      "path": "index/index.go",
      "line": 145,
      "signature": "func FormatOldSummary(s *Summary) string",
      "exported": true,
      "references": 0
    }
  ]
}
```

### Exclusion rules

- Skip `main` functions (they're entry points, not called by other code)
- Skip `init` functions (called implicitly by Go runtime)
- Skip `Test*`, `Benchmark*`, `Example*` functions (test entry points)
- Skip symbols in test files (`_test.go`, `.test.ts`, `.spec.ts`, `test_*.py`)
- Only analyze exported/public symbols by default (unexported symbols used only within a file are harder to detect as truly dead without full type resolution)

## Implementation

### New file: `index/deadcode.go`

Add a `DeadCode` method to `*Index`:

```go
type DeadCodeCandidate struct {
    Name       string `json:"name"`
    Kind       string `json:"kind"`
    Path       string `json:"path"`
    Line       int    `json:"line"`
    Signature  string `json:"signature"`
    Exported   bool   `json:"exported"`
    References int    `json:"references"`
}

type DeadCodeResult struct {
    TotalCandidates int                 `json:"totalCandidates"`
    Candidates      []DeadCodeCandidate `json:"candidates"`
}

func (idx *Index) DeadCode(kind string, pathPrefix string, max int) (*DeadCodeResult, error)
```

Steps inside the method:
1. Iterate over all file entries in the index
2. For each parseable file (has a parser via `parsers.ForExtension`), parse to get symbols
3. For each exported symbol that isn't excluded (main, init, Test*, etc.):
   a. Search all indexed text files for word-boundary occurrences of the symbol name
   b. Subtract self-references (same file, same line)
   c. If zero external references, add to candidates list
4. Sort candidates by file path, then line number
5. Apply `--kind` filter and `--max` limit
6. Return result

### New file: `index/deadcode_test.go`

Test cases:
- A function defined and used → not reported
- A function defined but never referenced → reported as dead code
- `main` and `init` functions → excluded
- Test functions → excluded
- `--kind` filtering works
- `--path` prefix filtering works

### Wire up in `main.go`

Add `case "dead-code":` to the switch statement, following the same pattern as other commands.

### Update README.md

Add `dead-code` to the commands table and quick start section.

### Update SKILL.md

Add usage examples for `dead-code`.

## Dependencies

- Requires a prior `scan` (needs the index)
- Uses existing `parsers` package for symbol extraction
- Uses word-boundary search logic similar to `refs`

## Performance note

This command will be slower than most others because it needs to parse every file AND search every file for each symbol. For large codebases, consider:
- Caching parsed symbols (they're already computed during `outline`)
- Short-circuiting: once a reference is found, stop searching for that symbol
- Processing files in batches to avoid excessive memory use

The `--path` flag helps scope the analysis to a subdirectory for faster results on large projects.

## Completion Notes

Implemented by agent 21e6e471. All aspects of the design were implemented:

- **`index/deadcode.go`**: `DeadCode` method on `*Index`, `DeadCodeCandidate` and `DeadCodeResult` types, `FormatDeadCode` formatter, `countExternalRefs` with short-circuit optimization, `isTestFile` and `isExcludedSymbol` helpers.
- **`main.go`**: Added `dead-code` case to the command switch with `--root`, `--max`, `--kind`, `--path`, and `--json` flag support. Added to usage text.
- **`index/deadcode_test.go`**: 12 tests covering unused function detection, main/init exclusion, test function exclusion, test file skipping, kind filtering, path filtering, max limit, no-candidates case, multi-language support, and format output (empty, with results, truncated).
- **`README.md`**: Added quick start examples, commands table entry, and project structure entry.
- **`SKILL.md`**: Added usage examples for agents.

All tests pass (`go test ./...`). The command works correctly against the project itself.
