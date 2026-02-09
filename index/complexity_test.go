package index

import (
	"strings"
	"testing"

	_ "github.com/mj1618/swarm-index/parsers" // register parsers
)

func TestComplexityGoBasic(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

import "fmt"

func simple() {
	fmt.Println("hello")
}

func complex() {
	for i := 0; i < 10; i++ {
		if i > 5 {
			if i%2 == 0 {
				fmt.Println(i)
			}
		}
	}
	switch {
	case true:
		fmt.Println("a")
	case false:
		fmt.Println("b")
	}
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	if result.TotalFunctions != 2 {
		t.Errorf("TotalFunctions = %d, want 2", result.TotalFunctions)
	}

	// complex() should be first (higher complexity).
	if len(result.Functions) < 2 {
		t.Fatalf("got %d functions, want 2", len(result.Functions))
	}
	if result.Functions[0].Name != "complex" {
		t.Errorf("first function = %q, want 'complex'", result.Functions[0].Name)
	}
	if result.Functions[0].Complexity <= result.Functions[1].Complexity {
		t.Errorf("complex() complexity (%d) should be > simple() complexity (%d)",
			result.Functions[0].Complexity, result.Functions[1].Complexity)
	}
}

func TestComplexitySortingDescending(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func low() {}

func medium() {
	if true {
		for i := 0; i < 1; i++ {}
	}
}

func high() {
	if true {
		if true {
			for i := 0; i < 1; i++ {
				switch {
				case true:
				case false:
				}
			}
		}
	}
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	if len(result.Functions) < 3 {
		t.Fatalf("got %d functions, want 3", len(result.Functions))
	}

	for i := 1; i < len(result.Functions); i++ {
		if result.Functions[i].Complexity > result.Functions[i-1].Complexity {
			t.Errorf("functions not sorted descending: [%d].complexity=%d > [%d].complexity=%d",
				i, result.Functions[i].Complexity, i-1, result.Functions[i-1].Complexity)
		}
	}
}

func TestComplexityMinThreshold(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func simple() {}

func medium() {
	if true {}
	if true {}
	for i := 0; i < 1; i++ {}
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 3)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	// Only medium() should be included (complexity >= 3).
	for _, f := range result.Functions {
		if f.Complexity < 3 {
			t.Errorf("function %s has complexity %d, below threshold 3", f.Name, f.Complexity)
		}
	}
	// TotalFunctions should still count all functions.
	if result.TotalFunctions != 2 {
		t.Errorf("TotalFunctions = %d, want 2", result.TotalFunctions)
	}
}

func TestComplexitySingleFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main
func A() {
	if true {}
}
`)
	mkFile(t, tmp, "b.go", `package main
func B() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Complexity(tmp+"/a.go", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	// Only functions from a.go should appear.
	if result.TotalFunctions != 1 {
		t.Errorf("TotalFunctions = %d, want 1 (single file mode)", result.TotalFunctions)
	}
	if len(result.Functions) != 1 || result.Functions[0].Name != "A" {
		t.Errorf("expected function A from a.go only, got: %v", result.Functions)
	}
}

func TestComplexityMaxResults(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func A() { if true {} }
func B() { if true {} }
func C() { if true {} }
func D() { if true {} }
func E() { if true {} }
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 2, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	if len(result.Functions) != 2 {
		t.Errorf("got %d functions, want 2 (max limit)", len(result.Functions))
	}
	if result.TotalFunctions != 5 {
		t.Errorf("TotalFunctions = %d, want 5 (all counted)", result.TotalFunctions)
	}
}

func TestComplexityGoMethods(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

type Server struct{}

func (s *Server) Handle() {
	if true {
		for i := 0; i < 1; i++ {}
	}
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	found := false
	for _, f := range result.Functions {
		if f.Name == "Server.Handle" {
			found = true
			if f.Complexity < 3 {
				t.Errorf("Server.Handle complexity = %d, want >= 3", f.Complexity)
			}
		}
	}
	if !found {
		t.Error("method 'Server.Handle' not found")
	}
}

func TestComplexityPythonBasic(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.py", `
def simple():
    print("hello")

def complex_func(a, b, c):
    if a > 0:
        for i in range(10):
            if i > 5:
                while True:
                    break
    elif b:
        pass
    if a and b:
        pass
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	if result.TotalFunctions < 2 {
		t.Errorf("TotalFunctions = %d, want >= 2", result.TotalFunctions)
	}

	// complex_func should be first.
	if len(result.Functions) >= 2 {
		if result.Functions[0].Name != "complex_func" {
			t.Errorf("first function = %q, want 'complex_func'", result.Functions[0].Name)
		}
		if result.Functions[0].Params != 3 {
			t.Errorf("complex_func params = %d, want 3", result.Functions[0].Params)
		}
	}
}

func TestComplexityJSBasic(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.js", `
function simple() {
  console.log("hello");
}

function complex(a, b) {
  if (a > 0) {
    for (let i = 0; i < 10; i++) {
      if (i > 5) {
        while (true) {
          break;
        }
      }
    }
  }
  if (a && b) {
    console.log("both");
  }
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	if result.TotalFunctions < 2 {
		t.Errorf("TotalFunctions = %d, want >= 2", result.TotalFunctions)
	}

	// complex should be first.
	if len(result.Functions) >= 2 {
		if result.Functions[0].Name != "complex" {
			t.Errorf("first function = %q, want 'complex'", result.Functions[0].Name)
		}
	}
}

func TestComplexityJSONStructure(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func A() {
	if true {}
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	if result.TotalFunctions != 1 {
		t.Errorf("TotalFunctions = %d, want 1", result.TotalFunctions)
	}
	if len(result.Functions) != 1 {
		t.Fatalf("got %d functions, want 1", len(result.Functions))
	}

	f := result.Functions[0]
	if f.Path != "main.go" {
		t.Errorf("Path = %q, want 'main.go'", f.Path)
	}
	if f.Name != "A" {
		t.Errorf("Name = %q, want 'A'", f.Name)
	}
	if f.Line <= 0 {
		t.Errorf("Line = %d, want > 0", f.Line)
	}
	if f.EndLine < f.Line {
		t.Errorf("EndLine = %d < Line = %d", f.EndLine, f.Line)
	}
	if f.Lines <= 0 {
		t.Errorf("Lines = %d, want > 0", f.Lines)
	}
	if f.Complexity < 1 {
		t.Errorf("Complexity = %d, want >= 1 (base complexity)", f.Complexity)
	}
	if result.AvgComplexity <= 0 {
		t.Errorf("AvgComplexity = %f, want > 0", result.AvgComplexity)
	}
}

func TestFormatComplexityEmpty(t *testing.T) {
	result := &ComplexityResult{
		Functions: nil,
	}
	out := FormatComplexity(result)
	if !strings.Contains(out, "No functions found") {
		t.Errorf("output missing 'No functions found': %s", out)
	}
}

func TestFormatComplexityWithResults(t *testing.T) {
	result := &ComplexityResult{
		Functions: []FunctionComplexity{
			{Path: "main.go", Name: "complex", Line: 10, EndLine: 50, Complexity: 8, Lines: 41, MaxDepth: 3},
			{Path: "main.go", Name: "simple", Line: 1, EndLine: 5, Complexity: 1, Lines: 5, MaxDepth: 0},
		},
		TotalFunctions:      2,
		AvgComplexity:       4.5,
		MaxComplexity:       8,
		HighComplexityCount: 0,
	}
	out := FormatComplexity(result)
	if !strings.Contains(out, "Complexity Report") {
		t.Errorf("output missing 'Complexity Report': %s", out)
	}
	if !strings.Contains(out, "complex()") {
		t.Errorf("output missing 'complex()': %s", out)
	}
	if !strings.Contains(out, "main.go:10") {
		t.Errorf("output missing 'main.go:10': %s", out)
	}
	if !strings.Contains(out, "avg complexity=4.5") {
		t.Errorf("output missing avg complexity: %s", out)
	}
}

func TestComplexityGoParams(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func noParams() {}
func twoParams(a int, b string) {}
func threeParams(a, b int, c string) {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	paramsByName := map[string]int{}
	for _, f := range result.Functions {
		paramsByName[f.Name] = f.Params
	}

	if paramsByName["noParams"] != 0 {
		t.Errorf("noParams params = %d, want 0", paramsByName["noParams"])
	}
	if paramsByName["twoParams"] != 2 {
		t.Errorf("twoParams params = %d, want 2", paramsByName["twoParams"])
	}
	if paramsByName["threeParams"] != 3 {
		t.Errorf("threeParams params = %d, want 3", paramsByName["threeParams"])
	}
}

func TestComplexityHighComplexityCount(t *testing.T) {
	tmp := t.TempDir()
	// Create a function with enough branching to hit complexity >= 10.
	mkFile(t, tmp, "main.go", `package main

func veryComplex() {
	if true {}
	if true {}
	if true {}
	if true {}
	if true {}
	if true {}
	if true {}
	if true {}
	if true {}
	for i := 0; i < 1; i++ {}
}

func simple() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Complexity("", 20, 0)
	if err != nil {
		t.Fatalf("Complexity() error: %v", err)
	}

	if result.HighComplexityCount < 1 {
		t.Errorf("HighComplexityCount = %d, want >= 1", result.HighComplexityCount)
	}
	if result.MaxComplexity < 10 {
		t.Errorf("MaxComplexity = %d, want >= 10", result.MaxComplexity)
	}
}
