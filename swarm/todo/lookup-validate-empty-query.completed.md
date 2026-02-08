# Validate empty query in lookup command

## Problem

Running `swarm-index lookup ""` matches **every entry** in the index instead of returning an error. This happens because `Match("")` does a substring search, and an empty string is a substring of everything.

This is confusing UX — if a user accidentally passes an empty query (or a script does), they get a dump of the entire index with no indication anything went wrong.

## Changes

### 1. Add validation in `main.go` (lookup case, around line 46)

After extracting `query := os.Args[2]`, check for empty/whitespace-only queries:

```go
query := os.Args[2]
if strings.TrimSpace(query) == "" {
    fmt.Fprintln(os.Stderr, "error: query must not be empty")
    os.Exit(1)
}
```

### 2. Add test in `main_test.go`

Add a `TestParseMaxAndResolveRoot`-style unit test (or integration test) that verifies:

- Empty string `""` is rejected
- Whitespace-only string `"  "` is rejected

Since the validation is in `main()` which is hard to unit-test directly, the simplest approach is to add a small helper `validateQuery(q string) error` that can be tested, then call it from the lookup case.

Example helper:

```go
func validateQuery(q string) error {
    if strings.TrimSpace(q) == "" {
        return fmt.Errorf("query must not be empty")
    }
    return nil
}
```

Tests:

```go
func TestValidateQuery(t *testing.T) {
    tests := []struct {
        query   string
        wantErr bool
    }{
        {"hello", false},
        {"", true},
        {"   ", true},
        {"\t", true},
        {"a", false},
    }
    for _, tt := range tests {
        err := validateQuery(tt.query)
        if (err != nil) != tt.wantErr {
            t.Errorf("validateQuery(%q) error = %v, wantErr %v", tt.query, err, tt.wantErr)
        }
    }
}
```

## Files to modify

- `main.go` — add `validateQuery()` helper, call it in the lookup case
- `main_test.go` — add `TestValidateQuery`

## Acceptance criteria

- `swarm-index lookup ""` prints an error and exits non-zero
- `swarm-index lookup "  "` prints an error and exits non-zero
- `swarm-index lookup "main"` continues to work normally
- All existing tests pass

## Completion notes

Implemented as specified. Added `validateQuery()` helper in `main.go` and called it in the lookup case right after extracting the query. Added `TestValidateQuery` table-driven test in `main_test.go`. All tests pass. Manual verification confirms empty `""` and whitespace `"   "` queries are rejected with error and exit code 1.
