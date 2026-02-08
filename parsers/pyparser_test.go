package parsers

import (
	"testing"
)

const samplePythonSource = `#!/usr/bin/env python3
"""Module docstring."""

import os
from pathlib import Path

MAX_RETRIES = 3
DEFAULT_TIMEOUT: int = 30
_INTERNAL_FLAG = True

def hello():
    """Say hello."""
    print("hello")

def _helper(x, y):
    return x + y

async def fetch(url: str) -> dict:
    pass

class MyClass:
    """A sample class."""

    def __init__(self, name):
        self.name = name

    def method(self):
        return self.name

    async def async_method(self):
        pass

    def _private_method(self):
        pass

class _PrivateClass:
    pass

@app.route("/")
def index():
    return "ok"

@login_required
@cache(timeout=300)
def dashboard(request):
    return render(request, "dashboard.html")
`

func TestPythonParserBasic(t *testing.T) {
	p := &PythonParser{}
	symbols, err := p.Parse("sample.py", []byte(samplePythonSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	// Top-level functions
	assertSymbol(t, byName, "hello", "func", true, "")
	assertSymbol(t, byName, "_helper", "func", false, "")
	assertSymbol(t, byName, "fetch", "func", true, "")

	// Classes
	assertSymbol(t, byName, "MyClass", "class", true, "")
	assertSymbol(t, byName, "_PrivateClass", "class", false, "")

	// Methods
	assertSymbol(t, byName, "__init__", "method", false, "MyClass")
	assertSymbol(t, byName, "method", "method", true, "MyClass")
	assertSymbol(t, byName, "async_method", "method", true, "MyClass")
	assertSymbol(t, byName, "_private_method", "method", false, "MyClass")

	// Constants
	assertSymbol(t, byName, "MAX_RETRIES", "const", true, "")
	assertSymbol(t, byName, "DEFAULT_TIMEOUT", "const", true, "")

	// Decorated functions
	assertSymbol(t, byName, "index", "func", true, "")
	assertSymbol(t, byName, "dashboard", "func", true, "")
}

func TestPythonParserSignatures(t *testing.T) {
	p := &PythonParser{}
	symbols, err := p.Parse("sample.py", []byte(samplePythonSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	tests := []struct {
		name string
		want string
	}{
		{"hello", "def hello():"},
		{"fetch", "async def fetch(url: str) -> dict:"},
		{"MyClass", "class MyClass:"},
		{"index", "@app.route(\"/\")\ndef index():"},
		{"dashboard", "@login_required\n@cache(timeout=300)\ndef dashboard(request):"},
		{"MAX_RETRIES", "MAX_RETRIES = 3"},
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

func TestPythonParserLineNumbers(t *testing.T) {
	p := &PythonParser{}
	symbols, err := p.Parse("sample.py", []byte(samplePythonSource))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	for _, s := range symbols {
		if s.Line <= 0 {
			t.Errorf("symbol %q has non-positive Line: %d", s.Name, s.Line)
		}
		if s.EndLine < s.Line {
			t.Errorf("symbol %q EndLine (%d) < Line (%d)", s.Name, s.EndLine, s.Line)
		}
	}
}

func TestPythonParserEmptyFile(t *testing.T) {
	p := &PythonParser{}
	symbols, err := p.Parse("empty.py", []byte(""))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols for empty file, got %d", len(symbols))
	}
}

func TestPythonParserCommentsOnly(t *testing.T) {
	src := `# This is a comment
# Another comment
"""
A module docstring.
"""
`
	p := &PythonParser{}
	symbols, err := p.Parse("comments.py", []byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(symbols))
	}
}

func TestPythonParserNestedClass(t *testing.T) {
	src := `class Outer:
    def outer_method(self):
        pass

    class Inner:
        def inner_method(self):
            pass

def standalone():
    pass
`
	p := &PythonParser{}
	symbols, err := p.Parse("nested.py", []byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	// Top-level class and its direct methods should be found.
	assertSymbol(t, byName, "Outer", "class", true, "")
	assertSymbol(t, byName, "outer_method", "method", true, "Outer")

	// standalone function after class should be found.
	assertSymbol(t, byName, "standalone", "func", true, "")
}

func TestPythonParserExtensions(t *testing.T) {
	p := &PythonParser{}
	exts := p.Extensions()
	if len(exts) != 1 || exts[0] != ".py" {
		t.Errorf("Extensions() = %v, want [\".py\"]", exts)
	}
}

func TestPythonParserForExtension(t *testing.T) {
	p := ForExtension(".py")
	if p == nil {
		t.Fatal("ForExtension(\".py\") returned nil")
	}
}

func TestPythonParserMixedFile(t *testing.T) {
	src := `"""A real-world-like module."""

import logging

LOG_LEVEL = "INFO"

logger = logging.getLogger(__name__)

class Config:
    def __init__(self, path: str):
        self.path = path

    def load(self) -> dict:
        pass

    def _parse(self):
        pass

def create_config(path: str) -> Config:
    return Config(path)

async def async_create(path: str) -> Config:
    return Config(path)

class _InternalProcessor:
    def process(self):
        pass

API_VERSION = "v2"
`
	p := &PythonParser{}
	symbols, err := p.Parse("mixed.py", []byte(src))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	byName := symbolsByName(symbols)

	// Constants
	assertSymbol(t, byName, "LOG_LEVEL", "const", true, "")
	assertSymbol(t, byName, "API_VERSION", "const", true, "")

	// Classes
	assertSymbol(t, byName, "Config", "class", true, "")
	assertSymbol(t, byName, "_InternalProcessor", "class", false, "")

	// Methods
	assertSymbol(t, byName, "__init__", "method", false, "Config")
	assertSymbol(t, byName, "load", "method", true, "Config")
	assertSymbol(t, byName, "_parse", "method", false, "Config")
	assertSymbol(t, byName, "process", "method", true, "_InternalProcessor")

	// Functions
	assertSymbol(t, byName, "create_config", "func", true, "")
	assertSymbol(t, byName, "async_create", "func", true, "")

	// Ensure total count is correct (no duplicates or extras).
	expectedCount := 10
	if len(symbols) != expectedCount {
		t.Errorf("expected %d symbols, got %d", expectedCount, len(symbols))
		for _, s := range symbols {
			t.Logf("  %s (kind=%s, parent=%s)", s.Name, s.Kind, s.Parent)
		}
	}
}
