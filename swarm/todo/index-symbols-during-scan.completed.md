# Index Symbols During Scan

## Problem

The `lookup` command doesn't work for finding symbols (functions, types, structs, methods, etc.). It only finds files because `Scan()` only adds file entries to the index — it never parses source files to extract symbols.

For example, `swarm-index lookup "handleAuth"` will only match if there's a **file** named `handleAuth` — it won't find a function called `handleAuth` inside `server.go`. This makes `lookup` nearly useless for its primary use case: quickly finding where a symbol is defined.

The `Entry` struct already supports symbols (`Kind`, `Line`, `Exported` fields), and parsers already exist for Go, JavaScript/TypeScript, and Python. They're just never called during scan.

## Goal

Make `Scan()` parse each source file using the existing parser registry and add symbol entries to the index alongside file entries. After this change, `lookup "handleAuth"` will return both the file match (if any) and the symbol definition with its file path and line number.

## Design

### Changes to `index/index.go` — `Scan()` function

After adding each file entry, check if a parser exists for the file's extension. If so, read the file and parse it, then append symbol entries:

```go
// After adding the file entry...
ext := filepath.Ext(name)
p := parsers.ForExtension(ext)
if p != nil {
    content, readErr := os.ReadFile(path)
    if readErr == nil {
        symbols, parseErr := p.Parse(relPath, content)
        if parseErr == nil {
            for _, sym := range symbols {
                idx.Entries = append(idx.Entries, Entry{
                    Name:     sym.Name,
                    Kind:     sym.Kind,
                    Path:     relPath,
                    Line:     sym.Line,
                    Package:  pkg,
                    Exported: sym.Exported,
                })
            }
        }
    }
}
```

Key points:
- Parse errors and read errors are silently skipped (don't fail the whole scan for one bad file)
- The `relPath` is used for both file and symbol entries so lookups show the correct file
- The `Package` field is inherited from the file's directory
- `Exported` is set from the parser's detection (Go: uppercase first letter, JS: `export` keyword, Python: no leading underscore)

### Changes to `index/fuzzy.go` — `scoreName()` function

The scoring function currently only considers file name/path. It needs to also handle symbol entries well. Since symbol entries have `Name` set to the symbol name (e.g., `HandleAuth`) rather than a filename, the existing scoring logic will mostly work — exact/prefix/substring/fuzzy matching on `Name` will naturally apply. However, the "strip extension" logic (`nameNoExt`) should be skipped for non-file entries, or it could accidentally strip parts of symbol names that happen to contain dots.

Update `scoreName` to accept an optional `kind` parameter, or better: update `matchFuzzy` to use `e.Kind` when deciding how to compare. For symbol entries, compare directly against `Name` without stripping extensions.

Alternatively, since `filepath.Ext` returns empty for names without dots (like most symbol names), the current logic is actually safe — `TrimSuffix(name, "")` is a no-op. **No changes needed to fuzzy.go** as long as symbol names don't contain dots.

### Changes to `index/index.go` — `FileCount()` method

Currently `FileCount()` uses `FilePaths()` which deduplicates by path. Since symbol entries share the same path as their file entry, `FilePaths()` will still return correct counts. **No changes needed.**

### Changes to `index/index.go` — `ExtensionCounts()` method

Same as `FileCount()` — it already deduplicates by path. **No changes needed.**

### Test Updates

Update or add tests in `index/index_test.go`:
1. **`TestScanIncludesSymbols`** — Scan a test fixture directory containing a `.go` file with known functions/types. Verify the index contains both `kind:"file"` and `kind:"func"`/`kind:"type"` entries.
2. **`TestLookupFindsSymbols`** — After scanning, use `Match("symbolName")` and verify the symbol entry is returned with correct `Line`, `Kind`, and `Exported` fields.
3. **`TestLookupRanksSymbolsAndFiles`** — Search for a name that matches both a file and a symbol. Verify both are returned and scoring is sensible.

Add a test fixture file, e.g. `testdata/sample.go`:
```go
package sample

func ExportedFunc() {}
func unexportedFunc() {}
type Config struct{}
```

### CLI Output

The `Entry.String()` method already handles symbol entries well:
- Files: `[file] main.go — main.go (root)`
- Symbols: `[func] HandleAuth — server.go:42 (handlers)`

No CLI output changes needed.

## Scope

- Only modify `index/index.go` (the `Scan` function)
- Add a `parsers` import to `index/index.go`
- Add test fixtures and tests
- No new dependencies — parsers already exist and are registered via init()
- Backward compatible — index.json gains more entries but the schema is unchanged
- The `meta.json` file count will still be correct since `FileCount()` deduplicates
- Other commands (`symbols`, `refs`, `locate`) that dynamically parse files will continue to work; they'll just have more overlap with the index now

## Files to Change

| File | Change |
|---|---|
| `index/index.go` | Add symbol extraction to `Scan()` after each file entry |
| `index/index_test.go` | Add tests for symbol indexing and lookup |
| `testdata/sample.go` | New test fixture with known symbols (if not already present) |
| `README.md` | Update to mention that lookup finds both files and symbols |
| `.cursor/skills/swarm-index/SKILL.md` | Update lookup description |

## Dependencies

None — all prerequisite work is complete:
- Parsers: `go`, `js/ts`, `python` parsers exist and are registered
- Fuzzy matching: already implemented
- Entry struct: already supports symbol fields
