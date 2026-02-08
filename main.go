package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/matt/swarm-index/index"
	"github.com/matt/swarm-index/parsers"
)

// extractJSONFlag strips --json from args and returns whether it was present.
func extractJSONFlag(args []string) ([]string, bool) {
	var filtered []string
	found := false
	for _, a := range args {
		if a == "--json" {
			found = true
		} else {
			filtered = append(filtered, a)
		}
	}
	return filtered, found
}

// fatal writes an error message to stderr and exits. When useJSON is true the
// message is emitted as a JSON object; otherwise it is printed as plain text.
func fatal(useJSON bool, msg string) {
	if useJSON {
		obj := map[string]string{"error": msg}
		data, _ := json.Marshal(obj)
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

func main() {
	args, jsonOutput := extractJSONFlag(os.Args)

	if len(args) < 2 {
		if !jsonOutput {
			printUsage()
		}
		fatal(jsonOutput, "usage: swarm-index <command> [args]")
	}

	switch args[1] {
	case "scan":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index scan <directory>")
		}
		dir := args[2]
		idx, err := index.Scan(dir)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if err := idx.Save(dir); err != nil {
			fatal(jsonOutput, fmt.Sprintf("error saving index: %v", err))
		}
		if jsonOutput {
			result := map[string]interface{}{
				"filesIndexed": idx.FileCount(),
				"packages":     idx.PackageCount(),
				"indexPath":    dir + "/swarm/index/",
				"extensions":   idx.ExtensionCounts(),
			}
			data, _ := json.Marshal(result)
			fmt.Println(string(data))
		} else {
			fmt.Printf("Index saved to %s/swarm/index/ (%d files, %d packages)\n", dir, idx.FileCount(), idx.PackageCount())
			if summary := extensionSummary(idx.ExtensionCounts()); summary != "" {
				fmt.Printf("  %s\n", summary)
			}
		}

	case "lookup":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index lookup <query> [--root <dir>] [--max N]")
		}
		query := args[2]
		if err := validateQuery(query); err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		extraArgs := args[3:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		max := parseIntFlag(extraArgs, "--max", 20)
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		results := idx.Match(query)
		if jsonOutput {
			limited := results
			if len(limited) > max {
				limited = limited[:max]
			}
			if limited == nil {
				limited = []index.Entry{}
			}
			data, _ := json.Marshal(limited)
			fmt.Println(string(data))
		} else {
			if len(results) == 0 {
				fmt.Println("no matches found")
			} else {
				for _, r := range results[:min(max, len(results))] {
					fmt.Println(r)
				}
				if len(results) > max {
					fmt.Printf("... and %d more matches (use --max to see more)\n", len(results)-max)
				}
			}
		}

	case "search":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index search <pattern> [--root <dir>] [--max N]")
		}
		pattern := args[2]
		extraArgs := args[3:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		max := parseIntFlag(extraArgs, "--max", 50)
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		matches, err := idx.Search(pattern, max)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: invalid pattern: %v", err))
		}
		if jsonOutput {
			if matches == nil {
				matches = []index.SearchMatch{}
			}
			data, _ := json.Marshal(matches)
			fmt.Println(string(data))
		} else {
			if len(matches) == 0 {
				fmt.Println("no matches found")
			} else {
				for _, m := range matches {
					fmt.Printf("%s:%d: %s\n", m.Path, m.Line, m.Content)
				}
				fmt.Printf("\n%d matches\n", len(matches))
			}
		}

	case "summary":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		summary := idx.Summary()
		if jsonOutput {
			data, _ := json.MarshalIndent(summary, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatSummary(summary))
		}

	case "tree":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index tree <directory> [--depth N]")
		}
		dir := args[2]
		depth := parseIntFlag(args[3:], "--depth", 0)
		tree, err := index.BuildTree(dir, depth)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(tree, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.RenderTree(tree))
		}

	case "show":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index show <path> [--lines M:N]")
		}
		filePath := args[2]
		startLine, endLine, err := parseLineRange(args[3:])
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		result, err := index.ShowFile(filePath, startLine, endLine)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			for _, line := range result.Lines {
				fmt.Printf("%6d\t%s\n", line.Number, line.Content)
			}
		}

	case "refs":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index refs <symbol> [--root <dir>] [--max N]")
		}
		symbol := args[2]
		extraArgs := args[3:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		max := parseIntFlag(extraArgs, "--max", 50)
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		refsResult, err := idx.Refs(symbol, max)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(refsResult, "", "  ")
			fmt.Println(string(data))
		} else {
			if refsResult.Definition != nil {
				fmt.Println("Definition:")
				fmt.Printf("  %s:%d  %s\n", refsResult.Definition.Path, refsResult.Definition.Line, refsResult.Definition.Content)
				fmt.Println()
			}
			if len(refsResult.References) == 0 {
				if refsResult.Definition == nil {
					fmt.Println("no matches found")
				} else {
					fmt.Println("No references found")
				}
			} else {
				fmt.Printf("References (%d matches):\n", refsResult.TotalRefs)
				for _, r := range refsResult.References {
					fmt.Printf("  %s:%d  %s\n", r.Path, r.Line, r.Content)
				}
			}
		}

	case "outline":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index outline <file>")
		}
		filePath := args[2]
		content, err := os.ReadFile(filePath)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		ext := filepath.Ext(filePath)
		p := parsers.ForExtension(ext)
		if p == nil {
			fatal(jsonOutput, fmt.Sprintf("no parser available for %s files", ext))
		}
		symbols, err := p.Parse(filePath, content)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			if symbols == nil {
				symbols = []parsers.Symbol{}
			}
			data, _ := json.MarshalIndent(symbols, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("%s:\n", filePath)
			for _, s := range symbols {
				fmt.Printf("  %-60s :%d\n", s.Signature, s.Line)
			}
		}

	case "exports":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index exports <file|directory> [--root <dir>]")
		}
		scope := args[2]
		extraArgs := args[3:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		exportsResult, err := idx.Exports(scope)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(exportsResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatExports(exportsResult))
		}

	case "todos":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		max := parseIntFlag(extraArgs, "--max", 100)
		tag := parseStringFlag(extraArgs, "--tag", "")
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		todosResult, err := idx.Todos(tag, max)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(todosResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatTodos(todosResult))
		}

	case "related":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index related <file> [--root <dir>]")
		}
		filePath := args[2]
		extraArgs := args[3:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		relatedResult, err := idx.Related(filePath)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(relatedResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatRelated(relatedResult))
		}

	case "diff-summary":
		extraArgs := args[2:]
		ref := "HEAD~1"
		// If first extra arg doesn't start with --, treat it as the git ref
		if len(extraArgs) > 0 && !strings.HasPrefix(extraArgs[0], "--") {
			ref = extraArgs[0]
			extraArgs = extraArgs[1:]
		}
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		diffResult, err := idx.DiffSummary(root, ref)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(diffResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatDiffSummary(diffResult))
		}

	case "history":
		if len(args) < 3 {
			fatal(jsonOutput, "usage: swarm-index history <file> [--root <dir>] [--max N]")
		}
		filePath := args[2]
		extraArgs := args[3:]
		root := parseStringFlag(extraArgs, "--root", ".")
		root, err := filepath.Abs(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		max := parseIntFlag(extraArgs, "--max", 10)
		historyResult, err := index.History(root, filePath, max)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(historyResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatHistory(historyResult))
		}

	case "hotspots":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		max := parseIntFlag(extraArgs, "--max", 20)
		since := parseStringFlag(extraArgs, "--since", "")
		pathPrefix := parseStringFlag(extraArgs, "--path", "")
		hotspotsResult, err := idx.Hotspots(root, max, since, pathPrefix)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(hotspotsResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatHotspots(hotspotsResult))
		}

	case "deps":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		depsResult, err := idx.Deps()
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(depsResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatDeps(depsResult))
		}

	case "entry-points":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		max := parseIntFlag(extraArgs, "--max", 100)
		kind := parseStringFlag(extraArgs, "--kind", "")
		epResult, err := idx.EntryPoints(kind, max)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(epResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatEntryPoints(epResult))
		}

	case "graph":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		focus := parseStringFlag(extraArgs, "--focus", "")
		depth := parseIntFlag(extraArgs, "--depth", 0)
		format := parseStringFlag(extraArgs, "--format", "list")
		var graphResult *index.GraphResult
		if focus != "" {
			graphResult, err = idx.GraphFocused(focus, depth)
			if err != nil {
				fatal(jsonOutput, fmt.Sprintf("error: %v", err))
			}
		} else {
			graphResult = idx.Graph()
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(graphResult, "", "  ")
			fmt.Println(string(data))
		} else if format == "dot" {
			fmt.Print(index.FormatGraphDOT(graphResult))
		} else {
			fmt.Print(index.FormatGraph(graphResult))
		}

	case "config":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		configResult, err := idx.Config()
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(configResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatConfig(configResult))
		}

	case "context":
		if len(args) < 4 {
			fatal(jsonOutput, "usage: swarm-index context <symbol> <file> [--root <dir>]")
		}
		symbol := args[2]
		filePath := args[3]
		extraArgs := args[4:]
		root := parseStringFlag(extraArgs, "--root", "")
		if root != "" {
			filePath = filepath.Join(root, filePath)
		}
		contextResult, err := index.Context(filePath, symbol)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(contextResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatContext(contextResult))
		}

	case "stale":
		extraArgs := args[2:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		idx, err := index.Load(root)
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		staleResult, err := idx.Stale()
		if err != nil {
			fatal(jsonOutput, fmt.Sprintf("error: %v", err))
		}
		if jsonOutput {
			data, _ := json.MarshalIndent(staleResult, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Print(index.FormatStale(staleResult))
		}

	case "version":
		if jsonOutput {
			data, _ := json.Marshal(map[string]string{"version": "v0.1.0"})
			fmt.Println(string(data))
		} else {
			fmt.Println("swarm-index v0.1.0")
		}

	default:
		if !jsonOutput {
			printUsage()
		}
		fatal(jsonOutput, "unknown command: "+args[1])
	}
}

// parseIntFlag scans args for --flag N and returns its value, or defaultVal if
// the flag is absent or invalid.
func parseIntFlag(args []string, flag string, defaultVal int) int {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return defaultVal
}

// parseStringFlag scans args for --flag value and returns its value, or
// defaultVal if the flag is absent.
func parseStringFlag(args []string, flag string, defaultVal string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return defaultVal
}

// resolveRoot checks args for --root <dir>. If not found, walks up from CWD.
func resolveRoot(args []string) (string, error) {
	for i, arg := range args {
		if arg == "--root" && i+1 < len(args) {
			return filepath.Abs(args[i+1])
		}
	}
	return findIndexRoot(".")
}

// findIndexRoot walks up from dir looking for swarm/index/meta.json.
func findIndexRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		metaPath := filepath.Join(dir, "swarm", "index", "meta.json")
		if _, err := os.Stat(metaPath); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no index found — run 'swarm-index scan <dir>' first")
		}
		dir = parent
	}
}

// extensionSummary returns a one-line summary of extension counts, sorted by
// count descending. Example: ".go: 28, .md: 8, .json: 4"
func extensionSummary(counts map[string]int) string {
	type extCount struct {
		ext   string
		count int
	}
	sorted := make([]extCount, 0, len(counts))
	for ext, n := range counts {
		sorted = append(sorted, extCount{ext, n})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].ext < sorted[j].ext
	})
	parts := make([]string, len(sorted))
	for i, ec := range sorted {
		parts[i] = fmt.Sprintf("%s: %d", ec.ext, ec.count)
	}
	return strings.Join(parts, ", ")
}

func validateQuery(q string) error {
	if strings.TrimSpace(q) == "" {
		return fmt.Errorf("query must not be empty")
	}
	return nil
}

// parseLineRange scans args for --lines and parses the value as M:N.
// Supports formats: "M:N" (range), "M:" (from M to end), ":N" (from start to N), "M" (single line).
// Returns (0, 0, nil) if --lines is absent.
func parseLineRange(args []string) (int, int, error) {
	for i, arg := range args {
		if arg == "--lines" {
			if i+1 >= len(args) {
				return 0, 0, fmt.Errorf("--lines requires a value (e.g. --lines 10:20)")
			}
			val := args[i+1]
			if idx := strings.Index(val, ":"); idx >= 0 {
				var start, end int
				var err error
				if idx > 0 {
					start, err = strconv.Atoi(val[:idx])
					if err != nil {
						return 0, 0, fmt.Errorf("invalid line range: %s", val)
					}
				}
				if idx < len(val)-1 {
					end, err = strconv.Atoi(val[idx+1:])
					if err != nil {
						return 0, 0, fmt.Errorf("invalid line range: %s", val)
					}
				}
				return start, end, nil
			}
			// Single line number.
			n, err := strconv.Atoi(val)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid line range: %s", val)
			}
			return n, n, nil
		}
	}
	return 0, 0, nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `swarm-index — a helpful index lookup for coding agents

Usage:
  swarm-index scan <directory>    Scan and index a codebase
  swarm-index lookup <query> [--root <dir>] [--max N]   Look up symbols, files, or concepts
  swarm-index search <pattern> [--root <dir>] [--max N]   Regex search across file contents
  swarm-index summary [--root <dir>]   Show project overview (languages, LOC, entry points)
  swarm-index tree <directory> [--depth N]   Print directory structure
  swarm-index show <path> [--lines M:N]   Read a file with line numbers
  swarm-index refs <symbol> [--root <dir>] [--max N]   Find all references to a symbol
  swarm-index outline <file>      Show top-level symbols (functions, types, etc.)
  swarm-index exports <file|directory> [--root <dir>]   List exported/public symbols
  swarm-index context <symbol> <file> [--root <dir>]   Show symbol definition with imports and doc comments
  swarm-index todos [--root <dir>] [--max N] [--tag TAG]   Find TODO/FIXME/HACK/XXX comments
  swarm-index related <file> [--root <dir>]   Show imports, importers, and test files for a file
  swarm-index graph [--root <dir>] [--format dot|list] [--focus <file>] [--depth N]   Show project-wide import dependency graph
  swarm-index deps [--root <dir>]   List dependencies from manifest files (go.mod, package.json, etc.)
  swarm-index entry-points [--root <dir>] [--max N] [--kind KIND]   Find main functions, route handlers, CLI commands, init functions
  swarm-index config [--root <dir>]   Detect project toolchain (framework, build, test, lint, format)
  swarm-index diff-summary [git-ref] [--root <dir>]   Show changed files and affected symbols since a git ref
  swarm-index history <file> [--root <dir>] [--max N]   Show recent git commits for a file
  swarm-index hotspots [--root <dir>] [--max N] [--since <time>] [--path <prefix>]   Show most frequently changed files
  swarm-index stale [--root <dir>]   Check if index is out of date
  swarm-index version             Print version info

Examples:
  swarm-index scan .
  swarm-index lookup "handleAuth"
  swarm-index search "func\s+\w+" --max 10
  swarm-index summary
  swarm-index tree . --depth 3
  swarm-index show main.go --lines 10:20
  swarm-index outline main.go`)
}
