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

func TestScanNonexistentDir(t *testing.T) {
	_, err := Scan("/tmp/nonexistent-path-swarm-index-test")
	if err == nil {
		t.Fatal("Scan() should return error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "cannot access") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "cannot access")
	}
}

func TestScanFileNotDir(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "afile.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Scan(f)
	if err == nil {
		t.Fatal("Scan() should return error when given a file")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not a directory")
	}
}

func TestExtensionCounts(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "lib/util.go", "package lib")
	mkFile(t, tmp, "lib/helper.go", "package lib")
	mkFile(t, tmp, "README.md", "# README")
	mkFile(t, tmp, "config.json", "{}")
	mkFile(t, tmp, "Makefile", "all:")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	counts := idx.ExtensionCounts()

	want := map[string]int{
		".go":    3,
		".md":    1,
		".json":  1,
		"(none)": 1,
	}

	if len(counts) != len(want) {
		t.Errorf("ExtensionCounts() has %d entries, want %d: %v", len(counts), len(want), counts)
	}
	for ext, wantN := range want {
		if counts[ext] != wantN {
			t.Errorf("ExtensionCounts()[%q] = %d, want %d", ext, counts[ext], wantN)
		}
	}
}

func TestSaveMetaExtensions(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", "package a")
	mkFile(t, tmp, "b.go", "package a")
	mkFile(t, tmp, "README.md", "# hi")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
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

	if meta.Extensions == nil {
		t.Fatal("meta.Extensions is nil")
	}
	if meta.Extensions[".go"] != 2 {
		t.Errorf("meta.Extensions[.go] = %d, want 2", meta.Extensions[".go"])
	}
	if meta.Extensions[".md"] != 1 {
		t.Errorf("meta.Extensions[.md] = %d, want 1", meta.Extensions[".md"])
	}
}

func TestExtensionCountsRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "doc.md", "# doc")
	mkFile(t, tmp, "data.json", "{}")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Read meta.json directly to verify extensions survived the round-trip.
	data, err := os.ReadFile(filepath.Join(tmp, "swarm", "index", "meta.json"))
	if err != nil {
		t.Fatalf("reading meta.json: %v", err)
	}

	var meta indexMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parsing meta.json: %v", err)
	}

	want := map[string]int{".go": 1, ".md": 1, ".json": 1}
	for ext, wantN := range want {
		if meta.Extensions[ext] != wantN {
			t.Errorf("round-trip meta.Extensions[%q] = %d, want %d", ext, meta.Extensions[ext], wantN)
		}
	}
}

func TestLoadIgnorePatterns(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, ".swarmignore", "# comment\n\nproto_out/\n*.generated.go\nsecrets.json\n")

	patterns := loadIgnorePatterns(tmp)
	if len(patterns) != 3 {
		t.Fatalf("loadIgnorePatterns() returned %d patterns, want 3: %v", len(patterns), patterns)
	}

	want := []string{"proto_out/", "*.generated.go", "secrets.json"}
	for i, p := range patterns {
		if p != want[i] {
			t.Errorf("pattern[%d] = %q, want %q", i, p, want[i])
		}
	}
}

func TestLoadIgnorePatternsNoFile(t *testing.T) {
	tmp := t.TempDir()
	patterns := loadIgnorePatterns(tmp)
	if patterns != nil {
		t.Errorf("loadIgnorePatterns() = %v, want nil when no .swarmignore exists", patterns)
	}
}

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		relPath  string
		isDir    bool
		patterns []string
		want     bool
	}{
		// Directory pattern with trailing /
		{"proto_out", true, []string{"proto_out/"}, true},
		{"proto_out", false, []string{"proto_out/"}, false}, // trailing / only matches dirs
		{"src/proto_out", true, []string{"proto_out/"}, true},

		// Glob pattern matching basename
		{"foo.generated.go", false, []string{"*.generated.go"}, true},
		{"src/bar.generated.go", false, []string{"*.generated.go"}, true},
		{"foo.go", false, []string{"*.generated.go"}, false},

		// Exact basename match
		{"secrets.json", false, []string{"secrets.json"}, true},
		{"src/secrets.json", false, []string{"secrets.json"}, true},
		{"secrets.txt", false, []string{"secrets.json"}, false},

		// Rooted pattern
		{"gen", true, []string{"/gen"}, true},
		{"src/gen", true, []string{"/gen"}, false}, // rooted: only matches at root

		// No patterns
		{"anything", false, nil, false},
		{"anything", false, []string{}, false},

		// Glob wildcard on dirs
		{"test_data", true, []string{"test_*/"}, true},
		{"src/test_data", true, []string{"test_*/"}, true},
	}

	for _, tt := range tests {
		got := shouldIgnore(tt.relPath, tt.isDir, tt.patterns)
		if got != tt.want {
			t.Errorf("shouldIgnore(%q, isDir=%v, %v) = %v, want %v",
				tt.relPath, tt.isDir, tt.patterns, got, tt.want)
		}
	}
}

func TestScanRespectsSwarmignore(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, ".swarmignore", "generated/\n*.min.js\n")
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "generated/output.go", "package gen")
	mkFile(t, tmp, "lib/app.min.js", "minified")
	mkFile(t, tmp, "lib/app.js", "normal")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if got := idx.FileCount(); got != 3 {
		// .swarmignore + main.go + lib/app.js
		t.Errorf("FileCount() = %d, want 3", got)
	}

	for _, e := range idx.Entries {
		if strings.Contains(e.Path, "generated") {
			t.Errorf("index contains ignored directory entry: %s", e.Path)
		}
		if strings.HasSuffix(e.Path, ".min.js") {
			t.Errorf("index contains ignored file entry: %s", e.Path)
		}
	}
}

func TestScanNoSwarmignore(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "lib/util.go", "package lib")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if got := idx.FileCount(); got != 2 {
		t.Errorf("FileCount() = %d, want 2", got)
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
