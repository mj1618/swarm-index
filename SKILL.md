# swarm-index

Codebase index and lookup tool. Scan once, then query instantly.

```bash
# Scan the project (required before most commands)
swarm-index scan .

# Unified search across files, symbols, and content
swarm-index locate "handleAuth"

# Look up files and symbols by name (fuzzy-ranked)
swarm-index lookup "config" --max 10

# Regex search across file contents
swarm-index search "func\s+\w+" --max 10

# Read a file with line numbers
swarm-index show main.go
swarm-index show main.go --lines 10:20

# Top-level symbols of a file
swarm-index outline main.go

# Symbol's full definition with imports and doc comments
swarm-index context Save index/index.go

# Find all references to a symbol
swarm-index refs "HandleAuth"

# Files connected to a given file (imports, importers, tests)
swarm-index related main.go

# Exported/public symbols of a file or directory
swarm-index exports index/index.go

# Project overview (languages, LOC, entry points)
swarm-index summary

# Directory tree
swarm-index tree . --depth 3

# What changed since a git ref, with affected symbols
swarm-index diff-summary
swarm-index diff-summary main

# Blast radius of a symbol or file
swarm-index impact Load
swarm-index impact index/index.go

# Check if the index needs re-scanning
swarm-index stale
```

Use `--json` on any command for structured output. Use `--max N` to limit results.
