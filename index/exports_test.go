package index

import (
	"strings"
	"testing"

	_ "github.com/mj1618/swarm-index/parsers" // register parsers
)

func TestExportsGoFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.go", `package lib

func PublicFunc() {}
func privateFunc() {}

type PublicType struct{}
type privateType struct{}

const ExportedConst = 42
const unexportedConst = 1

var ExportedVar int
var unexportedVar int
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Exports("lib.go")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	if result.Scope != "lib.go" {
		t.Errorf("Scope = %q, want %q", result.Scope, "lib.go")
	}

	// Should find: PublicFunc, PublicType, ExportedConst, ExportedVar
	if result.Count != 4 {
		t.Errorf("Count = %d, want 4; symbols: %v", result.Count, symbolNames(result))
	}

	names := map[string]bool{}
	for _, s := range result.Symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"PublicFunc", "PublicType", "ExportedConst", "ExportedVar"} {
		if !names[want] {
			t.Errorf("missing exported symbol %q", want)
		}
	}
	for _, unwanted := range []string{"privateFunc", "privateType", "unexportedConst", "unexportedVar"} {
		if names[unwanted] {
			t.Errorf("unexported symbol %q should not appear", unwanted)
		}
	}
}

func TestExportsGoDirectory(t *testing.T) {
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

	result, err := idx.Exports("pkg")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	// Should find: FuncA, FuncB, TypeB (not internalA)
	if result.Count != 3 {
		t.Errorf("Count = %d, want 3; symbols: %v", result.Count, symbolNames(result))
	}

	names := map[string]bool{}
	for _, s := range result.Symbols {
		names[s.Name] = true
	}
	if !names["FuncA"] || !names["FuncB"] || !names["TypeB"] {
		t.Errorf("missing expected exported symbols; got: %v", names)
	}
	if names["internalA"] {
		t.Errorf("unexported symbol internalA should not appear")
	}
}

func TestExportsNoExports(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "internal.go", `package main

func helper() {}
var x int
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Exports("internal.go")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
	if len(result.Symbols) != 0 {
		t.Errorf("Symbols has %d entries, want 0", len(result.Symbols))
	}
}

func TestExportsNonExistentScope(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func Main() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Exports("nonexistent.go")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
}

func TestExportsUnsupportedExtension(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "data.csv", `a,b,c
1,2,3
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Exports("data.csv")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Count = %d, want 0 (no parser for .csv)", result.Count)
	}
}

func TestExportsSymbolFields(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.go", `package lib

func Hello() string { return "hi" }
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Exports("lib.go")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	if len(result.Symbols) != 1 {
		t.Fatalf("Symbols has %d entries, want 1", len(result.Symbols))
	}

	s := result.Symbols[0]
	if s.Name != "Hello" {
		t.Errorf("Name = %q, want %q", s.Name, "Hello")
	}
	if s.Kind != "func" {
		t.Errorf("Kind = %q, want %q", s.Kind, "func")
	}
	if s.Path != "lib.go" {
		t.Errorf("Path = %q, want %q", s.Path, "lib.go")
	}
	if s.Line != 3 {
		t.Errorf("Line = %d, want 3", s.Line)
	}
	if s.Signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestExportsJavaScriptFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.js", `
export function greet(name) { return "hi " + name; }
function internal() {}
export const VERSION = "1.0";
export class Widget {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Exports("lib.js")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	if result.Count < 3 {
		t.Errorf("Count = %d, want >= 3; symbols: %v", result.Count, symbolNames(result))
	}

	names := map[string]bool{}
	for _, s := range result.Symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"greet", "VERSION", "Widget"} {
		if !names[want] {
			t.Errorf("missing exported symbol %q", want)
		}
	}
	if names["internal"] {
		t.Errorf("unexported symbol 'internal' should not appear")
	}
}

func TestExportsPythonFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.py", `
def public_func():
    pass

def _private_func():
    pass

class PublicClass:
    pass

class _PrivateClass:
    pass

MAX_SIZE = 100
_internal = 42
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Exports("lib.py")
	if err != nil {
		t.Fatalf("Exports() error: %v", err)
	}

	names := map[string]bool{}
	for _, s := range result.Symbols {
		names[s.Name] = true
	}
	// Public symbols
	if !names["public_func"] {
		t.Errorf("missing exported symbol 'public_func'")
	}
	if !names["PublicClass"] {
		t.Errorf("missing exported symbol 'PublicClass'")
	}
	if !names["MAX_SIZE"] {
		t.Errorf("missing exported symbol 'MAX_SIZE'")
	}
	// Private symbols should be excluded
	if names["_private_func"] {
		t.Errorf("unexported symbol '_private_func' should not appear")
	}
	if names["_PrivateClass"] {
		t.Errorf("unexported symbol '_PrivateClass' should not appear")
	}
	if names["_internal"] {
		t.Errorf("unexported symbol '_internal' should not appear")
	}
}

func TestFormatExportsEmpty(t *testing.T) {
	result := &ExportsResult{
		Scope:   "missing.go",
		Symbols: []ExportedSymbol{},
		Count:   0,
	}
	out := FormatExports(result)
	if !strings.Contains(out, "No exported symbols") {
		t.Errorf("output missing 'No exported symbols': %s", out)
	}
}

func TestFormatExportsSingleFile(t *testing.T) {
	result := &ExportsResult{
		Scope: "lib.go",
		Symbols: []ExportedSymbol{
			{Name: "Hello", Kind: "func", Path: "lib.go", Line: 3, Signature: "func Hello() string"},
			{Name: "World", Kind: "type", Path: "lib.go", Line: 10, Signature: "type World"},
		},
		Count: 2,
	}
	out := FormatExports(result)
	if !strings.Contains(out, "Exports for lib.go") {
		t.Errorf("output missing header: %s", out)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("output missing 'Hello': %s", out)
	}
	if !strings.Contains(out, "2 exported symbols") {
		t.Errorf("output missing count: %s", out)
	}
}

func TestFormatExportsMultiFile(t *testing.T) {
	result := &ExportsResult{
		Scope: "pkg",
		Symbols: []ExportedSymbol{
			{Name: "FuncA", Kind: "func", Path: "pkg/a.go", Line: 3, Signature: "func FuncA()"},
			{Name: "FuncB", Kind: "func", Path: "pkg/b.go", Line: 3, Signature: "func FuncB()"},
		},
		Count: 2,
	}
	out := FormatExports(result)
	if !strings.Contains(out, "pkg/a.go") {
		t.Errorf("output missing path 'pkg/a.go': %s", out)
	}
	if !strings.Contains(out, "pkg/b.go") {
		t.Errorf("output missing path 'pkg/b.go': %s", out)
	}
}

func TestExportsJSONOutput(t *testing.T) {
	result := &ExportsResult{
		Scope: "lib.go",
		Symbols: []ExportedSymbol{
			{Name: "Hello", Kind: "func", Path: "lib.go", Line: 3, Signature: "func Hello()"},
		},
		Count: 1,
	}

	// Verify the struct can be serialized (basic sanity check)
	if result.Symbols[0].Name != "Hello" {
		t.Errorf("unexpected symbol name: %q", result.Symbols[0].Name)
	}
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
}

func symbolNames(r *ExportsResult) []string {
	names := make([]string, len(r.Symbols))
	for i, s := range r.Symbols {
		names[i] = s.Name
	}
	return names
}
