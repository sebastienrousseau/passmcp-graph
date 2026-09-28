// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package analyze derives what the evidence implies: which agents inherit a
// server's critical finding, and which identities hold more privilege than
// the server's tools need. Nothing it derives is stored; it is recomputed
// from the graph every time, so fixing the evidence fixes the answer.
package analyze

import (
	"regexp"
	"sort"
	"strings"

	"satellion.com/passmcp-reporting/graph"
)

// Risk is a critical finding an agent inherits from a server it uses.
type Risk struct {
	Agent    string   `json:"agent"`
	Server   string   `json:"server"`
	Finding  string   `json:"finding"`
	Severity string   `json:"severity"`
	Path     []string `json:"path"`
}

// Risks returns every critical finding on a server, inherited by every agent
// with a uses path to it, sorted.
func Risks(g *graph.Graph) []Risk {
	var out []Risk
	for _, n := range g.Nodes {
		if n.Kind != graph.KindServer || n.Server == nil {
			continue
		}
		for _, f := range n.Server.Failing {
			if !strings.EqualFold(f.Severity, "critical") {
				continue
			}
			for _, e := range g.EdgesTo(n.ID, graph.Uses) {
				agent, _ := g.Node(e.From)
				out = append(out, Risk{Agent: agent.Label, Server: n.Label, Finding: f.ID, Severity: f.Severity,
					Path: []string{agent.Label, n.Label}})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		return a.Agent+"\x00"+a.Server+"\x00"+a.Finding < b.Agent+"\x00"+b.Server+"\x00"+b.Finding
	})
	return out
}

// Flag is an identity holding more privilege than a server's tools need.
type Flag struct {
	Identity string   `json:"identity"`
	Server   string   `json:"server"`
	Flag     string   `json:"flag"`
	Scopes   []string `json:"scopes"`
	Reason   string   `json:"reason"`
}

// privileged matches a scope that grants writing or administration. Scopes
// are free-form, so the match is on the conventional words, as a whole
// segment of the scope: "repo:write", "admin", "files.delete" and "*" match;
// "readwrite-log" does not claim to, and "read:all" does not.
var privileged = regexp.MustCompile(`(?i)(^|[:._/\-])(write|admin|delete|manage|owner)($|[:._/\-])|^\*$`)

// OverPrivileged returns every identity that holds a write or admin scope on
// a server whose tools are all declared read-only: the identity can do more
// than anything the server offers requires.
func OverPrivileged(g *graph.Graph) []Flag {
	var out []Flag
	for _, n := range g.Nodes {
		if n.Kind != graph.KindIdentity {
			continue
		}
		for _, e := range g.EdgesFrom(n.ID, graph.Authorizes) {
			wide := privilegedScopes(e.Scopes)
			if len(wide) == 0 || !allReadOnly(g, e.To) {
				continue
			}
			server, _ := g.Node(e.To)
			out = append(out, Flag{Identity: n.Label, Server: server.Label, Flag: "over-privileged", Scopes: wide,
				Reason: "holds " + strings.Join(wide, ", ") + " but every tool the server exposes is read-only"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identity+out[i].Server < out[j].Identity+out[j].Server })
	return out
}

func privilegedScopes(scopes []string) []string {
	var out []string
	for _, s := range scopes {
		if privileged.MatchString(s) {
			out = append(out, s)
		}
	}
	return out
}

// allReadOnly reports whether the server exposes at least one tool and every
// one declares readOnlyHint true.
func allReadOnly(g *graph.Graph, serverID string) bool {
	edges := g.EdgesFrom(serverID, graph.Exposes)
	if len(edges) == 0 {
		return false
	}
	for _, e := range edges {
		t, ok := g.Node(e.To)
		if !ok || t.Tool == nil || t.Tool.ReadOnlyHint == nil || !*t.Tool.ReadOnlyHint {
			return false
		}
	}
	return true
}
