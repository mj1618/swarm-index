package index

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLanguageMap(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".go", "Go"},
		{".js", "JavaScript"},
		{".jsx", "JavaScript"},
		{".ts", "TypeScript"},
		{".tsx", "TypeScript"},
		{".py", "Python"},
		{".rs", "Rust"},
		{".java", "Java"},
		{".md", "Markdown"},
		{".yml", "YAML"},
		{".yaml", "YAML"},
	}
	for _, tt := range tests {
		got, ok := languageMap[tt.ext]
		if !ok {
			t.Errorf("languageMap[%q] missing", tt.ext)
			continue
		}
		if got != tt.want {
			t.Errorf("languageMap[%q] = %q, want %q", tt.ext, got, tt.want)
		}
	}
}

func TestEntryPointDetection(t *testing.T) {
	entries := []Entry{
		{Name: "main.go", Path: "main.go"},
		{Name: "index.ts", Path: "src/index.ts"},
		{Name: "utils.go", Path: "pkg/utils.go"},
		{Name: "main.go", Path: "cmd/main.go"},
		{Name: "app.py", Path: "app.py"},
	}

	want := map[string]bool{
		"main.go":        true,
		"src/index.ts":   true,
		"pkg/utils.go":   false,
		"cmd/main.go":    true,
		"app.py":         true,
	}

	for _, e := range entries {
		got := isEntryPoint(e)
		if got != want[e.Path] {
			t.Errorf("isEntryPoint(%q) = %v, want %v", e.Path, got, want[e.Path])
		}
	}
}

func TestManifestDetection(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"go.mod", true},
		{"package.json", true},
		{"requirements.txt", true},
		{"Cargo.toml", true},
		{"Makefile", true},
		{"random.go", false},
		{"readme.md", false},
	}
	for _, tt := range tests {
		got := manifestNames[tt.name]
		if got != tt.want {
			t.Errorf("manifestNames[%q] = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"one line", 1},
		{"one\n", 1},
		{"one\ntwo\n", 2},
		{"one\ntwo\nthree", 3},
		{"\n", 1},
		{"\n\n\n", 3},
	}
	for _, tt := range tests {
		got := countLines([]byte(tt.input))
		if got != tt.want {
			t.Errorf("countLines(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSummaryLOC(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", "line1\nline2\nline3\n")
	mkFile(t, tmp, "b.go", "one\ntwo\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	s := idx.Summary()
	if s.LOC != 5 {
		t.Errorf("LOC = %d, want 5", s.LOC)
	}
	if s.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", s.FileCount)
	}
}

func TestSummaryLanguages(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n")
	mkFile(t, tmp, "lib.go", "package lib\n")
	mkFile(t, tmp, "README.md", "# Hello\n")
	mkFile(t, tmp, "Makefile", "all:\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	s := idx.Summary()
	if s.Languages["Go"].Files != 2 {
		t.Errorf("Go files = %d, want 2", s.Languages["Go"].Files)
	}
	if s.Languages["Markdown"].Files != 1 {
		t.Errorf("Markdown files = %d, want 1", s.Languages["Markdown"].Files)
	}
	if s.Languages["(other)"].Files != 1 {
		t.Errorf("(other) files = %d, want 1", s.Languages["(other)"].Files)
	}
}

func TestSummaryEntryPoints(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n")
	mkFile(t, tmp, "lib/util.go", "package lib\n")
	mkFile(t, tmp, "cmd/main.go", "package main\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	s := idx.Summary()
	if len(s.EntryPoints) != 2 {
		t.Fatalf("EntryPoints = %v, want 2 entries", s.EntryPoints)
	}
	// Should be sorted
	if s.EntryPoints[0] != "cmd/main.go" {
		t.Errorf("EntryPoints[0] = %q, want %q", s.EntryPoints[0], "cmd/main.go")
	}
	if s.EntryPoints[1] != "main.go" {
		t.Errorf("EntryPoints[1] = %q, want %q", s.EntryPoints[1], "main.go")
	}
}

func TestSummaryManifests(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "go.mod", "module example\n")
	mkFile(t, tmp, "go.sum", "")
	mkFile(t, tmp, "main.go", "package main\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	s := idx.Summary()
	if len(s.Manifests) != 2 {
		t.Fatalf("Manifests = %v, want 2 entries", s.Manifests)
	}
}

func TestSummaryTopDirectories(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n")
	mkFile(t, tmp, "cmd/run.go", "package cmd\n")
	mkFile(t, tmp, "index/index.go", "package index\n")
	mkFile(t, tmp, "internal/util.go", "package internal\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	s := idx.Summary()
	want := []string{"cmd/", "index/", "internal/"}
	if len(s.TopDirectories) != len(want) {
		t.Fatalf("TopDirectories = %v, want %v", s.TopDirectories, want)
	}
	for i, d := range s.TopDirectories {
		if d != want[i] {
			t.Errorf("TopDirectories[%d] = %q, want %q", i, d, want[i])
		}
	}
}

func TestSummaryFromLoadedIndex(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\nfunc main() {}\n")
	mkFile(t, tmp, "go.mod", "module example\n")
	mkFile(t, tmp, "lib/util.go", "package lib\n")

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

	s := loaded.Summary()
	if s.FileCount != 3 {
		t.Errorf("FileCount = %d, want 3", s.FileCount)
	}
	if s.Root != filepath.Clean(tmp) {
		t.Errorf("Root = %q, want %q", s.Root, tmp)
	}
}

func TestFormatSummary(t *testing.T) {
	s := SummaryResult{
		Root:         "/tmp/project",
		FileCount:    10,
		LOC:          250,
		PackageCount: 3,
		Languages: map[string]LanguageStat{
			"Go":       {Files: 7, Percentage: 70.0},
			"Markdown": {Files: 3, Percentage: 30.0},
		},
		EntryPoints:    []string{"main.go"},
		Manifests:      []string{"go.mod"},
		TopDirectories: []string{"cmd/", "index/"},
	}

	output := FormatSummary(s)

	checks := []string{
		"Project summary for /tmp/project",
		"10 files (250 LOC), 3 packages",
		"Languages:",
		"Go",
		"Markdown",
		"Entry points:",
		"main.go",
		"Dependency manifests:",
		"go.mod",
		"Top directories:",
		"cmd/",
		"index/",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("FormatSummary output missing %q\nGot:\n%s", check, output)
		}
	}
}
