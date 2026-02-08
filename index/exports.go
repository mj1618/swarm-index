package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matt/swarm-index/parsers"
)

// ExportedSymbol represents a single exported symbol from a file.
type ExportedSymbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Signature string `json:"signature"`
}

// ExportsResult holds the result of an exports query.
type ExportsResult struct {
	Scope   string           `json:"scope"`
	Symbols []ExportedSymbol `json:"symbols"`
	Count   int              `json:"count"`
}

// Exports returns the public API surface for a given file or directory scope.
// If scope is a file, it returns exported symbols from that file.
// If scope is a directory, it returns exported symbols from all parseable files
// in that directory (non-recursive).
func (idx *Index) Exports(scope string) (*ExportsResult, error) {
	// Determine if scope matches a file or directory in the index.
	var filePaths []string

	// Normalize scope — remove trailing slash.
	scope = strings.TrimRight(scope, "/"+string(filepath.Separator))

	for _, e := range idx.Entries {
		if e.Kind != "file" {
			continue
		}
		if e.Path == scope {
			filePaths = append(filePaths, e.Path)
			break
		}
	}

	if len(filePaths) == 0 {
		// Check if scope matches a directory (package).
		for _, e := range idx.Entries {
			if e.Kind != "file" {
				continue
			}
			if filepath.Dir(e.Path) == scope {
				filePaths = append(filePaths, e.Path)
			}
		}
	}

	if len(filePaths) == 0 {
		return &ExportsResult{
			Scope:   scope,
			Symbols: []ExportedSymbol{},
			Count:   0,
		}, nil
	}

	sort.Strings(filePaths)

	var symbols []ExportedSymbol
	for _, relPath := range filePaths {
		absPath := filepath.Join(idx.Root, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		ext := filepath.Ext(relPath)
		p := parsers.ForExtension(ext)
		if p == nil {
			continue
		}
		parsed, err := p.Parse(absPath, content)
		if err != nil {
			continue
		}
		for _, sym := range parsed {
			if sym.Exported {
				symbols = append(symbols, ExportedSymbol{
					Name:      sym.Name,
					Kind:      sym.Kind,
					Path:      relPath,
					Line:      sym.Line,
					Signature: sym.Signature,
				})
			}
		}
	}

	return &ExportsResult{
		Scope:   scope,
		Symbols: symbols,
		Count:   len(symbols),
	}, nil
}

// FormatExports returns a human-readable text rendering of the exports result.
func FormatExports(r *ExportsResult) string {
	var b strings.Builder

	if len(r.Symbols) == 0 {
		b.WriteString(fmt.Sprintf("No exported symbols found for %s\n", r.Scope))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Exports for %s:\n", r.Scope))

	// Group by path for directory scopes.
	grouped := make(map[string][]ExportedSymbol)
	var paths []string
	for _, s := range r.Symbols {
		if _, ok := grouped[s.Path]; !ok {
			paths = append(paths, s.Path)
		}
		grouped[s.Path] = append(grouped[s.Path], s)
	}

	multiFile := len(paths) > 1

	for _, path := range paths {
		syms := grouped[path]
		if multiFile {
			b.WriteString(fmt.Sprintf("\n  %s:\n", path))
		}

		for _, s := range syms {
			indent := "  "
			if multiFile {
				indent = "    "
			}
			b.WriteString(fmt.Sprintf("%s%-6s %-50s :%d\n", indent, s.Kind, s.Signature, s.Line))
		}
	}

	b.WriteString(fmt.Sprintf("\n%d exported symbols\n", r.Count))

	return b.String()
}
