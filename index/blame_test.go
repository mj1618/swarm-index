package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlameWithCommits(t *testing.T) {
	dir, run := initGitRepo(t)

	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "Add main.go")

	result, err := Blame(dir, "main.go", 0, 0)
	if err != nil {
		t.Fatalf("Blame() error: %v", err)
	}

	if result.File != "main.go" {
		t.Errorf("File = %q, want %q", result.File, "main.go")
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if len(result.Lines) != 3 {
		t.Fatalf("len(Lines) = %d, want 3", len(result.Lines))
	}

	// Verify fields are populated
	for i, l := range result.Lines {
		if l.Hash == "" {
			t.Errorf("Lines[%d].Hash is empty", i)
		}
		if l.Author != "Test Author" {
			t.Errorf("Lines[%d].Author = %q, want %q", i, l.Author, "Test Author")
		}
		if l.Date == "" {
			t.Errorf("Lines[%d].Date is empty", i)
		}
		if l.Line != i+1 {
			t.Errorf("Lines[%d].Line = %d, want %d", i, l.Line, i+1)
		}
	}

	// Verify content
	if result.Lines[0].Content != "package main" {
		t.Errorf("Lines[0].Content = %q, want %q", result.Lines[0].Content, "package main")
	}
	if result.Lines[2].Content != "func main() {}" {
		t.Errorf("Lines[2].Content = %q, want %q", result.Lines[2].Content, "func main() {}")
	}
}

func TestBlameLineRange(t *testing.T) {
	dir, run := initGitRepo(t)

	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3\nline4\nline5\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "Add main.go")

	result, err := Blame(dir, "main.go", 2, 4)
	if err != nil {
		t.Fatalf("Blame() error: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if len(result.Lines) != 3 {
		t.Fatalf("len(Lines) = %d, want 3", len(result.Lines))
	}
	if result.Lines[0].Line != 2 {
		t.Errorf("Lines[0].Line = %d, want 2", result.Lines[0].Line)
	}
	if result.Lines[2].Line != 4 {
		t.Errorf("Lines[2].Line = %d, want 4", result.Lines[2].Line)
	}
}

func TestBlameMultipleAuthors(t *testing.T) {
	dir, run := initGitRepo(t)

	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "First commit")

	// Modify with different author
	if err := os.WriteFile(filePath, []byte("line1\nline2 modified\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "Second commit")

	result, err := Blame(dir, "main.go", 0, 0)
	if err != nil {
		t.Fatalf("Blame() error: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}

	// Line 2 should have a different hash than line 1 (modified in second commit)
	if result.Lines[0].Hash == result.Lines[1].Hash {
		t.Errorf("expected different hashes for line 1 and line 2, got same: %s", result.Lines[0].Hash)
	}
}

func TestBlameNotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := Blame(dir, "main.go", 0, 0)
	if err == nil {
		t.Fatal("Blame() should fail outside a git repo")
	}
	if !strings.Contains(err.Error(), "git blame failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBlameNonexistentFile(t *testing.T) {
	dir, _ := initGitRepo(t)
	_, err := Blame(dir, "nonexistent.go", 0, 0)
	if err == nil {
		t.Fatal("Blame() should fail for nonexistent file")
	}
}

func TestParsePorcelain(t *testing.T) {
	input := `a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2 1 1 2
author Alice
author-mail <alice@example.com>
author-time 1710460800
author-tz +0000
committer Alice
committer-mail <alice@example.com>
committer-time 1710460800
committer-tz +0000
summary Add main
filename main.go
	func main() {
a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2 1 2
	    fmt.Println("hello")
`

	lines := parsePorcelain(input)
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}

	if lines[0].Hash != "a1b2c3d4" {
		t.Errorf("lines[0].Hash = %q, want %q", lines[0].Hash, "a1b2c3d4")
	}
	if lines[0].Author != "Alice" {
		t.Errorf("lines[0].Author = %q, want %q", lines[0].Author, "Alice")
	}
	if lines[0].Date != "2024-03-15" {
		t.Errorf("lines[0].Date = %q, want %q", lines[0].Date, "2024-03-15")
	}
	if lines[0].Line != 1 {
		t.Errorf("lines[0].Line = %d, want 1", lines[0].Line)
	}
	if lines[0].Content != "func main() {" {
		t.Errorf("lines[0].Content = %q, want %q", lines[0].Content, "func main() {")
	}

	if lines[1].Line != 2 {
		t.Errorf("lines[1].Line = %d, want 2", lines[1].Line)
	}
	if lines[1].Content != "    fmt.Println(\"hello\")" {
		t.Errorf("lines[1].Content = %q, want %q", lines[1].Content, "    fmt.Println(\"hello\")")
	}
}

func TestFormatBlameEmpty(t *testing.T) {
	result := &BlameResult{
		File:  "main.go",
		Lines: []BlameLine{},
		Total: 0,
	}
	out := FormatBlame(result)
	if !strings.Contains(out, "No blame info for main.go") {
		t.Errorf("output missing 'No blame info': %s", out)
	}
}

func TestFormatBlameWithLines(t *testing.T) {
	result := &BlameResult{
		File: "main.go",
		Lines: []BlameLine{
			{Line: 10, Hash: "a1b2c3d4", Author: "Alice", Date: "2024-03-15", Content: "func main() {"},
			{Line: 11, Hash: "f5e6d7c8", Author: "Bob VeryLongName", Date: "2024-04-01", Content: "    fmt.Println(\"hello\")"},
		},
		Total: 2,
	}
	out := FormatBlame(result)

	checks := []string{
		"main.go:",
		"10",
		"a1b2c3d4",
		"2024-03-15",
		"Alice",
		"func main() {",
		"11",
		"f5e6d7c8",
		"2024-04-01",
		"Bob VeryLong", // truncated to 12 chars
		"fmt.Println",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q:\n%s", check, out)
		}
	}

	// Verify long author name is truncated
	if strings.Contains(out, "Bob VeryLongName") {
		t.Errorf("author name should be truncated to 12 chars:\n%s", out)
	}
}
