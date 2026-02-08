package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseIntFlagDefault(t *testing.T) {
	got := parseIntFlag([]string{}, "--max", 20)
	if got != 20 {
		t.Errorf("parseIntFlag([], --max, 20) = %d, want 20", got)
	}
}

func TestParseIntFlagWithFlag(t *testing.T) {
	got := parseIntFlag([]string{"--max", "5"}, "--max", 20)
	if got != 5 {
		t.Errorf("parseIntFlag([--max 5]) = %d, want 5", got)
	}
}

func TestParseIntFlagWithOtherFlags(t *testing.T) {
	got := parseIntFlag([]string{"--root", "/tmp", "--max", "10"}, "--max", 20)
	if got != 10 {
		t.Errorf("parseIntFlag([--root /tmp --max 10]) = %d, want 10", got)
	}
}

func TestParseIntFlagInvalidValue(t *testing.T) {
	got := parseIntFlag([]string{"--max", "abc"}, "--max", 20)
	if got != 20 {
		t.Errorf("parseIntFlag([--max abc]) = %d, want 20 (default)", got)
	}
}

func TestParseIntFlagZero(t *testing.T) {
	got := parseIntFlag([]string{"--max", "0"}, "--max", 20)
	if got != 20 {
		t.Errorf("parseIntFlag([--max 0]) = %d, want 20 (default, 0 is not positive)", got)
	}
}

func TestParseIntFlagNoValue(t *testing.T) {
	got := parseIntFlag([]string{"--max"}, "--max", 20)
	if got != 20 {
		t.Errorf("parseIntFlag([--max]) = %d, want 20 (default)", got)
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		query   string
		wantErr bool
	}{
		{"hello", false},
		{"", true},
		{"   ", true},
		{"\t", true},
		{"a", false},
	}
	for _, tt := range tests {
		err := validateQuery(tt.query)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateQuery(%q) error = %v, wantErr %v", tt.query, err, tt.wantErr)
		}
	}
}

func TestResolveRootWithFlag(t *testing.T) {
	tmp := t.TempDir()
	args := []string{"--root", tmp}

	got, err := resolveRoot(args)
	if err != nil {
		t.Fatalf("resolveRoot() error: %v", err)
	}

	want, _ := filepath.Abs(tmp)
	if got != want {
		t.Errorf("resolveRoot() = %q, want %q", got, want)
	}
}

func TestResolveRootWithoutFlag(t *testing.T) {
	// Create a temp dir with a swarm/index/meta.json so findIndexRoot succeeds
	tmp := t.TempDir()
	metaDir := filepath.Join(tmp, "swarm", "index")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Resolve symlinks (macOS /var -> /private/var)
	tmp, _ = filepath.EvalSymlinks(tmp)

	// Change to the temp dir so findIndexRoot(".") finds the index
	orig, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(orig)

	got, err := resolveRoot([]string{})
	if err != nil {
		t.Fatalf("resolveRoot() error: %v", err)
	}

	want, _ := filepath.Abs(tmp)
	if got != want {
		t.Errorf("resolveRoot() = %q, want %q", got, want)
	}
}

func TestResolveRootFlagWithoutValue(t *testing.T) {
	// --root at end with no value should fall through to findIndexRoot
	tmp := t.TempDir()
	metaDir := filepath.Join(tmp, "swarm", "index")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Resolve symlinks (macOS /var -> /private/var)
	tmp, _ = filepath.EvalSymlinks(tmp)

	orig, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(orig)

	got, err := resolveRoot([]string{"--root"})
	if err != nil {
		t.Fatalf("resolveRoot() error: %v", err)
	}

	want, _ := filepath.Abs(tmp)
	if got != want {
		t.Errorf("resolveRoot() = %q, want %q", got, want)
	}
}

func TestExtractJSONFlagNotPresent(t *testing.T) {
	args, found := extractJSONFlag([]string{"swarm-index", "scan", "."})
	if found {
		t.Error("expected found=false when --json is absent")
	}
	want := []string{"swarm-index", "scan", "."}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestExtractJSONFlagAtEnd(t *testing.T) {
	args, found := extractJSONFlag([]string{"swarm-index", "scan", ".", "--json"})
	if !found {
		t.Error("expected found=true when --json is present")
	}
	want := []string{"swarm-index", "scan", "."}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestExtractJSONFlagBeforeCommand(t *testing.T) {
	args, found := extractJSONFlag([]string{"swarm-index", "--json", "version"})
	if !found {
		t.Error("expected found=true when --json is before command")
	}
	want := []string{"swarm-index", "version"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestExtractJSONFlagBetweenArgs(t *testing.T) {
	args, found := extractJSONFlag([]string{"swarm-index", "lookup", "--json", "query", "--root", "/tmp"})
	if !found {
		t.Error("expected found=true when --json is between args")
	}
	want := []string{"swarm-index", "lookup", "query", "--root", "/tmp"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestExtractJSONFlagEmpty(t *testing.T) {
	args, found := extractJSONFlag([]string{})
	if found {
		t.Error("expected found=false for empty args")
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}
