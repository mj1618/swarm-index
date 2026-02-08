package index

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// HotspotEntry represents a file ranked by how frequently it appears in git commits.
type HotspotEntry struct {
	Path         string `json:"path"`
	CommitCount  int    `json:"commitCount"`
	LastModified string `json:"lastModified"`
}

// HotspotsResult holds the ranked list of most frequently changed files.
type HotspotsResult struct {
	Entries []HotspotEntry `json:"entries"`
	Total   int            `json:"total"`
	Since   string         `json:"since,omitempty"`
}

// Hotspots returns the most frequently changed files in the git history,
// cross-referenced against the index to exclude deleted files.
func (idx *Index) Hotspots(root string, max int, since string, pathPrefix string) (*HotspotsResult, error) {
	// Build git log command to get file names from all commits
	gitArgs := []string{"log", "--format=format:", "--name-only"}
	if since != "" {
		gitArgs = append(gitArgs, "--since="+since)
	}
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("git log failed: %s", stderr)
			}
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	// Build a set of files that still exist in the index
	indexed := make(map[string]struct{})
	for _, e := range idx.Entries {
		if e.Kind == "file" {
			indexed[e.Path] = struct{}{}
		}
	}

	// Count occurrences of each file path
	counts := make(map[string]int)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Apply path prefix filter
		if pathPrefix != "" && !strings.HasPrefix(line, pathPrefix) {
			continue
		}
		// Only count files that still exist in the index
		if _, ok := indexed[line]; !ok {
			continue
		}
		counts[line]++
	}

	// Sort by commit count descending
	type fileCount struct {
		path  string
		count int
	}
	sorted := make([]fileCount, 0, len(counts))
	for path, count := range counts {
		sorted = append(sorted, fileCount{path, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].path < sorted[j].path
	})

	total := len(sorted)

	// Limit results
	if max > 0 && len(sorted) > max {
		sorted = sorted[:max]
	}

	// Get last modified date for each entry
	entries := make([]HotspotEntry, 0, len(sorted))
	for _, fc := range sorted {
		lastMod := getLastModified(root, fc.path)
		entries = append(entries, HotspotEntry{
			Path:         fc.path,
			CommitCount:  fc.count,
			LastModified: lastMod,
		})
	}

	return &HotspotsResult{
		Entries: entries,
		Total:   total,
		Since:   since,
	}, nil
}

// getLastModified returns the ISO 8601 date of the most recent commit for a file.
func getLastModified(root, filePath string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%aI", "--", filePath)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// FormatHotspots returns a human-readable rendering of the hotspots result.
func FormatHotspots(result *HotspotsResult) string {
	var b strings.Builder

	if len(result.Entries) == 0 {
		b.WriteString("No hotspots found\n")
		return b.String()
	}

	header := fmt.Sprintf("Hotspots (top %d most changed files)", len(result.Entries))
	if result.Since != "" {
		header += fmt.Sprintf(" since %s", result.Since)
	}
	b.WriteString(header + ":\n\n")

	for _, e := range result.Entries {
		date := e.LastModified
		if len(date) >= 10 {
			date = date[:10]
		}
		b.WriteString(fmt.Sprintf("  %3d commits  %-50s (last: %s)\n", e.CommitCount, e.Path, date))
	}

	b.WriteString(fmt.Sprintf("\n%d of %d files shown\n", len(result.Entries), result.Total))
	return b.String()
}
