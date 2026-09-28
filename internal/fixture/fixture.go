// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package fixture builds the evidence passmcp-graph's tests and published
// fixture use: valid attestations, passmcp JSON reports, and the fixture graph
// the query documentation's examples run against.
package fixture

import (
	"encoding/json"
	"fmt"
	"time"

	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/graph"
)

// Verdict is shorthand for one check's outcome.
type Verdict struct{ ID, Status, Severity string }

// Statement returns a valid MCP evaluation statement about the server at
// transport and endpoint, run at ranAt.
func Statement(transport, endpoint string, ranAt time.Time, score float64, grade string, credentials string, verdicts ...Verdict) []byte {
	t := attestation.Target{Transport: transport, Endpoint: endpoint}
	vs := make([]attestation.Verdict, 0, len(verdicts)+1)
	counts := attestation.Counts{}
	all := append([]Verdict{{ID: "net.tcp", Status: "pass"}}, verdicts...)
	for _, v := range all {
		vs = append(vs, attestation.Verdict{ID: v.ID, Phase: "net", Status: v.Status, Severity: v.Severity})
		switch v.Status {
		case "pass":
			counts.Pass++
		case "warn":
			counts.Warn++
		case "fail":
			counts.Fail++
		case "skip":
			counts.Skip++
		case "info":
			counts.Info++
		}
	}
	st := attestation.Statement{
		Type:          attestation.StatementType,
		Subject:       []attestation.Subject{attestation.SubjectFor(t)},
		PredicateType: attestation.PredicateType,
		Predicate: attestation.Evaluation{
			SubjectKind:   attestation.SubjectKindDescriptor,
			Target:        t,
			JudgedAgainst: attestation.Basis{Rubric: "1", CheckInventory: "0.0.1"},
			Instrument:    attestation.Instrument{Name: "passmcp", Version: "0.0.1", SchemaVersion: 1},
			RanAt:         ranAt.UTC(),
			Took:          "1s",
			Verdicts:      vs,
			Counts:        counts,
			Score:         &attestation.Score{Total: score, Grade: grade, Assessed: 6, Of: 6},
			Plan:          &attestation.Plan{Credentials: credentials, OS: "linux", Arch: "amd64"},
		},
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}

// Tool is one catalogue row in a report.
type Tool struct {
	Name                  string
	ReadOnly, Destructive bool
	Annotated             bool
}

// Auth is a report's auth summary.
type Auth struct {
	Required, Reached bool
	Mode, Issuer      string
	ClientID, Scope   string
}

// Report returns a passmcp JSON report with the parts the graph reads.
func Report(transport, endpoint string, auth Auth, tools ...Tool) []byte {
	type tool struct {
		Name        string `json:"name"`
		ReadOnly    bool   `json:"read_only"`
		Destructive bool   `json:"destructive"`
		Annotated   bool   `json:"annotated"`
	}
	ts := make([]tool, len(tools))
	for i, t := range tools {
		ts[i] = tool(t)
	}
	a := map[string]any{"mode": auth.Mode, "required": auth.Required, "reached": auth.Reached}
	if auth.Issuer != "" {
		a["issuer"], a["client_id"] = auth.Issuer, auth.ClientID
		a["token"] = map[string]any{"scope": auth.Scope}
	}
	b, err := json.MarshalIndent(map[string]any{
		"passmcp": map[string]string{"version": "0.0.1"},
		"target":  map[string]string{"transport": transport, "endpoint": endpoint},
		"server":  map[string]string{"name": "fixture", "version": "1"},
		"auth":    a,
		"catalog": map[string]any{"tools": ts},
		"phases":  []any{},
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}

// Now is the fixture's reference time: attestation ages are measured from it.
var Now = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// The fixture's servers.
const (
	CRM      = "https://crm.example/mcp"
	Mail     = "https://mail.example/mcp"
	Shell    = "npx @example/shell-mcp"
	Files    = "https://files.example/mcp"
	Issuer   = "https://auth.example"
	OpsID    = "ops-agent"
	ReaderID = "reader-agent"
)

func b(v bool) *bool         { return &v }
func f(v float64) *float64   { return &v }
func sid(t, e string) string { return graph.ServerID(t, e) }

// Graph is the published fixture graph (testdata/fixture/graph.json): two
// agents, four servers, their tools, two identities and three attestations,
// arranged so the documented queries each return a known set of paths.
func Graph() *graph.Graph {
	g := graph.New()
	server(g, "http", CRM, "none", 41.5, "F", "crm", Now.AddDate(0, 0, -2), []graph.Finding{{ID: "auth.unauthenticated_tools", Severity: "critical"}})
	server(g, "http", Mail, "oauth", 88, "B", "mail", Now.AddDate(0, 0, -40), nil)
	server(g, "stdio", Shell, "", 0, "", "", time.Time{}, nil)
	server(g, "http", Files, "oauth", 95, "A", "files", Now.AddDate(0, 0, -1), nil)

	tool(g, CRM, "http", "delete_record", b(false), b(true))
	tool(g, CRM, "http", "list_records", b(true), b(false))
	tool(g, Mail, "http", "send_email", nil, nil)
	tool(g, Shell, "stdio", "run_shell", nil, nil)
	tool(g, Files, "http", "read_file", b(true), b(false))

	agent(g, "claude-desktop", "/home/dev/claude_desktop_config.json", map[string][2]string{
		"crm": {"http", CRM}, "mail": {"http", Mail}, "shell": {"stdio", Shell},
	})
	agent(g, "cursor", "/home/dev/.cursor/mcp.json", map[string][2]string{
		"mail": {"http", Mail}, "files": {"http", Files},
	})

	identity(g, OpsID, sid("http", Files), "files:write", "admin")
	identity(g, ReaderID, sid("http", Mail), "mail:read")
	return g
}

func server(g *graph.Graph, transport, endpoint, auth string, score float64, grade, name string, ranAt time.Time, failing []graph.Finding) {
	id := sid(transport, endpoint)
	props := &graph.ServerProps{Transport: transport, Endpoint: endpoint, Auth: auth, Name: name, Failing: failing}
	if !ranAt.IsZero() {
		digest := graph.Digest([]byte(endpoint + ranAt.String()))
		props.Score, props.Grade, props.Attestation = f(score), grade, digest
		aid := graph.AttestationID(digest)
		g.Upsert(graph.Node{ID: aid, Kind: graph.KindAttestation, Label: name + ".intoto.json",
			Attestation: &graph.AttestationProps{Digest: digest, PredicateType: attestation.PredicateType, RanAt: ranAt.Format(time.RFC3339)}})
		g.Link(graph.Edge{From: id, To: aid, Kind: graph.AttestedBy})
	}
	g.Upsert(graph.Node{ID: id, Kind: graph.KindServer, Label: endpoint, Server: props})
}

func tool(g *graph.Graph, endpoint, transport, name string, ro, de *bool) {
	s := sid(transport, endpoint)
	id := graph.ToolID(s, name)
	g.Upsert(graph.Node{ID: id, Kind: graph.KindTool, Label: name, Tool: &graph.ToolProps{Name: name, ReadOnlyHint: ro, DestructiveHint: de}})
	g.Link(graph.Edge{From: s, To: id, Kind: graph.Exposes})
}

func agent(g *graph.Graph, client, file string, servers map[string][2]string) {
	id := graph.AgentID(client, file, client)
	g.Upsert(graph.Node{ID: id, Kind: graph.KindAgent, Label: client, Agent: &graph.AgentProps{Client: client, ConfigFile: file}})
	for name, s := range servers {
		g.Link(graph.Edge{From: id, To: sid(s[0], s[1]), Kind: graph.Uses, Ref: &graph.ConfigRef{File: file, Key: "mcpServers." + name}})
	}
}

func identity(g *graph.Graph, client, server string, scopes ...string) {
	id := graph.IdentityID(Issuer, client)
	g.Upsert(graph.Node{ID: id, Kind: graph.KindIdentity, Label: client, Identity: &graph.IdentityProps{Issuer: Issuer, ClientID: client}})
	g.Link(graph.Edge{From: id, To: server, Kind: graph.Authorizes, Scopes: scopes})
}

// Label returns the label a path step prints for a node ID in the fixture,
// for error messages.
func Label(g *graph.Graph, id string) string {
	n, ok := g.Node(id)
	if !ok {
		return fmt.Sprintf("<%s>", id)
	}
	return n.Label
}
