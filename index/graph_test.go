package index

import (
	"strings"
	"testing"
)

func TestGraphGoProject(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main

import (
	"myproject/utils"
	"myproject/handlers"
)

func main() {
	utils.Hello()
	handlers.Handle()
}
`)
	mkFile(t, tmp, "utils/helpers.go", `package utils

func Hello() string { return "hi" }
`)
	mkFile(t, tmp, "handlers/handler.go", `package handlers

import "myproject/utils"

func Handle() { utils.Hello() }
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result := idx.Graph()

	if result.Stats.TotalFiles < 3 {
		t.Errorf("TotalFiles = %d, want >= 3", result.Stats.TotalFiles)
	}
	if result.Stats.TotalEdges < 3 {
		t.Errorf("TotalEdges = %d, want >= 3", result.Stats.TotalEdges)
	}

	// utils/helpers.go should have the highest fan-in (imported by main.go and handlers/handler.go).
	if result.Stats.MostImported != "utils/helpers.go" {
		t.Errorf("MostImported = %q, want %q", result.Stats.MostImported, "utils/helpers.go")
	}

	// Check fan-in/fan-out for specific nodes.
	nodeMap := make(map[string]GraphNode)
	for _, n := range result.Nodes {
		nodeMap[n.Path] = n
	}

	if nodeMap["utils/helpers.go"].FanIn != 2 {
		t.Errorf("utils/helpers.go FanIn = %d, want 2", nodeMap["utils/helpers.go"].FanIn)
	}
	if nodeMap["main.go"].FanOut != 2 {
		t.Errorf("main.go FanOut = %d, want 2", nodeMap["main.go"].FanOut)
	}
}

func TestGraphJSProject(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "app.js", `
import { helper } from './utils';
import { render } from './render';
`)
	mkFile(t, tmp, "utils.js", `export function helper() {}`)
	mkFile(t, tmp, "render.js", `
import { helper } from './utils';
export function render() {}
`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result := idx.Graph()

	if result.Stats.TotalFiles != 3 {
		t.Errorf("TotalFiles = %d, want 3", result.Stats.TotalFiles)
	}
	if result.Stats.TotalEdges != 3 {
		t.Errorf("TotalEdges = %d, want 3", result.Stats.TotalEdges)
	}

	// utils.js should be most imported (fan-in 2).
	nodeMap := make(map[string]GraphNode)
	for _, n := range result.Nodes {
		nodeMap[n.Path] = n
	}
	if nodeMap["utils.js"].FanIn != 2 {
		t.Errorf("utils.js FanIn = %d, want 2", nodeMap["utils.js"].FanIn)
	}
}

func TestGraphEmptyProject(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "readme.md", `# Hello`)
	mkFile(t, tmp, "data.csv", `a,b,c`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	result := idx.Graph()

	if result.Stats.TotalFiles != 0 {
		t.Errorf("TotalFiles = %d, want 0", result.Stats.TotalFiles)
	}
	if result.Stats.TotalEdges != 0 {
		t.Errorf("TotalEdges = %d, want 0", result.Stats.TotalEdges)
	}
	if len(result.Nodes) != 0 {
		t.Errorf("Nodes = %v, want empty", result.Nodes)
	}
	if len(result.Edges) != 0 {
		t.Errorf("Edges = %v, want empty", result.Edges)
	}
}

func TestGraphFocused(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.js", `import { b } from './b';`)
	mkFile(t, tmp, "b.js", `
import { c } from './c';
export const b = 1;
`)
	mkFile(t, tmp, "c.js", `export const c = 1;`)
	mkFile(t, tmp, "d.js", `import { c } from './c';`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Focus on b.js — should include a.js (importer), b.js, c.js (import), and d.js (also imports c.js).
	result, err := idx.GraphFocused("b.js", 0)
	if err != nil {
		t.Fatalf("GraphFocused() error: %v", err)
	}

	nodeSet := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeSet[n.Path] = true
	}

	if !nodeSet["b.js"] {
		t.Error("expected b.js in focused graph")
	}
	if !nodeSet["a.js"] {
		t.Error("expected a.js in focused graph (imports b.js)")
	}
	if !nodeSet["c.js"] {
		t.Error("expected c.js in focused graph (imported by b.js)")
	}
}

func TestGraphFocusedWithDepth(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "a.js", `import { b } from './b';`)
	mkFile(t, tmp, "b.js", `
import { c } from './c';
export const b = 1;
`)
	mkFile(t, tmp, "c.js", `export const c = 1;`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Focus on a.js with depth 1 — should include a.js and b.js only.
	result, err := idx.GraphFocused("a.js", 1)
	if err != nil {
		t.Fatalf("GraphFocused() error: %v", err)
	}

	nodeSet := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeSet[n.Path] = true
	}

	if !nodeSet["a.js"] {
		t.Error("expected a.js in focused graph")
	}
	if !nodeSet["b.js"] {
		t.Error("expected b.js in focused graph (depth 1)")
	}
	if nodeSet["c.js"] {
		t.Error("c.js should NOT be in focused graph (depth 1)")
	}
}

func TestGraphFocusedFileNotFound(t *testing.T) {
	tmp := t.TempDir()
	mkFile(t, tmp, "main.go", `package main`)

	idx, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	_, err = idx.GraphFocused("nonexistent.go", 0)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestFormatGraphEmpty(t *testing.T) {
	r := &GraphResult{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
		Stats: GraphStats{},
	}
	out := FormatGraph(r)
	if !strings.Contains(out, "Import graph (0 files, 0 edges)") {
		t.Errorf("output missing header: %s", out)
	}
	if !strings.Contains(out, "No import relationships found") {
		t.Errorf("output missing empty message: %s", out)
	}
}

func TestFormatGraphWithData(t *testing.T) {
	r := &GraphResult{
		Nodes: []GraphNode{
			{Path: "utils.js", FanIn: 2, FanOut: 0},
			{Path: "main.js", FanIn: 0, FanOut: 2},
			{Path: "app.js", FanIn: 0, FanOut: 1},
		},
		Edges: []GraphEdge{
			{From: "app.js", To: "utils.js"},
			{From: "main.js", To: "utils.js"},
			{From: "main.js", To: "app.js"},
		},
		Stats: GraphStats{
			TotalFiles:    3,
			TotalEdges:    3,
			MostImported:  "utils.js",
			MostDependent: "main.js",
		},
	}
	out := FormatGraph(r)
	if !strings.Contains(out, "Import graph (3 files, 3 edges)") {
		t.Errorf("output missing header: %s", out)
	}
	if !strings.Contains(out, "Most imported (highest fan-in)") {
		t.Errorf("output missing fan-in section: %s", out)
	}
	if !strings.Contains(out, "utils.js") {
		t.Errorf("output missing utils.js: %s", out)
	}
	if !strings.Contains(out, "Most dependencies (highest fan-out)") {
		t.Errorf("output missing fan-out section: %s", out)
	}
	if !strings.Contains(out, "main.js") {
		t.Errorf("output missing main.js: %s", out)
	}
}

func TestFormatGraphDOT(t *testing.T) {
	r := &GraphResult{
		Edges: []GraphEdge{
			{From: "main.go", To: "index/index.go"},
			{From: "main.go", To: "parsers/parsers.go"},
		},
	}
	out := FormatGraphDOT(r)
	if !strings.Contains(out, "digraph imports {") {
		t.Errorf("output missing digraph header: %s", out)
	}
	if !strings.Contains(out, "rankdir=LR") {
		t.Errorf("output missing rankdir: %s", out)
	}
	if !strings.Contains(out, `"main.go" -> "index/index.go"`) {
		t.Errorf("output missing edge: %s", out)
	}
	if !strings.Contains(out, `"main.go" -> "parsers/parsers.go"`) {
		t.Errorf("output missing edge: %s", out)
	}
	if !strings.HasSuffix(out, "}\n") {
		t.Errorf("output should end with }\\n: %s", out)
	}
}

func TestGraphJSONOutput(t *testing.T) {
	// Test that the GraphResult struct serializes correctly by verifying fields.
	r := &GraphResult{
		Nodes: []GraphNode{
			{Path: "a.js", FanIn: 1, FanOut: 0},
		},
		Edges: []GraphEdge{
			{From: "b.js", To: "a.js"},
		},
		Stats: GraphStats{
			TotalFiles:    2,
			TotalEdges:    1,
			MostImported:  "a.js",
			MostDependent: "b.js",
		},
	}

	if len(r.Nodes) != 1 || r.Nodes[0].Path != "a.js" {
		t.Errorf("unexpected Nodes: %v", r.Nodes)
	}
	if len(r.Edges) != 1 || r.Edges[0].From != "b.js" {
		t.Errorf("unexpected Edges: %v", r.Edges)
	}
	if r.Stats.MostImported != "a.js" {
		t.Errorf("MostImported = %q, want %q", r.Stats.MostImported, "a.js")
	}
}
