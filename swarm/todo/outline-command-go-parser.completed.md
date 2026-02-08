# Go AST Parser & `outline` Command

## Problem

Agents currently have no way to see the structural skeleton of a file — functions, types, interfaces, methods — without reading the full source. For large files this wastes context window and makes it hard to get a quick overview. The PLAN identifies `outline` as "the highest-value planned command."

## Goal

Add an `outline <file>` command that prints the top-level symbols (functions, types, interfaces, methods, constants, variables) of a Go file with their line numbers and signatures. Start with Go only (using stdlib `go/parser` + `go/ast` — zero new dependencies).

## Design

### New `parsers` package

Create `parsers/parsers.go` with the shared interface:

```go
type Symbol struct {
    Name      string `json:"name"`
    Kind      string `json:"kind"`      // "func", "method", "type", "interface", "struct", "const", "var"
    Line      int    `json:"line"`
    EndLine   int    `json:"endLine"`
    Exported  bool   `json:"exported"`
    Signature string `json:"signature"` // e.g. "func HandleAuth(w http.ResponseWriter, r *http.Request) error"
    Parent    string `json:"parent"`    // enclosing type for methods, empty otherwise
}

type Parser interface {
    Parse(filePath string, content []byte) ([]Symbol, error)
    Extensions() []string
}
```

### Go parser: `parsers/goparser.go`

Use `go/parser.ParseFile` and `go/ast.Inspect` to extract:

- **Functions** — `*ast.FuncDecl` without receiver → kind="func"
- **Methods** — `*ast.FuncDecl` with receiver → kind="method", Parent = receiver type name
- **Type declarations** — `*ast.TypeSpec`:
  - `*ast.StructType` → kind="struct"
  - `*ast.InterfaceType` → kind="interface"
  - other → kind="type"
- **Constants** — `*ast.GenDecl` with `token.CONST` → kind="const"
- **Variables** — `*ast.GenDecl` with `token.VAR` → kind="var"

For signatures, reconstruct from AST nodes (func name + params + return types). For types, use the type name.

Exported is determined by `ast.IsExported(name)`.

### CLI: `outline` subcommand in `main.go`

```
swarm-index outline <file> [--json]
```

- Read the file, select the parser by extension.
- If no parser exists for the extension, print an error: "no parser available for .xyz files"
- Text output: indented list with line numbers and signatures.
  ```
  main.go:
    func main()                                          :42
    func extractJSONFlag(args []string) ([]string, bool) :16
    func fatal(useJSON bool, msg string)                 :31
  ```
- JSON output: array of Symbol objects.

### Tests

1. `parsers/goparser_test.go`:
   - Parse a sample Go source string containing funcs, methods, structs, interfaces, consts, vars.
   - Verify correct symbol names, kinds, line numbers, exported flags, parent fields.
   - Verify method receiver is captured in Parent.
   - Test with empty file and file with syntax errors (should return partial results or clear error).

2. `main_test.go` or integration test:
   - `outline` with no file arg → usage error.
   - `outline` on a non-existent file → error.
   - `outline` on an unsupported extension → "no parser" error.

## Implementation Steps

1. Create `parsers/parsers.go` with `Symbol` type and `Parser` interface.
2. Create `parsers/goparser.go` implementing the Go parser.
3. Create `parsers/goparser_test.go` with unit tests.
4. Add `outline` case to `main.go` switch statement.
5. Update `printUsage()` in `main.go` to include the outline command.
6. Update README.md and SKILL.md to document the new command.

## Notes

- Zero new dependencies — Go's stdlib `go/parser` and `go/ast` handle everything.
- Future parsers for JS/TS/Python can implement the same `Parser` interface and be registered by extension.
- This does NOT yet integrate symbols into the scan index (that can be a follow-up).

## Completion Notes

Implemented by agent 6c333b2d (task e8a349f8). All steps completed:

1. Created `parsers/parsers.go` with `Symbol` struct, `Parser` interface, and extension-based registry.
2. Created `parsers/goparser.go` — full Go AST parser extracting funcs, methods, structs, interfaces, type aliases, consts, and vars with accurate signatures (including variadic params, multiple return types, pointer receivers).
3. Created `parsers/goparser_test.go` — 8 tests covering: basic symbol extraction, signature accuracy, line numbers, empty files, syntax errors, extensions, variadic functions, and the registry lookup.
4. Added `outline` case to `main.go` switch with text and JSON output modes, error handling for missing files and unsupported extensions.
5. Updated `printUsage()` to include the outline command.
6. Updated `README.md` (quick start, commands table, project structure, roadmap checkbox) and `SKILL.md`.
7. All 56 tests pass across all packages (`go test ./...`).
