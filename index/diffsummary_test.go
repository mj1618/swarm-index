package index

import (
	"strings"
	"testing"
)

func TestParseDiffLineAdded(t *testing.T) {
	info := parseDiffLine("A\tpath/to/new.go")
	if info == nil {
		t.Fatal("parseDiffLine returned nil for added file")
	}
	if info.status != "A" {
		t.Errorf("status = %q, want %q", info.status, "A")
	}
	if info.path != "path/to/new.go" {
		t.Errorf("path = %q, want %q", info.path, "path/to/new.go")
	}
}

func TestParseDiffLineModified(t *testing.T) {
	info := parseDiffLine("M\tindex/index.go")
	if info == nil {
		t.Fatal("parseDiffLine returned nil for modified file")
	}
	if info.status != "M" {
		t.Errorf("status = %q, want %q", info.status, "M")
	}
	if info.path != "index/index.go" {
		t.Errorf("path = %q, want %q", info.path, "index/index.go")
	}
}

func TestParseDiffLineDeleted(t *testing.T) {
	info := parseDiffLine("D\told/deprecated.go")
	if info == nil {
		t.Fatal("parseDiffLine returned nil for deleted file")
	}
	if info.status != "D" {
		t.Errorf("status = %q, want %q", info.status, "D")
	}
	if info.path != "old/deprecated.go" {
		t.Errorf("path = %q, want %q", info.path, "old/deprecated.go")
	}
}

func TestParseDiffLineRenamed(t *testing.T) {
	info := parseDiffLine("R100\told/name.go\tnew/name.go")
	if info == nil {
		t.Fatal("parseDiffLine returned nil for renamed file")
	}
	if info.status != "R" {
		t.Errorf("status = %q, want %q", info.status, "R")
	}
	if info.oldPath != "old/name.go" {
		t.Errorf("oldPath = %q, want %q", info.oldPath, "old/name.go")
	}
	if info.path != "new/name.go" {
		t.Errorf("path = %q, want %q", info.path, "new/name.go")
	}
}

func TestParseDiffLineCopy(t *testing.T) {
	info := parseDiffLine("C100\tsource.go\tcopy.go")
	if info == nil {
		t.Fatal("parseDiffLine returned nil for copied file")
	}
	if info.status != "A" {
		t.Errorf("status = %q, want %q (copy treated as add)", info.status, "A")
	}
	if info.path != "copy.go" {
		t.Errorf("path = %q, want %q", info.path, "copy.go")
	}
}

func TestParseDiffLineEmpty(t *testing.T) {
	info := parseDiffLine("")
	if info != nil {
		t.Errorf("parseDiffLine(%q) = %+v, want nil", "", info)
	}
}

func TestParseDiffLineUnknownStatus(t *testing.T) {
	info := parseDiffLine("X\tsome/file.go")
	if info != nil {
		t.Errorf("parseDiffLine returned non-nil for unknown status X")
	}
}

func TestShouldSkipDiffPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"main.go", false},
		{"src/index.go", false},
		{"node_modules/dep/index.js", true},
		{".git/config", true},
		{"vendor/lib/util.go", true},
		{"__pycache__/mod.pyc", true},
		{"swarm/index/index.json", true},
	}
	for _, tt := range tests {
		got := shouldSkipDiffPath(tt.path)
		if got != tt.want {
			t.Errorf("shouldSkipDiffPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestFormatDiffSummaryNoChanges(t *testing.T) {
	result := &DiffSummaryResult{
		Ref:       "HEAD~1",
		Added:     []DiffFile{},
		Modified:  []DiffFile{},
		Deleted:   []DiffFile{},
		FileCount: 0,
	}
	out := FormatDiffSummary(result)
	if !strings.Contains(out, "HEAD~1") {
		t.Errorf("output missing ref: %s", out)
	}
	if !strings.Contains(out, "0 files") {
		t.Errorf("output missing '0 files': %s", out)
	}
	if !strings.Contains(out, "No changes found") {
		t.Errorf("output missing 'No changes found': %s", out)
	}
}

func TestFormatDiffSummaryWithChanges(t *testing.T) {
	result := &DiffSummaryResult{
		Ref: "main",
		Added: []DiffFile{
			{Path: "api/handlers/logout.go", Status: "added", Symbols: []string{"LogoutHandler", "validateSession"}},
		},
		Modified: []DiffFile{
			{Path: "index/index.go", Status: "modified", Symbols: []string{"Scan", "Match", "Save"}},
			{Path: "main.go", Status: "modified"},
		},
		Deleted: []DiffFile{
			{Path: "old/deprecated.go", Status: "deleted"},
		},
		FileCount: 4,
	}
	out := FormatDiffSummary(result)

	checks := []string{
		"Changes since main (4 files)",
		"Added:",
		"+ api/handlers/logout.go",
		"Symbols: LogoutHandler, validateSession",
		"Modified:",
		"~ index/index.go",
		"Symbols: Scan, Match, Save",
		"~ main.go",
		"Deleted:",
		"- old/deprecated.go",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q:\n%s", check, out)
		}
	}
}

func TestFormatDiffSummaryModifiedNoSymbols(t *testing.T) {
	result := &DiffSummaryResult{
		Ref: "HEAD~1",
		Modified: []DiffFile{
			{Path: "README.md", Status: "modified"},
		},
		Added:     []DiffFile{},
		Deleted:   []DiffFile{},
		FileCount: 1,
	}
	out := FormatDiffSummary(result)
	if strings.Contains(out, "Symbols:") {
		t.Errorf("output should not contain 'Symbols:' for file without symbols:\n%s", out)
	}
	if !strings.Contains(out, "~ README.md") {
		t.Errorf("output missing file entry:\n%s", out)
	}
}

func TestExtractSymbolsNonexistentFile(t *testing.T) {
	syms := extractSymbols("/nonexistent/root", "no/such/file.go")
	if syms != nil {
		t.Errorf("extractSymbols for nonexistent file = %v, want nil", syms)
	}
}

func TestExtractSymbolsUnsupportedExtension(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "data.csv", "a,b,c\n1,2,3")
	syms := extractSymbols(tmp, "data.csv")
	if syms != nil {
		t.Errorf("extractSymbols for .csv file = %v, want nil", syms)
	}
}

func TestExtractSymbolsGoFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "example.go", `package example

func HelloWorld() {}

type Config struct {}
`)
	syms := extractSymbols(tmp, "example.go")
	if len(syms) == 0 {
		t.Fatal("extractSymbols returned no symbols for Go file")
	}
	found := map[string]bool{}
	for _, s := range syms {
		found[s] = true
	}
	if !found["HelloWorld"] {
		t.Errorf("missing symbol HelloWorld, got %v", syms)
	}
	if !found["Config"] {
		t.Errorf("missing symbol Config, got %v", syms)
	}
}
