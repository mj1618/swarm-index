# JavaScript/TypeScript Heuristic Parser for Outline Command

## Summary

Add a regex-based heuristic parser for JavaScript (.js, .jsx) and TypeScript (.ts, .tsx) files, enabling the `outline` command to extract top-level symbols from the most widely used language family in web development.

## Priority

High — JS/TS is the most common language family for web projects. Without this parser, `outline` cannot be used on any JS/TS file, leaving a major gap for agents working on frontend, Node.js, or full-stack codebases.

## Depends On

`outline-command-go-parser` (completed) — the parser interface and registry are already in place.

## Details

The parser should detect these top-level constructs using regex heuristics:

### Functions
- `function name(...)` — named function declarations
- `async function name(...)` — async function declarations
- `export function name(...)` / `export default function name(...)` — exported functions
- `export async function name(...)` — exported async functions

### Arrow functions / const declarations
- `const name = (...)  => ...` — arrow function assigned to const
- `const name = async (...)  => ...` — async arrow function
- `export const name = ...` — exported const (arrow functions and other values)
- `let name = ...` / `var name = ...` at top level (lower priority)

### Classes
- `class Name { ... }` — class declarations
- `export class Name ...` / `export default class Name ...` — exported classes
- Methods inside classes (indent-based detection, similar to Python parser approach)

### TypeScript-specific
- `interface Name { ... }` — interface declarations
- `type Name = ...` — type aliases
- `enum Name { ... }` — enum declarations
- `export interface ...` / `export type ...` / `export enum ...`

### Exports
- `export default ...` — default exports
- `export { name1, name2 }` — named re-exports (lower priority, optional)

### Imports (optional, lower priority)
- `import ... from '...'` — can be skipped initially since outline focuses on declarations

## Symbol Kinds

Map to the existing `Symbol.Kind` values:
- `function`, `async function` → kind: `"func"`
- Class methods → kind: `"method"`, with `Parent` set to class name
- `class` → kind: `"class"`
- `interface` → kind: `"interface"`
- `type` → kind: `"type"`
- `enum` → kind: `"enum"`  (add new kind)
- `const`/`let`/`var` at top level → kind: `"const"` or `"var"`

## Exported Detection

- Symbols preceded by `export` keyword → `Exported: true`
- TypeScript: capitalized names without `export` are still not exported (unlike Go)
- `export default` items → `Exported: true`

## Implementation

### File: `parsers/jsparser.go`

Create a new parser implementing the `Parser` interface:

```go
type JSParser struct{}

func (p *JSParser) Extensions() []string {
    return []string{".js", ".jsx", ".ts", ".tsx"}
}
```

Register it in an `init()` function, same pattern as `goparser.go` and `pyparser.go`.

### Approach

Use the same line-by-line regex scanning approach as `pyparser.go`:

1. Track brace depth (`{` / `}`) to determine top-level vs nested scope
2. Only extract symbols at brace depth 0 (top-level) or depth 1 (class members when inside a class)
3. Use regexes for each construct type
4. Handle multi-line signatures by checking if a line ends with `{` or if the next few lines complete the signature

### Key Regexes

```
^(export\s+)?(export\s+default\s+)?(async\s+)?function\s+(\w+)
^(export\s+)?(default\s+)?class\s+(\w+)
^(export\s+)?(interface|type|enum)\s+(\w+)
^(export\s+)?(const|let|var)\s+(\w+)
```

### Brace Tracking

- Increment depth on `{`, decrement on `}`
- Skip braces inside string literals and comments (basic heuristic: ignore lines that are inside block comments or string templates)
- When depth returns to 0 after a class/function, record `EndLine`

### File: `parsers/jsparser_test.go`

Test with representative samples:

1. **Plain JS functions**: `function hello()`, `async function fetchData()`
2. **Arrow functions**: `const handler = () => {`, `const process = async (data) => {`
3. **Classes with methods**: `class UserService { constructor() {} async getUser(id) {} }`
4. **TypeScript interfaces**: `interface Config { port: number; host: string; }`
5. **TypeScript types**: `type Result<T> = Success<T> | Failure`
6. **TypeScript enums**: `enum Status { Active, Inactive }`
7. **Export variations**: `export function`, `export default class`, `export const`, `export interface`
8. **Nested functions should be skipped**: functions inside other functions shouldn't appear at top level
9. **React components**: `export function App()`, `export const App: React.FC = () =>`

## Testing

```bash
go test ./parsers/ -v -run TestJS
```

## README / SKILL.md Updates

- Update the `outline` command description to mention JS/TS support alongside Go and Python
- Update the project structure section if a new file is added
- Update the roadmap to check off JS/TS parser support

## Completion Notes

Implemented by agent 7a49f9cb. Created `parsers/jsparser.go` with:
- Brace-depth tracking with string literal awareness (single, double, template)
- Block comment handling (single-line and multi-line)
- Top-level symbol detection: functions, classes, interfaces, types, enums, const/let/var
- Class method detection at brace depth 1 (constructor, regular methods, async, get/set, public/private/protected/static)
- Export detection via `export` keyword prefix
- Private method detection via `private` modifier or `_`/`#` prefix
- Abstract class support

Tests in `parsers/jsparser_test.go` cover: plain JS functions, arrow functions, classes with methods, TS interfaces/types/enums, export variations, nested function exclusion, React components, signatures, braces in string literals, empty files, comments-only files, and extension registration. All 90 tests pass across all packages.
