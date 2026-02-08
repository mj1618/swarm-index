package main

import (
	"fmt"
	"os"

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
		fmt.Printf("Indexed %d files across %d packages\n", idx.FileCount(), idx.PackageCount())

	case "lookup":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: swarm-index lookup <query>")
			os.Exit(1)
		}
		query := os.Args[2]
		results, err := index.Lookup(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "version":
		fmt.Println("swarm-index v0.1.0")

	default:
		printUsage()
		os.Exit(1)
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
