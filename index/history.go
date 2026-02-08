package index

import (
	"fmt"
	"os/exec"
	"strings"
)

// HistoryCommit represents a single git commit for a file.
type HistoryCommit struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

// HistoryResult holds the git history for a file.
type HistoryResult struct {
	Path    string          `json:"path"`
	Commits []HistoryCommit `json:"commits"`
	Total   int             `json:"total"`
}

// History returns recent git commits that touched the given file.
func History(root, filePath string, max int) (*HistoryResult, error) {
	cmd := exec.Command("git", "log",
		fmt.Sprintf("-n%d", max),
		"--format=%h%x00%an%x00%aI%x00%s",
		"--", filePath,
	)
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

	var commits []HistoryCommit
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) != 4 {
			continue
		}
		commits = append(commits, HistoryCommit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Subject: parts[3],
		})
	}

	if commits == nil {
		commits = []HistoryCommit{}
	}

	return &HistoryResult{
		Path:    filePath,
		Commits: commits,
		Total:   len(commits),
	}, nil
}

// FormatHistory returns a human-readable rendering of the history result.
func FormatHistory(result *HistoryResult) string {
	var b strings.Builder

	if result.Total == 0 {
		b.WriteString(fmt.Sprintf("No commits found for %s\n", result.Path))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("History for %s (%d commits):\n", result.Path, result.Total))
	for _, c := range result.Commits {
		// Truncate date to just the date portion (YYYY-MM-DD) for readability
		date := c.Date
		if len(date) >= 10 {
			date = date[:10]
		}
		b.WriteString(fmt.Sprintf("  %s  %s  %-20s  %s\n", c.Hash, date, c.Author, c.Subject))
	}
	return b.String()
}
