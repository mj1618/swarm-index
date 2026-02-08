package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHotspotsBasic(t *testing.T) {
	dir, run := initGitRepo(t)

	// Create files with different commit frequencies
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(dir, "hot.go"), []byte(strings.Repeat("x", i+1)+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		run("add", "hot.go")
		run("commit", "-m", "update hot.go")
	}
	for i := 0; i < 2; i++ {
		if err := os.WriteFile(filepath.Join(dir, "cold.go"), []byte(strings.Repeat("y", i+1)+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		run("add", "cold.go")
		run("commit", "-m", "update cold.go")
	}

	// Build an index with these files
	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "hot.go", Kind: "file", Path: "hot.go"},
			{Name: "cold.go", Kind: "file", Path: "cold.go"},
			{Name: "README.md", Kind: "file", Path: "README.md"},
		},
	}

	result, err := idx.Hotspots(dir, 20, "", "")
	if err != nil {
		t.Fatalf("Hotspots() error: %v", err)
	}

	if len(result.Entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(result.Entries))
	}

	// hot.go should be first (most commits)
	if result.Entries[0].Path != "hot.go" {
		t.Errorf("expected hot.go first, got %s", result.Entries[0].Path)
	}
	if result.Entries[0].CommitCount < 5 {
		t.Errorf("expected hot.go to have at least 5 commits, got %d", result.Entries[0].CommitCount)
	}
}

func TestHotspotsMaxLimit(t *testing.T) {
	dir, run := initGitRepo(t)

	// Create 3 files with commits
	files := []string{"a.go", "b.go", "c.go"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
		run("add", f)
		run("commit", "-m", "add "+f)
	}

	entries := make([]Entry, len(files))
	for i, f := range files {
		entries[i] = Entry{Name: f, Kind: "file", Path: f}
	}
	// Include README.md from initGitRepo
	entries = append(entries, Entry{Name: "README.md", Kind: "file", Path: "README.md"})

	idx := &Index{Root: dir, Entries: entries}

	result, err := idx.Hotspots(dir, 2, "", "")
	if err != nil {
		t.Fatalf("Hotspots() error: %v", err)
	}

	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries (limited by max), got %d", len(result.Entries))
	}
	if result.Total < 3 {
		t.Errorf("expected total >= 3, got %d", result.Total)
	}
}

func TestHotspotsPathFilter(t *testing.T) {
	dir, run := initGitRepo(t)

	// Create files in different directories
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "app.go"), []byte("package src\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "add files")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "app.go", Kind: "file", Path: "src/app.go"},
			{Name: "root.go", Kind: "file", Path: "root.go"},
			{Name: "README.md", Kind: "file", Path: "README.md"},
		},
	}

	result, err := idx.Hotspots(dir, 20, "", "src/")
	if err != nil {
		t.Fatalf("Hotspots() error: %v", err)
	}

	for _, e := range result.Entries {
		if !strings.HasPrefix(e.Path, "src/") {
			t.Errorf("expected all entries to have prefix src/, got %s", e.Path)
		}
	}
	if len(result.Entries) != 1 {
		t.Errorf("expected 1 entry with src/ prefix, got %d", len(result.Entries))
	}
}

func TestHotspotsDeletedFilesExcluded(t *testing.T) {
	dir, run := initGitRepo(t)

	// Create a file, commit, then delete it
	if err := os.WriteFile(filepath.Join(dir, "deleted.go"), []byte("gone\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "deleted.go")
	run("commit", "-m", "add deleted.go")

	// Remove the file from disk and git
	os.Remove(filepath.Join(dir, "deleted.go"))
	run("add", "deleted.go")
	run("commit", "-m", "remove deleted.go")

	// Index does NOT contain deleted.go (simulating it was removed)
	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "README.md", Kind: "file", Path: "README.md"},
		},
	}

	result, err := idx.Hotspots(dir, 20, "", "")
	if err != nil {
		t.Fatalf("Hotspots() error: %v", err)
	}

	for _, e := range result.Entries {
		if e.Path == "deleted.go" {
			t.Error("deleted.go should not appear in hotspots")
		}
	}
}

func TestHotspotsSinceFilter(t *testing.T) {
	dir, run := initGitRepo(t)

	// Create a file with a commit
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "add main.go")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go"},
			{Name: "README.md", Kind: "file", Path: "README.md"},
		},
	}

	// Use a very recent "since" — all commits should be included
	result, err := idx.Hotspots(dir, 20, "1 hour ago", "")
	if err != nil {
		t.Fatalf("Hotspots() error: %v", err)
	}
	if result.Since != "1 hour ago" {
		t.Errorf("Since = %q, want %q", result.Since, "1 hour ago")
	}
	if len(result.Entries) == 0 {
		t.Error("expected entries with recent since filter")
	}

	// Use a future "since" — no commits should match
	result, err = idx.Hotspots(dir, 20, "2099-01-01", "")
	if err != nil {
		t.Fatalf("Hotspots() error: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries with future since filter, got %d", len(result.Entries))
	}
}

func TestHotspotsJSONStructure(t *testing.T) {
	dir, run := initGitRepo(t)

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "main.go")
	run("commit", "-m", "add main.go")

	idx := &Index{
		Root: dir,
		Entries: []Entry{
			{Name: "main.go", Kind: "file", Path: "main.go"},
			{Name: "README.md", Kind: "file", Path: "README.md"},
		},
	}

	result, err := idx.Hotspots(dir, 20, "", "")
	if err != nil {
		t.Fatalf("Hotspots() error: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// Check required fields exist
	if _, ok := parsed["entries"]; !ok {
		t.Error("JSON missing 'entries' field")
	}
	if _, ok := parsed["total"]; !ok {
		t.Error("JSON missing 'total' field")
	}
}

func TestHotspotsNotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	idx := &Index{Root: dir, Entries: []Entry{}}
	_, err := idx.Hotspots(dir, 20, "", "")
	if err == nil {
		t.Fatal("Hotspots() should fail outside a git repo")
	}
	if !strings.Contains(err.Error(), "git log failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFormatHotspotsEmpty(t *testing.T) {
	result := &HotspotsResult{
		Entries: []HotspotEntry{},
		Total:   0,
	}
	out := FormatHotspots(result)
	if !strings.Contains(out, "No hotspots found") {
		t.Errorf("expected 'No hotspots found', got: %s", out)
	}
}

func TestFormatHotspotsWithEntries(t *testing.T) {
	result := &HotspotsResult{
		Entries: []HotspotEntry{
			{Path: "main.go", CommitCount: 52, LastModified: "2025-01-15T10:30:00-05:00"},
			{Path: "index/index.go", CommitCount: 47, LastModified: "2025-01-14T09:00:00-05:00"},
		},
		Total: 145,
	}
	out := FormatHotspots(result)

	checks := []string{
		"Hotspots",
		"52 commits",
		"main.go",
		"2025-01-15",
		"47 commits",
		"index/index.go",
		"2025-01-14",
		"2 of 145 files shown",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q:\n%s", check, out)
		}
	}
}

func TestFormatHotspotsWithSince(t *testing.T) {
	result := &HotspotsResult{
		Entries: []HotspotEntry{
			{Path: "main.go", CommitCount: 10, LastModified: "2025-01-15T10:30:00-05:00"},
		},
		Total: 1,
		Since: "6 months ago",
	}
	out := FormatHotspots(result)
	if !strings.Contains(out, "since 6 months ago") {
		t.Errorf("output missing since info:\n%s", out)
	}
}
