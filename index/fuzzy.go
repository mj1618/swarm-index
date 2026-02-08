package index

import (
	"path/filepath"
	"sort"
	"strings"
)

// ScoredEntry pairs an Entry with a relevance score for ranked results.
type ScoredEntry struct {
	Entry
	Score float64 `json:"score"`
}

// levenshtein computes the Levenshtein edit distance between two strings.
// Returns early if the distance exceeds maxDist, returning maxDist+1.
func levenshtein(a, b string, maxDist int) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	// Quick reject: if length difference alone exceeds maxDist, skip.
	diff := la - lb
	if diff < 0 {
		diff = -diff
	}
	if diff > maxDist {
		return maxDist + 1
	}

	// Single-row DP.
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		cur := make([]int, lb+1)
		cur[0] = i
		rowMin := cur[0]
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := cur[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			v := ins
			if del < v {
				v = del
			}
			if sub < v {
				v = sub
			}
			cur[j] = v
			if v < rowMin {
				rowMin = v
			}
		}
		if rowMin > maxDist {
			return maxDist + 1
		}
		prev = cur
	}
	return prev[lb]
}

// scoreName computes a relevance score for a query against a file name and path.
// Returns 0 if there is no match at all.
func scoreName(query, name, path string) float64 {
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	pathLower := strings.ToLower(path)

	// Strip extension for name comparison.
	nameNoExt := strings.TrimSuffix(nameLower, strings.ToLower(filepath.Ext(name)))

	// 1. Exact name match (without extension).
	if q == nameNoExt {
		return 100
	}

	// 2. Exact name match (with extension).
	if q == nameLower {
		return 95
	}

	// 3. Name prefix match.
	if strings.HasPrefix(nameNoExt, q) {
		return 80
	}

	// 4. Name substring match.
	if strings.Contains(nameLower, q) {
		return 60
	}

	// 5. Path substring match (but not in name).
	if strings.Contains(pathLower, q) {
		return 40
	}

	// 6. Fuzzy match on filename (Levenshtein distance ≤ 2).
	const maxDist = 2
	// Only attempt fuzzy if the name length is in a reasonable range.
	if len(nameNoExt) <= len(q)+maxDist && len(q) <= len(nameNoExt)+maxDist {
		dist := levenshtein(q, nameNoExt, maxDist)
		if dist <= maxDist {
			// Scale score: distance 0 = 35, distance 1 = 28, distance 2 = 20.
			return 35 - float64(dist)*7.5
		}
	}

	return 0
}

// matchFuzzy returns entries scored and sorted by relevance. Entries with
// score 0 are excluded. Tie-breaking: shorter paths rank higher.
func (idx *Index) matchFuzzy(query string) []ScoredEntry {
	var scored []ScoredEntry
	for _, e := range idx.Entries {
		s := scoreName(query, e.Name, e.Path)
		if s > 0 {
			scored = append(scored, ScoredEntry{Entry: e, Score: s})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return len(scored[i].Path) < len(scored[j].Path)
	})
	return scored
}
