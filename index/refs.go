package index

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"
)

// RefMatch represents a single occurrence of a symbol.
type RefMatch struct {
	Path         string `json:"path"`
	Line         int    `json:"line"`
	Content      string `json:"content"`
	IsDefinition bool   `json:"isDefinition"`
}

// RefsResult holds the definition and all references to a symbol.
type RefsResult struct {
	Symbol     string     `json:"symbol"`
	Definition *RefMatch  `json:"definition"`
	References []RefMatch `json:"references"`
	TotalRefs  int        `json:"totalReferences"`
}

// definitionPatterns are regex patterns that indicate a line is a symbol definition.
var definitionPatterns = []string{
	`func\s+%s\b`,          // Go function
	`func\s+\([^)]+\)\s+%s\b`, // Go method
	`type\s+%s\b`,          // Go type
	`var\s+%s\b`,           // Go/JS var
	`const\s+%s\b`,         // Go/JS const
	`class\s+%s\b`,         // JS/Python/Java class
	`def\s+%s\b`,           // Python function
	`let\s+%s\b`,           // JS let
	`function\s+%s\b`,      // JS function
	`interface\s+%s\b`,     // Java/TS interface
	`struct\s+%s\b`,        // Go struct (redundant with type but catches inline)
	`enum\s+%s\b`,          // Java/TS enum
}

// Refs finds the definition and all references of a symbol across indexed files.
func (idx *Index) Refs(symbol string, maxResults int) (*RefsResult, error) {
	wordRe, err := regexp.Compile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
	if err != nil {
		return nil, err
	}

	// Build definition-detecting regexes.
	var defRegexes []*regexp.Regexp
	for _, pat := range definitionPatterns {
		re, err := regexp.Compile(strings.Replace(pat, "%s", regexp.QuoteMeta(symbol), 1))
		if err != nil {
			continue
		}
		defRegexes = append(defRegexes, re)
	}

	// Check if any index entry is the authoritative definition.
	var indexDefPath string
	var indexDefLine int
	for _, e := range idx.Entries {
		if e.Name == symbol && e.Line > 0 {
			indexDefPath = e.Path
			indexDefLine = e.Line
			break
		}
	}

	paths := idx.FilePaths()

	result := &RefsResult{
		Symbol:     symbol,
		References: []RefMatch{},
	}

	for _, p := range paths {
		if result.TotalRefs >= maxResults {
			break
		}
		full := filepath.Join(idx.Root, p)
		matches, err := refsInFile(full, p, wordRe, defRegexes, indexDefPath, indexDefLine, maxResults-result.TotalRefs)
		if err != nil {
			continue
		}
		for _, m := range matches {
			if m.IsDefinition && result.Definition == nil {
				def := m
				result.Definition = &def
			} else {
				if result.TotalRefs < maxResults {
					result.References = append(result.References, m)
					result.TotalRefs++
				}
			}
		}
	}

	return result, nil
}

// refsInFile scans a single file for symbol occurrences, classifying each as
// definition or reference. Binary files are skipped.
func refsInFile(fullPath, relPath string, wordRe *regexp.Regexp, defRegexes []*regexp.Regexp, indexDefPath string, indexDefLine int, limit int) ([]RefMatch, error) {
	f, err := openTextFile(fullPath)
	if err != nil || f == nil {
		return nil, err
	}
	defer f.Close()

	var matches []RefMatch
	refCount := 0
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if !wordRe.MatchString(line) {
			continue
		}

		isDef := false
		// Check authoritative index definition.
		if relPath == indexDefPath && lineNum == indexDefLine {
			isDef = true
		}
		// Check heuristic definition patterns.
		if !isDef {
			for _, re := range defRegexes {
				if re.MatchString(line) {
					isDef = true
					break
				}
			}
		}

		matches = append(matches, RefMatch{
			Path:         relPath,
			Line:         lineNum,
			Content:      strings.TrimSpace(line),
			IsDefinition: isDef,
		})

		if !isDef {
			refCount++
			if refCount >= limit {
				break
			}
		}
	}

	return matches, nil
}
