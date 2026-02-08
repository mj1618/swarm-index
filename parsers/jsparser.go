package parsers

import (
	"regexp"
	"strings"
)

func init() {
	Register(&JSParser{})
}

// JSParser extracts symbols from JavaScript and TypeScript source files
// using regex heuristics and brace-depth tracking.
type JSParser struct{}

func (p *JSParser) Extensions() []string {
	return []string{".js", ".jsx", ".ts", ".tsx"}
}

var (
	// Function declarations: [export] [default] [async] function name(
	jsFuncRe = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s+(\w+)`)

	// Class declarations: [export] [default] [abstract] class Name
	jsClassRe = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+(\w+)`)

	// Interface declarations: [export] interface Name
	jsInterfaceRe = regexp.MustCompile(`^(?:export\s+)?interface\s+(\w+)`)

	// Type alias declarations: [export] type Name =
	jsTypeRe = regexp.MustCompile(`^(?:export\s+)?type\s+(\w+)\b`)

	// Enum declarations: [export] [const] enum Name
	jsEnumRe = regexp.MustCompile(`^(?:export\s+)?(?:const\s+)?enum\s+(\w+)`)

	// Variable declarations: [export] const/let/var name
	jsVarRe = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(\w+)`)

	// Class method/property: name(...) { or async name(...) { or get/set name(
	jsMethodRe = regexp.MustCompile(`^(?:(?:public|private|protected|static|readonly|abstract|override|async|get|set)\s+)*(\w+)\s*[<(]`)
)

func (p *JSParser) Parse(filePath string, content []byte) ([]Symbol, error) {
	lines := strings.Split(string(content), "\n")
	var symbols []Symbol

	braceDepth := 0
	inBlockComment := false
	var currentClass string
	var classStartDepth int

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Handle block comments.
		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
				// Process the rest of the line after the comment end.
				idx := strings.Index(trimmed, "*/")
				trimmed = strings.TrimSpace(trimmed[idx+2:])
				if trimmed == "" {
					continue
				}
			} else {
				continue
			}
		}

		// Skip empty lines and single-line comments.
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check for block comment start on this line.
		if strings.Contains(trimmed, "/*") {
			if strings.Contains(trimmed, "*/") {
				// Single-line block comment — remove it and continue processing.
				trimmed = removeInlineBlockComments(trimmed)
				if strings.TrimSpace(trimmed) == "" {
					continue
				}
			} else {
				// Multi-line block comment starts here.
				// Process the part before the comment.
				idx := strings.Index(trimmed, "/*")
				before := strings.TrimSpace(trimmed[:idx])
				inBlockComment = true
				if before == "" {
					continue
				}
				trimmed = before
			}
		}

		// Skip decorators / annotations (lines starting with @).
		if strings.HasPrefix(trimmed, "@") {
			continue
		}

		// Skip import/require lines.
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import{") ||
			strings.HasPrefix(trimmed, "require(") {
			braceDepth += countBraces(trimmed)
			continue
		}

		exported := strings.HasPrefix(trimmed, "export ")

		if braceDepth == 0 {
			// Top-level: look for declarations.
			if sym, ok := p.matchTopLevel(trimmed, i+1, exported); ok {
				if sym.Kind == "class" {
					currentClass = sym.Name
					classStartDepth = braceDepth
				}
				// Find end line by scanning forward.
				sym.EndLine = findJSBlockEnd(lines, i, braceDepth)
				symbols = append(symbols, sym)
				braceDepth += countBraces(trimmed)
				continue
			}
		} else if braceDepth == classStartDepth+1 && currentClass != "" {
			// Inside a class body: look for methods.
			if sym, ok := p.matchMethod(trimmed, i+1, currentClass); ok {
				sym.EndLine = findJSBlockEnd(lines, i, braceDepth)
				symbols = append(symbols, sym)
				braceDepth += countBraces(trimmed)
				continue
			}
		}

		braceDepth += countBraces(trimmed)

		// If brace depth returns to class start depth, we've left the class.
		if currentClass != "" && braceDepth <= classStartDepth {
			currentClass = ""
		}
	}

	return symbols, nil
}

// matchTopLevel tries to match a top-level declaration on the given line.
func (p *JSParser) matchTopLevel(trimmed string, lineNum int, exported bool) (Symbol, bool) {
	// Order matters: check more specific patterns first.

	// Function declarations.
	if m := jsFuncRe.FindStringSubmatch(trimmed); m != nil {
		return Symbol{
			Name:      m[1],
			Kind:      "func",
			Line:      lineNum,
			Exported:  exported,
			Signature: trimmed,
		}, true
	}

	// Class declarations.
	if m := jsClassRe.FindStringSubmatch(trimmed); m != nil {
		return Symbol{
			Name:      m[1],
			Kind:      "class",
			Line:      lineNum,
			Exported:  exported,
			Signature: trimFirstBrace(trimmed),
		}, true
	}

	// Interface declarations.
	if m := jsInterfaceRe.FindStringSubmatch(trimmed); m != nil {
		return Symbol{
			Name:      m[1],
			Kind:      "interface",
			Line:      lineNum,
			Exported:  exported,
			Signature: trimFirstBrace(trimmed),
		}, true
	}

	// Enum declarations (check before type to avoid conflict with "const enum").
	if m := jsEnumRe.FindStringSubmatch(trimmed); m != nil {
		return Symbol{
			Name:      m[1],
			Kind:      "enum",
			Line:      lineNum,
			Exported:  exported,
			Signature: trimFirstBrace(trimmed),
		}, true
	}

	// Type alias declarations.
	if m := jsTypeRe.FindStringSubmatch(trimmed); m != nil {
		return Symbol{
			Name:      m[1],
			Kind:      "type",
			Line:      lineNum,
			Exported:  exported,
			Signature: trimmed,
		}, true
	}

	// Variable declarations (const/let/var).
	if m := jsVarRe.FindStringSubmatch(trimmed); m != nil {
		return Symbol{
			Name:      m[1],
			Kind:      "const",
			Line:      lineNum,
			Exported:  exported,
			Signature: trimmed,
		}, true
	}

	return Symbol{}, false
}

// matchMethod tries to match a class method or property declaration.
func (p *JSParser) matchMethod(trimmed string, lineNum int, className string) (Symbol, bool) {
	// Skip lines that are just a closing brace or similar.
	if trimmed == "}" || trimmed == "};" {
		return Symbol{}, false
	}

	// Skip lines that are property declarations without parens (e.g., "name: string;").
	// We only want methods (things with parentheses).
	if m := jsMethodRe.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		// Skip keywords that are not method names.
		if isJSKeyword(name) {
			return Symbol{}, false
		}
		exported := !strings.HasPrefix(name, "_") && !strings.HasPrefix(name, "#")
		// Private methods are not exported.
		if strings.HasPrefix(trimmed, "private ") {
			exported = false
		}
		return Symbol{
			Name:      name,
			Kind:      "method",
			Line:      lineNum,
			Exported:  exported,
			Parent:    className,
			Signature: trimmed,
		}, true
	}

	// Constructor.
	if strings.HasPrefix(trimmed, "constructor(") || strings.HasPrefix(trimmed, "constructor (") {
		return Symbol{
			Name:      "constructor",
			Kind:      "method",
			Line:      lineNum,
			Exported:  true,
			Parent:    className,
			Signature: trimmed,
		}, true
	}

	return Symbol{}, false
}

// countBraces returns the net brace count ({  minus }) in a line,
// ignoring braces in string literals and comments.
func countBraces(line string) int {
	depth := 0
	inSingle := false
	inDouble := false
	inTemplate := false
	escaped := false

	for _, ch := range line {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}

		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			}
		case inTemplate:
			if ch == '`' {
				inTemplate = false
			}
		default:
			switch ch {
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '`':
				inTemplate = true
			case '{':
				depth++
			case '}':
				depth--
			}
		}
	}

	return depth
}

// findJSBlockEnd finds the last line of a block (delimited by braces) starting at startLine.
func findJSBlockEnd(lines []string, startLine int, currentDepth int) int {
	depth := currentDepth
	for i := startLine; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		// Skip comment-only lines.
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		depth += countBraces(trimmed)
		if depth <= currentDepth && i > startLine {
			return i + 1 // 1-indexed
		}
	}
	return len(lines) // end of file
}

// trimFirstBrace trims everything from the first '{' onward for a cleaner signature.
func trimFirstBrace(s string) string {
	if idx := strings.Index(s, "{"); idx > 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// removeInlineBlockComments removes /* ... */ comments from a single line.
func removeInlineBlockComments(s string) string {
	for {
		start := strings.Index(s, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(s[start+2:], "*/")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+2+end+2:]
	}
	return s
}

// isJSKeyword returns true if the name is a JavaScript/TypeScript keyword
// that shouldn't be treated as a method name.
func isJSKeyword(name string) bool {
	switch name {
	case "if", "else", "for", "while", "do", "switch", "case", "break",
		"continue", "return", "throw", "try", "catch", "finally",
		"new", "delete", "typeof", "instanceof", "void", "in", "of",
		"class", "extends", "super", "import", "export", "default",
		"function", "const", "let", "var", "this", "true", "false", "null":
		return true
	}
	return false
}
