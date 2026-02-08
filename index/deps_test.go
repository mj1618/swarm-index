package index

import (
	"strings"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	content := []byte(`module github.com/example/project

go 1.22

require (
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	golang.org/x/sync v0.6.0 // indirect
)

require github.com/single/dep v0.1.0
`)

	deps, err := parseGoMod(content)
	if err != nil {
		t.Fatalf("parseGoMod() error: %v", err)
	}

	if len(deps) != 4 {
		t.Fatalf("got %d deps, want 4", len(deps))
	}

	if deps[0].Name != "github.com/gorilla/mux" || deps[0].Version != "v1.8.1" {
		t.Errorf("deps[0] = %+v, want gorilla/mux v1.8.1", deps[0])
	}
	if deps[3].Name != "github.com/single/dep" || deps[3].Version != "v0.1.0" {
		t.Errorf("deps[3] = %+v, want single/dep v0.1.0", deps[3])
	}
	for _, d := range deps {
		if d.Dev {
			t.Errorf("Go dep %q should not be dev", d.Name)
		}
	}
}

func TestParseGoModEmpty(t *testing.T) {
	content := []byte(`module github.com/example/project

go 1.22
`)
	deps, err := parseGoMod(content)
	if err != nil {
		t.Fatalf("parseGoMod() error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("got %d deps, want 0", len(deps))
	}
}

func TestParseGoModComments(t *testing.T) {
	content := []byte(`module github.com/example/project

go 1.22

require (
	// This is a comment
	github.com/pkg/errors v0.9.1

)
`)
	deps, err := parseGoMod(content)
	if err != nil {
		t.Fatalf("parseGoMod() error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("got %d deps, want 1", len(deps))
	}
	if deps[0].Name != "github.com/pkg/errors" {
		t.Errorf("name = %q, want github.com/pkg/errors", deps[0].Name)
	}
}

func TestParsePackageJSON(t *testing.T) {
	content := []byte(`{
  "name": "my-app",
  "dependencies": {
    "react": "^18.2.0",
    "next": "^14.0.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "eslint": "^8.56.0"
  }
}`)

	deps, err := parsePackageJSON(content)
	if err != nil {
		t.Fatalf("parsePackageJSON() error: %v", err)
	}

	if len(deps) != 4 {
		t.Fatalf("got %d deps, want 4", len(deps))
	}

	// Non-dev deps come first, sorted.
	if deps[0].Name != "next" || deps[0].Version != "^14.0.0" || deps[0].Dev {
		t.Errorf("deps[0] = %+v, want next ^14.0.0 (non-dev)", deps[0])
	}
	if deps[1].Name != "react" || deps[1].Version != "^18.2.0" || deps[1].Dev {
		t.Errorf("deps[1] = %+v, want react ^18.2.0 (non-dev)", deps[1])
	}

	// Dev deps sorted.
	if deps[2].Name != "@types/react" || !deps[2].Dev {
		t.Errorf("deps[2] = %+v, want @types/react (dev)", deps[2])
	}
	if deps[3].Name != "eslint" || !deps[3].Dev {
		t.Errorf("deps[3] = %+v, want eslint (dev)", deps[3])
	}
}

func TestParsePackageJSONNoDeps(t *testing.T) {
	content := []byte(`{"name": "empty-project"}`)
	deps, err := parsePackageJSON(content)
	if err != nil {
		t.Fatalf("parsePackageJSON() error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("got %d deps, want 0", len(deps))
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	content := []byte(`# Core dependencies
flask==2.3.3
requests>=2.31.0
numpy
pandas~=2.0
boto3!=1.0.0
click<=8.0.0

# Skip these
-r other-requirements.txt
-e .
`)

	deps, err := parseRequirementsTxt(content)
	if err != nil {
		t.Fatalf("parseRequirementsTxt() error: %v", err)
	}

	if len(deps) != 6 {
		t.Fatalf("got %d deps, want 6", len(deps))
	}

	cases := []struct {
		name    string
		version string
	}{
		{"flask", "==2.3.3"},
		{"requests", ">=2.31.0"},
		{"numpy", ""},
		{"pandas", "~=2.0"},
		{"boto3", "!=1.0.0"},
		{"click", "<=8.0.0"},
	}
	for i, tc := range cases {
		if deps[i].Name != tc.name {
			t.Errorf("deps[%d].Name = %q, want %q", i, deps[i].Name, tc.name)
		}
		if deps[i].Version != tc.version {
			t.Errorf("deps[%d].Version = %q, want %q", i, deps[i].Version, tc.version)
		}
	}
}

func TestParseRequirementsTxtInlineComments(t *testing.T) {
	content := []byte(`flask==2.3.3 # web framework
requests>=2.31.0 # HTTP client
`)

	deps, err := parseRequirementsTxt(content)
	if err != nil {
		t.Fatalf("parseRequirementsTxt() error: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("got %d deps, want 2", len(deps))
	}
	if deps[0].Name != "flask" || deps[0].Version != "==2.3.3" {
		t.Errorf("deps[0] = %+v, want flask ==2.3.3", deps[0])
	}
}

func TestParseRequirementsTxtExtras(t *testing.T) {
	content := []byte(`celery[redis,msgpack]>=5.0
`)

	deps, err := parseRequirementsTxt(content)
	if err != nil {
		t.Fatalf("parseRequirementsTxt() error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("got %d deps, want 1", len(deps))
	}
	if deps[0].Name != "celery" {
		t.Errorf("Name = %q, want celery", deps[0].Name)
	}
	if deps[0].Version != ">=5.0" {
		t.Errorf("Version = %q, want >=5.0", deps[0].Version)
	}
}

func TestParseCargoToml(t *testing.T) {
	content := []byte(`[package]
name = "my-crate"
version = "0.1.0"

[dependencies]
serde = "1.0"
tokio = { version = "1.35", features = ["full"] }
log = "0.4"

[dev-dependencies]
criterion = "0.5"
`)

	deps, err := parseCargoToml(content)
	if err != nil {
		t.Fatalf("parseCargoToml() error: %v", err)
	}

	if len(deps) != 4 {
		t.Fatalf("got %d deps, want 4", len(deps))
	}

	if deps[0].Name != "serde" || deps[0].Version != "1.0" || deps[0].Dev {
		t.Errorf("deps[0] = %+v, want serde 1.0 non-dev", deps[0])
	}
	if deps[1].Name != "tokio" || deps[1].Version != "1.35" || deps[1].Dev {
		t.Errorf("deps[1] = %+v, want tokio 1.35 non-dev", deps[1])
	}
	if deps[3].Name != "criterion" || !deps[3].Dev {
		t.Errorf("deps[3] = %+v, want criterion dev", deps[3])
	}
}

func TestParseCargoTomlEmpty(t *testing.T) {
	content := []byte(`[package]
name = "my-crate"
version = "0.1.0"
`)
	deps, err := parseCargoToml(content)
	if err != nil {
		t.Fatalf("parseCargoToml() error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("got %d deps, want 0", len(deps))
	}
}

func TestParsePyprojectToml(t *testing.T) {
	content := []byte(`[project]
name = "my-package"
dependencies = [
    "flask>=2.0",
    "requests",
    "click>=8.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "black",
]
`)

	deps, err := parsePyprojectToml(content)
	if err != nil {
		t.Fatalf("parsePyprojectToml() error: %v", err)
	}

	if len(deps) != 5 {
		t.Fatalf("got %d deps, want 5", len(deps))
	}

	if deps[0].Name != "flask" || deps[0].Version != ">=2.0" || deps[0].Dev {
		t.Errorf("deps[0] = %+v, want flask >=2.0 non-dev", deps[0])
	}
	if deps[1].Name != "requests" || deps[1].Version != "" || deps[1].Dev {
		t.Errorf("deps[1] = %+v, want requests (no version) non-dev", deps[1])
	}

	// Dev deps.
	if deps[3].Name != "pytest" || deps[3].Version != ">=7.0" || !deps[3].Dev {
		t.Errorf("deps[3] = %+v, want pytest >=7.0 dev", deps[3])
	}
	if deps[4].Name != "black" || deps[4].Dev != true {
		t.Errorf("deps[4] = %+v, want black dev", deps[4])
	}
}

func TestParsePyprojectTomlInline(t *testing.T) {
	content := []byte(`[project]
name = "my-package"
dependencies = ["flask>=2.0", "requests"]
`)

	deps, err := parsePyprojectToml(content)
	if err != nil {
		t.Fatalf("parsePyprojectToml() error: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("got %d deps, want 2", len(deps))
	}
	if deps[0].Name != "flask" {
		t.Errorf("deps[0].Name = %q, want flask", deps[0].Name)
	}
}

func TestDepsIntegration(t *testing.T) {
	tmp := t.TempDir()

	mkFile(t, tmp, "go.mod", `module github.com/example/project

go 1.22

require (
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
)
`)
	mkFile(t, tmp, "package.json", `{
  "dependencies": {
    "react": "^18.2.0"
  }
}`)
	mkFile(t, tmp, "main.go", `package main

func main() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Deps()
	if err != nil {
		t.Fatalf("Deps() error: %v", err)
	}

	if result.TotalManifests != 2 {
		t.Errorf("TotalManifests = %d, want 2", result.TotalManifests)
	}
	if result.TotalDependencies != 3 {
		t.Errorf("TotalDependencies = %d, want 3", result.TotalDependencies)
	}

	// Verify manifests are sorted by path.
	if len(result.Manifests) == 2 {
		if result.Manifests[0].Path > result.Manifests[1].Path {
			t.Errorf("manifests not sorted: %s > %s", result.Manifests[0].Path, result.Manifests[1].Path)
		}
	}
}

func TestDepsNoManifests(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

func main() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Deps()
	if err != nil {
		t.Fatalf("Deps() error: %v", err)
	}

	if result.TotalManifests != 0 {
		t.Errorf("TotalManifests = %d, want 0", result.TotalManifests)
	}
	if result.TotalDependencies != 0 {
		t.Errorf("TotalDependencies = %d, want 0", result.TotalDependencies)
	}
}

func TestDepsSubdirectoryManifest(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "frontend/package.json", `{
  "dependencies": {
    "vue": "^3.4.0"
  }
}`)
	mkFile(t, tmp, "main.go", `package main`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result, err := idx.Deps()
	if err != nil {
		t.Fatalf("Deps() error: %v", err)
	}

	if result.TotalManifests != 1 {
		t.Errorf("TotalManifests = %d, want 1", result.TotalManifests)
	}
	if result.Manifests[0].Path != "frontend/package.json" {
		t.Errorf("Path = %q, want frontend/package.json", result.Manifests[0].Path)
	}
}

func TestFormatDepsEmpty(t *testing.T) {
	result := &DepsResult{
		Manifests:         nil,
		TotalManifests:    0,
		TotalDependencies: 0,
	}
	out := FormatDeps(result)
	if !strings.Contains(out, "No dependency manifests found") {
		t.Errorf("output missing empty message: %s", out)
	}
}

func TestFormatDepsWithResults(t *testing.T) {
	result := &DepsResult{
		Manifests: []ManifestDeps{
			{
				Path:      "go.mod",
				Type:      "go.mod",
				Ecosystem: "Go",
				Dependencies: []Dependency{
					{Name: "github.com/gorilla/mux", Version: "v1.8.1"},
					{Name: "github.com/lib/pq", Version: "v1.10.9"},
				},
			},
			{
				Path:      "package.json",
				Type:      "package.json",
				Ecosystem: "Node.js",
				Dependencies: []Dependency{
					{Name: "react", Version: "^18.2.0"},
					{Name: "eslint", Version: "^8.56.0", Dev: true},
				},
			},
		},
		TotalManifests:    2,
		TotalDependencies: 4,
	}

	out := FormatDeps(result)
	if !strings.Contains(out, "go.mod (Go):") {
		t.Errorf("output missing 'go.mod (Go):': %s", out)
	}
	if !strings.Contains(out, "package.json (Node.js):") {
		t.Errorf("output missing 'package.json (Node.js):': %s", out)
	}
	if !strings.Contains(out, "gorilla/mux") {
		t.Errorf("output missing gorilla/mux: %s", out)
	}
	if !strings.Contains(out, "devDependencies:") {
		t.Errorf("output missing devDependencies: %s", out)
	}
	if !strings.Contains(out, "2 manifests, 4 dependencies") {
		t.Errorf("output missing summary line: %s", out)
	}
}
