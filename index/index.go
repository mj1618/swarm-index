// Package index provides codebase scanning and symbol lookup for coding agents.
package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a single indexed item (file, symbol, package, etc.).
type Entry struct {
	Name    string `json:"name"`    // symbol or file name
	Kind    string `json:"kind"`    // "file", "func", "type", "package", etc.
	Path    string `json:"path"`    // file path relative to the scanned root
	Line    int    `json:"line"`    // line number (0 if not applicable)
	Package string `json:"package"` // package or module the entry belongs to
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

// indexMeta holds metadata about a saved index.
type indexMeta struct {
	Root         string `json:"root"`
	ScannedAt    string `json:"scannedAt"`
	Version      string `json:"version"`
	FileCount    int    `json:"fileCount"`
	PackageCount int    `json:"packageCount"`
}

// Save writes the index to disk under <dir>/swarm/index/.
func (idx *Index) Save(dir string) error {
	indexDir := filepath.Join(dir, "swarm", "index")
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return fmt.Errorf("creating index directory: %w", err)
	}

	if err := writeJSON(filepath.Join(indexDir, "index.json"), idx.Entries); err != nil {
		return err
	}

	meta := indexMeta{
		Root:         idx.Root,
		ScannedAt:    time.Now().UTC().Format(time.RFC3339),
		Version:      "0.1.0",
		FileCount:    idx.FileCount(),
		PackageCount: idx.PackageCount(),
	}
	return writeJSON(filepath.Join(indexDir, "meta.json"), meta)
}

// writeJSON marshals v as indented JSON and writes it to path.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	return nil
}

// Load reads a persisted index from <dir>/swarm/index/.
func Load(dir string) (*Index, error) {
	indexDir := filepath.Join(dir, "swarm", "index")

	data, err := os.ReadFile(filepath.Join(indexDir, "index.json"))
	if err != nil {
		return nil, fmt.Errorf("reading index.json: %w", err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing index.json: %w", err)
	}

	metaData, err := os.ReadFile(filepath.Join(indexDir, "meta.json"))
	if err != nil {
		return nil, fmt.Errorf("reading meta.json: %w", err)
	}

	var meta indexMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("parsing meta.json: %w", err)
	}

	return &Index{Root: meta.Root, Entries: entries}, nil
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

// Match returns all entries whose name or path contains the query (case-insensitive).
func (idx *Index) Match(query string) []Entry {
	q := strings.ToLower(query)
	var results []Entry
	for _, e := range idx.Entries {
		if strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Path), q) {
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
		"swarm",
	}
	for _, s := range skip {
		if name == s {
			return true
		}
	}
	return strings.HasPrefix(name, ".")
}
