package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Dependency represents a single declared dependency in a manifest.
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dev     bool   `json:"dev"`
}

// ManifestDeps holds parsed dependencies from a single manifest file.
type ManifestDeps struct {
	Path         string       `json:"path"`
	Type         string       `json:"type"`
	Ecosystem    string       `json:"ecosystem"`
	Dependencies []Dependency `json:"dependencies"`
}

// DepsResult holds the aggregated dependency information.
type DepsResult struct {
	Manifests         []ManifestDeps `json:"manifests"`
	TotalDependencies int            `json:"totalDependencies"`
	TotalManifests    int            `json:"totalManifests"`
}

// knownManifests maps manifest filenames to their ecosystem and parser.
var knownManifests = map[string]struct {
	ecosystem string
	parser    func([]byte) ([]Dependency, error)
}{
	"go.mod":            {ecosystem: "Go", parser: parseGoMod},
	"package.json":      {ecosystem: "Node.js", parser: parsePackageJSON},
	"requirements.txt":  {ecosystem: "Python", parser: parseRequirementsTxt},
	"Cargo.toml":        {ecosystem: "Rust", parser: parseCargoToml},
	"pyproject.toml":    {ecosystem: "Python", parser: parsePyprojectToml},
}

// Deps scans the index for known dependency manifest files and parses them.
func (idx *Index) Deps() (*DepsResult, error) {
	seen := make(map[string]struct{})
	var manifests []ManifestDeps
	totalDeps := 0

	for _, e := range idx.Entries {
		if e.Kind != "file" {
			continue
		}
		filename := filepath.Base(e.Path)
		info, ok := knownManifests[filename]
		if !ok {
			continue
		}
		// Avoid processing duplicates of the same path.
		if _, dup := seen[e.Path]; dup {
			continue
		}
		seen[e.Path] = struct{}{}

		absPath := filepath.Join(idx.Root, e.Path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		deps, err := info.parser(content)
		if err != nil {
			continue
		}

		manifests = append(manifests, ManifestDeps{
			Path:         e.Path,
			Type:         filename,
			Ecosystem:    info.ecosystem,
			Dependencies: deps,
		})
		totalDeps += len(deps)
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].Path < manifests[j].Path
	})

	return &DepsResult{
		Manifests:         manifests,
		TotalDependencies: totalDeps,
		TotalManifests:    len(manifests),
	}, nil
}

// parseGoMod parses a go.mod file and extracts dependencies.
func parseGoMod(content []byte) ([]Dependency, error) {
	var deps []Dependency
	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(trimmed, "require (") || trimmed == "require (" {
			inRequireBlock = true
			continue
		}

		if inRequireBlock {
			// Lines inside require (...) block: "module/path v1.2.3"
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			// Strip inline comments.
			if idx := strings.Index(trimmed, "//"); idx > 0 {
				trimmed = strings.TrimSpace(trimmed[:idx])
			}
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				deps = append(deps, Dependency{
					Name:    parts[0],
					Version: parts[1],
				})
			}
			continue
		}

		// Single-line require: "require module/path v1.2.3"
		if strings.HasPrefix(trimmed, "require ") && !strings.Contains(trimmed, "(") {
			rest := strings.TrimPrefix(trimmed, "require ")
			parts := strings.Fields(rest)
			if len(parts) >= 2 {
				deps = append(deps, Dependency{
					Name:    parts[0],
					Version: parts[1],
				})
			}
		}
	}

	return deps, nil
}

// parsePackageJSON parses a package.json file and extracts dependencies.
func parsePackageJSON(content []byte) ([]Dependency, error) {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, err
	}

	var deps []Dependency

	// Sort keys for deterministic output.
	names := make([]string, 0, len(pkg.Dependencies))
	for name := range pkg.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		deps = append(deps, Dependency{
			Name:    name,
			Version: pkg.Dependencies[name],
		})
	}

	devNames := make([]string, 0, len(pkg.DevDependencies))
	for name := range pkg.DevDependencies {
		devNames = append(devNames, name)
	}
	sort.Strings(devNames)
	for _, name := range devNames {
		deps = append(deps, Dependency{
			Name:    name,
			Version: pkg.DevDependencies[name],
			Dev:     true,
		})
	}

	return deps, nil
}

// splitPySpec splits a Python package specifier into name and version.
// e.g. "flask>=2.0" -> ("flask", ">=2.0"), "requests" -> ("requests", "").
// It also strips extras like package[extra1,extra2].
func splitPySpec(spec string) (name, version string) {
	for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">", "<"} {
		if idx := strings.Index(spec, sep); idx >= 0 {
			name = strings.TrimSpace(spec[:idx])
			version = strings.TrimSpace(spec[idx:])
			break
		}
	}
	if name == "" {
		name = spec
	}
	// Strip extras like package[extra1,extra2].
	if bracketIdx := strings.Index(name, "["); bracketIdx >= 0 {
		name = name[:bracketIdx]
	}
	return name, version
}

// parseRequirementsTxt parses a requirements.txt file.
func parseRequirementsTxt(content []byte) ([]Dependency, error) {
	var deps []Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}

		// Strip inline comments.
		if idx := strings.Index(trimmed, " #"); idx >= 0 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}

		name, version := splitPySpec(trimmed)

		// Skip entries that look like URLs or paths.
		if strings.Contains(name, "/") || strings.Contains(name, "\\") {
			continue
		}

		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
		})
	}

	return deps, nil
}

// parseCargoToml parses a Cargo.toml file using simple line-based parsing.
func parseCargoToml(content []byte) ([]Dependency, error) {
	var deps []Dependency
	lines := strings.Split(string(content), "\n")
	section := "" // tracks current section

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Detect section headers.
		if strings.HasPrefix(trimmed, "[") {
			section = strings.Trim(trimmed, "[] ")
			continue
		}

		isDeps := section == "dependencies"
		isDevDeps := section == "dev-dependencies"
		if !isDeps && !isDevDeps {
			continue
		}

		// Parse "name = ..." lines.
		eqIdx := strings.Index(trimmed, "=")
		if eqIdx < 0 {
			continue
		}

		name := strings.TrimSpace(trimmed[:eqIdx])
		value := strings.TrimSpace(trimmed[eqIdx+1:])

		var version string
		if strings.HasPrefix(value, "\"") {
			// Simple form: name = "version"
			version = strings.Trim(value, "\"")
		} else if strings.HasPrefix(value, "{") {
			// Table form: name = { version = "...", ... }
			if vIdx := strings.Index(value, "version"); vIdx >= 0 {
				rest := value[vIdx:]
				if eqI := strings.Index(rest, "="); eqI >= 0 {
					verPart := strings.TrimSpace(rest[eqI+1:])
					// Extract quoted version string.
					if strings.HasPrefix(verPart, "\"") {
						verPart = verPart[1:]
						if endQ := strings.Index(verPart, "\""); endQ >= 0 {
							version = verPart[:endQ]
						}
					}
				}
			}
		}

		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Dev:     isDevDeps,
		})
	}

	return deps, nil
}

// parsePyprojectToml parses a pyproject.toml file using line-based parsing.
func parsePyprojectToml(content []byte) ([]Dependency, error) {
	var deps []Dependency
	lines := strings.Split(string(content), "\n")
	section := ""
	inList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			if inList && trimmed == "" {
				// Empty line doesn't necessarily end a list in TOML.
			}
			continue
		}

		// Detect section headers.
		if strings.HasPrefix(trimmed, "[") {
			section = strings.Trim(trimmed, "[] ")
			inList = false
			continue
		}

		if section == "project" {
			// Look for dependencies = [...]
			if strings.HasPrefix(trimmed, "dependencies") && strings.Contains(trimmed, "=") {
				inList = true
				// Check if inline: dependencies = ["pkg1", "pkg2"]
				if bracketIdx := strings.Index(trimmed, "["); bracketIdx >= 0 {
					listContent := trimmed[bracketIdx:]
					parsedDeps := parsePyDependencyList(listContent, false)
					deps = append(deps, parsedDeps...)
					if strings.Contains(listContent, "]") {
						inList = false
					}
				}
				continue
			}
		}

		if strings.HasPrefix(section, "project.optional-dependencies") {
			// Look for list items.
			if strings.Contains(trimmed, "=") && strings.Contains(trimmed, "[") {
				inList = true
				if bracketIdx := strings.Index(trimmed, "["); bracketIdx >= 0 {
					listContent := trimmed[bracketIdx:]
					parsedDeps := parsePyDependencyList(listContent, true)
					deps = append(deps, parsedDeps...)
					if strings.Contains(listContent, "]") {
						inList = false
					}
				}
				continue
			}
		}

		if inList {
			if strings.Contains(trimmed, "]") {
				// Parse remaining items before the closing bracket.
				parsedDeps := parsePyDependencyList(trimmed, section != "project")
				deps = append(deps, parsedDeps...)
				inList = false
				continue
			}
			// Parse list items.
			isDev := section != "project"
			parsedDeps := parsePyDependencyList(trimmed, isDev)
			deps = append(deps, parsedDeps...)
		}
	}

	return deps, nil
}

// parsePyDependencyList extracts dependency specifiers from a TOML list fragment.
// Input can be: `["flask>=2.0", "requests"]` or `"flask>=2.0",`
func parsePyDependencyList(s string, dev bool) []Dependency {
	// Strip brackets.
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")

	var deps []Dependency
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		item = strings.Trim(item, "\"' ")
		if item == "" {
			continue
		}

		name, version := splitPySpec(item)
		if name == "" {
			continue
		}

		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Dev:     dev,
		})
	}
	return deps
}

// FormatDeps returns a human-readable text rendering of the deps result.
func FormatDeps(r *DepsResult) string {
	var b strings.Builder

	if len(r.Manifests) == 0 {
		b.WriteString("No dependency manifests found.\n")
		return b.String()
	}

	for i, m := range r.Manifests {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%s (%s):\n", m.Type, m.Ecosystem))

		// Separate dev and non-dev deps.
		var regular, dev []Dependency
		for _, d := range m.Dependencies {
			if d.Dev {
				dev = append(dev, d)
			} else {
				regular = append(regular, d)
			}
		}

		// Find max name length for alignment.
		maxNameLen := 0
		for _, d := range m.Dependencies {
			if len(d.Name) > maxNameLen {
				maxNameLen = len(d.Name)
			}
		}

		for _, d := range regular {
			if d.Version != "" {
				b.WriteString(fmt.Sprintf("  %-*s  %s\n", maxNameLen, d.Name, d.Version))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", d.Name))
			}
		}

		if len(dev) > 0 {
			b.WriteString("\n  devDependencies:\n")
			for _, d := range dev {
				if d.Version != "" {
					b.WriteString(fmt.Sprintf("  %-*s  %s\n", maxNameLen, d.Name, d.Version))
				} else {
					b.WriteString(fmt.Sprintf("  %s\n", d.Name))
				}
			}
		}
	}

	b.WriteString(fmt.Sprintf("\n%d manifests, %d dependencies\n", r.TotalManifests, r.TotalDependencies))

	return b.String()
}
