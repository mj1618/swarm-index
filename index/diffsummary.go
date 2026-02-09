package index

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mj1618/swarm-index/parsers"
)

// DiffFile represents a single file in a diff summary.
type DiffFile struct {
	Path    string   `json:"path"`
	Status  string   `json:"status"`            // "added", "modified", "deleted", "renamed"
	Symbols []string `json:"symbols,omitempty"`  // affected symbol names (for added/modified files)
}

// DiffSummaryResult holds the result of comparing against a git ref.
type DiffSummaryResult struct {
	Ref       string     `json:"ref"`
	Added     []DiffFile `json:"added"`
	Modified  []DiffFile `json:"modified"`
	Deleted   []DiffFile `json:"deleted"`
	FileCount int        `json:"fileCount"`
}

// DiffSummary compares the current working tree against a git ref and reports
// which files changed and what symbols they contain.
func (idx *Index) DiffSummary(root string, ref string) (*DiffSummaryResult, error) {
	// Run git diff --name-status against the ref
	cmd := exec.Command("git", "diff", "--name-status", ref)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git diff failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	var added, modified, deleted []DiffFile

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parsed := parseDiffLine(line)
		if parsed == nil {
			continue
		}

		// Skip files that would be excluded by index skip rules
		if shouldSkipDiffPath(parsed.path) {
			continue
		}

		switch parsed.status {
		case "A":
			df := DiffFile{Path: parsed.path, Status: "added"}
			df.Symbols = extractSymbols(root, parsed.path)
			added = append(added, df)
		case "M":
			df := DiffFile{Path: parsed.path, Status: "modified"}
			df.Symbols = extractSymbols(root, parsed.path)
			modified = append(modified, df)
		case "D":
			deleted = append(deleted, DiffFile{Path: parsed.path, Status: "deleted"})
		case "R":
			// Renamed: treat as deleted old + added new
			if parsed.oldPath != "" {
				deleted = append(deleted, DiffFile{Path: parsed.oldPath, Status: "deleted"})
			}
			df := DiffFile{Path: parsed.path, Status: "added"}
			df.Symbols = extractSymbols(root, parsed.path)
			added = append(added, df)
		}
	}

	if added == nil {
		added = []DiffFile{}
	}
	if modified == nil {
		modified = []DiffFile{}
	}
	if deleted == nil {
		deleted = []DiffFile{}
	}

	sort.Slice(added, func(i, j int) bool { return added[i].Path < added[j].Path })
	sort.Slice(modified, func(i, j int) bool { return modified[i].Path < modified[j].Path })
	sort.Slice(deleted, func(i, j int) bool { return deleted[i].Path < deleted[j].Path })

	return &DiffSummaryResult{
		Ref:       ref,
		Added:     added,
		Modified:  modified,
		Deleted:   deleted,
		FileCount: len(added) + len(modified) + len(deleted),
	}, nil
}

// diffLineInfo holds parsed info from a git diff --name-status line.
type diffLineInfo struct {
	status  string // "A", "M", "D", "R"
	path    string
	oldPath string // only for renames
}

// parseDiffLine parses a line from git diff --name-status output.
func parseDiffLine(line string) *diffLineInfo {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	status := fields[0]

	// Rename status is like "R100" or "R075"
	if strings.HasPrefix(status, "R") {
		if len(fields) < 3 {
			return nil
		}
		return &diffLineInfo{
			status:  "R",
			oldPath: fields[1],
			path:    fields[2],
		}
	}

	// Copy status is like "C100"
	if strings.HasPrefix(status, "C") {
		if len(fields) < 3 {
			return nil
		}
		return &diffLineInfo{
			status: "A",
			path:   fields[2],
		}
	}

	switch status {
	case "A", "M", "D":
		return &diffLineInfo{status: status, path: fields[1]}
	default:
		return nil
	}
}

// extractSymbols parses a file and returns the names of its symbols.
func extractSymbols(root, relPath string) []string {
	absPath := filepath.Join(root, relPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	ext := filepath.Ext(relPath)
	p := parsers.ForExtension(ext)
	if p == nil {
		return nil
	}

	symbols, err := p.Parse(relPath, content)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(symbols))
	for _, s := range symbols {
		names = append(names, s.Name)
	}
	return names
}

// shouldSkipDiffPath checks if a path should be skipped based on directory
// skip rules (same rules as Scan).
func shouldSkipDiffPath(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if shouldSkipDir(part) {
			return true
		}
	}
	return false
}

// FormatDiffSummary returns a human-readable text rendering of the diff summary.
func FormatDiffSummary(result *DiffSummaryResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Changes since %s (%d files):\n", result.Ref, result.FileCount))

	if result.FileCount == 0 {
		b.WriteString("\nNo changes found.\n")
		return b.String()
	}

	if len(result.Added) > 0 {
		b.WriteString("\nAdded:\n")
		for _, f := range result.Added {
			b.WriteString(fmt.Sprintf("  + %s\n", f.Path))
			if len(f.Symbols) > 0 {
				b.WriteString(fmt.Sprintf("    Symbols: %s\n", strings.Join(f.Symbols, ", ")))
			}
		}
	}

	if len(result.Modified) > 0 {
		b.WriteString("\nModified:\n")
		for _, f := range result.Modified {
			b.WriteString(fmt.Sprintf("  ~ %s\n", f.Path))
			if len(f.Symbols) > 0 {
				b.WriteString(fmt.Sprintf("    Symbols: %s\n", strings.Join(f.Symbols, ", ")))
			}
		}
	}

	if len(result.Deleted) > 0 {
		b.WriteString("\nDeleted:\n")
		for _, f := range result.Deleted {
			b.WriteString(fmt.Sprintf("  - %s\n", f.Path))
		}
	}

	return b.String()
}
