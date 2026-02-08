package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEntryPointsGoMain(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "main.go", "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")
	mkFile(t, dir, "lib.go", "package main\n\nfunc helper() {}\n")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go"},
			{Name: "lib.go", Kind: "file", Path: "lib.go"},
		},
	}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	found := false
	for _, ep := range result.EntryPoints {
		if ep.Path == "main.go" && ep.Kind == "main" && ep.Line == 3 {
			found = true
			if !strings.Contains(ep.Signature, "func main()") {
				t.Errorf("expected signature to contain 'func main()', got %q", ep.Signature)
			}
		}
	}
	if !found {
		t.Error("expected to find func main() in main.go at line 3")
	}
}

func TestEntryPointsGoInit(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "setup.go", "package mypackage\n\nfunc init() {\n\t// setup\n}\n")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "setup.go", Kind: "file", Path: "setup.go"},
		},
	}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	found := false
	for _, ep := range result.EntryPoints {
		if ep.Kind == "init" && ep.Path == "setup.go" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find func init() in setup.go")
	}
}

func TestEntryPointsHTTPRoutes(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "routes.go", `package main

import "net/http"

func setupRoutes() {
	http.HandleFunc("/api/auth", handleAuth)
	http.HandleFunc("/api/users", handleUsers)
}
`)

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "routes.go", Kind: "file", Path: "routes.go"},
		},
	}

	result, err := idx.EntryPoints("route", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	if len(result.EntryPoints) != 2 {
		t.Fatalf("expected 2 route entries, got %d", len(result.EntryPoints))
	}
	for _, ep := range result.EntryPoints {
		if ep.Kind != "route" {
			t.Errorf("expected kind 'route', got %q", ep.Kind)
		}
	}
}

func TestEntryPointsPythonMain(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "app.py", `import sys

def main():
    print("hello")

if __name__ == "__main__":
    main()
`)

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "app.py", Kind: "file", Path: "app.py"},
		},
	}

	result, err := idx.EntryPoints("main", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	if len(result.EntryPoints) == 0 {
		t.Fatal("expected to find Python if __name__ entry point")
	}
	if result.EntryPoints[0].Kind != "main" {
		t.Errorf("expected kind 'main', got %q", result.EntryPoints[0].Kind)
	}
}

func TestEntryPointsKindFilter(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "main.go", "package main\n\nfunc main() {}\n\nfunc init() {}\n")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go"},
		},
	}

	// Filter by "main" — should not include init
	result, err := idx.EntryPoints("main", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}
	for _, ep := range result.EntryPoints {
		if ep.Kind != "main" {
			t.Errorf("expected only 'main' kind with filter, got %q", ep.Kind)
		}
	}

	// Filter by "init" — should not include main
	result, err = idx.EntryPoints("init", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}
	for _, ep := range result.EntryPoints {
		if ep.Kind != "init" {
			t.Errorf("expected only 'init' kind with filter, got %q", ep.Kind)
		}
	}
	if len(result.EntryPoints) == 0 {
		t.Error("expected at least 1 init entry point")
	}
}

func TestEntryPointsMaxLimit(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "routes.go", `package main

func setup() {
	http.HandleFunc("/a", a)
	http.HandleFunc("/b", b)
	http.HandleFunc("/c", c)
	http.HandleFunc("/d", d)
	http.HandleFunc("/e", e)
}
`)

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "routes.go", Kind: "file", Path: "routes.go"},
		},
	}

	result, err := idx.EntryPoints("", 3)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	if len(result.EntryPoints) != 3 {
		t.Errorf("expected 3 entries (limited by max), got %d", len(result.EntryPoints))
	}
	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
}

func TestEntryPointsTestFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	mkFile(t, dir, "main_test.go", "package main\n\nfunc init() {}\n")
	mkFile(t, dir, "app.test.ts", "app.get('/test', handler)\n")
	mkFile(t, dir, "app.spec.js", "app.post('/test', handler)\n")
	mkFile(t, dir, "test_app.py", "if __name__ == '__main__':\n    pass\n")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go"},
			{Name: "main_test.go", Kind: "file", Path: "main_test.go"},
			{Name: "app.test.ts", Kind: "file", Path: "app.test.ts"},
			{Name: "app.spec.js", Kind: "file", Path: "app.spec.js"},
			{Name: "test_app.py", Kind: "file", Path: "test_app.py"},
		},
	}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	for _, ep := range result.EntryPoints {
		if ep.Path != "main.go" {
			t.Errorf("expected only main.go entries, got entry from %s", ep.Path)
		}
	}
}

func TestEntryPointsJSRoutes(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "server.js", `const express = require('express');
const app = express();

app.get('/api/users', getUsers);
app.post('/api/users', createUser);
app.use('/api/auth', authRouter);
`)

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "server.js", Kind: "file", Path: "server.js"},
		},
	}

	result, err := idx.EntryPoints("route", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	if len(result.EntryPoints) != 3 {
		t.Errorf("expected 3 route entries, got %d", len(result.EntryPoints))
	}
}

func TestEntryPointsJSONStructure(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go"},
		},
	}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if _, ok := parsed["entryPoints"]; !ok {
		t.Error("JSON missing 'entryPoints' field")
	}
	if _, ok := parsed["total"]; !ok {
		t.Error("JSON missing 'total' field")
	}
}

func TestEntryPointsEmptyIndex(t *testing.T) {
	dir := t.TempDir()
	idx := &Index{Root: dir, Entries: []Entry{}}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}
	if len(result.EntryPoints) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.EntryPoints))
	}
	if result.Total != 0 {
		t.Errorf("expected total 0, got %d", result.Total)
	}
}

func TestEntryPointsSortOrder(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "main.go", "package main\n\nfunc init() {}\n\nfunc main() {}\n")
	mkFile(t, dir, "routes.go", "package main\n\nfunc setup() {\n\thttp.HandleFunc(\"/a\", a)\n}\n")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go"},
			{Name: "routes.go", Kind: "file", Path: "routes.go"},
		},
	}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	if len(result.EntryPoints) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(result.EntryPoints))
	}

	// main should come before route, which should come before init
	var kinds []string
	for _, ep := range result.EntryPoints {
		kinds = append(kinds, ep.Kind)
	}

	mainIdx := -1
	routeIdx := -1
	initIdx := -1
	for i, k := range kinds {
		if k == "main" && mainIdx == -1 {
			mainIdx = i
		}
		if k == "route" && routeIdx == -1 {
			routeIdx = i
		}
		if k == "init" && initIdx == -1 {
			initIdx = i
		}
	}

	if mainIdx > routeIdx {
		t.Error("expected main entries before route entries")
	}
	if routeIdx > initIdx {
		t.Error("expected route entries before init entries")
	}
}

func TestFormatEntryPointsEmpty(t *testing.T) {
	result := &EntryPointsResult{
		EntryPoints: []EntryPoint{},
		Total:       0,
	}
	out := FormatEntryPoints(result)
	if !strings.Contains(out, "No entry points found") {
		t.Errorf("expected 'No entry points found', got: %s", out)
	}
}

func TestFormatEntryPointsWithEntries(t *testing.T) {
	result := &EntryPointsResult{
		EntryPoints: []EntryPoint{
			{Path: "main.go", Line: 5, Kind: "main", Signature: "func main()"},
			{Path: "routes.go", Line: 10, Kind: "route", Signature: `http.HandleFunc("/api/auth", handleAuth)`},
			{Path: "setup.go", Line: 3, Kind: "init", Signature: "func init()"},
		},
		Total: 3,
	}
	out := FormatEntryPoints(result)

	checks := []string{
		"Main entry points",
		"func main()",
		"Route handlers",
		"http.HandleFunc",
		"Init functions",
		"func init()",
		"3 entry points found",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q:\n%s", check, out)
		}
	}
}

func TestEntryPointsBinaryFileSkipped(t *testing.T) {
	dir := t.TempDir()
	// Create a file with null bytes (binary)
	binContent := []byte("func main() {\x00\x00binary\x00}")
	if err := os.WriteFile(filepath.Join(dir, "binary.go"), binContent, 0644); err != nil {
		t.Fatal(err)
	}

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "binary.go", Kind: "file", Path: "binary.go"},
		},
	}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}
	if len(result.EntryPoints) != 0 {
		t.Errorf("expected 0 entries for binary file, got %d", len(result.EntryPoints))
	}
}

func TestEntryPointsPythonFlask(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "app.py", `from flask import Flask

app = Flask(__name__)

@app.route("/")
def index():
    return "hello"

@app.get("/users")
def users():
    return []
`)

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "app.py", Kind: "file", Path: "app.py"},
		},
	}

	result, err := idx.EntryPoints("", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	routeCount := 0
	initCount := 0
	for _, ep := range result.EntryPoints {
		switch ep.Kind {
		case "route":
			routeCount++
		case "init":
			initCount++
		}
	}
	if routeCount != 2 {
		t.Errorf("expected 2 route entries, got %d", routeCount)
	}
	if initCount != 1 {
		t.Errorf("expected 1 init entry (Flask()), got %d", initCount)
	}
}

func TestEntryPointsCobraCommands(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, dir, "cmd.go", `package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use: "myapp",
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
`)

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "cmd.go", Kind: "file", Path: "cmd.go"},
		},
	}

	result, err := idx.EntryPoints("cli", 0)
	if err != nil {
		t.Fatalf("EntryPoints() error: %v", err)
	}

	if len(result.EntryPoints) != 2 {
		t.Errorf("expected 2 cli entries (cobra.Command + AddCommand), got %d", len(result.EntryPoints))
	}
}
