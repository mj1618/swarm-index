# Refactoring Review

## Changes Reviewed
- `index/todos.go` — new TODO comment scanning implementation
- `index/todos_test.go` — tests for todos feature
- `main.go` — new `todos` CLI case, `parseStringFlag` helper, usage text
- `README.md` / `SKILL.md` — documentation updates
- `swarm/index/index.json` / `meta.json` — re-scan artifacts

## Refactoring Applied

### Extracted `todosInFile` helper in `todos.go`

The original `Todos()` method inlined file opening, scanning, and closing within a loop. This was inconsistent with `refs.go`, which extracts `refsInFile()` as a separate function enabling `defer f.Close()`.

**Before:** `f.Close()` called manually at end of loop body — no `defer`, risk of file leak if code is modified to add early `break`/`continue` paths.

**After:** Extracted `todosInFile(fullPath, relPath string) []TodoComment` helper that:
- Opens the file with `openTextFile` (shared helper)
- Uses `defer f.Close()` for safe cleanup
- Returns all found `TodoComment` entries
- Caller (`Todos`) handles filtering by tag and maxResults

This matches the `refsInFile` pattern in `refs.go` and is safer against future modifications.

## No Issues Found
- `parseStringFlag` mirrors `parseIntFlag` cleanly
- CLI `todos` case follows the same pattern as all other commands
- Tests are thorough and well-structured
- Documentation updates are accurate
- No dead code, unused imports, or naming issues
