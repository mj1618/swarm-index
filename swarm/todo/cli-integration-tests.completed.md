# CLI Integration Tests

## Problem

The existing test suite covers internal functions (`parseMax`, `validateQuery`, `resolveRoot`, `extractJSONFlag`) and the `index` package (scan, match, save/load), but there are **no tests that exercise the CLI commands end-to-end**. This means the command routing in `main()`, the output formatting, and the wiring between CLI args and library calls are all untested.

For a shippable MVP, the three commands (`scan`, `lookup`, `version`) should have integration tests that build the binary, run it with real arguments, and verify the output.

## Approach

Add a `TestMain`-based integration test file (`cli_test.go` or `integration_test.go`) that:

1. Uses `go build` in `TestMain` to compile the binary into a temp directory once.
2. Runs the binary via `exec.Command` for each test case.
3. Checks exit codes, stdout, and stderr.

## Test Cases

### `scan` command
- **Happy path (text):** Run `swarm-index scan <tmpdir>` on a directory with known files. Verify stdout contains the file count, package count, and `Index saved to` message.
- **Happy path (JSON):** Run `swarm-index scan <tmpdir> --json`. Verify stdout is valid JSON with `filesIndexed`, `packages`, `indexPath`, and `extensions` keys.
- **Missing directory arg:** Run `swarm-index scan` with no directory. Verify non-zero exit code and usage error on stderr.
- **Nonexistent directory:** Run `swarm-index scan /no/such/path`. Verify non-zero exit code and error message.

### `lookup` command
- **Happy path (text):** Scan a tmpdir first, then run `swarm-index lookup <query> --root <tmpdir>`. Verify matching results appear in stdout.
- **Happy path (JSON):** Same but with `--json`. Verify stdout is a valid JSON array of entry objects.
- **No matches:** Lookup a query that won't match. Verify "no matches found" text output (or empty JSON array with `--json`).
- **Empty query:** Run `swarm-index lookup "" --root <tmpdir>`. Verify error about empty query.
- **--max flag:** Scan a dir with many files, lookup with `--max 2`. Verify at most 2 results shown.
- **No index exists:** Run lookup with `--root` pointing to a dir with no index. Verify error message about running scan first.

### `version` command
- **Text:** Run `swarm-index version`. Verify output contains `v0.1.0`.
- **JSON:** Run `swarm-index version --json`. Verify valid JSON with `version` key.

### Error handling
- **No args:** Run `swarm-index` with no command. Verify non-zero exit and usage message.
- **Unknown command:** Run `swarm-index foobar`. Verify non-zero exit and "unknown command" error.

## Implementation Notes

- Use `os.Executable` or `go build -o` in `TestMain` to build the binary once per test run.
- Use `t.TempDir()` for scan targets — create known file structures.
- Parse JSON outputs with `json.Unmarshal` to verify structure, not just string matching.
- Check `cmd.Run()` error for exit code validation (use `exec.ExitError`).
- Keep tests in the `main` package (same as `main_test.go`) so they have access to the same build context.

## Completion Notes

Implemented by agent 6c333b2d (task e8a349f8). Created `cli_test.go` with 20 end-to-end integration tests:

- `TestMain` builds the binary once into a temp directory
- `runBinary` helper executes the binary and captures stdout/stderr
- `makeTestDir` creates a temp directory with known files (2 .go files, 1 .md file)

Tests cover all commands:
- **scan** (4 tests): text output, JSON output, missing dir arg, nonexistent dir
- **lookup** (7 tests): text, JSON, no matches (text+JSON), empty query, --max flag, no index
- **version** (2 tests): text, JSON
- **outline** (5 tests): text, JSON, no file arg, nonexistent file, unsupported extension
- **error handling** (2 tests): no args, unknown command

All 76 tests pass across all packages.
