# Python Heuristic Parser

## Summary

Add a regex-based Python parser that extracts top-level symbols (functions, classes, methods, imports, module-level variables/constants) from `.py` files. This enables the `outline` command to work on Python projects, matching the existing Go AST parser's feature set.

Corresponds to PLAN.md Phase 5, Task 8.

## Motivation

The `outline` command currently only supports Go files. Python is one of the most common languages encountered by coding agents. Adding Python support makes `swarm-index outline` useful for a much broader set of projects, with zero new dependencies (regex-based, no tree-sitter needed).

## Implementation

### File: `parsers/pyparser.go`

Create a new parser implementing the `Parser` interface:

```go
type PythonParser struct{}

func (p *PythonParser) Extensions() []string {
    return []string{".py"}
}

func (p *PythonParser) Parse(filePath string, content []byte) ([]Symbol, error) {
    // Line-by-line regex matching
}
```

Register via `init()` — same pattern as `goparser.go`.

### Symbols to extract

| Pattern | Kind | Exported | Notes |
|---|---|---|---|
| `def function_name(...)` at indent level 0 | `func` | `!strings.HasPrefix(name, "_")` | Top-level functions |
| `async def function_name(...)` at indent level 0 | `func` | same | Async top-level functions |
| `class ClassName(...)` at indent level 0 | `class` | same | Classes |
| `def method_name(self, ...)` indented inside a class | `method` | same | Methods — set `Parent` to enclosing class name |
| `async def method_name(self, ...)` indented inside a class | `method` | same | Async methods |
| `NAME = ...` at indent level 0 (UPPER_SNAKE_CASE) | `const` | true | Module-level constants |
| `NAME: type = ...` at indent level 0 (UPPER_SNAKE_CASE) | `const` | true | Typed module-level constants |

### Key design decisions

1. **Indentation-based scoping**: Python's indentation is meaningful. Track current class context by detecting `class` lines and resetting when indentation returns to level 0. A `def` inside a class (indented) is a `method`; at level 0 it's a `func`.

2. **Export convention**: In Python, names starting with `_` are conventionally private. Set `Exported = true` for names not starting with `_`.

3. **Signature**: Include the full `def name(params):` or `class Name(bases):` line (trimmed), similar to how the Go parser includes full function signatures.

4. **EndLine**: For simplicity, set `EndLine` to the last consecutive line before the next top-level definition or end-of-file. This doesn't need to be exact — it's a heuristic.

5. **Decorators**: If a `@decorator` line appears immediately before a `def` or `class`, include it in the signature but keep `Line` pointing to the `def`/`class` line itself.

### Regex patterns

```
^def\s+(\w+)\s*\(       — top-level function
^async\s+def\s+(\w+)\s*\(  — top-level async function
^class\s+(\w+)           — class definition
^\s+def\s+(\w+)\s*\(     — method (indented def)
^\s+async\s+def\s+(\w+)\s*\(  — async method
^([A-Z][A-Z0-9_]*)\s*[=:] — module-level constant
```

### File: `parsers/pyparser_test.go`

Test cases:

1. **Basic function**: `def hello():` → kind=func, name=hello, exported=true
2. **Private function**: `def _helper():` → kind=func, name=_helper, exported=false
3. **Async function**: `async def fetch():` → kind=func, name=fetch
4. **Class with methods**:
   ```python
   class MyClass:
       def __init__(self):
           pass
       def method(self):
           pass
   ```
   → class MyClass + method __init__ (parent=MyClass) + method method (parent=MyClass)
5. **Constants**: `MAX_RETRIES = 3` → kind=const, name=MAX_RETRIES
6. **Decorated function**: `@app.route("/")\ndef index():` → kind=func, name=index
7. **Mixed file**: Combination of all the above, verify ordering and correct parent assignment
8. **Empty file**: Returns empty symbol list, no error
9. **Nested classes**: Only extract top-level class and its direct methods (ignore deeper nesting for the heuristic parser)

### Wire-up

No changes needed to `main.go` — the `outline` command already uses `parsers.ForExtension(ext)`, so registering `.py` in the parser registry is sufficient. The `outline` command will automatically work for Python files once the parser is registered.

## Testing

```bash
go test ./parsers/ -run TestPython -v
```

## Acceptance criteria

- `swarm-index outline some_file.py` outputs Python symbols with correct kinds, names, line numbers, and signatures
- `swarm-index outline some_file.py --json` outputs valid JSON array of Symbol objects
- All test cases pass
- No new dependencies added (stdlib + regex only)

## Completion Notes

Completed by agent ab6360cb.

### Files created
- `parsers/pyparser.go` — Python heuristic parser using regex-based line-by-line parsing
- `parsers/pyparser_test.go` — 9 test functions covering all acceptance criteria

### Files updated
- `README.md` — Updated outline command description, project structure, and roadmap
- `SKILL.md` — Updated outline example to show Python support

### What was implemented
- `PythonParser` struct implementing `Parser` interface, registered via `init()`
- Extracts: top-level functions (incl. async), classes, methods (incl. async), module-level UPPER_SNAKE_CASE constants
- Indentation-based class scoping: methods inside a class get `Parent` set to the class name
- Export convention: names starting with `_` are marked `Exported=false`
- Decorator support: `@decorator` lines are included in the signature
- `EndLine` computed by finding the next line at same/lesser indent or EOF
- All 87 tests pass (9 new Python tests + 78 existing tests)
- End-to-end verified: `swarm-index outline sample.py` and `swarm-index outline sample.py --json` both work correctly
