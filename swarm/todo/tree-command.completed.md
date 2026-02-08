# `tree` Command

## Summary

Add a `tree` command that prints the directory structure of a project, respecting the same skip rules as `scan`. This gives agents instant visual orientation in a codebase without needing to run external tools.

## Usage

```bash
swarm-index tree <directory>
swarm-index tree . --depth 3
swarm-index tree . --json
```

## Implementation

### 1. Add `Tree` function to `index/index.go`

Define a `TreeNode` struct:

```go
type TreeNode struct {
    Name     string      `json:"name"`
    Type     string      `json:"type"`     // "file" or "dir"
    Children []*TreeNode `json:"children,omitempty"`
}
```

Add a `BuildTree(root string, maxDepth int) (*TreeNode, error)` function that:
- Walks the directory using `filepath.Walk` or `os.ReadDir` (prefer `os.ReadDir` for sorting control)
- Reuses `shouldSkipDir` to skip the same noise directories as `scan`
- Respects `maxDepth` (0 means unlimited)
- Returns a nested `TreeNode` representing the directory structure

### 2. Add text rendering

Add a `RenderTree(node *TreeNode, indent string, isLast bool) string` function (or method) that produces classic tree output with `├──` / `└──` connectors. Example output:

```
my-project/
├── main.go
├── index/
│   ├── index.go
│   └── index_test.go
├── go.mod
└── README.md
```

### 3. Wire up CLI in `main.go`

Add a `case "tree"` to the command switch:
- Parse `<directory>` argument (required)
- Parse `--depth N` flag (default 0 = unlimited)
- Call `index.BuildTree(dir, depth)`
- If `--json`: marshal the `TreeNode` and print
- Otherwise: print the text rendering

Extract `--depth` the same way `--max` is extracted (simple arg scanning).

### 4. Add tests in `index/index_test.go`

- **TestBuildTree**: Create a temp dir structure, build the tree, verify node names and nesting
- **TestBuildTreeDepth**: Verify `--depth 1` limits to top-level only
- **TestBuildTreeSkipsDirs**: Verify `.git`, `node_modules`, `swarm`, etc. are skipped
- **TestRenderTree**: Verify text output format matches expected connector characters

### 5. Update README.md and SKILL.md

- Add `tree` to the Commands table in README
- Move `tree` from "Planned commands" to the Commands section
- Add a `tree` example to SKILL.md

## Flags

| Flag | Default | Description |
|---|---|---|
| `--depth N` | 0 (unlimited) | Maximum directory depth to display |
| `--json` | false | Output as nested JSON structure |

## Acceptance criteria

- `swarm-index tree .` prints a human-readable directory tree
- `swarm-index tree . --depth 2` limits depth
- `swarm-index tree . --json` outputs valid nested JSON
- Skipped directories (`.git`, `node_modules`, `swarm`, etc.) are excluded
- All existing tests continue to pass
- README.md and SKILL.md are updated

## Completion notes

Implemented by agent 62ddecb1. All acceptance criteria met:

- Created `index/tree.go` with `TreeNode` struct, `BuildTree()` (uses `os.ReadDir` for sorted output, reuses `shouldSkipDir`), and `RenderTree()` with classic `├──`/`└──` connectors.
- Wired up `case "tree"` in `main.go` with `--depth N` flag parsing and `--json` support.
- Added 7 tests in `index/tree_test.go`: TestBuildTree, TestBuildTreeDepth, TestBuildTreeSkipsDirs, TestRenderTree, TestBuildTreeJSON, TestBuildTreeNonexistent, TestBuildTreeFile.
- Updated README.md: added `tree` to Commands table, removed from Planned commands, added Quick start example, updated project structure.
- Updated SKILL.md: added `tree` example.
- All 36 tests pass.
