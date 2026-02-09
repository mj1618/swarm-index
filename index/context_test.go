package index

import (
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mj1618/swarm-index/parsers" // register parsers
)

func TestContextGoFunction(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

import (
	"fmt"
	"os"
)

// Run executes the main logic.
// It returns an error on failure.
func Run(args []string) error {
	fmt.Println(args)
	return nil
}

func helper() {}
`)

	result, err := Context(filepath.Join(tmp, "main.go"), "Run")
	if err != nil {
		t.Fatalf("Context() error: %v", err)
	}

	if result.Symbol != "Run" {
		t.Errorf("Symbol = %q, want %q", result.Symbol, "Run")
	}
	if result.Kind != "func" {
		t.Errorf("Kind = %q, want %q", result.Kind, "func")
	}
	if result.Line != 10 {
		t.Errorf("Line = %d, want 10", result.Line)
	}

	// Check imports
	if len(result.Imports) != 2 {
		t.Errorf("Imports count = %d, want 2; got %v", len(result.Imports), result.Imports)
	}

	// Check doc comment
	if !strings.Contains(result.DocComment, "Run executes the main logic") {
		t.Errorf("DocComment missing expected text: %q", result.DocComment)
	}
	if !strings.Contains(result.DocComment, "It returns an error") {
		t.Errorf("DocComment missing second line: %q", result.DocComment)
	}

	// Check body
	if !strings.Contains(result.Body, "func Run(args []string) error") {
		t.Errorf("Body missing function signature: %q", result.Body)
	}
	if !strings.Contains(result.Body, "fmt.Println(args)") {
		t.Errorf("Body missing function body: %q", result.Body)
	}
}

func TestContextPythonFunction(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.py", `import os
import sys
from pathlib import Path

# Process the given input file.
def process(filename):
    data = Path(filename).read_text()
    return data

def helper():
    pass
`)

	result, err := Context(filepath.Join(tmp, "app.py"), "process")
	if err != nil {
		t.Fatalf("Context() error: %v", err)
	}

	if result.Symbol != "process" {
		t.Errorf("Symbol = %q, want %q", result.Symbol, "process")
	}

	// Check imports
	if len(result.Imports) < 3 {
		t.Errorf("Imports count = %d, want >= 3; got %v", len(result.Imports), result.Imports)
	}

	// Check doc comment
	if !strings.Contains(result.DocComment, "Process the given input") {
		t.Errorf("DocComment missing expected text: %q", result.DocComment)
	}

	// Check body contains the definition
	if !strings.Contains(result.Body, "def process(filename)") {
		t.Errorf("Body missing function signature: %q", result.Body)
	}
}

func TestContextJSFunction(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "utils.js", `import { readFile } from 'fs';
import path from 'path';

// Format a name for display.
export function formatName(first, last) {
  return first + ' ' + last;
}

function internal() {}
`)

	result, err := Context(filepath.Join(tmp, "utils.js"), "formatName")
	if err != nil {
		t.Fatalf("Context() error: %v", err)
	}

	if result.Symbol != "formatName" {
		t.Errorf("Symbol = %q, want %q", result.Symbol, "formatName")
	}

	// Check imports
	if len(result.Imports) != 2 {
		t.Errorf("Imports count = %d, want 2; got %v", len(result.Imports), result.Imports)
	}

	// Check doc comment
	if !strings.Contains(result.DocComment, "Format a name") {
		t.Errorf("DocComment missing expected text: %q", result.DocComment)
	}

	// Check body
	if !strings.Contains(result.Body, "function formatName") {
		t.Errorf("Body missing function signature: %q", result.Body)
	}
}

func TestContextUnknownSymbol(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func Hello() {}
`)

	_, err := Context(filepath.Join(tmp, "main.go"), "NonExistent")
	if err == nil {
		t.Fatal("Context() should have returned an error for unknown symbol")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestContextUnsupportedExtension(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "data.csv", `a,b,c
1,2,3
`)

	_, err := Context(filepath.Join(tmp, "data.csv"), "anything")
	if err == nil {
		t.Fatal("Context() should have returned an error for unsupported extension")
	}
	if !strings.Contains(err.Error(), "no parser") {
		t.Errorf("error should mention 'no parser': %v", err)
	}
}

func TestContextNoDocComment(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func NoComment() string {
	return "hello"
}
`)

	result, err := Context(filepath.Join(tmp, "main.go"), "NoComment")
	if err != nil {
		t.Fatalf("Context() error: %v", err)
	}

	if result.DocComment != "" {
		t.Errorf("DocComment should be empty, got %q", result.DocComment)
	}
}

func TestContextGoImports(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

import "fmt"

func Hello() { fmt.Println("hi") }
`)

	result, err := Context(filepath.Join(tmp, "main.go"), "Hello")
	if err != nil {
		t.Fatalf("Context() error: %v", err)
	}

	if len(result.Imports) != 1 || result.Imports[0] != "fmt" {
		t.Errorf("Imports = %v, want [fmt]", result.Imports)
	}
}

func TestContextFileNotFound(t *testing.T) {
	_, err := Context("/nonexistent/path/file.go", "Foo")
	if err == nil {
		t.Fatal("Context() should have returned an error for missing file")
	}
}

func TestFormatContextFull(t *testing.T) {
	result := &ContextResult{
		File:       "main.go",
		Symbol:     "Run",
		Kind:       "func",
		Line:       10,
		EndLine:    13,
		Signature:  "func Run() error",
		Imports:    []string{"fmt", "os"},
		DocComment: "// Run executes the main logic.",
		Body:       "func Run() error {\n\treturn nil\n}",
	}

	out := FormatContext(result)
	if !strings.Contains(out, "File: main.go") {
		t.Errorf("output missing file header: %s", out)
	}
	if !strings.Contains(out, "Imports:") {
		t.Errorf("output missing imports header: %s", out)
	}
	if !strings.Contains(out, "fmt") {
		t.Errorf("output missing import 'fmt': %s", out)
	}
	if !strings.Contains(out, "// Run executes") {
		t.Errorf("output missing doc comment: %s", out)
	}
	if !strings.Contains(out, "func Run() error") {
		t.Errorf("output missing body: %s", out)
	}
}

func TestFormatContextNoImportsNoDoc(t *testing.T) {
	result := &ContextResult{
		File:       "main.go",
		Symbol:     "helper",
		Kind:       "func",
		Line:       3,
		EndLine:    5,
		Signature:  "func helper()",
		Imports:    []string{},
		DocComment: "",
		Body:       "func helper() {\n}",
	}

	out := FormatContext(result)
	if strings.Contains(out, "Imports:") {
		t.Errorf("output should not contain Imports header when empty: %s", out)
	}
	if !strings.Contains(out, "func helper()") {
		t.Errorf("output missing body: %s", out)
	}
}
