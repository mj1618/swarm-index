# Feature: `test-map` command

## Summary

Add a `test-map` command that provides a project-wide view of source-to-test-file associations. It shows which source files have associated test files, which don't, and calculates an overall test-file coverage ratio. This helps agents quickly identify untested code areas and find the right test file when adding tests.

## Motivation

Agents frequently need to:
- Find the test file for a given source file before making changes
- Identify which parts of the codebase lack test coverage (by file association)
- Know where to add new tests when implementing features

The existing `related` command shows test files for a **single** file. `test-map` gives the **project-wide** picture in one call, making it easy for agents to assess overall testing hygiene and find gaps.

## CLI Interface

```bash
# Show full test map for the project
swarm-index test-map

# Filter to a specific directory
swarm-index test-map --path src/

# Show only files missing tests
swarm-index test-map --untested

# Show only files that have tests
swarm-index test-map --tested

# Limit results
swarm-index test-map --max 50

# JSON output
swarm-index test-map --json
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--root` | auto-detect | Project root |
| `--path` | (all) | Filter to files under this directory prefix |
| `--untested` | false | Show only source files without associated tests |
| `--tested` | false | Show only source files with associated tests |
| `--max` | 100 | Maximum number of entries to display |
| `--json` | false | Structured JSON output |

## Output Format

### Text output

```
Test Map (42/68 source files have tests — 61.8%)

Tested:
  index/index.go          → index/index_test.go
  index/search.go         → index/search_test.go
  parsers/goparser.go     → parsers/goparser_test.go
  ...

Untested:
  index/stale.go          (no test file found)
  main.go                 (no test file found)
  ...
```

### JSON output

```json
{
  "summary": {
    "totalSourceFiles": 68,
    "testedFiles": 42,
    "untestedFiles": 26,
    "coverageRatio": 0.618
  },
  "entries": [
    {
      "sourceFile": "index/index.go",
      "testFile": "index/index_test.go",
      "hasTesting": true
    },
    {
      "sourceFile": "main.go",
      "testFile": "",
      "hasTesting": false
    }
  ]
}
```

## Implementation Plan

### 1. Add `TestMapEntry` and `TestMapResult` types in `index/testmap.go`

```go
type TestMapEntry struct {
    SourceFile string `json:"sourceFile"`
    TestFile   string `json:"testFile"`
    HasTest    bool   `json:"hasTest"`
}

type TestMapSummary struct {
    TotalSourceFiles int     `json:"totalSourceFiles"`
    TestedFiles      int     `json:"testedFiles"`
    UntestedFiles    int     `json:"untestedFiles"`
    CoverageRatio    float64 `json:"coverageRatio"`
}

type TestMapResult struct {
    Summary TestMapSummary  `json:"summary"`
    Entries []TestMapEntry  `json:"entries"`
}
```

### 2. Implement `TestMap` method on `*Index`

- Iterate over all indexed file entries
- Skip test files themselves, non-source files (binary, config, etc.)
- For each source file, detect the associated test file using language-specific naming conventions:
  - **Go**: `foo.go` → `foo_test.go` (same directory)
  - **Python**: `foo.py` → `test_foo.py` or `foo_test.py` (same directory, or `tests/` subdirectory)
  - **JS/TS**: `foo.ts` → `foo.test.ts`, `foo.spec.ts` (same directory, or `__tests__/foo.ts`)
  - **Rust**: check for `#[cfg(test)]` module or `tests/` directory
  - **Java**: `Foo.java` → `FooTest.java` (in test source tree)
- Check if the candidate test file exists in the index
- Apply `--path` prefix filter
- Apply `--untested` / `--tested` filter
- Sort by path

### 3. Add `FormatTestMap` function

- Text output showing tested/untested sections with the source→test mapping
- Include summary line with coverage ratio

### 4. Wire up CLI in `main.go`

- Register `test-map` subcommand
- Parse `--root`, `--path`, `--untested`, `--tested`, `--max`, `--json` flags
- Call `idx.TestMap(...)` and format output

### 5. Tests in `index/testmap_test.go`

- Test Go naming convention detection
- Test Python naming convention detection (both `test_foo.py` and `foo_test.py`)
- Test JS/TS naming convention detection (`.test.ts`, `.spec.ts`, `__tests__/`)
- Test `--path` prefix filtering
- Test `--untested` and `--tested` filters
- Test summary statistics calculation
- Test JSON output structure

### 6. Update README.md and SKILL.md

- Add `test-map` to the commands table
- Add usage examples
- Move from roadmap to completed if applicable

## Dependencies

- Requires a prior `scan` (reads from persisted index)
- No new external dependencies
- Reuses existing language detection from `index.go` and naming patterns from `related.go`

## Complexity Estimate

Medium — the core logic (test file detection by naming convention) already partially exists in `related.go`. This feature extends that logic to operate project-wide and adds summary statistics.

## Completion Notes

Implemented by agent bfa5d20c. All deliverables completed:

1. **`index/testmap.go`** — Core `TestMap` method on `*Index` with types `TestMapEntry`, `TestMapSummary`, `TestMapResult`. Reuses `findTestFiles` from `related.go` for test detection. Supports `--path`, `--untested`, `--tested`, and `--max` filtering. Includes `FormatTestMap` for text output and `isTestFilePath` helper.

2. **`main.go`** — `test-map` subcommand wired with all flags (`--root`, `--path`, `--untested`, `--tested`, `--max`, `--json`). Added to usage help.

3. **`index/testmap_test.go`** — 13 test functions covering Go/Python/JS/TS naming conventions, path filtering, untested/tested filters, max limit, empty input, sort order, `isTestFilePath`, and format output.

4. **README.md** — Added Quick Start examples, Commands table entry, and project structure entry.

5. **SKILL.md** — Added usage examples.

All tests pass (`go test ./...`). Manual CLI testing verified text output, JSON output, and all flags.
