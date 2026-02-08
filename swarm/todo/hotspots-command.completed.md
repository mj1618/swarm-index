# Feature: `hotspots` command

## Summary

Add a `hotspots` command that identifies the most frequently modified files in the git history. This helps agents quickly understand which files are the most active, complex, and important in a project — the files most likely to need attention and most critical to understand.

## Motivation

When an agent starts working on a project, it needs to know which files matter most. The most frequently changed files are typically:
- Core business logic and architectural hotspots
- Bug-prone areas that need careful attention
- Files where most future work will happen
- Key integration points that many features touch

This is different from `history` (which shows commits for one file). `hotspots` gives a project-wide ranking of files by change frequency.

## CLI Interface

```bash
# Show top 20 most frequently changed files
swarm-index hotspots

# Limit results
swarm-index hotspots --max 10

# Only count commits since a git ref (e.g., last 6 months of changes)
swarm-index hotspots --since "6 months ago"

# Filter to a specific directory
swarm-index hotspots --path src/

# Point at a specific project root
swarm-index hotspots --root ~/code/my-project

# JSON output
swarm-index hotspots --json
```

## Implementation

### 1. Add `Hotspots` function to `index` package

Create `index/hotspots.go`:

```go
type HotspotEntry struct {
    Path       string `json:"path"`
    CommitCount int   `json:"commitCount"`
    LastModified string `json:"lastModified"` // RFC 3339 date of most recent commit
}

type HotspotsResult struct {
    Entries []HotspotEntry `json:"entries"`
    Total   int            `json:"total"`   // total files analyzed
    Since   string         `json:"since"`   // git ref or time constraint used, if any
}
```

- Shell out to `git log --format=format: --name-only` (or `git log --pretty=format: --name-only --since=<since>`) to get all changed file paths across commits.
- Count occurrences of each file path.
- Optionally filter by path prefix (the `--path` flag).
- Cross-reference against the persisted index to only include files that still exist (skip deleted files).
- Get last modified date for each file from `git log -1 --format=%aI -- <path>`.
- Sort by commit count descending.
- Limit to `--max` results (default 20).

### 2. Add `FormatHotspots` function

Text output format:
```
Hotspots (top 20 most changed files):

  52 commits  main.go                        (last: 2025-01-15)
  47 commits  index/index.go                 (last: 2025-01-14)
  31 commits  parsers/goparser.go            (last: 2025-01-13)
  ...

20 of 145 files shown
```

### 3. Wire up CLI in `main.go`

Add `case "hotspots":` to the switch statement. Parse `--max`, `--since`, `--path`, `--root`, and `--json` flags.

### 4. Tests

Create `index/hotspots_test.go`:
- Test with a temp git repo with known commit history.
- Test `--max` limiting.
- Test `--path` filtering.
- Test `--since` filtering.
- Test that deleted files are excluded.
- Test JSON output structure.

### 5. Update docs

- Add `hotspots` to `README.md` commands table and examples.
- Add `hotspots` to `SKILL.md` usage examples.
- Add `hotspots` to `printUsage()` in `main.go`.

## Dependencies

- Requires `git` to be available (same as `history` and `diff-summary`).
- Requires a prior `scan` (to cross-reference existing files).

## Acceptance Criteria

- `swarm-index hotspots` shows the most frequently changed files ranked by commit count.
- `--max`, `--since`, `--path`, `--root`, and `--json` flags work correctly.
- Deleted files are excluded from results.
- All tests pass.
- README.md, SKILL.md, and printUsage() are updated.

## Completion Notes

Implemented by agent 5417d7cd (task 0d8927c7).

### Files created:
- `index/hotspots.go` — `Hotspots()` method on `*Index`, `FormatHotspots()`, and `getLastModified()` helper
- `index/hotspots_test.go` — 10 tests covering basic ranking, max limit, path filtering, deleted file exclusion, since filtering, JSON structure, non-git-repo error, and format functions

### Files modified:
- `main.go` — Added `case "hotspots"` to switch statement and added to `printUsage()`
- `README.md` — Added Quick start examples, Commands table entry, Project structure entry, and Roadmap entry
- `SKILL.md` — Added hotspots usage examples

### All acceptance criteria met:
- `swarm-index hotspots` ranks files by commit count
- `--max`, `--since`, `--path`, `--root`, and `--json` flags all work
- Deleted files are excluded (cross-referenced against index)
- All 10 new tests pass, full test suite passes
- Documentation updated in all three locations
