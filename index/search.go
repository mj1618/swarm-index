package index

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchMatch represents a single line matching a search pattern.
type SearchMatch struct {
	Path    string `json:"path"`    // file path relative to root
	Line    int    `json:"line"`    // 1-based line number
	Content string `json:"content"` // the matching line (trimmed)
}

// Search finds lines matching a regex pattern across all indexed files.
// It returns up to maxResults matches. Binary files are skipped.
func (idx *Index) Search(pattern string, maxResults int) ([]SearchMatch, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	paths := idx.FilePaths()

	var matches []SearchMatch
	for _, p := range paths {
		if len(matches) >= maxResults {
			break
		}
		full := filepath.Join(idx.Root, p)
		m, err := searchFile(full, p, re, maxResults-len(matches))
		if err != nil {
			continue // skip files we can't read
		}
		matches = append(matches, m...)
	}

	return matches, nil
}

// searchFile scans a single file for regex matches, returning up to limit results.
// Binary files (containing null bytes in the first 512 bytes) are skipped.
func searchFile(fullPath, relPath string, re *regexp.Regexp, limit int) ([]SearchMatch, error) {
	f, err := openTextFile(fullPath)
	if err != nil || f == nil {
		return nil, err
	}
	defer f.Close()

	var matches []SearchMatch
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, SearchMatch{
				Path:    relPath,
				Line:    lineNum,
				Content: strings.TrimSpace(line),
			})
			if len(matches) >= limit {
				break
			}
		}
	}

	return matches, nil
}
