package parsers

// Symbol represents a top-level symbol extracted from a source file.
type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`      // "func", "method", "type", "interface", "struct", "const", "var"
	Line      int    `json:"line"`
	EndLine   int    `json:"endLine"`
	Exported  bool   `json:"exported"`
	Signature string `json:"signature"` // e.g. "func HandleAuth(w http.ResponseWriter, r *http.Request) error"
	Parent    string `json:"parent"`    // enclosing type for methods, empty otherwise
}

// Parser extracts symbols from a source file.
type Parser interface {
	Parse(filePath string, content []byte) ([]Symbol, error)
	Extensions() []string
}

// registry maps file extensions to parsers.
var registry = map[string]Parser{}

// Register adds a parser for the given extensions.
func Register(p Parser) {
	for _, ext := range p.Extensions() {
		registry[ext] = p
	}
}

// ForExtension returns the parser registered for the given file extension,
// or nil if none is available.
func ForExtension(ext string) Parser {
	return registry[ext]
}
