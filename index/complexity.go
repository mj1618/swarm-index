package index

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/matt/swarm-index/parsers"
)

// FunctionComplexity holds complexity metrics for a single function/method.
type FunctionComplexity struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	Line       int    `json:"line"`
	EndLine    int    `json:"endLine"`
	Complexity int    `json:"complexity"`
	Lines      int    `json:"lines"`
	MaxDepth   int    `json:"maxDepth"`
	Params     int    `json:"params"`
	Signature  string `json:"signature"`
}

// ComplexityResult holds the full complexity analysis result.
type ComplexityResult struct {
	Functions           []FunctionComplexity `json:"functions"`
	TotalFunctions      int                  `json:"totalFunctions"`
	AvgComplexity       float64              `json:"avgComplexity"`
	MaxComplexity       int                  `json:"maxComplexity"`
	HighComplexityCount int                  `json:"highComplexityCount"`
}

// Complexity analyzes all parseable files (or a single file) and returns
// complexity metrics for each function, sorted by complexity descending.
func (idx *Index) Complexity(file string, maxResults int, minComplexity int) (*ComplexityResult, error) {
	var allFuncs []FunctionComplexity

	if file != "" {
		funcs, err := analyzeFileComplexity(file, file)
		if err != nil {
			return nil, err
		}
		allFuncs = append(allFuncs, funcs...)
	} else {
		for _, relPath := range idx.FilePaths() {
			ext := filepath.Ext(relPath)
			p := parsers.ForExtension(ext)
			if p == nil {
				continue
			}
			absPath := filepath.Join(idx.Root, relPath)
			funcs, err := analyzeFileComplexity(absPath, relPath)
			if err != nil {
				continue
			}
			allFuncs = append(allFuncs, funcs...)
		}
	}

	// Sort by complexity descending, then by path+line for stability.
	sort.Slice(allFuncs, func(i, j int) bool {
		if allFuncs[i].Complexity != allFuncs[j].Complexity {
			return allFuncs[i].Complexity > allFuncs[j].Complexity
		}
		if allFuncs[i].Path != allFuncs[j].Path {
			return allFuncs[i].Path < allFuncs[j].Path
		}
		return allFuncs[i].Line < allFuncs[j].Line
	})

	// Compute summary stats before filtering.
	totalFunctions := len(allFuncs)
	var sumComplexity int
	var maxC int
	var highCount int
	for _, f := range allFuncs {
		sumComplexity += f.Complexity
		if f.Complexity > maxC {
			maxC = f.Complexity
		}
		if f.Complexity >= 10 {
			highCount++
		}
	}
	var avgC float64
	if totalFunctions > 0 {
		avgC = float64(sumComplexity) / float64(totalFunctions)
	}

	// Filter by minimum complexity.
	if minComplexity > 0 {
		var filtered []FunctionComplexity
		for _, f := range allFuncs {
			if f.Complexity >= minComplexity {
				filtered = append(filtered, f)
			}
		}
		allFuncs = filtered
	}

	// Limit results.
	if maxResults > 0 && len(allFuncs) > maxResults {
		allFuncs = allFuncs[:maxResults]
	}

	return &ComplexityResult{
		Functions:           allFuncs,
		TotalFunctions:      totalFunctions,
		AvgComplexity:       avgC,
		MaxComplexity:       maxC,
		HighComplexityCount: highCount,
	}, nil
}

// analyzeFileComplexity computes complexity for all functions in a file.
func analyzeFileComplexity(absPath, displayPath string) ([]FunctionComplexity, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(absPath)
	switch ext {
	case ".go":
		return analyzeGoComplexity(absPath, displayPath, content)
	case ".py":
		return analyzeHeuristicComplexity(displayPath, content, pythonBranchPatterns)
	case ".js", ".jsx", ".ts", ".tsx":
		return analyzeHeuristicComplexity(displayPath, content, jsBranchPatterns)
	default:
		return nil, nil
	}
}

// analyzeGoComplexity uses go/ast for accurate complexity metrics.
func analyzeGoComplexity(absPath, displayPath string, content []byte) ([]FunctionComplexity, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, content, 0)
	if err != nil {
		return nil, err
	}

	var results []FunctionComplexity

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		complexity := 1 // base complexity
		maxDepth := 0
		complexity += countGoComplexity(fn.Body, 0, &maxDepth)

		params := 0
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				if len(field.Names) == 0 {
					params++
				} else {
					params += len(field.Names)
				}
			}
		}

		name := fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recvType := goReceiverName(fn.Recv.List[0].Type)
			name = recvType + "." + name
		}

		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line

		results = append(results, FunctionComplexity{
			Path:       displayPath,
			Name:       name,
			Line:       startLine,
			EndLine:    endLine,
			Complexity: complexity,
			Lines:      endLine - startLine + 1,
			MaxDepth:   maxDepth,
			Params:     params,
			Signature:  goFuncSignature(fn),
		})
	}

	return results, nil
}

// countGoComplexity walks AST nodes and counts branching constructs.
func countGoComplexity(node ast.Node, depth int, maxDepth *int) int {
	if depth > *maxDepth {
		*maxDepth = depth
	}

	complexity := 0

	// Track which nodes increased depth so we can decrement on exit.
	var nestingStack []bool
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			// Leaving a node: pop the nesting stack and decrement if needed.
			if len(nestingStack) > 0 {
				if nestingStack[len(nestingStack)-1] {
					depth--
				}
				nestingStack = nestingStack[:len(nestingStack)-1]
			}
			return false
		}
		isNesting := false
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
			isNesting = true
		case *ast.ForStmt, *ast.RangeStmt:
			complexity++
			isNesting = true
		case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			complexity++
			isNesting = true
		case *ast.CaseClause, *ast.CommClause:
			complexity++
		case *ast.BinaryExpr:
			bin := n.(*ast.BinaryExpr)
			if bin.Op == token.LAND || bin.Op == token.LOR {
				complexity++
			}
		}
		nestingStack = append(nestingStack, isNesting)
		if isNesting {
			depth++
			if depth > *maxDepth {
				*maxDepth = depth
			}
		}
		return true
	})

	return complexity
}

// goReceiverName extracts the type name from a receiver expression.
func goReceiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return goReceiverName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return goReceiverName(t.X)
	case *ast.IndexListExpr:
		return goReceiverName(t.X)
	}
	return ""
}

// goFuncSignature returns a short function signature string.
func goFuncSignature(fn *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		b.WriteString("(")
		b.WriteString(goReceiverName(fn.Recv.List[0].Type))
		b.WriteString(") ")
	}
	b.WriteString(fn.Name.Name)
	b.WriteString("()")
	return b.String()
}

// Heuristic-based complexity for Python/JS/TS files.

var (
	pythonBranchPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bif\b`),
		regexp.MustCompile(`\belif\b`),
		regexp.MustCompile(`\bfor\b`),
		regexp.MustCompile(`\bwhile\b`),
		regexp.MustCompile(`\bexcept\b`),
		regexp.MustCompile(`\band\b`),
		regexp.MustCompile(`\bor\b`),
	}

	jsBranchPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bif\s*\(`),
		regexp.MustCompile(`\belse\s+if\s*\(`),
		regexp.MustCompile(`\bfor\s*\(`),
		regexp.MustCompile(`\bwhile\s*\(`),
		regexp.MustCompile(`\bcase\b`),
		regexp.MustCompile(`\bcatch\s*\(`),
		regexp.MustCompile(`&&`),
		regexp.MustCompile(`\|\|`),
		regexp.MustCompile(`\?\s`),
	}
)

// analyzeHeuristicComplexity calculates complexity for non-Go files using
// the parsers package for function boundaries and regex for branch counting.
func analyzeHeuristicComplexity(displayPath string, content []byte, branchPatterns []*regexp.Regexp) ([]FunctionComplexity, error) {
	ext := filepath.Ext(displayPath)
	p := parsers.ForExtension(ext)
	if p == nil {
		return nil, nil
	}

	symbols, err := p.Parse(displayPath, content)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var results []FunctionComplexity

	for _, sym := range symbols {
		if sym.Kind != "func" && sym.Kind != "method" {
			continue
		}
		if sym.Line <= 0 || sym.EndLine <= 0 || sym.EndLine > len(lines) {
			continue
		}

		complexity := 1 // base complexity
		maxDepth := 0
		params := countSignatureParams(sym.Signature)

		// Analyze lines within the function body.
		for li := sym.Line; li < sym.EndLine && li <= len(lines); li++ {
			line := lines[li-1]
			trimmed := strings.TrimSpace(line)

			// Skip comments and empty lines.
			if trimmed == "" || strings.HasPrefix(trimmed, "//") ||
				strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") ||
				strings.HasPrefix(trimmed, "*") {
				continue
			}

			for _, pat := range branchPatterns {
				if pat.MatchString(trimmed) {
					complexity++
				}
			}

			// Track nesting depth via indentation relative to function start.
			if ext == ".py" {
				baseIndent := lineIndentCount(lines[sym.Line-1])
				indent := lineIndentCount(line)
				relativeDepth := 0
				if indent > baseIndent {
					relativeDepth = (indent - baseIndent) / 4
				}
				if relativeDepth > maxDepth {
					maxDepth = relativeDepth
				}
			} else {
				// For JS/TS, count brace nesting.
				depth := countBraceDepthInRange(lines, sym.Line-1, li-1)
				if depth > maxDepth {
					maxDepth = depth
				}
			}
		}

		lineCount := sym.EndLine - sym.Line + 1

		results = append(results, FunctionComplexity{
			Path:       displayPath,
			Name:       sym.Name,
			Line:       sym.Line,
			EndLine:    sym.EndLine,
			Complexity: complexity,
			Lines:      lineCount,
			MaxDepth:   maxDepth,
			Params:     params,
			Signature:  sym.Signature,
		})
	}

	return results, nil
}

// lineIndentCount returns the number of leading space/tab characters.
func lineIndentCount(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

// countBraceDepthInRange counts the net brace depth from startLine to endLine
// (0-indexed) in the given lines slice.
func countBraceDepthInRange(lines []string, startLine, endLine int) int {
	depth := 0
	maxDepth := 0
	for i := startLine; i <= endLine && i < len(lines); i++ {
		for _, ch := range lines[i] {
			if ch == '{' {
				depth++
				if depth > maxDepth {
					maxDepth = depth
				}
			} else if ch == '}' {
				depth--
			}
		}
	}
	return maxDepth
}

// countSignatureParams counts parameters from a function signature string.
func countSignatureParams(sig string) int {
	// Find the parameter list between parentheses.
	start := strings.Index(sig, "(")
	if start < 0 {
		return 0
	}
	end := strings.Index(sig[start:], ")")
	if end < 0 {
		return 0
	}
	paramStr := sig[start+1 : start+end]
	paramStr = strings.TrimSpace(paramStr)
	if paramStr == "" || paramStr == "self" || paramStr == "cls" {
		return 0
	}
	// Split by comma and count.
	parts := strings.Split(paramStr, ",")
	count := 0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "self" && p != "cls" {
			count++
		}
	}
	return count
}

// ComplexityFile analyzes a single file without requiring a loaded index.
func ComplexityFile(file string, maxResults int, minComplexity int) (*ComplexityResult, error) {
	idx := &Index{}
	return idx.Complexity(file, maxResults, minComplexity)
}

// FormatComplexity renders the complexity result as human-readable text.
func FormatComplexity(result *ComplexityResult) string {
	var b strings.Builder

	if len(result.Functions) == 0 {
		b.WriteString("No functions found.\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Complexity Report (top %d most complex functions):\n\n", len(result.Functions)))

	for _, f := range result.Functions {
		b.WriteString(fmt.Sprintf("  %-40s %-30s complexity=%-4d lines=%-4d depth=%d\n",
			fmt.Sprintf("%s:%d", f.Path, f.Line),
			f.Name+"()",
			f.Complexity,
			f.Lines,
			f.MaxDepth,
		))
	}

	b.WriteString(fmt.Sprintf("\n%d functions analyzed, avg complexity=%.1f, max=%d, high complexity (>=10): %d\n",
		result.TotalFunctions, result.AvgComplexity, result.MaxComplexity, result.HighComplexityCount))

	return b.String()
}
