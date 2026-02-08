# Refactoring Review

## Changes Reviewed
Unstaged changes adding a `deps` command for parsing dependency manifests (go.mod, package.json, requirements.txt, Cargo.toml, pyproject.toml).

## Refactoring Applied

### Extracted `splitPySpec` helper in `index/deps.go`

**Problem:** `parseRequirementsTxt` and `parsePyDependencyList` both contained identical logic for splitting Python package specifiers on version operators (`==`, `>=`, `<=`, `~=`, `!=`, `>`, `<`) and stripping extras like `package[extra1,extra2]`.

**Fix:** Extracted a shared `splitPySpec(spec string) (name, version string)` function and updated both callers to use it. This eliminates ~15 lines of duplication and ensures both code paths stay in sync if the parsing logic needs updating.

## Other Observations
- The `main.go` deps case follows the same pattern as other commands (stale, diff-summary) — no issues.
- Parser functions are well-structured with clear separation of concerns.
- Test coverage is thorough across all manifest formats.
- No dead code, unused imports, or naming issues found.
