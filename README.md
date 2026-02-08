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
│   ├── search.go        # Regex search across indexed file contents
│   ├── search_test.go   # Tests for search functionality
│   ├── summary.go       # Project summary: languages, LOC, entry points
│   ├── summary_test.go  # Tests for summary logic
│   ├── tree.go          # Directory tree building and rendering
│   └── tree_test.go     # Tests for tree functionality
├── go.mod               # Go module definition
└── README.md
```

## Running tests

```bash
go test ./... -v
```

## Roadmap

### Planned commands

- [ ] `outline` — structural skeleton of a file (functions, classes, types)
- [ ] `show` — read a file or line range with structural context
- [ ] `exports` — public API surface of a file or package
- [ ] `config` — detect project toolchain (framework, build tool, test runner)
- [ ] `deps` — parse dependency manifests and list libraries with versions
- [ ] `entry-points` — find main functions, route handlers, CLI commands
- [ ] `context` — symbol definition with imports and doc comments
- [ ] `refs` — find all usages of a symbol
- [ ] `related` — files connected to a given file (imports, importers, tests)
- [ ] `todos` — collect TODO/FIXME/HACK/XXX comments
- [ ] `diff-summary` — files changed since a git ref with affected symbols
- [ ] `stale` — report new, deleted, or modified files since last scan
- [ ] `history` — recent git commits that touched a file

### Other improvements

- [ ] AST parsing for symbol extraction (Go, Python, JS/TS, Rust, Java)
- [ ] Fuzzy matching and relevance-ranked results for `lookup`
- [ ] Watch mode to keep the index up to date as files change
- [ ] Support for ignoring custom paths via config file
- [ ] Language-aware symbol resolution for `context` and `refs`
- [ ] MCP server mode for direct integration with coding agents

## Requirements

- Go 1.22+

## License

MIT
