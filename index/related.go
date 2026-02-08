package index

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// RelatedResult holds the dependency neighborhood of a file.
type RelatedResult struct {
	File      string   `json:"file"`      // the target file path (relative to root)
	Imports   []string `json:"imports"`   // files this file imports/requires
	Importers []string `json:"importers"` // files that import/require this file
	TestFiles []string `json:"testFiles"` // associated test files
}

// importable file extensions with known import syntax.
var importableExts = map[string]bool{
	".go":  true,
	".js":  true,
	".jsx": true,
	".ts":  true,
	".tsx": true,
	".py":  true,
}

// Import extraction regexes.
var (
	// Go: import "path" or import ( "path" )
	goImportSingle = regexp.MustCompile(`^\s*import\s+"([^"]+)"`)
	goImportLine   = regexp.MustCompile(`^\s*"([^"]+)"`)
	goImportBlock  = regexp.MustCompile(`^\s*import\s*\(`)
	goImportEnd    = regexp.MustCompile(`^\s*\)`)

	// JS/TS: import ... from '...' or require('...')
	jsImportFrom = regexp.MustCompile(`(?:import|export)\s+.*?from\s+['"]([^'"]+)['"]`)
	jsRequire    = regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	// Python: from X import ... or import X
	pyFromImport = regexp.MustCompile(`^\s*from\s+(\S+)\s+import`)
	pyImport     = regexp.MustCompile(`^\s*import\s+(\S+)`)
)

// Related finds files connected to the given file path: imports, importers, and test files.
func (idx *Index) Related(filePath string) (*RelatedResult, error) {
	// Normalize to relative path.
	relPath := filePath
	if filepath.IsAbs(filePath) {
		var err error
		relPath, err = filepath.Rel(idx.Root, filePath)
		if err != nil {
			return nil, fmt.Errorf("cannot make path relative to root: %w", err)
		}
	}

	// Verify the file exists in the index.
	found := false
	for _, e := range idx.Entries {
		if e.Path == relPath {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("file %s not found in index", relPath)
	}

	// Build a set of all indexed file paths for fast lookup.
	indexedPaths := make(map[string]bool)
	for _, p := range idx.FilePaths() {
		indexedPaths[p] = true
	}

	// 1. Extract imports from the target file.
	imports := idx.extractImports(relPath, indexedPaths)

	// 2. Find importers — files that import the target.
	importers := idx.findImporters(relPath, indexedPaths)

	// 3. Find associated test files.
	testFiles := idx.findTestFiles(relPath, indexedPaths)

	return &RelatedResult{
		File:      relPath,
		Imports:   imports,
		Importers: importers,
		TestFiles: testFiles,
	}, nil
}

// extractImports reads the target file and extracts import paths, resolving them
// to indexed file paths.
func (idx *Index) extractImports(relPath string, indexedPaths map[string]bool) []string {
	ext := filepath.Ext(relPath)
	if !importableExts[ext] {
		return []string{}
	}

	absPath := filepath.Join(idx.Root, relPath)
	f, err := openTextFile(absPath)
	if err != nil || f == nil {
		return []string{}
	}
	defer f.Close()

	var rawImports []string
	scanner := bufio.NewScanner(f)

	switch ext {
	case ".go":
		rawImports = extractGoImports(scanner)
	case ".js", ".jsx", ".ts", ".tsx":
		rawImports = extractJSImports(scanner)
	case ".py":
		rawImports = extractPyImports(scanner)
	}

	// Resolve raw imports to indexed file paths.
	fileDir := filepath.Dir(relPath)
	var resolved []string
	seen := make(map[string]bool)

	for _, imp := range rawImports {
		for _, candidate := range resolveImport(imp, ext, fileDir, indexedPaths) {
			if !seen[candidate] && candidate != relPath {
				seen[candidate] = true
				resolved = append(resolved, candidate)
			}
		}
	}

	if resolved == nil {
		resolved = []string{}
	}
	return resolved
}

// extractGoImports parses Go import statements from a scanner.
func extractGoImports(scanner *bufio.Scanner) []string {
	var imports []string
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if inBlock {
			if goImportEnd.MatchString(line) {
				inBlock = false
				continue
			}
			if m := goImportLine.FindStringSubmatch(line); m != nil {
				imports = append(imports, m[1])
			}
			continue
		}

		if goImportBlock.MatchString(line) {
			inBlock = true
			continue
		}

		if m := goImportSingle.FindStringSubmatch(line); m != nil {
			imports = append(imports, m[1])
		}
	}
	return imports
}

// extractJSImports parses JS/TS import and require statements.
func extractJSImports(scanner *bufio.Scanner) []string {
	var imports []string
	for scanner.Scan() {
		line := scanner.Text()
		if m := jsImportFrom.FindStringSubmatch(line); m != nil {
			imports = append(imports, m[1])
		}
		if m := jsRequire.FindStringSubmatch(line); m != nil {
			imports = append(imports, m[1])
		}
	}
	return imports
}

// extractPyImports parses Python import statements.
func extractPyImports(scanner *bufio.Scanner) []string {
	var imports []string
	for scanner.Scan() {
		line := scanner.Text()
		if m := pyFromImport.FindStringSubmatch(line); m != nil {
			imports = append(imports, m[1])
		} else if m := pyImport.FindStringSubmatch(line); m != nil {
			imports = append(imports, m[1])
		}
	}
	return imports
}

// resolveImport tries to resolve a raw import string to indexed file paths.
func resolveImport(imp string, ext string, fileDir string, indexedPaths map[string]bool) []string {
	switch ext {
	case ".go":
		return resolveGoImport(imp, indexedPaths)
	case ".js", ".jsx", ".ts", ".tsx":
		return resolveJSImport(imp, fileDir, indexedPaths)
	case ".py":
		return resolvePyImport(imp, fileDir, indexedPaths)
	}
	return nil
}

// resolveGoImport matches a Go import path against indexed files.
// Only local packages are matched (import paths that are subpaths within the project).
func resolveGoImport(imp string, indexedPaths map[string]bool) []string {
	var matches []string
	// Try matching each suffix of the import path against indexed directories.
	// For example, "github.com/user/project/utils" should match the "utils" directory.
	parts := strings.Split(imp, "/")
	for p := range indexedPaths {
		if filepath.Ext(p) != ".go" || strings.HasSuffix(p, "_test.go") {
			continue
		}
		dir := filepath.Dir(p)
		// Try each suffix of the import path.
		for i := 0; i < len(parts); i++ {
			suffix := strings.Join(parts[i:], "/")
			if dir == suffix {
				matches = append(matches, p)
				break
			}
		}
	}
	return matches
}

// resolveJSImport resolves a JS/TS relative import to indexed files.
func resolveJSImport(imp string, fileDir string, indexedPaths map[string]bool) []string {
	// Only resolve relative imports.
	if !strings.HasPrefix(imp, ".") {
		return nil
	}

	resolved := filepath.Join(fileDir, imp)
	resolved = filepath.Clean(resolved)

	// Try exact match first.
	if indexedPaths[resolved] {
		return []string{resolved}
	}

	// Try adding extensions.
	jsExts := []string{".js", ".jsx", ".ts", ".tsx"}
	for _, ext := range jsExts {
		candidate := resolved + ext
		if indexedPaths[candidate] {
			return []string{candidate}
		}
	}

	// Try index files.
	for _, ext := range jsExts {
		candidate := filepath.Join(resolved, "index"+ext)
		if indexedPaths[candidate] {
			return []string{candidate}
		}
	}

	return nil
}

// resolvePyImport resolves a Python import to indexed files.
func resolvePyImport(imp string, fileDir string, indexedPaths map[string]bool) []string {
	// Convert dots to path separators.
	parts := strings.Split(imp, ".")

	// Try as relative import from file directory.
	relPath := filepath.Join(append([]string{fileDir}, parts...)...) + ".py"
	if indexedPaths[relPath] {
		return []string{relPath}
	}

	// Try as absolute import from project root.
	absPath := filepath.Join(parts...) + ".py"
	if indexedPaths[absPath] {
		return []string{absPath}
	}

	// Try as package directory with __init__.py.
	pkgInit := filepath.Join(parts...) + string(filepath.Separator) + "__init__.py"
	pkgInit = filepath.Clean(pkgInit)
	if indexedPaths[pkgInit] {
		return []string{pkgInit}
	}

	return nil
}

// findImporters scans all importable files to find ones that import the target.
func (idx *Index) findImporters(targetPath string, indexedPaths map[string]bool) []string {
	var importers []string

	for p := range indexedPaths {
		ext := filepath.Ext(p)
		if !importableExts[ext] || p == targetPath {
			continue
		}

		imports := idx.extractImports(p, indexedPaths)
		for _, imp := range imports {
			if imp == targetPath {
				importers = append(importers, p)
				break
			}
		}
	}

	if importers == nil {
		importers = []string{}
	}
	return importers
}

// findTestFiles looks for test files associated with the target file using
// language-specific naming conventions.
func (idx *Index) findTestFiles(relPath string, indexedPaths map[string]bool) []string {
	ext := filepath.Ext(relPath)
	dir := filepath.Dir(relPath)
	base := strings.TrimSuffix(filepath.Base(relPath), ext)

	var testFiles []string
	seen := make(map[string]bool)

	addIfExists := func(candidate string) {
		candidate = filepath.Clean(candidate)
		if indexedPaths[candidate] && !seen[candidate] && candidate != relPath {
			seen[candidate] = true
			testFiles = append(testFiles, candidate)
		}
	}

	switch ext {
	case ".go":
		// Go: <name>_test.go in the same directory.
		addIfExists(filepath.Join(dir, base+"_test.go"))

	case ".js", ".jsx", ".ts", ".tsx":
		// JS/TS: <name>.test.{ext} and <name>.spec.{ext} in same directory.
		jsExts := []string{".js", ".jsx", ".ts", ".tsx"}
		for _, e := range jsExts {
			addIfExists(filepath.Join(dir, base+".test"+e))
			addIfExists(filepath.Join(dir, base+".spec"+e))
		}
		// Also check __tests__/<name>.{ext}
		for _, e := range jsExts {
			addIfExists(filepath.Join(dir, "__tests__", base+e))
			addIfExists(filepath.Join(dir, "__tests__", base+".test"+e))
		}

	case ".py":
		// Python: test_<name>.py and <name>_test.py in same directory.
		addIfExists(filepath.Join(dir, "test_"+base+".py"))
		addIfExists(filepath.Join(dir, base+"_test.py"))
		// Also check tests/ directory.
		addIfExists(filepath.Join(dir, "tests", "test_"+base+".py"))
		addIfExists(filepath.Join(dir, "tests", base+"_test.py"))
	}

	if testFiles == nil {
		testFiles = []string{}
	}
	return testFiles
}

// FormatRelated returns a human-readable text rendering of the related result.
func FormatRelated(r *RelatedResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Related files for %s:\n", r.File))

	if len(r.Imports) == 0 && len(r.Importers) == 0 && len(r.TestFiles) == 0 {
		b.WriteString("\n  No related files found\n")
		return b.String()
	}

	if len(r.Imports) > 0 {
		b.WriteString(fmt.Sprintf("\nImports (%d):\n", len(r.Imports)))
		for _, p := range r.Imports {
			b.WriteString(fmt.Sprintf("  %s\n", p))
		}
	}

	if len(r.Importers) > 0 {
		b.WriteString(fmt.Sprintf("\nImported by (%d):\n", len(r.Importers)))
		for _, p := range r.Importers {
			b.WriteString(fmt.Sprintf("  %s\n", p))
		}
	}

	if len(r.TestFiles) > 0 {
		b.WriteString(fmt.Sprintf("\nTest files (%d):\n", len(r.TestFiles)))
		for _, p := range r.TestFiles {
			b.WriteString(fmt.Sprintf("  %s\n", p))
		}
	}

	return b.String()
}
