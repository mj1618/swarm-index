---
name: swarm-index
description: Navigate and search codebases using swarm-index, a fast CLI tool that scans projects and builds a symbol/file index. Use when exploring an unfamiliar codebase, looking up symbols or files, or orienting in a new project.
---

# swarm-index

A CLI tool that scans a project directory, builds a lightweight index of files and symbols, and lets you query it instantly. Scan once, look up anything by name.

## Prerequisites

The `swarm-index` binary must be installed and on your PATH. Verify with:

```bash
swarm-index version
```

If not installed, build from source (`go build -o swarm-index .` in the repo) or download a release from GitHub.

## Core Workflow

### 1. Scan the project (required first step)

Before any lookups, scan the project root to build the index:

```bash
swarm-index scan /path/to/project
```

This persists the index to `<project>/swarm/index/`. You only need to re-scan when files change significantly.

### 2. Look up symbols or files

```bash
swarm-index lookup "handleAuth"
swarm-index lookup "config.yaml"
swarm-index lookup "UserService"
```

Lookup performs case-insensitive substring matching with fuzzy and ranked results. It matches against file names, symbol names, and paths.

### 3. Check for stale index

```bash
swarm-index stale
```

Reports new, deleted, or modified files since the last scan. Re-scan if the index is stale.

## Orientation Commands

When first entering a codebase, use these to get oriented quickly:

```bash
# Project overview: languages, file counts, entry points, LOC
swarm-index summary /path/to/project

# Directory structure (respects skip rules)
swarm-index tree /path/to/project --depth 3

# Toolchain detection: framework, build tool, test runner, linter
swarm-index config

# External dependencies with versions
swarm-index deps

# Main functions, HTTP handlers, CLI commands, exports
swarm-index entry-points
```

## Understanding Files

```bash
# Structural skeleton: functions, classes, types, imports, exports
swarm-index outline src/auth/handler.go

# Read a file or line range with structural context
swarm-index show src/auth/handler.go
swarm-index show src/auth/handler.go --lines 40:60

# Public API surface of a file or package
swarm-index exports src/auth/

# Recent git history for a file
swarm-index history src/auth/handler.go
```

## Understanding Symbols

```bash
# Find where a symbol is defined (fuzzy, ranked)
swarm-index lookup "handleAuth"

# Full context: definition + imports + enclosing type + doc comments
swarm-index context handleAuth

# Find all callers/consumers of a symbol
swarm-index refs handleAuth
```

## Navigating Code

```bash
# Regex search across file contents
swarm-index search "TODO|FIXME"

# Files connected to a given file (imports, importers, test file)
swarm-index related src/auth/handler.go

# Collect all TODO/FIXME/HACK/XXX comments
swarm-index todos
```

## Change Awareness

```bash
# Files changed since a git ref, with affected symbols
swarm-index diff-summary
swarm-index diff-summary HEAD~5
```

## Key Flags

| Flag | Description |
|---|---|
| `--json` | Structured JSON output (supported by every command) |
| `--root <dir>` | Specify the project root explicitly (for `lookup`) |
| `--max N` | Limit number of results (default: 20, for `lookup`) |
| `--depth N` | Limit directory depth (for `tree`) |
| `--lines M:N` | Specify line range (for `show`) |

## Index Auto-Discovery

When you run `lookup` without `--root`, swarm-index walks up from the current working directory looking for `swarm/index/meta.json`. Run commands from within the project directory and it will find the index automatically.

## Recommended Agent Workflow

1. **On entering a new project**: Run `swarm-index scan .` then `swarm-index summary .` and `swarm-index tree . --depth 2`
2. **When searching for code**: Use `swarm-index lookup` before resorting to grep/find
3. **When understanding a file**: Use `swarm-index outline` to see structure without reading the whole file
4. **When understanding a symbol**: Use `swarm-index context` for definition + surrounding context
5. **When checking what changed**: Use `swarm-index diff-summary` to see affected files and symbols
6. **Prefer `--json`** when parsing output programmatically
