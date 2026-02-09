# `impact` command — blast radius analysis for a symbol or file

## Completion Notes

Implemented by agent ef7c9408. All features working:

- **New file: `index/impact.go`** — Types (ImpactTarget, ImpactRef, ImpactLayer, ImpactSummary, ImpactResult), `Impact()` method on Index with symbol and file modes, `findEnclosingSymbol()` for parser-based enclosing symbol detection, `FormatImpact()` for human-readable output.
- **Symbol mode**: Finds direct refs (depth 1), resolves enclosing symbols via parsers, then finds refs to those symbols (depth 2+). Cycle detection via visited map.
- **File mode**: Finds direct importers (depth 1) via `Related()`, then importers of importers (depth 2+). Cycle detection via visited map.
- **Wired in `main.go`**: `case "impact"` with `--root`, `--depth` (default 3), `--max` (default 100), and `--json` support.
- **Tests: `index/impact_test.go`** — 11 tests covering: direct refs, transitive chains, file mode, cycle detection, depth limiting, max results, JSON output, symbol not found, file not found, format output, and empty results.
- **Docs updated**: README.md (quick start, commands table, project structure), SKILL.md (agent usage examples).
- All tests pass (`go test ./...`).

## Problem

When an agent is about to modify a function, type, or file, it needs to understand the full blast radius: not just direct references, but the transitive chain of dependents. Currently, `refs` shows direct usages and `related` shows direct importers, but neither traces the chain recursively. An agent modifying `index.Load()` would see its 15 direct callers, but wouldn't know that those callers are themselves called from 40 other locations — meaning the true blast radius is much larger than it appears.

## Command

```
swarm-index impact <symbol-or-file> [--root <dir>] [--depth N] [--max N]
```

### Arguments

| Arg/Flag | Description |
|---|---|
| `<symbol-or-file>` | A symbol name (e.g. `Load`, `HandleAuth`) or a file path (e.g. `index/index.go`). If it looks like a file path (contains `/` or `.`), treat as file; otherwise treat as symbol. |
| `--root <dir>` | Project root (default: auto-detect) |
| `--depth N` | Maximum depth of transitive traversal (default 3). Depth 1 = direct refs only (like `refs`). Depth 2 = refs of refs. |
| `--max N` | Maximum total results to return (default 100) |

### Behavior

**Symbol mode** (e.g. `swarm-index impact Load`):
1. Find the symbol's definition using the index/parsers (like `refs` does).
2. Find all direct references to the symbol (depth 1).
3. For each referencing function, find references to *that* function (depth 2).
4. Continue until `--depth` is reached or no new dependents are found.
5. Deduplicate and report as a tree/layered list showing the transitive dependency chain.

**File mode** (e.g. `swarm-index impact index/index.go`):
1. Find all files that import this file (using `related`'s importer logic).
2. For each importer, find files that import *that* file (depth 2).
3. Continue until `--depth` is reached.
4. Report as a layered list.

### Output (text)

```
Impact analysis for symbol "Load" (index/index.go:45)

Depth 1 — direct references (8 files):
  main.go:97           idx, err := index.Load(root)
  main.go:155          idx, err := index.Load(root)
  index/stale.go:23    idx, err := Load(dir)
  index/symbols.go:15  idx, err := Load(root)
  ...

Depth 2 — transitive dependents (3 files):
  main.go              (already at depth 1)
  cmd/server.go:44     calls Symbols() which calls Load()

Total blast radius: 9 files, 14 call sites
```

### Output (JSON, with `--json`)

```json
{
  "target": {
    "name": "Load",
    "file": "index/index.go",
    "line": 45,
    "kind": "func"
  },
  "layers": [
    {
      "depth": 1,
      "label": "direct references",
      "refs": [
        {"file": "main.go", "line": 97, "content": "idx, err := index.Load(root)", "enclosingSymbol": "main"},
        ...
      ]
    },
    {
      "depth": 2,
      "label": "transitive dependents",
      "refs": [...]
    }
  ],
  "summary": {
    "totalFiles": 9,
    "totalCallSites": 14,
    "maxDepthReached": 2
  }
}
```

## Implementation

### New file: `index/impact.go`

```go
type ImpactTarget struct {
    Name string `json:"name"`
    File string `json:"file"`
    Line int    `json:"line"`
    Kind string `json:"kind"` // "func", "type", "file"
}

type ImpactLayer struct {
    Depth int          `json:"depth"`
    Label string       `json:"label"`
    Refs  []ImpactRef  `json:"refs"`
}

type ImpactRef struct {
    File            string `json:"file"`
    Line            int    `json:"line"`
    Content         string `json:"content"`
    EnclosingSymbol string `json:"enclosingSymbol,omitempty"`
}

type ImpactResult struct {
    Target  ImpactTarget  `json:"target"`
    Layers  []ImpactLayer `json:"layers"`
    Summary ImpactSummary `json:"summary"`
}

type ImpactSummary struct {
    TotalFiles    int `json:"totalFiles"`
    TotalRefSites int `json:"totalRefSites"`
    MaxDepth      int `json:"maxDepthReached"`
}
```

### Algorithm (symbol mode)

1. Use existing `idx.Refs(symbol, max)` to find depth-1 references.
2. Extract the enclosing function/symbol name from each reference site (use parser on that file, find which symbol's line range contains the reference line).
3. For each unique enclosing symbol found, recursively call `idx.Refs(enclosingSymbol, max)` to get depth-2 references.
4. Track visited symbols in a `map[string]bool` to avoid cycles.
5. Continue to `--depth` limit.
6. Aggregate into layers.

### Algorithm (file mode)

1. Use existing `idx.Related(file)` to get importers (depth 1).
2. For each importer file, call `idx.Related(importerFile)` to get its importers (depth 2).
3. Track visited files to avoid cycles.
4. Continue to `--depth` limit.
5. Aggregate into layers.

### Wire up in main.go

Add a `case "impact":` block following the standard pattern (resolve root, load index, parse flags, call method, format output).

### Tests: `index/impact_test.go`

- Test symbol mode with a known call chain (A calls B calls C — impact of C should show B at depth 1, A at depth 2).
- Test file mode with known import chains.
- Test cycle detection (A imports B imports A).
- Test depth limiting.
- Test `--max` result limiting.
- Test JSON output structure.

### Formatter: `FormatImpact()`

Human-readable layered output as shown above. Include summary line at the bottom.

## Dependencies

- Requires a prior `scan` (needs loaded index).
- Reuses `idx.Refs()` and `idx.Related()` internally.
- Reuses parsers for enclosing-symbol detection.

## Why this matters for agents

When an agent is tasked with modifying a function, it needs to know: "What could break?" Today it would need to manually chain multiple `refs` calls and mentally build the dependency tree. The `impact` command does this automatically, giving the agent a complete picture of the blast radius before making changes. This directly reduces the risk of introducing regressions and helps agents scope their testing appropriately.
