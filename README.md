# swarm-index

A fast codebase index and lookup tool designed for coding agents. Scan a project directory, build a lightweight index of its files and symbols, and query it instantly — so agents spend less time searching and more time coding.

## Why

Coding agents (LLM-powered or otherwise) waste significant context windows and API calls exploring unfamiliar codebases. **swarm-index** gives them a pre-built map: scan once, look up anything by name, and get back precise file paths and locations.

## Installation

### Go install (recommended)

```bash
go install github.com/matt/swarm-index@latest
```

This puts the `swarm-index` binary in your `$GOPATH/bin` (or `$HOME/go/bin` by default). Make sure that directory is in your `PATH`.

### Download a prebuilt binary

Grab the latest release for your platform from [GitHub Releases](https://github.com/matt/swarm-index/releases/latest), extract it, and move the binary somewhere on your `PATH`:

```bash
# Example for macOS (Apple Silicon)
curl -sL https://github.com/matt/swarm-index/releases/latest/download/swarm-index_darwin_arm64.tar.gz | tar xz
sudo mv swarm-index /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/matt/swarm-index.git
cd swarm-index
go build -o swarm-index .
```

## Quick start

```bash
# Scan a project and persist the index
swarm-index scan ~/code/my-project

# Get a quick orientation
swarm-index summary ~/code/my-project
swarm-index tree ~/code/my-project --depth 3

# Look up a symbol or filename
swarm-index lookup "handleAuth"

# Understand a file without reading the whole thing
swarm-index outline src/auth/handler.go

# All commands support --json for structured output
swarm-index outline src/auth/handler.go --json
```

## Global flags

| Flag | Description |
|---|---|
| `--json` | Output structured JSON instead of human-readable text. Supported by every command. |

## Commands

### Indexing

| Command | Description |
|---|---|
| `scan <directory>` | Walk a directory tree, index all source files and symbols, and persist the index to disk |
| `stale` | Compare the current index against the filesystem and report new, deleted, or modified files since the last scan |

### Orientation

| Command | Description |
|---|---|
| `tree <directory>` | Print the directory structure, respecting the same skip rules as `scan`. Use `--depth N` to limit depth |
| `summary <directory>` | Auto-detect languages, file counts by extension, entry points, dependency manifests, and total lines of code |
| `config` | Detect the project's toolchain — framework, build tool, test runner, linter, formatter — and how to invoke each |
| `deps` | Parse dependency manifests (`go.mod`, `package.json`, `requirements.txt`, etc.) and list all external libraries with versions |
| `entry-points` | Find main functions, HTTP route handlers, CLI command definitions, event handlers, and exported module entry points |

### Understanding a file

| Command | Description |
|---|---|
| `show <path> [--lines M:N]` | Read a file or line range with line numbers and structural context (e.g. which function encloses the viewed lines) |
| `outline <file>` | Show the structural skeleton of a file — functions, classes, types, exports, imports — without the full source |
| `exports <file\|package>` | List the public API surface of a file or package — only exported/public symbols |
| `history <file>` | Show recent git commits that touched a file, with one-line summaries and dates |

### Understanding a symbol

| Command | Description |
|---|---|
| `lookup <query>` | Search the index for files or symbols matching a query (case-insensitive substring match, with fuzzy and ranked results) |
| `context <symbol>` | Show a symbol's definition along with its imports, enclosing type/class, and doc comments — the minimum viable context to understand it |
| `refs <symbol>` | Show everywhere a symbol is used (callers and consumers), not just where it's defined |

### Navigating code

| Command | Description |
|---|---|
| `search <pattern>` | Regex search across file contents, using the index's skip rules and returning structured results with file, line, and match context |
| `related <file>` | Show files connected to a given file: what it imports, what imports it, and its test file if one exists |
| `todos` | Collect all `TODO`, `FIXME`, `HACK`, and `XXX` comments across the codebase with file locations |

### Change-awareness

| Command | Description |
|---|---|
| `diff-summary [git-ref]` | Show files changed since a git ref (default `HEAD~1`) and which indexed symbols were affected |

### Meta

| Command | Description |
|---|---|
| `version` | Print the current version |

## How it works

1. **Scan** recursively walks the target directory, collecting every file while automatically skipping noise directories (`.git`, `node_modules`, `vendor`, `__pycache__`, `dist`, `build`, hidden dirs, etc.). The index is persisted to disk so subsequent commands work without re-scanning.

2. Each file is parsed and recorded as one or more **Entries** with:
   - `Name` — the file or symbol name
   - `Kind` — what it is (`file`, `func`, `type`, `package`, ...)
   - `Path` — path relative to the scanned root
   - `Line` — line number, when applicable
   - `Package` — the package or directory it belongs to

3. **Lookup** and other query commands perform case-insensitive matching across all entries, with support for fuzzy matching and relevance-ranked results, returning output formatted for quick consumption (or `--json` for structured agent consumption).

## Project structure

```
swarm-index/
├── main.go              # CLI entrypoint and command routing
├── index/
│   ├── index.go         # Core library: scanning, indexing, matching
│   └── index_test.go    # Tests for scan, match, and directory filtering
├── go.mod               # Go module definition
└── README.md
```

## Running tests

```bash
go test ./... -v
```

## Roadmap

- [ ] AST parsing for more languages (Go, Python, JS/TS, Rust, Java)
- [ ] Watch mode to keep the index up to date as files change
- [ ] Support for ignoring custom paths via config file
- [ ] Language-aware symbol resolution for `context` and `refs`
- [ ] MCP server mode for direct integration with coding agents

## Requirements

- Go 1.22+

## License

MIT
