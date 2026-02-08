# `show` Command — Read Files with Line Numbers

## Problem

Agents frequently need to read a specific file or a range of lines from a file. Currently, they must shell out to `cat -n` or use language-specific tools. The `show` command gives agents a single, consistent way to read file contents with line numbers, integrated into the swarm-index CLI. It respects `--json` for structured output and supports line range selection.

This is Phase 7 from the PLAN. The structural context feature (prepending enclosing function/class signature) is deferred until the `outline` command and AST parsers exist.

## Approach

Add a `show` command to the CLI with the following interface:

```
swarm-index show <path> [--lines M:N] [--json]
```

- `show <path>` — print the full file with line numbers.
- `show <path> --lines M:N` — print only lines M through N (1-indexed, inclusive).
- `--json` — output as structured JSON.

### Implementation Details

#### 1. Add `ShowFile` function to the `index` package

Create a new file `index/show.go` with:

```go
type ShowResult struct {
    Path      string     `json:"path"`
    StartLine int        `json:"startLine"`
    EndLine   int        `json:"endLine"`
    TotalLines int       `json:"totalLines"`
    Lines     []ShowLine `json:"lines"`
}

type ShowLine struct {
    Number  int    `json:"number"`
    Content string `json:"content"`
}

func ShowFile(path string, startLine, endLine int) (*ShowResult, error)
```

- Reads the file at `path`.
- If `startLine` and `endLine` are both 0, returns all lines.
- If specified, returns only lines in the range `[startLine, endLine]` (1-indexed, inclusive).
- Returns an error if the file doesn't exist, is a directory, or if the line range is invalid (e.g., start > end, or start > total lines).
- Detects binary files (contains null bytes in first 512 bytes) and returns an error for those.

#### 2. Wire up CLI in `main.go`

Add a `case "show":` block that:

1. Requires at least one argument (the file path).
2. Parses `--lines M:N` flag by splitting on `:`.
3. Calls `index.ShowFile(path, start, end)`.
4. In text mode: prints each line as `{lineNumber}\t{content}` (matching `cat -n` style).
5. In JSON mode: marshals the `ShowResult` struct.

#### 3. Parse the `--lines` flag

Add a helper `parseLineRange(args []string) (int, int, error)` in `main.go`:

- Scans args for `--lines` followed by a value.
- Splits the value on `:` to get start and end.
- Supports formats: `M:N` (range), `M:` (from M to end), `:N` (from start to N), `M` (single line).
- Returns `(0, 0, nil)` if `--lines` is absent (meaning "show all").

#### 4. Update `printUsage()` and README

Add `show` to the usage string and the README commands table.

## Test Cases (`index/show_test.go`)

- **Full file display:** Create a temp file with known content, call `ShowFile` with no range, verify all lines returned with correct numbers.
- **Line range:** Create a 10-line file, request lines 3:7, verify exactly lines 3-7 returned.
- **Single line:** Request `--lines 5:5`, verify only line 5 returned.
- **Open-ended range (M:):** Request lines 5 to end, verify lines 5 through last line.
- **Open-ended range (:N):** Request lines 1 to N.
- **Out-of-range start:** Request start line beyond file length, verify error.
- **Start > end:** Request `--lines 7:3`, verify error.
- **Nonexistent file:** Verify error returned.
- **Binary file:** Create a file with null bytes, verify error about binary file.
- **Empty file:** Verify returns empty lines array with totalLines=0.

## Acceptance Criteria

- `swarm-index show <path>` prints the file with line numbers.
- `swarm-index show <path> --lines 10:20` prints only lines 10-20.
- `swarm-index show <path> --json` outputs structured JSON with path, line range, and line content.
- Binary files are rejected with a clear error.
- Missing file produces a clear error.
- Invalid line ranges produce clear errors.
- README and usage text are updated.

## Completion Notes

Implemented by agent 0c7c38e3. All acceptance criteria met:

- Created `index/show.go` with `ShowFile()` function supporting full file display and line ranges.
- Created `index/show_test.go` with 12 tests covering all specified test cases plus a directory test and end-beyond-file clamping test.
- Wired up `show` command in `main.go` with `parseLineRange()` helper supporting `M:N`, `M:`, `:N`, and `M` formats.
- Updated `printUsage()`, README (commands table, quick start, project structure, roadmap), and SKILL.md.
- All 53 tests pass. Manual CLI testing confirms both text and JSON output work correctly.
