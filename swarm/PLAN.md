# swarm-index — Implementation Plan

## Current State

We have a minimal CLI (`main.go`) with three commands:

- **`scan`** — walks a directory tree, builds an in-memory `[]Entry` of files, skips noise dirs. Works, but the result is never persisted.
- **`lookup`** — stubbed out; always returns an error ("no index loaded").
- **`version`** — prints `v0.1.0`.

The `index` package has the core types (`Entry`, `Index`), a `Match` method for substring search, and helper counters (`FileCount`, `PackageCount`). Tests cover scanning, matching, and hidden-dir skipping.

**What doesn't work yet:** lookup (no persistence), AST symbol extraction, JSON output, and every planned command.

---

## Index Storage Location

All index files are stored in the **end-user's project** at:

```
<project-root>/swarm/index/
```

When a user runs `swarm-index scan .`, the tool writes its index data into `./swarm/index/`. This directory should be committed to the repo (it's small and useful for all agents working on the project), or added to `.gitignore` — the user's choice. The tool should create the directory automatically if it doesn't exist.

Files inside `./swarm/index/`:

| File | Purpose |
|---|---|
| `index.json` | The primary index — flat array of all `Entry` records |
| `meta.json` | Metadata: scan root, timestamp, swarm-index version, file/package counts, language breakdown |
| `tree.json` | Cached directory tree structure (written by `scan`, used by `tree` command) |

Using JSON keeps the format inspectable, diffable, and trivially consumable by LLMs and other tools.

---

## Implementation Phases

### Phase 1 — Persistence & Working Lookup (MVP critical)

**Goal:** `scan` writes to disk, `lookup` reads from disk. The tool is actually usable end-to-end.

#### 1.1 Index serialization

- Add `Save(dir string) error` method to `*Index` that:
  1. Creates `<dir>/swarm/index/` if it doesn't exist.
  2. Marshals `idx.Entries` → `swarm/index/index.json` (indented JSON).
  3. Writes `swarm/index/meta.json` with: `root`, `scannedAt` (RFC 3339), `version`, `fileCount`, `packageCount`.
- Add `Load(dir string) (*Index, error)` that reads `swarm/index/index.json` back into an `*Index`.

#### 1.2 Wire up CLI

- `scan <directory>`:
  1. Run `Scan(dir)` as today.
  2. Call `idx.Save(dir)` to persist.
  3. Print summary including path to index dir.
- `lookup <query>`:
  1. Determine project root (walk up from CWD looking for `swarm/index/meta.json`, or accept an optional `--root` flag).
  2. Call `Load(root)`.
  3. Call `idx.Match(query)`.
  4. Print results.

#### 1.3 Tests

- Round-trip test: scan a temp dir → save → load → verify entries match.
- Lookup integration test using a saved index.
- Test that `swarm/index/` directory is created automatically.

---

### Phase 2 — `--json` Global Flag & Structured Output

**Goal:** Every command can emit JSON for agent consumption.

#### 2.1 Output abstraction

- Create a small `output` package (or just a helper in `index`) with:
  - `PrintText(v interface{})` — human-friendly formatting (current behavior).
  - `PrintJSON(v interface{})` — `json.MarshalIndent` to stdout.
- A global `--json` flag parsed in `main.go` (use `flag` package or switch to a lightweight CLI lib like `cobra` only if complexity warrants it — prefer stdlib for now).

#### 2.2 Apply to existing commands

- `scan` — JSON output: `{ "filesIndexed": N, "packages": N, "indexPath": "swarm/index/" }`.
- `lookup` — JSON output: array of entry objects.
- `version` — JSON output: `{ "version": "0.1.0" }`.

#### 2.3 Tests

- Test JSON output is valid and parseable for each command.

---

### Phase 3 — `tree` Command

**Goal:** Print the directory structure, respecting the same skip rules as `scan`.

#### 3.1 Implementation

- `tree <directory>` walks the dir (reusing `shouldSkipDir`) and builds a nested structure.
- Flags: `--depth N` (default unlimited), `--json`.
- Text output mimics classic `tree` format with `├──` / `└──` connectors.
- JSON output: recursive `{ "name": "...", "type": "file"|"dir", "children": [...] }`.
- Cache the tree structure in `swarm/index/tree.json` during `scan` so `tree` can read from cache when available.

#### 3.2 Tests

- Verify depth limiting.
- Verify skip rules are applied.
- Verify JSON output structure.

---

### Phase 4 — `summary` Command

**Goal:** Instant project orientation in one call.

#### 4.1 Implementation

- Auto-detect languages by file extension (maintain a map: `.go` → Go, `.ts`/`.tsx` → TypeScript, etc.).
- Count files by extension.
- Detect entry points: `main.go`, `index.ts`, `index.js`, `app.py`, `manage.py`, etc.
- Detect dependency manifests: `go.mod`, `package.json`, `requirements.txt`, `Cargo.toml`, `pyproject.toml`, etc.
- Compute total LOC (line count per file, sum).
- Output as a structured report (text table or JSON).

#### 4.2 Data source

- Reads from the persisted index (`swarm/index/index.json`) plus lightweight file inspection for LOC.
- If no index exists, runs a scan first.

#### 4.3 Tests

- Test language detection mapping.
- Test entry point detection.
- Test LOC counting.

---

### Phase 5 — `outline <file>` (AST Parsing)

**Goal:** Show the structural skeleton of a file — functions, classes, types, exports, imports — without the full source. This is the highest-value planned command.

#### 5.1 Language-specific parsers

Create a `parsers` package with a common interface:

```go
type Symbol struct {
    Name       string // e.g. "HandleAuth"
    Kind       string // "func", "type", "interface", "class", "export", "import"
    Line       int
    EndLine    int    // for scope
    Exported   bool
    Signature  string // e.g. "func HandleAuth(w http.ResponseWriter, r *http.Request)"
    Parent     string // enclosing type/class, if any
}

type Parser interface {
    Parse(filePath string, content []byte) ([]Symbol, error)
    Extensions() []string
}
```

Implement parsers in priority order:

1. **Go** — use `go/parser` and `go/ast` (stdlib, no deps).
2. **TypeScript / JavaScript** — use regex-based heuristic parser initially (function/class/export declarations). A proper parser (tree-sitter bindings) can come later.
3. **Python** — regex-based heuristic parser (def, class, import).

#### 5.2 Integration with scan

- During `scan`, after recording the file entry, run the appropriate parser.
- Append resulting `Symbol`s as additional `Entry` records (kind = `func`, `type`, `class`, etc.) to the index.
- This means the persisted `swarm/index/index.json` will contain both file entries and symbol entries.

#### 5.3 `outline` command

- Read the target file, run the parser, print the skeleton.
- If the file is already in the index, pull symbols from there instead of re-parsing.
- Text output: indented list of symbols with line numbers.
- JSON output: array of `Symbol` objects.

#### 5.4 Tests

- Test each parser against sample files with known symbols.
- Test outline output formatting.

---

### Phase 6 — `search <pattern>` Command

**Goal:** Regex search across file contents using the index's skip rules.

#### 6.1 Implementation

- Accept a regex pattern, walk indexed files (from `swarm/index/index.json`), search content.
- Return matches with file path, line number, and matching line.
- Flags: `--max-results N` (default 50), `--json`.

#### 6.2 Tests

- Test regex matching across multiple files.
- Test result limiting.

---

### Phase 7 — `show <path>` Command

**Goal:** Read a file or line range with line numbers and structural context.

#### 7.1 Implementation

- `show <path>` — print the full file with line numbers.
- `show <path> --lines M:N` — print only lines M through N.
- Add structural context: if the range falls inside a function/class (using outline data), prepend the enclosing symbol's signature.
- Flags: `--json`.

#### 7.2 Tests

- Test full file display.
- Test line range extraction.
- Test structural context injection.

---

### Phase 8 — `refs <symbol>` Command

**Goal:** Find all usages of a symbol (callers/consumers).

#### 8.1 Implementation

- Two-phase approach:
  1. Find the symbol definition in the index.
  2. Grep all indexed files for references to that symbol name.
- Filter out the definition itself.
- Group results by file.
- Flags: `--json`.

#### 8.2 Tests

- Test with known symbol usage patterns.

---

### Phase 9 — `exports <file|package>` Command

**Goal:** List the public API surface.

#### 9.1 Implementation

- Pull symbols from the index where `Exported == true`.
- If given a file, filter to that file.
- If given a package/directory, filter to all files in that package.
- Flags: `--json`.

---

### Phase 10 — `related <file>` Command

**Goal:** Show files connected to a given file.

#### 10.1 Implementation

- Parse imports from the target file.
- Search the index for files that import the target file.
- Detect test file by naming convention (`_test.go`, `.test.ts`, `.spec.ts`, `test_*.py`).
- Return three groups: imports, importers, test files.
- Flags: `--json`.

---

### Phase 11 — `diff-summary` & `stale` Commands

**Goal:** Change-awareness.

#### 11.1 `diff-summary [git-ref]`

- Shell out to `git diff --name-only <ref>` (default `HEAD~1`).
- Cross-reference changed files against the index.
- Report which symbols were affected.
- Flags: `--json`.

#### 11.2 `stale`

- Compare `swarm/index/meta.json` timestamp and file list against current filesystem.
- Report new, deleted, and modified files.
- Flags: `--json`.

---

### Phase 12 — Infrastructure Improvements

These can be done incrementally alongside the phases above:

| Item | Phase dependency | Notes |
|---|---|---|
| Fuzzy matching | After Phase 1 | Add Levenshtein or trigram scoring to `Match` |
| Relevance ranking | After Phase 5 | Rank by kind (symbol > file), exact match > prefix > substring |
| Watch mode | After Phase 1 | Use `fsnotify` to re-scan on changes |
| Custom ignore config | After Phase 1 | Read a `.swarmignore` file (same syntax as `.gitignore`) |
| CLI framework | After Phase 2 | Migrate to `cobra` if arg parsing gets unwieldy |

---

## Suggested Task Ordering for Swarm Workers

Each task below maps to a file in `./swarm/todo/` and can be picked up independently (unless dependencies noted):

| # | Task | Depends on |
|---|---|---|
| 1 | Persist index to `./swarm/index/` (Phase 1.1 + 1.3) | — |
| 2 | Wire up `scan` to save and `lookup` to load (Phase 1.2) | Task 1 |
| 3 | `--json` global flag and structured output (Phase 2) | Task 2 |
| 4 | `tree` command (Phase 3) | — |
| 5 | `summary` command (Phase 4) | Task 1 |
| 6 | Go AST parser + `outline` command (Phase 5) | Task 1 |
| 7 | JS/TS heuristic parser (Phase 5) | Task 6 |
| 8 | Python heuristic parser (Phase 5) | Task 6 |
| 9 | `search` command (Phase 6) | Task 1 |
| 10 | `show` command (Phase 7) | Task 6 (for structural context) |
| 11 | `refs` command (Phase 8) | Task 6 |
| 12 | `exports` command (Phase 9) | Task 6 |
| 13 | `related` command (Phase 10) | Task 6 |
| 14 | `diff-summary` command (Phase 11.1) | Task 1 |
| 15 | `stale` command (Phase 11.2) | Task 1 |
| 16 | Fuzzy matching + relevance ranking (Phase 12) | Task 6 |

---

## Key Design Decisions

1. **Index location is `./swarm/index/` inside the target project.** This keeps index data co-located with the code, visible in the repo, and available to all agents without needing a central server or absolute paths.

2. **JSON everywhere.** The index format is JSON, the output supports `--json`, and the tree cache is JSON. This is intentional — LLMs parse JSON reliably, humans can inspect it, and we avoid binary format complexity.

3. **Regex-first parsers for non-Go languages.** Full AST parsing for JS/TS/Python would require external dependencies (tree-sitter CGo bindings or WASM). Regex heuristics cover 90% of use cases (top-level declarations) and keep the binary small and dependency-free. Upgrade path: swap in tree-sitter later behind the same `Parser` interface.

4. **Stdlib CLI parsing until it hurts.** No `cobra` or `urfave/cli` until the arg parsing logic genuinely gets unwieldy. Fewer deps = faster builds = simpler distribution.

5. **Automatic project root detection.** For commands like `lookup` that need the index, walk up from CWD looking for `swarm/index/meta.json`. Fall back to `--root` flag.
