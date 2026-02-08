package parsers

import (
	"regexp"
	"strings"
	"unicode"
)

func init() {
	Register(&PythonParser{})
}

// PythonParser extracts symbols from Python source files using regex heuristics.
type PythonParser struct{}

func (p *PythonParser) Extensions() []string {
	return []string{".py"}
}

var (
	pyFuncRe     = regexp.MustCompile(`^(async\s+)?def\s+(\w+)\s*\((.*)`)
	pyClassRe    = regexp.MustCompile(`^class\s+(\w+)(.*)`)
	pyConstRe    = regexp.MustCompile(`^([A-Z][A-Z0-9_]*)\s*[=:]`)
	pyDecoratorRe = regexp.MustCompile(`^@\S+`)
)

func (p *PythonParser) Parse(filePath string, content []byte) ([]Symbol, error) {
	lines := strings.Split(string(content), "\n")
	var symbols []Symbol

	var currentClass string
	var classIndent int
	var decorators []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines and comments.
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			// Reset decorators on blank line or comment between decorator and def/class.
			if trimmed == "" {
				decorators = nil
			}
			continue
		}

		indent := lineIndent(line)

		// If we're inside a class and hit a line at indent level 0 that isn't
		// a blank/comment (already filtered above), leave the class context.
		if currentClass != "" && indent == 0 {
			// Check if this is a decorator — decorators at level 0 might precede
			// a new class or a standalone function, so leave class context.
			currentClass = ""
		}

		// Collect decorators.
		if pyDecoratorRe.MatchString(trimmed) {
			decorators = append(decorators, trimmed)
			continue
		}

		// Class definition at indent level 0.
		if indent == 0 {
			if m := pyClassRe.FindStringSubmatch(trimmed); m != nil {
				name := m[1]
				sig := buildSignature(decorators, trimmed)
				endLine := findPythonBlockEnd(lines, i)
				symbols = append(symbols, Symbol{
					Name:      name,
					Kind:      "class",
					Line:      i + 1,
					EndLine:   endLine,
					Exported:  !strings.HasPrefix(name, "_"),
					Signature: sig,
				})
				currentClass = name
				classIndent = 0
				decorators = nil
				continue
			}
		}

		// Function/method definition.
		if m := pyFuncRe.FindStringSubmatch(trimmed); m != nil {
			name := m[2]
			sig := buildSignature(decorators, trimmed)
			endLine := findPythonBlockEnd(lines, i)

			sym := Symbol{
				Name:      name,
				Line:      i + 1,
				EndLine:   endLine,
				Exported:  !strings.HasPrefix(name, "_"),
				Signature: sig,
			}

			if indent == 0 {
				sym.Kind = "func"
			} else if currentClass != "" && indent > classIndent {
				sym.Kind = "method"
				sym.Parent = currentClass
			} else {
				// Indented function outside a class context — skip (nested function).
				decorators = nil
				continue
			}

			symbols = append(symbols, sym)
			decorators = nil
			continue
		}

		// Module-level constants (UPPER_SNAKE_CASE at indent 0).
		if indent == 0 {
			if m := pyConstRe.FindStringSubmatch(trimmed); m != nil {
				name := m[1]
				// Make sure it's truly UPPER_SNAKE_CASE (at least 2 chars to avoid
				// matching single-letter variable like I or X used conventionally).
				if isUpperSnakeCase(name) {
					symbols = append(symbols, Symbol{
						Name:      name,
						Kind:      "const",
						Line:      i + 1,
						EndLine:   i + 1,
						Exported:  true,
						Signature: trimmed,
					})
				}
			}
		}

		// Any non-decorator, non-def, non-class line resets decorators.
		decorators = nil
	}

	return symbols, nil
}

// lineIndent returns the number of leading whitespace characters.
func lineIndent(line string) int {
	for i, ch := range line {
		if !unicode.IsSpace(ch) {
			return i
		}
	}
	return len(line)
}

// isUpperSnakeCase returns true if name matches [A-Z][A-Z0-9_]* pattern.
func isUpperSnakeCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i, ch := range name {
		if i == 0 {
			if ch < 'A' || ch > 'Z' {
				return false
			}
		} else {
			if !((ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
				return false
			}
		}
	}
	return true
}

// findPythonBlockEnd finds the last line of a Python block starting at startLine.
// It looks for the next line at the same or lesser indentation level (that isn't
// blank or a comment), or the end of the file.
func findPythonBlockEnd(lines []string, startLine int) int {
	baseIndent := lineIndent(lines[startLine])
	lastNonEmpty := startLine

	for i := startLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := lineIndent(lines[i])
		if indent <= baseIndent {
			return lastNonEmpty + 1 // 1-indexed
		}
		lastNonEmpty = i
	}

	return lastNonEmpty + 1 // 1-indexed
}

// buildSignature creates the signature string, optionally prepending decorators.
func buildSignature(decorators []string, defLine string) string {
	if len(decorators) == 0 {
		return defLine
	}
	return strings.Join(decorators, "\n") + "\n" + defLine
}
