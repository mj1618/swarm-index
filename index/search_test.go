package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchHappyPath(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")
	mkFile(t, tmp, "lib/util.go", "package lib\n\nfunc Helper() string {\n\treturn \"ok\"\n}\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if err := idx.Save(tmp); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	matches, err := loaded.Search("hello", 50)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("Search('hello') = %d matches, want 1", len(matches))
	}
	if matches[0].Path != "main.go" {
		t.Errorf("match path = %q, want %q", matches[0].Path, "main.go")
	}
	if matches[0].Line != 4 {
		t.Errorf("match line = %d, want 4", matches[0].Line)
	}
	if matches[0].Content != `fmt.Println("hello")` {
		t.Errorf("match content = %q, want %q", matches[0].Content, `fmt.Println("hello")`)
	}
}

func TestSearchRegex(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n\nfunc main() {}\n\nfunc helper() {}\n")
	mkFile(t, tmp, "lib.go", "package main\n\nvar x = 1\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	matches, err := idx.Search(`func\s+\w+`, 50)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("Search('func\\s+\\w+') = %d matches, want 2", len(matches))
	}
}

func TestSearchMaxResults(t *testing.T) {
	tmp := t.TempDir()
	// Create a file with many matching lines.
	content := ""
	for i := 0; i < 20; i++ {
		content += "match line here\n"
	}
	mkFile(t, tmp, "many.txt", content)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	matches, err := idx.Search("match", 5)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(matches) != 5 {
		t.Fatalf("Search with max 5 = %d matches, want 5", len(matches))
	}
}

func TestSearchNoMatches(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n\nfunc main() {}\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	matches, err := idx.Search("nonexistent_string_xyz", 50)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("Search('nonexistent') = %d matches, want 0", len(matches))
	}
}

func TestSearchBinaryFileSkipped(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "text.txt", "findme here\n")

	// Create a binary file with null bytes that also contains the search term.
	binPath := filepath.Join(tmp, "binary.dat")
	binContent := []byte("findme\x00\x00\x00binary data")
	if err := os.WriteFile(binPath, binContent, 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	matches, err := idx.Search("findme", 50)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("Search('findme') = %d matches, want 1 (binary should be skipped)", len(matches))
	}
	if matches[0].Path != "text.txt" {
		t.Errorf("match path = %q, want %q", matches[0].Path, "text.txt")
	}
}

func TestSearchInvalidRegex(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main\n")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	_, err = idx.Search("[invalid", 50)
	if err == nil {
		t.Fatal("Search() should return error for invalid regex")
	}
}
