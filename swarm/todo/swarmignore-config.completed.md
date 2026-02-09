# Feature: `.swarmignore` — Custom Path Ignore Rules

## Problem

The `shouldSkipDir` function in `index/index.go` has a hardcoded skip list (`node_modules`, `vendor`, `dist`, `build`, etc.). Projects with non-standard directory layouts—e.g., generated code in `gen/`, proto outputs in `proto_out/`, large fixture directories, or monorepo-specific noise—have no way to tell `swarm-index` to skip them. This leads to bloated indexes and noisy search results.

## Solution

Support a `.swarmignore` file at the project root (same level as `swarm/index/`). Syntax follows `.gitignore` conventions (one pattern per line, `#` comments, blank lines ignored, directory patterns with trailing `/`). When `scan` runs, it reads `.swarmignore` and applies those patterns **in addition to** the built-in skip list.

## File format

```
# .swarmignore — paths to exclude from swarm-index scan

# Generated protobuf code
proto_out/

# Large test fixtures
testdata/fixtures/snapshots/

# Specific files
*.generated.go
*.min.js
```

## Implementation

### 1. Parse `.swarmignore` (new function in `index/index.go`)

Add a function `loadIgnorePatterns(root string) []string` that:
1. Reads `<root>/.swarmignore` if it exists (no error if missing).
2. Strips blank lines and `#` comment lines.
3. Returns a list of patterns.

### 2. Pattern matching function

Add `shouldIgnore(path string, patterns []string) bool` that checks if a relative path matches any pattern:
- Patterns ending in `/` match directory names anywhere in the path.
- Patterns with a `/` prefix match from the root.
- Patterns without `/` match against the basename.
- Support `*` as a glob wildcard using `filepath.Match`.

### 3. Wire into `Scan`

Modify `Scan(dir)` to:
1. Call `loadIgnorePatterns(dir)` once at the start.
2. Pass patterns to the walk function.
3. In the `WalkDir` callback, check `shouldIgnore(relativePath, patterns)` before processing each file or directory.
4. Skip matched directories entirely (return `fs.SkipDir`), skip matched files silently.

### 4. Wire into `BuildTree`

Modify `BuildTree` to also respect `.swarmignore` patterns so that `tree` output matches `scan` behavior.

### 5. Tests

Add tests in `index/index_test.go`:
- Test `loadIgnorePatterns` with a temp `.swarmignore` file.
- Test `shouldIgnore` with various pattern types (dir trailing `/`, glob `*`, basename match, rooted `/prefix` match).
- Test that `Scan` skips directories and files matching `.swarmignore` patterns.
- Test that missing `.swarmignore` causes no errors (graceful fallback).
- Test that `BuildTree` also respects the ignore patterns.

## Files to modify

| File | Change |
|---|---|
| `index/index.go` | Add `loadIgnorePatterns`, `shouldIgnore`; modify `Scan` to load and apply patterns |
| `index/tree.go` | Modify `BuildTree` to load and apply patterns |
| `index/index_test.go` | Add tests for ignore pattern loading, matching, and scan integration |
| `README.md` | Document `.swarmignore` support |
| `SKILL.md` | Mention `.swarmignore` |

## Acceptance criteria

- [x] `swarm-index scan .` reads `.swarmignore` from the scan root and skips matching paths
- [x] Patterns support: directory names (`dirname/`), globs (`*.generated.go`), basenames (`secrets.json`)
- [x] Missing `.swarmignore` is silently ignored (no error)
- [x] `swarm-index tree` also respects `.swarmignore`
- [x] All new code has tests
- [x] README and SKILL.md updated

## Completion notes

Implemented by agent e527d941. All acceptance criteria met:

- Added `loadIgnorePatterns(root string) []string` to parse `.swarmignore` files
- Added `shouldIgnore(relPath string, isDir bool, patterns []string) bool` with support for:
  - Directory patterns (`dirname/`) — only match directories, match basename anywhere
  - Glob patterns (`*.ext`) — matched against basename using `filepath.Match`
  - Basename patterns (`name`) — exact basename match
  - Rooted patterns (`/path`) — match only at project root
- Wired into `Scan` in `index/index.go` — loads patterns once, checks each dir/file
- Wired into `BuildTree` in `index/tree.go` — loads patterns once, passes through recursive calls
- Added 6 new tests: `TestLoadIgnorePatterns`, `TestLoadIgnorePatternsNoFile`, `TestShouldIgnore` (16 cases), `TestScanRespectsSwarmignore`, `TestScanNoSwarmignore`, `TestBuildTreeRespectsSwarmignore`
- Updated README.md with documentation section and roadmap checkbox
- Updated SKILL.md with `.swarmignore` usage note
- All tests pass (go test ./... — 3 packages OK)
