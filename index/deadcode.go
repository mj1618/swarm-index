package index

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mj1618/swarm-index/parsers"
)

// DeadCodeCandidate represents a symbol with zero external references.
type DeadCodeCandidate struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Path       string `json:"path"`
	Line       int    `json:"line"`
	Signature  string `json:"signature"`
	Exported   bool   `json:"exported"`
	References int    `json:"references"`
}

// DeadCodeResult holds the result of a dead-code analysis.
type DeadCodeResult struct {
	TotalCandidates int                 `json:"totalCandidates"`
	Candidates      []DeadCodeCandidate `json:"candidates"`
}

// isExcludedSymbol returns true if the symbol should be excluded from dead-code analysis.
func isExcludedSymbol(name string) bool {
	// Skip main and init (entry points / implicit calls).
	if name == "main" || name == "init" {
		return true
	}
	// Skip Go test entry points.
	if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") || strings.HasPrefix(name, "Example") {
		return true
	}
	return false
}

// DeadCode finds exported symbols that have zero references outside their
// definition file/line. kind filters by symbol kind (empty = all). pathPrefix
// limits analysis to files whose path starts with the given prefix. At most
// max candidates are returned.
func (idx *Index) DeadCode(kind string, pathPrefix string, max int) (*DeadCodeResult, error) {
	kindLower := strings.ToLower(kind)
	allPaths := idx.FilePaths()

	// Collect all exported symbols from parseable, non-test files.
	type symbolInfo struct {
		name      string
		kind      string
		path      string
		line      int
		signature string
		exported  bool
	}
	var symbols []symbolInfo

	for _, relPath := range allPaths {
		if testFilePattern.MatchString(relPath) {
			continue
		}
		if pathPrefix != "" && !strings.HasPrefix(relPath, pathPrefix) {
			continue
		}

		ext := filepath.Ext(relPath)
		p := parsers.ForExtension(ext)
		if p == nil {
			continue
		}

		absPath := filepath.Join(idx.Root, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		parsed, err := p.Parse(absPath, content)
		if err != nil {
			continue
		}

		for _, sym := range parsed {
			if !sym.Exported {
				continue
			}
			if isExcludedSymbol(sym.Name) {
				continue
			}
			if kindLower != "" && strings.ToLower(sym.Kind) != kindLower {
				continue
			}
			symbols = append(symbols, symbolInfo{
				name:      sym.Name,
				kind:      sym.Kind,
				path:      relPath,
				line:      sym.Line,
				signature: sym.Signature,
				exported:  sym.Exported,
			})
		}
	}

	// For each symbol, search for references across all indexed files.
	candidates := []DeadCodeCandidate{}
	for _, sym := range symbols {
		wordRe, err := regexp.Compile(`\b` + regexp.QuoteMeta(sym.name) + `\b`)
		if err != nil {
			continue
		}

		refCount := countExternalRefs(idx, sym.path, sym.line, wordRe, allPaths)

		if refCount == 0 {
			candidates = append(candidates, DeadCodeCandidate{
				Name:       sym.name,
				Kind:       sym.kind,
				Path:       sym.path,
				Line:       sym.line,
				Signature:  sym.signature,
				Exported:   sym.exported,
				References: 0,
			})
		}
	}

	// Sort by file path, then line number.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Path != candidates[j].Path {
			return candidates[i].Path < candidates[j].Path
		}
		return candidates[i].Line < candidates[j].Line
	})

	total := len(candidates)
	if max > 0 && len(candidates) > max {
		candidates = candidates[:max]
	}

	return &DeadCodeResult{
		TotalCandidates: total,
		Candidates:      candidates,
	}, nil
}

// countExternalRefs counts how many references to a symbol exist outside its
// definition (different file, or same file but different line). Returns 1 as
// soon as any external reference is found (short-circuit).
func countExternalRefs(idx *Index, defPath string, defLine int, wordRe *regexp.Regexp, paths []string) int {
	for _, p := range paths {
		if hasRefInFile(filepath.Join(idx.Root, p), p, defPath, defLine, wordRe) {
			return 1
		}
	}
	return 0
}

// hasRefInFile returns true if the file contains a reference to the symbol
// outside of its definition line.
func hasRefInFile(fullPath, relPath, defPath string, defLine int, wordRe *regexp.Regexp) bool {
	f, err := openTextFile(fullPath)
	if err != nil || f == nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if !wordRe.MatchString(scanner.Text()) {
			continue
		}
		if relPath == defPath && lineNum == defLine {
			continue
		}
		return true
	}
	return false
}

// FormatDeadCode returns a human-readable text rendering of the dead-code result.
func FormatDeadCode(r *DeadCodeResult) string {
	var b strings.Builder

	if len(r.Candidates) == 0 {
		b.WriteString("No dead code candidates found\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Dead code candidates (%d found):\n", r.TotalCandidates))

	// Group by path.
	grouped := make(map[string][]DeadCodeCandidate)
	var paths []string
	for _, c := range r.Candidates {
		if _, ok := grouped[c.Path]; !ok {
			paths = append(paths, c.Path)
		}
		grouped[c.Path] = append(grouped[c.Path], c)
	}

	for _, path := range paths {
		b.WriteString(fmt.Sprintf("\n  %s:\n", path))
		for _, c := range grouped[path] {
			b.WriteString(fmt.Sprintf("    %-6s %-40s :%d    (0 references)\n", c.Kind, c.Name, c.Line))
		}
	}

	if r.TotalCandidates > len(r.Candidates) {
		b.WriteString(fmt.Sprintf("\n... %d more (use --max to see more)\n", r.TotalCandidates-len(r.Candidates)))
	}

	return b.String()
}
