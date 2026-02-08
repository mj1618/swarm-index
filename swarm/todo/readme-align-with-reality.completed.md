# Align README with implemented functionality

## Problem

The README documents ~20 commands but only 3 are implemented (`scan`, `lookup`, `version`). It claims features like fuzzy matching, relevance-ranked results, and AST symbol extraction that don't exist. It also references GitHub Releases and `go install` URLs that don't work yet. A user or agent reading the README will expect a fully-featured tool and be confused when most commands fail with "unknown command."

This is the first thing anyone sees when they encounter the project — it needs to honestly reflect what the tool can do today.

## Specific issues

1. **Commands table lists 16+ unimplemented commands** — `tree`, `summary`, `config`, `deps`, `entry-points`, `show`, `outline`, `exports`, `history`, `context`, `refs`, `search`, `related`, `todos`, `diff-summary`, `stale` are all documented but don't exist.

2. **Quick start examples use unimplemented commands** — `summary`, `tree --depth 3`, `outline` are shown as working examples.

3. **"How it works" section overclaims** — says entries are "parsed" with symbol extraction and that lookup supports "fuzzy matching and relevance-ranked results." In reality, scan only records file entries (no parsing), and lookup does simple case-insensitive substring matching.

4. **Installation section references non-existent releases** — `go install github.com/matt/swarm-index@latest` and GitHub Releases links won't work until the module is published.

5. **lookup command description says "fuzzy and ranked results"** — it's actually a simple substring match with no ranking.

## Changes required

### Quick start
- Remove examples for unimplemented commands (`summary`, `tree`, `outline`)
- Show only `scan`, `lookup`, and `version`

### Commands section
- Remove all unimplemented commands from the commands table
- Keep only: `scan`, `lookup`, `version`
- Update `lookup` description: remove "fuzzy and ranked results", say "case-insensitive substring match"
- Document the `--root` and `--max` flags for `lookup`

### "How it works" section
- Remove claims about symbol parsing/extraction
- Remove claims about fuzzy matching and relevance ranking
- Accurately describe: scan walks directories and records file entries; lookup does substring matching

### Installation section
- Replace `go install` and GitHub Releases with "Build from source" as the primary method (since the module isn't published)
- Keep the build-from-source instructions

### Roadmap
- Add the unimplemented commands to the roadmap section so it's clear they're planned but not yet available
- Keep existing roadmap items (AST parsing, watch mode, etc.)

## Files to modify

- `README.md`

## Acceptance criteria

- Every command documented in the README can actually be run successfully
- No feature claims that aren't implemented (fuzzy matching, symbol extraction, relevance ranking)
- Installation instructions that work for a fresh user (build from source)
- Unimplemented features are in the Roadmap section, not in the Commands section
- Quick start examples all work end-to-end

## Completion notes

All changes made to `README.md`. Verified all acceptance criteria:

1. **Installation**: Removed `go install` and GitHub Releases sections. "Build from source" is now the primary (and only) installation method.
2. **Quick start**: Removed `summary`, `tree --depth 3`, and `outline` examples. Now shows only `scan`, `lookup` (with `--max` and `--root` flags), and `--json`.
3. **Commands**: Collapsed from 6 subsections with 16+ commands to a single table with the 3 implemented commands (`scan`, `lookup`, `version`). Updated `lookup` description to say "case-insensitive substring match" instead of "fuzzy and ranked results". Documented `--root` and `--max` flags.
4. **How it works**: Removed claims about symbol parsing/extraction, fuzzy matching, and relevance ranking. Accurately describes file-only indexing and substring matching.
5. **Roadmap**: Added all 16 unimplemented commands as "Planned commands" and moved AST parsing/fuzzy matching to "Other improvements".
6. **Intro**: Removed "and symbols" from the tagline.

All three commands (`scan`, `lookup`, `version`) tested and working. All flags (`--json`, `--root`, `--max`) tested and working. All tests pass.
