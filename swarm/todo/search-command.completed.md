# `search` Command — Regex Search Across Indexed Files

## Problem

Agents frequently need to search for patterns across a codebase — finding function calls, string literals, import statements, error messages, etc. Currently they must rely on external tools like `grep` or `rg` and manage their own ignore rules. The `swarm-index` tool already knows which directories to skip (via `shouldSkipDir`), and has a persisted index of all relevant files. A built-in `search` command would let agents search file contents using the same skip rules, with consistent `--json` output.

## Command Signature

```
swarm-index search <pattern> [--root <dir>] [--max N] [--json]
```

- `<pattern>` — a Go regexp pattern to match against file contents.
- `--root <dir>` — project root (defaults to auto-detection via `findIndexRoot`).
- `--max N` — maximum number of matches to return (default 50).
- `--json` — output results as JSON.

## Implementation

### 1. Add `SearchResult` type and `Search` method to the `index` package

In a new file `index/search.go`:

```go
type SearchMatch struct {
    Path    string `json:"path"`    // file path relative to root
    Line    int    `json:"line"`    // 1-based line number
    Content string `json:"content"` // the matching line (trimmed)
}

func (idx *Index) Search(pattern string, maxResults int) ([]SearchMatch, error)
```

The method should:

1. Compile `pattern` using `regexp.Compile`.
2. Iterate over unique file paths in `idx.Entries`.
3. For each file, read its contents from disk (`filepath.Join(idx.Root, entry.Path)`).
4. Scan line-by-line, checking `re.MatchString(line)`.
5. On match, append a `SearchMatch` with the path, line number, and trimmed line content.
6. Stop early once `maxResults` matches are collected.
7. Return the matches slice.

Skip binary files by checking for null bytes in the first 512 bytes of each file.

### 2. Wire up the CLI in `main.go`

Add a `case "search":` block in the command switch:

- Validate that a pattern argument is provided.
- Call `resolveRoot` for `--root`.
- Parse `--max` with `parseIntFlag` (default 50).
- Call `index.Load(root)` then `idx.Search(pattern, max)`.
- Text output: print each match as `path:line: content`.
- JSON output: marshal the `[]SearchMatch` array.

### 3. Update `printUsage()` in `main.go`

Add the `search` command to the usage text.

### 4. Tests

Add `index/search_test.go`:

- **Happy path:** Create a temp directory with a few files containing known text. Scan, save, load, then search for a pattern. Verify correct matches with correct line numbers.
- **Regex pattern:** Test that regex features work (e.g. `func\s+\w+` matches Go function declarations).
- **Max results:** Create enough matches to exceed a small max, verify truncation.
- **No matches:** Search for a pattern that doesn't exist, verify empty results.
- **Binary file skip:** Create a file with null bytes, verify it's skipped.
- **Invalid regex:** Pass an invalid regex pattern, verify an error is returned.

### 5. Update README.md

Add `search` to the Commands table and move it from the Roadmap to implemented.

### 6. Update SKILL.md

Add a `search` example to the agent quick-reference.

## Text Output Format

```
index/index.go:146:  func Scan(root string) (*Index, error) {
index/index.go:199:  func (idx *Index) Match(query string) []Entry {
main.go:58:           idx, err := index.Scan(dir)

3 matches
```

## JSON Output Format

```json
[
  {
    "path": "index/index.go",
    "line": 146,
    "content": "func Scan(root string) (*Index, error) {"
  },
  {
    "path": "index/index.go",
    "line": 199,
    "content": "func (idx *Index) Match(query string) []Entry {"
  }
]
```

## Dependencies

- Phase 1 (persistence) — already completed.
- No dependency on Phase 5 (AST parsing).

## Completion Notes (agent 04c65702)

Implemented all 6 steps as specified:

1. **index/search.go** — `SearchMatch` type and `Search` method on `*Index`. Uses `regexp.Compile`, iterates unique file paths, reads files line-by-line, skips binary files (null byte check in first 512 bytes), stops at `maxResults`.
2. **main.go** — Added `case "search":` with pattern validation, `--root`, `--max 50` default, text output as `path:line: content` with match count, JSON output as `[]SearchMatch`.
3. **main.go printUsage()** — Added search command and example.
4. **index/search_test.go** — 6 tests: happy path, regex, max results, no matches, binary skip, invalid regex. All pass.
5. **README.md** — Added to Commands table, Quick start, project structure. Removed from Roadmap.
6. **SKILL.md** — Added search example.

All 42 tests pass. Manual testing verified text and JSON output.
