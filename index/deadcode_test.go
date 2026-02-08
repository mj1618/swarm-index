package index

import (
	"strings"
	"testing"

	_ "github.com/matt/swarm-index/parsers" // register parsers
)

func TestDeadCodeUnusedFunction(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func UsedFunc() {}
func UnusedFunc() {}
`)
	mkFile(t, tmp, "caller.go", `package main

func caller() {
	UsedFunc()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	names := map[string]bool{}
	for _, c := range result.Candidates {
		names[c.Name] = true
	}

	if names["UsedFunc"] {
		t.Error("UsedFunc should not be reported as dead code")
	}
	if !names["UnusedFunc"] {
		t.Error("UnusedFunc should be reported as dead code")
	}
}

func TestDeadCodeMainExcluded(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func main() {}
func init() {}
func Helper() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	for _, c := range result.Candidates {
		if c.Name == "main" || c.Name == "init" {
			t.Errorf("excluded symbol %q should not be in candidates", c.Name)
		}
	}
}

func TestDeadCodeTestFunctionsExcluded(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func TestSomething() {}
func BenchmarkFoo() {}
func ExampleBar() {}
func UnusedExport() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	for _, c := range result.Candidates {
		if c.Name == "TestSomething" || c.Name == "BenchmarkFoo" || c.Name == "ExampleBar" {
			t.Errorf("test entry point %q should not be in candidates", c.Name)
		}
	}
}

func TestDeadCodeTestFilesSkipped(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.go", `package main

func UsedInTest() {}
`)
	mkFile(t, tmp, "lib_test.go", `package main

func TestLib() {
	UsedInTest()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	// Symbols defined in test files should not appear as candidates.
	for _, c := range result.Candidates {
		if testFilePattern.MatchString(c.Path) {
			t.Errorf("symbol %q from test file %q should not be in candidates", c.Name, c.Path)
		}
	}
}

func TestDeadCodeKindFilter(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func UnusedFunc() {}
type UnusedType struct{}
var UnusedVar int
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("func", "", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	for _, c := range result.Candidates {
		if c.Kind != "func" {
			t.Errorf("got kind %q, want only 'func' results", c.Kind)
		}
	}
}

func TestDeadCodePathFilter(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func TopUnused() {}
`)
	mkFile(t, tmp, "lib/util.go", `package lib

func LibUnused() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "lib/", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	for _, c := range result.Candidates {
		if !strings.HasPrefix(c.Path, "lib/") {
			t.Errorf("candidate %q has path %q, but expected prefix 'lib/'", c.Name, c.Path)
		}
	}

	// Should find LibUnused but not TopUnused.
	names := map[string]bool{}
	for _, c := range result.Candidates {
		names[c.Name] = true
	}
	if !names["LibUnused"] {
		t.Error("expected LibUnused in candidates")
	}
	if names["TopUnused"] {
		t.Error("TopUnused should be excluded by path filter")
	}
}

func TestDeadCodeMaxLimit(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func UnusedA() {}
func UnusedB() {}
func UnusedC() {}
func UnusedD() {}
func UnusedE() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "", 2)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	if len(result.Candidates) != 2 {
		t.Errorf("got %d candidates, want 2 (max limit)", len(result.Candidates))
	}
	if result.TotalCandidates < 5 {
		t.Errorf("TotalCandidates = %d, want >= 5", result.TotalCandidates)
	}
}

func TestDeadCodeNoCandidates(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func UsedFunc() {}
`)
	mkFile(t, tmp, "caller.go", `package main

func Caller() {
	UsedFunc()
}
`)
	mkFile(t, tmp, "other.go", `package main

func Other() {
	Caller()
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	// Other is used by nobody, UsedFunc is used by caller, Caller is used by Other.
	// Only Other should be dead.
	for _, c := range result.Candidates {
		if c.Name == "UsedFunc" || c.Name == "Caller" {
			t.Errorf("%q should not be dead code", c.Name)
		}
	}
}

func TestFormatDeadCodeEmpty(t *testing.T) {
	result := &DeadCodeResult{
		TotalCandidates: 0,
		Candidates:      []DeadCodeCandidate{},
	}
	out := FormatDeadCode(result)
	if !strings.Contains(out, "No dead code candidates found") {
		t.Errorf("output missing 'No dead code candidates found': %s", out)
	}
}

func TestFormatDeadCodeWithResults(t *testing.T) {
	result := &DeadCodeResult{
		TotalCandidates: 2,
		Candidates: []DeadCodeCandidate{
			{Name: "UnusedFunc", Kind: "func", Path: "main.go", Line: 10, Signature: "func UnusedFunc()", Exported: true, References: 0},
			{Name: "OldType", Kind: "type", Path: "lib/old.go", Line: 5, Signature: "type OldType struct", Exported: true, References: 0},
		},
	}
	out := FormatDeadCode(result)
	if !strings.Contains(out, "2 found") {
		t.Errorf("output missing '2 found': %s", out)
	}
	if !strings.Contains(out, "UnusedFunc") {
		t.Errorf("output missing 'UnusedFunc': %s", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Errorf("output missing 'main.go': %s", out)
	}
	if !strings.Contains(out, "lib/old.go") {
		t.Errorf("output missing 'lib/old.go': %s", out)
	}
	if !strings.Contains(out, "0 references") {
		t.Errorf("output missing '0 references': %s", out)
	}
}

func TestFormatDeadCodeTruncated(t *testing.T) {
	result := &DeadCodeResult{
		TotalCandidates: 10,
		Candidates: []DeadCodeCandidate{
			{Name: "FuncA", Kind: "func", Path: "a.go", Line: 1},
		},
	}
	out := FormatDeadCode(result)
	if !strings.Contains(out, "9 more") {
		t.Errorf("output missing truncation notice: %s", out)
	}
}

func TestDeadCodeMultiLanguage(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func GoUnused() {}
`)
	mkFile(t, tmp, "lib.js", `
export function jsUnused() {}
export function jsUsed() {}
`)
	mkFile(t, tmp, "caller.js", `
import { jsUsed } from './lib';
jsUsed();
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.DeadCode("", "", 50)
	if err != nil {
		t.Fatalf("DeadCode() error: %v", err)
	}

	names := map[string]bool{}
	for _, c := range result.Candidates {
		names[c.Name] = true
	}

	if !names["GoUnused"] {
		t.Error("GoUnused should be reported as dead code")
	}
	if names["jsUsed"] {
		t.Error("jsUsed should not be reported as dead code")
	}
}
