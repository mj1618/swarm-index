package index

import (
	"strings"
	"testing"

	_ "github.com/mj1618/swarm-index/parsers" // register parsers
)

func TestSymbolsBasicSearch(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func HandleAuth() {}
func HandleLogin() {}
func helper() {}
`)
	mkFile(t, tmp, "lib/config.go", `package lib

type Config struct{}
func NewConfig() *Config { return nil }
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Symbols("config", "", 50)
	if err != nil {
		t.Fatalf("Symbols() error: %v", err)
	}

	if result.Total < 2 {
		t.Errorf("Total = %d, want >= 2 (Config struct + NewConfig func)", result.Total)
	}

	names := map[string]bool{}
	for _, m := range result.Matches {
		names[m.Name] = true
	}
	if !names["Config"] {
		t.Error("missing symbol 'Config'")
	}
	if !names["NewConfig"] {
		t.Error("missing symbol 'NewConfig'")
	}
}

func TestSymbolsKindFilter(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func HandleAuth() {}
type AuthConfig struct{}
var AuthEnabled bool
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Symbols("auth", "func", 50)
	if err != nil {
		t.Fatalf("Symbols() error: %v", err)
	}

	for _, m := range result.Matches {
		if m.Kind != "func" {
			t.Errorf("got kind %q, want only 'func' results", m.Kind)
		}
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1 (HandleAuth only)", result.Total)
	}
}

func TestSymbolsCaseInsensitive(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func HandleAuth() {}
func handleauth() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Symbols("HANDLEAUTH", "", 50)
	if err != nil {
		t.Fatalf("Symbols() error: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Total = %d, want 2 (case-insensitive match)", result.Total)
	}
}

func TestSymbolsResultOrdering(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func Config() {}
func ConfigManager() {}
func GetConfig() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Symbols("config", "", 50)
	if err != nil {
		t.Fatalf("Symbols() error: %v", err)
	}

	if len(result.Matches) < 3 {
		t.Fatalf("got %d matches, want >= 3", len(result.Matches))
	}

	// First result should be exact match (Config)
	if strings.ToLower(result.Matches[0].Name) != "config" {
		t.Errorf("first result = %q, want exact match 'Config'", result.Matches[0].Name)
	}

	// Second should be prefix match (ConfigManager)
	if !strings.HasPrefix(strings.ToLower(result.Matches[1].Name), "config") {
		t.Errorf("second result = %q, want prefix match", result.Matches[1].Name)
	}
}

func TestSymbolsMaxLimit(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func FuncA() {}
func FuncB() {}
func FuncC() {}
func FuncD() {}
func FuncE() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Symbols("func", "", 2)
	if err != nil {
		t.Fatalf("Symbols() error: %v", err)
	}

	if len(result.Matches) != 2 {
		t.Errorf("got %d matches, want 2 (max limit)", len(result.Matches))
	}
	if result.Total < 5 {
		t.Errorf("Total = %d, want >= 5 (all matches counted)", result.Total)
	}
}

func TestSymbolsNoMatches(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func Hello() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Symbols("zzzznotfound", "", 50)
	if err != nil {
		t.Fatalf("Symbols() error: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if len(result.Matches) != 0 {
		t.Errorf("Matches has %d entries, want 0", len(result.Matches))
	}
}

func TestSymbolsMultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main
func HandleAuth() {}
`)
	mkFile(t, tmp, "b.go", `package main
func AuthMiddleware() {}
`)
	mkFile(t, tmp, "c.py", `
def authenticate():
    pass
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Symbols("auth", "", 50)
	if err != nil {
		t.Fatalf("Symbols() error: %v", err)
	}

	if result.Total < 3 {
		t.Errorf("Total = %d, want >= 3 (HandleAuth + AuthMiddleware + authenticate)", result.Total)
	}

	paths := map[string]bool{}
	for _, m := range result.Matches {
		paths[m.Path] = true
	}
	if !paths["a.go"] || !paths["b.go"] || !paths["c.py"] {
		t.Errorf("expected matches from a.go, b.go, and c.py; got paths: %v", paths)
	}
}

func TestFormatSymbolsEmpty(t *testing.T) {
	result := &SymbolsResult{
		Query:   "xyz",
		Matches: []SymbolMatch{},
		Total:   0,
	}
	out := FormatSymbols(result)
	if !strings.Contains(out, "No symbols matching") {
		t.Errorf("output missing 'No symbols matching': %s", out)
	}
	if !strings.Contains(out, "xyz") {
		t.Errorf("output missing query: %s", out)
	}
}

func TestFormatSymbolsWithResults(t *testing.T) {
	result := &SymbolsResult{
		Query: "auth",
		Matches: []SymbolMatch{
			{Name: "HandleAuth", Kind: "func", Path: "main.go", Line: 5, Signature: "func HandleAuth()"},
			{Name: "AuthConfig", Kind: "type", Path: "config.go", Line: 10, Signature: "type AuthConfig struct"},
		},
		Total: 2,
	}
	out := FormatSymbols(result)
	if !strings.Contains(out, "2 found") {
		t.Errorf("output missing count: %s", out)
	}
	if !strings.Contains(out, "HandleAuth") {
		t.Errorf("output missing 'HandleAuth': %s", out)
	}
	if !strings.Contains(out, "main.go:5") {
		t.Errorf("output missing 'main.go:5': %s", out)
	}
}

func TestFormatSymbolsTruncated(t *testing.T) {
	result := &SymbolsResult{
		Query: "func",
		Matches: []SymbolMatch{
			{Name: "FuncA", Kind: "func", Path: "a.go", Line: 1},
		},
		Total: 10,
	}
	out := FormatSymbols(result)
	if !strings.Contains(out, "9 more") {
		t.Errorf("output missing truncation notice: %s", out)
	}
}
