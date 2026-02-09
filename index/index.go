// Package index provides codebase scanning and symbol lookup for coding agents.
package index

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matt/swarm-index/parsers"
)

// Entry represents a single indexed item (file, symbol, package, etc.).
type Entry struct {
	Name     string `json:"name"`               // symbol or file name
	Kind     string `json:"kind"`               // "file", "func", "type", "package", etc.
	Path     string `json:"path"`               // file path relative to the scanned root
	Line     int    `json:"line"`               // line number (0 if not applicable)
	Package  string `json:"package"`            // package or module the entry belongs to
	Exported bool   `json:"exported,omitempty"` // true if the symbol is publicly exported
}

func (e Entry) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("[%s] %s — %s:%d (%s)", e.Kind, e.Name, e.Path, e.Line, e.Package)
	}
	return fmt.Sprintf("[%s] %s — %s (%s)", e.Kind, e.Name, e.Path, e.Package)
}

// Index holds the scanned codebase data.
type Index struct {
	Root      string
	Entries   []Entry
	ScannedAt string
}

// FilePaths returns the unique file paths in the index, preserving first-seen order.
func (idx *Index) FilePaths() []string {
	seen := make(map[string]struct{})
	var paths []string
	for _, e := range idx.Entries {
		if _, ok := seen[e.Path]; ok {
			continue
		}
		seen[e.Path] = struct{}{}
		paths = append(paths, e.Path)
	}
	return paths
}

// FileCount returns the number of unique files in the index.
func (idx *Index) FileCount() int {
	return len(idx.FilePaths())
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

// ExtensionCounts returns a map of file extension to count across unique file paths.
// Files with no extension are counted under "(none)".
func (idx *Index) ExtensionCounts() map[string]int {
	seen := make(map[string]struct{})
	counts := make(map[string]int)
	for _, e := range idx.Entries {
		if _, ok := seen[e.Path]; ok {
			continue
		}
		seen[e.Path] = struct{}{}
		ext := filepath.Ext(e.Path)
		if ext == "" {
			ext = "(none)"
		}
		counts[ext]++
	}
	return counts
}

// indexMeta holds metadata about a saved index.
type indexMeta struct {
	Root         string         `json:"root"`
	ScannedAt    string         `json:"scannedAt"`
	Version      string         `json:"version"`
	FileCount    int            `json:"fileCount"`
	PackageCount int            `json:"packageCount"`
	Extensions   map[string]int `json:"extensions"`
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
		Extensions:   idx.ExtensionCounts(),
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

	return &Index{Root: meta.Root, Entries: entries, ScannedAt: meta.ScannedAt}, nil
}

// Scan walks a directory tree and builds an index of files and packages.
func Scan(root string) (*Index, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	idx := &Index{Root: root}
	ignorePatterns := loadIgnorePatterns(root)

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip entries we can't read
		}

		// Skip hidden directories and common noise
		name := info.Name()
		relPath, _ := filepath.Rel(root, path)

		if info.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			if relPath != "." && shouldIgnore(relPath, true, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldIgnore(relPath, false, ignorePatterns) {
			return nil
		}

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

		// Parse symbols from source files using the parser registry.
		ext := filepath.Ext(name)
		if p := parsers.ForExtension(ext); p != nil {
			content, readErr := os.ReadFile(path)
			if readErr == nil {
				symbols, parseErr := p.Parse(relPath, content)
				if parseErr == nil {
					for _, sym := range symbols {
						idx.Entries = append(idx.Entries, Entry{
							Name:     sym.Name,
							Kind:     sym.Kind,
							Path:     relPath,
							Line:     sym.Line,
							Package:  pkg,
							Exported: sym.Exported,
						})
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return idx, nil
}

// Match returns entries ranked by relevance using fuzzy matching and scoring.
// Results are sorted by score descending, with shorter paths as tie-breaker.
func (idx *Index) Match(query string) []Entry {
	scored := idx.matchFuzzy(query)
	results := make([]Entry, len(scored))
	for i, s := range scored {
		results[i] = s.Entry
	}
	return results
}

// MatchScored returns entries with their relevance scores for JSON output.
func (idx *Index) MatchScored(query string) []ScoredEntry {
	return idx.matchFuzzy(query)
}

// MatchExact returns all entries whose name or path contains the query
// (case-insensitive substring match, unranked).
func (idx *Index) MatchExact(query string) []Entry {
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

// openTextFile opens a file and verifies it's not binary (no null bytes in
// first 512 bytes). Returns the open file seeked back to the start, ready for
// reading. Returns nil, nil for binary files. Caller must close the file.
func openTextFile(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	header := make([]byte, 512)
	n, err := f.Read(header)
	if err != nil && n == 0 {
		f.Close()
		return nil, err
	}
	for _, b := range header[:n] {
		if b == 0 {
			f.Close()
			return nil, nil // binary file
		}
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// loadIgnorePatterns reads ignore patterns from .swarmignore at root and
// swarm/.swarmindexignore, merging them. Returns nil if neither file exists.
func loadIgnorePatterns(root string) []string {
	var patterns []string
	for _, name := range []string{".swarmignore", filepath.Join("swarm", ".swarmindexignore")} {
		patterns = append(patterns, readIgnoreFile(filepath.Join(root, name))...)
	}
	if len(patterns) == 0 {
		return nil
	}
	return patterns
}

// readIgnoreFile parses a gitignore-style file and returns its patterns.
// Returns nil if the file doesn't exist.
func readIgnoreFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// shouldIgnore checks if a relative path matches any .swarmignore pattern.
// For directories, pass the relative dir path. For files, pass the relative file path.
// isDir should be true when checking a directory entry.
func shouldIgnore(relPath string, isDir bool, patterns []string) bool {
	basename := filepath.Base(relPath)
	for _, pattern := range patterns {
		// Directory-only pattern (trailing /)
		if strings.HasSuffix(pattern, "/") {
			if !isDir {
				continue
			}
			dirPattern := strings.TrimSuffix(pattern, "/")
			// Match against basename
			if matched, _ := filepath.Match(dirPattern, basename); matched {
				return true
			}
			// Match against full relative path
			if matched, _ := filepath.Match(dirPattern, relPath); matched {
				return true
			}
			continue
		}

		// Rooted pattern (leading /)
		if strings.HasPrefix(pattern, "/") {
			rooted := strings.TrimPrefix(pattern, "/")
			if matched, _ := filepath.Match(rooted, relPath); matched {
				return true
			}
			continue
		}

		// Basename glob pattern (no path separator in pattern)
		if !strings.Contains(pattern, "/") {
			if matched, _ := filepath.Match(pattern, basename); matched {
				return true
			}
			continue
		}

		// Path pattern with separator — match against full relative path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
	}
	return false
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
