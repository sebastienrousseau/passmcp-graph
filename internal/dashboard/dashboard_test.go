// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-graph/internal/policy"
	"satellion.com/passmcp-graph/internal/query"
	"satellion.com/passmcp-reporting/graph"
)

// endpointOf is the endpoint of the server a link points at: the name the
// query language gives it.
func endpointOf(s *Summary, anchor string) string {
	for _, sv := range s.Servers {
		if sv.Anchor == anchor {
			return sv.Endpoint
		}
	}
	return ""
}

func build(t *testing.T, g *graph.Graph, vs []policy.Violation) *Summary {
	t.Helper()
	s, err := Build(g, Options{Now: fixture.Now, StaleDays: 30, Violations: vs, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The fixture's figures, worked out by hand from fixture.Graph and agreeing
// with the published answers in testdata/fixture/queries.json.
func TestTheFixtureSummary(t *testing.T) {
	s := build(t, fixture.Graph(), nil)
	want := Totals{Agents: 2, Servers: 4, Tools: 5, Identities: 2, Attested: 3, Unattested: 1, Stale: 1, Scored: 3, AvgScore: "74.8", DestructiveTools: 3}
	if s.Totals != want {
		t.Errorf("totals %+v, want %+v", s.Totals, want)
	}
	if s.Generated != "2026-09-27T12:00:00Z" || s.Version != "test" || s.StaleDays != 30 {
		t.Errorf("header %q %q %d", s.Generated, s.Version, s.StaleDays)
	}
	agents := map[string]AgentRow{}
	for _, a := range s.Agents {
		agents[a.Name] = a
	}
	cd, cu := agents["claude-desktop"], agents["cursor"]
	if cd.Servers != 3 || cd.ReadOnly != 1 || cd.Writable != 3 || cd.Destructive != 3 || cd.Unattested != 1 || cd.Risks != 1 {
		t.Errorf("claude-desktop %+v", cd)
	}
	if cu.Servers != 2 || cu.ReadOnly != 1 || cu.Writable != 1 || cu.Destructive != 1 || cu.Unattested != 0 || cu.Risks != 0 {
		t.Errorf("cursor %+v", cu)
	}
	if s.Agents[0].Name != "claude-desktop" || cd.ReadOnlyPct != 25 || cd.MutatingPct != 25 || cd.UnannotatedPct != 50 || cu.UnannotatedPct != 25 {
		t.Errorf("the widest agent comes first and sets the scale: %+v", s.Agents)
	}
	var got []string
	for _, r := range s.Servers {
		got = append(got, r.Status+" "+r.Endpoint)
	}
	if strings.Join(got, "|") != "critical https://crm.example/mcp|unattested npx @example/shell-mcp|stale https://mail.example/mcp|ok https://files.example/mcp" {
		t.Errorf("servers, worst first: %v", got)
	}
	crm := s.Servers[0]
	if crm.Score != "41.5" || crm.Grade != "F" || crm.AgeDays != "2" || crm.Tools != 2 || crm.Destructive != 1 || crm.Agents != 1 ||
		strings.Join(crm.Critical, ",") != "auth.unauthenticated_tools" {
		t.Errorf("crm %+v", crm)
	}
	if len(s.Risks) != 1 || len(s.OverPrivileged) != 1 || s.PolicyChecked || len(s.Violations) != 0 {
		t.Errorf("findings %+v %+v %v %+v", s.Risks, s.OverPrivileged, s.PolicyChecked, s.Violations)
	}
}

// The destructive count is the query language's predicate, so the
// dashboard and `query` cannot disagree.
func TestReachAgreesWithTheQueryLanguage(t *testing.T) {
	g := fixture.Graph()
	q, err := query.Parse("agents -> tools where destructive")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := query.Eval(g, q, query.Context{Now: fixture.Now})
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, a := range build(t, g, nil).Agents {
		total += a.Destructive
	}
	if total != len(paths) {
		t.Fatalf("dashboard counts %d destructive reaches, query returns %d", total, len(paths))
	}
}

// A tool that declares itself neither read-only nor destructive can change
// things without destroying them: it is not shown as read-only.
func TestAToolThatChangesButDoesNotDestroy(t *testing.T) {
	g := fixture.Graph()
	sid := graph.ServerID("http", fixture.Files)
	no := false
	tid := graph.ToolID(sid, "write_file")
	g.Upsert(graph.Node{ID: tid, Kind: graph.KindTool, Label: "write_file",
		Tool: &graph.ToolProps{Name: "write_file", ReadOnlyHint: &no, DestructiveHint: &no}})
	g.Link(graph.Edge{From: sid, To: tid, Kind: graph.Exposes})
	for _, a := range build(t, g, nil).Agents {
		if a.Name == "cursor" && (a.ReadOnly != 1 || a.Writable != 2 || a.Destructive != 1) {
			t.Fatalf("cursor %+v", a)
		}
	}
}

func TestStaleDaysMovesTheThreshold(t *testing.T) {
	s, err := Build(fixture.Graph(), Options{Now: fixture.Now, StaleDays: 1})
	if err != nil {
		t.Fatal(err)
	}
	// crm (2 days) and mail (40 days) are now stale; crm stays critical.
	if s.Totals.Stale != 2 || s.Servers[0].Status != "critical" || !s.Servers[0].Stale {
		t.Fatalf("stale %d, first %+v", s.Totals.Stale, s.Servers[0])
	}
}

func TestAnEmptyGraph(t *testing.T) {
	s := build(t, graph.New(), []policy.Violation{})
	if s.Totals != (Totals{}) || len(s.Agents) != 0 || len(s.Servers) != 0 || !s.PolicyChecked {
		t.Fatalf("%+v", s)
	}
	var b bytes.Buffer
	if err := Render(&b, s); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"No configured client in the graph", "No server.", "No finding in this evidence", "Every server has an attestation"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("empty page lacks %q", want)
		}
	}
}

// A server or agent without its property struct, and an attestation that
// is missing or unreadable, still produce a row rather than a panic.
func TestIncompleteNodes(t *testing.T) {
	g := graph.New()
	sid := graph.ServerID("http", "https://bare.example/mcp")
	g.Upsert(graph.Node{ID: sid, Kind: graph.KindServer, Label: "https://bare.example/mcp"})
	g.Upsert(graph.Node{ID: "agent:bare", Kind: graph.KindAgent, Label: "bare"})
	g.Link(graph.Edge{From: "agent:bare", To: sid, Kind: graph.Uses})
	gone := graph.ServerID("http", "https://gone.example/mcp")
	g.Upsert(graph.Node{ID: gone, Kind: graph.KindServer, Label: "gone", Server: &graph.ServerProps{Attestation: "missing"}})
	bad := graph.ServerID("http", "https://bad.example/mcp")
	g.Upsert(graph.Node{ID: bad, Kind: graph.KindServer, Label: "bad", Server: &graph.ServerProps{Attestation: "d"}})
	g.Upsert(graph.Node{ID: graph.AttestationID("d"), Kind: graph.KindAttestation, Label: "d", Attestation: &graph.AttestationProps{RanAt: "yesterday"}})

	s := build(t, g, nil)
	if len(s.Agents) != 1 || s.Agents[0].Unattested != 1 || s.Agents[0].Client != "" {
		t.Errorf("agents %+v", s.Agents)
	}
	for _, r := range s.Servers {
		if r.AgeDays != "" || r.Stale {
			t.Errorf("%s has an age it cannot know: %+v", r.Endpoint, r)
		}
	}
	if s.Totals.AvgScore != "" {
		t.Errorf("no score, yet an average %q", s.Totals.AvgScore)
	}
}

// Labels come from operators' configuration files; the page escapes them.
func TestRenderEscapesLabels(t *testing.T) {
	g := fixture.Graph()
	g.Upsert(graph.Node{ID: "agent:x", Kind: graph.KindAgent, Label: `<script>alert(1)</script>`,
		Agent: &graph.AgentProps{Client: `"><img src=x onerror=alert(1)>`}})
	s := build(t, g, []policy.Violation{{Rule: "<b>r</b>", Path: query.Path{{Label: "<i>p</i>"}}}})
	var b bytes.Buffer
	if err := Render(&b, s); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, bad := range []string{"<script>alert", "<img src=x", "<b>r</b>", "<i>p</i>"} {
		if strings.Contains(out, bad) {
			t.Errorf("unescaped %q in the page", bad)
		}
	}
	if !strings.Contains(out, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Error("the agent's label is not shown escaped")
	}
}

// The page loads nothing: no external script, stylesheet, image, font or
// frame, so it renders the same offline and leaks nothing when opened. Its
// only references are data: URLs and fragments within the page, and its
// Content-Security-Policy has the browser refuse anything else.
func TestThePageLoadsNothing(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, build(t, fixture.Graph(), []policy.Violation{})); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	// A CSS or attribute url() follows ':', a space, ',' or '('; the
	// script's URL.createObjectURL( does not, and loads nothing.
	loads := regexp.MustCompile(`(?i)<script[^>]*\bsrc|<img|<iframe|<object|<embed|@import|[:\s,(]url\((?:[^#)]|$)`)
	if m := loads.FindString(out); m != "" {
		t.Fatalf("the page loads something: %q", m)
	}
	// Attributes, which the template always quotes; the script's
	// a.href = URL.createObjectURL(...) builds the export download in memory.
	refs := regexp.MustCompile(`(?i)\s(?:src|href|xlink:href)="([^"]*)"`).FindAllStringSubmatch(out, -1)
	data := 0
	for _, m := range refs {
		switch {
		case strings.HasPrefix(m[1], "data:"):
			data++
		case !strings.HasPrefix(m[1], "#"):
			t.Errorf("the page references %q", m[1])
		}
	}
	if data != 1 || !strings.Contains(out, "<svg") {
		t.Errorf("want the inlined logo and one data: favicon, got %d data: reference(s)", data)
	}
	if !strings.Contains(out, `content="default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:;`) {
		t.Error("the page lacks its Content-Security-Policy")
	}
}

// The overview's figures and its status distribution, which counts each
// server once and reconciles with the total.
func TestTheHero(t *testing.T) {
	h := build(t, fixture.Graph(), nil).Hero
	if h.Paths != 6 || h.Changing != (Ratio{4, 6}) || h.Destructive != (Ratio{4, 6}) || h.Unannotated != (Ratio{3, 6}) || h.Attention != 3 {
		t.Errorf("hero figures %+v", h)
	}
	var got []string
	total := 0
	for _, g := range h.Groups {
		got = append(got, fmt.Sprintf("%s=%d", g.Status, g.Count))
		total += g.Count
		if g.Share.Den != 4 {
			t.Errorf("group %s is not a share of the 4 servers: %+v", g.Status, g.Share)
		}
	}
	if strings.Join(got, " ") != "critical=1 unattested=1 stale=1 ok=1" || total != 4 {
		t.Errorf("groups %v", got)
	}
}

// A bubble's area, not its radius, follows the tools: crm exposes two, so
// its diameter is sqrt(2) times a one-tool server's.
func TestBubbleAreaFollowsTools(t *testing.T) {
	s := build(t, fixture.Graph(), nil)
	for _, r := range s.Servers {
		want := map[int]int{2: 168, 1: 119}[r.Tools]
		if r.Bubble != want {
			t.Errorf("%s (%d tools): bubble %d, want %d", r.Endpoint, r.Tools, r.Bubble, want)
		}
	}
	rows := []ServerRow{{Tools: 100}, {Tools: 1}, {Tools: 0}}
	sizeBubbles(rows)
	if rows[0].Bubble != 168 || rows[1].Bubble != 72 || rows[2].Bubble != 72 {
		t.Errorf("small servers keep a legible minimum: %+v", rows)
	}
}

// CC-002: the published fixture's counts, each in its own unit.
func TestTheFixtureReconciles(t *testing.T) {
	s := build(t, fixture.Graph(), nil)
	byID := map[string]Metric{}
	for _, m := range s.Metrics {
		byID[m.ID] = m
	}
	for id, want := range map[string]string{
		"tools": "5", "paths": "6", "clients": "2", "servers": "4",
		"paths.destructive": "66.7%", "paths.changing": "66.7%", "paths.unannotated": "50%",
		"servers.evidence": "75%", "servers.fresh": "50%", "score.mean": "74.8",
	} {
		if got := byID[id].Display; got != want {
			t.Errorf("%s displays %q, want %q", id, got, want)
		}
	}
	if s.Totals.DestructiveTools != 3 || byID["paths.destructive"].Value != 4 || *byID["paths.destructive"].Of != 6 {
		t.Errorf("3 destructive-marked tools and 4 of 6 paths to them: %d, %+v", s.Totals.DestructiveTools, byID["paths.destructive"])
	}
	if byID["score.mean"].Value != 3 || *byID["score.mean"].Of != 4 {
		t.Errorf("the mean is over 3 scored servers of 4: %+v", byID["score.mean"])
	}
	for _, m := range s.Metrics {
		if m.Label == "" || m.Unit == "" || m.Counts == "" {
			t.Errorf("metric %s is not fully defined: %+v", m.ID, m)
		}
	}
}

// CC-003: one rounding rule, and no percentage of nothing.
func TestRatioRounding(t *testing.T) {
	for r, want := range map[Ratio]string{{4, 6}: "66.7%", {3, 5}: "60%", {0, 0}: "n/a", {1, 3}: "33.3%", {6, 6}: "100%"} {
		if got := r.Pct(); got != want {
			t.Errorf("%v: %q, want %q", r, got, want)
		}
	}
	if (Ratio{1, 0}).Value() != 0 || (Ratio{1, 4}).Value() != 25 {
		t.Error("Value")
	}
	if orNA("") != "n/a" || orNA("1") != "1" {
		t.Error("orNA")
	}
}

// CC-006: a server both critical and stale keeps both on the page.
func TestCriticalAndStaleBothShow(t *testing.T) {
	s, err := Build(fixture.Graph(), Options{Now: fixture.Now, StaleDays: 1})
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := Render(&b, s); err != nil {
		t.Fatal(err)
	}
	// The evidence table keeps both on the critical server's row.
	row := regexp.MustCompile(`(?s)<tr data-f data-kind="server"[^>]*>.*?</tr>`).FindString(b.String())
	if !strings.Contains(row, "Critical findings") || !strings.Contains(row, "Stale evidence") {
		t.Errorf("the critical server's staleness is hidden: %s", row)
	}
}

// The top issue is the worst finding first, says what each count counts,
// and an empty graph's banner makes no safety claim.
func TestTopIssue(t *testing.T) {
	s := build(t, fixture.Graph(), []policy.Violation{{Rule: "r", Path: query.Path{{Label: "a"}}}})
	ti := s.TopIssue
	if ti == nil || ti.Severity != "critical" || ti.Title != "crm failed a critical check in its latest attestation" ||
		ti.Rule != "auth.unauthenticated_tools" || ti.Evidence != "fnd-0" || ti.Client != "cli-0" || ti.Findings != 3 || ti.Gaps != 2 {
		t.Fatalf("top %+v", ti)
	}
	if !strings.Contains(ti.Context, "claude-desktop is configured to reach it. Evidence age: 2 days; stale after 30.") {
		t.Errorf("context %q", ti.Context)
	}
	var b bytes.Buffer
	if err := Render(&b, s); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"View all 5 attention items (3 findings, 2 evidence gaps)", `href="#fnd-0"`, ">View evidence<",
		`data-show-path="cli-0"`, "Download PDF report", `href="#mitigation" data-guide`} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("banner lacks %q", want)
		}
	}
	gapsOnly := build(t, fixture.Graph(), nil)
	gapsOnly.Findings = nil
	if gi := topIssue(gapsOnly); gi == nil || gi.Severity != "warning" || !strings.HasPrefix(gi.Title, "No attestation loaded for") || gi.Client == "" {
		t.Errorf("a gap leads when there is no finding: %+v", gi)
	}
	if empty := build(t, graph.New(), nil); empty.TopIssue != nil {
		t.Errorf("an empty graph has a top issue: %+v", empty.TopIssue)
	}
	b.Reset()
	if err := Render(&b, build(t, graph.New(), nil)); err != nil || !strings.Contains(b.String(), "This covers what was ingested, nothing more") {
		t.Error("the no-issue state overclaims or is missing")
	}
}

type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestRenderReportsAWriteError(t *testing.T) {
	if err := Render(failing{}, build(t, fixture.Graph(), nil)); err == nil {
		t.Fatal("a failed write was not reported")
	}
}

// A server is named the same way wherever a reader meets it: the name it
// reports, or its endpoint when it reports none. The top issue, finding
// and gap titles, and the links to a server, agree with the ring, in the
// full page and the share-safe one. Only its details panel shows the
// endpoint.
func TestOneNamePerServer(t *testing.T) {
	for _, safe := range []bool{false, true} {
		s, err := Build(fixture.Graph(), Options{Now: fixture.Now, StaleDays: 30, ShareSafe: safe})
		if err != nil {
			t.Fatal(err)
		}
		if got := s.TopIssue.Title; got != "crm failed a critical check in its latest attestation" {
			t.Errorf("share-safe %v: top issue %q", safe, got)
		}
		var got []string
		for _, f := range s.Findings {
			got = append(got, f.Title+" | "+f.Server.Name)
		}
		for _, g := range s.Gaps {
			got = append(got, g.Title+" | "+g.Server.Name)
		}
		shell := server(s, map[bool]string{false: fixture.Shell, true: "stdio-server-2"}[safe]).Endpoint
		want := []string{
			"A critical check failed on crm, inherited by claude-desktop | crm",
			"Identity " + map[bool]string{false: "ops-agent", true: "identity-1"}[safe] + " holds more than files's tools need | files",
			"No attestation loaded for " + shell + " | " + shell,
			"Evidence for mail is 40 days old | mail",
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("share-safe %v:\n%s\nwant:\n%s", safe, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
		if !strings.Contains(s.Mitigation.Fix, "failing check on crm,") {
			t.Errorf("share-safe %v: mitigation %q", safe, s.Mitigation.Fix)
		}
		// Search still finds a server's items by its endpoint.
		crm := map[bool]string{false: "crm.example", true: "http-server-1"}[safe]
		if !strings.Contains(s.Findings[0].Text, crm) || !strings.Contains(s.Agents[0].Text, crm) || !strings.Contains(s.Agents[0].Paths[0].Text, crm) {
			t.Errorf("share-safe %v: search text lost the endpoint: %q", safe, s.Findings[0].Text)
		}
		for _, a := range s.Agents {
			for _, p := range a.Paths {
				if strings.Contains(p.Server.Name, "://") || strings.Contains(p.Server.Name, "http-server-") {
					t.Errorf("share-safe %v: path %s -> %s names the server by its endpoint", safe, a.Name, p.Server.Name)
				}
			}
		}
	}
}

// The ring's key reads as a count of servers: "1 with stale evidence", not
// "1 Stale evidence".
func TestTheRingKeyReadsWithItsCounts(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, build(t, fixture.Graph(), nil)); err != nil {
		t.Fatal(err)
	}
	key := regexp.MustCompile(`<p class="ringkey">.*?</p>`).FindString(b.String())
	key = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(key, "")
	want := "Ring: servers by status · 1 with a failed critical check · 1 with no evidence · 1 with stale evidence · 1 with no failing server checks"
	if key != want {
		t.Errorf("ring key %q, want %q", key, want)
	}
}
