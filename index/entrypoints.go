package index

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EntryPoint represents a detected entry point in the codebase.
type EntryPoint struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Kind      string `json:"kind"`      // "main", "route", "cli", "init"
	Signature string `json:"signature"` // the matching line, trimmed
}

// EntryPointsResult holds the collected entry points.
type EntryPointsResult struct {
	EntryPoints []EntryPoint `json:"entryPoints"`
	Total       int          `json:"total"`
}

// entryPointPattern ties a compiled regex to a kind classification.
type entryPointPattern struct {
	re   *regexp.Regexp
	kind string
}

// testFilePattern matches common test file naming conventions.
var testFilePattern = regexp.MustCompile(`(?i)(_test\.go|\.test\.[jt]sx?|\.spec\.[jt]sx?|test_[^/]*\.py)$`)

// patterns grouped by file extension.
var entryPointPatterns = map[string][]entryPointPattern{
	".go": {
		// main
		{regexp.MustCompile(`^\s*func\s+main\s*\(`), "main"},
		// init
		{regexp.MustCompile(`^\s*func\s+init\s*\(`), "init"},
		// route — stdlib
		{regexp.MustCompile(`\bhttp\.HandleFunc\s*\(`), "route"},
		{regexp.MustCompile(`\bhttp\.Handle\s*\(`), "route"},
		{regexp.MustCompile(`\bmux\.Handle`), "route"},
		{regexp.MustCompile(`\brouter\.HandleFunc\s*\(`), "route"},
		// route — popular routers (chi, echo, gin, fiber)
		{regexp.MustCompile(`\b[re]\.(?:GET|POST|PUT|DELETE|PATCH)\s*\(`), "route"},
		{regexp.MustCompile(`\bapp\.(?:Get|Post|Put|Delete|Patch)\s*\(`), "route"},
		// cli — cobra
		{regexp.MustCompile(`cobra\.Command\s*\{`), "cli"},
		{regexp.MustCompile(`\bAddCommand\s*\(`), "cli"},
		// cli — stdlib flag
		{regexp.MustCompile(`\bflag\.(?:String|Bool|Int|Float64|Duration)\s*\(`), "cli"},
	},
	".py": {
		// main
		{regexp.MustCompile(`^\s*if\s+__name__\s*==`), "main"},
		// route — Flask/FastAPI
		{regexp.MustCompile(`@app\.(?:route|get|post|put|delete|patch)\s*\(`), "route"},
		{regexp.MustCompile(`@router\.(?:get|post|put|delete|patch)\s*\(`), "route"},
		// route — Django
		{regexp.MustCompile(`\bpath\s*\(`), "route"},
		// cli — argparse
		{regexp.MustCompile(`\.add_argument\s*\(`), "cli"},
		{regexp.MustCompile(`\.add_subparsers\s*\(`), "cli"},
		// cli — click
		{regexp.MustCompile(`@click\.(?:command|group)`), "cli"},
		// init — framework bootstrap
		{regexp.MustCompile(`\bFlask\s*\(`), "init"},
		{regexp.MustCompile(`\bFastAPI\s*\(`), "init"},
		{regexp.MustCompile(`\bdef\s+setup\s*\(`), "init"},
	},
	".js": {
		// main — server bootstrap
		{regexp.MustCompile(`\bcreateServer\s*\(`), "main"},
		{regexp.MustCompile(`\.listen\s*\(`), "main"},
		{regexp.MustCompile(`\bserve\s*\(`), "main"},
		// route — Express
		{regexp.MustCompile(`\b(?:app|router)\.(?:get|post|put|delete|patch|use)\s*\(`), "route"},
		// cli — Commander/yargs
		{regexp.MustCompile(`\.command\s*\(`), "cli"},
		// init — React/Vue
		{regexp.MustCompile(`\bcreateApp\s*\(`), "init"},
		{regexp.MustCompile(`\bcreateRoot\s*\(`), "init"},
		{regexp.MustCompile(`\bReactDOM\.render\s*\(`), "init"},
	},
	".rs": {
		{regexp.MustCompile(`^\s*fn\s+main\s*\(`), "main"},
	},
	".java": {
		{regexp.MustCompile(`public\s+static\s+void\s+main\s*\(`), "main"},
	},
}

func init() {
	// .ts/.tsx/.jsx share the JS patterns
	entryPointPatterns[".ts"] = entryPointPatterns[".js"]
	entryPointPatterns[".tsx"] = entryPointPatterns[".js"]
	entryPointPatterns[".jsx"] = entryPointPatterns[".js"]
}

// EntryPoints scans indexed files for executable entry points.
// kind filters by entry-point kind ("main", "route", "cli", "init"); empty means all.
// max limits the returned results; 0 means default of 100.
func (idx *Index) EntryPoints(kind string, max int) (*EntryPointsResult, error) {
	if max <= 0 {
		max = 100
	}
	kind = strings.ToLower(kind)

	var all []EntryPoint

	paths := idx.FilePaths()
	sort.Strings(paths)

	for _, relPath := range paths {
		// Skip test files
		if testFilePattern.MatchString(relPath) {
			continue
		}

		ext := strings.ToLower(filepath.Ext(relPath))
		patterns, ok := entryPointPatterns[ext]
		if !ok {
			continue
		}

		absPath := filepath.Join(idx.Root, relPath)
		found := entryPointsInFile(absPath, relPath, patterns)
		all = append(all, found...)
	}

	// Sort by kind then path then line
	kindOrder := map[string]int{"main": 0, "route": 1, "cli": 2, "init": 3}
	sort.Slice(all, func(i, j int) bool {
		ki, kj := kindOrder[all[i].Kind], kindOrder[all[j].Kind]
		if ki != kj {
			return ki < kj
		}
		if all[i].Path != all[j].Path {
			return all[i].Path < all[j].Path
		}
		return all[i].Line < all[j].Line
	})

	// Filter by kind if specified
	if kind != "" {
		var filtered []EntryPoint
		for _, ep := range all {
			if ep.Kind == kind {
				filtered = append(filtered, ep)
			}
		}
		all = filtered
	}

	total := len(all)

	// Limit results
	if len(all) > max {
		all = all[:max]
	}

	if all == nil {
		all = []EntryPoint{}
	}

	return &EntryPointsResult{
		EntryPoints: all,
		Total:       total,
	}, nil
}

// entryPointsInFile scans a single file for entry-point patterns.
func entryPointsInFile(fullPath, relPath string, patterns []entryPointPattern) []EntryPoint {
	f, err := openTextFile(fullPath)
	if err != nil || f == nil {
		return nil
	}
	defer f.Close()

	var results []EntryPoint
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, p := range patterns {
			if p.re.MatchString(line) {
				results = append(results, EntryPoint{
					Path:      relPath,
					Line:      lineNum,
					Kind:      p.kind,
					Signature: strings.TrimSpace(line),
				})
				break // only match one pattern per line
			}
		}
	}
	return results
}

// FormatEntryPoints returns a human-readable rendering of entry points.
func FormatEntryPoints(r *EntryPointsResult) string {
	var b strings.Builder

	if len(r.EntryPoints) == 0 {
		b.WriteString("No entry points found\n")
		return b.String()
	}

	// Group by kind
	groups := map[string][]EntryPoint{}
	for _, ep := range r.EntryPoints {
		groups[ep.Kind] = append(groups[ep.Kind], ep)
	}

	headers := map[string]string{
		"main":  "Main entry points",
		"route": "Route handlers",
		"cli":   "CLI commands",
		"init":  "Init functions",
	}

	first := true
	for _, kind := range []string{"main", "route", "cli", "init"} {
		eps, ok := groups[kind]
		if !ok {
			continue
		}
		if !first {
			b.WriteString("\n")
		}
		first = false
		b.WriteString(headers[kind] + ":\n")
		for _, ep := range eps {
			b.WriteString(fmt.Sprintf("  %-30s %s\n", fmt.Sprintf("%s:%d", ep.Path, ep.Line), ep.Signature))
		}
	}

	b.WriteString(fmt.Sprintf("\n%d entry points found\n", r.Total))
	return b.String()
}
