// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"strconv"
	"strings"
)

// Ratio is a count out of a denominator, so a percentage on the page always
// travels with what it is a percentage of.
type Ratio struct {
	Num int `json:"numerator"`
	Den int `json:"denominator"`
}

// Pct is the ratio as a percentage to one decimal, without a trailing
// ".0": 4/6 is "66.7%", 3/5 is "60%", and a zero denominator is "n/a".
func (r Ratio) Pct() string {
	if r.Den == 0 {
		return "n/a"
	}
	s := strconv.FormatFloat(float64(r.Num)*100/float64(r.Den), 'f', 1, 64)
	return strings.TrimSuffix(s, ".0") + "%"
}

// Value is the percentage as a number for drawing, 0 when undefined.
func (r Ratio) Value() float64 {
	if r.Den == 0 {
		return 0
	}
	return float64(r.Num) * 100 / float64(r.Den)
}

// Metric is one figure the page shows, with what it counts, so the HTML,
// the JSON summary and the documentation cannot disagree about a unit.
type Metric struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Unit    string `json:"unit"`
	Value   int    `json:"value"`
	Of      *int   `json:"of,omitempty"`
	Display string `json:"display"`
	// Counts says what one unit is and what makes two of them distinct.
	Counts string `json:"counts"`
}

func orNA(s string) string {
	if s == "" {
		return "n/a"
	}
	return s
}

// metrics is the registry of the page's headline figures.
func metrics(s *Summary) []Metric {
	t, h := s.Totals, s.Hero
	of := func(n int) *int { return &n }
	return []Metric{
		{ID: "clients", Label: "Configured clients", Unit: "client", Value: t.Agents,
			Display: strconv.Itoa(t.Agents), Counts: "agent nodes read from MCP client configurations; not running processes"},
		{ID: "servers", Label: "MCP servers", Unit: "server", Value: t.Servers,
			Display: strconv.Itoa(t.Servers), Counts: "servers, identified by passmcp's attestation subject (transport and endpoint)"},
		{ID: "tools", Label: "Tools", Unit: "tool", Value: t.Tools,
			Display: strconv.Itoa(t.Tools), Counts: "tools, distinct per server: the same name on two servers is two tools"},
		{ID: "paths", Label: "Configured tool paths", Unit: "path", Value: h.Paths,
			Display: strconv.Itoa(h.Paths), Counts: "client -> server -> tool paths derived from configuration; not observed calls"},
		{ID: "paths.changing", Label: "Paths to tools not declared read-only", Unit: "path", Value: h.Changing.Num, Of: of(h.Changing.Den),
			Display: h.Changing.Pct(), Counts: "paths whose tool lacks readOnlyHint true, including unannotated tools"},
		{ID: "paths.destructive", Label: "Paths to potentially destructive tools", Unit: "path", Value: h.Destructive.Num, Of: of(h.Destructive.Den),
			Display: h.Destructive.Pct(), Counts: "paths whose tool declares destructiveHint true, or is not declared read-only and declares no destructiveHint (the MCP default)"},
		{ID: "paths.destructive.declared", Label: "Paths to tools declared destructive", Unit: "path", Value: h.DeclaredDestructive, Of: of(h.Paths),
			Display: Ratio{h.DeclaredDestructive, h.Paths}.Pct(), Counts: "paths whose tool declares destructiveHint true"},
		{ID: "paths.unannotated", Label: "Paths to unannotated tools", Unit: "path", Value: h.Unannotated.Num, Of: of(h.Unannotated.Den),
			Display: h.Unannotated.Pct(), Counts: "paths whose tool declares no readOnlyHint: behaviour unknown"},
		{ID: "servers.evidence", Label: "Servers with evidence", Unit: "server", Value: t.Attested, Of: of(t.Servers),
			Display: Ratio{t.Attested, t.Servers}.Pct(), Counts: "servers with at least one attestation, whatever its age"},
		{ID: "servers.fresh", Label: "Servers with fresh evidence", Unit: "server", Value: t.Attested - t.Stale, Of: of(t.Servers),
			Display: Ratio{t.Attested - t.Stale, t.Servers}.Pct(), Counts: "servers whose latest attestation is no older than the stale threshold"},
		{ID: "score.mean", Label: "Mean score of scored servers", Unit: "score", Value: t.Scored, Of: of(t.Servers),
			Display: orNA(t.AvgScore), Counts: "mean of the latest score of each scored server; value is how many were scored"},
	}
}
