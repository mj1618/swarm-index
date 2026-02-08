# Refactoring Applied

## Changes

### 1. Simplified root resolution in `main.go` (history case)
The root resolution had two identical `filepath.Abs()` branches — one for when `--root` was provided and one defaulting to `"."`. Collapsed into a single call by using `"."` as the default value in `parseStringFlag`.

**Before (15 lines):**
```go
root := parseStringFlag(extraArgs, "--root", "")
if root == "" {
    var err error
    root, err = filepath.Abs(".")
    if err != nil { fatal(...) }
} else {
    var err error
    root, err = filepath.Abs(root)
    if err != nil { fatal(...) }
}
```

**After (4 lines):**
```go
root := parseStringFlag(extraArgs, "--root", ".")
root, err := filepath.Abs(root)
if err != nil { fatal(...) }
```

### 2. Eliminated duplicated git helper in `history_test.go`
The `run` helper closure (creates git commands with test env vars) was defined identically in `initGitRepo`, `TestHistoryWithCommits`, and `TestHistoryMaxLimit`. Changed `initGitRepo` to return both the directory and the `run` function, eliminating all duplication.

**Before:** `initGitRepo` returned `string`, each test redefined `run`.
**After:** `initGitRepo` returns `(string, func(args ...string))`, tests use the returned helper.
