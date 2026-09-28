// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package query

import (
	"fmt"
	"sort"
	"strings"

	"satellion.com/passmcp-reporting/graph"
)

// Step is one node on a result path.
type Step struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

// Path is one result.
type Path []Step

// String renders a path as labels joined by arrows.
func (p Path) String() string {
	labels := make([]string, len(p))
	for i, s := range p {
		labels[i] = s.Label
	}
	return strings.Join(labels, " -> ")
}

// hop is how one kind reaches the next: along an edge kind, forwards or
// backwards.
type hop struct {
	edge    graph.EdgeKind
	forward bool
}

// hops lists the direct relationships between kinds.
var hops = map[[2]string]hop{
	{"agent", "server"}:       {graph.Uses, true},
	{"server", "agent"}:       {graph.Uses, false},
	{"server", "tool"}:        {graph.Exposes, true},
	{"tool", "server"}:        {graph.Exposes, false},
	{"identity", "server"}:    {graph.Authorizes, true},
	{"server", "identity"}:    {graph.Authorizes, false},
	{"server", "source"}:      {graph.DiscoveredBy, true},
	{"source", "server"}:      {graph.DiscoveredBy, false},
	{"server", "attestation"}: {graph.AttestedBy, true},
	{"attestation", "server"}: {graph.AttestedBy, false},
}

// expand inserts the server between two kinds that have no direct relation.
func expand(kinds []string) ([]string, error) {
	out := []string{kinds[0]}
	for _, k := range kinds[1:] {
		prev := out[len(out)-1]
		if _, ok := hops[[2]string{prev, k}]; !ok {
			_, a := hops[[2]string{prev, "server"}]
			_, b := hops[[2]string{"server", k}]
			if !a || !b {
				return nil, fmt.Errorf("query: no relationship from %ss to %ss", prev, k)
			}
			out = append(out, "server")
		}
		out = append(out, k)
	}
	return out, nil
}

// Eval returns every path through g that matches q, sorted and without
// duplicates. ctx supplies what fields need beyond the graph, such as the
// current time for attestation ages.
func Eval(g *graph.Graph, q *Query, ctx Context) ([]Path, error) {
	path, err := expand(q.Kinds)
	if err != nil {
		return nil, err
	}
	if q.Where != nil {
		if err := check(q.Where, path); err != nil {
			return nil, err
		}
	}
	var out []Path
	seen := map[string]bool{}
	var walk func(i int, cur []graph.Node)
	walk = func(i int, cur []graph.Node) {
		if i == len(path) {
			b := binding{g: g, nodes: cur, kinds: path, ctx: ctx}
			if q.Where == nil || b.eval(q.Where) {
				p := toPath(cur)
				if key := pathKey(p); !seen[key] {
					seen[key] = true
					out = append(out, p)
				}
			}
			return
		}
		for _, n := range candidates(g, path, i, cur) {
			walk(i+1, append(append([]graph.Node(nil), cur...), n))
		}
	}
	walk(0, nil)
	sort.Slice(out, func(i, j int) bool { return pathKey(out[i]) < pathKey(out[j]) })
	return out, nil
}

// candidates lists the nodes of kind path[i] reachable from the previous one.
func candidates(g *graph.Graph, path []string, i int, cur []graph.Node) []graph.Node {
	if i == 0 {
		var ns []graph.Node
		for _, n := range g.Nodes {
			if string(n.Kind) == path[0] {
				ns = append(ns, n)
			}
		}
		return ns
	}
	h := hops[[2]string{path[i-1], path[i]}]
	from := cur[i-1].ID
	var ns []graph.Node
	var edges []graph.Edge
	if h.forward {
		edges = g.EdgesFrom(from, h.edge)
	} else {
		edges = g.EdgesTo(from, h.edge)
	}
	for _, e := range edges {
		id := e.To
		if !h.forward {
			id = e.From
		}
		if n, ok := g.Node(id); ok {
			ns = append(ns, n)
		}
	}
	return ns
}

func toPath(ns []graph.Node) Path {
	p := make(Path, len(ns))
	for i, n := range ns {
		p[i] = Step{ID: n.ID, Kind: string(n.Kind), Label: n.Label}
	}
	return p
}

func pathKey(p Path) string {
	ids := make([]string, len(p))
	for i, s := range p {
		ids[i] = s.Label + "\x01" + s.ID
	}
	return strings.Join(ids, "\x00")
}

// check resolves every field in the condition against the path, so a typo is
// an error rather than a query that silently matches nothing.
func check(e Expr, path []string) error {
	switch x := e.(type) {
	case And:
		return firstErr(check(x.L, path), check(x.R, path))
	case Or:
		return firstErr(check(x.L, path), check(x.R, path))
	case Not:
		return check(x.X, path)
	case Cmp:
		_, err := resolve(x, path)
		return err
	}
	return nil
}

func firstErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// resolve returns the kind a comparison's field belongs to on this path.
func resolve(c Cmp, path []string) (string, error) {
	onPath := map[string]bool{}
	for _, k := range path {
		onPath[k] = true
	}
	if c.Kind != "" {
		if !onPath[c.Kind] {
			return "", fmt.Errorf("query: %s.%s: no %s on this path", c.Kind, c.Field, c.Kind)
		}
		if !hasField(c.Kind, c.Field) {
			return "", fmt.Errorf("query: %ss have no field %q", c.Kind, c.Field)
		}
		return c.Kind, nil
	}
	var owners []string
	for k := range onPath {
		if hasField(k, c.Field) {
			owners = append(owners, k)
		}
	}
	sort.Strings(owners)
	switch len(owners) {
	case 0:
		return "", fmt.Errorf("query: no node on this path has a field %q", c.Field)
	case 1:
		return owners[0], nil
	}
	return "", fmt.Errorf("query: %q is ambiguous here (%s); qualify it, as %s.%s", c.Field, strings.Join(owners, ", "), owners[0], c.Field)
}
