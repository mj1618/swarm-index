package index

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// TestMapEntry represents a single source-to-test-file association.
type TestMapEntry struct {
	SourceFile string `json:"sourceFile"`
	TestFile   string `json:"testFile"`
	HasTest    bool   `json:"hasTest"`
}

// TestMapSummary holds aggregate statistics about test coverage by file.
type TestMapSummary struct {
	TotalSourceFiles int     `json:"totalSourceFiles"`
	TestedFiles      int     `json:"testedFiles"`
	UntestedFiles    int     `json:"untestedFiles"`
	CoverageRatio    float64 `json:"coverageRatio"`
}

// TestMapResult is the full result of the test-map command.
type TestMapResult struct {
	Summary TestMapSummary `json:"summary"`
	Entries []TestMapEntry `json:"entries"`
}

// sourceExts are file extensions considered source files for test mapping.
var sourceExts = map[string]bool{
	".go":  true,
	".js":  true,
	".jsx": true,
	".ts":  true,
	".tsx": true,
	".py":  true,
}

// isTestFilePath returns true if the file path looks like a test file.
func isTestFilePath(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	switch ext {
	case ".go":
		return strings.HasSuffix(name, "_test")
	case ".js", ".jsx", ".ts", ".tsx":
		return strings.HasSuffix(name, ".test") || strings.HasSuffix(name, ".spec")
	case ".py":
		return strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test")
	}
	return false
}

// TestMap produces a project-wide mapping of source files to their associated test files.
func (idx *Index) TestMap(pathPrefix string, untested, tested bool, max int) (*TestMapResult, error) {
	indexedPaths := make(map[string]bool)
	for _, p := range idx.FilePaths() {
		indexedPaths[p] = true
	}

	var entries []TestMapEntry
	testedCount := 0

	for _, relPath := range idx.FilePaths() {
		ext := filepath.Ext(relPath)
		if !sourceExts[ext] {
			continue
		}
		if isTestFilePath(relPath) {
			continue
		}

		// Apply path prefix filter.
		if pathPrefix != "" {
			prefix := strings.TrimSuffix(pathPrefix, "/")
			if !strings.HasPrefix(relPath, prefix+"/") && relPath != prefix {
				continue
			}
		}

		// Find associated test files using the same logic as related.go.
		testFiles := idx.findTestFiles(relPath, indexedPaths)

		testFile := ""
		hasTest := len(testFiles) > 0
		if hasTest {
			testFile = testFiles[0]
			testedCount++
		}

		entries = append(entries, TestMapEntry{
			SourceFile: relPath,
			TestFile:   testFile,
			HasTest:    hasTest,
		})
	}

	// Sort by path.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SourceFile < entries[j].SourceFile
	})

	// Compute summary from unfiltered counts (before --untested/--tested filter).
	totalSource := len(entries)
	untestedCount := totalSource - testedCount

	// Apply --untested / --tested filter.
	if untested || tested {
		var filtered []TestMapEntry
		for _, e := range entries {
			if untested && !e.HasTest {
				filtered = append(filtered, e)
			}
			if tested && e.HasTest {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
	ratio := 0.0
	if totalSource > 0 {
		ratio = float64(testedCount) / float64(totalSource)
	}

	// Apply max limit to entries.
	if entries == nil {
		entries = []TestMapEntry{}
	}
	if max > 0 && len(entries) > max {
		entries = entries[:max]
	}

	return &TestMapResult{
		Summary: TestMapSummary{
			TotalSourceFiles: totalSource,
			TestedFiles:      testedCount,
			UntestedFiles:    untestedCount,
			CoverageRatio:    ratio,
		},
		Entries: entries,
	}, nil
}

// FormatTestMap returns a human-readable rendering of the test map result.
func FormatTestMap(r *TestMapResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Test Map (%d/%d source files have tests — %.1f%%)\n",
		r.Summary.TestedFiles, r.Summary.TotalSourceFiles, r.Summary.CoverageRatio*100))

	if len(r.Entries) == 0 {
		b.WriteString("\n  No source files found\n")
		return b.String()
	}

	// Separate tested and untested.
	var testedEntries, untestedEntries []TestMapEntry
	for _, e := range r.Entries {
		if e.HasTest {
			testedEntries = append(testedEntries, e)
		} else {
			untestedEntries = append(untestedEntries, e)
		}
	}

	if len(testedEntries) > 0 {
		b.WriteString("\nTested:\n")
		for _, e := range testedEntries {
			b.WriteString(fmt.Sprintf("  %-40s → %s\n", e.SourceFile, e.TestFile))
		}
	}

	if len(untestedEntries) > 0 {
		b.WriteString("\nUntested:\n")
		for _, e := range untestedEntries {
			b.WriteString(fmt.Sprintf("  %-40s (no test file found)\n", e.SourceFile))
		}
	}

	return b.String()
}
