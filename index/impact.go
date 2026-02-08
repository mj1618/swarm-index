package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matt/swarm-index/parsers"
)

// ImpactTarget describes the symbol or file being analyzed.
type ImpactTarget struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
	Kind string `json:"kind"` // "func", "type", "file"
}

// ImpactRef is a single reference site within a layer.
type ImpactRef struct {
	File            string `json:"file"`
	Line            int    `json:"line"`
	Content         string `json:"content"`
	EnclosingSymbol string `json:"enclosingSymbol,omitempty"`
}

// ImpactLayer groups references at a given depth.
type ImpactLayer struct {
	Depth int         `json:"depth"`
	Label string      `json:"label"`
	Refs  []ImpactRef `json:"refs"`
}

// ImpactSummary gives aggregate counts.
type ImpactSummary struct {
	TotalFiles    int `json:"totalFiles"`
	TotalRefSites int `json:"totalRefSites"`
	MaxDepth      int `json:"maxDepthReached"`
}

// ImpactResult is the full output of an impact analysis.
type ImpactResult struct {
	Target  ImpactTarget  `json:"target"`
	Layers  []ImpactLayer `json:"layers"`
	Summary ImpactSummary `json:"summary"`
}

// Impact performs transitive blast-radius analysis for a symbol or file.
// If target contains '/' or '.', it is treated as a file path; otherwise as a symbol name.
func (idx *Index) Impact(target string, maxDepth, maxResults int) (*ImpactResult, error) {
	isFile := strings.Contains(target, "/") || strings.Contains(target, ".")
	if isFile {
		return idx.impactFile(target, maxDepth, maxResults)
	}
	return idx.impactSymbol(target, maxDepth, maxResults)
}

// impactSymbol traces transitive references for a symbol.
func (idx *Index) impactSymbol(symbol string, maxDepth, maxResults int) (*ImpactResult, error) {
	// Find the symbol definition.
	refsResult, err := idx.Refs(symbol, maxResults)
	if err != nil {
		return nil, err
	}

	target := ImpactTarget{
		Name: symbol,
		Kind: "symbol",
	}
	if refsResult.Definition != nil {
		target.File = refsResult.Definition.Path
		target.Line = refsResult.Definition.Line
		target.Kind = "func" // best guess from definition pattern
	}

	// Depth 1: direct references.
	var layer1Refs []ImpactRef
	visited := map[string]bool{symbol: true}
	totalRefs := 0

	for _, ref := range refsResult.References {
		if totalRefs >= maxResults {
			break
		}
		enclosing := idx.findEnclosingSymbol(ref.Path, ref.Line)
		layer1Refs = append(layer1Refs, ImpactRef{
			File:            ref.Path,
			Line:            ref.Line,
			Content:         ref.Content,
			EnclosingSymbol: enclosing,
		})
		totalRefs++
	}

	layers := []ImpactLayer{
		{Depth: 1, Label: "direct references", Refs: layer1Refs},
	}

	// Depths 2..maxDepth: transitive dependents.
	prevRefs := layer1Refs
	for depth := 2; depth <= maxDepth; depth++ {
		if totalRefs >= maxResults {
			break
		}

		// Collect unique enclosing symbols from previous layer.
		var nextSymbols []string
		for _, ref := range prevRefs {
			if ref.EnclosingSymbol != "" && !visited[ref.EnclosingSymbol] {
				visited[ref.EnclosingSymbol] = true
				nextSymbols = append(nextSymbols, ref.EnclosingSymbol)
			}
		}

		if len(nextSymbols) == 0 {
			break
		}

		var layerRefs []ImpactRef
		for _, sym := range nextSymbols {
			if totalRefs >= maxResults {
				break
			}
			symRefs, err := idx.Refs(sym, maxResults-totalRefs)
			if err != nil {
				continue
			}
			for _, ref := range symRefs.References {
				if totalRefs >= maxResults {
					break
				}
				enclosing := idx.findEnclosingSymbol(ref.Path, ref.Line)
				layerRefs = append(layerRefs, ImpactRef{
					File:            ref.Path,
					Line:            ref.Line,
					Content:         ref.Content,
					EnclosingSymbol: enclosing,
				})
				totalRefs++
			}
		}

		if len(layerRefs) == 0 {
			break
		}

		label := "transitive dependents"
		if depth > 2 {
			label = fmt.Sprintf("depth-%d dependents", depth)
		}
		layers = append(layers, ImpactLayer{Depth: depth, Label: label, Refs: layerRefs})
		prevRefs = layerRefs
	}

	return &ImpactResult{
		Target:  target,
		Layers:  layers,
		Summary: computeImpactSummary(layers),
	}, nil
}

// impactFile traces transitive importers for a file.
func (idx *Index) impactFile(filePath string, maxDepth, maxResults int) (*ImpactResult, error) {
	// Normalize path.
	relPath := filePath
	if filepath.IsAbs(filePath) {
		var err error
		relPath, err = filepath.Rel(idx.Root, filePath)
		if err != nil {
			return nil, fmt.Errorf("cannot make path relative: %w", err)
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

	target := ImpactTarget{
		Name: filepath.Base(relPath),
		File: relPath,
		Kind: "file",
	}

	visited := map[string]bool{relPath: true}
	totalRefs := 0

	// Depth 1: direct importers.
	related, err := idx.Related(relPath)
	if err != nil {
		return nil, err
	}

	var layer1Refs []ImpactRef
	for _, imp := range related.Importers {
		if totalRefs >= maxResults {
			break
		}
		layer1Refs = append(layer1Refs, ImpactRef{
			File: imp,
		})
		visited[imp] = true
		totalRefs++
	}

	layers := []ImpactLayer{
		{Depth: 1, Label: "direct importers", Refs: layer1Refs},
	}

	// Depths 2..maxDepth.
	prevFiles := related.Importers
	for depth := 2; depth <= maxDepth; depth++ {
		if totalRefs >= maxResults {
			break
		}

		var layerRefs []ImpactRef
		var nextFiles []string
		for _, f := range prevFiles {
			if totalRefs >= maxResults {
				break
			}
			rel, err := idx.Related(f)
			if err != nil {
				continue
			}
			for _, imp := range rel.Importers {
				if totalRefs >= maxResults {
					break
				}
				if visited[imp] {
					continue
				}
				visited[imp] = true
				layerRefs = append(layerRefs, ImpactRef{
					File: imp,
				})
				nextFiles = append(nextFiles, imp)
				totalRefs++
			}
		}

		if len(layerRefs) == 0 {
			break
		}

		label := "transitive importers"
		if depth > 2 {
			label = fmt.Sprintf("depth-%d importers", depth)
		}
		layers = append(layers, ImpactLayer{Depth: depth, Label: label, Refs: layerRefs})
		prevFiles = nextFiles
	}

	return &ImpactResult{
		Target:  target,
		Layers:  layers,
		Summary: computeImpactSummary(layers),
	}, nil
}

// computeImpactSummary aggregates file and reference counts across layers.
func computeImpactSummary(layers []ImpactLayer) ImpactSummary {
	fileSet := map[string]bool{}
	totalSites := 0
	maxDepthReached := 0
	for _, layer := range layers {
		if layer.Depth > maxDepthReached {
			maxDepthReached = layer.Depth
		}
		for _, ref := range layer.Refs {
			fileSet[ref.File] = true
			totalSites++
		}
	}
	return ImpactSummary{
		TotalFiles:    len(fileSet),
		TotalRefSites: totalSites,
		MaxDepth:      maxDepthReached,
	}
}

// findEnclosingSymbol determines which symbol's line range contains the given line.
func (idx *Index) findEnclosingSymbol(relPath string, line int) string {
	ext := filepath.Ext(relPath)
	p := parsers.ForExtension(ext)
	if p == nil {
		return ""
	}

	absPath := filepath.Join(idx.Root, relPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}

	symbols, err := p.Parse(absPath, content)
	if err != nil {
		return ""
	}

	// Find the innermost symbol whose range contains the line.
	var best string
	bestLine := 0
	for _, s := range symbols {
		if s.Line <= line && (s.EndLine == 0 || s.EndLine >= line) {
			if s.Line > bestLine {
				best = s.Name
				bestLine = s.Line
			}
		}
	}
	return best
}

// FormatImpact returns a human-readable text rendering of the impact result.
func FormatImpact(r *ImpactResult) string {
	var b strings.Builder

	if r.Target.Kind == "file" {
		b.WriteString(fmt.Sprintf("Impact analysis for file %q\n", r.Target.File))
	} else if r.Target.File != "" {
		b.WriteString(fmt.Sprintf("Impact analysis for symbol %q (%s:%d)\n", r.Target.Name, r.Target.File, r.Target.Line))
	} else {
		b.WriteString(fmt.Sprintf("Impact analysis for %q\n", r.Target.Name))
	}

	if len(r.Layers) == 0 || (len(r.Layers) == 1 && len(r.Layers[0].Refs) == 0) {
		b.WriteString("\n  No dependents found\n")
		return b.String()
	}

	for _, layer := range r.Layers {
		if len(layer.Refs) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("\nDepth %d — %s (%d):\n", layer.Depth, layer.Label, len(layer.Refs)))
		for _, ref := range layer.Refs {
			if ref.Content != "" {
				b.WriteString(fmt.Sprintf("  %s:%d  %s\n", ref.File, ref.Line, ref.Content))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", ref.File))
			}
		}
	}

	b.WriteString(fmt.Sprintf("\nTotal blast radius: %d files, %d reference sites\n", r.Summary.TotalFiles, r.Summary.TotalRefSites))

	return b.String()
}
