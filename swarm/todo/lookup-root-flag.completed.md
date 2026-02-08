# Add --root flag to lookup command

## Problem

The `lookup` command always starts searching for the index from CWD (`.`) by calling `findIndexRoot(".")`. The PLAN.md (Phase 1.2) specifies that `lookup` should "accept an optional `--root` flag" so users and agents can point to a specific project root.

Without this flag, `lookup` fails if the tool is run from a directory that isn't inside the indexed project tree. For example, an agent might invoke `swarm-index lookup "auth" --root /path/to/project` from its own working directory. Currently this is impossible — the only way to make `lookup` work is to `cd` into the project first.

This is an existing Phase 1 requirement that was not fully implemented.

## Goal

Add a `--root` flag to `lookup` that lets the caller specify the project root directly, bypassing `findIndexRoot`.

## Plan

### 1. Parse `--root` flag in the `lookup` case in `main.go`

Before calling `findIndexRoot`, check `os.Args` for a `--root <dir>` flag. If present, use that directory directly instead of walking up from CWD.

Approach: iterate over `os.Args[3:]` looking for `--root` followed by a value. This keeps things simple without adding a flag-parsing library (per PLAN.md design decision #4: "stdlib CLI parsing until it hurts").

```go
case "lookup":
    if len(os.Args) < 3 {
        fmt.Fprintln(os.Stderr, "usage: swarm-index lookup <query> [--root <dir>]")
        os.Exit(1)
    }
    query := os.Args[2]
    root, err := resolveRoot(os.Args[3:])
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
    // ... rest unchanged
```

Add a helper:

```go
// resolveRoot checks args for --root <dir>. If not found, walks up from CWD.
func resolveRoot(args []string) (string, error) {
    for i, arg := range args {
        if arg == "--root" && i+1 < len(args) {
            return filepath.Abs(args[i+1])
        }
    }
    return findIndexRoot(".")
}
```

### 2. Update usage string in `printUsage`

Update the lookup usage line to mention the flag:

```
swarm-index lookup <query> [--root <dir>]   Look up symbols, files, or concepts
```

### 3. Update the lookup error message

Update the usage hint in the lookup case to include `--root`:

```
usage: swarm-index lookup <query> [--root <dir>]
```

## Files to Modify

- `main.go` — add `resolveRoot` helper, update `lookup` case to use it, update usage strings

## Notes

- No changes to the `index` package — this is purely CLI wiring.
- `findIndexRoot` stays as the fallback when `--root` is not provided.
- Manual arg parsing keeps us in stdlib-only territory per PLAN.md design decisions.

## Completion Notes

Completed by agent db26ca0f (task 433b762e).

Changes made:
- **main.go**: Added `resolveRoot` helper that checks args for `--root <dir>` and falls back to `findIndexRoot(".")`. Updated the `lookup` case to call `resolveRoot(os.Args[3:])` instead of `findIndexRoot(".")` directly. Updated usage strings in both the `lookup` error message and `printUsage()`.
- **main_test.go** (new): Added 3 unit tests for `resolveRoot`: with `--root` flag, without flag (fallback), and with `--root` but no value (graceful fallback).

All tests pass (`go test ./...`).
