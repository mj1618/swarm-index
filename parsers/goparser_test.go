package parsers

import (
	"testing"
)

const sampleGoSource = `package sample

import "fmt"

const MaxRetries = 3

var (
	Version   = "1.0"
	debugMode = false
)

type Config struct {
	Host string
	Port int
}

type Handler interface {
	Handle(req Request) error
}

type StringAlias = string

func Init() error {
	return nil
}

func helperFunc(x, y int) (int, bool) {
	return x + y, true
}

func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host required")
	}
	return nil
}

func (c Config) String() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
`

func TestGoParserBasic(t *testing.T) {
	p := &GoParser{}
	symbols, err := p.Parse("sample.go", []byte(sampleGoSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	// Check constants
	assertSymbol(t, byName, "MaxRetries", "const", true, "")
	assertSymbol(t, byName, "Version", "var", true, "")
	assertSymbol(t, byName, "debugMode", "var", false, "")

	// Check types
	assertSymbol(t, byName, "Config", "struct", true, "")
	assertSymbol(t, byName, "Handler", "interface", true, "")
	assertSymbol(t, byName, "StringAlias", "type", true, "")

	// Check functions
	assertSymbol(t, byName, "Init", "func", true, "")
	assertSymbol(t, byName, "helperFunc", "func", false, "")

	// Check methods
	assertSymbol(t, byName, "Validate", "method", true, "Config")
	assertSymbol(t, byName, "String", "method", true, "Config")
}

func TestGoParserSignatures(t *testing.T) {
	p := &GoParser{}
	symbols, err := p.Parse("sample.go", []byte(sampleGoSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	tests := []struct {
		name string
		want string
	}{
		{"Init", "func Init() error"},
		{"helperFunc", "func helperFunc(x, y int) (int, bool)"},
		{"Validate", "func (*Config) Validate() error"},
		{"String", "func (Config) String() string"},
	}

	for _, tt := range tests {
		sym, ok := byName[tt.name]
		if !ok {
			t.Errorf("symbol %q not found", tt.name)
			continue
		}
		if sym.Signature != tt.want {
			t.Errorf("symbol %q signature = %q, want %q", tt.name, sym.Signature, tt.want)
		}
	}
}

func TestGoParserLineNumbers(t *testing.T) {
	p := &GoParser{}
	symbols, err := p.Parse("sample.go", []byte(sampleGoSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// All symbols should have positive line numbers.
	for _, s := range symbols {
		if s.Line <= 0 {
			t.Errorf("symbol %q has non-positive Line: %d", s.Name, s.Line)
		}
		if s.EndLine < s.Line {
			t.Errorf("symbol %q EndLine (%d) < Line (%d)", s.Name, s.EndLine, s.Line)
		}
	}
}

func TestGoParserEmptyFile(t *testing.T) {
	p := &GoParser{}
	symbols, err := p.Parse("empty.go", []byte("package empty\n"))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols for empty file, got %d", len(symbols))
	}
}

func TestGoParserSyntaxError(t *testing.T) {
	p := &GoParser{}
	_, err := p.Parse("bad.go", []byte("this is not valid go"))
	if err == nil {
		t.Error("expected error for invalid Go source")
	}
}

func TestGoParserExtensions(t *testing.T) {
	p := &GoParser{}
	exts := p.Extensions()
	if len(exts) != 1 || exts[0] != ".go" {
		t.Errorf("Extensions() = %v, want [\".go\"]", exts)
	}
}

func TestGoParserVariadic(t *testing.T) {
	src := `package x

func Printf(format string, args ...interface{}) {
}
`
	p := &GoParser{}
	symbols, err := p.Parse("x.go", []byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	want := "func Printf(format string, args ...interface{})"
	if symbols[0].Signature != want {
		t.Errorf("signature = %q, want %q", symbols[0].Signature, want)
	}
}

func TestForExtension(t *testing.T) {
	p := ForExtension(".go")
	if p == nil {
		t.Fatal("ForExtension(\".go\") returned nil")
	}
	p = ForExtension(".xyz")
	if p != nil {
		t.Error("ForExtension(\".xyz\") should return nil")
	}
}

func symbolsByName(symbols []Symbol) map[string]Symbol {
	m := map[string]Symbol{}
	for _, s := range symbols {
		m[s.Name] = s
	}
	return m
}

func assertSymbol(t *testing.T, byName map[string]Symbol, name, kind string, exported bool, parent string) {
	t.Helper()
	sym, ok := byName[name]
	if !ok {
		t.Errorf("symbol %q not found", name)
		return
	}
	if sym.Kind != kind {
		t.Errorf("symbol %q kind = %q, want %q", name, sym.Kind, kind)
	}
	if sym.Exported != exported {
		t.Errorf("symbol %q exported = %v, want %v", name, sym.Exported, exported)
	}
	if sym.Parent != parent {
		t.Errorf("symbol %q parent = %q, want %q", name, sym.Parent, parent)
	}
}
