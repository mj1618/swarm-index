package index

import (
	"strings"
	"testing"
)

func TestRelatedGoImports(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

import (
	"fmt"
	"myproject/utils"
)

func main() {
	fmt.Println(utils.Hello())
}
`)
	mkFile(t, tmp, "utils/helpers.go", `package utils

func Hello() string { return "hi" }
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("main.go")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	if result.File != "main.go" {
		t.Errorf("File = %q, want %q", result.File, "main.go")
	}

	// main.go imports "myproject/utils" which should resolve to utils/helpers.go
	found := false
	for _, imp := range result.Imports {
		if imp == "utils/helpers.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected utils/helpers.go in imports, got %v", result.Imports)
	}
}

func TestRelatedGoSingleImport(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

import "myproject/lib"

func main() {
	lib.Do()
}
`)
	mkFile(t, tmp, "lib/lib.go", `package lib

func Do() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("main.go")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	found := false
	for _, imp := range result.Imports {
		if imp == "lib/lib.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected lib/lib.go in imports, got %v", result.Imports)
	}
}

func TestRelatedJSImports(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.js", `
import { helper } from './utils';
const other = require('./lib');
`)
	mkFile(t, tmp, "utils.js", `export function helper() {}`)
	mkFile(t, tmp, "lib.js", `module.exports = {}`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("app.js")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	imports := make(map[string]bool)
	for _, imp := range result.Imports {
		imports[imp] = true
	}

	if !imports["utils.js"] {
		t.Errorf("expected utils.js in imports, got %v", result.Imports)
	}
	if !imports["lib.js"] {
		t.Errorf("expected lib.js in imports, got %v", result.Imports)
	}
}

func TestRelatedTSImports(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.ts", `
import { Component } from './component';
`)
	mkFile(t, tmp, "component.ts", `export class Component {}`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("app.ts")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	found := false
	for _, imp := range result.Imports {
		if imp == "component.ts" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected component.ts in imports, got %v", result.Imports)
	}
}

func TestRelatedPyImports(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.py", `
from utils import helper
import lib
`)
	mkFile(t, tmp, "utils.py", `def helper(): pass`)
	mkFile(t, tmp, "lib.py", `x = 1`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("app.py")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	imports := make(map[string]bool)
	for _, imp := range result.Imports {
		imports[imp] = true
	}

	if !imports["utils.py"] {
		t.Errorf("expected utils.py in imports, got %v", result.Imports)
	}
	if !imports["lib.py"] {
		t.Errorf("expected lib.py in imports, got %v", result.Imports)
	}
}

func TestRelatedImporters(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "utils.js", `export function helper() {}`)
	mkFile(t, tmp, "app.js", `import { helper } from './utils';`)
	mkFile(t, tmp, "main.js", `const u = require('./utils');`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("utils.js")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	importers := make(map[string]bool)
	for _, imp := range result.Importers {
		importers[imp] = true
	}

	if !importers["app.js"] {
		t.Errorf("expected app.js in importers, got %v", result.Importers)
	}
	if !importers["main.js"] {
		t.Errorf("expected main.js in importers, got %v", result.Importers)
	}
}

func TestRelatedGoTestFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "handler.go", `package main
func Handle() {}
`)
	mkFile(t, tmp, "handler_test.go", `package main
func TestHandle(t *testing.T) {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("handler.go")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	found := false
	for _, tf := range result.TestFiles {
		if tf == "handler_test.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected handler_test.go in test files, got %v", result.TestFiles)
	}
}

func TestRelatedJSTestFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "widget.ts", `export class Widget {}`)
	mkFile(t, tmp, "widget.test.ts", `describe("Widget", () => {})`)
	mkFile(t, tmp, "widget.spec.ts", `describe("Widget", () => {})`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("widget.ts")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	tests := make(map[string]bool)
	for _, tf := range result.TestFiles {
		tests[tf] = true
	}

	if !tests["widget.test.ts"] {
		t.Errorf("expected widget.test.ts in test files, got %v", result.TestFiles)
	}
	if !tests["widget.spec.ts"] {
		t.Errorf("expected widget.spec.ts in test files, got %v", result.TestFiles)
	}
}

func TestRelatedPyTestFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "handler.py", `def handle(): pass`)
	mkFile(t, tmp, "test_handler.py", `def test_handle(): pass`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("handler.py")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	found := false
	for _, tf := range result.TestFiles {
		if tf == "test_handler.py" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected test_handler.py in test files, got %v", result.TestFiles)
	}
}

func TestRelatedNoRelations(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lonely.go", `package main
func main() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("lonely.go")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	if len(result.Imports) != 0 {
		t.Errorf("Imports = %v, want empty", result.Imports)
	}
	if len(result.Importers) != 0 {
		t.Errorf("Importers = %v, want empty", result.Importers)
	}
	if len(result.TestFiles) != 0 {
		t.Errorf("TestFiles = %v, want empty", result.TestFiles)
	}
}

func TestRelatedFileNotInIndex(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	_, err = idx.Related("nonexistent.go")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestRelatedJSIndexFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.js", `import stuff from './components';`)
	mkFile(t, tmp, "components/index.js", `export default {}`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("app.js")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	found := false
	for _, imp := range result.Imports {
		if imp == "components/index.js" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected components/index.js in imports, got %v", result.Imports)
	}
}

func TestRelatedNonImportableFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "data.csv", `a,b,c`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("data.csv")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	if len(result.Imports) != 0 {
		t.Errorf("Imports = %v, want empty", result.Imports)
	}
}

func TestFormatRelatedEmpty(t *testing.T) {
	r := &RelatedResult{
		File:      "main.go",
		Imports:   []string{},
		Importers: []string{},
		TestFiles: []string{},
	}
	out := FormatRelated(r)
	if !strings.Contains(out, "Related files for main.go") {
		t.Errorf("output missing header: %s", out)
	}
	if !strings.Contains(out, "No related files found") {
		t.Errorf("output missing 'No related files found': %s", out)
	}
}

func TestFormatRelatedWithData(t *testing.T) {
	r := &RelatedResult{
		File:      "main.go",
		Imports:   []string{"index/index.go", "parsers/parsers.go"},
		Importers: []string{"main_test.go"},
		TestFiles: []string{"main_test.go"},
	}
	out := FormatRelated(r)
	if !strings.Contains(out, "Related files for main.go") {
		t.Errorf("output missing header: %s", out)
	}
	if !strings.Contains(out, "Imports (2)") {
		t.Errorf("output missing 'Imports (2)': %s", out)
	}
	if !strings.Contains(out, "index/index.go") {
		t.Errorf("output missing 'index/index.go': %s", out)
	}
	if !strings.Contains(out, "Imported by (1)") {
		t.Errorf("output missing 'Imported by (1)': %s", out)
	}
	if !strings.Contains(out, "Test files (1)") {
		t.Errorf("output missing 'Test files (1)': %s", out)
	}
}

func TestRelatedPyTestsDir(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "handler.py", `def handle(): pass`)
	mkFile(t, tmp, "tests/test_handler.py", `def test_handle(): pass`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("handler.py")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	found := false
	for _, tf := range result.TestFiles {
		if tf == "tests/test_handler.py" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected tests/test_handler.py in test files, got %v", result.TestFiles)
	}
}

func TestRelatedJSTestsDir(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "widget.js", `export class Widget {}`)
	mkFile(t, tmp, "__tests__/widget.js", `describe("Widget", () => {})`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Related("widget.js")
	if err != nil {
		t.Fatalf("Related() error: %v", err)
	}

	found := false
	for _, tf := range result.TestFiles {
		if tf == "__tests__/widget.js" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected __tests__/widget.js in test files, got %v", result.TestFiles)
	}
}
