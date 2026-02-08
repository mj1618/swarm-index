# Feature: `config` command — detect project toolchain

## Problem

When an agent starts working on an unfamiliar codebase, it needs to quickly answer: What framework is this? How do I run tests? How do I build? What linter/formatter is configured? Today agents waste context window and API calls grepping through manifests and config files to figure this out.

## Solution

Add a `config` command that auto-detects the project's toolchain by inspecting manifest files and config files already known to the index.

```
swarm-index config [--root <dir>]
swarm-index config --json
```

## Output

### Text output example

```
Project toolchain
=================

Language:    Go
Framework:   (none detected)
Build:       go build
Test:        go test ./...
Lint:        golangci-lint (detected: .golangci.yml)
Format:      gofmt

Package manager: go modules (go.mod)
Go version:      1.22
```

### JSON output

```json
{
  "language": "Go",
  "framework": "",
  "build": "go build",
  "test": "go test ./...",
  "lint": "golangci-lint",
  "format": "gofmt",
  "packageManager": "go modules",
  "configFiles": [".golangci.yml", "go.mod"],
  "scripts": {}
}
```

For JS/TS projects with package.json scripts:

```json
{
  "language": "TypeScript",
  "framework": "Next.js",
  "build": "npm run build",
  "test": "npm test",
  "lint": "eslint",
  "format": "prettier",
  "packageManager": "npm",
  "configFiles": ["package.json", "tsconfig.json", ".eslintrc.json", ".prettierrc"],
  "scripts": {
    "build": "next build",
    "dev": "next dev",
    "start": "next start",
    "test": "jest",
    "lint": "eslint ."
  }
}
```

## Implementation

### 1. New file: `index/config.go`

Add a `ConfigResult` struct and a `Config()` method on `*Index`.

**Detection strategy** — inspect indexed files and read key config files:

#### Language detection
- Already have extension counts from the index. Pick primary language by file count (reuse `languageMap` from summary.go).

#### Framework detection
Detect by checking manifest contents and config file presence:

| Signal | Framework |
|---|---|
| `package.json` has `next` dep | Next.js |
| `package.json` has `react` dep (no next) | React |
| `package.json` has `vue` dep | Vue |
| `package.json` has `@angular/core` dep | Angular |
| `package.json` has `express` dep | Express |
| `package.json` has `fastify` dep | Fastify |
| `requirements.txt` or `pyproject.toml` has `django` | Django |
| `requirements.txt` or `pyproject.toml` has `flask` | Flask |
| `requirements.txt` or `pyproject.toml` has `fastapi` | FastAPI |
| `Cargo.toml` has `actix-web` | Actix |
| `Cargo.toml` has `axum` | Axum |
| `go.mod` has `gin-gonic/gin` | Gin |
| `go.mod` has `labstack/echo` | Echo |
| `go.mod` has `gorilla/mux` | Gorilla Mux |

#### Build/test/lint/format detection
Check for config files in the index:

| Config file | Detects |
|---|---|
| `Makefile` | Build: `make` |
| `Dockerfile` | Build: `docker build` |
| `tsconfig.json` | Build: `tsc` |
| `.eslintrc*`, `eslint.config.*` | Lint: `eslint` |
| `.prettierrc*`, `prettier.config.*` | Format: `prettier` |
| `.golangci.yml` | Lint: `golangci-lint` |
| `jest.config.*`, package.json "jest" key | Test: `jest` |
| `vitest.config.*` | Test: `vitest` |
| `pytest.ini`, `pyproject.toml [tool.pytest]` | Test: `pytest` |
| `.github/workflows/*.yml` | CI: GitHub Actions |

#### Package.json scripts extraction
If `package.json` exists, parse it and extract the `scripts` field verbatim — this is the single most useful piece of info for a JS/TS agent.

### 2. Wire up CLI in `main.go`

Add `case "config":` that calls `idx.Config()` with text/JSON output.

### 3. Add `FormatConfig()` in `index/config.go`

Human-readable text formatter.

### 4. Tests: `index/config_test.go`

- Test framework detection from mock package.json contents.
- Test lint/format/test tool detection from config file presence.
- Test primary language detection.
- Test scripts extraction from package.json.

## Dependencies

- Requires a prior `scan` (reads from the persisted index).
- Reads select config files from disk for content inspection (package.json, go.mod, etc.) — same pattern as `deps` command.

## Files to modify

- `index/config.go` (new)
- `index/config_test.go` (new)
- `main.go` (add case)
- `README.md` (add to commands table and examples)
- `SKILL.md` (add usage example)

## Completion notes

Implemented by agent c2900d2a. All files listed above were created/modified. 22 tests added covering:
- Language detection (Go, Python, TypeScript with extension aggregation)
- Framework detection (Next.js, React, Django, Flask, Gin, Axum)
- Tool detection (eslint, prettier, golangci-lint, jest, vitest, pytest, pytest from pyproject.toml, jest from package.json)
- Go/Rust language defaults (build, test, format)
- Package manager detection (npm, yarn, pnpm)
- Scripts extraction from package.json
- Full integration test (Next.js project with all tools)
- FormatConfig text output tests

All tests pass (`go test ./...`).
