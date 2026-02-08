package index

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildTree(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "lib/util.go", "package lib")
	mkFile(t, tmp, "lib/helper.go", "package lib")

	tree, err := BuildTree(tmp, 0)
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}

	if tree.Type != "dir" {
		t.Errorf("root Type = %q, want %q", tree.Type, "dir")
	}

	// Should have lib/ dir and main.go file
	if len(tree.Children) != 2 {
		t.Fatalf("root has %d children, want 2", len(tree.Children))
	}

	// Directories come first in sort order
	libNode := tree.Children[0]
	if libNode.Name != "lib" || libNode.Type != "dir" {
		t.Errorf("first child = %+v, want lib dir", libNode)
	}
	if len(libNode.Children) != 2 {
		t.Errorf("lib has %d children, want 2", len(libNode.Children))
	}

	mainNode := tree.Children[1]
	if mainNode.Name != "main.go" || mainNode.Type != "file" {
		t.Errorf("second child = %+v, want main.go file", mainNode)
	}
}

func TestBuildTreeDepth(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "top.go", "package main")
	mkFile(t, tmp, "a/b.go", "package a")
	mkFile(t, tmp, "a/deep/c.go", "package deep")

	tree, err := BuildTree(tmp, 1)
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}

	// Depth 1 means we see top-level entries but don't recurse into subdirs
	var aNode *TreeNode
	for _, c := range tree.Children {
		if c.Name == "a" {
			aNode = c
		}
	}
	if aNode == nil {
		t.Fatal("expected 'a' directory in tree")
	}
	if len(aNode.Children) != 0 {
		t.Errorf("depth=1: 'a' has %d children, want 0", len(aNode.Children))
	}
}

func TestBuildTreeSkipsDirs(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, ".git/config", "")
	mkFile(t, tmp, "node_modules/dep/index.js", "")
	mkFile(t, tmp, "swarm/todo/task.md", "")
	mkFile(t, tmp, ".hidden/secret.go", "")

	tree, err := BuildTree(tmp, 0)
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}

	for _, c := range tree.Children {
		switch c.Name {
		case ".git", "node_modules", "swarm", ".hidden":
			t.Errorf("tree contains skipped directory: %s", c.Name)
		}
	}

	if len(tree.Children) != 1 {
		t.Errorf("tree has %d children, want 1 (only main.go)", len(tree.Children))
	}
}

func TestRenderTree(t *testing.T) {
	node := &TreeNode{
		Name: "my-project",
		Type: "dir",
		Children: []*TreeNode{
			{
				Name: "index",
				Type: "dir",
				Children: []*TreeNode{
					{Name: "index.go", Type: "file"},
					{Name: "index_test.go", Type: "file"},
				},
			},
			{Name: "main.go", Type: "file"},
			{Name: "README.md", Type: "file"},
		},
	}

	output := RenderTree(node)

	// Check key structural elements
	if !strings.Contains(output, "my-project/") {
		t.Error("output missing root directory name")
	}
	if !strings.Contains(output, "├── index/") {
		t.Error("output missing index/ directory with connector")
	}
	if !strings.Contains(output, "│   ├── index.go") {
		t.Error("output missing index.go with nested connector")
	}
	if !strings.Contains(output, "│   └── index_test.go") {
		t.Error("output missing index_test.go with last connector")
	}
	if !strings.Contains(output, "├── main.go") {
		t.Error("output missing main.go")
	}
	if !strings.Contains(output, "└── README.md") {
		t.Error("output missing README.md with last connector")
	}
}

func TestBuildTreeJSON(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", "package main")
	mkFile(t, tmp, "sub/b.go", "package sub")

	tree, err := BuildTree(tmp, 0)
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}

	data, err := json.Marshal(tree)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var roundTrip TreeNode
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if roundTrip.Name != tree.Name {
		t.Errorf("Name = %q, want %q", roundTrip.Name, tree.Name)
	}
	if roundTrip.Type != "dir" {
		t.Errorf("Type = %q, want %q", roundTrip.Type, "dir")
	}
	if len(roundTrip.Children) != 2 {
		t.Errorf("Children count = %d, want 2", len(roundTrip.Children))
	}
}

func TestBuildTreeNonexistent(t *testing.T) {
	_, err := BuildTree("/tmp/nonexistent-tree-test-path", 0)
	if err == nil {
		t.Fatal("BuildTree() should return error for nonexistent path")
	}
}

func TestBuildTreeFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "afile.txt", "hello")

	_, err := BuildTree(tmp+"/afile.txt", 0)
	if err == nil {
		t.Fatal("BuildTree() should return error when given a file")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not a directory")
	}
}
