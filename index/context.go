package index

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mj1618/swarm-index/parsers"
)

// extractFileImports reads the file content and extracts import statements
// as raw strings (the import paths/modules, not resolved file paths).
// It reuses the per-language extractors defined in related.go.
func extractFileImports(content []byte, ext string) []string {
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	var imports []string
	switch ext {
	case ".go":
		imports = extractGoImports(scanner)
	case ".js", ".jsx", ".ts", ".tsx":
		imports = extractJSImports(scanner)
	case ".py":
		imports = extractPyImports(scanner)
	}

	if imports == nil {
		imports = []string{}
	}
	return imports
}

// ContextResult holds the full context of a symbol: imports, doc comment, and body.
type ContextResult struct {
	File       string   `json:"file"`
	Symbol     string   `json:"symbol"`
	Kind       string   `json:"kind"`
	Line       int      `json:"line"`
	EndLine    int      `json:"endLine"`
	Signature  string   `json:"signature"`
	Imports    []string `json:"imports"`
	DocComment string   `json:"docComment"`
	Body       string   `json:"body"`
}

// Context extracts the full definition context for a symbol in a file:
// imports, doc comment, and the complete definition body.
func Context(filePath string, symbolName string) (*ContextResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	ext := filepath.Ext(filePath)
	p := parsers.ForExtension(ext)
	if p == nil {
		return nil, fmt.Errorf("no parser available for %s files", ext)
	}

	symbols, err := p.Parse(filePath, content)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	// Find the matching symbol — prefer top-level (no parent) if multiple match.
	var match *parsers.Symbol
	for i := range symbols {
		if symbols[i].Name == symbolName {
			if match == nil || (match.Parent != "" && symbols[i].Parent == "") {
				match = &symbols[i]
			}
		}
	}
	if match == nil {
		return nil, fmt.Errorf("symbol %q not found in %s", symbolName, filePath)
	}

	lines := strings.Split(string(content), "\n")

	// Extract body (Line and EndLine are 1-indexed).
	startIdx := match.Line - 1
	endIdx := match.EndLine - 1
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(lines) {
		endIdx = len(lines) - 1
	}
	if match.EndLine == 0 {
		// If EndLine is not set, use just the start line.
		endIdx = startIdx
	}
	body := strings.Join(lines[startIdx:endIdx+1], "\n")

	// Extract doc comment: contiguous comment lines immediately before the symbol.
	docComment := extractDocComment(lines, startIdx, ext)

	// Extract imports.
	imports := extractFileImports(content, ext)

	return &ContextResult{
		File:       filePath,
		Symbol:     match.Name,
		Kind:       match.Kind,
		Line:       match.Line,
		EndLine:    match.EndLine,
		Signature:  match.Signature,
		Imports:    imports,
		DocComment: docComment,
		Body:       body,
	}, nil
}

// extractDocComment walks backwards from the line before the symbol's definition,
// collecting contiguous comment lines.
func extractDocComment(lines []string, symbolLineIdx int, ext string) string {
	if symbolLineIdx <= 0 {
		return ""
	}

	var commentLines []string
	for i := symbolLineIdx - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if isCommentLine(trimmed, ext) {
			commentLines = append(commentLines, lines[i])
		} else {
			break
		}
	}

	if len(commentLines) == 0 {
		return ""
	}

	// Reverse since we collected bottom-up.
	for i, j := 0, len(commentLines)-1; i < j; i, j = i+1, j-1 {
		commentLines[i], commentLines[j] = commentLines[j], commentLines[i]
	}

	return strings.Join(commentLines, "\n")
}

// isCommentLine checks if a trimmed line is a comment in the given language.
func isCommentLine(trimmed string, ext string) bool {
	if trimmed == "" {
		return false
	}
	switch ext {
	case ".go":
		return strings.HasPrefix(trimmed, "//")
	case ".py":
		return strings.HasPrefix(trimmed, "#")
	case ".js", ".jsx", ".ts", ".tsx":
		return strings.HasPrefix(trimmed, "//") ||
			strings.HasPrefix(trimmed, "/*") ||
			strings.HasPrefix(trimmed, "*")
	}
	return false
}

// FormatContext returns a human-readable text rendering of the context result.
func FormatContext(r *ContextResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("File: %s\n", r.File))

	if len(r.Imports) > 0 {
		b.WriteString("Imports:\n")
		for _, imp := range r.Imports {
			b.WriteString(fmt.Sprintf("  %s\n", imp))
		}
		b.WriteString("\n")
	}

	if r.DocComment != "" {
		b.WriteString(r.DocComment)
		b.WriteString("\n")
	}

	b.WriteString(r.Body)
	b.WriteString("\n")

	return b.String()
}
