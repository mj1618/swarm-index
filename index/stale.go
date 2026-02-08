package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// StaleResult holds the result of comparing the index against the filesystem.
type StaleResult struct {
	ScannedAt     string       `json:"scannedAt"`
	IsStale       bool         `json:"isStale"`
	NewFiles      []string     `json:"newFiles"`
	DeletedFiles  []string     `json:"deletedFiles"`
	ModifiedFiles []string     `json:"modifiedFiles"`
	Summary       StaleSummary `json:"summary"`
}

// StaleSummary holds counts for the stale check.
type StaleSummary struct {
	New      int `json:"new"`
	Deleted  int `json:"deleted"`
	Modified int `json:"modified"`
}

// Stale compares the persisted index against the current filesystem and reports
// new, deleted, and modified files.
func (idx *Index) Stale() (*StaleResult, error) {
	scannedAt, err := time.Parse(time.RFC3339, idx.ScannedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing scannedAt timestamp: %w", err)
	}
	// RFC3339 truncates to second precision. A file created in the same
	// second as the scan may have sub-second precision that makes it appear
	// newer. Add 1 second to avoid false positives.
	scannedAt = scannedAt.Add(time.Second)

	// Build set of indexed file paths
	indexed := make(map[string]struct{})
	for _, e := range idx.Entries {
		if e.Kind == "file" {
			indexed[e.Path] = struct{}{}
		}
	}

	newFiles := []string{}
	modifiedFiles := []string{}

	// Walk filesystem using the same skip rules as Scan
	err = filepath.Walk(idx.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip entries we can't read
		}

		name := info.Name()
		if info.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(idx.Root, path)

		if _, ok := indexed[relPath]; ok {
			// File exists in index — check if modified since scan
			if info.ModTime().After(scannedAt) {
				modifiedFiles = append(modifiedFiles, relPath)
			}
			delete(indexed, relPath)
		} else {
			// File not in index — it's new
			newFiles = append(newFiles, relPath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	// Remaining entries in indexed map are deleted files
	deletedFiles := make([]string, 0, len(indexed))
	for p := range indexed {
		deletedFiles = append(deletedFiles, p)
	}
	sort.Strings(deletedFiles)
	sort.Strings(newFiles)
	sort.Strings(modifiedFiles)

	result := &StaleResult{
		ScannedAt:     idx.ScannedAt,
		IsStale:       len(newFiles)+len(deletedFiles)+len(modifiedFiles) > 0,
		NewFiles:      newFiles,
		DeletedFiles:  deletedFiles,
		ModifiedFiles: modifiedFiles,
		Summary: StaleSummary{
			New:      len(newFiles),
			Deleted:  len(deletedFiles),
			Modified: len(modifiedFiles),
		},
	}

	return result, nil
}

// FormatStale returns a human-readable text rendering of the stale result.
func FormatStale(r *StaleResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Index scanned at %s\n", r.ScannedAt))

	if !r.IsStale {
		b.WriteString("\nNo changes detected — index is up to date.\n")
		return b.String()
	}

	if len(r.NewFiles) > 0 {
		b.WriteString(fmt.Sprintf("\nNew files (not in index): %d\n", len(r.NewFiles)))
		for _, f := range r.NewFiles {
			b.WriteString(fmt.Sprintf("  %s\n", f))
		}
	}

	if len(r.DeletedFiles) > 0 {
		b.WriteString(fmt.Sprintf("\nDeleted files (in index but missing from disk): %d\n", len(r.DeletedFiles)))
		for _, f := range r.DeletedFiles {
			b.WriteString(fmt.Sprintf("  %s\n", f))
		}
	}

	if len(r.ModifiedFiles) > 0 {
		b.WriteString(fmt.Sprintf("\nModified files (changed since last scan): %d\n", len(r.ModifiedFiles)))
		for _, f := range r.ModifiedFiles {
			b.WriteString(fmt.Sprintf("  %s\n", f))
		}
	}

	b.WriteString(fmt.Sprintf("\nSummary: %d new, %d deleted, %d modified — index is STALE (run 'swarm-index scan' to update)\n",
		r.Summary.New, r.Summary.Deleted, r.Summary.Modified))

	return b.String()
}
