// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import "fmt"

// The page's attention items are two collections, counted apart so every
// count names its unit: findings (an inherited critical check, a policy
// violation, an over-privileged identity) and evidence gaps (a server
// with no attestation, or a stale one).

// Gap is one server whose evidence is missing or out of date.
type Gap struct {
	Anchor  string `json:"anchor"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Server  Ref    `json:"server"`
	Clients []Ref  `json:"clients"`
	Next    string `json:"next"`
	Tags    string `json:"-"`
	Text    string `json:"-"`
}

// Issue is the attention item the page puts first, with what is needed to
// inspect it: the panel holding its evidence, the client whose path it
// is, and a command that re-reads the local graph.
type Issue struct {
	// Severity is "critical" or "warning".
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Context  string `json:"context"`
	// Rule is the check ID or policy rule, kept under the readable title.
	Rule string `json:"rule,omitempty"`
	// Evidence is the anchor of the panel with the item's evidence; Client
	// the anchor of the client whose path it is, when there is one.
	Evidence string `json:"evidence"`
	Client   string `json:"client,omitempty"`
	Command  string `json:"command"`
	// Findings and Gaps count every attention item on the page.
	Findings int `json:"findings"`
	Gaps     int `json:"evidenceGaps"`
}

// CommandNote says what running the command does and does not do.
const CommandNote = "Re-reads the local graph; collects no new evidence."

func buildGaps(s *Summary) []Gap {
	out := []Gap{}
	for _, sv := range s.Servers {
		g := Gap{Server: Ref{sv.Endpoint, sv.Anchor}, Clients: sv.Clients}
		switch {
		case sv.Status == "unattested":
			g.Kind, g.Title = "no-evidence", "No attestation loaded for "+sv.Endpoint
			g.Detail = fmt.Sprintf("%s configured to reach it; no passmcp attestation for it was read.", clientCount(len(sv.Clients)))
			g.Next = `passmcp-graph query "agents -> servers where not attested"`
		case sv.Stale:
			g.Kind, g.Title = "stale-evidence", fmt.Sprintf("Evidence for %s is %s days old", sv.Endpoint, sv.AgeDays)
			g.Detail = fmt.Sprintf("Its latest attestation is older than the %d-day threshold. %s configured to reach it.", s.StaleDays, clientCount(len(sv.Clients)))
			g.Next = fmt.Sprintf(`passmcp-graph query "agents -> servers where attestation_age_days > %d"`, s.StaleDays)
		default:
			continue
		}
		g.Anchor = fmt.Sprintf("gap-%d", len(out))
		out = append(out, g)
	}
	return out
}

func clientCount(n int) string {
	if n == 1 {
		return "1 client is"
	}
	return fmt.Sprintf("%d clients are", n)
}

// relate counts, per server, the findings that name it.
func relate(s *Summary) {
	n := map[string]int{}
	for _, f := range s.Findings {
		n[f.Server.Anchor]++
	}
	for i := range s.Servers {
		s.Servers[i].RelatedFindings = n[s.Servers[i].Anchor]
	}
}

// topIssue is the first finding, in the order critical, policy, identity,
// or else the first evidence gap. Nil means the evidence shows neither,
// which is not a claim that the estate is safe.
func topIssue(s *Summary) *Issue {
	var i Issue
	switch {
	case len(s.Findings) > 0:
		i = findingIssue(s, s.Findings[0])
	case len(s.Gaps) > 0:
		g := s.Gaps[0]
		i = Issue{Severity: "warning", Title: g.Title, Context: g.Detail, Evidence: g.Server.Anchor, Command: g.Next}
		if len(g.Clients) > 0 {
			i.Client = g.Clients[0].Anchor
		}
	default:
		return nil
	}
	i.Findings, i.Gaps = len(s.Findings), len(s.Gaps)
	return &i
}

func findingIssue(s *Summary, f Finding) Issue {
	i := Issue{Severity: f.Severity, Title: f.Title, Context: f.Evidence, Rule: f.Rule, Evidence: f.Anchor, Command: f.Next}
	if len(f.Clients) > 0 {
		i.Client = f.Clients[0].Anchor
	}
	if f.Kind != "inherited-critical" || len(f.Clients) == 0 {
		return i
	}
	i.Title = f.Server.Name + " failed a critical check in its latest attestation"
	for _, sv := range s.Servers {
		if sv.Anchor == f.Server.Anchor {
			i.Context = fmt.Sprintf("%s is configured to reach it. Evidence age: %s days; stale after %d.",
				f.Clients[0].Name, orDefault(sv.AgeDays, "unknown"), s.StaleDays)
		}
	}
	return i
}
