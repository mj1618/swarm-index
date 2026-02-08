package index

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"kitten", "sitting", 3},
		{"handler", "hadnler", 2},
		{"config", "config", 0},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
		{"abc", "axyz", 3},
	}

	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b, 10)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestLevenshteinMaxDist(t *testing.T) {
	// When maxDist is 2, distances > 2 should return 3.
	got := levenshtein("kitten", "sitting", 2)
	if got <= 2 {
		t.Errorf("levenshtein(%q, %q, maxDist=2) = %d, want > 2", "kitten", "sitting", got)
	}

	// Distance exactly 2 should still be computed correctly.
	got = levenshtein("handler", "hadnler", 2)
	if got != 2 {
		t.Errorf("levenshtein(%q, %q, maxDist=2) = %d, want 2", "handler", "hadnler", got)
	}
}

func TestScoreName(t *testing.T) {
	tests := []struct {
		query string
		name  string
		path  string
		want  float64
	}{
		// Exact name match (without extension) = 100.
		{"config", "config.go", "config.go", 100},
		{"handler", "handler.go", "api/handler.go", 100},

		// Exact name match (with extension) = 95.
		{"config.go", "config.go", "config.go", 95},

		// Name prefix match = 80.
		{"hand", "handler.go", "api/handler.go", 80},
		{"config", "config_helper.go", "config_helper.go", 80},

		// Name substring match = 60.
		{"auth", "oauth.go", "pkg/oauth.go", 60},
		{"andl", "handler.go", "api/handler.go", 60},

		// Path substring match = 40.
		{"api/", "handler.go", "api/handler.go", 40},

		// Fuzzy match (distance 2) = 20.
		{"hadnler", "handler.go", "api/handler.go", 20},

		// Fuzzy match (distance 3) — excluded.
		{"hadnelr", "handler.go", "api/handler.go", 0},

		// No match = 0.
		{"zzzzz", "handler.go", "api/handler.go", 0},
	}

	for _, tt := range tests {
		got := scoreName(tt.query, tt.name, tt.path)
		if got != tt.want {
			t.Errorf("scoreName(%q, %q, %q) = %v, want %v", tt.query, tt.name, tt.path, got, tt.want)
		}
	}
}

func TestScoreNamePriority(t *testing.T) {
	// Verify that exact > prefix > substring > path > fuzzy.
	exact := scoreName("config", "config.go", "config.go")
	prefix := scoreName("conf", "config.go", "config.go")
	substr := scoreName("onfi", "config.go", "config.go")
	path := scoreName("src/", "config.go", "src/config.go")
	fuzzy := scoreName("confg", "config.go", "config.go")

	if !(exact > prefix && prefix > substr && substr > path && path > fuzzy && fuzzy > 0) {
		t.Errorf("score priority violated: exact=%v, prefix=%v, substr=%v, path=%v, fuzzy=%v",
			exact, prefix, substr, path, fuzzy)
	}
}

func TestMatchFuzzy(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "config.go", Kind: "file", Path: "config.go", Package: "(root)"},
			{Name: "config_helper.go", Kind: "file", Path: "lib/config_helper.go", Package: "lib"},
			{Name: "reconfigure.go", Kind: "file", Path: "pkg/reconfigure.go", Package: "pkg"},
			{Name: "handler.go", Kind: "file", Path: "api/handler.go", Package: "api"},
			{Name: "unrelated.go", Kind: "file", Path: "unrelated.go", Package: "(root)"},
		},
	}

	results := idx.matchFuzzy("config")
	if len(results) < 3 {
		t.Fatalf("matchFuzzy('config') returned %d results, want >= 3", len(results))
	}

	// First result should be the exact name match.
	if results[0].Entry.Name != "config.go" {
		t.Errorf("matchFuzzy('config')[0].Name = %q, want %q", results[0].Entry.Name, "config.go")
	}

	// Second should be prefix match.
	if results[1].Entry.Name != "config_helper.go" {
		t.Errorf("matchFuzzy('config')[1].Name = %q, want %q", results[1].Entry.Name, "config_helper.go")
	}

	// Third should be substring match.
	if results[2].Entry.Name != "reconfigure.go" {
		t.Errorf("matchFuzzy('config')[2].Name = %q, want %q", results[2].Entry.Name, "reconfigure.go")
	}

	// "unrelated.go" should not appear.
	for _, r := range results {
		if r.Entry.Name == "unrelated.go" {
			t.Error("matchFuzzy('config') should not include 'unrelated.go'")
		}
	}
}

func TestMatchFuzzyTypo(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "handler.go", Kind: "file", Path: "api/handler.go", Package: "api"},
			{Name: "helper.go", Kind: "file", Path: "lib/helper.go", Package: "lib"},
		},
	}

	results := idx.matchFuzzy("hadnler")
	if len(results) == 0 {
		t.Fatal("matchFuzzy('hadnler') returned 0 results, want >= 1 (should find 'handler.go')")
	}
	if results[0].Entry.Name != "handler.go" {
		t.Errorf("matchFuzzy('hadnler')[0].Name = %q, want %q", results[0].Entry.Name, "handler.go")
	}
}

func TestMatchRankedResults(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "reconfigure.go", Kind: "file", Path: "pkg/reconfigure.go", Package: "pkg"},
			{Name: "config.go", Kind: "file", Path: "config.go", Package: "(root)"},
			{Name: "myconfig_helper.go", Kind: "file", Path: "lib/myconfig_helper.go", Package: "lib"},
		},
	}

	results := idx.Match("config")
	if len(results) < 3 {
		t.Fatalf("Match('config') returned %d results, want 3", len(results))
	}

	// Exact match should be first.
	if results[0].Name != "config.go" {
		t.Errorf("Match('config')[0].Name = %q, want %q", results[0].Name, "config.go")
	}
}

func TestMatchExactFallback(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "config.go", Kind: "file", Path: "config.go", Package: "(root)"},
			{Name: "handler.go", Kind: "file", Path: "api/handler.go", Package: "api"},
		},
	}

	// MatchExact should return substring matches without ranking.
	results := idx.MatchExact("config")
	if len(results) != 1 {
		t.Errorf("MatchExact('config') returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Name != "config.go" {
		t.Errorf("MatchExact('config')[0].Name = %q, want %q", results[0].Name, "config.go")
	}

	// MatchExact should NOT find fuzzy matches (typos).
	results = idx.MatchExact("hadnler")
	if len(results) != 0 {
		t.Errorf("MatchExact('hadnler') returned %d results, want 0 (no fuzzy)", len(results))
	}
}

func TestMatchEmptyQuery(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "a.go", Kind: "file", Path: "a.go", Package: "(root)"},
			{Name: "b.go", Kind: "file", Path: "b.go", Package: "(root)"},
		},
	}

	// Empty query should match everything in both modes.
	results := idx.Match("")
	if len(results) != 2 {
		t.Errorf("Match('') returned %d results, want 2", len(results))
	}

	results = idx.MatchExact("")
	if len(results) != 2 {
		t.Errorf("MatchExact('') returned %d results, want 2", len(results))
	}
}

func TestMatchScoredJSON(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "config.go", Kind: "file", Path: "config.go", Package: "(root)"},
		},
	}

	scored := idx.MatchScored("config")
	if len(scored) != 1 {
		t.Fatalf("MatchScored('config') returned %d results, want 1", len(scored))
	}
	if scored[0].Score != 100 {
		t.Errorf("MatchScored('config')[0].Score = %v, want 100", scored[0].Score)
	}
}

func TestMatchFuzzyDistanceThreshold(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "handler.go", Kind: "file", Path: "handler.go", Package: "(root)"},
		},
	}

	// "hadnelr" has distance 3 from "handler", should be excluded.
	results := idx.matchFuzzy("hadnelr")
	if len(results) != 0 {
		t.Errorf("matchFuzzy('hadnelr') returned %d results, want 0 (distance > 2)", len(results))
	}
}

func TestMatchTieBreakingShorterPath(t *testing.T) {
	idx := &Index{
		Entries: []Entry{
			{Name: "config.go", Kind: "file", Path: "deeply/nested/config.go", Package: "nested"},
			{Name: "config.go", Kind: "file", Path: "config.go", Package: "(root)"},
		},
	}

	results := idx.Match("config")
	if len(results) < 2 {
		t.Fatalf("Match('config') returned %d results, want 2", len(results))
	}

	// Shorter path should come first when scores are equal.
	if results[0].Path != "config.go" {
		t.Errorf("Match('config')[0].Path = %q, want %q (shorter path first)", results[0].Path, "config.go")
	}
}
