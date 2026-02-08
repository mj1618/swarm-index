# Agent Guide

## Codebase Index

This project uses [swarm-index](https://github.com/matt/swarm-index) to give you a fast map of the codebase. Use it instead of crawling directories or grepping blindly. All commands support `--json` for structured output.

### Usage

```bash
# Scan the project first (only needed once per session)
swarm-index scan .

# Then look up any symbol, filename, or keyword
swarm-index lookup "handleAuth"
```

**Always run `scan` first, then use `lookup` before exploring files by hand.** It returns file paths and line numbers so you can jump straight to the right place.

### Commands

#### Indexing

| Command | Description |
|---|---|
| `scan <directory>` | Scan and index a codebase. Run once per session. |
| `stale` | Report new, deleted, or modified files since the last scan. |

#### Orientation

| Command | Description |
|---|---|
| `tree <directory>` | Print the directory structure. Use `--depth N` to limit depth. |
| `summary <directory>` | Languages, file counts, entry points, dependency manifests, and total LOC. |
| `config` | Detect the project's toolchain — framework, build tool, test runner, linter, formatter. |
| `deps` | List all external dependencies with versions from manifest files. |
| `entry-points` | Find main functions, route handlers, CLI commands, and module entry points. |

#### Understanding a file

| Command | Description |
|---|---|
| `show <path> [--lines M:N]` | Read a file or line range with line numbers and structural context. |
| `outline <file>` | Structural skeleton — functions, classes, types, exports, imports — without full source. |
| `exports <file\|package>` | List the public API surface — only exported/public symbols. |
| `history <file>` | Recent git commits that touched a file, with summaries and dates. |

#### Understanding a symbol

| Command | Description |
|---|---|
| `lookup <query>` | Search the index for files or symbols matching a query. |
| `context <symbol>` | Show a symbol's definition, imports, enclosing type, and doc comments. |
| `refs <symbol>` | Show everywhere a symbol is used (callers/consumers). |

#### Navigating code

| Command | Description |
|---|---|
| `search <pattern>` | Regex search across file contents with structured results. |
| `related <file>` | Files connected to a given file: imports, importers, and its test file. |
| `todos` | Collect all `TODO`, `FIXME`, `HACK`, and `XXX` comments with locations. |

#### Change-awareness

| Command | Description |
|---|---|
| `diff-summary [git-ref]` | Files changed since a git ref (default `HEAD~1`) and affected symbols. |

#### Meta

| Command | Description |
|---|---|
| `version` | Print version info. |
