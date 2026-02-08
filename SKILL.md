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
```

Use `--json` on any command for structured output. Use `--max N` to limit `lookup` results (default 20).
