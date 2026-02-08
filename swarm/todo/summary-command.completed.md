# `summary` Command

## Problem

When an agent lands on an unfamiliar codebase, the first thing it needs is a quick orientation: what languages are used, how big is the project, where are the entry points, and what dependencies exist. Currently, `scan` reports file counts and extension breakdown, but there's no single command that gives a comprehensive project overview. Agents have to piece this together manually from multiple sources.

## Goal

Add a `summary` command that reads the persisted index and performs lightweight file inspection to produce a structured project overview in one call.

## Output

The summary should include:

1. **Language breakdown** — files grouped by language (mapped from file extensions), with counts and percentages.
2. **Total file count and LOC** — line counts per file (sum total). LOC is computed by counting newlines in each indexed file.
3. **Entry points detected** — files matching known entry-point patterns: `main.go`, `index.ts`, `index.js`, `app.py`, `manage.py`, `main.py`, `main.rs`, `Main.java`, `Program.cs`, etc.
4. **Dependency manifests detected** — files matching known manifest patterns: `go.mod`, `package.json`, `requirements.txt`, `Cargo.toml`, `pyproject.toml`, `Gemfile`, `pom.xml`, `build.gradle`, etc.
5. **Top-level directories** — list of immediate subdirectories under the scanned root (excluding skipped dirs).

## CLI Interface

```
swarm-index summary [--root <dir>] [--json]
```

- `--root <dir>`: Specify the project root (same resolution logic as `lookup` — walks up from CWD looking for `swarm/index/meta.json`).
- `--json`: Output as structured JSON.
- If no index exists, print a clear error telling the user to run `scan` first.

## Implementation Plan

### 1. Extension-to-language map

Add a `languageMap` in the `index` package (or a new `summary.go` file within `index/`):

```go
var languageMap = map[string]string{
    ".go":    "Go",
    ".js":    "JavaScript",
    ".ts":    "TypeScript",
    ".tsx":   "TypeScript",
    ".jsx":   "JavaScript",
    ".py":    "Python",
    ".rs":    "Rust",
    ".java":  "Java",
    ".rb":    "Ruby",
    ".c":     "C",
    ".h":     "C",
    ".cpp":   "C++",
    ".hpp":   "C++",
    ".cs":    "C#",
    ".swift": "Swift",
    ".kt":    "Kotlin",
    ".md":    "Markdown",
    ".json":  "JSON",
    ".yaml":  "YAML",
    ".yml":   "YAML",
    ".toml":  "TOML",
    ".html":  "HTML",
    ".css":   "CSS",
    ".scss":  "SCSS",
    ".sh":    "Shell",
    ".sql":   "SQL",
}
```

Unknown extensions fall through to "(other)".

### 2. Entry point detection

A list of known entry point filenames:

```go
var entryPoints = []string{
    "main.go", "main.py", "main.rs", "main.ts", "main.js",
    "index.ts", "index.js", "index.html",
    "app.py", "app.ts", "app.js",
    "manage.py", "Main.java", "Program.cs",
    "server.go", "server.ts", "server.js",
    "cmd/main.go",
}
```

Match against entry `Name` (and `Path` for patterns like `cmd/main.go`).

### 3. Dependency manifest detection

A list of known manifest filenames:

```go
var manifests = []string{
    "go.mod", "go.sum",
    "package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
    "requirements.txt", "setup.py", "pyproject.toml", "Pipfile",
    "Cargo.toml", "Cargo.lock",
    "Gemfile", "Gemfile.lock",
    "pom.xml", "build.gradle", "build.gradle.kts",
    "composer.json",
    "Makefile", "CMakeLists.txt",
}
```

### 4. LOC counting

Add a `Summary` method on `*Index` (in `index/summary.go`) that:

1. Iterates over entries where `Kind == "file"`.
2. For each file, reads the file content from disk (using `idx.Root` + entry `Path`) and counts newlines.
3. Aggregates total LOC.

This is lightweight since we're just counting bytes/newlines, not parsing.

### 5. Wire up the CLI

In `main.go`, add a `case "summary":` block that:

1. Resolves the project root (same as `lookup`).
2. Calls `index.Load(root)`.
3. Calls `idx.Summary()` to get the summary data struct.
4. Prints text or JSON depending on `--json` flag.

### 6. Text output format

```
Project summary for /path/to/project
=====================================

Files: 42 (1,284 LOC)
Packages: 8

Languages:
  Go           28 files  (66.7%)
  Markdown      8 files  (19.0%)
  JSON          4 files   (9.5%)
  YAML          2 files   (4.8%)

Entry points:
  main.go

Dependency manifests:
  go.mod
  go.sum

Top directories:
  cmd/
  index/
  internal/
```

### 7. JSON output structure

```json
{
  "root": "/path/to/project",
  "fileCount": 42,
  "loc": 1284,
  "packageCount": 8,
  "languages": {
    "Go": {"files": 28, "percentage": 66.7},
    "Markdown": {"files": 8, "percentage": 19.0}
  },
  "entryPoints": ["main.go"],
  "manifests": ["go.mod", "go.sum"],
  "topDirectories": ["cmd/", "index/", "internal/"]
}
```

### 8. Tests

Add `index/summary_test.go`:

- **Language mapping**: Verify known extensions map to correct languages.
- **Entry point detection**: Create a temp index with `main.go` and `index.ts` entries, verify they're detected.
- **Manifest detection**: Create a temp index with `go.mod` and `package.json` entries, verify they're detected.
- **LOC counting**: Create temp files with known line counts, scan, run summary, verify LOC is correct.
- **Top directories**: Verify top-level dirs are extracted correctly from entry paths.

## Dependencies

- Requires persisted index (Phase 1) — already implemented.
- No external dependencies needed.

## Files to create/modify

- **Create**: `index/summary.go` — `Summary()` method and supporting types/maps
- **Create**: `index/summary_test.go` — tests for summary logic
- **Modify**: `main.go` — add `case "summary":` command routing
- **Modify**: `README.md` — add `summary` to commands table and move it from roadmap to implemented
- **Modify**: `SKILL.md` — add `summary` command documentation

## Completion Notes

Implemented by agent 6de5af56 (task 705de660). All items completed:

- **Created** `index/summary.go` — `Summary()` method on `*Index`, `FormatSummary()` for text output, `SummaryResult`/`LanguageStat` types, language map (26 extensions), entry point detection (17 names + path patterns), manifest detection (20 names), LOC counting via newline counting.
- **Created** `index/summary_test.go` — 11 tests covering: language mapping, entry point detection, manifest detection, line counting (7 edge cases), LOC from scan, language stats, entry points, manifests, top directories, round-trip through save/load, and FormatSummary text output.
- **Modified** `main.go` — added `case "summary":` with `--root` and `--json` support, updated usage text.
- **Modified** `README.md` — added summary to quick start, commands table, project structure; removed from roadmap.
- **Modified** `SKILL.md` — added summary command examples.
- All 47 tests pass (`go test ./...`). End-to-end CLI test verified both text and JSON output.
