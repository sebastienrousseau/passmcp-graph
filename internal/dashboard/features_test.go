// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-graph/internal/policy"
	"satellion.com/passmcp-graph/internal/query"
	"satellion.com/passmcp-reporting/graph"
)

func server(s *Summary, endpoint string) ServerRow {
	for _, sv := range s.Servers {
		if sv.Endpoint == endpoint {
			return sv
		}
	}
	return ServerRow{}
}

// CC-019: the topology draws exactly the configured relationships, and a
// server sits within its own block of tools, so its edges cannot cross.
func TestTopologyIsExact(t *testing.T) {
	s := build(t, fixture.Graph(), nil)
	tp := s.Topology
	cs, ts := 0, 0
	for _, e := range tp.Edges {
		switch {
		case strings.HasPrefix(e.Class, "cs"):
			cs++
		case strings.HasPrefix(e.Class, "ts"):
			ts++
		}
	}
	// claude-desktop uses 3 servers and cursor 2; the 4 servers expose 5 tools.
	if len(tp.Clients) != 2 || len(tp.Servers) != 4 || len(tp.Tool) != 5 || cs != 5 || ts != 5 || tp.Folded || tp.Hidden != "" {
		t.Fatalf("clients %d servers %d tools %d, edges %d+%d, folded %v, hidden %q",
			len(tp.Clients), len(tp.Servers), len(tp.Tool), cs, ts, tp.Folded, tp.Hidden)
	}
	ti := 0
	for _, sv := range tp.Servers {
		row := server(s, sv.Full)
		first, last := tp.Tool[ti].Y, tp.Tool[ti+len(row.ToolList)-1].Y
		if sv.Y < first || sv.Y > last || math.Abs(sv.Y-(first+last)/2) > 0.01 {
			t.Errorf("%s at %v is not centred on its tools %v..%v", sv.Full, sv.Y, first, last)
		}
		ti += len(row.ToolList)
	}
	for i := 1; i < len(tp.Tool); i++ {
		if tp.Tool[i].Y <= tp.Tool[i-1].Y {
			t.Errorf("tool rows are not in order at %d", i)
		}
	}
	for _, n := range tp.Clients {
		if !strings.Contains(n.Tags, n.Anchor) {
			t.Errorf("client node %s is not tagged with itself: %q", n.Anchor, n.Tags)
		}
	}
}

// A large estate folds each server's tools into one node and draws the
// first clients and servers; the path table stays complete.
func TestTopologyFoldsAndCaps(t *testing.T) {
	s := &Summary{}
	for i := range 45 {
		sv := ServerRow{Anchor: fmt.Sprintf("srv-%d", i), Endpoint: fmt.Sprintf("https://s%d.example/mcp", i), Status: "ok"}
		for j := range 2 {
			sv.ToolList = append(sv.ToolList, ToolInfo{Name: fmt.Sprintf("t%d", j), Behavior: readOnly})
		}
		s.Servers = append(s.Servers, sv)
	}
	for i := range 41 {
		s.Agents = append(s.Agents, AgentRow{Anchor: fmt.Sprintf("cli-%d", i), Name: fmt.Sprintf("c%d", i)})
	}
	tp := buildTopology(s)
	if !tp.Folded || len(tp.Servers) != topoMaxServers || len(tp.Clients) != topoMaxClients || len(tp.Tool) != topoMaxServers ||
		tp.Hidden != "1 more client(s) and 5 more server(s)" || !strings.HasPrefix(tp.Tool[0].Label, "2 tools") {
		t.Fatalf("folded %v, %d servers, %d clients, %d tool nodes, hidden %q, %q", tp.Folded, len(tp.Servers), len(tp.Clients), len(tp.Tool), tp.Hidden, tp.Tool[0].Label)
	}
	if clip("abcdef", 4) != "abc…" || clip("abc", 4) != "abc" {
		t.Error("clip")
	}
}

// CC-024 to CC-026: findings, servers, clients and identities carry what
// their panels show.
func TestDetails(t *testing.T) {
	s := build(t, fixture.Graph(), []policy.Violation{})
	crm := server(s, fixture.CRM)
	if crm.Anchor == "" || len(crm.ToolList) != 2 || crm.ToolList[0].Name != "delete_record" || crm.ToolList[0].Behavior != mutating ||
		crm.ToolList[0].DestructiveBasis != "declared" || crm.ToolList[1].Behavior != readOnly || crm.ToolList[1].Destructive ||
		crm.RanAt == "" || crm.Attestation == "" || len(crm.Failing) != 1 || len(crm.Clients) != 1 || !strings.Contains(crm.Query, "server.endpoint = 'https://crm.example/mcp'") {
		t.Errorf("crm %+v", crm.ServerDetail)
	}
	shell := server(s, fixture.Shell)
	if shell.ToolList[0].Behavior != unannotated || shell.ToolList[0].DestructiveBasis != "presumed" || shell.AuthLabel != "not in evidence" {
		t.Errorf("shell %+v, auth %q", shell.ToolList, shell.AuthLabel)
	}
	files := server(s, fixture.Files)
	if files.StatusLabel != "No failing server checks" || files.RelatedFindings != 1 || len(files.Grants) != 1 || !files.Grants[0].OverPrivileged {
		t.Errorf("files: %q, %d related, grants %+v", files.StatusLabel, files.RelatedFindings, files.Grants)
	}
	var claude AgentRow
	for _, a := range s.Agents {
		if a.Name == "claude-desktop" {
			claude = a
		}
	}
	if len(claude.Paths) != 4 || claude.DeclaredDestructive != 1 || claude.PresumedDestructive != 2 || len(claude.Identities) != 1 {
		t.Errorf("claude-desktop %d paths, %d declared, %d presumed, identities %+v", len(claude.Paths), claude.DeclaredDestructive, claude.PresumedDestructive, claude.Identities)
	}
	if len(s.Identities) != 2 {
		t.Fatalf("identities %+v", s.Identities)
	}
	for _, id := range s.Identities {
		if id.Label == "ops-agent" && (!id.OverPrivileged || len(id.Clients) != 1 || id.Clients[0].Name != "cursor") {
			t.Errorf("ops-agent %+v", id)
		}
	}
	quoted := ServerRow{Endpoint: "npx 'odd'", ID: "x"}
	if d := serverDetail(graph.New(), anchors{}, &reach{}, s, quoted); d.Query != "" {
		t.Errorf("an endpoint that cannot be quoted still gets a query: %q", d.Query)
	}
	if authLabel("bearer") != "Bearer" || authLabel("OAuth") != "OAuth" || authLabel("none") != "None" {
		t.Error("authLabel")
	}
}

// P0 from review: attention items are findings and evidence gaps, counted
// apart, and every finding names its evidence and a read-only next step.
func TestFindingsAndGaps(t *testing.T) {
	vs := []policy.Violation{{Rule: "unattested", Path: query.Path{{Kind: "agent", Label: "claude-desktop"}, {Kind: "server", Label: fixture.Shell}}}}
	s := build(t, fixture.Graph(), vs)
	var kinds []string
	for i, f := range s.Findings {
		kinds = append(kinds, f.Kind)
		if f.Anchor != fmt.Sprintf("fnd-%d", i) || f.Title == "" || f.Evidence == "" || f.Uncertainty == "" || f.Next == "" || f.Server.Anchor == "" {
			t.Errorf("finding %d is incomplete: %+v", i, f)
		}
	}
	if strings.Join(kinds, " ") != "inherited-critical policy over-privileged" {
		t.Errorf("kinds %v", kinds)
	}
	if v := s.Findings[1]; v.Server.Name != fixture.Shell || len(v.Clients) != 1 || v.Clients[0].Name != "claude-desktop" {
		t.Errorf("violation %+v", v)
	}
	if len(s.Gaps) != 2 || s.Gaps[0].Kind != "no-evidence" || s.Gaps[1].Kind != "stale-evidence" || s.Gaps[1].Anchor != "gap-1" ||
		!strings.Contains(s.Gaps[1].Title, "is 40 days old") {
		t.Errorf("gaps %+v", s.Gaps)
	}
	if server(s, fixture.Shell).RelatedFindings != 1 || server(s, fixture.CRM).RelatedFindings != 1 {
		t.Errorf("related findings: shell %d, crm %d", server(s, fixture.Shell).RelatedFindings, server(s, fixture.CRM).RelatedFindings)
	}
	if scopeQuery([]string{"it's"}) != "passmcp-graph analyze" || scopeQuery(nil) != "passmcp-graph analyze" {
		t.Error("scopeQuery quotes what it cannot")
	}
	if attestationEvidence(ServerRow{}) != "No attestation on record." || shortDigest("abc") != "abc" {
		t.Error("evidence helpers")
	}
}

// CC-027: tags let one filter state scope every widget alike.
func TestTags(t *testing.T) {
	s := build(t, fixture.Graph(), nil)
	for _, a := range s.Agents {
		for _, p := range a.Paths {
			has := func(tag string) bool { return strings.Contains(" "+p.Tags+" ", " "+tag+" ") }
			if !has(a.Anchor) {
				t.Errorf("%s -> %s is not tagged with its client", a.Name, p.Name)
			}
			if has("destr") != p.Destructive || has("unann") != (p.Behavior == unannotated) {
				t.Errorf("%s -> %s tags %q", a.Name, p.Name, p.Tags)
			}
			if endpointOf(s, p.Server.Anchor) == fixture.Mail && !has("st-stale") {
				t.Errorf("a path through the stale server is not tagged stale: %q", p.Tags)
			}
		}
	}
	if crm := server(s, fixture.CRM); !strings.Contains(crm.Text, "delete_record") || !strings.Contains(crm.Tags, "st-critical") {
		t.Errorf("crm text %q tags %q", crm.Text, crm.Tags)
	}
}

// CC-037: a share-safe page or summary contains none of the planted
// private values, and keeps what it needs to be read.
func TestShareSafeRemovesCanaries(t *testing.T) {
	g := graph.New()
	ep := "https://mcp.internal.acme-corp.example/v1"
	cmd := "/home/alice/bin/secret-mcp --token sk-live-abcdefghijklmnopqrstuv"
	for _, sv := range []struct{ tr, ep string }{{"http", ep}, {"stdio", cmd}} {
		id := graph.ServerID(sv.tr, sv.ep)
		g.Upsert(graph.Node{ID: id, Kind: graph.KindServer, Label: sv.ep, Server: &graph.ServerProps{Transport: sv.tr, Endpoint: sv.ep, Name: "crm"}})
		tid := graph.ToolID(id, "read_file")
		g.Upsert(graph.Node{ID: tid, Kind: graph.KindTool, Label: "read_file", Tool: &graph.ToolProps{Name: "read_file"}})
		g.Link(graph.Edge{From: id, To: tid, Kind: graph.Exposes})
		g.Link(graph.Edge{From: "agent:a", To: id, Kind: graph.Uses})
	}
	tok := graph.ToolID(graph.ServerID("http", ep), "ghp_abcdefghijklmnopqrstuvwxyz0123")
	g.Upsert(graph.Node{ID: tok, Kind: graph.KindTool, Label: "ghp_abcdefghijklmnopqrstuvwxyz0123", Tool: &graph.ToolProps{Name: "x"}})
	g.Link(graph.Edge{From: graph.ServerID("http", ep), To: tok, Kind: graph.Exposes})
	g.Upsert(graph.Node{ID: "agent:a", Kind: graph.KindAgent, Label: "claude-desktop",
		Agent: &graph.AgentProps{Client: "claude-desktop", ConfigFile: "/Users/alice/Library/Application Support/Claude/claude_desktop_config.json"}})
	idn := graph.IdentityID("https://login.acme-corp.example", "alice-deploy-bot")
	g.Upsert(graph.Node{ID: idn, Kind: graph.KindIdentity, Label: "alice-deploy-bot",
		Identity: &graph.IdentityProps{Issuer: "https://login.acme-corp.example", ClientID: "alice-deploy-bot"}})
	g.Link(graph.Edge{From: idn, To: graph.ServerID("http", ep), Kind: graph.Authorizes, Scopes: []string{"admin"}})
	vs := []policy.Violation{{Rule: "no-internal", Reason: "mail alice@acme-corp.example", Path: query.Path{{Kind: "agent", Label: "claude-desktop"}, {Kind: "server", Label: ep}}}}

	s, err := Build(g, Options{Now: fixture.Now, StaleDays: 30, Violations: vs, ShareSafe: true})
	if err != nil {
		t.Fatal(err)
	}
	var page bytes.Buffer
	if err := Render(&page, s); err != nil {
		t.Fatal(err)
	}
	js, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string]string{"html": page.String(), "json": string(js)} {
		for _, canary := range []string{"alice", "acme-corp", "internal.", "sk-live", "ghp_", "/Users/", "/home/", "secret-mcp", "Application Support"} {
			if strings.Contains(out, canary) {
				t.Errorf("%s output contains %q", name, canary)
			}
		}
		for _, kept := range []string{"claude_desktop_config.json", "read_file", "claude-desktop", "http-server-", "identity-1"} {
			if !strings.Contains(out, kept) {
				t.Errorf("%s output lost %q", name, kept)
			}
		}
	}
	if !s.ShareSafe || !strings.Contains(page.String(), "Share-safe") {
		t.Error("the page does not say it is share-safe")
	}
	if scrub("digest 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef") != "digest 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Error("a SHA-256 digest is evidence and must survive")
	}
}

// CC-033: the text and status colours meet WCAG 2.2 AA against the page
// background and a panel, in both themes, computed from the page's own
// tokens so a change to a colour is checked when it is made.
func TestContrast(t *testing.T) {
	light := tokens(t, page[strings.Index(page, ":root {"):])
	dark := tokens(t, page[strings.Index(page, `:root[data-theme="dark"] {`):])
	for theme, tk := range map[string]map[string]string{"light": light, "dark": dark} {
		bg := rgb(t, tk["--bg"])
		panel := over(rgba(t, tk["--glass"]), bg)
		for _, name := range []string{"--ink", "--ink-2", "--muted", "--accent"} {
			for surface, s := range map[string][3]float64{"background": bg, "panel": panel} {
				if c := contrast(rgb(t, tk[name]), s); c < 4.5 {
					t.Errorf("%s: text %s on the %s is %.2f:1, under 4.5:1", theme, name, surface, c)
				}
			}
		}
		for _, name := range []string{"--crit", "--unatt", "--stale", "--ok", "--ro", "--wr"} {
			if c := contrast(rgb(t, tk[name]), panel); c < 3 {
				t.Errorf("%s: graphic %s on a panel is %.2f:1, under 3:1", theme, name, c)
			}
		}
	}
}

var tokenRe = regexp.MustCompile(`(--[a-z0-9-]+):\s*([^;]+);`)

func tokens(t *testing.T, css string) map[string]string {
	t.Helper()
	block := css[:strings.Index(css, "}")]
	out := map[string]string{}
	for _, m := range tokenRe.FindAllStringSubmatch(block, -1) {
		out[m[1]] = strings.TrimSpace(m[2])
	}
	return out
}

func rgb(t *testing.T, hex string) [3]float64 {
	t.Helper()
	if len(hex) != 7 || hex[0] != '#' {
		t.Fatalf("not a #rrggbb colour: %q", hex)
	}
	var c [3]float64
	for i := range 3 {
		v, err := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		if err != nil {
			t.Fatal(err)
		}
		c[i] = float64(v)
	}
	return c
}

func rgba(t *testing.T, s string) [4]float64 {
	t.Helper()
	var c [4]float64
	if _, err := fmt.Sscanf(s, "rgba(%g, %g, %g, %g)", &c[0], &c[1], &c[2], &c[3]); err != nil {
		t.Fatalf("not an rgba() colour: %q", s)
	}
	return c
}

func over(fg [4]float64, bg [3]float64) [3]float64 {
	var c [3]float64
	for i := range 3 {
		c[i] = fg[i]*fg[3] + bg[i]*(1-fg[3])
	}
	return c
}

func luminance(c [3]float64) float64 {
	var l [3]float64
	for i, v := range c {
		v /= 255
		if v <= 0.04045 {
			l[i] = v / 12.92
		} else {
			l[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*l[0] + 0.7152*l[1] + 0.0722*l[2]
}

func contrast(a, b [3]float64) float64 {
	la, lb := luminance(a), luminance(b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// The browser check of the page's script: filters, client selection,
// fragments and reflow at 320 CSS pixels, driven in a real Chrome. It needs
// Chrome, so it runs only when PASSMCP_GRAPH_BROWSER is the path to one.
func TestInTheBrowser(t *testing.T) {
	chrome := os.Getenv("PASSMCP_GRAPH_BROWSER")
	if chrome == "" {
		t.Skip("set PASSMCP_GRAPH_BROWSER to a Chrome binary to run the browser check")
	}
	g := fixture.Graph()
	got := runProbe(t, chrome, build(t, g, []policy.Violation{}))
	for _, want := range []string{
		"hash srv-1 open: true",
		"pick cursor: paths 2, servers 2",
		"2 of 6 paths · 2 of 4 attention items (1 finding, 1 evidence gap)",
		"attention: 2 attention items in this selection: 1 finding · 1 evidence gap. 4 attention items in the whole snapshot.",
		"selection pill: true cursor",
		"topology filtered, hidden: 12 of 12",
		"topology with context, hidden: 0",
		"repro label: Reproduce this view:",
		"search crm: servers 1, paths 2, findings 1",
		"search repro label: Closest command:",
		"status stale: servers 1",
		"unannotated: paths 3",
		"cleared: paths 6, servers 4, attention 4, hidden topology 0 | Showing the whole snapshot.",
		"all columns: true",
		"back shown true, ← Back to A critical check failed",
		"after back: hash #fnd-0, focus in fnd-0 true",
		"320px: scrollWidth 320 clientWidth 320; server status clipped without ellipsis: 0, at 1440px 0",
		"flows drawn: 18",
		"severity low: rows 2",
		"ctrl-k focuses search: true",
		"themes: light dark system",
		"hero pick: cursor, pressed true",
		"export: blobs 1, CSP violations 0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("browser check lacks %q in:\n%s", want, got)
		}
	}
	// The command the page printed for cursor and stale evidence returns the
	// number of paths the page showed.
	cmd := regexp.MustCompile(`repro: passmcp-graph query "(.*)"`).FindStringSubmatch(got)
	shown := regexp.MustCompile(`repro paths shown: (\d+)`).FindStringSubmatch(got)
	if cmd == nil || shown == nil {
		t.Fatalf("no command or count in:\n%s", got)
	}
	q, err := query.Parse(cmd[1])
	if err != nil {
		t.Fatal(err)
	}
	paths, err := query.Eval(g, q, query.Context{Now: fixture.Now})
	if err != nil {
		t.Fatal(err)
	}
	if strconv.Itoa(len(paths)) != shown[1] {
		t.Errorf("the page showed %s path(s); its command %q returns %d", shown[1], cmd[1], len(paths))
	}
}

// The page's "reproduce this view" command, assembled as ReproQuery
// assembles it, returns exactly the paths the page's filters keep, for
// every client, every set of statuses and every set of tool behaviours.
func TestReproduceThisViewReturnsTheShownPaths(t *testing.T) {
	g := fixture.Graph()
	s := build(t, g, nil)
	statuses := []string{"st-critical", "st-unattested", "st-stale", "st-ok"}
	behaviours := []string{"destr", "unann"}
	clients := append([]AgentRow{{}}, s.Agents...)
	combos := 0
	for _, c := range clients {
		for sm := range 1 << len(statuses) {
			for bm := range 1 << len(behaviours) {
				sts, bhs := pick(statuses, sm), pick(behaviours, bm)
				want := shownPaths(s, c.Anchor, sts, bhs)
				var stq, bhq []string
				for _, x := range sts {
					stq = append(stq, s.StatusQuery[x])
				}
				for _, x := range bhs {
					bhq = append(bhq, s.BehaviourQuery[x])
				}
				src := ReproQuery(c.Query, stq, bhq)
				q, err := query.Parse(src)
				if err != nil {
					t.Fatalf("%s: %v", src, err)
				}
				paths, err := query.Eval(g, q, query.Context{Now: fixture.Now})
				if err != nil {
					t.Fatal(err)
				}
				got := map[string]bool{}
				for _, p := range paths {
					got[p.String()] = true
				}
				if fmt.Sprint(sortedKeys(got)) != fmt.Sprint(sortedKeys(want)) {
					t.Errorf("%s\n page shows %v\n query gives %v", src, sortedKeys(want), sortedKeys(got))
				}
				combos++
			}
		}
	}
	if combos != 3*16*4 {
		t.Fatalf("ran %d combinations", combos)
	}
}

func pick(all []string, mask int) []string {
	var out []string
	for i, x := range all {
		if mask&(1<<i) != 0 {
			out = append(out, x)
		}
	}
	return out
}

// shownPaths mirrors the page script's matches() for the path table.
func shownPaths(s *Summary, client string, sts, bhs []string) map[string]bool {
	has := func(tags, t string) bool { return strings.Contains(" "+tags+" ", " "+t+" ") }
	any := func(tags string, ts []string) bool {
		for _, t := range ts {
			if has(tags, t) {
				return true
			}
		}
		return len(ts) == 0
	}
	out := map[string]bool{}
	for _, a := range s.Agents {
		for _, p := range a.Paths {
			if (client == "" || has(p.Tags, client)) && any(p.Tags, sts) && any(p.Tags, bhs) {
				out[a.Name+" -> "+endpointOf(s, p.Server.Anchor)+" -> "+p.Name] = true
			}
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// The command centre's list, mitigation path and score bars come from the
// summary, and its icons are chosen from what is known about each entity.
func TestCommandCentre(t *testing.T) {
	s := build(t, fixture.Graph(), nil)
	var got []string
	for _, r := range s.Attention {
		got = append(got, r.Severity+": "+r.Title+" | "+r.Path+" | "+r.Age)
	}
	want := []string{
		"critical: crm failed a critical check | claude-desktop → crm | 2 days",
		"medium: Over-privileged identity on files | cursor → files | 1 day",
		"low: No attestation for npx @example/shell-mcp | claude-desktop → npx @example/shell-mcp | Not supplied",
		"low: Stale evidence for mail | claude-desktop and 1 more → mail | 40 days",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("attention rows:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !strings.Contains(s.Mitigation.Fix, "Fix the failing check on crm,") ||
		!strings.Contains(s.Mitigation.Verify, `passmcp-graph query "agents -> servers where critical"`) {
		t.Errorf("mitigation %+v", s.Mitigation)
	}
	gapsOnly := build(t, fixture.Graph(), nil)
	gapsOnly.Findings = nil
	if m := mitigation(gapsOnly); !strings.HasPrefix(m.Fix, "Run passmcp against npx @example/shell-mcp") {
		t.Errorf("gap mitigation %+v", m)
	}
	if none := mitigation(&Summary{}); !strings.HasPrefix(none.Review, "Open the evidence") {
		t.Errorf("default mitigation %+v", none)
	}
	if len(s.Scores) != 3 || s.Scores[0].Name != "crm" || s.Scores[0].Pct != 41 {
		t.Errorf("score bars %+v", s.Scores)
	}
	for _, c := range []struct {
		sv   ServerRow
		icon string
	}{{ServerRow{Transport: "stdio", Endpoint: "npx x"}, "term"}, {ServerRow{Name: "mail"}, "mail"}, {ServerRow{Name: "files"}, "folder"}, {ServerRow{Name: "crm"}, "db"}} {
		if got := serverIcon(c.sv); got != c.icon {
			t.Errorf("serverIcon(%s) = %q, want %q", c.sv.Name, got, c.icon)
		}
	}
	for c, icon := range map[string]string{"claude-desktop": "spark", "cursor": "cube", "vscode": "brackets", "zed": "code"} {
		if got := clientIcon(c); got != icon {
			t.Errorf("clientIcon(%q) = %q, want %q", c, got, icon)
		}
	}
	if ageText(ServerRow{ServerDetail: ServerDetail{Attestation: "d"}}) != "Unknown" || refPath(nil, Ref{Name: "s"}) != "s" {
		t.Error("ageText or refPath")
	}
	if shortTitle(Finding{Kind: "other", Title: "T"}, "s") != "T" {
		t.Error("shortTitle falls back to the title")
	}
}

// The page's "Export JSON (share-safe)" carries the share-safe summary even
// when the page itself is not share-safe.
func TestTheEmbeddedExportIsShareSafe(t *testing.T) {
	s := build(t, fixture.Graph(), nil)
	var doc map[string]any
	if err := json.Unmarshal([]byte(s.ExportJSON), &doc); err != nil {
		t.Fatalf("the export is not JSON: %v", err)
	}
	if doc["shareSafe"] != true || strings.Contains(s.ExportJSON, "/home/dev/") || strings.Contains(s.ExportJSON, "crm.example") {
		t.Error("the embedded export is not share-safe")
	}
	var b bytes.Buffer
	if err := Render(&b, s); err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)<script type="application/json" id="export">(.*?)</script>`).FindStringSubmatch(b.String())
	if m == nil {
		t.Fatal("the page carries no export")
	}
	var text string
	if err := json.Unmarshal([]byte(m[1]), &text); err != nil || text != s.ExportJSON {
		t.Errorf("the page's export does not decode to the summary: %v", err)
	}
	if !strings.Contains(b.String(), "/home/dev/") {
		t.Error("the page itself was redacted; only the export should be")
	}
}
