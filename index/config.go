package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ConfigResult holds the detected project toolchain information.
type ConfigResult struct {
	Language       string            `json:"language"`
	Framework      string            `json:"framework"`
	Build          string            `json:"build"`
	Test           string            `json:"test"`
	Lint           string            `json:"lint"`
	Format         string            `json:"format"`
	PackageManager string            `json:"packageManager"`
	ConfigFiles    []string          `json:"configFiles"`
	Scripts        map[string]string `json:"scripts"`
}

// Config detects the project toolchain by inspecting indexed files and
// reading key config files from disk.
func (idx *Index) Config() (*ConfigResult, error) {
	result := &ConfigResult{
		ConfigFiles: []string{},
		Scripts:     map[string]string{},
	}

	// Build a set of indexed filenames for quick lookup.
	// Maps base filename -> first relative path.
	pathSet := make(map[string]string)
	for _, e := range idx.Entries {
		if e.Kind != "file" {
			continue
		}
		base := filepath.Base(e.Path)
		if _, ok := pathSet[base]; !ok {
			pathSet[base] = e.Path
		}
	}

	// 1. Language detection — pick primary language by file count.
	result.Language = idx.detectLanguage()

	// 2. Framework detection — inspect manifest contents.
	result.Framework = idx.detectFramework(pathSet)

	// 3. Build/test/lint/format detection — check for config files.
	idx.detectTools(result, pathSet)

	// 4. Package manager detection.
	result.PackageManager = detectPackageManager(pathSet)

	// 5. Package.json scripts extraction.
	if p, ok := pathSet["package.json"]; ok {
		idx.extractScripts(result, p)
	}

	// Sort config files for deterministic output.
	sort.Strings(result.ConfigFiles)

	return result, nil
}

// detectLanguage picks the primary language by source file count,
// aggregating extensions that map to the same language (e.g. .ts and .tsx).
func (idx *Index) detectLanguage() string {
	counts := idx.ExtensionCounts()

	// Only count source code extensions (not config/data files).
	sourceExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".rs": true, ".java": true, ".rb": true,
		".c": true, ".h": true, ".cpp": true, ".hpp": true,
		".cs": true, ".swift": true, ".kt": true, ".sh": true,
	}

	// Aggregate by language name so e.g. .ts + .tsx both count as TypeScript.
	langCounts := make(map[string]int)
	for ext, count := range counts {
		if !sourceExts[ext] {
			continue
		}
		lang, ok := languageMap[ext]
		if !ok {
			continue
		}
		langCounts[lang] += count
	}

	bestLang := ""
	bestCount := 0
	for lang, count := range langCounts {
		if count > bestCount || (count == bestCount && lang < bestLang) {
			bestCount = count
			bestLang = lang
		}
	}

	return bestLang
}

// frameworkSignal maps a dependency name to its framework name and ecosystem.
type frameworkSignal struct {
	dep       string
	framework string
	ecosystem string
}

var frameworkSignals = []frameworkSignal{
	// JS/TS (order matters — next before react since next includes react)
	{dep: "next", framework: "Next.js", ecosystem: "node"},
	{dep: "react", framework: "React", ecosystem: "node"},
	{dep: "vue", framework: "Vue", ecosystem: "node"},
	{dep: "@angular/core", framework: "Angular", ecosystem: "node"},
	{dep: "express", framework: "Express", ecosystem: "node"},
	{dep: "fastify", framework: "Fastify", ecosystem: "node"},
	// Python
	{dep: "django", framework: "Django", ecosystem: "python"},
	{dep: "flask", framework: "Flask", ecosystem: "python"},
	{dep: "fastapi", framework: "FastAPI", ecosystem: "python"},
	// Rust
	{dep: "actix-web", framework: "Actix", ecosystem: "rust"},
	{dep: "axum", framework: "Axum", ecosystem: "rust"},
	// Go
	{dep: "gin-gonic/gin", framework: "Gin", ecosystem: "go"},
	{dep: "labstack/echo", framework: "Echo", ecosystem: "go"},
	{dep: "gorilla/mux", framework: "Gorilla Mux", ecosystem: "go"},
}

// detectFramework reads manifest files to detect the project framework.
func (idx *Index) detectFramework(pathSet map[string]string) string {
	// Check Node.js (package.json)
	if p, ok := pathSet["package.json"]; ok {
		absPath := filepath.Join(idx.Root, p)
		if content, err := os.ReadFile(absPath); err == nil {
			var pkg struct {
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
			}
			if json.Unmarshal(content, &pkg) == nil {
				allDeps := make(map[string]bool)
				for k := range pkg.Dependencies {
					allDeps[k] = true
				}
				for k := range pkg.DevDependencies {
					allDeps[k] = true
				}
				for _, sig := range frameworkSignals {
					if sig.ecosystem == "node" && allDeps[sig.dep] {
						return sig.framework
					}
				}
			}
		}
	}

	// Check other ecosystems by scanning manifest file contents.
	type manifestCheck struct {
		files     []string
		ecosystem string
	}
	checks := []manifestCheck{
		{files: []string{"requirements.txt", "pyproject.toml"}, ecosystem: "python"},
		{files: []string{"Cargo.toml"}, ecosystem: "rust"},
		{files: []string{"go.mod"}, ecosystem: "go"},
	}
	for _, check := range checks {
		for _, manifest := range check.files {
			if p, ok := pathSet[manifest]; ok {
				absPath := filepath.Join(idx.Root, p)
				if content, err := os.ReadFile(absPath); err == nil {
					lower := strings.ToLower(string(content))
					for _, sig := range frameworkSignals {
						if sig.ecosystem == check.ecosystem && strings.Contains(lower, sig.dep) {
							return sig.framework
						}
					}
				}
			}
		}
	}

	return ""
}

// toolDetector maps a config file pattern to a tool category and tool name.
type toolDetector struct {
	filename string // exact match on base filename
	prefix   string // prefix match on base filename (if filename is empty)
	category string // "build", "test", "lint", "format", "ci"
	tool     string
}

var toolDetectors = []toolDetector{
	// Build
	{filename: "Makefile", category: "build", tool: "make"},
	{filename: "Dockerfile", category: "build", tool: "docker build"},
	{filename: "tsconfig.json", category: "build", tool: "tsc"},
	// Lint
	{prefix: ".eslintrc", category: "lint", tool: "eslint"},
	{prefix: "eslint.config", category: "lint", tool: "eslint"},
	{filename: ".golangci.yml", category: "lint", tool: "golangci-lint"},
	{filename: ".golangci.yaml", category: "lint", tool: "golangci-lint"},
	// Format
	{prefix: ".prettierrc", category: "format", tool: "prettier"},
	{prefix: "prettier.config", category: "format", tool: "prettier"},
	// Test
	{prefix: "jest.config", category: "test", tool: "jest"},
	{prefix: "vitest.config", category: "test", tool: "vitest"},
	{filename: "pytest.ini", category: "test", tool: "pytest"},
}

// detectTools checks for config files in the index and populates build/test/lint/format.
func (idx *Index) detectTools(result *ConfigResult, pathSet map[string]string) {
	// Check exact filename matches and prefix matches.
	for _, e := range idx.Entries {
		if e.Kind != "file" {
			continue
		}
		base := filepath.Base(e.Path)
		for _, det := range toolDetectors {
			matched := false
			if det.filename != "" && base == det.filename {
				matched = true
			} else if det.prefix != "" && strings.HasPrefix(base, det.prefix) {
				matched = true
			}
			if !matched {
				continue
			}
			// Add to config files list.
			result.ConfigFiles = appendUnique(result.ConfigFiles, e.Path)
			// Set the tool if not already set.
			switch det.category {
			case "build":
				if result.Build == "" {
					result.Build = det.tool
				}
			case "test":
				if result.Test == "" {
					result.Test = det.tool
				}
			case "lint":
				if result.Lint == "" {
					result.Lint = det.tool
				}
			case "format":
				if result.Format == "" {
					result.Format = det.tool
				}
			}
		}
	}

	// Check pyproject.toml for [tool.pytest] section.
	if p, ok := pathSet["pyproject.toml"]; ok {
		absPath := filepath.Join(idx.Root, p)
		if content, err := os.ReadFile(absPath); err == nil {
			if strings.Contains(string(content), "[tool.pytest") {
				if result.Test == "" {
					result.Test = "pytest"
				}
				result.ConfigFiles = appendUnique(result.ConfigFiles, p)
			}
		}
	}

	// Check package.json for jest config key.
	if p, ok := pathSet["package.json"]; ok {
		absPath := filepath.Join(idx.Root, p)
		if content, err := os.ReadFile(absPath); err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(content, &pkg) == nil {
				if _, hasJest := pkg["jest"]; hasJest {
					if result.Test == "" {
						result.Test = "jest"
					}
					result.ConfigFiles = appendUnique(result.ConfigFiles, p)
				}
			}
		}
	}

	// Apply language-specific defaults.
	switch result.Language {
	case "Go":
		if result.Test == "" {
			result.Test = "go test ./..."
		}
		if result.Build == "" {
			result.Build = "go build"
		}
		if result.Format == "" {
			result.Format = "gofmt"
		}
	case "Python":
		if result.Test == "" {
			result.Test = "python -m pytest"
		}
	case "Rust":
		if result.Test == "" {
			result.Test = "cargo test"
		}
		if result.Build == "" {
			result.Build = "cargo build"
		}
		if result.Format == "" {
			result.Format = "rustfmt"
		}
	}

	// Add manifest files to config files list.
	for _, name := range []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "requirements.txt"} {
		if p, ok := pathSet[name]; ok {
			result.ConfigFiles = appendUnique(result.ConfigFiles, p)
		}
	}

	// Add CI config files.
	for _, e := range idx.Entries {
		if e.Kind != "file" {
			continue
		}
		if strings.HasPrefix(e.Path, ".github/workflows/") && (strings.HasSuffix(e.Path, ".yml") || strings.HasSuffix(e.Path, ".yaml")) {
			result.ConfigFiles = appendUnique(result.ConfigFiles, e.Path)
		}
	}
}

// detectPackageManager determines the package manager based on lock files and manifests.
func detectPackageManager(pathSet map[string]string) string {
	has := func(name string) bool {
		_, ok := pathSet[name]
		return ok
	}
	// Check lock files first (most specific signal).
	if has("pnpm-lock.yaml") {
		return "pnpm"
	}
	if has("yarn.lock") {
		return "yarn"
	}
	if has("package-lock.json") {
		return "npm"
	}
	if has("package.json") {
		return "npm"
	}
	if has("go.mod") {
		return "go modules"
	}
	if has("Cargo.toml") {
		return "cargo"
	}
	if has("Pipfile") {
		return "pipenv"
	}
	if has("pyproject.toml") {
		return "pip"
	}
	if has("requirements.txt") {
		return "pip"
	}
	if has("Gemfile") {
		return "bundler"
	}
	return ""
}

// extractScripts reads package.json and extracts the scripts field.
func (idx *Index) extractScripts(result *ConfigResult, relPath string) {
	absPath := filepath.Join(idx.Root, relPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(content, &pkg) == nil && pkg.Scripts != nil {
		result.Scripts = pkg.Scripts
	}
}

// appendUnique appends s to slice only if not already present.
func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}

// FormatConfig returns a human-readable text rendering of the config result.
func FormatConfig(r *ConfigResult) string {
	var b strings.Builder

	b.WriteString("Project toolchain\n")
	b.WriteString("=================\n\n")

	writeField := func(label, value string) {
		if value == "" {
			value = "(none detected)"
		}
		b.WriteString(fmt.Sprintf("%-16s %s\n", label+":", value))
	}

	writeField("Language", r.Language)
	writeField("Framework", r.Framework)
	writeField("Build", r.Build)
	writeField("Test", r.Test)
	writeField("Lint", r.Lint)
	writeField("Format", r.Format)
	writeField("Package manager", r.PackageManager)

	if len(r.ConfigFiles) > 0 {
		b.WriteString("\nConfig files:\n")
		for _, f := range r.ConfigFiles {
			b.WriteString(fmt.Sprintf("  %s\n", f))
		}
	}

	if len(r.Scripts) > 0 {
		b.WriteString("\nScripts (package.json):\n")
		// Sort keys for deterministic output.
		keys := make([]string, 0, len(r.Scripts))
		for k := range r.Scripts {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		maxKeyLen := 0
		for _, k := range keys {
			if len(k) > maxKeyLen {
				maxKeyLen = len(k)
			}
		}
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("  %-*s  %s\n", maxKeyLen, k, r.Scripts[k]))
		}
	}

	return b.String()
}
