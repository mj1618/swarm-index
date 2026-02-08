package index

import (
	"testing"
)

func TestRefsFindsDefinitionAndReferences(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "lib.go", `package main

func Helper() string {
	return "ok"
}
`)
	mkFile(t, tmp, "main.go", `package main

func main() {
	x := Helper()
	_ = x
	y := Helper()
	_ = y
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Refs("Helper", 50)
	if err != nil {
		t.Fatalf("Refs() error: %v", err)
	}

	if result.Symbol != "Helper" {
		t.Errorf("Symbol = %q, want %q", result.Symbol, "Helper")
	}
	if result.Definition == nil {
		t.Fatal("Definition is nil, want a definition match")
	}
	if result.Definition.Path != "lib.go" {
		t.Errorf("Definition.Path = %q, want %q", result.Definition.Path, "lib.go")
	}
	if result.Definition.Line != 3 {
		t.Errorf("Definition.Line = %d, want 3", result.Definition.Line)
	}
	if result.TotalRefs < 2 {
		t.Errorf("TotalRefs = %d, want >= 2", result.TotalRefs)
	}

	// Definition should not appear in references.
	for _, r := range result.References {
		if r.IsDefinition {
			t.Errorf("found IsDefinition=true in References: %s:%d", r.Path, r.Line)
		}
	}
}

func TestRefsMaxResults(t *testing.T) {
	tmp := t.TempDir()
	content := "package main\n\nfunc Foo() {}\n"
	for i := 0; i < 20; i++ {
		content += "var _ = Foo()\n"
	}
	mkFile(t, tmp, "main.go", content)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Refs("Foo", 5)
	if err != nil {
		t.Fatalf("Refs() error: %v", err)
	}

	if len(result.References) > 5 {
		t.Errorf("References count = %d, want <= 5", len(result.References))
	}
}

func TestRefsSymbolNotFound(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n\nfunc main() {}\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Refs("NonexistentSymbol", 50)
	if err != nil {
		t.Fatalf("Refs() should not error for missing symbol, got: %v", err)
	}

	if result.Definition != nil {
		t.Errorf("Definition should be nil for missing symbol")
	}
	if len(result.References) != 0 {
		t.Errorf("References = %d, want 0", len(result.References))
	}
}

func TestRefsAcrossMultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "types.go", `package main

type Config struct {
	Name string
}
`)
	mkFile(t, tmp, "main.go", `package main

func main() {
	c := Config{Name: "test"}
	_ = c
}
`)
	mkFile(t, tmp, "util.go", `package main

func newConfig() Config {
	return Config{}
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Refs("Config", 50)
	if err != nil {
		t.Fatalf("Refs() error: %v", err)
	}

	if result.Definition == nil {
		t.Fatal("Definition is nil, want a definition match for 'Config'")
	}
	if result.Definition.Path != "types.go" {
		t.Errorf("Definition.Path = %q, want %q", result.Definition.Path, "types.go")
	}

	// Should find references in main.go and util.go.
	if result.TotalRefs < 2 {
		t.Errorf("TotalRefs = %d, want >= 2", result.TotalRefs)
	}

	// Verify we see references from multiple files.
	fileSeen := make(map[string]bool)
	for _, r := range result.References {
		fileSeen[r.Path] = true
	}
	if !fileSeen["main.go"] {
		t.Error("expected reference in main.go")
	}
	if !fileSeen["util.go"] {
		t.Error("expected reference in util.go")
	}
}

func TestRefsDefinitionExcludedFromRefs(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func Greet() string {
	return "hello"
}

func main() {
	fmt.Println(Greet())
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Refs("Greet", 50)
	if err != nil {
		t.Fatalf("Refs() error: %v", err)
	}

	if result.Definition == nil {
		t.Fatal("Definition is nil")
	}

	// The definition line should not be in References.
	for _, r := range result.References {
		if r.Path == result.Definition.Path && r.Line == result.Definition.Line {
			t.Errorf("definition line appears in References: %s:%d", r.Path, r.Line)
		}
	}
}

func TestRefsPythonDef(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.py", `def handle_request(req):
    return process(req)
`)
	mkFile(t, tmp, "main.py", `from app import handle_request

handle_request(my_req)
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Refs("handle_request", 50)
	if err != nil {
		t.Fatalf("Refs() error: %v", err)
	}

	if result.Definition == nil {
		t.Fatal("Definition is nil for Python def")
	}
	if result.Definition.Path != "app.py" {
		t.Errorf("Definition.Path = %q, want %q", result.Definition.Path, "app.py")
	}
}
