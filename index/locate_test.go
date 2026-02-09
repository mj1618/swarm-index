package index

import (
	"strings"
	"testing"

	_ "github.com/mj1618/swarm-index/parsers" // register parsers
)

func TestLocateFileMatches(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "config.go", `package main

func main() {}
`)
	mkFile(t, tmp, "lib/config_test.go", `package lib
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Locate("config", 20)
	if err != nil {
		t.Fatalf("Locate() error: %v", err)
	}

	if result.Total == 0 {
		t.Fatal("expected at least one match")
	}

	hasFile := false
	for _, m := range result.Matches {
		if m.Category == "file" {
			hasFile = true
			break
		}
	}
	if !hasFile {
		t.Error("expected at least one file category match")
	}
}

func TestLocateSymbolMatches(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func HandleAuth() {}
func HandleLogin() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Locate("HandleAuth", 20)
	if err != nil {
		t.Fatalf("Locate() error: %v", err)
	}

	hasSymbol := false
	for _, m := range result.Matches {
		if m.Category == "symbol" && m.Name == "HandleAuth" {
			hasSymbol = true
			break
		}
	}
	if !hasSymbol {
		t.Error("expected a symbol match for HandleAuth")
	}
}

func TestLocateContentMatches(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

// This is a special marker: XYZZY123
func main() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Locate("XYZZY123", 20)
	if err != nil {
		t.Fatalf("Locate() error: %v", err)
	}

	hasContent := false
	for _, m := range result.Matches {
		if m.Category == "content" {
			hasContent = true
			break
		}
	}
	if !hasContent {
		t.Error("expected a content match for XYZZY123")
	}
}

func TestLocateRelevanceRanking(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "config.go", `package main

func Config() {}
func GetConfig() {}
`)
	mkFile(t, tmp, "lib/myconfig.go", `package lib

func MyConfigHelper() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Locate("config", 20)
	if err != nil {
		t.Fatalf("Locate() error: %v", err)
	}

	if len(result.Matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(result.Matches))
	}

	// First match should have a higher score than the last.
	if result.Matches[0].Score <= result.Matches[len(result.Matches)-1].Score {
		t.Errorf("first match score (%d) should be higher than last (%d)",
			result.Matches[0].Score, result.Matches[len(result.Matches)-1].Score)
	}
}

func TestLocateDeduplication(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "config.go", `package main

func Config() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Locate("config", 20)
	if err != nil {
		t.Fatalf("Locate() error: %v", err)
	}

	// Count how many times config.go appears with line 0 (file matches).
	fileMatches := 0
	for _, m := range result.Matches {
		if m.Path == "config.go" && m.Category == "file" {
			fileMatches++
		}
	}
	if fileMatches > 1 {
		t.Errorf("config.go file match appears %d times, expected at most 1", fileMatches)
	}
}

func TestLocateMaxLimit(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main

func FuncA() {}
func FuncB() {}
func FuncC() {}
func FuncD() {}
func FuncE() {}
`)
	mkFile(t, tmp, "b.go", `package main

func FuncF() {}
func FuncG() {}
func FuncH() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Locate("Func", 3)
	if err != nil {
		t.Fatalf("Locate() error: %v", err)
	}

	if len(result.Matches) > 3 {
		t.Errorf("got %d matches, want at most 3 (max limit)", len(result.Matches))
	}
	if result.Total <= 3 {
		t.Logf("Total = %d (may be <= max if few matches)", result.Total)
	}
}

func TestLocateEmptyQuery(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main
func main() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	_, err = idx.Locate("", 20)
	if err == nil {
		t.Error("expected error for empty query, got nil")
	}

	_, err = idx.Locate("   ", 20)
	if err == nil {
		t.Error("expected error for whitespace query, got nil")
	}
}

func TestFormatLocateEmpty(t *testing.T) {
	result := &LocateResult{
		Query:   "notfound",
		Matches: []LocateMatch{},
		Total:   0,
	}
	out := FormatLocate(result)
	if !strings.Contains(out, "No matches") {
		t.Errorf("expected 'No matches' in output, got: %s", out)
	}
}

func TestFormatLocateWithResults(t *testing.T) {
	result := &LocateResult{
		Query: "config",
		Matches: []LocateMatch{
			{Category: "file", Path: "config.go", Name: "config.go", Score: 100},
			{Category: "symbol", Path: "config.go", Name: "Config", Line: 5, Kind: "func", Score: 90},
			{Category: "content", Path: "main.go", Name: "main.go", Line: 10, Content: "cfg := Config()", Score: 50},
		},
		Total: 3,
	}
	out := FormatLocate(result)
	if !strings.Contains(out, "Files:") {
		t.Error("output missing 'Files:' section")
	}
	if !strings.Contains(out, "Symbols:") {
		t.Error("output missing 'Symbols:' section")
	}
	if !strings.Contains(out, "Content:") {
		t.Error("output missing 'Content:' section")
	}
	if !strings.Contains(out, "3 total matches") {
		t.Errorf("output missing total count: %s", out)
	}
}
