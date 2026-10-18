// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package dashboard summarises a graph as one self-contained HTML page: what
// each agent reaches, how each server was last attested, and what the
// analysis and an optional policy found. The page loads nothing: no script,
// no font, no stylesheet from anywhere else, so it can be opened offline,
// attached to a ticket or archived as evidence.
//
// Every number on the page is derived from the graph through the same query
// and analysis code the other commands use, so the dashboard cannot disagree
// with `query`, `analyze` or `check`.
package dashboard

import (
	"fmt"
	"sort"
	"time"

	"satellion.com/passmcp-graph/internal/analyze"
	"satellion.com/passmcp-graph/internal/policy"
	"satellion.com/passmcp-graph/internal/query"
	"satellion.com/passmcp-reporting/graph"
)

// Options are what a summary is computed against.
type Options struct {
	// Now is the time attestation ages are measured from.
	Now time.Time
	// StaleDays is the age in days past which an attestation counts as stale.
	StaleDays int
	// Violations are the result of a policy check, or nil when no policy was
	// given; an empty, non-nil slice means the policy holds.
	Violations []policy.Violation
	// Version is the passmcp-graph version that produced the page.
	Version string
	// ShareSafe replaces endpoints, identities, local paths and
	// token-shaped strings, so the page or JSON can leave the machine.
	ShareSafe bool
	// skipExport leaves out the embedded share-safe export, for the build
	// that produces it.
	skipExport bool
}

// Summary is everything the page shows.
type Summary struct {
	Generated string `json:"generated"`
	Version   string `json:"version"`
	StaleDays int    `json:"staleDays"`

	Totals  Totals      `json:"totals"`
	Agents  []AgentRow  `json:"agents"`
	Servers []ServerRow `json:"servers"`

	Identities     []IdentityRow      `json:"identities"`
	Risks          []analyze.Risk     `json:"risks"`
	OverPrivileged []analyze.Flag     `json:"overPrivileged"`
	Violations     []policy.Violation `json:"violations"`
	// Findings is every finding occurrence above, as its panel shows it;
	// Gaps is every server whose evidence is missing or stale. Together
	// they are the page's attention items.
	Findings []Finding `json:"findings"`
	Gaps     []Gap     `json:"evidenceGaps"`
	// PolicyChecked is true when Violations is the result of a policy.
	PolicyChecked bool `json:"policyChecked"`

	// Hero is the page's centrepiece, laid out from the figures above.
	Hero Hero `json:"-"`
	// Topology is the exact client -> server -> tool diagram.
	Topology Topology `json:"-"`
	// ShareSafe is true when identifying strings were replaced.
	ShareSafe bool `json:"shareSafe"`
	// StatusQuery and BehaviourQuery are each filter tag in the query
	// language, for the page's "reproduce this view" command.
	StatusQuery    map[string]string `json:"-"`
	BehaviourQuery map[string]string `json:"-"`
	// Metrics defines every headline figure: unit, denominator, display.
	Metrics []Metric `json:"metrics"`
	// TopIssue is the one thing to look at first, or nil when the evidence
	// shows none.
	TopIssue *Issue `json:"topIssue"`
	// Attention, Mitigation and Scores feed the command centre; ExportJSON
	// is the share-safe summary the page offers as a download.
	Attention  []AttentionRow `json:"-"`
	Mitigation Mitigation     `json:"-"`
	Scores     []ScoreBar     `json:"-"`
	ExportJSON string         `json:"-"`
}

// Totals are the headline counts.
type Totals struct {
	Agents     int `json:"agents"`
	Servers    int `json:"servers"`
	Tools      int `json:"tools"`
	Identities int `json:"identities"`
	Attested   int `json:"attested"`
	Unattested int `json:"unattested"`
	Stale      int `json:"stale"`
	// Scored is how many servers carry a score; AvgScore is their mean,
	// or "" when none does.
	Scored   int    `json:"scored"`
	AvgScore string `json:"avgScore"`
	// DestructiveTools counts distinct potentially destructive tools, per
	// server.
	DestructiveTools int `json:"destructiveTools"`
}

// AgentRow is one agent and what it reaches.
type AgentRow struct {
	// ID is the graph node; Anchor is the page fragment of its details.
	ID         string `json:"-"`
	Anchor     string `json:"anchor"`
	Name       string `json:"name"`
	Client     string `json:"client"`
	ConfigFile string `json:"configFile"`
	Servers    int    `json:"servers"`
	// ReadOnly tools declare readOnlyHint true; Writable tools do not, so
	// they may change things; Destructive is the part of Writable that may
	// also destroy them, and Unannotated the part that declares nothing, so
	// its behaviour is unknown rather than observed.
	ReadOnly    int `json:"readOnlyTools"`
	Writable    int `json:"writableTools"`
	Destructive int `json:"destructiveTools"`
	Unannotated int `json:"unannotatedTools"`
	// DeclaredDestructive and PresumedDestructive split Destructive by
	// basis: the tool declares it, or the MCP default applies.
	DeclaredDestructive int `json:"declaredDestructiveTools"`
	PresumedDestructive int `json:"presumedDestructiveTools"`
	Unattested          int `json:"unattestedServers"`
	Risks               int `json:"risks"`
	// Bar widths, in percent of the widest agent, for the reach chart.
	ReadOnlyPct, MutatingPct, UnannotatedPct int `json:"-"`
	// Paths and Identities are the agent's details.
	Paths      []PathInfo `json:"paths"`
	Identities []Ref      `json:"identities"`
	// Tags and Text are what the page's filters match; Query selects the
	// agent's paths in the query language, or is empty when it cannot.
	Tags, Text string `json:"-"`
	Query      string `json:"query,omitempty"`
	// Status is the worst status among the servers the agent is
	// configured to reach, "" when it reaches none; StatusLabel names it.
	// The command centre colours the agent's entry and flow with it.
	Status      string `json:"-"`
	StatusLabel string `json:"-"`
}

// Mutating is the writable tools that declare their behaviour.
func (a AgentRow) Mutating() int { return a.Writable - a.Unannotated }

// Tools is every tool the agent reaches.
func (a AgentRow) Tools() int { return a.ReadOnly + a.Writable }

// ServerRow is one server and its latest attested verdict.
type ServerRow struct {
	ID          string   `json:"-"`
	Anchor      string   `json:"anchor"`
	Endpoint    string   `json:"endpoint"`
	Name        string   `json:"name"`
	Transport   string   `json:"transport"`
	Auth        string   `json:"auth"`
	Grade       string   `json:"grade"`
	Score       string   `json:"score"`
	AgeDays     string   `json:"ageDays"`
	Tools       int      `json:"tools"`
	Destructive int      `json:"destructiveTools"`
	Agents      int      `json:"agents"`
	Critical    []string `json:"critical"`
	// Stale is true when the latest attestation is older than StaleDays.
	Stale bool `json:"stale"`
	// Status is "critical", "unattested", "stale" or "ok", worst first;
	// StatusLabel is how the page names it.
	Status      string `json:"status"`
	StatusLabel string `json:"statusLabel"`
	// AuthLabel names Auth for people, "not in evidence" when absent.
	AuthLabel string `json:"-"`
	// RelatedFindings counts the findings that name this server: a server
	// with no failing checks can still carry an identity or policy finding.
	RelatedFindings int `json:"relatedFindings"`
	// Bubble is the diameter in pixels of the server's bubble on the page,
	// grown with the tools it exposes.
	Bubble int `json:"-"`
	ServerDetail
	Tags, Text string `json:"-"`
}

// Build computes the summary of g.
func Build(g *graph.Graph, o Options) (*Summary, error) {
	ctx := query.Context{Now: o.Now}
	reach, err := reachOf(g, ctx)
	if err != nil {
		return nil, err
	}
	s := &Summary{
		Generated:      o.Now.UTC().Format(time.RFC3339),
		Version:        o.Version,
		StaleDays:      o.StaleDays,
		Risks:          nonNil(analyze.Risks(g)),
		OverPrivileged: nonNil(analyze.OverPrivileged(g)),
		Violations:     nonNil(o.Violations),
		PolicyChecked:  o.Violations != nil,
	}
	s.Servers = serverRows(g, o, reach)
	s.Agents = agentRows(g, reach, s.Risks)
	worstReached(s.Agents, s.Servers, reach)
	s.Totals = totals(g, s.Servers)
	addDetails(g, s, reach)
	s.StatusQuery, s.BehaviourQuery = statusQuery(o.StaleDays), behaviourQuery
	for i := range s.Agents {
		s.Agents[i].Query = clientQuery(s.Agents[i])
	}
	sizeBubbles(s.Servers)
	s.Hero = buildHero(s)
	s.Metrics = metrics(s)
	s.Findings = buildFindings(s)
	s.Gaps = buildGaps(s)
	relate(s)
	s.TopIssue = topIssue(s)
	addTags(s)
	s.Topology = buildTopology(s)
	if o.ShareSafe {
		shareSafe(s)
	}
	s.Attention, s.Mitigation, s.Scores = attentionRows(s), mitigation(s), scoreBars(s)
	if !o.skipExport {
		if s.ExportJSON, err = exportJSON(g, o); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// reach is the result of the queries the page is built from, keyed by node ID.
type reach struct {
	// agentServers and serverAgents are the uses edges, both ways.
	agentServers, serverAgents map[string]set
	// readOnly, writable, destructive and unannotated are the tools each
	// agent reaches, by what they declare.
	readOnly, writable, destructive, unannotated map[string]set
	// serverDestructive is each server's destructive tools.
	serverDestructive map[string]set
	// tools classifies every tool by the query language's predicates.
	tools struct{ readOnly, unannotated, destructive set }
}

type set map[string]bool

func (s set) add(k string) set {
	if s == nil {
		s = set{}
	}
	s[k] = true
	return s
}

// reachOf runs the queries that define what an agent reaches. Using the
// query language, not a second walk of the graph, keeps "destructive" here
// the same predicate `query` documents.
func reachOf(g *graph.Graph, ctx query.Context) (*reach, error) {
	r := &reach{
		agentServers: map[string]set{}, serverAgents: map[string]set{},
		readOnly: map[string]set{}, writable: map[string]set{}, destructive: map[string]set{}, unannotated: map[string]set{},
		serverDestructive: map[string]set{},
	}
	runs := []struct {
		q    string
		into func(query.Path)
	}{
		{"agents -> servers", func(p query.Path) {
			r.agentServers[p[0].ID] = r.agentServers[p[0].ID].add(p[1].ID)
			r.serverAgents[p[1].ID] = r.serverAgents[p[1].ID].add(p[0].ID)
		}},
		{"agents -> tools where destructive", func(p query.Path) { r.destructive[p[0].ID] = r.destructive[p[0].ID].add(p[2].ID) }},
		{"agents -> tools where readonly", func(p query.Path) { r.readOnly[p[0].ID] = r.readOnly[p[0].ID].add(p[2].ID) }},
		{"agents -> tools where not readonly", func(p query.Path) { r.writable[p[0].ID] = r.writable[p[0].ID].add(p[2].ID) }},
		{"agents -> tools where not annotated", func(p query.Path) { r.unannotated[p[0].ID] = r.unannotated[p[0].ID].add(p[2].ID) }},
		{"servers -> tools where destructive", func(p query.Path) {
			r.serverDestructive[p[0].ID] = r.serverDestructive[p[0].ID].add(p[1].ID)
		}},
		{"tools where readonly", func(p query.Path) { r.tools.readOnly = r.tools.readOnly.add(p[0].ID) }},
		{"tools where not annotated", func(p query.Path) { r.tools.unannotated = r.tools.unannotated.add(p[0].ID) }},
		{"tools where destructive", func(p query.Path) { r.tools.destructive = r.tools.destructive.add(p[0].ID) }},
	}
	for _, run := range runs {
		q, err := query.Parse(run.q)
		if err != nil {
			return nil, fmt.Errorf("dashboard: %q: %w", run.q, err)
		}
		paths, err := query.Eval(g, q, ctx)
		if err != nil {
			return nil, fmt.Errorf("dashboard: %q: %w", run.q, err)
		}
		for _, p := range paths {
			run.into(p)
		}
	}
	return r, nil
}

// agentRows lists every agent, the widest reach first, with its bar widths.
func agentRows(g *graph.Graph, r *reach, risks []analyze.Risk) []AgentRow {
	riskCount := map[string]int{}
	for _, k := range risks {
		riskCount[k.Agent]++
	}
	var rows []AgentRow
	for _, n := range g.Nodes {
		if n.Kind != graph.KindAgent {
			continue
		}
		row := AgentRow{ID: n.ID, Name: n.Label, Servers: len(r.agentServers[n.ID]),
			ReadOnly: len(r.readOnly[n.ID]), Writable: len(r.writable[n.ID]), Destructive: len(r.destructive[n.ID]),
			Unannotated: len(r.unannotated[n.ID]),
			Risks:       riskCount[n.Label]}
		if n.Agent != nil {
			row.Client, row.ConfigFile = n.Agent.Client, n.Agent.ConfigFile
		}
		for sid := range r.agentServers[n.ID] {
			if s, ok := g.Node(sid); ok && (s.Server == nil || s.Server.Attestation == "") {
				row.Unattested++
			}
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Tools() != rows[j].Tools() {
			return rows[i].Tools() > rows[j].Tools()
		}
		return rows[i].Name+rows[i].ConfigFile < rows[j].Name+rows[j].ConfigFile
	})
	scaleBars(rows)
	return rows
}

// worstReached sets each agent's Status to the worst status among the
// servers its configuration names.
func worstReached(agents []AgentRow, servers []ServerRow, r *reach) {
	byID := map[string]string{}
	for _, sv := range servers {
		byID[sv.ID] = sv.Status
	}
	for i := range agents {
		for sid := range r.agentServers[agents[i].ID] {
			st, ok := byID[sid]
			if ok && (agents[i].Status == "" || statusRank[st] < statusRank[agents[i].Status]) {
				agents[i].Status = st
			}
		}
		agents[i].StatusLabel = statusLabel[agents[i].Status]
	}
}

// scaleBars sets each row's bar widths relative to the widest row.
func scaleBars(rows []AgentRow) {
	widest := 0
	for _, a := range rows {
		widest = max(widest, a.Tools())
	}
	if widest == 0 {
		return
	}
	for i := range rows {
		rows[i].ReadOnlyPct = rows[i].ReadOnly * 100 / widest
		rows[i].MutatingPct = rows[i].Mutating() * 100 / widest
		rows[i].UnannotatedPct = rows[i].Unannotated * 100 / widest
	}
}
