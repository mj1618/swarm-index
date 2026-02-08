# Refactoring Review

## Changes Reviewed
- Unstaged: README.md, SKILL.md (documentation updates for JS/TS parser support)
- Untracked: parsers/jsparser.go, parsers/jsparser_test.go (new JS/TS heuristic parser)

## Refactoring Applied
- Removed unused `jsBlockCommentStartRe` and `jsBlockCommentEndRe` compiled regex variables from `parsers/jsparser.go`. These were declared but never referenced; the block comment handling uses `strings.Contains`/`strings.Index` directly.

## No Other Issues Found
- Code follows existing patterns from Go and Python parsers
- No cross-parser duplication
- Clear naming, good test coverage
- All tests pass after refactoring
