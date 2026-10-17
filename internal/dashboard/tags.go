// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"sort"
	"strings"
)

// Every filterable item on the page carries tags and a lowercased search
// text, computed here so the page's script only compares strings:
//
//	st-critical st-unattested st-stale st-ok  server status, and staleness
//	destr unann                              destructive-marked or unannotated tools
//	cli-N                                    a client the item belongs to
//
// One filter state then scopes the topology, lists and tables alike.

// tagSet collects tags without repeats and renders them sorted.
type tagSet map[string]bool

func (t tagSet) add(tags ...string) tagSet {
	for _, x := range tags {
		for _, f := range strings.Fields(x) {
			t[f] = true
		}
	}
	return t
}

func (t tagSet) String() string {
	out := make([]string, 0, len(t))
	for k := range t {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

func text(parts ...string) string { return strings.ToLower(strings.Join(parts, " ")) }

// statusTags are a server's own tags: its status, staleness and the kinds
// of tool it exposes.
func statusTags(sv ServerRow) tagSet {
	t := tagSet{}.add("st-" + sv.Status)
	if sv.Stale {
		t.add("st-stale")
	}
	if sv.Destructive > 0 {
		t.add("destr")
	}
	for _, ti := range sv.ToolList {
		if ti.Behavior == unannotated {
			t.add("unann")
		}
	}
	return t
}

func behaviourTags(ti ToolInfo) tagSet {
	t := tagSet{}
	if ti.Destructive {
		t.add("destr")
	}
	if ti.Behavior == unannotated {
		t.add("unann")
	}
	return t
}

func refTags(refs []Ref) tagSet {
	t := tagSet{}
	for _, r := range refs {
		t.add(r.Anchor)
	}
	return t
}

// addTags tags every server, client, path, identity and finding.
func addTags(s *Summary) {
	own := map[string]tagSet{}
	for i, sv := range s.Servers {
		own[sv.Anchor] = statusTags(sv)
		s.Servers[i].Tags = tagSet{}.add(own[sv.Anchor].String(), refTags(sv.Clients).String()).String()
		s.Servers[i].Text = text(sv.Endpoint, sv.Name, sv.Transport, sv.Status, strings.Join(sv.Critical, " "), toolNames(sv.ToolList))
	}
	for i, a := range s.Agents {
		t := tagSet{}.add(a.Anchor)
		for j, p := range a.Paths {
			pt := tagSet{}.add(a.Anchor, statusOnly(own[p.Server.Anchor]).String(), behaviourTags(p.ToolInfo).String())
			s.Agents[i].Paths[j].Tags = pt.String()
			s.Agents[i].Paths[j].Text = text(a.Name, p.Server.Name, p.Name, p.Behavior)
			t.add(pt.String())
		}
		s.Agents[i].Tags = t.String()
		s.Agents[i].Text = text(a.Name, a.Client, a.ConfigFile, pathText(a.Paths))
	}
	for i, id := range s.Identities {
		t := refTags(id.Clients)
		for _, g := range id.Grants {
			t.add(statusOnly(own[g.Server.Anchor]).String())
		}
		s.Identities[i].Tags = t.String()
		s.Identities[i].Text = text(id.Label, id.Issuer, id.ClientID)
	}
	for i, g := range s.Gaps {
		s.Gaps[i].Tags = refTags(g.Clients).add(own[g.Server.Anchor].String()).String()
		s.Gaps[i].Text = text(g.Title, g.Kind)
	}
	for i, f := range s.Findings {
		t := refTags(f.Clients).add(own[f.Server.Anchor].String())
		s.Findings[i].Tags = t.String()
		s.Findings[i].Text = text(f.Title, f.Rule, f.Path, f.Kind)
	}
}

// statusOnly keeps a server's status tags, dropping the tool kinds, so a
// read-only path to a server with a destructive tool is not tagged destr.
func statusOnly(t tagSet) tagSet {
	out := tagSet{}
	for k := range t {
		if strings.HasPrefix(k, "st-") {
			out.add(k)
		}
	}
	return out
}

func toolNames(ts []ToolInfo) string {
	names := make([]string, len(ts))
	for i, t := range ts {
		names[i] = t.Name
	}
	return strings.Join(names, " ")
}

func pathText(ps []PathInfo) string {
	parts := make([]string, 0, 2*len(ps))
	for _, p := range ps {
		parts = append(parts, p.Server.Name, p.Name)
	}
	return strings.Join(parts, " ")
}
