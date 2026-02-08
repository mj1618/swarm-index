# Refactoring: graph.go code deduplication

## Changes applied

Extracted two shared helpers from the duplicated logic in `Graph()` and `GraphFocused()`:

### `buildAdjacency(indexedPaths)` (new helper)
Both methods independently built the same forward adjacency map by iterating all indexed paths and calling `extractImports`. This is now a single shared method.

### `buildGraphResult(edges)` (new helper)
Both methods duplicated identical logic for:
- Computing fan-in/fan-out counts from edges
- Building the node list from participating files
- Sorting nodes (by fan-in desc, path asc) and edges (by from, then to)
- Computing stats (most imported, most dependent)
- Nil-to-empty-slice normalization

This is now a single shared function that takes a slice of edges and returns the complete `*GraphResult`.

### Also cleaned up in `GraphFocused`
- Named the anonymous BFS queue struct as `bfsEntry` type for clarity
- Reused `buildAdjacency` instead of duplicating the adjacency-building loop
- Simplified the reverse adjacency construction to iterate `forward` (the result of `buildAdjacency`) rather than re-calling `extractImports`

## Net result
- `graph.go` reduced from 333 lines to 263 lines (-70 lines, ~21% reduction)
- All 7 graph tests pass
- Full test suite passes
- No behavioral changes
