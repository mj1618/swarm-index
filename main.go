package main

import (
	"fmt"
	"os"
	"path/filepath"

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

	case "lookup":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: swarm-index lookup <query>")
			os.Exit(1)
		}
		query := os.Args[2]
		root, err := findIndexRoot(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		idx, err := index.Load(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		results := idx.Match(query)
		if len(results) == 0 {
			fmt.Println("no matches found")
		} else {
			for _, r := range results {
				fmt.Println(r)
			}
		}

	case "version":
		fmt.Println("swarm-index v0.1.0")

	default:
		printUsage()
		os.Exit(1)
	}
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

func printUsage() {
	fmt.Fprintln(os.Stderr, `swarm-index — a helpful index lookup for coding agents

Usage:
  swarm-index scan <directory>    Scan and index a codebase
  swarm-index lookup <query>      Look up symbols, files, or concepts
  swarm-index version             Print version info

Examples:
  swarm-index scan .
  swarm-index lookup "handleAuth"`)
}
