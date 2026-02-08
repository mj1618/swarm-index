package index

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// BlameLine represents a single line of git blame output.
type BlameLine struct {
	Line    int    `json:"line"`
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Content string `json:"content"`
}

// BlameResult holds the git blame output for a file.
type BlameResult struct {
	File  string      `json:"file"`
	Lines []BlameLine `json:"lines"`
	Total int         `json:"total"`
}

// Blame returns git blame information for a file, optionally filtered to a line range.
// When startLine and endLine are both 0, the entire file is blamed.
func Blame(root string, file string, startLine, endLine int) (*BlameResult, error) {
	args := []string{"blame", "--porcelain"}
	if startLine > 0 || endLine > 0 {
		var rangeSpec string
		if startLine > 0 && endLine > 0 {
			rangeSpec = fmt.Sprintf("-L%d,%d", startLine, endLine)
		} else if startLine > 0 {
			rangeSpec = fmt.Sprintf("-L%d,", startLine)
		} else {
			rangeSpec = fmt.Sprintf("-L,%d", endLine)
		}
		args = append(args, rangeSpec)
	}
	args = append(args, "--", file)

	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("git blame failed: %s", stderr)
			}
		}
		return nil, fmt.Errorf("git blame failed: %w", err)
	}

	lines := parsePorcelain(string(out))
	if lines == nil {
		lines = []BlameLine{}
	}

	return &BlameResult{
		File:  file,
		Lines: lines,
		Total: len(lines),
	}, nil
}

// parsePorcelain parses git blame --porcelain output into BlameLine entries.
func parsePorcelain(output string) []BlameLine {
	var result []BlameLine
	rawLines := strings.Split(output, "\n")

	var hash, author, date string
	var finalLine int

	for _, raw := range rawLines {
		if raw == "" {
			continue
		}

		// Content line: starts with a tab
		if strings.HasPrefix(raw, "\t") {
			result = append(result, BlameLine{
				Line:    finalLine,
				Hash:    hash,
				Author:  author,
				Date:    date,
				Content: raw[1:], // strip leading tab
			})
			continue
		}

		// Header line: <hash> <orig-line> <final-line> [<num-lines>]
		parts := strings.Fields(raw)
		if len(parts) >= 3 && len(parts[0]) == 40 {
			hash = parts[0][:8]
			if n, err := strconv.Atoi(parts[2]); err == nil {
				finalLine = n
			}
			continue
		}

		// Metadata lines
		if strings.HasPrefix(raw, "author ") {
			author = strings.TrimPrefix(raw, "author ")
		} else if strings.HasPrefix(raw, "author-time ") {
			ts := strings.TrimPrefix(raw, "author-time ")
			if epoch, err := strconv.ParseInt(ts, 10, 64); err == nil {
				date = time.Unix(epoch, 0).UTC().Format("2006-01-02")
			}
		}
	}

	return result
}

// FormatBlame returns a human-readable rendering of the blame result.
func FormatBlame(result *BlameResult) string {
	var b strings.Builder

	if result.Total == 0 {
		b.WriteString(fmt.Sprintf("No blame info for %s\n", result.File))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("%s:\n", result.File))
	for _, l := range result.Lines {
		author := l.Author
		if len(author) > 12 {
			author = author[:12]
		}
		b.WriteString(fmt.Sprintf("  %4d  %s  %s  %-12s  %s\n", l.Line, l.Hash, l.Date, author, l.Content))
	}
	return b.String()
}
