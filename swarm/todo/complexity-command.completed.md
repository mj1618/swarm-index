# Feature: `complexity` Command

## Summary

Add a `complexity` command that analyzes code complexity metrics per function/method across the project. This helps coding agents identify high-risk, hard-to-maintain code that needs careful modification, and prioritize which files to review most carefully before making changes.

## Motivation

When an agent is about to modify code, it needs to know which functions are the riskiest — long functions with deep nesting are harder to change safely. Currently no command provides this insight. The `outline` command shows function signatures but not how complex the bodies are. The `summary` command shows LOC totals but not per-function metrics. This command fills that gap.

## Usage

```bash
# Show complexity report for the whole project (top N most complex functions)
swarm-index complexity [--root <dir>] [--max N]

# Show complexity for a specific file
swarm-index complexity <file>

# Filter by minimum complexity threshold
swarm-index complexity --min 5

# JSON output for agent consumption
swarm-index complexity --json
```

## Output (text mode)

```
Complexity Report (top 20 most complex functions):

  main.go:43      main()                    complexity=12  lines=574  depth=4
  index/graph.go:85  buildGraph()           complexity=9   lines=120  depth=5
  index/search.go:12 Search()               complexity=7   lines=45   depth=3
  parsers/jsparser.go:30 parseJS()          complexity=6   lines=90   depth=4
  ...

20 functions shown (use --max to see more)
```

## Output (JSON mode)

```json
{
  "functions": [
    {
      "path": "main.go",
      "name": "main",
      "line": 43,
      "endLine": 617,
      "complexity": 12,
      "lines": 574,
      "maxDepth": 4,
      "params": 0
    }
  ],
  "summary": {
    "totalFunctions": 150,
    "avgComplexity": 3.2,
    "maxComplexity": 12,
    "highComplexityCount": 5
  }
}
```

## Implementation

### 1. New file: `index/complexity.go`

#### ComplexityResult types

```go
type FunctionComplexity struct {
    Path       string `json:"path"`
    Name       string `json:"name"`
    Line       int    `json:"line"`
    EndLine    int    `json:"endLine"`
    Complexity int    `json:"complexity"`
    Lines      int    `json:"lines"`
    MaxDepth   int    `json:"maxDepth"`
    Params     int    `json:"params"`
    Signature  string `json:"signature"`
}

type ComplexityResult struct {
    Functions          []FunctionComplexity `json:"functions"`
    TotalFunctions     int                  `json:"totalFunctions"`
    AvgComplexity      float64              `json:"avgComplexity"`
    MaxComplexity      int                  `json:"maxComplexity"`
    HighComplexityCount int                 `json:"highComplexityCount"` // complexity >= 10
}
```

#### Complexity calculation approach

Use a language-aware but lightweight approach:

**For Go files** (using `go/ast`):
- Count branching statements: `if`, `else if`, `for`, `switch`, `case`, `select`, `&&`, `||` within each function
- Track nesting depth via AST walk
- Get parameter count from function signature
- Line count from `EndLine - Line + 1`

**For Python/JS/TS files** (regex-based heuristic):
- Count lines matching `if `, `elif `, `else`, `for `, `while `, `case `, `catch `, `&&`, `||`, `? ` (ternary)
- Track indentation depth changes for nesting
- Function boundaries from the parser's Symbol data (Line to EndLine)

#### Methods

```go
// Complexity analyzes all parseable files and returns complexity metrics
// for each function, sorted by complexity descending.
func (idx *Index) Complexity(file string, maxResults int, minComplexity int) (*ComplexityResult, error)

// FormatComplexity renders the result as human-readable text.
func FormatComplexity(result *ComplexityResult) string
```

### 2. Wire up in `main.go`

Add a `case "complexity":` block in the switch. Parse `--max` (default 20), `--min` (default 0), optional file argument, `--root`, and `--json`.

### 3. Tests: `index/complexity_test.go`

- Test Go complexity calculation against a sample Go file with known branching
- Test Python complexity calculation against a sample Python file
- Test JS/TS complexity calculation against a sample JS file
- Test sorting (highest complexity first)
- Test `--min` threshold filtering
- Test single-file mode vs project-wide mode
- Test JSON output structure

### 4. Update README.md

Add `complexity` to the Commands table and usage examples.

### 5. Update SKILL.md

Add `complexity` usage examples.

## Dependencies

- Requires a prior `scan` (for project-wide mode) to know which files to analyze
- Uses existing parsers for function boundary detection (Line/EndLine from Symbol)
- Go files: uses `go/ast` (already a dependency via goparser)
- Python/JS/TS: regex-based heuristic (no new dependencies)

## Complexity threshold guidance (for docs)

| Complexity | Risk Level |
|---|---|
| 1-5        | Low — straightforward |
| 6-10       | Moderate — review carefully |
| 11-20      | High — consider refactoring |
| 21+        | Very high — major risk |

## Completion Notes

Implemented by agent 28342378. All items completed:

1. **`index/complexity.go`** — Types (`FunctionComplexity`, `ComplexityResult`), `Complexity()` method on Index, `ComplexityFile()` for single-file mode, `FormatComplexity()` for text output. Go files use `go/ast` for accurate cyclomatic complexity counting (if, for, switch, case, select, &&, ||). Python/JS/TS use regex-based heuristic with parser-provided function boundaries.
2. **`main.go`** — Added `case "complexity":` with `--max` (default 20), `--min` (default 0), optional file argument, `--root`, and `--json` support. Updated usage text.
3. **`index/complexity_test.go`** — 13 tests covering: Go basic complexity, sorting, min threshold, single-file mode, max results, Go methods, Python analysis, JS analysis, JSON structure, Go params counting, high complexity count, and format tests.
4. **README.md** — Added `complexity` to commands table, usage examples, and project structure.
5. **SKILL.md** — Added `complexity` usage examples.

All tests pass (`go test ./...`).
