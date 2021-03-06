// Copyright ©2014 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package multi

import (
	"fmt"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/internal/uid"
)

var (
	ug *UndirectedGraph

	_ graph.Graph                = ug
	_ graph.Undirected           = ug
	_ graph.Multigraph           = ug
	_ graph.UndirectedMultigraph = ug
)

// UndirectedGraph implements a generalized undirected graph.
type UndirectedGraph struct {
	nodes map[int64]graph.Node
	lines map[int64]map[int64]map[int64]graph.Line

	nodeIDs uid.Set
	lineIDs uid.Set
}

// NewUndirectedGraph returns an UndirectedGraph.
func NewUndirectedGraph() *UndirectedGraph {
	return &UndirectedGraph{
		nodes: make(map[int64]graph.Node),
		lines: make(map[int64]map[int64]map[int64]graph.Line),

		nodeIDs: uid.NewSet(),
		lineIDs: uid.NewSet(),
	}
}

// NewNode returns a new unique Node to be added to g. The Node's ID does
// not become valid in g until the Node is added to g.
func (g *UndirectedGraph) NewNode() graph.Node {
	if len(g.nodes) == 0 {
		return Node(0)
	}
	if int64(len(g.nodes)) == uid.Max {
		panic("simple: cannot allocate node: no slot")
	}
	return Node(g.nodeIDs.NewID())
}

// AddNode adds n to the graph. It panics if the added node ID matches an existing node ID.
func (g *UndirectedGraph) AddNode(n graph.Node) {
	if _, exists := g.nodes[n.ID()]; exists {
		panic(fmt.Sprintf("simple: node ID collision: %d", n.ID()))
	}
	g.nodes[n.ID()] = n
	g.lines[n.ID()] = make(map[int64]map[int64]graph.Line)
	g.nodeIDs.Use(n.ID())
}

// RemoveNode removes n from the graph, as well as any edges attached to it. If the node
// is not in the graph it is a no-op.
func (g *UndirectedGraph) RemoveNode(n graph.Node) {
	if _, ok := g.nodes[n.ID()]; !ok {
		return
	}
	delete(g.nodes, n.ID())

	for from := range g.lines[n.ID()] {
		delete(g.lines[from], n.ID())
	}
	delete(g.lines, n.ID())

	g.nodeIDs.Release(n.ID())
}

// NewLine returns a new Line from the source to the destination node.
// The returned Line will have a graph-unique ID.
// The Line's ID does not become valid in g until the Line is added to g.
func (g *UndirectedGraph) NewLine(from, to graph.Node) graph.Line {
	return &Line{F: from, T: to, UID: g.lineIDs.NewID()}
}

// SetLine adds l, a line from one node to another. If the nodes do not exist, they are added.
func (g *UndirectedGraph) SetLine(l graph.Line) {
	var (
		from = l.From()
		fid  = from.ID()
		to   = l.To()
		tid  = to.ID()
		lid  = l.ID()
	)

	if !g.Has(from) {
		g.AddNode(from)
	}
	if g.lines[fid][tid] == nil {
		g.lines[fid][tid] = make(map[int64]graph.Line)
	}
	if !g.Has(to) {
		g.AddNode(to)
	}
	if g.lines[tid][fid] == nil {
		g.lines[tid][fid] = make(map[int64]graph.Line)
	}

	g.lines[fid][tid][lid] = l
	g.lines[tid][fid][lid] = l
	g.lineIDs.Use(lid)
}

// RemoveLine removes l from the graph, leaving the terminal nodes. If the line does not exist
// it is a no-op.
func (g *UndirectedGraph) RemoveLine(l graph.Line) {
	from, to := l.From(), l.To()
	if _, ok := g.nodes[from.ID()]; !ok {
		return
	}
	if _, ok := g.nodes[to.ID()]; !ok {
		return
	}

	delete(g.lines[from.ID()], to.ID())
	delete(g.lines[to.ID()], from.ID())
	if len(g.lines[to.ID()][from.ID()]) == 0 {
		delete(g.lines[to.ID()], from.ID())
	}
	g.lineIDs.Release(l.ID())
}

// Node returns the node in the graph with the given ID.
func (g *UndirectedGraph) Node(id int64) graph.Node {
	return g.nodes[id]
}

// Has returns whether the node exists within the graph.
func (g *UndirectedGraph) Has(n graph.Node) bool {
	_, ok := g.nodes[n.ID()]
	return ok
}

// Nodes returns all the nodes in the graph.
func (g *UndirectedGraph) Nodes() []graph.Node {
	if len(g.nodes) == 0 {
		return nil
	}
	nodes := make([]graph.Node, len(g.nodes))
	i := 0
	for _, n := range g.nodes {
		nodes[i] = n
		i++
	}
	return nodes
}

// Edges returns all the edges in the graph. Each edge in the returned slice
// is a multi.Edge.
func (g *UndirectedGraph) Edges() []graph.Edge {
	if len(g.lines) == 0 {
		return nil
	}
	var edges []graph.Edge
	seen := make(map[int64]struct{})
	for _, u := range g.lines {
		for _, e := range u {
			var lines Edge
			for _, l := range e {
				lid := l.ID()
				if _, ok := seen[lid]; ok {
					continue
				}
				seen[lid] = struct{}{}
				lines = append(lines, l)
			}
			if len(lines) != 0 {
				edges = append(edges, lines)
			}
		}
	}
	return edges
}

// From returns all nodes in g that can be reached directly from n.
func (g *UndirectedGraph) From(n graph.Node) []graph.Node {
	if !g.Has(n) {
		return nil
	}

	nodes := make([]graph.Node, len(g.lines[n.ID()]))
	i := 0
	for from := range g.lines[n.ID()] {
		nodes[i] = g.nodes[from]
		i++
	}
	return nodes
}

// HasEdgeBetween returns whether an edge exists between nodes x and y.
func (g *UndirectedGraph) HasEdgeBetween(x, y graph.Node) bool {
	_, ok := g.lines[x.ID()][y.ID()]
	return ok
}

// EdgeBetween returns the edge between nodes x and y.
func (g *UndirectedGraph) EdgeBetween(x, y graph.Node) graph.Edge {
	return g.Edge(x, y)
}

// Edge returns the edge from u to v if such an edge exists and nil otherwise.
// The node v must be directly reachable from u as defined by the From method.
// The returned graph.Edge is a multi.Edge if an edge exists.
func (g *UndirectedGraph) Edge(u, v graph.Node) graph.Edge {
	lines := g.LinesBetween(u, v)
	if len(lines) == 0 {
		return nil
	}
	return Edge(lines)
}

// Lines returns the lines from u to v if such an edge exists and nil otherwise.
// The node v must be directly reachable from u as defined by the From method.
func (g *UndirectedGraph) Lines(u, v graph.Node) []graph.Line {
	return g.LinesBetween(u, v)
}

// LinesBetween returns the lines between nodes x and y.
func (g *UndirectedGraph) LinesBetween(x, y graph.Node) []graph.Line {
	var lines []graph.Line
	for _, l := range g.lines[x.ID()][y.ID()] {
		lines = append(lines, l)
	}
	return lines
}

// Degree returns the degree of n in g.
func (g *UndirectedGraph) Degree(n graph.Node) int {
	if _, ok := g.nodes[n.ID()]; !ok {
		return 0
	}
	var deg int
	for _, e := range g.lines[n.ID()] {
		deg += len(e)
	}
	return deg
}
