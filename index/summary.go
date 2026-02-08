package index

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// languageMap maps file extensions to human-readable language names.
var languageMap = map[string]string{
	".go":    "Go",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".tsx":   "TypeScript",
	".jsx":   "JavaScript",
	".py":    "Python",
	".rs":    "Rust",
	".java":  "Java",
	".rb":    "Ruby",
	".c":     "C",
	".h":     "C",
	".cpp":   "C++",
	".hpp":   "C++",
	".cs":    "C#",
	".swift": "Swift",
	".kt":    "Kotlin",
	".md":    "Markdown",
	".json":  "JSON",
	".yaml":  "YAML",
	".yml":   "YAML",
	".toml":  "TOML",
	".html":  "HTML",
	".css":   "CSS",
	".scss":  "SCSS",
	".sh":    "Shell",
	".sql":   "SQL",
}

// entryPointNames lists known entry-point filenames.
var entryPointNames = map[string]bool{
	"main.go":    true,
	"main.py":    true,
	"main.rs":    true,
	"main.ts":    true,
	"main.js":    true,
	"index.ts":   true,
	"index.js":   true,
	"index.html": true,
	"app.py":     true,
	"app.ts":     true,
	"app.js":     true,
	"manage.py":  true,
	"Main.java":  true,
	"Program.cs": true,
	"server.go":  true,
	"server.ts":  true,
	"server.js":  true,
}

// entryPointPaths lists known entry-point path patterns (checked via suffix).
var entryPointPaths = []string{
	"cmd/main.go",
}

// manifestNames lists known dependency manifest filenames.
var manifestNames = map[string]bool{
	"go.mod":            true,
	"go.sum":            true,
	"package.json":      true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"requirements.txt":  true,
	"setup.py":          true,
	"pyproject.toml":    true,
	"Pipfile":           true,
	"Cargo.toml":        true,
	"Cargo.lock":        true,
	"Gemfile":           true,
	"Gemfile.lock":      true,
	"pom.xml":           true,
	"build.gradle":      true,
	"build.gradle.kts":  true,
	"composer.json":     true,
	"Makefile":          true,
	"CMakeLists.txt":    true,
}

// LanguageStat holds file count and percentage for a language.
type LanguageStat struct {
	Files      int     `json:"files"`
	Percentage float64 `json:"percentage"`
}

// SummaryResult holds the complete project summary.
type SummaryResult struct {
	Root           string                  `json:"root"`
	FileCount      int                     `json:"fileCount"`
	LOC            int                     `json:"loc"`
	PackageCount   int                     `json:"packageCount"`
	Languages      map[string]LanguageStat `json:"languages"`
	EntryPoints    []string                `json:"entryPoints"`
	Manifests      []string                `json:"manifests"`
	TopDirectories []string                `json:"topDirectories"`
}

// Summary computes a project overview from the index, reading files from disk
// to count lines of code.
func (idx *Index) Summary() SummaryResult {
	seen := make(map[string]struct{})
	langCounts := make(map[string]int)
	totalFiles := 0
	totalLOC := 0
	var entryPoints []string
	var manifests []string
	topDirs := make(map[string]struct{})

	for _, e := range idx.Entries {
		if _, ok := seen[e.Path]; ok {
			continue
		}
		seen[e.Path] = struct{}{}
		totalFiles++

		// Language stats
		ext := filepath.Ext(e.Name)
		lang := "(other)"
		if l, ok := languageMap[ext]; ok {
			lang = l
		}
		langCounts[lang]++

		// LOC counting
		fullPath := filepath.Join(idx.Root, e.Path)
		if data, err := os.ReadFile(fullPath); err == nil {
			totalLOC += countLines(data)
		}

		// Entry point detection
		if isEntryPoint(e) {
			entryPoints = append(entryPoints, e.Path)
		}

		// Manifest detection
		if manifestNames[e.Name] {
			manifests = append(manifests, e.Path)
		}

		// Top-level directories
		parts := strings.SplitN(e.Path, string(filepath.Separator), 2)
		if len(parts) == 2 {
			topDirs[parts[0]] = struct{}{}
		}
	}

	// Build language stats with percentages
	languages := make(map[string]LanguageStat, len(langCounts))
	for lang, count := range langCounts {
		pct := 0.0
		if totalFiles > 0 {
			pct = math.Round(float64(count)*1000.0/float64(totalFiles)) / 10
		}
		languages[lang] = LanguageStat{Files: count, Percentage: pct}
	}

	// Sort top directories
	sortedDirs := make([]string, 0, len(topDirs))
	for d := range topDirs {
		sortedDirs = append(sortedDirs, d+"/")
	}
	sort.Strings(sortedDirs)

	sort.Strings(entryPoints)
	sort.Strings(manifests)

	return SummaryResult{
		Root:           idx.Root,
		FileCount:      totalFiles,
		LOC:            totalLOC,
		PackageCount:   idx.PackageCount(),
		Languages:      languages,
		EntryPoints:    entryPoints,
		Manifests:      manifests,
		TopDirectories: sortedDirs,
	}
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	n := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}

func isEntryPoint(e Entry) bool {
	if entryPointNames[e.Name] {
		return true
	}
	for _, p := range entryPointPaths {
		if strings.HasSuffix(e.Path, p) {
			return true
		}
	}
	return false
}

// FormatSummary returns a human-readable text rendering of the summary.
func FormatSummary(s SummaryResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Project summary for %s\n", s.Root))
	b.WriteString(strings.Repeat("=", len("Project summary for ")+len(s.Root)) + "\n\n")
	b.WriteString(fmt.Sprintf("%d files (%d LOC), %d packages\n", s.FileCount, s.LOC, s.PackageCount))

	if len(s.Languages) > 0 {
		b.WriteString("\nLanguages:\n")
		type langEntry struct {
			name string
			stat LanguageStat
		}
		sorted := make([]langEntry, 0, len(s.Languages))
		for name, stat := range s.Languages {
			sorted = append(sorted, langEntry{name, stat})
		}
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].stat.Files != sorted[j].stat.Files {
				return sorted[i].stat.Files > sorted[j].stat.Files
			}
			return sorted[i].name < sorted[j].name
		})

		maxLen := 0
		for _, le := range sorted {
			if len(le.name) > maxLen {
				maxLen = len(le.name)
			}
		}

		for _, le := range sorted {
			label := "files"
			if le.stat.Files == 1 {
				label = "file "
			}
			b.WriteString(fmt.Sprintf("  %-*s  %4d %s  (%5.1f%%)\n",
				maxLen, le.name, le.stat.Files, label, le.stat.Percentage))
		}
	}

	if len(s.EntryPoints) > 0 {
		b.WriteString("\nEntry points:\n")
		for _, ep := range s.EntryPoints {
			b.WriteString(fmt.Sprintf("  %s\n", ep))
		}
	}

	if len(s.Manifests) > 0 {
		b.WriteString("\nDependency manifests:\n")
		for _, m := range s.Manifests {
			b.WriteString(fmt.Sprintf("  %s\n", m))
		}
	}

	if len(s.TopDirectories) > 0 {
		b.WriteString("\nTop directories:\n")
		for _, d := range s.TopDirectories {
			b.WriteString(fmt.Sprintf("  %s\n", d))
		}
	}

	return b.String()
}
