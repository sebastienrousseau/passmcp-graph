// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"satellion.com/passmcp-reporting/graph"
)

// sizeBubbles sets each server's bubble so its area, not its radius, is
// proportional to the tools it exposes: the server exposing most is 168
// pixels across, and none is under 72 so its label stays legible.
func sizeBubbles(rows []ServerRow) {
	most := 1
	for _, r := range rows {
		most = max(most, r.Tools)
	}
	for i := range rows {
		d := 168 * math.Sqrt(float64(rows[i].Tools)/float64(most))
		rows[i].Bubble = max(72, int(math.Round(d)))
	}
}

// statusLabel is each status as the page names it. "No failing server
// checks" is scoped to the server's own diagnostics on purpose: identity
// and policy findings are counted apart.
var statusLabel = map[string]string{
	"critical":   "Critical findings",
	"unattested": "No evidence",
	"stale":      "Stale evidence",
	"ok":         "No failing server checks",
}

func authLabel(a string) string {
	switch strings.ToLower(a) {
	case "":
		return "not in evidence"
	case "oauth":
		return "OAuth"
	case "none":
		return "None"
	}
	return strings.ToUpper(a[:1]) + a[1:]
}

// statusRank orders server statuses worst first.
var statusRank = map[string]int{"critical": 0, "unattested": 1, "stale": 2, "ok": 3}

// serverRows lists every server, the worst status first.
func serverRows(g *graph.Graph, o Options, r *reach) []ServerRow {
	var rows []ServerRow
	for _, n := range g.Nodes {
		if n.Kind == graph.KindServer {
			rows = append(rows, serverRow(g, o, r, n))
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if statusRank[rows[i].Status] != statusRank[rows[j].Status] {
			return statusRank[rows[i].Status] < statusRank[rows[j].Status]
		}
		return rows[i].Endpoint < rows[j].Endpoint
	})
	return rows
}

// serverRow is one server's row.
func serverRow(g *graph.Graph, o Options, r *reach, n graph.Node) ServerRow {
	p := graph.ServerProps{}
	if n.Server != nil {
		p = *n.Server
	}
	row := ServerRow{ID: n.ID, Endpoint: n.Label, Name: p.Name, Transport: p.Transport, Auth: p.Auth, Grade: p.Grade,
		Tools: len(g.EdgesFrom(n.ID, graph.Exposes)), Destructive: len(r.serverDestructive[n.ID]),
		Agents: len(r.serverAgents[n.ID]), Critical: []string{}}
	if p.Score != nil {
		row.Score = strconv.FormatFloat(*p.Score, 'f', -1, 64)
	}
	for _, f := range p.Failing {
		if strings.EqualFold(f.Severity, "critical") {
			row.Critical = append(row.Critical, f.ID)
		}
	}
	age, aged := attestationAge(g, p.Attestation, o.Now)
	if aged {
		row.AgeDays = strconv.Itoa(int(age))
	}
	row.Stale = aged && age > float64(o.StaleDays)
	row.Status = status(row, p.Attestation != "")
	row.StatusLabel = statusLabel[row.Status]
	row.AuthLabel = authLabel(p.Auth)
	return row
}

func status(row ServerRow, attested bool) string {
	switch {
	case len(row.Critical) > 0:
		return "critical"
	case !attested:
		return "unattested"
	case row.Stale:
		return "stale"
	}
	return "ok"
}

// attestationAge is the age in days of the attestation with digest d, as the
// query field attestation_age_days measures it.
func attestationAge(g *graph.Graph, d string, now time.Time) (float64, bool) {
	if d == "" {
		return 0, false
	}
	a, ok := g.Node(graph.AttestationID(d))
	if !ok || a.Attestation == nil {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339, a.Attestation.RanAt)
	if err != nil {
		return 0, false
	}
	return now.Sub(t).Hours() / 24, true
}

// totals counts the graph's nodes and the servers' attestation state.
func totals(g *graph.Graph, servers []ServerRow) Totals {
	var t Totals
	kinds := map[graph.Kind]*int{graph.KindAgent: &t.Agents, graph.KindServer: &t.Servers,
		graph.KindTool: &t.Tools, graph.KindIdentity: &t.Identities}
	for _, n := range g.Nodes {
		if c, ok := kinds[n.Kind]; ok {
			*c++
		}
	}
	var sum float64
	scored := 0
	for _, s := range servers {
		if s.Status == "unattested" {
			t.Unattested++
		}
		if s.Stale {
			t.Stale++
		}
		if s.Score != "" {
			v, _ := strconv.ParseFloat(s.Score, 64)
			sum += v
			scored++
		}
		t.DestructiveTools += s.Destructive
	}
	t.Scored = scored
	t.Attested = t.Servers - t.Unattested
	if scored > 0 {
		t.AvgScore = strconv.FormatFloat(sum/float64(scored), 'f', 1, 64)
	}
	return t
}
