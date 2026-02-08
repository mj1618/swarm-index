# Python Parser: Minimum Constant Name Length

## Summary

The Python heuristic parser currently matches single-letter uppercase variables (e.g., `X = 10`) as constants. Consider requiring a minimum length of 2 characters for UPPER_SNAKE_CASE constant detection to reduce false positives.

## Priority

Low — this is a minor quality-of-life improvement, not a bug.

## Details

Single uppercase letters (`X`, `Y`, `N`, `T`, `I`) are commonly used as:
- Loop variables
- Generic type parameters (`T = TypeVar("T")`)
- Mathematical variables

These are rarely intended as module-level constants. Requiring at least 2 characters (`MAX`, `DB`, `OK`, etc.) would reduce false positives while keeping all meaningful constants.

## Implementation

In `parsers/pyparser.go`, update the `isUpperSnakeCase` function to require `len(name) >= 2`, or adjust the regex to `^([A-Z][A-Z0-9_]+)\s*[=:]`.

## Testing

Update `pyparser_test.go` to verify that single-letter uppercase names are not matched as constants.
