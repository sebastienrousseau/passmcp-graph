// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package ingest

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/graph"
)

// attestationFile ingests an MCP evaluation statement.
func attestationFile(g *graph.Graph, path string, raw []byte) error {
	// Parse validates too: a statement that fails validation is not
	// evidence, and ingesting it would put an unverifiable verdict on a node.
	st, err := attestation.Parse(raw)
	if err != nil {
		if strings.Contains(err.Error(), "predicateType is") {
			// Another predicate, such as an A2A evaluation: not an MCP server.
			return fmt.Errorf("%w: %w", errSkip, err)
		}
		return err
	}
	p := st.Predicate
	digest := graph.Digest(raw)
	sid := graph.ServerID(p.Target.Transport, p.Target.Endpoint)
	aid := graph.AttestationID(digest)
	ranAt := p.RanAt.UTC().Format(time.RFC3339)
	g.Upsert(graph.Node{ID: aid, Kind: graph.KindAttestation, Label: filepath.Base(path),
		Attestation: &graph.AttestationProps{Digest: digest, PredicateType: st.PredicateType, RanAt: ranAt, File: path}})
	g.Link(graph.Edge{From: sid, To: aid, Kind: graph.AttestedBy})

	server := &graph.ServerProps{Transport: p.Target.Transport, Endpoint: p.Target.Endpoint}
	if p.Target.Server != nil {
		server.Name, server.Version = p.Target.Server.Name, p.Target.Server.Version
	}
	// A run that reached the server with no credentials and was not stopped
	// shows the server admits anyone. Otherwise the statement alone does not
	// say how it admits clients, and the auth is left for a report to set.
	if p.Plan != nil && p.Plan.Credentials == "none" && p.Blocked == "" {
		server.Auth = "none"
	}
	if newest(g, sid, ranAt) {
		server.Attestation = digest
		if p.Score != nil {
			total := p.Score.Total
			server.Score, server.Grade = &total, p.Score.Grade
		}
		server.Failing = failing(p.Verdicts)
	}
	g.Upsert(graph.Node{ID: sid, Kind: graph.KindServer, Label: p.Target.Endpoint, Server: server})
	return nil
}

// newest reports whether an attestation that ran at ranAt is at least as new
// as the one the server already carries, so an older statement ingested
// later never replaces a newer verdict.
func newest(g *graph.Graph, serverID, ranAt string) bool {
	n, ok := g.Node(serverID)
	if !ok || n.Server == nil || n.Server.Attestation == "" {
		return true
	}
	cur, ok := g.Node(graph.AttestationID(n.Server.Attestation))
	if !ok || cur.Attestation == nil {
		return true
	}
	return ranAt >= cur.Attestation.RanAt
}

// failing lists the checks that failed, first verdict per id.
func failing(vs []attestation.Verdict) []graph.Finding {
	var out []graph.Finding
	seen := map[string]bool{}
	for _, v := range vs {
		if v.Status != "fail" || seen[v.ID] {
			continue
		}
		seen[v.ID] = true
		out = append(out, graph.Finding{ID: v.ID, Severity: v.Severity})
	}
	return out
}

// report is the part of passmcp's JSON report the graph reads.
type report struct {
	Target struct {
		Transport string `json:"transport"`
		Endpoint  string `json:"endpoint"`
	} `json:"target"`
	Server *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"server"`
	Auth *struct {
		Mode     string `json:"mode"`
		Reached  bool   `json:"reached"`
		Required bool   `json:"required"`
		Issuer   string `json:"issuer"`
		ClientID string `json:"client_id"`
		Token    *struct {
			Scope     string `json:"scope"`
			Requested string `json:"requested_scope"`
		} `json:"token"`
	} `json:"auth"`
	Catalog *struct {
		Tools []struct {
			Name        string `json:"name"`
			ReadOnly    bool   `json:"read_only"`
			Destructive bool   `json:"destructive"`
			Annotated   bool   `json:"annotated"`
		} `json:"tools"`
	} `json:"catalog"`
}

// reportFile ingests passmcp's JSON report: the server, its tools and the
// identity passmcp authenticated as.
func reportFile(g *graph.Graph, raw []byte) error {
	var r report
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	if r.Target.Transport == "" || r.Target.Endpoint == "" {
		return fmt.Errorf("the report names no target")
	}
	sid := graph.ServerID(r.Target.Transport, r.Target.Endpoint)
	server := &graph.ServerProps{Transport: r.Target.Transport, Endpoint: r.Target.Endpoint, Auth: authOf(r)}
	if r.Server != nil {
		server.Name, server.Version = r.Server.Name, r.Server.Version
	}
	g.Upsert(graph.Node{ID: sid, Kind: graph.KindServer, Label: r.Target.Endpoint, Server: server})
	if r.Catalog != nil {
		replaceTools(g, sid, r)
	}
	if r.Auth != nil && r.Auth.Issuer != "" {
		identity(g, sid, r)
	}
	return nil
}

// authOf says how the server admits clients, from what passmcp recorded.
func authOf(r report) string {
	a := r.Auth
	switch {
	case a == nil:
		return "unknown"
	case !a.Required && a.Reached:
		return "none"
	case a.Issuer != "":
		return "oauth"
	case a.Mode == "bearer":
		return "bearer"
	}
	return "unknown"
}

// replaceTools sets the server's tools to the report's catalogue: tools the
// report lists are upserted, and tools it no longer lists stop being exposed.
func replaceTools(g *graph.Graph, sid string, r report) {
	listed := map[string]bool{}
	for _, t := range r.Catalog.Tools {
		tid := graph.ToolID(sid, t.Name)
		listed[tid] = true
		props := &graph.ToolProps{Name: t.Name}
		// An unannotated tool declared nothing, which the MCP specification
		// reads as not read-only and possibly destructive; the graph keeps
		// the absence rather than inventing an answer.
		if t.Annotated {
			ro, de := t.ReadOnly, t.Destructive
			props.ReadOnlyHint, props.DestructiveHint = &ro, &de
		}
		g.Upsert(graph.Node{ID: tid, Kind: graph.KindTool, Label: t.Name, Tool: props})
		g.Link(graph.Edge{From: sid, To: tid, Kind: graph.Exposes})
	}
	for _, e := range g.EdgesFrom(sid, graph.Exposes) {
		if !listed[e.To] {
			g.Unlink(sid, graph.Exposes, e.To)
		}
	}
}

// identity records the client identity passmcp authenticated as, and the
// scopes it was granted.
func identity(g *graph.Graph, sid string, r report) {
	a := r.Auth
	iid := graph.IdentityID(a.Issuer, a.ClientID)
	label := a.ClientID
	if label == "" {
		label = a.Issuer
	}
	g.Upsert(graph.Node{ID: iid, Kind: graph.KindIdentity, Label: label,
		Identity: &graph.IdentityProps{Issuer: a.Issuer, ClientID: a.ClientID}})
	var scopes []string
	if a.Token != nil {
		scope := a.Token.Scope
		if scope == "" {
			scope = a.Token.Requested
		}
		scopes = strings.Fields(strings.ReplaceAll(scope, ",", " "))
	}
	g.Link(graph.Edge{From: iid, To: sid, Kind: graph.Authorizes, Scopes: scopes})
}
