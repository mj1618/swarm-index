# Fuzzy Matching & Relevance Ranking for `lookup`

## Problem

The current `lookup` command uses simple case-insensitive substring matching (`strings.Contains`) with no ranking. Results are returned in scan order (filesystem walk order), which means:

1. Exact matches are buried among partial matches (searching "config" returns "config.go", "myconfig_helper.go", "reconfigure.go" in arbitrary order)
2. Typos or slight misspellings return zero results (searching "hadnler" finds nothing instead of suggesting "handler")
3. Agents waste context window space sifting through unranked results

## Goal

Add fuzzy matching and relevance-ranked scoring to `Match()` so that `lookup` returns results ordered by quality, with exact and close matches first. Optionally support a `--fuzzy` flag (or make it the default with `--exact` to opt out).

## Design

### Scoring Algorithm

Implement a multi-signal scoring function in `index/match.go`:

```go
type ScoredEntry struct {
    Entry
    Score float64 `json:"score"`
}
```

**Scoring signals (highest to lowest weight):**

1. **Exact name match** (score: 100) — query equals filename without extension
2. **Exact name match with extension** (score: 95) — query equals full filename
3. **Name prefix match** (score: 80) — filename starts with query
4. **Name substring match** (score: 60) — filename contains query as substring
5. **Path substring match** (score: 40) — path contains query but name doesn't
6. **Fuzzy match** (score: 20-35, scaled by edit distance) — Levenshtein distance ≤ 2 on the filename

**Tie-breaking:** shorter file paths rank higher (prefer closer-to-root files).

### Levenshtein Distance

Implement a minimal Levenshtein distance function in `index/fuzzy.go`. No external dependencies — it's ~20 lines of Go. Only compute on the filename (not the full path) to keep it fast.

Use a max-distance cutoff (default 2) to avoid expensive computation on unrelated strings. For names longer than `len(query) + maxDist`, skip the distance calculation entirely.

### Changes to Existing Code

1. **`index/fuzzy.go`** (new file):
   - `func levenshtein(a, b string) int` — standard DP algorithm
   - `func scoreName(query, name, path string) float64` — multi-signal scorer

2. **`index/index.go`** — modify `Match()`:
   - Rename current `Match()` to `MatchExact()` (substring-only, unranked)
   - New `Match()` calls `scoreName()` for each entry, collects entries with score > 0, sorts by score descending
   - Returns `[]Entry` (same signature) so no downstream changes needed
   - The score is used for sorting only; it's not exposed in the default output

3. **`main.go`** — add `--exact` flag to `lookup`:
   - `--exact`: use substring-only matching (old behavior)
   - Default: use fuzzy+ranked matching (new behavior)

4. **JSON output** — when `--json` is used, include `"score"` field in each result so agents can see match quality

### Tests

- `index/fuzzy_test.go`:
  - Levenshtein distance: known pairs (e.g., "kitten"/"sitting" = 3, "handler"/"hadnler" = 2)
  - Scoring: exact > prefix > substring > path > fuzzy
  - Results are sorted by score descending
  - Fuzzy matches with distance > 2 are excluded
  - `--exact` falls back to old behavior
  - Empty query returns all entries (both modes)

## Scope

- No external dependencies (Levenshtein is trivial to implement)
- Backward compatible: `Match()` returns `[]Entry` as before, just better ordered
- JSON output gains a `score` field
- The `locate` command already does its own ranking, so this only affects `lookup`
- Keep the fuzzy distance threshold at 2 to avoid false positives

## Files to Change

| File | Change |
|---|---|
| `index/fuzzy.go` | New: Levenshtein distance + scoring function |
| `index/fuzzy_test.go` | New: tests for fuzzy matching and scoring |
| `index/index.go` | Modify `Match()` to use scoring; add `MatchExact()` |
| `main.go` | Add `--exact` flag to `lookup` command |
| `README.md` | Update `lookup` docs to mention fuzzy matching and `--exact` |
| `SKILL.md` | Update `lookup` examples |

## Completion Notes

Implemented by agent 83d637b0 (task ea83f561).

All items from the design were implemented:

1. **`index/fuzzy.go`** — Created with `levenshtein()` (single-row DP with max-distance early exit) and `scoreName()` (6-tier scoring: exact=100, exact+ext=95, prefix=80, substring=60, path=40, fuzzy=20-35). Also includes `matchFuzzy()` which scores all entries and sorts by score desc with shorter-path tie-breaking.

2. **`index/index.go`** — `Match()` now uses fuzzy scoring and returns ranked results. Added `MatchScored()` for JSON output with scores. Added `MatchExact()` preserving old unranked substring behavior.

3. **`main.go`** — Added `--exact` flag to `lookup`. Default uses fuzzy+ranked. `--exact` falls back to old substring matching. JSON output includes `score` field when using default mode.

4. **`index/fuzzy_test.go`** — 12 test functions covering: Levenshtein distance correctness, max-distance cutoff, scoring priority (exact > prefix > substring > path > fuzzy), ranked results ordering, typo tolerance, exact fallback, empty query, JSON score exposure, distance threshold exclusion, and tie-breaking by path length.

5. **README.md** and **SKILL.md** — Updated with new `--exact` flag docs and fuzzy matching examples. Marked roadmap item as complete.

All tests pass (`go test ./...`).
