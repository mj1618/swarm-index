# swarm-index

A fast codebase index and lookup tool designed for coding agents. Scan a project directory, build a lightweight index of its files, and query it instantly — so agents spend less time searching and more time coding.

## Why

Coding agents (LLM-powered or otherwise) waste significant context windows and API calls exploring unfamiliar codebases. **swarm-index** gives them a pre-built map: scan once, look up anything by name, and get back precise file paths and locations.

## Installation

### Build from source

```bash
git clone https://github.com/matt/swarm-index.git
cd swarm-index
go build -o swarm-index .
```

This produces a `swarm-index` binary in the current directory. Move it somewhere on your `PATH` to use it globally.

## Quick start

```bash
# Scan a project and persist the index
swarm-index scan ~/code/my-project

# Look up a symbol or filename
swarm-index lookup "handleAuth"

# Limit results
swarm-index lookup "test" --max 5

# Point lookup at a specific project root
swarm-index lookup "config" --root ~/code/my-project

# Regex search across file contents
swarm-index search "func\s+\w+" --max 10

# Project overview (languages, LOC, entry points, manifests)
swarm-index summary

# Print directory tree
swarm-index tree . --depth 3

# Read a file with line numbers
swarm-index show main.go

# Read specific lines
swarm-index show main.go --lines 10:20

# Find all references to a symbol
swarm-index refs "HandleAuth"

# Show top-level symbols of a file (functions, types, etc.)
swarm-index outline main.go

# Show a symbol's full definition with imports and doc comments
swarm-index context Save index/index.go
swarm-index context handleAuth server.go --root ~/code/my-project

# List exported/public symbols of a file or directory
swarm-index exports index/index.go
swarm-index exports parsers

# Find imports, importers, and test files for a file
swarm-index related main.go

# Find TODO/FIXME/HACK/XXX comments
swarm-index todos

# Filter by tag
swarm-index todos --tag FIXME

# List dependencies from manifest files
swarm-index deps

# List dependencies as JSON
swarm-index deps --json

# Show what changed since the last commit
swarm-index diff-summary

# Show what changed since a specific ref
swarm-index diff-summary main

# Show git blame for a file (line-level attribution)
swarm-index blame main.go

# Blame specific lines
swarm-index blame main.go --lines 10:20

# Show recent commits for a file
swarm-index history main.go

# Limit to last 3 commits
swarm-index history main.go --max 3

# Show most frequently changed files (hotspots)
swarm-index hotspots

# Limit to top 10
swarm-index hotspots --max 10

# Only changes in the last 6 months
swarm-index hotspots --since "6 months ago"

# Filter to a specific directory
swarm-index hotspots --path src/

# Detect project toolchain (framework, build, test, lint, format)
swarm-index config

# Find entry points (main functions, route handlers, CLI commands, init functions)
swarm-index entry-points

# Filter by kind
swarm-index entry-points --kind route

# Limit results
swarm-index entry-points --max 10

# Show project-wide import dependency graph
swarm-index graph

# Focus on a specific file's dependency neighborhood
swarm-index graph --focus main.go

# Limit traversal depth when focused
swarm-index graph --focus main.go --depth 2

# Output as Graphviz DOT format
swarm-index graph --format dot

# Search for symbols by name across the entire project
swarm-index symbols "auth"

# Filter by symbol kind (func, type, class, interface, method, const, var)
swarm-index symbols "Handle" --kind func

# Limit results
swarm-index symbols "Config" --max 10

# Analyze code complexity per function (top N most complex)
swarm-index complexity

# Complexity for a specific file
swarm-index complexity main.go

# Filter by minimum complexity threshold
swarm-index complexity --min 5

# Limit results
swarm-index complexity --max 10

# Detect potentially unused exports (dead code candidates)
swarm-index dead-code

# Filter by symbol kind
swarm-index dead-code --kind func

# Limit analysis to a specific directory
swarm-index dead-code --path src/utils

# Limit results
swarm-index dead-code --max 10

# Check if the index is out of date
swarm-index stale

# All commands support --json for structured output
swarm-index lookup "handleAuth" --json
```

## Global flags

| Flag | Description |
|---|---|
| `--json` | Output structured JSON instead of human-readable text. Supported by every command. |

## Commands

| Command | Description |
|---|---|
| `scan <directory>` | Walk a directory tree, index all source files, and persist the index to disk. Prints file counts and language breakdown. |
| `lookup <query> [--root <dir>] [--max N]` | Search the index for files matching a query by case-insensitive substring match. Use `--root` to specify the project root and `--max` to limit results (default 20). |
| `search <pattern> [--root <dir>] [--max N]` | Regex search across indexed file contents. Returns matching lines with file paths and line numbers. Use `--max` to limit results (default 50). Binary files are skipped. |
| `summary [--root <dir>]` | Show a project overview: language breakdown, file count, LOC, entry points, dependency manifests, and top-level directories. Requires a prior `scan`. |
| `tree <directory> [--depth N]` | Print the directory structure of a project, respecting the same skip rules as `scan`. Use `--depth` to limit depth (default unlimited). Supports `--json`. |
| `show <path> [--lines M:N]` | Read a file with line numbers. Use `--lines M:N` to show a specific range (1-indexed, inclusive). Supports formats: `M:N`, `M:`, `:N`, `M`. Binary files are rejected. |
| `refs <symbol> [--root <dir>] [--max N]` | Find all references to a symbol across indexed files. Shows the definition and all usage sites, grouped by file. Uses word-boundary matching and heuristic definition detection. Default max 50. |
| `outline <file>` | Show top-level symbols (functions, types, structs, interfaces, methods, constants, variables) with line numbers and signatures. Supports Go, Python, JavaScript, and TypeScript files. |
| `exports <file\|directory> [--root <dir>]` | List exported/public symbols of a file or package directory. Uses language-aware parsers to identify exports (Go: uppercase names, JS/TS: `export` keyword, Python: names not starting with `_`). Supports `--json`. |
| `context <symbol> <file> [--root <dir>]` | Show a symbol's full definition context: file imports, doc comments, and the complete definition body. Supports Go, Python, JS, and TS files. |
| `related <file> [--root <dir>]` | Show files connected to a given file: imports (files it depends on), importers (files that depend on it), and associated test files. Supports Go, JS/TS, and Python import resolution. |
| `graph [--root <dir>] [--format dot\|list] [--focus <file>] [--depth N]` | Show project-wide import dependency graph with fan-in/fan-out analysis. Use `--focus` to extract a subgraph around a specific file, `--depth` to limit traversal, and `--format dot` for Graphviz output. Requires a prior `scan`. |
| `todos [--root <dir>] [--max N] [--tag TAG]` | Find TODO, FIXME, HACK, and XXX comments across indexed files. Use `--tag` to filter by tag type and `--max` to limit results (default 100). |
| `deps [--root <dir>]` | Parse dependency manifests (go.mod, package.json, requirements.txt, Cargo.toml, pyproject.toml) and list all declared dependencies with version constraints. Requires a prior `scan`. |
| `entry-points [--root <dir>] [--max N] [--kind KIND]` | Find executable entry points: main functions, HTTP route handlers, CLI command registrations, and init/bootstrap code. Supports Go, Python, JS/TS, Rust, and Java. Use `--kind` to filter (main, route, cli, init). Default max 100. Requires a prior `scan`. |
| `config [--root <dir>]` | Detect the project toolchain: primary language, framework, build/test/lint/format tools, package manager, and package.json scripts. Requires a prior `scan`. |
| `diff-summary [git-ref] [--root <dir>]` | Show files changed since a git ref (default `HEAD~1`) and list affected symbols in added/modified files. Requires `git` and a prior `scan`. Renames are treated as deleted + added. |
| `blame <file> [--lines M:N] [--root <dir>]` | Show git blame for a file with line-level attribution: commit hash, date, author, and line content. Use `--lines M:N` to blame a specific range. Does not require a prior `scan`. |
| `history <file> [--root <dir>] [--max N]` | Show recent git commits that touched a file. Displays hash, date, author, and subject. Default max 10. Does not require a prior `scan`. |
| `hotspots [--root <dir>] [--max N] [--since <time>] [--path <prefix>]` | Rank files by git commit frequency to find the most actively changed files. Use `--since` to limit to recent history (e.g. "6 months ago") and `--path` to filter by directory prefix. Default max 20. Requires `git` and a prior `scan`. |
| `symbols <query> [--root <dir>] [--max N] [--kind KIND]` | Search all parseable files for symbols (functions, types, classes, etc.) matching the query by name. Case-insensitive substring match. Use `--kind` to filter by symbol kind and `--max` to limit results (default 50). Requires a prior `scan`. |
| `complexity [file] [--root <dir>] [--max N] [--min N]` | Analyze code complexity per function/method. Shows cyclomatic complexity, line count, nesting depth, and parameter count. Sorted by complexity descending. Use `--min` to filter by threshold and `--max` to limit results (default 20). Supports Go, Python, JS/TS. Single-file mode does not require a prior `scan`. |
| `dead-code [--root <dir>] [--max N] [--kind KIND] [--path PREFIX]` | Detect potentially unused exported symbols. Parses all files to collect exported symbols, then searches the entire codebase for references. Symbols with zero external references are reported as dead code candidates. Excludes main/init, Test*/Benchmark*/Example* functions, and test files. Use `--kind` to filter by symbol kind and `--path` to scope analysis to a directory prefix. Default max 50. Requires a prior `scan`. |
| `stale [--root <dir>]` | Check if the index is out of date by comparing against the filesystem. Reports new, deleted, and modified files since the last scan. |
| `version` | Print the current version |

## How it works

1. **Scan** recursively walks the target directory, recording every file while automatically skipping noise directories (`.git`, `node_modules`, `vendor`, `__pycache__`, `dist`, `build`, hidden dirs, etc.). It also skips any `swarm/index/` directory to avoid indexing its own output. The index is persisted to disk so subsequent commands work without re-scanning.

2. Each file is recorded as an **Entry** with:
   - `Name` — the filename
   - `Kind` — currently always `file`
   - `Path` — path relative to the scanned root
   - `Package` — the parent directory

3. **Lookup** performs case-insensitive substring matching across all entries and returns results formatted for quick consumption (or `--json` for structured agent consumption).

## Project structure

```
swarm-index/
├── main.go              # CLI entrypoint and command routing
├── index/
│   ├── index.go         # Core library: scanning, indexing, matching
│   ├── index_test.go    # Tests for scan, match, and directory filtering
│   ├── refs.go          # Symbol reference finder (definition + usages)
│   ├── refs_test.go     # Tests for refs functionality
│   ├── related.go       # File dependency neighborhood (imports, importers, tests)
│   ├── related_test.go  # Tests for related functionality
│   ├── search.go        # Regex search across indexed file contents
│   ├── search_test.go   # Tests for search functionality
│   ├── summary.go       # Project summary: languages, LOC, entry points
│   ├── summary_test.go  # Tests for summary logic
│   ├── show.go          # File reading with line numbers
│   ├── show_test.go     # Tests for show functionality
│   ├── deps.go          # Dependency manifest parsing (go.mod, package.json, etc.)
│   ├── deps_test.go     # Tests for deps functionality
│   ├── diffsummary.go   # Git diff summary with affected symbols
│   ├── diffsummary_test.go # Tests for diff summary
│   ├── symbols.go       # Project-wide symbol search by name
│   ├── symbols_test.go  # Tests for symbols functionality
│   ├── stale.go         # Stale index detection (new/deleted/modified files)
│   ├── stale_test.go    # Tests for stale detection
│   ├── complexity.go    # Code complexity analysis per function
│   ├── complexity_test.go # Tests for complexity functionality
│   ├── config.go        # Project toolchain detection (framework, build, test, lint)
│   ├── config_test.go   # Tests for config functionality
│   ├── context.go       # Symbol definition context (imports, doc comments, body)
│   ├── context_test.go  # Tests for context functionality
│   ├── deadcode.go      # Dead code detection (unused exported symbols)
│   ├── deadcode_test.go # Tests for dead code functionality
│   ├── entrypoints.go   # Entry point detection (main, routes, CLI, init)
│   ├── entrypoints_test.go # Tests for entry points functionality
│   ├── exports.go       # Exported/public symbol listing
│   ├── exports_test.go  # Tests for exports functionality
│   ├── graph.go         # Project-wide import dependency graph
│   ├── graph_test.go    # Tests for graph functionality
│   ├── blame.go         # Git blame (line-level attribution)
│   ├── blame_test.go    # Tests for blame functionality
│   ├── history.go       # Git commit history for a file
│   ├── history_test.go  # Tests for history functionality
│   ├── hotspots.go      # Most frequently changed files ranking
│   ├── hotspots_test.go # Tests for hotspots functionality
│   ├── todos.go         # TODO/FIXME/HACK/XXX comment collection
│   ├── todos_test.go    # Tests for todos functionality
│   ├── tree.go          # Directory tree building and rendering
│   └── tree_test.go     # Tests for tree functionality
├── parsers/
│   ├── parsers.go       # Symbol type, Parser interface, and registry
│   ├── goparser.go      # Go AST parser implementation
│   ├── goparser_test.go # Tests for Go parser
│   ├── pyparser.go      # Python heuristic parser (regex-based)
│   ├── pyparser_test.go # Tests for Python parser
│   ├── jsparser.go      # JS/TS heuristic parser (regex + brace tracking)
│   └── jsparser_test.go # Tests for JS/TS parser
├── go.mod               # Go module definition
└── README.md
```

## Running tests

```bash
go test ./... -v
```

## Roadmap

### Planned commands

- [x] `outline` — structural skeleton of a file (functions, classes, types)
- [x] `show` — read a file or line range with line numbers
- [x] `exports` — public API surface of a file or package
- [x] `config` — detect project toolchain (framework, build tool, test runner)
- [x] `deps` — parse dependency manifests and list libraries with versions
- [x] `entry-points` — find main functions, route handlers, CLI commands
- [x] `context` — symbol definition with imports and doc comments
- [x] `refs` — find all usages of a symbol
- [x] `related` — files connected to a given file (imports, importers, tests)
- [x] `todos` — collect TODO/FIXME/HACK/XXX comments
- [x] `diff-summary` — files changed since a git ref with affected symbols
- [x] `stale` — report new, deleted, or modified files since last scan
- [x] `history` — recent git commits that touched a file
- [x] `hotspots` — most frequently changed files ranked by commit count
- [x] `graph` — project-wide import dependency graph with fan-in/fan-out analysis

### Other improvements

- [ ] AST parsing for symbol extraction (Rust, Java) — Go, Python, and JS/TS already supported
- [ ] Fuzzy matching and relevance-ranked results for `lookup`
- [ ] Watch mode to keep the index up to date as files change
- [ ] Support for ignoring custom paths via config file
- [ ] Language-aware symbol resolution for `context` and `refs`
- [ ] MCP server mode for direct integration with coding agents

## Requirements

- Go 1.22+

## License

MIT
