# `history` command — recent git commits for a file

## Summary

Add a `history` command that shows recent git commits that touched a specific file. This is essential for agents investigating why code looks the way it does, understanding recent changes, and tracing the origin of bugs.

## Motivation

When an agent is debugging or reviewing code, knowing the recent commit history for a file provides critical context:
- **Bug investigation**: "When was this file last changed? What was the intent?"
- **Code understanding**: "Who changed this and why?" (commit messages carry intent)
- **Change awareness**: "Has this file been actively evolving or stable?"

This is listed in the README roadmap as a planned command.

## Design

### CLI interface

```
swarm-index history <file> [--root <dir>] [--max N] [--json]
```

- `<file>` — path to the file (relative or absolute)
- `--root <dir>` — project root (uses auto-detection if omitted)
- `--max N` — maximum number of commits to show (default 10)
- `--json` — structured JSON output

### Implementation

#### 1. New file: `index/history.go`

Define result types:

```go
type HistoryCommit struct {
    Hash    string `json:"hash"`     // short commit hash (7 chars)
    Author  string `json:"author"`   // author name
    Date    string `json:"date"`     // ISO 8601 date
    Subject string `json:"subject"`  // first line of commit message
}

type HistoryResult struct {
    Path    string          `json:"path"`
    Commits []HistoryCommit `json:"commits"`
    Total   int             `json:"total"` // total shown
}
```

Implement `History(root, filePath string, max int) (*HistoryResult, error)`:

1. Resolve the file path relative to the project root.
2. Shell out to `git log --format="%h%x00%an%x00%aI%x00%s" -n <max> -- <file>` from the root directory.
3. Parse the NUL-delimited output into `HistoryCommit` structs.
4. Return the result.

Implement `FormatHistory(result *HistoryResult) string`:
- Human-readable output like:
  ```
  History for main.go (5 commits):
    abc1234  2025-01-15  John Doe     Add error handling to main
    def5678  2025-01-14  Jane Smith   Refactor CLI argument parsing
  ```

#### 2. New file: `index/history_test.go`

Test cases:
- History for a file with commits — verify commits are returned with correct fields.
- History for a non-existent file — returns empty commits list (git log returns nothing).
- History with `--max` limiting — verify truncation.
- Test outside a git repo — returns a clear error.

Use a temp directory with `git init`, create a file, make commits, then call `History()`.

#### 3. Wire up in `main.go`

Add a `case "history":` block following the established pattern:
- Parse `<file>` from args[2].
- Resolve `--root` via `resolveRoot()`.
- Parse `--max` via `parseIntFlag()` (default 10).
- Call `index.History(root, filePath, max)`.
- Output text or JSON based on `--json` flag.

Add to `printUsage()`:
```
swarm-index history <file> [--root <dir>] [--max N]   Show recent git commits for a file
```

#### 4. Update README.md

- Mark `history` as `[x]` in the roadmap.
- Add to the Commands table.
- Add example to Quick Start section.

#### 5. Update SKILL.md

Add history examples:
```bash
# Show recent commits for a file
swarm-index history main.go

# Limit to last 3 commits
swarm-index history main.go --max 3
```

## Notes

- This command does NOT require a scanned index — it works directly with git. The `--root` flag is only needed to locate the git repository when running from a subdirectory.
- Uses the same `exec.Command("git", ...)` pattern as `diffsummary.go`.
- The `%x00` (NUL) delimiter in `git log --format` avoids ambiguity from commit messages containing special characters.
- Does not need to call `index.Load()` since it operates on git directly, not the index.

## Completion Notes

Implemented by agent ee37ced3. All items completed:
- Created `index/history.go` with `History()` and `FormatHistory()` functions
- Created `index/history_test.go` with 6 tests (commits, nonexistent file, max limit, not a git repo, format with/without commits)
- Wired up `case "history":` in `main.go` with `--root`, `--max`, and `--json` support
- Added to `printUsage()` in `main.go`
- Updated README.md: commands table, quick start examples, project structure, roadmap checkbox
- Updated SKILL.md with history examples
- All tests pass (`go test ./...`), end-to-end text and JSON output verified
