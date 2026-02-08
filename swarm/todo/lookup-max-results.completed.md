# Add `--max` flag to `lookup` command

## Problem

The `lookup` command returns all matching entries with no limit. On a real codebase with hundreds or thousands of files, a broad query (e.g. `.go` or `util`) can produce overwhelming output. This makes `lookup` impractical for both humans and LLM agents, who have limited context windows.

## Solution

Add a `--max N` flag to `lookup` that caps the number of results printed. Default to 20 results when the flag is not provided, which keeps output manageable while still being useful. Print a trailing message like `... and 15 more matches (use --max to see more)` when results are truncated so the user knows they're not seeing everything.

## Changes

### `main.go`

- Parse `--max N` from `os.Args` in the `lookup` case (alongside the existing `--root` parsing).
- After calling `idx.Match(query)`, truncate the printed output to `max` results.
- If results were truncated, print a summary line indicating how many total matches exist.

### `main_test.go`

- Test that `--max` flag is parsed correctly by `resolveRoot` or a new arg-parsing helper.
- Ideally test via a helper that the truncation logic works (e.g. 50 matches with `--max 5` prints 5 + summary).

## Notes

- This is purely a display-layer cap — `Match()` still returns all results. This keeps the index package clean and the limiting in the CLI layer where it belongs.
- The default of 20 balances usefulness with readability. Agents can pass `--max 100` if they want more.
- Keep the flag parsing manual (consistent with existing `--root` parsing style) — no new dependencies.

## Completion Notes

Implemented by agent 1aabebe4 (task 9048c447).

### What was done

- Added `parseMax()` function in `main.go` that parses `--max N` from CLI args, defaulting to 20.
- Updated the `lookup` case in `main()` to truncate output to `max` results and print a summary line (`... and N more matches (use --max to see more)`) when truncated.
- Added `strconv` import for integer parsing.
- Updated usage string to include `[--max N]`.
- Added 6 tests in `main_test.go` covering: default value, explicit value, combined with other flags, invalid value, zero value, and missing value.
- All tests pass, project builds cleanly.
