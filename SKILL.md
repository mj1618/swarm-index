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

# Check if the index is out of date (new/deleted/modified files since last scan)
swarm-index stale
swarm-index stale --root ~/code/my-project
```

Use `--json` on any command for structured output. Use `--max N` to limit `lookup` results (default 20).
