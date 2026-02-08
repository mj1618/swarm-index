package index

import (
	"strings"
	"testing"
)

func TestDetectLanguageGo(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "handler.go", "package main")
	mkFile(t, tmp, "utils.go", "package main")
	mkFile(t, tmp, "README.md", "# readme")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Language != "Go" {
		t.Errorf("Language = %q, want Go", result.Language)
	}
}

func TestDetectLanguagePython(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.py", "print('hello')")
	mkFile(t, tmp, "utils.py", "def foo(): pass")
	mkFile(t, tmp, "README.md", "# readme")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Language != "Python" {
		t.Errorf("Language = %q, want Python", result.Language)
	}
}

func TestDetectFrameworkNextJS(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.2.0"
  }
}`)
	mkFile(t, tmp, "index.ts", "export default function Home() {}")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Framework != "Next.js" {
		t.Errorf("Framework = %q, want Next.js", result.Framework)
	}
}

func TestDetectFrameworkReact(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  }
}`)
	mkFile(t, tmp, "index.tsx", "export default function App() {}")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Framework != "React" {
		t.Errorf("Framework = %q, want React", result.Framework)
	}
}

func TestDetectFrameworkDjango(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "requirements.txt", "django>=4.2\ncelery>=5.0\n")
	mkFile(t, tmp, "manage.py", "import django")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Framework != "Django" {
		t.Errorf("Framework = %q, want Django", result.Framework)
	}
}

func TestDetectFrameworkFlaskFromPyproject(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pyproject.toml", `[project]
name = "my-app"
dependencies = ["flask>=2.0", "requests"]
`)
	mkFile(t, tmp, "app.py", "from flask import Flask")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Framework != "Flask" {
		t.Errorf("Framework = %q, want Flask", result.Framework)
	}
}

func TestDetectFrameworkGin(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "go.mod", `module github.com/example/app

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
)
`)
	mkFile(t, tmp, "main.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Framework != "Gin" {
		t.Errorf("Framework = %q, want Gin", result.Framework)
	}
}

func TestDetectFrameworkAxum(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "Cargo.toml", `[package]
name = "my-app"
version = "0.1.0"

[dependencies]
axum = "0.7"
tokio = { version = "1", features = ["full"] }
`)
	mkFile(t, tmp, "src/main.rs", "fn main() {}")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Framework != "Axum" {
		t.Errorf("Framework = %q, want Axum", result.Framework)
	}
}

func TestDetectToolsEslintPrettier(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{"name": "my-app"}`)
	mkFile(t, tmp, ".eslintrc.json", `{}`)
	mkFile(t, tmp, ".prettierrc", `{}`)
	mkFile(t, tmp, "index.ts", "export default 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Lint != "eslint" {
		t.Errorf("Lint = %q, want eslint", result.Lint)
	}
	if result.Format != "prettier" {
		t.Errorf("Format = %q, want prettier", result.Format)
	}
}

func TestDetectToolsGolangciLint(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "go.mod", "module example.com/app\n\ngo 1.22\n")
	mkFile(t, tmp, ".golangci.yml", "linters:\n  enable:\n    - errcheck\n")
	mkFile(t, tmp, "main.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Lint != "golangci-lint" {
		t.Errorf("Lint = %q, want golangci-lint", result.Lint)
	}
}

func TestDetectToolsJest(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{"name": "my-app"}`)
	mkFile(t, tmp, "jest.config.js", "module.exports = {}")
	mkFile(t, tmp, "index.js", "module.exports = 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Test != "jest" {
		t.Errorf("Test = %q, want jest", result.Test)
	}
}

func TestDetectToolsVitest(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{"name": "my-app"}`)
	mkFile(t, tmp, "vitest.config.ts", "export default {}")
	mkFile(t, tmp, "index.ts", "export default 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Test != "vitest" {
		t.Errorf("Test = %q, want vitest", result.Test)
	}
}

func TestDetectToolsPytest(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pytest.ini", "[pytest]\naddopts = -v\n")
	mkFile(t, tmp, "app.py", "print('hello')")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Test != "pytest" {
		t.Errorf("Test = %q, want pytest", result.Test)
	}
}

func TestDetectToolsPytestFromPyproject(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "pyproject.toml", `[project]
name = "my-app"
dependencies = ["flask"]

[tool.pytest.ini_options]
addopts = "-v"
`)
	mkFile(t, tmp, "app.py", "from flask import Flask")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Test != "pytest" {
		t.Errorf("Test = %q, want pytest", result.Test)
	}
}

func TestDetectToolsJestFromPackageJSON(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{
  "name": "my-app",
  "jest": {
    "testMatch": ["**/*.test.js"]
  }
}`)
	mkFile(t, tmp, "index.js", "module.exports = 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Test != "jest" {
		t.Errorf("Test = %q, want jest", result.Test)
	}
}

func TestDetectGoDefaults(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "go.mod", "module example.com/app\n\ngo 1.22\n")
	mkFile(t, tmp, "main.go", "package main")
	mkFile(t, tmp, "handler.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Build != "go build" {
		t.Errorf("Build = %q, want 'go build'", result.Build)
	}
	if result.Test != "go test ./..." {
		t.Errorf("Test = %q, want 'go test ./...'", result.Test)
	}
	if result.Format != "gofmt" {
		t.Errorf("Format = %q, want gofmt", result.Format)
	}
}

func TestDetectPackageManagerNPM(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{"name": "my-app"}`)
	mkFile(t, tmp, "package-lock.json", `{}`)
	mkFile(t, tmp, "index.js", "module.exports = 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.PackageManager != "npm" {
		t.Errorf("PackageManager = %q, want npm", result.PackageManager)
	}
}

func TestDetectPackageManagerYarn(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{"name": "my-app"}`)
	mkFile(t, tmp, "yarn.lock", "")
	mkFile(t, tmp, "index.js", "module.exports = 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.PackageManager != "yarn" {
		t.Errorf("PackageManager = %q, want yarn", result.PackageManager)
	}
}

func TestDetectPackageManagerPnpm(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{"name": "my-app"}`)
	mkFile(t, tmp, "pnpm-lock.yaml", "")
	mkFile(t, tmp, "index.js", "module.exports = 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want pnpm", result.PackageManager)
	}
}

func TestExtractScripts(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{
  "name": "my-app",
  "scripts": {
    "build": "next build",
    "dev": "next dev",
    "start": "next start",
    "test": "jest",
    "lint": "eslint ."
  },
  "dependencies": {
    "next": "^14.0.0"
  }
}`)
	mkFile(t, tmp, "index.ts", "export default 1")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if len(result.Scripts) != 5 {
		t.Errorf("Scripts has %d entries, want 5", len(result.Scripts))
	}
	if result.Scripts["build"] != "next build" {
		t.Errorf("Scripts[build] = %q, want 'next build'", result.Scripts["build"])
	}
	if result.Scripts["test"] != "jest" {
		t.Errorf("Scripts[test] = %q, want 'jest'", result.Scripts["test"])
	}
}

func TestConfigNoManifests(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", "package main")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}
	if result.Language != "Go" {
		t.Errorf("Language = %q, want Go", result.Language)
	}
	if result.Framework != "" {
		t.Errorf("Framework = %q, want empty", result.Framework)
	}
	if result.PackageManager != "" {
		t.Errorf("PackageManager = %q, want empty", result.PackageManager)
	}
}

func TestFormatConfigBasic(t *testing.T) {
	result := &ConfigResult{
		Language:       "Go",
		Framework:      "",
		Build:          "go build",
		Test:           "go test ./...",
		Lint:           "golangci-lint",
		Format:         "gofmt",
		PackageManager: "go modules",
		ConfigFiles:    []string{".golangci.yml", "go.mod"},
		Scripts:        map[string]string{},
	}

	out := FormatConfig(result)
	if !strings.Contains(out, "Project toolchain") {
		t.Errorf("output missing header: %s", out)
	}
	if !strings.Contains(out, "Go") {
		t.Errorf("output missing language: %s", out)
	}
	if !strings.Contains(out, "golangci-lint") {
		t.Errorf("output missing lint tool: %s", out)
	}
	if !strings.Contains(out, "(none detected)") {
		t.Errorf("output missing framework placeholder: %s", out)
	}
	if !strings.Contains(out, "go.mod") {
		t.Errorf("output missing config file: %s", out)
	}
}

func TestFormatConfigWithScripts(t *testing.T) {
	result := &ConfigResult{
		Language:       "TypeScript",
		Framework:      "Next.js",
		Build:          "tsc",
		Test:           "jest",
		Lint:           "eslint",
		Format:         "prettier",
		PackageManager: "npm",
		ConfigFiles:    []string{"package.json", "tsconfig.json"},
		Scripts: map[string]string{
			"build": "next build",
			"dev":   "next dev",
			"test":  "jest",
		},
	}

	out := FormatConfig(result)
	if !strings.Contains(out, "Next.js") {
		t.Errorf("output missing framework: %s", out)
	}
	if !strings.Contains(out, "Scripts (package.json):") {
		t.Errorf("output missing scripts section: %s", out)
	}
	if !strings.Contains(out, "next build") {
		t.Errorf("output missing build script: %s", out)
	}
}

func TestConfigFullIntegration(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "package.json", `{
  "name": "my-nextjs-app",
  "scripts": {
    "build": "next build",
    "dev": "next dev",
    "start": "next start",
    "test": "jest",
    "lint": "eslint ."
  },
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.2.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "eslint": "^8.56.0"
  }
}`)
	mkFile(t, tmp, "tsconfig.json", `{"compilerOptions": {}}`)
	mkFile(t, tmp, ".eslintrc.json", `{}`)
	mkFile(t, tmp, ".prettierrc", `{}`)
	mkFile(t, tmp, "jest.config.js", `module.exports = {}`)
	mkFile(t, tmp, "package-lock.json", `{}`)
	mkFile(t, tmp, "src/index.tsx", "export default function App() {}")
	mkFile(t, tmp, "src/utils.ts", "export function helper() {}")

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	result, err := idx.Config()
	if err != nil {
		t.Fatalf("Config() error: %v", err)
	}

	if result.Language != "TypeScript" {
		t.Errorf("Language = %q, want TypeScript", result.Language)
	}
	if result.Framework != "Next.js" {
		t.Errorf("Framework = %q, want Next.js", result.Framework)
	}
	if result.Test != "jest" {
		t.Errorf("Test = %q, want jest", result.Test)
	}
	if result.Lint != "eslint" {
		t.Errorf("Lint = %q, want eslint", result.Lint)
	}
	if result.Format != "prettier" {
		t.Errorf("Format = %q, want prettier", result.Format)
	}
	if result.PackageManager != "npm" {
		t.Errorf("PackageManager = %q, want npm", result.PackageManager)
	}
	if result.Build != "tsc" {
		t.Errorf("Build = %q, want tsc", result.Build)
	}
	if len(result.Scripts) != 5 {
		t.Errorf("Scripts has %d entries, want 5", len(result.Scripts))
	}
}
