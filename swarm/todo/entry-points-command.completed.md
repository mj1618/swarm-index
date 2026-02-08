# Feature: `entry-points` command

## Summary

Add an `entry-points` command that finds executable entry points in a codebase — main functions, route handlers, CLI command registrations, and app bootstrapping code. Unlike the `summary` command which only detects entry-point *files* by name, this command looks *inside* files using pattern matching to find the actual entry-point code with line numbers and context.

## Motivation

When an agent starts working on a project, one of the first questions is: "Where does execution begin?" and "Where are routes/commands registered?" The `summary` command lists files named `main.go` or `app.py`, but doesn't tell you which functions are the actual entry points, where routes are defined, or where CLI subcommands are wired up. This command fills that gap.

## Design

### New file: `index/entrypoints.go`

Create an `EntryPoints(max int) (*EntryPointsResult, error)` method on `*Index`.

### Types

```go
type EntryPoint struct {
    Path      string `json:"path"`
    Line      int    `json:"line"`
    Kind      string `json:"kind"`      // "main", "route", "cli", "init"
    Signature string `json:"signature"` // the matching line, trimmed
}

type EntryPointsResult struct {
    EntryPoints []EntryPoint `json:"entryPoints"`
    Total       int          `json:"total"`
}
```

### Entry point kinds to detect

**"main" — program entry points:**
- Go: lines matching `func main()` in files with `package main`
- Python: `if __name__` patterns (e.g., `if __name__ == "__main__"`)
- JS/TS: no direct equivalent, but detect top-level `createServer`, `app.listen`, `serve(`
- Rust: `fn main()`
- Java: `public static void main`

**"route" — HTTP route handlers:**
- Go: `http.HandleFunc(`, `http.Handle(`, `r.GET(`, `r.POST(`, `router.HandleFunc(`, `mux.Handle`, `e.GET(`, `app.Get(`
- JS/TS: `app.get(`, `app.post(`, `app.put(`, `app.delete(`, `app.use(`, `router.get(`, `router.post(`
- Python: `@app.route(`, `@app.get(`, `@app.post(`, `@router.get(`, `path(` (Django)

**"cli" — CLI command/subcommand registrations:**
- Go: `cobra.Command{`, `AddCommand(`, `flag.String(`, `flag.Bool(`
- Python: `add_argument(`, `add_subparsers(`, `@click.command`, `@click.group`
- JS/TS: `.command(` (Commander.js, yargs)

**"init" — initialization/bootstrap code:**
- Go: `func init()`
- Python: `def setup(`, calls to `Flask(`, `FastAPI(`, `Django`
- JS/TS: `createApp(`, `createRoot(`, `ReactDOM.render(`

### Implementation approach

1. Iterate over indexed files, read each file's content
2. For each line, check against language-appropriate regex patterns
3. Classify each match by kind
4. Collect results up to `max`, sorted by kind then path then line
5. Skip binary files and test files (files matching `_test.go`, `.test.ts`, `.spec.ts`, `test_*.py`)

### CLI integration in `main.go`

```
swarm-index entry-points [--root <dir>] [--max N] [--kind KIND]
```

- `--max N` — limit results (default 100)
- `--kind KIND` — filter by kind: `main`, `route`, `cli`, `init` (default: all)
- `--json` — structured output (global flag)

### Text output format

```
Main entry points:
  main.go:43             func main()
  cmd/server/main.go:12  func main()

Route handlers:
  api/routes.go:25       http.HandleFunc("/api/auth", handleAuth)
  api/routes.go:26       http.HandleFunc("/api/users", handleUsers)

CLI commands:
  cmd/root.go:15         rootCmd = &cobra.Command{

Init functions:
  index/index.go:10      func init()
```

### Tests: `index/entrypoints_test.go`

- Test detection of Go `func main()` and `func init()`
- Test detection of HTTP route patterns
- Test detection of Python `if __name__` patterns
- Test `--kind` filtering
- Test `--max` limiting
- Test that test files are skipped
- Test JSON output structure

## Files to create/modify

1. **Create** `index/entrypoints.go` — core logic
2. **Create** `index/entrypoints_test.go` — tests
3. **Modify** `main.go` — add `entry-points` case to switch, add to `printUsage()`
4. **Modify** `README.md` — check off `entry-points` in roadmap, add to commands table
5. **Modify** `SKILL.md` — add usage example

## Dependencies

Requires a prior `scan` (reads from persisted index to get file list, then reads file contents from disk).

## Completion Notes

Implemented by agent 5e95d27d (task c9b1ee20). All items completed:

1. **Created** `index/entrypoints.go` — Core logic with regex-based pattern matching for Go, Python, JS/TS, Rust, and Java. Detects main functions, HTTP route handlers (Express, Flask, stdlib, chi/echo/gin), CLI commands (cobra, argparse, click, commander), and init/bootstrap code. Skips binary and test files. Results sorted by kind > path > line.
2. **Created** `index/entrypoints_test.go` — 16 tests covering Go main/init, HTTP routes, Python main/Flask, JS routes, kind filtering, max limiting, test file skipping, sort order, binary file skipping, cobra commands, JSON structure, empty index, and format output.
3. **Modified** `main.go` — Added `entry-points` case with `--max`, `--kind`, `--root`, and `--json` flag support. Added to `printUsage()`.
4. **Modified** `README.md` — Checked off entry-points in roadmap, added usage examples, added command table entry, added to project structure.
5. **Modified** `SKILL.md` — Added usage examples for agents.

All tests pass (`go test ./...`). CLI smoke-tested with text and JSON output.
