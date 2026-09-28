// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package policy

import (
	"strings"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-graph/internal/query"
)

const strict = `
version: 1
forbid:
  - name: unattested
    reason: an agent uses a server nobody has attested
    query: agents -> servers where not attested
  - name: stale
    reason: the attestation is older than 30 days
    query: agents -> servers where attestation_age_days > 30
`

// AC: SG-06
// A policy fails on every forbidden path and names each; a graph with none
// passes.
func TestPolicyNamesEveryForbiddenPathAndPassesAClean(t *testing.T) {
	p, err := Parse([]byte(strict))
	if err != nil {
		t.Fatal(err)
	}
	vs, err := p.Check(fixture.Graph(), query.Context{Now: fixture.Now})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range vs {
		got = append(got, v.Rule+": "+v.Path.String())
	}
	want := []string{
		"unattested: claude-desktop -> npx @example/shell-mcp",
		"stale: claude-desktop -> https://mail.example/mcp",
		"stale: cursor -> https://mail.example/mcp",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if vs[0].Reason == "" {
		t.Error("a violation carries its rule's reason")
	}
	clean, err := Parse([]byte("version: 1\nforbid:\n  - name: none\n    query: servers where grade = Z\n"))
	if err != nil {
		t.Fatal(err)
	}
	if vs, _ := clean.Check(fixture.Graph(), query.Context{Now: fixture.Now}); len(vs) != 0 {
		t.Fatalf("want a pass, got %v", vs)
	}
	// Thirty days later every attestation is stale.
	later, _ := p.Check(fixture.Graph(), query.Context{Now: fixture.Now.AddDate(0, 0, 60)})
	if len(later) != 1+4 {
		t.Fatalf("want every attested use stale after 60 days, got %d", len(later))
	}
}

func TestPolicyRefusesWhatItCannotCheck(t *testing.T) {
	for name, doc := range map[string]string{
		"unknown key": "version: 1\nforbid:\n  - name: a\n    query: agents\n    severity: high\n",
		"version":     "version: 2\nforbid:\n  - name: a\n    query: agents\n",
		"empty":       "version: 1\n",
		"unnamed":     "version: 1\nforbid:\n  - query: agents\n",
		"duplicate":   "version: 1\nforbid:\n  - name: a\n    query: agents\n  - name: a\n    query: tools\n",
		"bad query":   "version: 1\nforbid:\n  - name: a\n    query: widgets\n",
		"not yaml":    "version: [\n",
		"nothing":     "",
	} {
		if _, err := Parse([]byte(doc)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	p, err := Parse([]byte("version: 1\nforbid:\n  - name: a\n    query: agents where destructive\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Check(fixture.Graph(), query.Context{}); err == nil {
		t.Error("a rule whose field does not resolve must be an error at check time")
	}
}
