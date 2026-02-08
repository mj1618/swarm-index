package index

import (
	"strings"
	"testing"

	_ "github.com/matt/swarm-index/parsers" // register parsers
)

func TestScopeBasicDirectory(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pkg/a.go", `package pkg

func FuncA() {}
func internalA() {}
`)
	mkFile(t, tmp, "pkg/b.go", `package pkg

func FuncB() {}
type TypeB struct{}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Scope("pkg", false)
	if err != nil {
		t.Fatalf("Scope() error: %v", err)
	}

	if result.Directory != "pkg" {
		t.Errorf("Directory = %q, want %q", result.Directory, "pkg")
	}
	if result.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", result.FileCount)
	}
	if result.LOC == 0 {
		t.Error("LOC should be > 0")
	}

	// Check symbol counts.
	funcCount, ok := result.Symbols["func"]
	if !ok {
		t.Fatal("missing 'func' in symbol counts")
	}
	// FuncA, FuncB are exported; internalA is internal
	if funcCount.Exported != 2 {
		t.Errorf("func exported = %d, want 2", funcCount.Exported)
	}
	if funcCount.Internal != 1 {
		t.Errorf("func internal = %d, want 1", funcCount.Internal)
	}
}

func TestScopeNonRecursiveExcludesSubdirs(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pkg/a.go", `package pkg
func FuncA() {}
`)
	mkFile(t, tmp, "pkg/sub/b.go", `package sub
func FuncB() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Scope("pkg", false)
	if err != nil {
		t.Fatalf("Scope() error: %v", err)
	}

	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1 (non-recursive should exclude subdirs)", result.FileCount)
	}
}

func TestScopeRecursiveIncludesSubdirs(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pkg/a.go", `package pkg
func FuncA() {}
`)
	mkFile(t, tmp, "pkg/sub/b.go", `package sub
func FuncB() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Scope("pkg", true)
	if err != nil {
		t.Fatalf("Scope() error: %v", err)
	}

	if result.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2 (recursive should include subdirs)", result.FileCount)
	}
}

func TestScopeEmptyDirectory(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "other/a.go", `package other
func FuncA() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Scope("nonexistent", false)
	if err != nil {
		t.Fatalf("Scope() error: %v", err)
	}

	if result.FileCount != 0 {
		t.Errorf("FileCount = %d, want 0", result.FileCount)
	}
	if len(result.Files) != 0 {
		t.Errorf("Files = %v, want empty", result.Files)
	}
}

func TestScopeDependencies(t *testing.T) {
	tmp := t.TempDir()
	// pkg/a.go imports from utils/
	mkFile(t, tmp, "pkg/a.go", `package pkg

import "./utils"

func FuncA() {}
`)
	mkFile(t, tmp, "utils/helper.go", `package utils

func Helper() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Scope("pkg", false)
	if err != nil {
		t.Fatalf("Scope() error: %v", err)
	}

	// Dependencies and dependents may be empty depending on import resolution,
	// but the command should not error.
	if result.Dependencies == nil {
		t.Error("Dependencies should not be nil")
	}
	if result.Dependents == nil {
		t.Error("Dependents should not be nil")
	}
}

func TestScopeTrailingSlash(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pkg/a.go", `package pkg
func FuncA() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Scope("pkg/", false)
	if err != nil {
		t.Fatalf("Scope() error: %v", err)
	}

	if result.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1 (trailing slash should be normalized)", result.FileCount)
	}
}

func TestFormatScopeEmpty(t *testing.T) {
	result := &ScopeResult{
		Directory:    "missing",
		Files:        []string{},
		FileCount:    0,
		LOC:          0,
		Symbols:      map[string]SymbolCount{},
		Dependencies: []string{},
		Dependents:   []string{},
	}
	out := FormatScope(result)
	if !strings.Contains(out, "No files found") {
		t.Errorf("output missing 'No files found': %s", out)
	}
}

func TestFormatScopeWithData(t *testing.T) {
	result := &ScopeResult{
		Directory: "index",
		Files:     []string{"a.go", "b.go"},
		FileCount: 2,
		LOC:       150,
		Symbols: map[string]SymbolCount{
			"func": {Exported: 5, Internal: 2},
			"type": {Exported: 3, Internal: 1},
		},
		Dependencies: []string{"parsers"},
		Dependents:   []string{"."},
	}
	out := FormatScope(result)
	if !strings.Contains(out, "Scope: index/") {
		t.Errorf("output missing header: %s", out)
	}
	if !strings.Contains(out, "2 files, 150 LOC") {
		t.Errorf("output missing file/LOC summary: %s", out)
	}
	if !strings.Contains(out, "func") {
		t.Errorf("output missing 'func' symbol kind: %s", out)
	}
	if !strings.Contains(out, "parsers/") {
		t.Errorf("output missing dependency: %s", out)
	}
}

func TestInScope(t *testing.T) {
	tests := []struct {
		filePath  string
		dir       string
		recursive bool
		want      bool
	}{
		{"pkg/a.go", "pkg", false, true},
		{"pkg/sub/b.go", "pkg", false, false},
		{"pkg/sub/b.go", "pkg", true, true},
		{"other/a.go", "pkg", false, false},
		{"other/a.go", "pkg", true, false},
		{"a.go", ".", false, true},
		{"pkg/a.go", ".", false, false},
		{"pkg/a.go", ".", true, true},
	}

	for _, tt := range tests {
		got := inScope(tt.filePath, tt.dir, tt.recursive)
		if got != tt.want {
			t.Errorf("inScope(%q, %q, %v) = %v, want %v", tt.filePath, tt.dir, tt.recursive, got, tt.want)
		}
	}
}
