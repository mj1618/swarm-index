package index

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo creates a temporary git repository with an initial commit.
// It returns the repo directory and a helper function for running git commands.
func initGitRepo(t *testing.T) (string, func(args ...string)) {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test Author",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test Author",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test Author")

	// Create an initial file and commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "Initial commit")

	return dir, run
}

func TestHistoryWithCommits(t *testing.T) {
	dir, run := initGitRepo(t)

	// Create a file and make multiple commits
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "Add main.go")

	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "Add main function")

	result, err := History(dir, "main.go", 10)
	if err != nil {
		t.Fatalf("History() error: %v", err)
	}

	if result.Path != "main.go" {
		t.Errorf("Path = %q, want %q", result.Path, "main.go")
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if len(result.Commits) != 2 {
		t.Fatalf("len(Commits) = %d, want 2", len(result.Commits))
	}

	// Most recent commit first
	if result.Commits[0].Subject != "Add main function" {
		t.Errorf("Commits[0].Subject = %q, want %q", result.Commits[0].Subject, "Add main function")
	}
	if result.Commits[1].Subject != "Add main.go" {
		t.Errorf("Commits[1].Subject = %q, want %q", result.Commits[1].Subject, "Add main.go")
	}

	// Verify fields are populated
	for i, c := range result.Commits {
		if c.Hash == "" {
			t.Errorf("Commits[%d].Hash is empty", i)
		}
		if c.Author != "Test Author" {
			t.Errorf("Commits[%d].Author = %q, want %q", i, c.Author, "Test Author")
		}
		if c.Date == "" {
			t.Errorf("Commits[%d].Date is empty", i)
		}
	}
}

func TestHistoryNonexistentFile(t *testing.T) {
	dir, _ := initGitRepo(t)

	result, err := History(dir, "nonexistent.go", 10)
	if err != nil {
		t.Fatalf("History() error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0 for nonexistent file", result.Total)
	}
	if len(result.Commits) != 0 {
		t.Errorf("len(Commits) = %d, want 0", len(result.Commits))
	}
}

func TestHistoryMaxLimit(t *testing.T) {
	dir, run := initGitRepo(t)

	filePath := filepath.Join(dir, "main.go")
	for i := 0; i < 5; i++ {
		content := strings.Repeat("x", i+1) + "\n"
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		run("add", "main.go")
		run("commit", "-m", "Commit "+strings.Repeat("x", i+1))
	}

	result, err := History(dir, "main.go", 3)
	if err != nil {
		t.Fatalf("History() error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3 (limited by max)", result.Total)
	}
	if len(result.Commits) != 3 {
		t.Errorf("len(Commits) = %d, want 3", len(result.Commits))
	}
}

func TestHistoryNotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := History(dir, "main.go", 10)
	if err == nil {
		t.Fatal("History() should fail outside a git repo")
	}
	if !strings.Contains(err.Error(), "git log failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFormatHistoryNoCommits(t *testing.T) {
	result := &HistoryResult{
		Path:    "main.go",
		Commits: []HistoryCommit{},
		Total:   0,
	}
	out := FormatHistory(result)
	if !strings.Contains(out, "No commits found for main.go") {
		t.Errorf("output missing 'No commits found': %s", out)
	}
}

func TestFormatHistoryWithCommits(t *testing.T) {
	result := &HistoryResult{
		Path: "main.go",
		Commits: []HistoryCommit{
			{Hash: "abc1234", Author: "John Doe", Date: "2025-01-15T10:30:00-05:00", Subject: "Add error handling"},
			{Hash: "def5678", Author: "Jane Smith", Date: "2025-01-14T09:00:00-05:00", Subject: "Initial implementation"},
		},
		Total: 2,
	}
	out := FormatHistory(result)

	checks := []string{
		"History for main.go (2 commits):",
		"abc1234",
		"2025-01-15",
		"John Doe",
		"Add error handling",
		"def5678",
		"2025-01-14",
		"Jane Smith",
		"Initial implementation",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q:\n%s", check, out)
		}
	}
}
