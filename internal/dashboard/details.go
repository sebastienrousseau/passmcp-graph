// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"fmt"
	"sort"
	"strings"

	"satellion.com/passmcp-reporting/graph"
)

// Ref links to another entity's details on the page.
type Ref struct {
	Name   string `json:"name"`
	Anchor string `json:"anchor"`
}

// Tool behaviours, as the tool declares them. "unannotated" means the
// server declared nothing, so the behaviour is unknown, not observed.
const (
	readOnly    = "read-only"
	mutating    = "mutating"
	unannotated = "unannotated"
)

// ToolInfo is one tool a server exposes and what it declares.
type ToolInfo struct {
	Name        string `json:"name"`
	Behavior    string `json:"behavior"`
	Destructive bool   `json:"destructive"`
	// DestructiveBasis says why a tool counts as potentially destructive:
	// "declared" (destructiveHint true) or "presumed" (no destructiveHint
	// on a tool not declared read-only, the MCP specification's default).
	DestructiveBasis string `json:"destructiveBasis,omitempty"`
}

// PathInfo is one configured client -> server -> tool path.
type PathInfo struct {
	Server Ref `json:"server"`
	ToolInfo
	Tags, Text string `json:"-"`
}

// Grant is the scopes an identity holds on a server.
type Grant struct {
	Identity       Ref      `json:"identity"`
	Server         Ref      `json:"server"`
	Scopes         []string `json:"scopes"`
	OverPrivileged bool     `json:"overPrivileged"`
}

// ServerDetail is what a server's details panel shows beyond its row.
type ServerDetail struct {
	Version     string          `json:"version,omitempty"`
	Attestation string          `json:"attestation,omitempty"`
	RanAt       string          `json:"ranAt,omitempty"`
	Failing     []graph.Finding `json:"failing"`
	ToolList    []ToolInfo      `json:"tools"`
	Clients     []Ref           `json:"clients"`
	Grants      []Grant         `json:"grants"`
	// Query reproduces the server's tool list, or is empty when the
	// endpoint cannot be quoted in the query language.
	Query string `json:"query,omitempty"`
}

// IdentityRow is one identity and the servers it is authorised on.
type IdentityRow struct {
	ID             string  `json:"-"`
	Anchor         string  `json:"anchor"`
	Label          string  `json:"label"`
	Issuer         string  `json:"issuer"`
	ClientID       string  `json:"clientId"`
	Grants         []Grant `json:"grants"`
	Clients        []Ref   `json:"clients"`
	OverPrivileged bool    `json:"overPrivileged"`
	Tags, Text     string  `json:"-"`
}

// anchors maps graph node IDs to page fragments, and names them.
type anchors map[string]Ref

func (a anchors) refs(ids []string) []Ref {
	out := make([]Ref, 0, len(ids))
	for _, id := range ids {
		if r, ok := a[id]; ok {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// addDetails gives every server, agent and identity an anchor and fills in
// the details each panel shows.
func addDetails(g *graph.Graph, s *Summary, r *reach) {
	a := anchors{}
	for i := range s.Servers {
		s.Servers[i].Anchor = fmt.Sprintf("srv-%d", i)
		a[s.Servers[i].ID] = Ref{s.Servers[i].Endpoint, s.Servers[i].Anchor}
	}
	for i := range s.Agents {
		s.Agents[i].Anchor = fmt.Sprintf("cli-%d", i)
		a[s.Agents[i].ID] = Ref{s.Agents[i].Name, s.Agents[i].Anchor}
	}
	s.Identities = identityRows(g, a, s)
	for i := range s.Servers {
		s.Servers[i].ServerDetail = serverDetail(g, a, r, s, s.Servers[i])
	}
	for i := range s.Agents {
		s.Agents[i].Paths, s.Agents[i].Identities = agentDetail(g, a, r, s.Agents[i].ID)
		for _, p := range s.Agents[i].Paths {
			switch p.DestructiveBasis {
			case "declared":
				s.Agents[i].DeclaredDestructive++
			case "presumed":
				s.Agents[i].PresumedDestructive++
			}
		}
	}
}

// toolInfo describes a tool with the query language's own predicates.
func toolInfo(g *graph.Graph, r *reach, toolID string) ToolInfo {
	t, _ := g.Node(toolID)
	ti := ToolInfo{Name: t.Label, Behavior: mutating, Destructive: r.tools.destructive[toolID]}
	if ti.Destructive {
		ti.DestructiveBasis = "presumed"
		if t.Tool != nil && t.Tool.DestructiveHint != nil {
			ti.DestructiveBasis = "declared"
		}
	}
	switch {
	case r.tools.readOnly[toolID]:
		ti.Behavior = readOnly
	case r.tools.unannotated[toolID]:
		ti.Behavior = unannotated
	}
	return ti
}

func serverDetail(g *graph.Graph, a anchors, r *reach, s *Summary, row ServerRow) ServerDetail {
	n, _ := g.Node(row.ID)
	d := ServerDetail{Failing: []graph.Finding{}, ToolList: []ToolInfo{}, Grants: []Grant{}}
	if p := n.Server; p != nil {
		d.Version, d.Attestation = p.Version, p.Attestation
		d.Failing = append(d.Failing, p.Failing...)
		if at, ok := g.Node(graph.AttestationID(p.Attestation)); ok && at.Attestation != nil {
			d.RanAt = at.Attestation.RanAt
		}
	}
	for _, e := range g.EdgesFrom(row.ID, graph.Exposes) {
		d.ToolList = append(d.ToolList, toolInfo(g, r, e.To))
	}
	sort.Slice(d.ToolList, func(i, j int) bool { return d.ToolList[i].Name < d.ToolList[j].Name })
	d.Clients = a.refs(keys(r.serverAgents[row.ID]))
	for _, id := range s.Identities {
		for _, gr := range id.Grants {
			if gr.Server.Anchor == row.Anchor {
				d.Grants = append(d.Grants, gr)
			}
		}
	}
	if !strings.ContainsAny(row.Endpoint, `'"`) {
		d.Query = fmt.Sprintf(`passmcp-graph query "servers -> tools where server.endpoint = '%s'"`, row.Endpoint)
	}
	return d
}

func agentDetail(g *graph.Graph, a anchors, r *reach, agentID string) ([]PathInfo, []Ref) {
	paths := []PathInfo{}
	idents := set{}
	for sid := range r.agentServers[agentID] {
		for _, e := range g.EdgesFrom(sid, graph.Exposes) {
			paths = append(paths, PathInfo{Server: a[sid], ToolInfo: toolInfo(g, r, e.To)})
		}
		for _, e := range g.EdgesTo(sid, graph.Authorizes) {
			idents = idents.add(e.From)
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Server.Name+"\x00"+paths[i].Name < paths[j].Server.Name+"\x00"+paths[j].Name
	})
	return paths, nonNil(a.refs(keys(idents)))
}

// identityRows lists every identity with its grants, flagging the grants
// analyze found over-privileged, and the clients that reach its servers.
func identityRows(g *graph.Graph, a anchors, s *Summary) []IdentityRow {
	flagged := set{}
	for _, f := range s.OverPrivileged {
		flagged = flagged.add(f.Identity + "\x00" + f.Server)
	}
	rows := []IdentityRow{}
	for _, n := range g.Nodes {
		if n.Kind != graph.KindIdentity {
			continue
		}
		row := IdentityRow{ID: n.ID, Label: n.Label, Anchor: fmt.Sprintf("idn-%d", len(rows)), Grants: []Grant{}}
		if n.Identity != nil {
			row.Issuer, row.ClientID = n.Identity.Issuer, n.Identity.ClientID
		}
		a[n.ID] = Ref{n.Label, row.Anchor}
		clients := set{}
		for _, e := range g.EdgesFrom(n.ID, graph.Authorizes) {
			gr := Grant{Identity: a[n.ID], Server: a[e.To], Scopes: append([]string{}, e.Scopes...),
				OverPrivileged: flagged[n.Label+"\x00"+a[e.To].Name]}
			row.OverPrivileged = row.OverPrivileged || gr.OverPrivileged
			row.Grants = append(row.Grants, gr)
			for _, ae := range g.EdgesTo(e.To, graph.Uses) {
				clients = clients.add(ae.From)
			}
		}
		row.Clients = a.refs(keys(clients))
		rows = append(rows, row)
	}
	return rows
}

func keys(s set) []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
