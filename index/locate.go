package index

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// LocateMatch represents a single match from the unified locate search.
type LocateMatch struct {
	Category string `json:"category"`           // "file", "symbol", or "content"
	Path     string `json:"path"`               // file path relative to root
	Name     string `json:"name"`               // filename or symbol name
	Line     int    `json:"line,omitempty"`      // for symbol and content matches
	Kind     string `json:"kind,omitempty"`      // for symbols: "func", "type", etc.
	Content  string `json:"content,omitempty"`   // for content matches: the matching line
	Score    int    `json:"score"`               // relevance score for ranking
}

// LocateResult holds the result of a unified locate search.
type LocateResult struct {
	Query   string        `json:"query"`
	Matches []LocateMatch `json:"matches"`
	Total   int           `json:"total"` // total before limiting
}

// Locate searches across filenames, symbols, and file contents simultaneously,
// returning a unified, relevance-ranked result set.
func (idx *Index) Locate(query string, max int) (*LocateResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	queryLower := strings.ToLower(query)
	var matches []LocateMatch

	// 1. File matches — use idx.Match() to find matching filenames/paths.
	fileMatches := idx.Match(query)
	for _, e := range fileMatches {
		if e.Kind != "file" {
			continue
		}
		score := 60 // path-only match
		nameLower := strings.ToLower(e.Name)
		if nameLower == queryLower || nameWithoutExt(nameLower) == queryLower {
			score = 100 // exact filename match
		} else if strings.Contains(nameLower, queryLower) {
			score = 80 // filename contains query
		}
		matches = append(matches, LocateMatch{
			Category: "file",
			Path:     e.Path,
			Name:     e.Name,
			Score:    score,
		})
	}

	// 2. Symbol matches — use idx.Symbols() to find matching symbols.
	// Use a higher internal limit to avoid missing relevant symbols.
	symbolLimit := max * 5
	if symbolLimit < 100 {
		symbolLimit = 100
	}
	symbolsResult, err := idx.Symbols(query, "", symbolLimit)
	if err == nil {
		for _, sym := range symbolsResult.Matches {
			symNameLower := strings.ToLower(sym.Name)
			score := 65 // name contains query
			if symNameLower == queryLower {
				score = 90 // exact name match
			} else if strings.HasPrefix(symNameLower, queryLower) {
				score = 75 // name starts with query
			}
			matches = append(matches, LocateMatch{
				Category: "symbol",
				Path:     sym.Path,
				Name:     sym.Name,
				Line:     sym.Line,
				Kind:     sym.Kind,
				Score:    score,
			})
		}
	}

	// 3. Content matches — use idx.Search() for literal text matches.
	// Use a higher internal limit so broad queries don't miss relevant files
	// that happen to appear later in the file list.
	contentLimit := max * 10
	if contentLimit < 200 {
		contentLimit = 200
	}
	contentMatches, err := idx.Search(query, contentLimit)
	if err == nil {
		for _, m := range contentMatches {
			matches = append(matches, LocateMatch{
				Category: "content",
				Path:     m.Path,
				Name:     filepath.Base(m.Path),
				Line:     m.Line,
				Content:  m.Content,
				Score:    50,
			})
		}
	}

	// Sort by score descending, then by path for stability.
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Path != matches[j].Path {
			return matches[i].Path < matches[j].Path
		}
		return matches[i].Line < matches[j].Line
	})

	// Deduplicate: if a file appears in multiple categories, keep the higher-scored entry.
	matches = deduplicateLocateMatches(matches)

	total := len(matches)
	if max > 0 && len(matches) > max {
		matches = matches[:max]
	}

	return &LocateResult{
		Query:   query,
		Matches: matches,
		Total:   total,
	}, nil
}

// deduplicateLocateMatches removes lower-scored duplicate entries for the same
// file+line combination. Matches are assumed to be sorted by score descending.
func deduplicateLocateMatches(matches []LocateMatch) []LocateMatch {
	type key struct {
		path string
		line int
	}
	seen := make(map[key]bool)
	var result []LocateMatch
	for _, m := range matches {
		k := key{m.Path, m.Line}
		if m.Line == 0 {
			// For file matches (line=0), deduplicate by path only.
			k = key{m.Path, -1}
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		result = append(result, m)
	}
	return result
}

// nameWithoutExt returns a filename without its extension, lowercased.
func nameWithoutExt(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return name[:len(name)-len(ext)]
}

// FormatLocate returns a human-readable text rendering of the locate result.
func FormatLocate(result *LocateResult) string {
	var b strings.Builder

	if len(result.Matches) == 0 {
		b.WriteString(fmt.Sprintf("No matches for %q\n", result.Query))
		return b.String()
	}

	// Group matches by category.
	var files, symbols, content []LocateMatch
	for _, m := range result.Matches {
		switch m.Category {
		case "file":
			files = append(files, m)
		case "symbol":
			symbols = append(symbols, m)
		case "content":
			content = append(content, m)
		}
	}

	if len(files) > 0 {
		b.WriteString("Files:\n")
		for _, m := range files {
			b.WriteString(fmt.Sprintf("  %-50s (score: %d)\n", m.Path, m.Score))
		}
		b.WriteString("\n")
	}

	if len(symbols) > 0 {
		b.WriteString("Symbols:\n")
		for _, m := range symbols {
			b.WriteString(fmt.Sprintf("  %-10s %-40s %s:%d\n", m.Kind, m.Name, m.Path, m.Line))
		}
		b.WriteString("\n")
	}

	if len(content) > 0 {
		b.WriteString("Content:\n")
		for _, m := range content {
			b.WriteString(fmt.Sprintf("  %s:%d\t%s\n", m.Path, m.Line, m.Content))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("%d total matches\n", result.Total))

	return b.String()
}
