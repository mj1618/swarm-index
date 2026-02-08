package index

import (
	"strings"
	"testing"
)

func TestTestMapGoFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pkg/a.go", `package pkg
func FuncA() {}
`)
	mkFile(t, tmp, "pkg/a_test.go", `package pkg
func TestFuncA(t *testing.T) {}
`)
	mkFile(t, tmp, "pkg/b.go", `package pkg
func FuncB() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	if result.Summary.TotalSourceFiles != 2 {
		t.Errorf("TotalSourceFiles = %d, want 2", result.Summary.TotalSourceFiles)
	}
	if result.Summary.TestedFiles != 1 {
		t.Errorf("TestedFiles = %d, want 1", result.Summary.TestedFiles)
	}
	if result.Summary.UntestedFiles != 1 {
		t.Errorf("UntestedFiles = %d, want 1", result.Summary.UntestedFiles)
	}
	if result.Summary.CoverageRatio != 0.5 {
		t.Errorf("CoverageRatio = %f, want 0.5", result.Summary.CoverageRatio)
	}

	// a.go should have a test, b.go should not.
	for _, e := range result.Entries {
		if e.SourceFile == "pkg/a.go" {
			if !e.HasTest {
				t.Error("pkg/a.go should have a test")
			}
			if e.TestFile != "pkg/a_test.go" {
				t.Errorf("pkg/a.go test = %q, want %q", e.TestFile, "pkg/a_test.go")
			}
		}
		if e.SourceFile == "pkg/b.go" {
			if e.HasTest {
				t.Error("pkg/b.go should not have a test")
			}
		}
	}
}

func TestTestMapPythonFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.py", `def main(): pass
`)
	mkFile(t, tmp, "test_app.py", `def test_main(): pass
`)
	mkFile(t, tmp, "utils.py", `def helper(): pass
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	if result.Summary.TotalSourceFiles != 2 {
		t.Errorf("TotalSourceFiles = %d, want 2", result.Summary.TotalSourceFiles)
	}
	if result.Summary.TestedFiles != 1 {
		t.Errorf("TestedFiles = %d, want 1", result.Summary.TestedFiles)
	}

	for _, e := range result.Entries {
		if e.SourceFile == "app.py" && !e.HasTest {
			t.Error("app.py should have test_app.py as test")
		}
		if e.SourceFile == "utils.py" && e.HasTest {
			t.Error("utils.py should not have a test")
		}
	}
}

func TestTestMapJSFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "src/app.ts", `export function main() {}
`)
	mkFile(t, tmp, "src/app.test.ts", `test('main', () => {})
`)
	mkFile(t, tmp, "src/utils.ts", `export function helper() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	if result.Summary.TotalSourceFiles != 2 {
		t.Errorf("TotalSourceFiles = %d, want 2", result.Summary.TotalSourceFiles)
	}
	if result.Summary.TestedFiles != 1 {
		t.Errorf("TestedFiles = %d, want 1", result.Summary.TestedFiles)
	}

	for _, e := range result.Entries {
		if e.SourceFile == "src/app.ts" && !e.HasTest {
			t.Error("src/app.ts should have src/app.test.ts as test")
		}
	}
}

func TestTestMapPathFilter(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pkg/a.go", `package pkg
func FuncA() {}
`)
	mkFile(t, tmp, "pkg/a_test.go", `package pkg
func TestFuncA(t *testing.T) {}
`)
	mkFile(t, tmp, "other/b.go", `package other
func FuncB() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("pkg", false, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	if result.Summary.TotalSourceFiles != 1 {
		t.Errorf("TotalSourceFiles = %d, want 1 (filtered to pkg)", result.Summary.TotalSourceFiles)
	}
	if len(result.Entries) != 1 {
		t.Errorf("Entries = %d, want 1", len(result.Entries))
	}
}

func TestTestMapUntestedFilter(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main
func FuncA() {}
`)
	mkFile(t, tmp, "a_test.go", `package main
func TestFuncA(t *testing.T) {}
`)
	mkFile(t, tmp, "b.go", `package main
func FuncB() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", true, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	// Only untested files should appear.
	for _, e := range result.Entries {
		if e.HasTest {
			t.Errorf("entry %q should not appear with --untested", e.SourceFile)
		}
	}
	if len(result.Entries) != 1 {
		t.Errorf("Entries = %d, want 1 (only untested)", len(result.Entries))
	}
}

func TestTestMapTestedFilter(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main
func FuncA() {}
`)
	mkFile(t, tmp, "a_test.go", `package main
func TestFuncA(t *testing.T) {}
`)
	mkFile(t, tmp, "b.go", `package main
func FuncB() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, true, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	// Only tested files should appear.
	for _, e := range result.Entries {
		if !e.HasTest {
			t.Errorf("entry %q should not appear with --tested", e.SourceFile)
		}
	}
	if len(result.Entries) != 1 {
		t.Errorf("Entries = %d, want 1 (only tested)", len(result.Entries))
	}
}

func TestTestMapMaxLimit(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main
func A() {}
`)
	mkFile(t, tmp, "b.go", `package main
func B() {}
`)
	mkFile(t, tmp, "c.go", `package main
func C() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, false, 2)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	if len(result.Entries) != 2 {
		t.Errorf("Entries = %d, want 2 (max limit)", len(result.Entries))
	}
	// Summary should reflect all files regardless of max.
	if result.Summary.TotalSourceFiles != 3 {
		t.Errorf("TotalSourceFiles = %d, want 3", result.Summary.TotalSourceFiles)
	}
}

func TestTestMapNoSourceFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "README.md", `# Hello`)
	mkFile(t, tmp, "config.yaml", `key: value`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	if result.Summary.TotalSourceFiles != 0 {
		t.Errorf("TotalSourceFiles = %d, want 0", result.Summary.TotalSourceFiles)
	}
	if len(result.Entries) != 0 {
		t.Errorf("Entries = %d, want 0", len(result.Entries))
	}
}

func TestTestMapSortedByPath(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "z.go", `package main
func Z() {}
`)
	mkFile(t, tmp, "a.go", `package main
func A() {}
`)
	mkFile(t, tmp, "m.go", `package main
func M() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	for i := 1; i < len(result.Entries); i++ {
		if result.Entries[i].SourceFile < result.Entries[i-1].SourceFile {
			t.Errorf("entries not sorted: %q before %q", result.Entries[i-1].SourceFile, result.Entries[i].SourceFile)
		}
	}
}

func TestIsTestFilePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"foo_test.go", true},
		{"foo.go", false},
		{"foo.test.ts", true},
		{"foo.spec.js", true},
		{"foo.ts", false},
		{"test_foo.py", true},
		{"foo_test.py", true},
		{"foo.py", false},
		{"README.md", false},
	}
	for _, tt := range tests {
		got := isTestFilePath(tt.path)
		if got != tt.want {
			t.Errorf("isTestFilePath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestFormatTestMapEmpty(t *testing.T) {
	result := &TestMapResult{
		Summary: TestMapSummary{},
		Entries: []TestMapEntry{},
	}
	out := FormatTestMap(result)
	if !strings.Contains(out, "No source files found") {
		t.Errorf("output missing 'No source files found': %s", out)
	}
}

func TestFormatTestMapWithData(t *testing.T) {
	result := &TestMapResult{
		Summary: TestMapSummary{
			TotalSourceFiles: 3,
			TestedFiles:      2,
			UntestedFiles:    1,
			CoverageRatio:    0.667,
		},
		Entries: []TestMapEntry{
			{SourceFile: "a.go", TestFile: "a_test.go", HasTest: true},
			{SourceFile: "b.go", TestFile: "b_test.go", HasTest: true},
			{SourceFile: "c.go", TestFile: "", HasTest: false},
		},
	}
	out := FormatTestMap(result)
	if !strings.Contains(out, "2/3 source files have tests") {
		t.Errorf("output missing summary: %s", out)
	}
	if !strings.Contains(out, "Tested:") {
		t.Errorf("output missing 'Tested:' section: %s", out)
	}
	if !strings.Contains(out, "Untested:") {
		t.Errorf("output missing 'Untested:' section: %s", out)
	}
	if !strings.Contains(out, "a_test.go") {
		t.Errorf("output missing test file name: %s", out)
	}
	if !strings.Contains(out, "no test file found") {
		t.Errorf("output missing 'no test file found': %s", out)
	}
}

func TestTestMapJSSpecFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "src/component.jsx", `export function Component() {}
`)
	mkFile(t, tmp, "src/component.spec.jsx", `test('Component', () => {})
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.TestMap("", false, false, 100)
	if err != nil {
		t.Fatalf("TestMap() error: %v", err)
	}

	if result.Summary.TestedFiles != 1 {
		t.Errorf("TestedFiles = %d, want 1", result.Summary.TestedFiles)
	}
}
