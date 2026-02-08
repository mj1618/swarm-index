package index

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestShowFile_FullFile(t *testing.T) {
	dir := t.TempDir()
	content := "line one\nline two\nline three\n"
	path := writeTestFile(t, dir, "test.txt", content)

	result, err := ShowFile(path, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalLines != 3 {
		t.Errorf("expected 3 total lines, got %d", result.TotalLines)
	}
	if len(result.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result.Lines))
	}
	if result.Lines[0].Number != 1 || result.Lines[0].Content != "line one" {
		t.Errorf("unexpected first line: %+v", result.Lines[0])
	}
	if result.Lines[2].Number != 3 || result.Lines[2].Content != "line three" {
		t.Errorf("unexpected third line: %+v", result.Lines[2])
	}
}

func TestShowFile_LineRange(t *testing.T) {
	dir := t.TempDir()
	lines := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	path := writeTestFile(t, dir, "ten.txt", lines)

	result, err := ShowFile(path, 3, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(result.Lines))
	}
	if result.Lines[0].Number != 3 || result.Lines[0].Content != "3" {
		t.Errorf("unexpected first line: %+v", result.Lines[0])
	}
	if result.Lines[4].Number != 7 || result.Lines[4].Content != "7" {
		t.Errorf("unexpected last line: %+v", result.Lines[4])
	}
	if result.StartLine != 3 || result.EndLine != 7 {
		t.Errorf("unexpected range: %d-%d", result.StartLine, result.EndLine)
	}
}

func TestShowFile_SingleLine(t *testing.T) {
	dir := t.TempDir()
	lines := "a\nb\nc\nd\ne\n"
	path := writeTestFile(t, dir, "five.txt", lines)

	result, err := ShowFile(path, 5, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(result.Lines))
	}
	if result.Lines[0].Content != "e" {
		t.Errorf("expected 'e', got %q", result.Lines[0].Content)
	}
}

func TestShowFile_OpenEndedStart(t *testing.T) {
	dir := t.TempDir()
	lines := "1\n2\n3\n4\n5\n"
	path := writeTestFile(t, dir, "five.txt", lines)

	// startLine=3, endLine=0 means from line 3 to end
	result, err := ShowFile(path, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result.Lines))
	}
	if result.Lines[0].Number != 3 {
		t.Errorf("expected start at 3, got %d", result.Lines[0].Number)
	}
	if result.Lines[2].Number != 5 {
		t.Errorf("expected end at 5, got %d", result.Lines[2].Number)
	}
}

func TestShowFile_OpenEndedEnd(t *testing.T) {
	dir := t.TempDir()
	lines := "1\n2\n3\n4\n5\n"
	path := writeTestFile(t, dir, "five.txt", lines)

	// startLine=0, endLine=3 means from line 1 to 3
	result, err := ShowFile(path, 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result.Lines))
	}
	if result.Lines[0].Number != 1 {
		t.Errorf("expected start at 1, got %d", result.Lines[0].Number)
	}
	if result.Lines[2].Number != 3 {
		t.Errorf("expected end at 3, got %d", result.Lines[2].Number)
	}
}

func TestShowFile_StartBeyondEnd(t *testing.T) {
	dir := t.TempDir()
	lines := "1\n2\n3\n"
	path := writeTestFile(t, dir, "three.txt", lines)

	_, err := ShowFile(path, 10, 15)
	if err == nil {
		t.Fatal("expected error for start beyond file length")
	}
}

func TestShowFile_StartGreaterThanEnd(t *testing.T) {
	dir := t.TempDir()
	lines := "1\n2\n3\n4\n5\n"
	path := writeTestFile(t, dir, "five.txt", lines)

	_, err := ShowFile(path, 7, 3)
	if err == nil {
		t.Fatal("expected error for start > end")
	}
}

func TestShowFile_NonexistentFile(t *testing.T) {
	_, err := ShowFile("/no/such/file.txt", 0, 0)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestShowFile_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.dat")
	data := make([]byte, 100)
	data[50] = 0 // null byte
	for i := range data {
		if i != 50 {
			data[i] = 'A'
		}
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ShowFile(path, 0, 0)
	if err == nil {
		t.Fatal("expected error for binary file")
	}
}

func TestShowFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "empty.txt", "")

	result, err := ShowFile(path, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalLines != 0 {
		t.Errorf("expected 0 total lines, got %d", result.TotalLines)
	}
	if len(result.Lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(result.Lines))
	}
}

func TestShowFile_Directory(t *testing.T) {
	dir := t.TempDir()
	_, err := ShowFile(dir, 0, 0)
	if err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestShowFile_EndBeyondFileClamps(t *testing.T) {
	dir := t.TempDir()
	lines := "1\n2\n3\n"
	path := writeTestFile(t, dir, "three.txt", lines)

	// endLine beyond file length should clamp to last line
	result, err := ShowFile(path, 2, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(result.Lines))
	}
	if result.EndLine != 3 {
		t.Errorf("expected EndLine=3, got %d", result.EndLine)
	}
}
