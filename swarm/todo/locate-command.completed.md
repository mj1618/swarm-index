# `locate` — Unified Smart Search Command

## Problem

Agents frequently need to find something in a codebase but don't know whether it's a filename, a symbol name, or text inside a file. Currently they must run three separate commands (`lookup`, `symbols`, `search`) and mentally combine the results. This wastes context window tokens and API calls.

## Solution

Add a `locate <query>` command that searches across all three dimensions simultaneously and returns a unified, relevance-ranked result set.

## Usage

```bash
# Find anything matching "handleAuth" — filenames, symbols, and content
swarm-index locate "handleAuth"

# Limit results
swarm-index locate "config" --max 10

# JSON output for agent consumption
swarm-index locate "parseArgs" --json
```

## Implementation

### CLI (main.go)

Add a `locate` case to the switch statement:

```
swarm-index locate <query> [--root <dir>] [--max N]
```

- Default `--max` is 20.
- Supports `--json` and `--root` like other commands.

### Core Logic (index/locate.go)

Create `func (idx *Index) Locate(query string, max int) (*LocateResult, error)`.

#### Result Types

```go
type LocateMatch struct {
    Category string `json:"category"` // "file", "symbol", or "content"
    Path     string `json:"path"`
    Name     string `json:"name"`            // filename or symbol name
    Line     int    `json:"line,omitempty"`   // for symbol and content matches
    Kind     string `json:"kind,omitempty"`   // for symbols: "func", "type", etc.
    Content  string `json:"content,omitempty"` // for content matches: the matching line
    Score    int    `json:"score"`            // relevance score for ranking
}

type LocateResult struct {
    Query   string        `json:"query"`
    Matches []LocateMatch `json:"matches"`
    Total   int           `json:"total"` // total before limiting
}
```

#### Search Strategy

1. **File matches** — run `idx.Match(query)` to find matching filenames/paths. Score:
   - Exact filename match: 100
   - Filename contains query: 80
   - Path-only match: 60

2. **Symbol matches** — run `idx.Symbols(query, "", max)` to find matching symbols. Score:
   - Exact name match: 90
   - Name starts with query: 75
   - Name contains query: 65

3. **Content matches** — run `idx.Search(query, max)` for literal text matches in file contents. Score:
   - Content match: 50

4. Merge all results, sort by score descending, deduplicate (if a file matches both by name and content, keep the higher-scored entry), and limit to `max`.

#### Formatting (index/locate.go)

`FormatLocate(result *LocateResult) string` — text output grouped by category:

```
Files:
  index/index.go                              (exact match)
  index/index_test.go                         (path match)

Symbols:
  func Index.Match        index/index.go:42   (func)
  type Index              index/index.go:10   (type)

Content:
  index/search.go:15      idx.Match(query)
  main.go:98              results := idx.Match(query)

12 total matches
```

### Tests (index/locate_test.go)

- Test that file matches are returned for filename queries.
- Test that symbol matches are returned for function/type name queries.
- Test that content matches are returned for code snippet queries.
- Test relevance ranking (exact matches rank above substring matches).
- Test deduplication (same file doesn't appear multiple times with lower score).
- Test `--max` limiting.
- Test empty query returns error.

### Documentation

- Add to README.md commands table.
- Add to SKILL.md examples.
- Add to `printUsage()` in main.go.

## Dependencies

- Requires a prior `scan` (loads index).
- Uses existing `Match`, `Symbols`, and `Search` methods — no new index infrastructure needed.

## Why This Is Valuable

An agent working on an unfamiliar codebase can issue a single `locate "UserService"` and immediately see the file where it's defined, all the functions/types with that name, and every line that references it — all in one call, ranked by relevance. This replaces what would otherwise be 3 separate commands and saves significant context window space.

## Completion Notes (agent 5e0f6db8)

Implemented as specified. All files created/modified:

- **index/locate.go** — `Locate()` method, `LocateMatch`/`LocateResult` types, `FormatLocate()`, deduplication and scoring logic
- **index/locate_test.go** — 9 tests covering file matches, symbol matches, content matches, relevance ranking, deduplication, max limiting, empty query validation, and formatting
- **main.go** — Added `locate` case to CLI switch and `printUsage()`
- **README.md** — Added to quick start, commands table, and project structure
- **SKILL.md** — Added locate examples

All tests pass (`go test ./...`). Integration tested with both text and JSON output modes.
