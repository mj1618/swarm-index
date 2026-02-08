# Refactoring Summary

Applied 3 refactorings to `index/config.go`:

1. **Eliminated redundant `fileSet` map**: `fileSet` (map[string]bool) duplicated `pathSet` (map[string]string) — both tracked indexed filenames. Removed `fileSet` entirely and used `pathSet` lookups throughout, including updating `detectTools` and `detectPackageManager` signatures.

2. **Consolidated duplicate CI config detection**: Two separate `if` blocks checking `.github/workflows/` paths for `.yml` and `.yaml` suffixes were merged into a single condition using `||`.

3. **Consolidated duplicate manifest-reading pattern in `detectFramework`**: Three nearly identical blocks (Python, Rust, Go) that each read a manifest file and checked contents against `frameworkSignals` for a specific ecosystem were replaced with a data-driven loop using a `manifestCheck` struct.

All tests pass after refactoring (full suite: 3 packages, 0 failures).
