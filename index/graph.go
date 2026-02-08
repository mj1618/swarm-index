package index

import (
	"fmt"
	"sort"
	"strings"
)

// GraphEdge represents a single import relationship.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// GraphNode represents a file in the import graph with fan-in/fan-out counts.
type GraphNode struct {
	Path   string `json:"path"`
	FanIn  int    `json:"fanIn"`
	FanOut int    `json:"fanOut"`
}

// GraphStats holds aggregate statistics about the import graph.
type GraphStats struct {
	TotalFiles    int    `json:"totalFiles"`
	TotalEdges    int    `json:"totalEdges"`
	MostImported  string `json:"mostImported"`
	MostDependent string `json:"mostDependent"`
}

// GraphResult holds the full import dependency graph.
type GraphResult struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// buildAdjacency builds the forward import adjacency map for all indexed files.
func (idx *Index) buildAdjacency(indexedPaths map[string]bool) map[string][]string {
	adjacency := make(map[string][]string)
	for p := range indexedPaths {
		imports := idx.extractImports(p, indexedPaths)
		if len(imports) > 0 {
			adjacency[p] = imports
		}
	}
	return adjacency
}

// buildGraphResult constructs a GraphResult from a set of edges.
func buildGraphResult(edges []GraphEdge) *GraphResult {
	fanIn := make(map[string]int)
	fanOut := make(map[string]int)
	nodeSet := make(map[string]bool)

	for _, e := range edges {
		fanOut[e.From]++
		fanIn[e.To]++
		nodeSet[e.From] = true
		nodeSet[e.To] = true
	}

	nodes := make([]GraphNode, 0, len(nodeSet))
	for p := range nodeSet {
		nodes = append(nodes, GraphNode{
			Path:   p,
			FanIn:  fanIn[p],
			FanOut: fanOut[p],
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].FanIn != nodes[j].FanIn {
			return nodes[i].FanIn > nodes[j].FanIn
		}
		return nodes[i].Path < nodes[j].Path
	})

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	stats := GraphStats{
		TotalFiles: len(nodes),
		TotalEdges: len(edges),
	}
	if len(nodes) > 0 {
		stats.MostImported = nodes[0].Path
		maxFanOut := 0
		for _, n := range nodes {
			if n.FanOut > maxFanOut {
				maxFanOut = n.FanOut
				stats.MostDependent = n.Path
			}
		}
	}

	return &GraphResult{
		Nodes: nodes,
		Edges: edges,
		Stats: stats,
	}
}

// Graph builds the full project-wide import dependency graph.
func (idx *Index) Graph() *GraphResult {
	indexedPaths := make(map[string]bool)
	for _, p := range idx.FilePaths() {
		indexedPaths[p] = true
	}

	adjacency := idx.buildAdjacency(indexedPaths)

	var edges []GraphEdge
	for from, imports := range adjacency {
		for _, to := range imports {
			edges = append(edges, GraphEdge{From: from, To: to})
		}
	}

	return buildGraphResult(edges)
}

// GraphFocused builds a subgraph reachable from the given file in both
// directions (imports and importers), limited by depth. A depth of 0 means
// unlimited.
func (idx *Index) GraphFocused(focusFile string, depth int) (*GraphResult, error) {
	indexedPaths := make(map[string]bool)
	for _, p := range idx.FilePaths() {
		indexedPaths[p] = true
	}
	if !indexedPaths[focusFile] {
		return nil, fmt.Errorf("file %s not found in index", focusFile)
	}

	// Build forward and reverse adjacency maps.
	forward := idx.buildAdjacency(indexedPaths)
	reverse := make(map[string][]string)
	for p, imports := range forward {
		for _, imp := range imports {
			reverse[imp] = append(reverse[imp], p)
		}
	}

	// BFS in both directions from focusFile.
	type bfsEntry struct {
		path string
		dist int
	}
	visited := make(map[string]bool)
	visited[focusFile] = true
	queue := []bfsEntry{{focusFile, 0}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if depth > 0 && cur.dist >= depth {
			continue
		}

		for _, imp := range forward[cur.path] {
			if !visited[imp] {
				visited[imp] = true
				queue = append(queue, bfsEntry{imp, cur.dist + 1})
			}
		}

		for _, importer := range reverse[cur.path] {
			if !visited[importer] {
				visited[importer] = true
				queue = append(queue, bfsEntry{importer, cur.dist + 1})
			}
		}
	}

	// Collect edges within the visited subgraph.
	var edges []GraphEdge
	for from := range visited {
		for _, to := range forward[from] {
			if visited[to] {
				edges = append(edges, GraphEdge{From: from, To: to})
			}
		}
	}

	return buildGraphResult(edges), nil
}

// FormatGraph returns a human-readable text rendering of the graph result.
func FormatGraph(r *GraphResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Import graph (%d files, %d edges):\n", r.Stats.TotalFiles, r.Stats.TotalEdges))

	if len(r.Nodes) == 0 {
		b.WriteString("\n  No import relationships found\n")
		return b.String()
	}

	// Most imported (highest fan-in), top 10.
	b.WriteString("\nMost imported (highest fan-in):\n")
	count := 0
	for _, n := range r.Nodes {
		if n.FanIn == 0 {
			break
		}
		if count >= 10 {
			break
		}
		b.WriteString(fmt.Sprintf("  %-50s <- %d files\n", n.Path, n.FanIn))
		count++
	}

	// Most dependencies (highest fan-out), top 10.
	byFanOut := make([]GraphNode, len(r.Nodes))
	copy(byFanOut, r.Nodes)
	sort.Slice(byFanOut, func(i, j int) bool {
		if byFanOut[i].FanOut != byFanOut[j].FanOut {
			return byFanOut[i].FanOut > byFanOut[j].FanOut
		}
		return byFanOut[i].Path < byFanOut[j].Path
	})

	b.WriteString("\nMost dependencies (highest fan-out):\n")
	count = 0
	for _, n := range byFanOut {
		if n.FanOut == 0 {
			break
		}
		if count >= 10 {
			break
		}
		b.WriteString(fmt.Sprintf("  %-50s -> %d files\n", n.Path, n.FanOut))
		count++
	}

	// All edges.
	b.WriteString(fmt.Sprintf("\nAll edges (%d):\n", len(r.Edges)))
	for _, e := range r.Edges {
		b.WriteString(fmt.Sprintf("  %s -> %s\n", e.From, e.To))
	}

	return b.String()
}

// FormatGraphDOT returns a Graphviz DOT representation of the graph.
func FormatGraphDOT(r *GraphResult) string {
	var b strings.Builder

	b.WriteString("digraph imports {\n")
	b.WriteString("  rankdir=LR;\n")

	for _, e := range r.Edges {
		b.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", e.From, e.To))
	}

	b.WriteString("}\n")

	return b.String()
}
