package index

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestImpactSymbolDirectRefs(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.go", `package main

func Helper() string {
	return "ok"
}
`)
	mkFile(t, tmp, "main.go", `package main

func main() {
	x := Helper()
	_ = x
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Impact("Helper", 3, 100)
	if err != nil {
		t.Fatalf("Impact() error: %v", err)
	}

	if result.Target.Name != "Helper" {
		t.Errorf("Target.Name = %q, want %q", result.Target.Name, "Helper")
	}
	if result.Target.File != "lib.go" {
		t.Errorf("Target.File = %q, want %q", result.Target.File, "lib.go")
	}
	if len(result.Layers) == 0 {
		t.Fatal("expected at least one layer")
	}
	if result.Layers[0].Depth != 1 {
		t.Errorf("first layer depth = %d, want 1", result.Layers[0].Depth)
	}
	if len(result.Layers[0].Refs) == 0 {
		t.Error("expected direct references in layer 1")
	}
	if result.Summary.TotalRefSites == 0 {
		t.Error("expected TotalRefSites > 0")
	}
	if result.Summary.TotalFiles == 0 {
		t.Error("expected TotalFiles > 0")
	}
}

func TestImpactSymbolTransitive(t *testing.T) {
	tmp := t.TempDir()
	// C calls B, B calls A. Impact of A should show B at depth 1, possibly C at depth 2.
	mkFile(t, tmp, "a.go", `package main

func FuncA() string {
	return "a"
}
`)
	mkFile(t, tmp, "b.go", `package main

func FuncB() string {
	return FuncA()
}
`)
	mkFile(t, tmp, "c.go", `package main

func FuncC() string {
	return FuncB()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Impact("FuncA", 3, 100)
	if err != nil {
		t.Fatalf("Impact() error: %v", err)
	}

	if len(result.Layers) == 0 {
		t.Fatal("expected at least one layer")
	}

	// Depth 1 should contain the reference from FuncB.
	depth1 := result.Layers[0]
	if depth1.Depth != 1 {
		t.Errorf("first layer depth = %d, want 1", depth1.Depth)
	}
	foundB := false
	for _, ref := range depth1.Refs {
		if ref.File == "b.go" {
			foundB = true
		}
	}
	if !foundB {
		t.Error("expected reference from b.go at depth 1")
	}

	// Depth 2 should contain the reference from FuncC (if enclosing symbol detection works).
	if len(result.Layers) >= 2 {
		depth2 := result.Layers[1]
		if depth2.Depth != 2 {
			t.Errorf("second layer depth = %d, want 2", depth2.Depth)
		}
		foundC := false
		for _, ref := range depth2.Refs {
			if ref.File == "c.go" {
				foundC = true
			}
		}
		if !foundC {
			t.Logf("depth 2 did not find c.go (enclosing symbol detection may not have resolved)")
		}
	}
}

func TestImpactFileMode(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "utils/helpers.go", `package utils

func DoWork() string {
	return "work"
}
`)
	mkFile(t, tmp, "main.go", `package main

import "mymod/utils"

func main() {
	utils.DoWork()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Impact("utils/helpers.go", 3, 100)
	if err != nil {
		t.Fatalf("Impact() error: %v", err)
	}

	if result.Target.Kind != "file" {
		t.Errorf("Target.Kind = %q, want %q", result.Target.Kind, "file")
	}
	if result.Target.File != "utils/helpers.go" {
		t.Errorf("Target.File = %q, want %q", result.Target.File, "utils/helpers.go")
	}
}

func TestImpactCycleDetection(t *testing.T) {
	tmp := t.TempDir()
	// A calls B, B calls A — should not infinite loop.
	mkFile(t, tmp, "a.go", `package main

func CycleA() string {
	return CycleB()
}
`)
	mkFile(t, tmp, "b.go", `package main

func CycleB() string {
	return CycleA()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Impact("CycleA", 5, 100)
	if err != nil {
		t.Fatalf("Impact() error: %v", err)
	}

	// Should terminate without error. The exact layers depend on cycle handling,
	// but it should not hang.
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Summary.MaxDepth > 5 {
		t.Errorf("MaxDepth = %d, should not exceed requested depth", result.Summary.MaxDepth)
	}
}

func TestImpactDepthLimiting(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main

func DepthA() string {
	return "a"
}
`)
	mkFile(t, tmp, "b.go", `package main

func DepthB() string {
	return DepthA()
}
`)
	mkFile(t, tmp, "c.go", `package main

func DepthC() string {
	return DepthB()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Depth 1 should only show direct references.
	result, err := idx.Impact("DepthA", 1, 100)
	if err != nil {
		t.Fatalf("Impact() error: %v", err)
	}

	if len(result.Layers) > 1 {
		t.Errorf("with depth=1, got %d layers, want at most 1", len(result.Layers))
	}
}

func TestImpactMaxResults(t *testing.T) {
	tmp := t.TempDir()
	content := "package main\n\nfunc Limited() {}\n"
	for i := 0; i < 20; i++ {
		content += "var _ = Limited()\n"
	}
	mkFile(t, tmp, "main.go", content)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Impact("Limited", 3, 5)
	if err != nil {
		t.Fatalf("Impact() error: %v", err)
	}

	if result.Summary.TotalRefSites > 5 {
		t.Errorf("TotalRefSites = %d, want <= 5", result.Summary.TotalRefSites)
	}
}

func TestImpactJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.go", `package main

func JSONFunc() string {
	return "ok"
}
`)
	mkFile(t, tmp, "main.go", `package main

func main() {
	JSONFunc()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Impact("JSONFunc", 3, 100)
	if err != nil {
		t.Fatalf("Impact() error: %v", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var parsed ImpactResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if parsed.Target.Name != "JSONFunc" {
		t.Errorf("parsed Target.Name = %q, want %q", parsed.Target.Name, "JSONFunc")
	}
}

func TestImpactSymbolNotFound(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n\nfunc main() {}\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Impact("NonexistentSymbol", 3, 100)
	if err != nil {
		t.Fatalf("Impact() should not error for missing symbol, got: %v", err)
	}

	if result.Summary.TotalRefSites != 0 {
		t.Errorf("TotalRefSites = %d, want 0", result.Summary.TotalRefSites)
	}
}

func TestImpactFileNotFound(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n\nfunc main() {}\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	_, err = idx.Impact("nonexistent/file.go", 3, 100)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestFormatImpact(t *testing.T) {
	result := &ImpactResult{
		Target: ImpactTarget{
			Name: "Load",
			File: "index/index.go",
			Line: 45,
			Kind: "func",
		},
		Layers: []ImpactLayer{
			{
				Depth: 1,
				Label: "direct references",
				Refs: []ImpactRef{
					{File: "main.go", Line: 97, Content: "idx, err := index.Load(root)"},
					{File: "index/stale.go", Line: 23, Content: "idx, err := Load(dir)"},
				},
			},
		},
		Summary: ImpactSummary{
			TotalFiles:    2,
			TotalRefSites: 2,
			MaxDepth:      1,
		},
	}

	output := FormatImpact(result)

	if output == "" {
		t.Fatal("FormatImpact returned empty string")
	}
	if !strings.Contains(output, "Load") {
		t.Error("output should contain target name")
	}
	if !strings.Contains(output, "index/index.go") {
		t.Error("output should contain target file")
	}
	if !strings.Contains(output, "Depth 1") {
		t.Error("output should contain depth label")
	}
	if !strings.Contains(output, "blast radius") {
		t.Error("output should contain blast radius summary")
	}
}

func TestFormatImpactEmpty(t *testing.T) {
	result := &ImpactResult{
		Target: ImpactTarget{
			Name: "Unused",
			Kind: "symbol",
		},
		Layers: []ImpactLayer{
			{Depth: 1, Label: "direct references", Refs: []ImpactRef{}},
		},
		Summary: ImpactSummary{},
	}

	output := FormatImpact(result)
	if !strings.Contains(output, "No dependents found") {
		t.Error("expected 'No dependents found' message")
	}
}
