# Feature: `graph` Command — Project-Wide Import Dependency Graph

## Completion Note

Implemented by agent 7b525975. All files created/modified:

- `index/graph.go` — New file: `Graph()`, `GraphFocused()`, `FormatGraph()`, `FormatGraphDOT()` with `GraphResult`, `GraphNode`, `GraphEdge`, `GraphStats` types
- `index/graph_test.go` — New file: 10 tests covering Go/JS projects, empty graphs, focused subgraphs with depth limiting, DOT output format, JSON structure, and error handling
- `main.go` — Added `graph` subcommand with `--root`, `--format`, `--focus`, `--depth`, `--json` flags; updated usage text
- `README.md` — Added graph command documentation, usage examples, project structure entry, and roadmap entry
- `SKILL.md` — Added graph usage examples

All tests pass (go test ./...).

## Problem

The `related` command shows imports/importers for a single file, but agents frequently need to understand the full dependency structure of a project before making changes. Today, an agent must call `related` on every file individually to piece together the graph — expensive and error-prone.

A `graph` command gives agents instant visibility into:
- Which files are the most depended-upon (high fan-in = risky to change)
- Which files depend on the most others (high fan-out = complex/coupled)
- Dependency chains between two files (impact analysis)
- Clusters of tightly-coupled files (logical modules)

## Command Signature

```
swarm-index graph [--root <dir>] [--format dot|list] [--focus <file>] [--depth N] [--json]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--root <dir>` | auto-detect | Project root directory |
| `--format dot\|list` | `list` | Output format: `list` for text summary, `dot` for Graphviz DOT |
| `--focus <file>` | (none) | Show only the subgraph reachable from this file (both directions) |
| `--depth N` | unlimited | When used with `--focus`, limit traversal depth |
| `--json` | false | Structured JSON output |

## Implementation Plan

### 1. Add `graph.go` in `index/`

Create a `Graph` method on `*Index` that:
1. Iterates over all unique file paths in the index
2. For each importable file (Go, JS/TS, Python), calls the existing `extractImports` to get its dependencies
3. Builds an adjacency list: `map[string][]string` (file -> files it imports)
4. Computes fan-in (number of importers) and fan-out (number of imports) for each file
5. Returns a `GraphResult` struct

```go
type GraphEdge struct {
    From string `json:"from"`
    To   string `json:"to"`
}

type GraphNode struct {
    Path   string `json:"path"`
    FanIn  int    `json:"fanIn"`  // number of files that import this file
    FanOut int    `json:"fanOut"` // number of files this file imports
}

type GraphResult struct {
    Nodes []GraphNode `json:"nodes"` // sorted by fanIn desc
    Edges []GraphEdge `json:"edges"`
    Stats GraphStats  `json:"stats"`
}

type GraphStats struct {
    TotalFiles     int `json:"totalFiles"`     // files with import relationships
    TotalEdges     int `json:"totalEdges"`     // total import edges
    MostImported   string `json:"mostImported"`   // highest fan-in file
    MostDependent  string `json:"mostDependent"`  // highest fan-out file
}
```

When `--focus <file>` is provided, use BFS from that file in both directions (imports and importers) to extract the relevant subgraph, limited by `--depth`.

### 2. Add `FormatGraph` and `FormatGraphDOT` helpers

**List format** (default text output):
```
Import graph (42 files, 87 edges):

Most imported (highest fan-in):
  index/index.go          <- 12 files
  parsers/parsers.go      <- 8 files
  utils/config.go         <- 5 files

Most dependencies (highest fan-out):
  main.go                 -> 15 files
  cmd/serve.go            -> 9 files
  index/related.go        -> 7 files

All edges:
  main.go -> index/index.go
  main.go -> parsers/parsers.go
  ...
```

**DOT format** (for Graphviz visualization):
```dot
digraph imports {
  rankdir=LR;
  "main.go" -> "index/index.go";
  "main.go" -> "parsers/parsers.go";
  ...
}
```

### 3. Wire up in `main.go`

Add a `graph` subcommand that:
1. Loads the index (requires prior `scan`)
2. Calls `idx.Graph()` (or `idx.GraphFocused(file, depth)`)
3. Outputs in the requested format

### 4. Tests in `graph_test.go`

- Test graph construction with a known set of files and imports
- Test fan-in/fan-out computation
- Test `--focus` subgraph extraction with depth limiting
- Test DOT output format
- Test JSON output structure
- Test with no importable files (empty graph)

## Dependencies

- Requires a prior `scan` (reads from persisted index)
- Reuses existing `extractImports`, `resolveImport`, and the importable-extensions infrastructure from `related.go`
- No new external dependencies

## Changes Required

| File | Change |
|---|---|
| `index/graph.go` | New file: `Graph()`, `GraphFocused()`, `FormatGraph()`, `FormatGraphDOT()` |
| `index/graph_test.go` | New file: tests |
| `main.go` | Add `graph` subcommand |
| `README.md` | Add `graph` command documentation |
| `SKILL.md` | Add `graph` usage examples |
