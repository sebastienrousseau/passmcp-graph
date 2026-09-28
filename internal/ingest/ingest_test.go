// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package ingest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-reporting/graph"
)

func write(t *testing.T, dir, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// AC: SG-01
// Attestations make server nodes with score, grade and digest; reports add
// tools with their annotations; and ingesting the same statements twice
// changes nothing.
func TestAttestationsBecomeServersAndIngestingTwiceChangesNothing(t *testing.T) {
	dir := t.TempDir()
	st := fixture.Statement("http", fixture.CRM, fixture.Now, 72.5, "C", "none",
		fixture.Verdict{ID: "auth.unauthenticated_tools", Status: "fail", Severity: "critical"})
	write(t, dir, "crm.intoto.json", st)
	write(t, dir, "crm.report.json", fixture.Report("http", fixture.CRM, fixture.Auth{Reached: true},
		fixture.Tool{Name: "delete_record", Destructive: true, Annotated: true},
		fixture.Tool{Name: "list_records", ReadOnly: true, Annotated: true},
		fixture.Tool{Name: "undeclared"}))

	g := graph.New()
	res, err := Paths(g, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Read["attestation"] != 1 || res.Read["report"] != 1 {
		t.Fatalf("read %v", res.Read)
	}
	sid := graph.ServerID("http", fixture.CRM)
	s, ok := g.Node(sid)
	if !ok || s.Server == nil {
		t.Fatal("no server node")
	}
	if *s.Server.Score != 72.5 || s.Server.Grade != "C" || s.Server.Attestation != graph.Digest(st) || s.Server.Auth != "none" {
		t.Fatalf("server %+v", *s.Server)
	}
	if len(s.Server.Failing) != 1 || s.Server.Failing[0].Severity != "critical" {
		t.Fatalf("failing %+v", s.Server.Failing)
	}
	if a, ok := g.Node(graph.AttestationID(graph.Digest(st))); !ok || a.Attestation.RanAt != "2026-09-27T12:00:00Z" {
		t.Fatalf("attestation node %+v", a)
	}
	tools := g.EdgesFrom(sid, graph.Exposes)
	if len(tools) != 3 {
		t.Fatalf("exposes %d tools", len(tools))
	}
	del, _ := g.Node(graph.ToolID(sid, "delete_record"))
	ro, _ := g.Node(graph.ToolID(sid, "list_records"))
	un, _ := g.Node(graph.ToolID(sid, "undeclared"))
	if *del.Tool.DestructiveHint != true || *del.Tool.ReadOnlyHint != false || *ro.Tool.ReadOnlyHint != true {
		t.Fatal("annotations lost")
	}
	if un.Tool.ReadOnlyHint != nil || un.Tool.DestructiveHint != nil {
		t.Fatal("an undeclared annotation must stay undeclared")
	}

	first, _ := g.Marshal()
	if _, err := Paths(g, []string{dir}); err != nil {
		t.Fatal(err)
	}
	second, _ := g.Marshal()
	if !bytes.Equal(first, second) {
		t.Fatal("ingesting the same evidence twice changed the graph")
	}
}

// A newer report replaces the catalogue; an older attestation ingested later
// never replaces a newer verdict.
func TestNewerEvidenceWinsAndOlderDoesNot(t *testing.T) {
	dir := t.TempDir()
	g := graph.New()
	newer := write(t, dir, "new.json", fixture.Statement("http", fixture.Mail, fixture.Now, 90, "A", "bearer"))
	older := write(t, dir, "old.json", fixture.Statement("http", fixture.Mail, fixture.Now.Add(-48*time.Hour), 30, "F", "bearer",
		fixture.Verdict{ID: "x", Status: "fail", Severity: "critical"}))
	for _, p := range []string{newer, older} {
		if _, err := File(g, p); err != nil {
			t.Fatal(err)
		}
	}
	s, _ := g.Node(graph.ServerID("http", fixture.Mail))
	if *s.Server.Score != 90 || len(s.Server.Failing) != 0 || s.Server.Auth != "" {
		t.Fatalf("an older attestation replaced a newer one: %+v", *s.Server)
	}
	r1 := write(t, dir, "r1.json", fixture.Report("http", fixture.Mail, fixture.Auth{Required: true, Mode: "bearer"},
		fixture.Tool{Name: "a"}, fixture.Tool{Name: "b"}))
	r2 := write(t, dir, "r2.json", fixture.Report("http", fixture.Mail, fixture.Auth{Required: true}, fixture.Tool{Name: "a"}))
	for _, p := range []string{r1, r2} {
		if _, err := File(g, p); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(g.EdgesFrom(graph.ServerID("http", fixture.Mail), graph.Exposes)); n != 1 {
		t.Fatalf("a tool the newer report no longer lists is still exposed (%d)", n)
	}
	s, _ = g.Node(graph.ServerID("http", fixture.Mail))
	if s.Server.Auth != "unknown" {
		t.Fatalf("auth %q", s.Server.Auth)
	}
}

// AC: SG-02
// Client configurations become agents that use their servers, and no secret
// from them is stored: a test seeds secrets everywhere a configuration can
// hold one and searches the whole store.
func TestConfigsBecomeAgentsAndNoSecretIsStored(t *testing.T) {
	secrets := []string{
		"sk-live-ENVSECRET123", "HEADERSECRET-abc", "FLAGSECRET-99", "ASSIGNSECRET-77",
		"ghp_barePATsecret0000", "USERINFOSECRET", "QUERYSECRET-5", "ZEDENVSECRET",
		"abcdefghijklmnopqrstuvwxyz0123456789ABCD",
	}
	dir := t.TempDir()
	claude := write(t, dir, "Claude/claude_desktop_config.json", []byte(`{"mcpServers": {
	  "crm": {"url": "https://user:USERINFOSECRET@crm.example/mcp?api_key=QUERYSECRET-5&team=blue",
	          "headers": {"Authorization": "Bearer HEADERSECRET-abc"}},
	  "shell": {"command": "npx", "args": ["@example/shell-mcp", "--api-key", "FLAGSECRET-99", "--token=ASSIGNSECRET-77", "ghp_barePATsecret0000", "abcdefghijklmnopqrstuvwxyz0123456789ABCD"],
	            "env": {"OPENAI_API_KEY": "sk-live-ENVSECRET123"}},
	  "broken": {"nothing": true}
	}}`))
	cursor := write(t, dir, ".cursor/mcp.json", []byte(`{"mcpServers": {"mail": {"serverUrl": "https://mail.example/mcp"}}}`))
	vscode := write(t, dir, ".vscode/mcp.json", []byte(`{"servers": {"files": {"type": "http", "url": "https://files.example/mcp"}}}`))
	settings := write(t, dir, "Code/User/settings.json", []byte(`{"mcp": {"servers": {"files": {"url": "https://files.example/mcp"}}}}`))
	zed := write(t, dir, "zed/settings.json", []byte(`{"context_servers": {
	  "local": {"command": {"path": "/usr/bin/tool", "args": ["--port", "3"], "env": {"K": "ZEDENVSECRET"}}},
	  "flat": {"command": "tool2", "args": []}
	}}`))
	generic := write(t, dir, "other.json", []byte(`{"mcpServers": {"x": {"url": "https://x.example"}}}`))

	store := filepath.Join(dir, "store")
	g := graph.New()
	res, err := Paths(g, []string{claude, cursor, vscode, settings, zed, generic})
	if err != nil {
		t.Fatal(err)
	}
	if res.Read["config"] != 6 {
		t.Fatalf("read %v skipped %v", res.Read, res.Skipped)
	}
	if err := graph.Save(store, g); err != nil {
		t.Fatal(err)
	}

	clients := map[string]bool{}
	for _, n := range g.Nodes {
		if n.Kind == graph.KindAgent {
			clients[n.Agent.Client] = true
		}
	}
	for _, c := range []string{"claude-desktop", "cursor", "vscode", "zed", "mcp-client"} {
		if !clients[c] {
			t.Errorf("no %s agent (have %v)", c, clients)
		}
	}
	agent := graph.AgentID("claude-desktop", claude, "claude-desktop")
	uses := g.EdgesFrom(agent, graph.Uses)
	if len(uses) != 2 {
		t.Fatalf("claude uses %d servers, want 2 (the broken entry names no server)", len(uses))
	}
	for _, e := range uses {
		if e.Ref == nil || e.Ref.File != claude || !strings.HasPrefix(e.Ref.Key, "mcpServers.") {
			t.Errorf("uses edge without a file and key reference: %+v", e)
		}
	}

	var stored []byte
	err = filepath.Walk(store, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			b, _ := os.ReadFile(p)
			stored = append(stored, b...)
		}
		return err
	})
	if err != nil || len(stored) == 0 {
		t.Fatalf("store unreadable: %v", err)
	}
	for _, s := range secrets {
		if bytes.Contains(stored, []byte(s)) {
			t.Errorf("secret %q is in the store", s)
		}
	}
	if !bytes.Contains(stored, []byte("team=blue")) || !bytes.Contains(stored, []byte("redacted")) {
		t.Error("redaction removed more than secrets, or nothing at all")
	}
}

func TestUnrecognisedAndInvalidInput(t *testing.T) {
	dir := t.TempDir()
	g := graph.New()
	skip := []string{
		write(t, dir, "list.json", []byte(`[1,2]`)),
		write(t, dir, "other.json", []byte(`{"hello": "world"}`)),
		write(t, dir, "a2a.json", []byte(`{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://satellion.com/attestation/a2a-evaluation/v1","subject":[],"predicate":{}}`)),
	}
	res, err := Paths(g, skip)
	if err != nil || len(res.Skipped) != 3 {
		t.Fatalf("res %+v err %v", res, err)
	}
	bad := write(t, dir, "bad.json", []byte(`{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://satellion.com/attestation/mcp-evaluation/v1","subject":[],"predicate":{}}`))
	if _, err := Paths(g, []string{bad}); err == nil {
		t.Error("an invalid MCP attestation must be an error, not skipped")
	}
	noTarget := write(t, dir, "r.json", []byte(`{"passmcp":{},"target":{},"phases":[]}`))
	if _, err := File(g, noTarget); err == nil {
		t.Error("a report with no target must be an error")
	}
	badReport := write(t, dir, "r2.json", []byte(`{"passmcp":{},"target":"x","phases":[]}`))
	if _, err := File(g, badReport); err == nil {
		t.Error("a malformed report must be an error")
	}
	if _, err := Paths(g, []string{filepath.Join(dir, "missing")}); err == nil {
		t.Error("a missing path must be an error")
	}
	if _, err := File(g, filepath.Join(dir, "missing.json")); err == nil {
		t.Error("an unreadable file must be an error")
	}
	if got := redactURL("::not a url"); got != Redacted {
		t.Errorf("unparsable URL: %q", got)
	}
	if isConfig(map[string]json.RawMessage{}) {
		t.Error("empty object is not a config")
	}
}
