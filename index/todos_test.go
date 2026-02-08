package index

import (
	"strings"
	"testing"
)

func TestTodosDetectsAllTags(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

// TODO: implement feature
// FIXME: broken edge case
// HACK: workaround for bug
// XXX: review this
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if result.Total != 4 {
		t.Errorf("Total = %d, want 4", result.Total)
	}
	if len(result.Comments) != 4 {
		t.Fatalf("Comments has %d entries, want 4", len(result.Comments))
	}

	tags := map[string]bool{}
	for _, c := range result.Comments {
		tags[c.Tag] = true
	}
	for _, tag := range []string{"TODO", "FIXME", "HACK", "XXX"} {
		if !tags[tag] {
			t.Errorf("missing tag %s in results", tag)
		}
	}
}

func TestTodosCaseInsensitive(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.py", `# todo: lowercase
# Todo: mixed case
# TODO: uppercase
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	for _, c := range result.Comments {
		if c.Tag != "TODO" {
			t.Errorf("Tag = %q, want TODO (normalized)", c.Tag)
		}
	}
}

func TestTodosMessageExtraction(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main
// TODO: add fuzzy matching support
// FIXME(matt): handle edge case
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if len(result.Comments) != 2 {
		t.Fatalf("Comments has %d entries, want 2", len(result.Comments))
	}

	if result.Comments[0].Message != "add fuzzy matching support" {
		t.Errorf("Message = %q, want %q", result.Comments[0].Message, "add fuzzy matching support")
	}
	if !strings.Contains(result.Comments[1].Message, "handle edge case") {
		t.Errorf("Message = %q, want it to contain %q", result.Comments[1].Message, "handle edge case")
	}
}

func TestTodosTagFilter(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main
// TODO: do something
// FIXME: fix something
// TODO: do another thing
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("FIXME", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if len(result.Comments) != 1 {
		t.Fatalf("Comments has %d entries, want 1", len(result.Comments))
	}
	if result.Comments[0].Tag != "FIXME" {
		t.Errorf("Tag = %q, want FIXME", result.Comments[0].Tag)
	}
	// Total should still reflect all comments found
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3 (all comments regardless of filter)", result.Total)
	}
	// ByTag counts should reflect all comments
	if result.ByTag["TODO"] != 2 {
		t.Errorf("ByTag[TODO] = %d, want 2", result.ByTag["TODO"])
	}
}

func TestTodosTagFilterCaseInsensitive(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main
// TODO: do something
// FIXME: fix something
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("fixme", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if len(result.Comments) != 1 {
		t.Fatalf("Comments has %d entries, want 1", len(result.Comments))
	}
	if result.Comments[0].Tag != "FIXME" {
		t.Errorf("Tag = %q, want FIXME", result.Comments[0].Tag)
	}
}

func TestTodosMaxLimit(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main
// TODO: first
// TODO: second
// TODO: third
// TODO: fourth
// TODO: fifth
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 3)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if len(result.Comments) != 3 {
		t.Errorf("Comments has %d entries, want 3 (limited)", len(result.Comments))
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5 (all found)", result.Total)
	}
}

func TestTodosSkipsBinaryFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main
// TODO: this should be found
`)
	// Create a binary file with a null byte
	mkFile(t, tmp, "binary.dat", "TODO: hidden\x00binary data")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("Total = %d, want 1 (binary file should be skipped)", result.Total)
	}
}

func TestTodosLineNumbers(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func main() {
	// TODO: line 4
}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if len(result.Comments) != 1 {
		t.Fatalf("Comments has %d entries, want 1", len(result.Comments))
	}
	if result.Comments[0].Line != 4 {
		t.Errorf("Line = %d, want 4", result.Comments[0].Line)
	}
}

func TestTodosByTagCounts(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main
// TODO: one
// TODO: two
// FIXME: three
// HACK: four
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if result.ByTag["TODO"] != 2 {
		t.Errorf("ByTag[TODO] = %d, want 2", result.ByTag["TODO"])
	}
	if result.ByTag["FIXME"] != 1 {
		t.Errorf("ByTag[FIXME] = %d, want 1", result.ByTag["FIXME"])
	}
	if result.ByTag["HACK"] != 1 {
		t.Errorf("ByTag[HACK] = %d, want 1", result.ByTag["HACK"])
	}
	if result.ByTag["XXX"] != 0 {
		t.Errorf("ByTag[XXX] = %d, want 0", result.ByTag["XXX"])
	}
}

func TestTodosMultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.go", `package main
// TODO: in file a
`)
	mkFile(t, tmp, "b.go", `package main
// FIXME: in file b
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	// Results should be sorted by path
	if result.Comments[0].Path > result.Comments[1].Path {
		t.Errorf("comments not sorted by path: %s > %s", result.Comments[0].Path, result.Comments[1].Path)
	}
}

func TestTodosNoResults(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func main() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Todos("", 100)
	if err != nil {
		t.Fatalf("Todos() error: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if len(result.Comments) != 0 {
		t.Errorf("Comments has %d entries, want 0", len(result.Comments))
	}
}

func TestFormatTodosEmpty(t *testing.T) {
	result := &TodosResult{
		Comments: nil,
		Total:    0,
		ByTag:    map[string]int{},
	}
	out := FormatTodos(result)
	if !strings.Contains(out, "No TODO") {
		t.Errorf("output missing 'No TODO': %s", out)
	}
}

func TestFormatTodosWithResults(t *testing.T) {
	result := &TodosResult{
		Comments: []TodoComment{
			{Path: "main.go", Line: 10, Tag: "TODO", Message: "add feature", Content: "// TODO: add feature"},
			{Path: "lib.go", Line: 5, Tag: "FIXME", Message: "broken", Content: "// FIXME: broken"},
		},
		Total: 2,
		ByTag: map[string]int{"TODO": 1, "FIXME": 1, "HACK": 0, "XXX": 0},
	}
	out := FormatTodos(result)
	if !strings.Contains(out, "main.go:10") {
		t.Errorf("output missing 'main.go:10': %s", out)
	}
	if !strings.Contains(out, "lib.go:5") {
		t.Errorf("output missing 'lib.go:5': %s", out)
	}
	if !strings.Contains(out, "Summary:") {
		t.Errorf("output missing 'Summary:': %s", out)
	}
	if !strings.Contains(out, "1 TODO") {
		t.Errorf("output missing '1 TODO': %s", out)
	}
	if !strings.Contains(out, "1 FIXME") {
		t.Errorf("output missing '1 FIXME': %s", out)
	}
}
