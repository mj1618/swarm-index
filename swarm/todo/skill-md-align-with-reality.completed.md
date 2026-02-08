# Align SKILL.md with implemented commands

## Problem

SKILL.md currently references commands that don't exist yet: `summary`, `tree`, `outline`, and `stale`. This will mislead agents that read SKILL.md to understand how to use the tool — they'll try to run commands that fail.

Per CLAUDE.md: "Always keep SKILL.md updated with a minimal set of instructions for agents to make use of the installed tool."

## Current SKILL.md (incorrect)

```bash
swarm-index summary .
swarm-index tree . --depth 3
swarm-index outline src/auth/handler.go
swarm-index stale
```

## What actually exists

Only three commands are implemented:
- `scan <directory>` — index a codebase
- `lookup <query> [--root <dir>] [--max N]` — search the index
- `version` — print version

Global flag: `--json` on any command.

## Changes

1. Update SKILL.md to only reference `scan`, `lookup`, and `version`
2. Include the `--json` and `--max` flags since those are useful for agents
3. Remove references to `summary`, `tree`, `outline`, and `stale`
4. Keep it minimal and accurate — agents should be able to copy-paste and succeed

## Proposed SKILL.md

```markdown
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
```

Use `--json` on any command for structured output. Use `--max N` to limit `lookup` results (default 20).
```

## Verification

- Read SKILL.md and confirm it only references implemented commands
- Run each command mentioned in SKILL.md and confirm they work

## Completion Notes

Completed by agent 14bdb2fd. Updated SKILL.md to match the proposed content exactly. Verified all commands (`scan .`, `lookup "test" --max 5`, `lookup "config" --root .`, `version`) and the `--json` flag all work correctly. Removed references to non-existent `summary`, `tree`, `outline`, and `stale` commands.
