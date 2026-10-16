// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// The topology is drawn in a 1000-wide viewBox, as tall as its rows need:
// clients on the left, servers in the middle, tools on the right, joined
// by the exact uses and exposes edges. Each server sits level with the
// middle of its own block of tools, so server-to-tool edges never cross,
// and servers are ordered by the mean position of their clients, which
// keeps client-to-server crossings few.
const (
	topoClientX = 200.0
	topoServerX = 500.0
	topoServerW = 220.0
	topoToolX   = 790.0
	topoRow     = 30.0 // a tool row
	topoGap     = 14.0 // between servers' blocks of tools
	topoTop     = 30.0
	// Past these, the topology shows the first rows and the path table
	// stays complete; past topoMaxTools, each server's tools are one node.
	topoMaxClients = 40
	topoMaxServers = 40
	topoMaxTools   = 60
)

// Topology is the exact client -> server -> tool diagram.
type Topology struct {
	Height                 float64
	Clients, Servers, Tool []TNode
	Edges                  []TEdge
	// Folded is true when each server's tools are drawn as one node.
	Folded bool
	// Hidden says what is not drawn, or is empty.
	Hidden string
}

// TNode is one node, tagged like the page's other filterable items.
type TNode struct {
	Label, Full, Anchor, Class, Tags, Text string
	Y                                      float64
}

// TEdge is one edge, tagged with what both its ends are tagged with.
type TEdge struct {
	Path, Class, Tags, Text string
}

func buildTopology(s *Summary) Topology {
	t := Topology{}
	clients := s.Agents[:min(len(s.Agents), topoMaxClients)]
	servers := orderServers(s.Servers, clients)
	servers = servers[:min(len(servers), topoMaxServers)]
	t.Folded = countTools(servers) > topoMaxTools
	y := topoTop
	for _, sv := range servers {
		y = t.addServer(sv, y)
	}
	t.Height = t.placeClients(clients, y)
	t.Hidden = hidden(len(s.Agents)-len(clients), len(s.Servers)-len(servers))
	return t
}

// orderServers sorts servers by the mean index of the clients that use
// them; servers no drawn client uses go last, in their table order.
func orderServers(rows []ServerRow, clients []AgentRow) []ServerRow {
	idx := map[string]int{}
	for i, c := range clients {
		idx[c.Anchor] = i
	}
	bary := func(sv ServerRow) float64 {
		sum, n := 0, 0
		for _, c := range sv.Clients {
			if i, ok := idx[c.Anchor]; ok {
				sum, n = sum+i, n+1
			}
		}
		if n == 0 {
			return float64(len(clients) + 1)
		}
		return float64(sum) / float64(n)
	}
	out := append([]ServerRow{}, rows...)
	sort.SliceStable(out, func(i, j int) bool { return bary(out[i]) < bary(out[j]) })
	return out
}

func countTools(servers []ServerRow) int {
	n := 0
	for _, sv := range servers {
		n += len(sv.ToolList)
	}
	return n
}

// addServer lays out one server and its tools from y, returning where the
// next server's block starts.
func (t *Topology) addServer(sv ServerRow, y float64) float64 {
	tools := t.toolNodes(sv)
	rows := float64(max(1, len(tools)))
	sy := y + (rows-1)*topoRow/2
	t.Servers = append(t.Servers, TNode{Label: clip(sv.Endpoint, 28), Full: sv.Endpoint, Anchor: sv.Anchor,
		Class: "st-" + sv.Status, Tags: sv.Tags, Text: sv.Text, Y: sy})
	for i, tn := range tools {
		tn.Y = y + float64(i)*topoRow
		t.Tool = append(t.Tool, tn)
		t.Edges = append(t.Edges, TEdge{Class: "ts " + tn.Class, Tags: tn.Tags, Text: tn.Text,
			Path: fmt.Sprintf("M%.0f,%.1f C690,%.1f 690,%.1f %.0f,%.1f", topoServerX+topoServerW/2, sy, sy, tn.Y, topoToolX-6, tn.Y)})
	}
	return y + rows*topoRow + topoGap
}

// toolNodes are a server's tools, or one node standing for all of them
// when the topology is folded.
func (t *Topology) toolNodes(sv ServerRow) []TNode {
	base := statusOnly(statusTags(sv)).add(refTags(sv.Clients).String())
	if !t.Folded {
		out := make([]TNode, 0, len(sv.ToolList))
		for _, ti := range sv.ToolList {
			out = append(out, TNode{Label: clip(ti.Name, 26), Full: sv.Endpoint + " · " + ti.Name, Anchor: sv.Anchor,
				Class: toolClass(ti), Tags: tagSet{}.add(base.String(), behaviourTags(ti).String()).String(),
				Text: text(sv.Endpoint, ti.Name, ti.Behavior)})
		}
		return out
	}
	if len(sv.ToolList) == 0 {
		return nil
	}
	class := "b-" + readOnly
	for _, ti := range sv.ToolList {
		if ti.Behavior != readOnly {
			class = "b-" + mutating
		}
	}
	return []TNode{{Label: fmt.Sprintf("%d tools, %d destructive-marked", len(sv.ToolList), sv.Destructive),
		Full: sv.Endpoint, Anchor: sv.Anchor, Class: class, Tags: sv.Tags, Text: sv.Text}}
}

func toolClass(ti ToolInfo) string {
	if ti.Destructive {
		return "b-" + ti.Behavior + " destr"
	}
	return "b-" + ti.Behavior
}

// placeClients spreads the clients down the left, at least 40 apart, and
// joins each to its servers; it returns the diagram's height.
func (t *Topology) placeClients(clients []AgentRow, toolsBottom float64) float64 {
	height := max(toolsBottom, topoTop+40*float64(len(clients))) + 10
	server := map[string]TNode{}
	for _, sv := range t.Servers {
		server[sv.Anchor] = sv
	}
	for i, c := range clients {
		y := spread(i, len(clients), topoTop+10, height-30)
		t.Clients = append(t.Clients, TNode{Label: clip(c.Name, 22), Full: c.Name, Anchor: c.Anchor,
			Class: "client", Tags: c.Tags, Text: c.Text, Y: y})
		for _, p := range uniqueServers(c.Paths) {
			sv, ok := server[p]
			if !ok {
				continue
			}
			sy := sv.Y
			t.Edges = append(t.Edges, TEdge{Class: "cs", Tags: tagSet{}.add(c.Anchor, statusOnly(tagSet{}.add(sv.Tags)).String()).String(),
				Text: text(c.Name) + " " + sv.Text,
				Path: fmt.Sprintf("M%.0f,%.1f C340,%.1f 340,%.1f %.0f,%.1f", topoClientX+6, y, y, sy, topoServerX-topoServerW/2, sy)})
		}
	}
	return height
}

func uniqueServers(paths []PathInfo) []string {
	seen := set{}
	var out []string
	for _, p := range paths {
		if !seen[p.Server.Anchor] {
			seen = seen.add(p.Server.Anchor)
			out = append(out, p.Server.Anchor)
		}
	}
	return out
}

func hidden(clients, servers int) string {
	var parts []string
	if clients > 0 {
		parts = append(parts, fmt.Sprintf("%d more client(s)", clients))
	}
	if servers > 0 {
		parts = append(parts, fmt.Sprintf("%d more server(s)", servers))
	}
	return strings.Join(parts, " and ")
}

// clip shortens s to n runes with an ellipsis; the full text goes in the
// node's tooltip and the tables.
func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
