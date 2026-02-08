package index

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// TodoComment represents a single TODO/FIXME/HACK/XXX comment found in source.
type TodoComment struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Tag     string `json:"tag"`     // "TODO", "FIXME", "HACK", "XXX"
	Message string `json:"message"` // Text after the tag
	Content string `json:"content"` // Full line content (trimmed)
}

// TodosResult holds the collected TODO comments.
type TodosResult struct {
	Comments []TodoComment  `json:"comments"`
	Total    int            `json:"total"`
	ByTag    map[string]int `json:"byTag"`
}

var todoPattern = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|XXX)\b[:\s]*(.*)`)

// Todos scans indexed files for TODO/FIXME/HACK/XXX comments.
// If tag is non-empty, only comments matching that tag are returned.
// maxResults limits the number of returned comments (default 100 if <= 0).
func (idx *Index) Todos(tag string, maxResults int) (*TodosResult, error) {
	if maxResults <= 0 {
		maxResults = 100
	}
	tag = strings.ToUpper(tag)

	var comments []TodoComment
	byTag := map[string]int{"TODO": 0, "FIXME": 0, "HACK": 0, "XXX": 0}
	total := 0

	paths := idx.FilePaths()
	sort.Strings(paths)

	for _, relPath := range paths {
		absPath := filepath.Join(idx.Root, relPath)
		found := todosInFile(absPath, relPath)
		for _, tc := range found {
			byTag[tc.Tag]++
			total++
			if tag != "" && tc.Tag != tag {
				continue
			}
			if len(comments) < maxResults {
				comments = append(comments, tc)
			}
		}
	}

	return &TodosResult{
		Comments: comments,
		Total:    total,
		ByTag:    byTag,
	}, nil
}

// todosInFile scans a single file for TODO/FIXME/HACK/XXX comments.
// Binary files are skipped.
func todosInFile(fullPath, relPath string) []TodoComment {
	f, err := openTextFile(fullPath)
	if err != nil || f == nil {
		return nil
	}
	defer f.Close()

	var comments []TodoComment
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matches := todoPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		comments = append(comments, TodoComment{
			Path:    relPath,
			Line:    lineNum,
			Tag:     strings.ToUpper(matches[1]),
			Message: strings.TrimSpace(matches[2]),
			Content: strings.TrimSpace(line),
		})
	}
	return comments
}

// FormatTodos returns a human-readable text rendering of the todos result.
func FormatTodos(r *TodosResult) string {
	var b strings.Builder

	if len(r.Comments) == 0 {
		b.WriteString("No TODO/FIXME/HACK/XXX comments found.\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("TODO comments (%d found):\n\n", r.Total))

	for _, c := range r.Comments {
		b.WriteString(fmt.Sprintf("  %s:%d  %s: %s\n", c.Path, c.Line, c.Tag, c.Message))
	}

	if len(r.Comments) < r.Total {
		b.WriteString(fmt.Sprintf("\n  ... %d more (use --max to see more)\n", r.Total-len(r.Comments)))
	}

	// Summary line
	var parts []string
	for _, tag := range []string{"TODO", "FIXME", "HACK", "XXX"} {
		if count, ok := r.ByTag[tag]; ok && count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, tag))
		}
	}
	b.WriteString(fmt.Sprintf("\nSummary: %s\n", strings.Join(parts, ", ")))

	return b.String()
}
