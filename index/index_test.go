package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
			{Name: "util.go", Kind: "file", Path: "lib/util.go", Package: "lib"},
			{Name: "helper.go", Kind: "file", Path: "lib/helper.go", Package: "lib"},
		},
	}

	tests := []struct {
		query string
		want  int
	}{
		{"main", 1},
		{"auth", 1},
		{".go", 6},
		{"handler", 1},
		{"nonexistent", 0},
		// Path-based matching
		{"api/handler", 1},
		{"api/", 2},
		{"lib/", 2},
		{"lib/util", 1},
		// No duplicates when both name and path match
		{"handler.go", 1},
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

func TestScanSkipsSwarmDir(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "swarm/index/index.json", `[]`)
	mkFile(t, tmp, "swarm/index/meta.json", `{}`)
	mkFile(t, tmp, "swarm/todo/task.md", "# Task")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if got := idx.FileCount(); got != 1 {
		t.Errorf("FileCount() = %d, want 1 (only main.go)", got)
	}

	for _, e := range idx.Entries {
		if strings.Contains(e.Path, "swarm") {
			t.Errorf("index contains swarm entry: %s", e.Path)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	// Create a temp directory with files, scan it, save, load, and verify round-trip.
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "lib/util.go", "package lib")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(loaded.Entries) != len(idx.Entries) {
		t.Fatalf("loaded %d entries, want %d", len(loaded.Entries), len(idx.Entries))
	}

	for i, e := range loaded.Entries {
		orig := idx.Entries[i]
		if e.Name != orig.Name || e.Kind != orig.Kind || e.Path != orig.Path || e.Package != orig.Package {
			t.Errorf("entry %d mismatch: got %+v, want %+v", i, e, orig)
		}
	}

	if loaded.Root != idx.Root {
		t.Errorf("Root = %q, want %q", loaded.Root, idx.Root)
	}
}

func TestSaveMeta(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", "package a")
	mkFile(t, tmp, "b/c.go", "package b")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	before := time.Now().UTC()
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "swarm", "index", "meta.json"))
	if err != nil {
		t.Fatalf("reading meta.json: %v", err)
	}

	var meta indexMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parsing meta.json: %v", err)
	}

	if meta.Version != "0.1.0" {
		t.Errorf("version = %q, want %q", meta.Version, "0.1.0")
	}
	if meta.FileCount != 2 {
		t.Errorf("fileCount = %d, want 2", meta.FileCount)
	}
	if meta.PackageCount != 2 {
		t.Errorf("packageCount = %d, want 2", meta.PackageCount)
	}
	if meta.Root != idx.Root {
		t.Errorf("root = %q, want %q", meta.Root, idx.Root)
	}

	ts, err := time.Parse(time.RFC3339, meta.ScannedAt)
	if err != nil {
		t.Fatalf("scannedAt not RFC3339: %v", err)
	}
	if ts.Before(before.Add(-time.Second)) {
		t.Errorf("scannedAt %v is before test start %v", ts, before)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	idx := &Index{Root: tmp, Entries: []Entry{{Name: "test.go", Kind: "file", Path: "test.go", Package: "(root)"}}}

	// swarm/index/ does not exist yet
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "swarm", "index", "index.json")); err != nil {
		t.Errorf("index.json not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "swarm", "index", "meta.json")); err != nil {
		t.Errorf("meta.json not created: %v", err)
	}
}

func TestLoadNonexistent(t *testing.T) {
	tmp := t.TempDir()
	_, err := Load(tmp)
	if err == nil {
		t.Fatal("Load() should return error for nonexistent index")
	}
}

func TestLookupIntegration(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "handler.go", "package main")
	mkFile(t, tmp, "auth.go", "package main")
	mkFile(t, tmp, "utils.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	results := loaded.Match("auth")
	if len(results) != 1 {
		t.Errorf("Match('auth') = %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Name != "auth.go" {
		t.Errorf("Match('auth')[0].Name = %q, want %q", results[0].Name, "auth.go")
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
