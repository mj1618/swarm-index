package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	// Create a temp directory structure
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "lib/util.go", "package lib")
	mkFile(t, tmp, "lib/helper.go", "package lib")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if got := idx.FileCount(); got != 3 {
		t.Errorf("FileCount() = %d, want 3", got)
	}

	if got := idx.PackageCount(); got != 2 {
		t.Errorf("PackageCount() = %d, want 2 (root + lib)", got)
	}
}

func TestMatch(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go", Package: "(root)"},
			{Name: "handler.go", Kind: "file", Path: "api/handler.go", Package: "api"},
			{Name: "auth.go", Kind: "file", Path: "api/auth.go", Package: "api"},
			{Name: "utils.go", Kind: "file", Path: "pkg/utils.go", Package: "pkg"},
		},
	}

	tests := []struct {
		query string
		want  int
	}{
		{"main", 1},
		{"auth", 1},
		{".go", 4},
		{"handler", 1},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		results := idx.Match(tt.query)
		if len(results) != tt.want {
			t.Errorf("Match(%q) returned %d results, want %d", tt.query, len(results), tt.want)
		}
	}
}

func TestScanSkipsHiddenDirs(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "visible.go", "package main")
	mkFile(t, tmp, ".hidden/secret.go", "package hidden")
	mkFile(t, tmp, "node_modules/dep/index.js", "module.exports = {}")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if got := idx.FileCount(); got != 1 {
		t.Errorf("FileCount() = %d, want 1 (only visible.go)", got)
	}
}

func mkFile(t *testing.T, base, relPath, content string) {
	t.Helper()
	full := filepath.Join(base, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
