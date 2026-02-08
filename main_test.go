package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
