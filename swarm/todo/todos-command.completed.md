# `todos` Command — Collect TODO/FIXME/HACK/XXX Comments

## Summary

Add a `todos` command that scans indexed files for TODO, FIXME, HACK, and XXX comments, returning them with file paths, line numbers, and surrounding context. This gives agents an instant list of known issues, incomplete features, and technical debt markers left by developers.

## Priority

Medium — high value for agent orientation, no dependencies on unimplemented features.

## Motivation

When an agent starts working on a project, knowing what the developers themselves have flagged as incomplete or problematic is extremely valuable. TODO comments are a direct signal from humans about what needs attention. This command surfaces all such markers across the codebase in one call, saving agents from having to grep manually.

## Design

### CLI Interface

```bash
# Scan for all TODO/FIXME/HACK/XXX comments
swarm-index todos [--root <dir>] [--max N] [--tag TAG]

# Filter by tag
swarm-index todos --tag FIXME

# Limit results
swarm-index todos --max 20

# JSON output
swarm-index todos --json
```

### Data Structures

Add to `index/todos.go`:

```go
// TodoComment represents a single TODO/FIXME/HACK/XXX comment found in source.
type TodoComment struct {
    Path    string `json:"path"`
    Line    int    `json:"line"`
    Tag     string `json:"tag"`     // "TODO", "FIXME", "HACK", "XXX"
    Message string `json:"message"` // Text after the tag
    Content string `json:"content"` // Full line content (trimmed)
}

// TodosResult holds the collected TODO comments.
type TodosResult struct {
    Comments []TodoComment `json:"comments"`
    Total    int           `json:"total"`
    ByTag    map[string]int `json:"byTag"` // Count per tag
}
```

### Implementation

1. **`index/todos.go`** — Core logic:
   - `func (idx *Index) Todos(tag string, maxResults int) (*TodosResult, error)`
   - Iterate over indexed file paths (reuse `idx.FilePaths()`)
   - Open each file, scan line by line using a regex like `(?i)\b(TODO|FIXME|HACK|XXX)\b[:\s]*(.*)` to detect comment markers
   - Skip binary files (reuse `openTextFile` helper)
   - If `--tag` is specified, filter to just that tag
   - Respect `--max` limit (default 100)
   - Return results sorted by file path, then line number
   - Compute `ByTag` counts for the summary

2. **`index/todos_test.go`** — Tests:
   - Test detection of each tag type (TODO, FIXME, HACK, XXX)
   - Test case-insensitive matching (e.g., `todo:` and `TODO:`)
   - Test message extraction (text after the tag)
   - Test `--tag` filtering
   - Test `--max` limiting
   - Test that binary files are skipped

3. **`main.go`** — Wire up the CLI:
   - Add `case "todos":` to the command switch
   - Parse `--root`, `--max`, and `--tag` flags
   - Load index, call `idx.Todos(tag, max)`
   - Format output (text or JSON)

4. **Text output format**:
   ```
   TODO comments (42 found):

     index/index.go:45  TODO: add fuzzy matching support
     index/refs.go:12   FIXME: handle edge case for anonymous functions
     main.go:88         HACK: workaround for flag parsing limitation

   Summary: 30 TODO, 8 FIXME, 3 HACK, 1 XXX
   ```

5. **`FormatTodos(r *TodosResult) string`** — Human-readable formatter following the same pattern as `FormatStale` and `FormatSummary`.

### Regex Pattern

Use a pattern that matches common TODO comment styles across languages:

```
(?i)\b(TODO|FIXME|HACK|XXX)\b[:\s]*(.*)
```

This handles:
- `// TODO: do something` (Go, JS, Java, C)
- `# TODO do something` (Python, Ruby, Shell)
- `// FIXME(username): broken thing` (Go convention)
- `/* HACK: workaround */` (CSS, C)
- `-- XXX: review this` (SQL, Lua)

The parenthetical author `(username)` should be included in the message, not stripped.

## Update README and SKILL.md

- Add `todos` to the commands table in README.md
- Mark `todos` as `[x]` in the roadmap
- Add usage example to SKILL.md

## Testing

```bash
go test ./index/ -run TestTodos -v
go test ./... -v  # full suite to verify no regressions
```

## Completion Notes

Implemented by agent 54aade69. All items completed:

- **`index/todos.go`**: Core `Todos()` method on `*Index` with regex-based scanning, tag filtering, max limit, binary file skipping, and `FormatTodos()` text formatter.
- **`index/todos_test.go`**: 13 tests covering all tag detection, case-insensitive matching, message extraction, tag filtering (including case-insensitive filter), max limiting, binary file skipping, line numbers, byTag counts, multi-file sorting, empty results, and formatter output.
- **`main.go`**: Added `case "todos":` with `--root`, `--max`, `--tag` flag parsing. Added `parseStringFlag` helper. Updated `printUsage()`.
- **README.md**: Added to commands table, quick start examples, project structure, and marked `[x]` in roadmap.
- **SKILL.md**: Added usage examples for todos command.
- All 107 tests pass (`go test ./...`).
