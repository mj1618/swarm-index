package index

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mj1618/swarm-index/parsers"
)

// SymbolCount tracks exported vs internal counts for a symbol kind.
type SymbolCount struct {
	Exported int `json:"exported"`
	Internal int `json:"internal"`
}

// ScopeResult holds the focused summary of a single directory.
type ScopeResult struct {
	Directory    string                  `json:"directory"`
	Files        []string                `json:"files"`
	FileCount    int                     `json:"fileCount"`
	LOC          int                     `json:"loc"`
	Symbols      map[string]SymbolCount  `json:"symbols"`
	Dependencies []string                `json:"dependencies"`
	Dependents   []string                `json:"dependents"`
}

// Scope produces a focused summary of a single directory/package.
// If recursive is true, files in subdirectories are included.
func (idx *Index) Scope(dir string, recursive bool) (*ScopeResult, error) {
	// Normalize directory — remove trailing slash.
	dir = strings.TrimRight(dir, "/"+string(filepath.Separator))
	if dir == "" {
		dir = "."
	}

	// Collect file paths within the target directory.
	var filePaths []string
	for _, e := range idx.Entries {
		if e.Kind != "file" {
			continue
		}
		if inScope(e.Path, dir, recursive) {
			filePaths = append(filePaths, e.Path)
		}
	}

	if len(filePaths) == 0 {
		return &ScopeResult{
			Directory:    dir,
			Files:        []string{},
			FileCount:    0,
			LOC:          0,
			Symbols:      map[string]SymbolCount{},
			Dependencies: []string{},
			Dependents:   []string{},
		}, nil
	}

	sort.Strings(filePaths)

	// Build set of in-scope paths for fast lookup.
	scopeSet := make(map[string]bool, len(filePaths))
	for _, p := range filePaths {
		scopeSet[p] = true
	}

	// Count symbols and LOC.
	symbolCounts := make(map[string]SymbolCount)
	totalLOC := 0

	for _, relPath := range filePaths {
		absPath := filepath.Join(idx.Root, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		// Count lines of code.
		totalLOC += bytes.Count(content, []byte("\n"))
		if len(content) > 0 && content[len(content)-1] != '\n' {
			totalLOC++ // count last line if no trailing newline
		}

		// Parse symbols.
		ext := filepath.Ext(relPath)
		p := parsers.ForExtension(ext)
		if p == nil {
			continue
		}
		symbols, err := p.Parse(absPath, content)
		if err != nil {
			continue
		}
		for _, sym := range symbols {
			sc := symbolCounts[sym.Kind]
			if sym.Exported {
				sc.Exported++
			} else {
				sc.Internal++
			}
			symbolCounts[sym.Kind] = sc
		}
	}

	// Build set of all indexed file paths for import resolution.
	indexedPaths := make(map[string]bool)
	for _, p := range idx.FilePaths() {
		indexedPaths[p] = true
	}

	// Find dependencies: directories that files in this scope import from.
	depDirs := make(map[string]bool)
	for _, relPath := range filePaths {
		imports := idx.extractImports(relPath, indexedPaths)
		for _, imp := range imports {
			impDir := filepath.Dir(imp)
			if !scopeSet[imp] {
				// Imported file is outside scope — record its directory.
				depDirs[impDir] = true
			}
		}
	}

	// Find dependents: directories whose files import files from this scope.
	dependentDirs := make(map[string]bool)
	for p := range indexedPaths {
		if scopeSet[p] {
			continue
		}
		imports := idx.extractImports(p, indexedPaths)
		for _, imp := range imports {
			if scopeSet[imp] {
				importerDir := filepath.Dir(p)
				dependentDirs[importerDir] = true
				break
			}
		}
	}

	// Convert file paths to basenames for the output.
	fileNames := make([]string, len(filePaths))
	for i, p := range filePaths {
		fileNames[i] = filepath.Base(p)
	}

	return &ScopeResult{
		Directory:    dir,
		Files:        fileNames,
		FileCount:    len(filePaths),
		LOC:          totalLOC,
		Symbols:      symbolCounts,
		Dependencies: sortedKeys(depDirs),
		Dependents:   sortedKeys(dependentDirs),
	}, nil
}

// inScope checks if a file path is within the given directory scope.
func inScope(filePath, dir string, recursive bool) bool {
	if dir == "." {
		if recursive {
			return true
		}
		return filepath.Dir(filePath) == "."
	}
	if recursive {
		return strings.HasPrefix(filePath, dir+"/") || strings.HasPrefix(filePath, dir+string(filepath.Separator))
	}
	return filepath.Dir(filePath) == dir
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FormatScope returns a human-readable text rendering of the scope result.
func FormatScope(r *ScopeResult) string {
	var b strings.Builder

	if r.FileCount == 0 {
		b.WriteString(fmt.Sprintf("Scope: %s/\n  No files found\n", r.Directory))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Scope: %s/\n", r.Directory))
	b.WriteString(fmt.Sprintf("  %d files, %d LOC\n", r.FileCount, r.LOC))

	// Symbols section.
	if len(r.Symbols) > 0 {
		b.WriteString("\n  Symbols:\n")

		// Sort symbol kinds for deterministic output.
		kinds := make([]string, 0, len(r.Symbols))
		for k := range r.Symbols {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)

		for _, kind := range kinds {
			sc := r.Symbols[kind]
			total := sc.Exported + sc.Internal
			b.WriteString(fmt.Sprintf("    %-10s %d (%d exported, %d internal)\n", kind, total, sc.Exported, sc.Internal))
		}
	}

	// Dependencies section.
	if len(r.Dependencies) > 0 {
		b.WriteString("\n  Dependencies (imports from):\n")
		for _, dep := range r.Dependencies {
			b.WriteString(fmt.Sprintf("    %s/\n", dep))
		}
	}

	// Dependents section.
	if len(r.Dependents) > 0 {
		b.WriteString("\n  Depended on by:\n")
		for _, dep := range r.Dependents {
			b.WriteString(fmt.Sprintf("    %s/\n", dep))
		}
	}

	return b.String()
}
