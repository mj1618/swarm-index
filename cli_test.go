package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "swarm-index-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "swarm-index")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// runBinary executes the test binary with the given args and returns stdout, stderr, and any error.
func runBinary(args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// makeTestDir creates a temp directory with some sample files for scanning.
func makeTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Create a Go file.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a subdirectory with a file.
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "helper.go"), []byte("package pkg\n\nfunc Helper() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a markdown file.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// --- scan command ---

func TestCLIScanText(t *testing.T) {
	dir := makeTestDir(t)
	stdout, _, err := runBinary("scan", dir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !strings.Contains(stdout, "Index saved to") {
		t.Errorf("expected 'Index saved to' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "3 files") {
		t.Errorf("expected '3 files' in output, got: %s", stdout)
	}
}

func TestCLIScanJSON(t *testing.T) {
	dir := makeTestDir(t)
	stdout, _, err := runBinary("scan", dir, "--json")
	if err != nil {
		t.Fatalf("scan --json failed: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, stdout)
	}
	for _, key := range []string{"filesIndexed", "packages", "indexPath", "extensions"} {
		if _, ok := result[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}
}

func TestCLIScanMissingDir(t *testing.T) {
	_, stderr, err := runBinary("scan")
	if err == nil {
		t.Fatal("expected non-zero exit for scan with no directory")
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage error on stderr, got: %s", stderr)
	}
}

func TestCLIScanNonexistentDir(t *testing.T) {
	_, stderr, err := runBinary("scan", "/no/such/path/exists")
	if err == nil {
		t.Fatal("expected non-zero exit for scan on nonexistent directory")
	}
	if !strings.Contains(stderr, "error") {
		t.Errorf("expected error message on stderr, got: %s", stderr)
	}
}

// --- lookup command ---

func TestCLILookupText(t *testing.T) {
	dir := makeTestDir(t)
	// Scan first.
	if _, _, err := runBinary("scan", dir); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	stdout, _, err := runBinary("lookup", "helper", "--root", dir)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if !strings.Contains(stdout, "helper") {
		t.Errorf("expected 'helper' in lookup output, got: %s", stdout)
	}
}

func TestCLILookupJSON(t *testing.T) {
	dir := makeTestDir(t)
	if _, _, err := runBinary("scan", dir); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	stdout, _, err := runBinary("lookup", "helper", "--root", dir, "--json")
	if err != nil {
		t.Fatalf("lookup --json failed: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, stdout)
	}
	if len(results) == 0 {
		t.Error("expected at least one result for 'helper' query")
	}
}

func TestCLILookupNoMatches(t *testing.T) {
	dir := makeTestDir(t)
	if _, _, err := runBinary("scan", dir); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	stdout, _, err := runBinary("lookup", "zzzznonexistent", "--root", dir)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if !strings.Contains(stdout, "no matches found") {
		t.Errorf("expected 'no matches found', got: %s", stdout)
	}
}

func TestCLILookupNoMatchesJSON(t *testing.T) {
	dir := makeTestDir(t)
	if _, _, err := runBinary("scan", dir); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	stdout, _, err := runBinary("lookup", "zzzznonexistent", "--root", dir, "--json")
	if err != nil {
		t.Fatalf("lookup --json failed: %v", err)
	}
	var results []interface{}
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, stdout)
	}
	if len(results) != 0 {
		t.Errorf("expected empty array, got %d results", len(results))
	}
}

func TestCLILookupEmptyQuery(t *testing.T) {
	dir := makeTestDir(t)
	if _, _, err := runBinary("scan", dir); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	_, stderr, err := runBinary("lookup", "", "--root", dir)
	if err == nil {
		t.Fatal("expected non-zero exit for empty query")
	}
	if !strings.Contains(stderr, "empty") {
		t.Errorf("expected 'empty' in error, got: %s", stderr)
	}
}

func TestCLILookupMaxFlag(t *testing.T) {
	dir := makeTestDir(t)
	if _, _, err := runBinary("scan", dir); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	// Query "." which matches all files, limit to 1.
	stdout, _, err := runBinary("lookup", ".", "--root", dir, "--max", "1")
	if err != nil {
		t.Fatalf("lookup --max failed: %v", err)
	}
	// Count non-empty lines (excluding the "... and N more" line).
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	resultLines := 0
	for _, l := range lines {
		if l != "" && !strings.HasPrefix(l, "...") {
			resultLines++
		}
	}
	if resultLines > 1 {
		t.Errorf("expected at most 1 result line, got %d:\n%s", resultLines, stdout)
	}
}

func TestCLILookupNoIndex(t *testing.T) {
	dir := t.TempDir() // Empty, no index.
	_, stderr, err := runBinary("lookup", "test", "--root", dir)
	if err == nil {
		t.Fatal("expected non-zero exit when no index exists")
	}
	if !strings.Contains(stderr, "error") {
		t.Errorf("expected error message, got: %s", stderr)
	}
}

// --- version command ---

func TestCLIVersionText(t *testing.T) {
	stdout, _, err := runBinary("version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(stdout, "v0.1.0") {
		t.Errorf("expected 'v0.1.0' in output, got: %s", stdout)
	}
}

func TestCLIVersionJSON(t *testing.T) {
	stdout, _, err := runBinary("version", "--json")
	if err != nil {
		t.Fatalf("version --json failed: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, stdout)
	}
	if v, ok := result["version"]; !ok || v != "v0.1.0" {
		t.Errorf("expected version 'v0.1.0', got: %v", result)
	}
}

// --- error handling ---

func TestCLINoArgs(t *testing.T) {
	_, stderr, err := runBinary()
	if err == nil {
		t.Fatal("expected non-zero exit with no args")
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage message on stderr, got: %s", stderr)
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	_, stderr, err := runBinary("foobar")
	if err == nil {
		t.Fatal("expected non-zero exit for unknown command")
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("expected 'unknown command' on stderr, got: %s", stderr)
	}
}

// --- outline command ---

func TestCLIOutlineText(t *testing.T) {
	dir := makeTestDir(t)
	goFile := filepath.Join(dir, "main.go")
	stdout, _, err := runBinary("outline", goFile)
	if err != nil {
		t.Fatalf("outline failed: %v", err)
	}
	if !strings.Contains(stdout, "func main()") {
		t.Errorf("expected 'func main()' in outline output, got: %s", stdout)
	}
}

func TestCLIOutlineJSON(t *testing.T) {
	dir := makeTestDir(t)
	goFile := filepath.Join(dir, "main.go")
	stdout, _, err := runBinary("outline", goFile, "--json")
	if err != nil {
		t.Fatalf("outline --json failed: %v", err)
	}
	var symbols []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &symbols); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, stdout)
	}
	if len(symbols) == 0 {
		t.Error("expected at least one symbol")
	}
}

func TestCLIOutlineNoFile(t *testing.T) {
	_, stderr, err := runBinary("outline")
	if err == nil {
		t.Fatal("expected non-zero exit for outline with no file")
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("expected usage error on stderr, got: %s", stderr)
	}
}

func TestCLIOutlineNonexistentFile(t *testing.T) {
	_, stderr, err := runBinary("outline", "/no/such/file.go")
	if err == nil {
		t.Fatal("expected non-zero exit for outline on nonexistent file")
	}
	if !strings.Contains(stderr, "error") {
		t.Errorf("expected error on stderr, got: %s", stderr)
	}
}

func TestCLIOutlineUnsupportedExt(t *testing.T) {
	dir := makeTestDir(t)
	mdFile := filepath.Join(dir, "README.md")
	_, stderr, err := runBinary("outline", mdFile)
	if err == nil {
		t.Fatal("expected non-zero exit for outline on unsupported file type")
	}
	if !strings.Contains(stderr, "no parser") {
		t.Errorf("expected 'no parser' error on stderr, got: %s", stderr)
	}
}
