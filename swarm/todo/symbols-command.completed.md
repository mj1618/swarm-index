# Feature: `symbols` command — project-wide symbol search

## Problem

An agent working on a project often needs to answer: "Where is function `HandleAuth` defined?" or "What file contains the `Config` struct?" Currently, `lookup` only matches on file names and paths. To find a symbol, the agent must either guess the file and use `outline`, or use `search` with a regex — both are indirect and noisy. There is no way to search across all symbols in the project by name.

## Solution

Add a `symbols <query>` command that searches all parseable files in the index for symbols (functions, types, classes, interfaces, methods, constants, variables) matching the query by name. Returns matching symbols with their file, line number, kind, and signature.

## Usage

```bash
# Find all symbols matching "auth"
swarm-index symbols "auth"

# Limit results
swarm-index symbols "Config" --max 10

# Filter by kind (func, type, class, interface, method, const, var)
swarm-index symbols "Handle" --kind func

# JSON output
swarm-index symbols "auth" --json
```

## Implementation

### 1. Add `Symbols` method to `*Index` in `index/symbols.go`

```go
type SymbolMatch struct {
    Name      string `json:"name"`
    Kind      string `json:"kind"`
    Path      string `json:"path"`
    Line      int    `json:"line"`
    Signature string `json:"signature"`
    Exported  bool   `json:"exported"`
}

type SymbolsResult struct {
    Matches []SymbolMatch `json:"matches"`
    Total   int           `json:"total"`
}

func (idx *Index) Symbols(query string, kind string, max int) (*SymbolsResult, error)
```

- Iterate over all unique file paths in the index
- For each file with a supported extension (.go, .py, .js, .jsx, .ts, .tsx), run the appropriate parser from the `parsers` package
- Collect all symbols whose name contains the query (case-insensitive substring match)
- If `kind` is non-empty, filter to symbols of that kind only
- Sort results: exact name matches first, then prefix matches, then substring matches
- Limit to `max` results
- Return the result

### 2. Add `FormatSymbols` function

Text output format:
```
Symbols matching "auth" (12 found):

  func HandleAuth           main.go:45
  func AuthMiddleware       middleware/auth.go:12
  type AuthConfig           config/auth.go:8
  interface Authenticator   auth/interface.go:3
  ...
```

### 3. Wire up CLI in `main.go`

Add `case "symbols":` block:
- Parse `<query>` (required, first positional arg)
- Parse `--root`, `--max` (default 50), `--kind` flags
- Load index, call `idx.Symbols(query, kind, max)`
- Output text or JSON based on `--json` flag

### 4. Update `printUsage()` in `main.go`

Add the symbols command to the usage text.

### 5. Tests in `index/symbols_test.go`

- Test basic symbol search across multiple files
- Test kind filtering
- Test case-insensitive matching
- Test result ordering (exact > prefix > substring)
- Test max limiting
- Test with no matches

### 6. Update README.md and SKILL.md

Add `symbols` command documentation and examples.

## Dependencies

- Requires the `parsers` package (already exists with Go, Python, JS/TS parsers)
- Requires a prior `scan` (uses the index to know which files to parse)

## Notes

- This command parses files on-the-fly (like `outline` and `exports` do) rather than requiring symbols to be stored in the index. This avoids needing to change the index format or re-scan.
- Future optimization: once Phase 5.2 (symbol extraction during scan) is implemented, this command can read from the index directly instead of re-parsing.

## Completion Notes

Implemented by agent 92a2b22c (task 94787cc9). All items completed:

1. Created `index/symbols.go` with `SymbolMatch`, `SymbolsResult` types, `Symbols()` method, `matchRank()` helper, and `FormatSymbols()` function.
2. Wired up `case "symbols":` in `main.go` with `--root`, `--max` (default 50), `--kind` flags and `--json` support.
3. Added `symbols` to `printUsage()`.
4. Created `index/symbols_test.go` with 10 tests covering: basic search, kind filtering, case-insensitive matching, result ordering (exact > prefix > substring), max limiting, no matches, multi-file search, and format function tests.
5. Updated `README.md` (examples, commands table, project structure) and `SKILL.md` (examples).
6. All tests pass. Build succeeds. Smoke tested with real index.
