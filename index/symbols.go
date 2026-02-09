package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mj1618/swarm-index/parsers"
)

// SymbolMatch represents a single symbol found across the project.
type SymbolMatch struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Signature string `json:"signature"`
	Exported  bool   `json:"exported"`
}

// SymbolsResult holds the result of a project-wide symbol search.
type SymbolsResult struct {
	Query   string        `json:"query"`
	Matches []SymbolMatch `json:"matches"`
	Total   int           `json:"total"`
}

// Symbols searches all parseable files in the index for symbols matching
// the query by name (case-insensitive substring). If kind is non-empty,
// only symbols of that kind are returned. Results are sorted: exact name
// matches first, then prefix matches, then substring matches. At most max
// results are returned.
func (idx *Index) Symbols(query string, kind string, max int) (*SymbolsResult, error) {
	queryLower := strings.ToLower(query)
	kindLower := strings.ToLower(kind)

	var matches []SymbolMatch

	for _, relPath := range idx.FilePaths() {
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

		symbols, err := p.Parse(absPath, content)
		if err != nil {
			continue
		}

		for _, sym := range symbols {
			nameLower := strings.ToLower(sym.Name)
			if !strings.Contains(nameLower, queryLower) {
				continue
			}
			if kindLower != "" && strings.ToLower(sym.Kind) != kindLower {
				continue
			}
			matches = append(matches, SymbolMatch{
				Name:      sym.Name,
				Kind:      sym.Kind,
				Path:      relPath,
				Line:      sym.Line,
				Signature: sym.Signature,
				Exported:  sym.Exported,
			})
		}
	}

	// Sort: exact > prefix > substring, then alphabetically by name.
	sort.Slice(matches, func(i, j int) bool {
		ri := matchRank(matches[i].Name, queryLower)
		rj := matchRank(matches[j].Name, queryLower)
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(matches[i].Name) < strings.ToLower(matches[j].Name)
	})

	total := len(matches)
	if max > 0 && len(matches) > max {
		matches = matches[:max]
	}

	return &SymbolsResult{
		Query:   query,
		Matches: matches,
		Total:   total,
	}, nil
}

// matchRank returns 0 for exact match, 1 for prefix match, 2 for substring.
func matchRank(name string, queryLower string) int {
	nameLower := strings.ToLower(name)
	if nameLower == queryLower {
		return 0
	}
	if strings.HasPrefix(nameLower, queryLower) {
		return 1
	}
	return 2
}

// FormatSymbols returns a human-readable text rendering of the symbols result.
func FormatSymbols(r *SymbolsResult) string {
	var b strings.Builder

	if len(r.Matches) == 0 {
		b.WriteString(fmt.Sprintf("No symbols matching %q\n", r.Query))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Symbols matching %q (%d found):\n\n", r.Query, r.Total))

	for _, m := range r.Matches {
		b.WriteString(fmt.Sprintf("  %-10s %-40s %s:%d\n", m.Kind, m.Name, m.Path, m.Line))
	}

	if r.Total > len(r.Matches) {
		b.WriteString(fmt.Sprintf("\n... %d more (use --max to see more)\n", r.Total-len(r.Matches)))
	}

	return b.String()
}
