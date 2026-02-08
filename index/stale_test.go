package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStaleFreshIndex(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "lib/util.go", "package lib")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	result, err := loaded.Stale()
	if err != nil {
		t.Fatalf("Stale() error: %v", err)
	}

	if result.IsStale {
		t.Errorf("IsStale = true, want false for fresh index")
	}
	if len(result.NewFiles) != 0 {
		t.Errorf("NewFiles = %v, want empty", result.NewFiles)
	}
	if len(result.DeletedFiles) != 0 {
		t.Errorf("DeletedFiles = %v, want empty", result.DeletedFiles)
	}
	if len(result.ModifiedFiles) != 0 {
		t.Errorf("ModifiedFiles = %v, want empty", result.ModifiedFiles)
	}
}

func TestStaleNewFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Create a new file after saving the index
	time.Sleep(10 * time.Millisecond)
	mkFile(t, tmp, "new_file.go", "package main")

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	result, err := loaded.Stale()
	if err != nil {
		t.Fatalf("Stale() error: %v", err)
	}

	if !result.IsStale {
		t.Errorf("IsStale = false, want true")
	}
	if len(result.NewFiles) != 1 {
		t.Fatalf("NewFiles has %d entries, want 1", len(result.NewFiles))
	}
	if result.NewFiles[0] != "new_file.go" {
		t.Errorf("NewFiles[0] = %q, want %q", result.NewFiles[0], "new_file.go")
	}
	if result.Summary.New != 1 {
		t.Errorf("Summary.New = %d, want 1", result.Summary.New)
	}
}

func TestStaleDeletedFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "to_delete.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Delete the file
	if err := os.Remove(filepath.Join(tmp, "to_delete.go")); err != nil {
		t.Fatalf("removing file: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	result, err := loaded.Stale()
	if err != nil {
		t.Fatalf("Stale() error: %v", err)
	}

	if !result.IsStale {
		t.Errorf("IsStale = false, want true")
	}
	if len(result.DeletedFiles) != 1 {
		t.Fatalf("DeletedFiles has %d entries, want 1", len(result.DeletedFiles))
	}
	if result.DeletedFiles[0] != "to_delete.go" {
		t.Errorf("DeletedFiles[0] = %q, want %q", result.DeletedFiles[0], "to_delete.go")
	}
	if result.Summary.Deleted != 1 {
		t.Errorf("Summary.Deleted = %d, want 1", result.Summary.Deleted)
	}
}

func TestStaleModifiedFile(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Touch the file with a future mtime
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(tmp, "main.go"), future, future); err != nil {
		t.Fatalf("Chtimes() error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	result, err := loaded.Stale()
	if err != nil {
		t.Fatalf("Stale() error: %v", err)
	}

	if !result.IsStale {
		t.Errorf("IsStale = false, want true")
	}
	if len(result.ModifiedFiles) != 1 {
		t.Fatalf("ModifiedFiles has %d entries, want 1", len(result.ModifiedFiles))
	}
	if result.ModifiedFiles[0] != "main.go" {
		t.Errorf("ModifiedFiles[0] = %q, want %q", result.ModifiedFiles[0], "main.go")
	}
	if result.Summary.Modified != 1 {
		t.Errorf("Summary.Modified = %d, want 1", result.Summary.Modified)
	}
}

func TestStaleCombined(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "keep.go", "package main")
	mkFile(t, tmp, "modify.go", "package main")
	mkFile(t, tmp, "delete.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Create a new file
	time.Sleep(10 * time.Millisecond)
	mkFile(t, tmp, "new.go", "package main")

	// Modify a file (touch with future time)
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(tmp, "modify.go"), future, future); err != nil {
		t.Fatalf("Chtimes() error: %v", err)
	}

	// Delete a file
	if err := os.Remove(filepath.Join(tmp, "delete.go")); err != nil {
		t.Fatalf("removing file: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	result, err := loaded.Stale()
	if err != nil {
		t.Fatalf("Stale() error: %v", err)
	}

	if !result.IsStale {
		t.Errorf("IsStale = false, want true")
	}
	if result.Summary.New != 1 {
		t.Errorf("Summary.New = %d, want 1", result.Summary.New)
	}
	if result.Summary.Deleted != 1 {
		t.Errorf("Summary.Deleted = %d, want 1", result.Summary.Deleted)
	}
	if result.Summary.Modified != 1 {
		t.Errorf("Summary.Modified = %d, want 1", result.Summary.Modified)
	}
}

func TestFormatStaleUpToDate(t *testing.T) {
	result := &StaleResult{
		ScannedAt: "2025-06-01T12:00:00Z",
		IsStale:   false,
	}
	out := FormatStale(result)
	if out == "" {
		t.Fatal("FormatStale returned empty string")
	}
	if !strings.Contains(out,"up to date") {
		t.Errorf("output missing 'up to date': %s", out)
	}
}

func TestFormatStaleWithChanges(t *testing.T) {
	result := &StaleResult{
		ScannedAt:     "2025-06-01T12:00:00Z",
		IsStale:       true,
		NewFiles:      []string{"new.go"},
		DeletedFiles:  []string{"old.go"},
		ModifiedFiles: []string{"changed.go"},
		Summary:       StaleSummary{New: 1, Deleted: 1, Modified: 1},
	}
	out := FormatStale(result)
	if !strings.Contains(out,"STALE") {
		t.Errorf("output missing 'STALE': %s", out)
	}
	if !strings.Contains(out,"new.go") {
		t.Errorf("output missing new file: %s", out)
	}
	if !strings.Contains(out,"old.go") {
		t.Errorf("output missing deleted file: %s", out)
	}
	if !strings.Contains(out,"changed.go") {
		t.Errorf("output missing modified file: %s", out)
	}
}

