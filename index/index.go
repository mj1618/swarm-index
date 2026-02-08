// Package index provides codebase scanning and symbol lookup for coding agents.
package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Entry represents a single indexed item (file, symbol, package, etc.).
type Entry struct {
	Name    string // symbol or file name
	Kind    string // "file", "func", "type", "package", etc.
	Path    string // file path relative to the scanned root
	Line    int    // line number (0 if not applicable)
	Package string // package or module the entry belongs to
}

func (e Entry) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("[%s] %s — %s:%d (%s)", e.Kind, e.Name, e.Path, e.Line, e.Package)
	}
	return fmt.Sprintf("[%s] %s — %s (%s)", e.Kind, e.Name, e.Path, e.Package)
}

// Index holds the scanned codebase data.
type Index struct {
	Root    string
	Entries []Entry
}

// FileCount returns the number of unique files in the index.
func (idx *Index) FileCount() int {
	seen := make(map[string]struct{})
	for _, e := range idx.Entries {
		seen[e.Path] = struct{}{}
	}
	return len(seen)
}

// PackageCount returns the number of unique packages in the index.
func (idx *Index) PackageCount() int {
	seen := make(map[string]struct{})
	for _, e := range idx.Entries {
		if e.Package != "" {
			seen[e.Package] = struct{}{}
		}
	}
	return len(seen)
}

// Scan walks a directory tree and builds an index of files and packages.
func Scan(root string) (*Index, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	idx := &Index{Root: root}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip entries we can't read
		}

		// Skip hidden directories and common noise
		name := info.Name()
		if info.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(root, path)
		pkg := filepath.Dir(relPath)
		if pkg == "." {
			pkg = "(root)"
		}

		idx.Entries = append(idx.Entries, Entry{
			Name:    name,
			Kind:    "file",
			Path:    relPath,
			Package: pkg,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return idx, nil
}

// Lookup searches the most recently scanned index for entries matching the query.
// For now this is a simple substring match; future versions will support fuzzy and semantic search.
func Lookup(query string) ([]Entry, error) {
	// TODO: persist and load index from disk
	return nil, fmt.Errorf("no index loaded — run 'swarm-index scan <dir>' first")
}

// Match returns all entries whose name contains the query (case-insensitive).
func (idx *Index) Match(query string) []Entry {
	q := strings.ToLower(query)
	var results []Entry
	for _, e := range idx.Entries {
		if strings.Contains(strings.ToLower(e.Name), q) {
			results = append(results, e)
		}
	}
	return results
}

func shouldSkipDir(name string) bool {
	skip := []string{
		".git", ".hg", ".svn",
		"node_modules", "vendor", "__pycache__",
		".idea", ".vscode", ".cursor",
		"dist", "build", ".next",
	}
	for _, s := range skip {
		if name == s {
			return true
		}
	}
	return strings.HasPrefix(name, ".")
}
