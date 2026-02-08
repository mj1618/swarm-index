# deps command — Parse dependency manifests and list libraries with versions

## Summary

Add a `deps` command that parses dependency manifest files (go.mod, package.json, requirements.txt, Cargo.toml, pyproject.toml) and lists all declared dependencies with their version constraints. This gives agents instant visibility into what libraries are available in a project without manually reading manifest files.

## Motivation

Agents frequently need to know:
- What libraries are available to import/use
- What version of a library is in use (to know which APIs are available)
- Whether a dependency exists before suggesting adding it
- The full dependency list for understanding the project's tech stack

Currently, the `summary` command lists manifest file paths but doesn't parse their contents. An agent must manually read and parse each manifest. The `deps` command eliminates this friction.

## CLI Interface

```
swarm-index deps [--root <dir>] [--json]
```

### Text output example

```
Dependencies for /Users/matt/code/my-project
=============================================

go.mod (Go):
  github.com/gorilla/mux     v1.8.1
  github.com/lib/pq          v1.10.9
  golang.org/x/sync          v0.6.0

package.json (Node.js):
  react                      ^18.2.0
  next                       ^14.0.0
  typescript                 ^5.3.0

  devDependencies:
  @types/react               ^18.2.0
  eslint                     ^8.56.0

requirements.txt (Python):
  flask                      ==2.3.3
  requests                   >=2.31.0
  numpy

4 manifests, 13 dependencies
```

### JSON output

```json
{
  "manifests": [
    {
      "path": "go.mod",
      "type": "go.mod",
      "ecosystem": "Go",
      "dependencies": [
        {"name": "github.com/gorilla/mux", "version": "v1.8.1", "dev": false},
        {"name": "github.com/lib/pq", "version": "v1.10.9", "dev": false}
      ]
    },
    {
      "path": "package.json",
      "type": "package.json",
      "ecosystem": "Node.js",
      "dependencies": [
        {"name": "react", "version": "^18.2.0", "dev": false},
        {"name": "eslint", "version": "^8.56.0", "dev": true}
      ]
    }
  ],
  "totalDependencies": 13,
  "totalManifests": 4
}
```

## Implementation

### 1. New file: `index/deps.go`

Add types and manifest parsers:

```go
type Dependency struct {
    Name    string `json:"name"`
    Version string `json:"version"`
    Dev     bool   `json:"dev"`
}

type ManifestDeps struct {
    Path         string       `json:"path"`
    Type         string       `json:"type"`
    Ecosystem    string       `json:"ecosystem"`
    Dependencies []Dependency `json:"dependencies"`
}

type DepsResult struct {
    Manifests         []ManifestDeps `json:"manifests"`
    TotalDependencies int            `json:"totalDependencies"`
    TotalManifests    int            `json:"totalManifests"`
}
```

Method on `*Index`:

```go
func (idx *Index) Deps() (DepsResult, error)
```

This method:
1. Iterates over `idx.Entries` looking for known manifest filenames
2. Reads each manifest file from disk
3. Dispatches to the appropriate parser based on filename
4. Aggregates results

### 2. Manifest parsers to implement

Each parser is a function: `func parseXxx(content []byte) ([]Dependency, error)`

#### `go.mod`
- Parse `require (...)` blocks and single `require` lines
- Extract module path and version
- All deps are non-dev (Go doesn't distinguish)

#### `package.json`
- JSON unmarshal into a struct with `dependencies` and `devDependencies` maps
- Mark devDependencies with `dev: true`

#### `requirements.txt`
- Line-based parsing
- Handle `package==version`, `package>=version`, `package~=version`, bare `package`
- Skip comments (`#`) and blank lines
- Handle `-r` includes by noting them but not following

#### `Cargo.toml`
- Parse `[dependencies]` and `[dev-dependencies]` sections
- Handle `name = "version"` and `name = { version = "...", ... }` forms
- Use simple line-based parsing (not a full TOML parser to avoid deps)

#### `pyproject.toml`
- Parse `[project.dependencies]` list
- Parse `[project.optional-dependencies]` as dev
- Simple line-based extraction of dependency specifiers

### 3. Wire up in `main.go`

Add `case "deps":` block following the same pattern as other index-based commands.

### 4. Add `FormatDeps` function for text output

Group by manifest, show ecosystem, list deps with aligned version columns.

### 5. Tests: `index/deps_test.go`

- Test each parser individually with sample manifest content
- Test the full `Deps()` method with a temp directory containing multiple manifests
- Test edge cases: empty manifests, comments, inline version specs

### 6. Update README.md, SKILL.md, and usage text

Add `deps` to the command table, examples, and usage help.

## Files to create/modify

- **Create**: `index/deps.go` — types, parsers, Deps method, FormatDeps
- **Create**: `index/deps_test.go` — tests
- **Modify**: `main.go` — add `deps` case and usage line
- **Modify**: `README.md` — add to commands table and roadmap
- **Modify**: `SKILL.md` — add usage example

## Complexity

Medium — each parser is straightforward line-based or JSON parsing. No external dependencies needed. The main effort is covering the format variations in each manifest type.

## Completion Notes

Implemented by agent 22ad821f (task b35404fb).

### Files created:
- `index/deps.go` — Types (Dependency, ManifestDeps, DepsResult), five manifest parsers (go.mod, package.json, requirements.txt, Cargo.toml, pyproject.toml), `Deps()` method on `*Index`, and `FormatDeps()` text formatter.
- `index/deps_test.go` — 15 tests covering each parser individually (including edge cases: empty manifests, comments, inline comments, extras, inline lists), integration tests with `Scan()`, and format output tests.

### Files modified:
- `main.go` — Added `case "deps":` block and usage line.
- `README.md` — Added to commands table, examples, project structure, and marked as completed in roadmap.
- `SKILL.md` — Added usage examples.

### All tests pass: `go test ./... -count=1` succeeds.
