// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package analyze

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-graph/internal/ingest"
	"satellion.com/passmcp-reporting/graph"
)

func put(t *testing.T, dir, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func ingestAll(t *testing.T, g *graph.Graph, paths ...string) {
	t.Helper()
	if _, err := ingest.Paths(g, paths); err != nil {
		t.Fatal(err)
	}
}

// AC: SG-03
// An identity recorded by passmcp's auth phase becomes a node authorising the
// server with its scopes, and one holding a write or admin scope on a server
// whose tools are all read-only is flagged over-privileged.
func TestIdentitiesAuthoriseServersAndOverPrivilegeIsFlagged(t *testing.T) {
	dir := t.TempDir()
	g := graph.New()
	ingestAll(t, g,
		put(t, dir, "files.json", fixture.Report("http", fixture.Files,
			fixture.Auth{Required: true, Issuer: fixture.Issuer, ClientID: "ops", Scope: "files:read files:write"},
			fixture.Tool{Name: "read_file", ReadOnly: true, Annotated: true})),
		put(t, dir, "mail.json", fixture.Report("http", fixture.Mail,
			fixture.Auth{Required: true, Issuer: fixture.Issuer, ClientID: "reader", Scope: "mail:read"},
			fixture.Tool{Name: "list", ReadOnly: true, Annotated: true})),
		put(t, dir, "crm.json", fixture.Report("http", fixture.CRM,
			fixture.Auth{Required: true, Issuer: fixture.Issuer, ClientID: "admin", Scope: "admin"},
			fixture.Tool{Name: "delete", Destructive: true, Annotated: true})),
	)
	id := graph.IdentityID(fixture.Issuer, "ops")
	edges := g.EdgesFrom(id, graph.Authorizes)
	if len(edges) != 1 || edges[0].To != graph.ServerID("http", fixture.Files) ||
		!reflect.DeepEqual(edges[0].Scopes, []string{"files:read", "files:write"}) {
		t.Fatalf("authorizes %+v", edges)
	}
	flags := OverPrivileged(g)
	if len(flags) != 1 || flags[0].Identity != "ops" || flags[0].Flag != "over-privileged" ||
		!reflect.DeepEqual(flags[0].Scopes, []string{"files:write"}) {
		t.Fatalf("flags %+v", flags)
	}
	// The read scope is not flagged, and neither is the admin identity: its
	// server has a tool that is not read-only.
}

// AC: SG-04
// A failing critical finding on a server is inherited by every agent with a
// uses path to it, explained by the finding and the path, and fixing the
// finding removes the risk on the next ingest.
func TestCriticalFindingsAreInheritedAndFixingThemRemovesTheRisk(t *testing.T) {
	dir := t.TempDir()
	g := graph.New()
	config := put(t, dir, "claude_desktop_config.json", []byte(`{"mcpServers": {"crm": {"url": "`+fixture.CRM+`"}, "mail": {"url": "`+fixture.Mail+`"}}}`))
	cursor := put(t, dir, "cursor-mcp.json", []byte(`{"mcpServers": {"crm": {"url": "`+fixture.CRM+`"}}}`))
	broken := put(t, dir, "crm-1.json", fixture.Statement("http", fixture.CRM, fixture.Now, 40, "F", "none",
		fixture.Verdict{ID: "auth.unauthenticated_tools", Status: "fail", Severity: "critical"},
		fixture.Verdict{ID: "catalog.names", Status: "fail", Severity: "minor"}))
	mail := put(t, dir, "mail.json", fixture.Statement("http", fixture.Mail, fixture.Now, 90, "A", "bearer"))
	ingestAll(t, g, config, cursor, broken, mail)

	risks := Risks(g)
	if len(risks) != 2 {
		t.Fatalf("want both agents to inherit the one critical finding, got %+v", risks)
	}
	for _, r := range risks {
		if r.Finding != "auth.unauthenticated_tools" || r.Server != fixture.CRM || len(r.Path) != 2 || r.Path[1] != fixture.CRM {
			t.Errorf("risk %+v", r)
		}
	}

	fixed := put(t, dir, "crm-2.json", fixture.Statement("http", fixture.CRM, fixture.Now.AddDate(0, 0, 1), 95, "A", "none"))
	ingestAll(t, g, fixed)
	if r := Risks(g); len(r) != 0 {
		t.Fatalf("the fixed finding is still inherited: %+v", r)
	}
}

func TestPrivilegedScopeWords(t *testing.T) {
	for s, want := range map[string]bool{
		"admin": true, "repo:write": true, "files.delete": true, "*": true, "org/manage": true, "owner": true,
		"read": false, "read:all": false, "readwrite": false, "mail:read": false,
	} {
		if got := privileged.MatchString(s); got != want {
			t.Errorf("%q: %v", s, got)
		}
	}
	// A server with no tools does not make its identities over-privileged.
	g := graph.New()
	sid := graph.ServerID("http", "https://empty")
	g.Upsert(graph.Node{ID: sid, Kind: graph.KindServer, Label: "empty", Server: &graph.ServerProps{Transport: "http", Endpoint: "https://empty"}})
	iid := graph.IdentityID("i", "c")
	g.Upsert(graph.Node{ID: iid, Kind: graph.KindIdentity, Label: "c", Identity: &graph.IdentityProps{Issuer: "i"}})
	g.Link(graph.Edge{From: iid, To: sid, Kind: graph.Authorizes, Scopes: []string{"admin"}})
	if f := OverPrivileged(g); len(f) != 0 {
		t.Fatalf("%+v", f)
	}
}
