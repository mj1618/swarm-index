package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/matt/swarm-index/index"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: swarm-index scan <directory>")
			os.Exit(1)
		}
		dir := os.Args[2]
		idx, err := index.Scan(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := idx.Save(dir); err != nil {
			fmt.Fprintf(os.Stderr, "error saving index: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Index saved to %s/swarm/index/ (%d files, %d packages)\n", dir, idx.FileCount(), idx.PackageCount())
		if summary := extensionSummary(idx.ExtensionCounts()); summary != "" {
			fmt.Printf("  %s\n", summary)
		}

	case "lookup":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: swarm-index lookup <query> [--root <dir>] [--max N]")
			os.Exit(1)
		}
		query := os.Args[2]
		extraArgs := os.Args[3:]
		root, err := resolveRoot(extraArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		max := parseMax(extraArgs)
		idx, err := index.Load(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		results := idx.Match(query)
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

	case "version":
		fmt.Println("swarm-index v0.1.0")

	default:
		printUsage()
		os.Exit(1)
	}
}

// parseMax checks args for --max N. Returns 20 as the default.
func parseMax(args []string) int {
	for i, arg := range args {
		if arg == "--max" && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return 20
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

func printUsage() {
	fmt.Fprintln(os.Stderr, `swarm-index — a helpful index lookup for coding agents

Usage:
  swarm-index scan <directory>    Scan and index a codebase
  swarm-index lookup <query> [--root <dir>] [--max N]   Look up symbols, files, or concepts
  swarm-index version             Print version info

Examples:
  swarm-index scan .
  swarm-index lookup "handleAuth"`)
}
