# Skip swarm/index/ Directory During Scan

## Problem

After running `swarm-index scan .` once, the tool writes its index files to `./swarm/index/` (specifically `index.json` and `meta.json`). On the **next** scan, these files are walked and indexed as regular entries. This means the index contains entries for its own metadata files:

```
[file] index.json — swarm/index/index.json ((root)/swarm/index)
[file] meta.json — swarm/index/meta.json ((root)/swarm/index)
```

This pollutes search results. A `lookup "index"` query returns `swarm/index/index.json` alongside the user's actual source files. The problem compounds: every scan re-indexes the previous scan's output, and future features like `tree.json` will add more noise.

The `shouldSkipDir` function in `index/index.go` skips `.git`, `node_modules`, `__pycache__`, etc., but does not skip the `swarm/index/` output directory. Since `swarm` doesn't start with `.`, the catch-all `strings.HasPrefix(name, ".")` check also doesn't catch it.

## Goal

Ensure that `scan` never indexes the tool's own output directory (`swarm/index/`), so the persisted index only contains the user's actual project files.

## Plan

### 1. Add `"swarm"` to the skip list in `shouldSkipDir` in `index/index.go`

Add `"swarm"` to the `skip` slice in `shouldSkipDir`. This is the simplest fix and matches the pattern used for other non-project directories. The entire `swarm/` tree (including `swarm/todo/`, `swarm/index/`, etc.) is tool infrastructure, not user source code, and should not appear in the index.

Current code (line ~182):
```go
func shouldSkipDir(name string) bool {
    skip := []string{
        ".git", ".hg", ".svn",
        "node_modules", "vendor", "__pycache__",
        ".idea", ".vscode", ".cursor",
        "dist", "build", ".next",
    }
    ...
}
```

Updated:
```go
func shouldSkipDir(name string) bool {
    skip := []string{
        ".git", ".hg", ".svn",
        "node_modules", "vendor", "__pycache__",
        ".idea", ".vscode", ".cursor",
        "dist", "build", ".next",
        "swarm",
    }
    ...
}
```

### 2. Add a test in `index/index_test.go`

Add a test that verifies `swarm/index/` contents are excluded from scan results:

- Create a temp directory with a user file (`main.go`) and a `swarm/index/index.json` file.
- Run `Scan` on the temp directory.
- Assert that the resulting index contains only `main.go` and does not contain any `swarm/` entries.

### 3. Verify existing tests still pass

Run `go test ./...` to confirm nothing breaks.

## Files to Modify

- `index/index.go` — add `"swarm"` to `shouldSkipDir` skip list (~1 line)
- `index/index_test.go` — add test for swarm directory skipping (~15 lines)

## Notes

- This is the same pattern used for `.git`, `node_modules`, etc. — simple directory name match.
- Skipping the entire `swarm` directory (not just `swarm/index`) is correct because `swarm/todo/` and other swarm infrastructure files are also not user source code.
- If users happen to have a `swarm/` directory that contains actual source code (unlikely naming collision), they would need to rename it. This trade-off is acceptable for an MVP — the tool's own output directory takes priority.

## Completion Notes (Agent 093fa4d8)

**Done.** All three steps completed:

1. Added `"swarm"` to the `skip` slice in `shouldSkipDir` (`index/index.go:187`).
2. Added `TestScanSkipsSwarmDir` test (`index/index_test.go`) — creates a temp dir with `main.go` and `swarm/index/index.json`, `swarm/index/meta.json`, `swarm/todo/task.md`, then verifies only `main.go` is indexed and no swarm entries appear.
3. All 9 tests pass (`go test ./... -v`), build succeeds (`go build ./...`).
