# Feature: `blame` command

## Summary

Add a `blame` command that shows git blame information for a file, attributing each line to its last-modifying commit. This is essential for agents investigating bugs, understanding code history at the line level, and determining the intent behind specific code.

## Motivation

Agents frequently need to understand *why* a line of code exists — who wrote it, when, and in what commit. The existing `history` command shows file-level commits and `diff-summary` shows recent changes, but neither provides line-level attribution. `blame` fills this gap and is one of the most commonly used git operations in real-world debugging workflows.

## Usage

```bash
# Blame an entire file
swarm-index blame main.go

# Blame a specific line range
swarm-index blame main.go --lines 10:20

# Blame with full commit hashes
swarm-index blame main.go --root ~/code/my-project

# JSON output for agent consumption
swarm-index blame main.go --json
```

## Implementation

### Data types (`index/blame.go`)

```go
type BlameLine struct {
    Line    int    `json:"line"`
    Hash    string `json:"hash"`    // short commit hash
    Author  string `json:"author"`
    Date    string `json:"date"`    // YYYY-MM-DD
    Content string `json:"content"` // the line content
}

type BlameResult struct {
    File  string      `json:"file"`
    Lines []BlameLine `json:"lines"`
    Total int         `json:"total"` // total lines blamed
}
```

### Core function

```go
func Blame(root string, file string, startLine, endLine int) (*BlameResult, error)
```

- Shell out to `git blame --porcelain <file>` (or use `-L M,N` for line ranges)
- Parse the porcelain output to extract commit hash, author, date, and line content
- Run from the project `root` directory
- Does NOT require a prior `scan` (same as `history`)

### Porcelain parsing

Git blame porcelain format outputs blocks like:
```
<hash> <orig-line> <final-line> <num-lines>
author <name>
author-mail <email>
author-time <timestamp>
author-tz <tz>
committer <name>
...
	<line content>
```

Parse each block to extract hash (first 8 chars), author name, author-time (format as YYYY-MM-DD), and the content line (prefixed with tab).

### Formatting

`FormatBlame(result *BlameResult) string` — text output:

```
main.go:
  10  a1b2c3d4  2024-03-15  Alice    func main() {
  11  a1b2c3d4  2024-03-15  Alice        args, jsonOutput := extractJSONFlag(os.Args)
  12  f5e6d7c8  2024-04-01  Bob          if len(args) < 2 {
```

Align columns for readability. Truncate author name to 12 chars.

### CLI wiring (`main.go`)

```go
case "blame":
    if len(args) < 3 {
        fatal(jsonOutput, "usage: swarm-index blame <file> [--lines M:N] [--root <dir>]")
    }
    filePath := args[2]
    extraArgs := args[3:]
    root := parseStringFlag(extraArgs, "--root", ".")
    root, err := filepath.Abs(root)
    startLine, endLine, err := parseLineRange(extraArgs)
    result, err := index.Blame(root, filePath, startLine, endLine)
    // print text or JSON
```

Reuse the existing `parseLineRange` helper from main.go.

### Tests (`index/blame_test.go`)

- Test porcelain output parsing with a mock/fixture string
- Test line range filtering
- Test formatting output
- Integration test: run blame on a known file in the test repo (if in a git repo)

### Documentation updates

- Add `blame` to README.md command table
- Add `blame` to SKILL.md
- Add `blame` to `printUsage()` in main.go

## Completion Notes

Implemented by agent 0b2f6c74. All items completed:

- `index/blame.go`: `BlameLine`, `BlameResult` types; `Blame()` function using `git blame --porcelain`; `parsePorcelain()` parser; `FormatBlame()` text formatter
- `index/blame_test.go`: 8 tests covering integration (full file, line range, multiple authors, non-git repo, nonexistent file), porcelain parsing with fixture, and format output (empty + with data)
- `main.go`: CLI wiring with `--lines` and `--root` flags; added to `printUsage()`
- `README.md`: Added to quick-start examples, commands table, and project structure
- `SKILL.md`: Added usage examples
- All tests pass (`go test ./...`)
