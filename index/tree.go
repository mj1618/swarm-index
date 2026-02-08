package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode represents a file or directory in a tree structure.
type TreeNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"` // "file" or "dir"
	Children []*TreeNode `json:"children,omitempty"`
}

// BuildTree walks a directory and returns a nested TreeNode, respecting the
// same skip rules as Scan. maxDepth of 0 means unlimited.
func BuildTree(root string, maxDepth int) (*TreeNode, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	node := &TreeNode{
		Name: filepath.Base(root),
		Type: "dir",
	}

	if err := buildChildren(node, root, 1, maxDepth); err != nil {
		return nil, err
	}

	return node, nil
}

func buildChildren(parent *TreeNode, dirPath string, currentDepth, maxDepth int) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil // skip unreadable directories
	}

	// Sort entries: directories first, then files, alphabetically within each group
	sort.Slice(entries, func(i, j int) bool {
		iDir := entries[i].IsDir()
		jDir := entries[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		name := e.Name()

		if e.IsDir() {
			if shouldSkipDir(name) {
				continue
			}
			child := &TreeNode{Name: name, Type: "dir"}
			if maxDepth == 0 || currentDepth < maxDepth {
				if err := buildChildren(child, filepath.Join(dirPath, name), currentDepth+1, maxDepth); err != nil {
					return err
				}
			}
			parent.Children = append(parent.Children, child)
		} else {
			parent.Children = append(parent.Children, &TreeNode{Name: name, Type: "file"})
		}
	}

	return nil
}

// RenderTree returns a human-readable tree string with connectors.
func RenderTree(node *TreeNode) string {
	var b strings.Builder
	b.WriteString(node.Name + "/\n")
	renderChildren(&b, node.Children, "")
	return b.String()
}

func renderChildren(b *strings.Builder, children []*TreeNode, prefix string) {
	for i, child := range children {
		isLast := i == len(children)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		if child.Type == "dir" {
			b.WriteString(prefix + connector + child.Name + "/\n")
			childPrefix := prefix + "│   "
			if isLast {
				childPrefix = prefix + "    "
			}
			renderChildren(b, child.Children, childPrefix)
		} else {
			b.WriteString(prefix + connector + child.Name + "\n")
		}
	}
}
