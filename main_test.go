package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMaxDefault(t *testing.T) {
	got := parseMax([]string{})
	if got != 20 {
		t.Errorf("parseMax([]) = %d, want 20", got)
	}
}

func TestParseMaxWithFlag(t *testing.T) {
	got := parseMax([]string{"--max", "5"})
	if got != 5 {
		t.Errorf("parseMax([--max 5]) = %d, want 5", got)
	}
}

func TestParseMaxWithOtherFlags(t *testing.T) {
	got := parseMax([]string{"--root", "/tmp", "--max", "10"})
	if got != 10 {
		t.Errorf("parseMax([--root /tmp --max 10]) = %d, want 10", got)
	}
}

func TestParseMaxInvalidValue(t *testing.T) {
	got := parseMax([]string{"--max", "abc"})
	if got != 20 {
		t.Errorf("parseMax([--max abc]) = %d, want 20 (default)", got)
	}
}

func TestParseMaxZero(t *testing.T) {
	got := parseMax([]string{"--max", "0"})
	if got != 20 {
		t.Errorf("parseMax([--max 0]) = %d, want 20 (default, 0 is not positive)", got)
	}
}

func TestParseMaxNoValue(t *testing.T) {
	got := parseMax([]string{"--max"})
	if got != 20 {
		t.Errorf("parseMax([--max]) = %d, want 20 (default)", got)
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
