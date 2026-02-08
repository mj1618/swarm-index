# Using swarm-index (for agents)

Scan a project to build an index, then look up files instantly.

```bash
# First-time setup: scan the project
swarm-index scan .

# Look up files by name or path
swarm-index lookup "handleAuth"

# Limit results
swarm-index lookup "test" --max 5

# Point lookup at a specific project root
swarm-index lookup "config" --root ~/code/my-project

# Regex search across file contents
swarm-index search "func\s+\w+" --max 10

# Project overview (languages, LOC, entry points, manifests)
swarm-index summary

# Point summary at a specific project root
swarm-index summary --root ~/code/my-project

# Print directory tree (respects same skip rules as scan)
swarm-index tree . --depth 3

# Read a file with line numbers
swarm-index show main.go

# Read specific lines (1-indexed, inclusive)
swarm-index show main.go --lines 10:20

# Find all references to a symbol (definition + usages)
swarm-index refs "HandleAuth"

# Limit refs results
swarm-index refs "Config" --max 10

# Show top-level symbols of a file (functions, types, methods, classes, etc.)
# Supports Go (.go), Python (.py), JS (.js, .jsx), and TS (.ts, .tsx) files
swarm-index outline main.go
swarm-index outline app.py
swarm-index outline app.tsx

# Show a symbol's full definition with imports and doc comments
swarm-index context Save index/index.go
swarm-index context handleAuth server.go --root ~/code/my-project

# List exported/public symbols of a file or directory
swarm-index exports index/index.go
swarm-index exports parsers
swarm-index exports src/utils --root ~/code/my-project

# Find imports, importers, and test files for a file
swarm-index related main.go
swarm-index related src/utils.ts --root ~/code/my-project

# List dependencies from manifest files (go.mod, package.json, etc.)
swarm-index deps
swarm-index deps --root ~/code/my-project

# Find TODO/FIXME/HACK/XXX comments across the codebase
swarm-index todos
swarm-index todos --tag FIXME
swarm-index todos --max 20

# Show files changed since a git ref with affected symbols
swarm-index diff-summary
swarm-index diff-summary HEAD~3
swarm-index diff-summary main --root ~/code/my-project

# Show recent git commits for a file
swarm-index history main.go
swarm-index history main.go --max 3
swarm-index history src/utils.ts --root ~/code/my-project

# Check if the index is out of date (new/deleted/modified files since last scan)
swarm-index stale
swarm-index stale --root ~/code/my-project
```

Use `--json` on any command for structured output. Use `--max N` to limit `lookup` results (default 20).
