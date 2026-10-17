// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"satellion.com/passmcp-reporting/graph"
)

// The command centre is the page's first screen: the attention items in
// one list by severity, a mitigation path for the item to look at first,
// and the share-safe export the page can download. Everything here is
// derived from the summary; nothing is invented to fill a panel.

// AttentionRow is one finding or evidence gap in the command centre list.
type AttentionRow struct {
	Anchor string
	Title  string
	Path   string
	// Severity is "critical", "medium" or "low": critical and policy
	// findings, identity findings, and evidence gaps.
	Severity string
	Age      string
	Tags     string
}

// Mitigation is the four-step path for the item the page puts first.
type Mitigation struct {
	Review, Fix, Verify, Document string
}

func attentionRows(s *Summary) []AttentionRow {
	ages, names := map[string]string{}, map[string]string{}
	for _, sv := range s.Servers {
		ages[sv.Anchor], names[sv.Anchor] = ageText(sv), displayName(sv)
	}
	rows := []AttentionRow{}
	for _, f := range s.Findings {
		sev := "critical"
		if f.Severity != "critical" {
			sev = "medium"
		}
		rows = append(rows, AttentionRow{Anchor: f.Anchor, Title: shortTitle(f, names[f.Server.Anchor]),
			Path: refPath(f.Clients, Ref{names[f.Server.Anchor], f.Server.Anchor}), Severity: sev, Age: ages[f.Server.Anchor], Tags: f.Tags})
	}
	for _, g := range s.Gaps {
		title := "No attestation for " + names[g.Server.Anchor]
		if g.Kind == "stale-evidence" {
			title = "Stale evidence for " + names[g.Server.Anchor]
		}
		rows = append(rows, AttentionRow{Anchor: g.Anchor, Title: title, Path: refPath(g.Clients, Ref{names[g.Server.Anchor], g.Server.Anchor}),
			Severity: "low", Age: ages[g.Server.Anchor], Tags: g.Tags})
	}
	return rows
}

// shortTitle names a finding in a few words for the command-centre list;
// its details panel keeps the full title.
func shortTitle(f Finding, server string) string {
	switch f.Kind {
	case "inherited-critical":
		return server + " failed a critical check"
	case "policy":
		return "Policy rule " + f.Rule + " forbids a path"
	case "over-privileged":
		return "Over-privileged identity on " + server
	}
	return f.Title
}

func ageText(sv ServerRow) string {
	switch {
	case sv.Attestation == "":
		return "Not supplied"
	case sv.AgeDays == "":
		return "Unknown"
	case sv.AgeDays == "1":
		return "1 day"
	}
	return sv.AgeDays + " days"
}

// refPath is "client -> server" for the first client, or the server alone.
func refPath(clients []Ref, server Ref) string {
	if len(clients) == 0 {
		return server.Name
	}
	if len(clients) == 1 {
		return clients[0].Name + " → " + server.Name
	}
	return fmt.Sprintf("%s and %d more → %s", clients[0].Name, len(clients)-1, server.Name)
}

// mitigation fills the four steps for the first attention item. The fix is
// what passmcp-graph can say from the evidence it holds; it does not claim
// to know the server's internals.
func mitigation(s *Summary) Mitigation {
	m := Mitigation{
		Review:   "Open the evidence for the item and confirm the finding against the supplied attestation.",
		Fix:      "Change the configuration, policy or server so the item no longer applies.",
		Verify:   "Run passmcp against the server again, ingest the new attestation, and check the item has gone.",
		Document: "Save the PDF report or export the share-safe JSON with your notes.",
	}
	switch {
	case len(s.Findings) > 0:
		f := s.Findings[0]
		m.Review = fmt.Sprintf("Open the evidence for %q and confirm it against the supplied attestation.", f.Rule)
		m.Fix = map[string]string{
			"inherited-critical": "Fix the failing check on " + f.Server.Name + ", or remove the server from the clients' configuration until it is fixed.",
			"policy":             "Remove the forbidden path from the client configuration, or change the policy if the path is intended.",
			"over-privileged":    "Reduce the identity's scopes on " + f.Server.Name + " to what its tools need.",
		}[f.Kind]
		m.Verify = "Run passmcp against " + f.Server.Name + " again, ingest the new attestation, then run " + f.Next + "."
	case len(s.Gaps) > 0:
		g := s.Gaps[0]
		m.Review = "Check which clients are configured to reach " + g.Server.Name + "."
		m.Fix = "Run passmcp against " + g.Server.Name + " and ingest the attestation it writes."
		m.Verify = "Run " + g.Next + " and check the server is no longer listed."
	}
	return m
}

// ScoreBar is one scored server in the mean-score card's bars.
type ScoreBar struct {
	Name  string
	Score string
	Pct   int
}

// Rest is the part of the bar's height above it, where the bar starts
// when drawn from the top as an SVG rect.
func (b ScoreBar) Rest() int { return 100 - b.Pct }

func scoreBars(s *Summary) []ScoreBar {
	var out []ScoreBar
	for _, sv := range s.Servers {
		if sv.Score == "" {
			continue
		}
		v, _ := strconv.ParseFloat(sv.Score, 64)
		out = append(out, ScoreBar{Name: displayName(sv), Score: sv.Score, Pct: int(min(100, max(4, v)))})
	}
	return out
}

// displayName is the name a server reports, or its endpoint.
func displayName(sv ServerRow) string {
	if sv.Name != "" {
		return sv.Name
	}
	return sv.Endpoint
}

// serverIcon picks a glyph from what is known about a server: a command
// line is a terminal; otherwise its reported name hints at mail or files.
func serverIcon(sv ServerRow) string {
	n := strings.ToLower(sv.Name + " " + sv.Endpoint)
	switch {
	case sv.Transport == "stdio":
		return "term"
	case strings.Contains(n, "mail"):
		return "mail"
	case strings.Contains(n, "file") || strings.Contains(n, "drive"):
		return "folder"
	}
	return "db"
}

// clientIcon picks a glyph and an accent for a client product. They are
// generic marks, not the products' logos.
func clientIcon(client string) string {
	c := strings.ToLower(client)
	switch {
	case strings.Contains(c, "claude"):
		return "spark"
	case strings.Contains(c, "cursor"):
		return "cube"
	case strings.Contains(c, "vscode") || strings.Contains(c, "vs-code") || strings.Contains(c, "code"):
		return "brackets"
	}
	return "code"
}

// exportJSON is the share-safe summary the page offers as a download: the
// same summary --share-safe --format json prints.
func exportJSON(g *graph.Graph, o Options) (string, error) {
	o.ShareSafe = true
	o.skipExport = true
	s, err := Build(g, o)
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	return string(b), err
}
