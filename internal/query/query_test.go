// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-reporting/graph"
)

func loadPublished(t *testing.T) *graph.Graph {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixture", "graph.json"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := graph.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func run(t *testing.T, g *graph.Graph, q string) []string {
	t.Helper()
	parsed, err := Parse(q)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	paths, err := Eval(g, parsed, Context{Now: fixture.Now})
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	out := []string{}
	for _, p := range paths {
		out = append(out, p.String())
	}
	return out
}

// AC: SG-05
// The issue's own query, and every query documented against the published
// fixture, return exactly the expected paths.
func TestQueriesOverThePublishedFixtureReturnExactlyTheExpectedPaths(t *testing.T) {
	g := loadPublished(t)
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixture", "queries.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Query string   `json:"query"`
		Paths []string `json:"paths"`
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 || cases[0].Query != "agents -> tools where destructive and server.auth = none" {
		t.Fatal("the issue's query must lead the published examples")
	}
	for _, c := range cases {
		if got := run(t, g, c.Query); !reflect.DeepEqual(got, c.Paths) {
			t.Errorf("%s\n got %q\nwant %q", c.Query, got, c.Paths)
		}
	}
}

func TestComparisonsAndTruthiness(t *testing.T) {
	g := loadPublished(t)
	cases := map[string]int{
		"tools where readonly":                                      2,
		"tools where readonly = false":                              3,
		"tools where readonly != true":                              3,
		"tools where not annotated":                                 2,
		"tools where name contains record":                          2,
		"servers where transport = stdio":                           1,
		"servers where score":                                       3,
		"servers where score < 50":                                  1,
		"servers where score <= 41.5":                               1,
		"servers where score > 90":                                  1,
		"servers where score = 88":                                  1,
		"servers where score != 88":                                 3,
		"servers where failing contains auth.unauthenticated_tools": 1,
		"servers where failing = auth.unauthenticated_tools":        1,
		"servers where failing != auth.unauthenticated_tools":       3,
		"servers where name = crm":                                  1,
		"servers where name != crm":                                 3,
		"servers where endpoint contains example":                   4,
		"servers where version":                                     0,
		"servers where auth":                                        3,
		"servers where attestation_age_days < 3":                    2,
		"agents where client = cursor":                              1,
		"agents where config contains .cursor":                      1,
		"agents where name = cursor":                                1,
		"identities where issuer = 'https://auth.example'":          2,
		"identities where scopes contains 'mail:read'":              1,
		"attestations where age_days > 30":                          1,
		"attestations where digest":                                 3,
		"attestations where ran_at contains 2026":                   3,
		"attestations -> servers where grade = B":                   1,
		"sources":            0,
		"servers -> sources": 0,
		"tools -> servers -> agents where client = cursor":        2,
		"servers where (grade = A or grade = B) and not critical": 2,
		"servers where score > abc":                               0,
		"tools where readonly = maybe":                            0,
		"servers where failing < 3":                               0,
		"servers where name < z":                                  0,
	}
	for q, want := range cases {
		if got := run(t, g, q); len(got) != want {
			t.Errorf("%s: got %d paths %q, want %d", q, len(got), got, want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, q := range []string{
		"", "widgets", "agents ->", "agents where", "agents where (client = x", "agents where client =",
		"agents where = x", "agents where client = x extra", "agents flub", "agents where client = 'x",
		"agents where nope.client = x", "agents where )",
	} {
		if _, err := Parse(q); err == nil {
			t.Errorf("%q: want a parse error", q)
		}
	}
}

func TestEvalRefusesFieldsTheyCannotResolve(t *testing.T) {
	g := fixture.Graph()
	for _, q := range []string{
		"agents where destructive",              // no tool on the path
		"agents -> servers where tool.readonly", // qualified, not on the path
		"tools where colour",                    // no such field
		"agents -> servers where name",          // ambiguous: agent.name and server.name
		"servers where server.colour",           // qualified unknown field
		"agents -> sources where x",
		"tools -> identities -> agents where client = x and nope",
		"attestations -> sources or x",
	} {
		parsed, err := Parse(q)
		if err != nil {
			continue
		}
		if _, err := Eval(g, parsed, Context{Now: time.Now()}); err == nil {
			t.Errorf("%q: want an evaluation error", q)
		}
	}
	parsed, _ := Parse("servers -> servers")
	if _, err := Eval(g, parsed, Context{}); err == nil || !strings.Contains(err.Error(), "no relationship") {
		t.Errorf("servers -> servers: %v", err)
	}
}

func TestAbsentValuesAndAges(t *testing.T) {
	g := graph.New()
	id := graph.ServerID("http", "https://x")
	g.Upsert(graph.Node{ID: id, Kind: graph.KindServer, Label: "x", Server: &graph.ServerProps{Transport: "http", Endpoint: "https://x", Attestation: "dead"}})
	g.Upsert(graph.Node{ID: graph.AttestationID("bad"), Kind: graph.KindAttestation, Label: "a", Attestation: &graph.AttestationProps{Digest: "bad", PredicateType: "p", RanAt: "not a time"}})
	for q, want := range map[string]int{
		"servers where attestation_age_days > 0": 0, // the attestation node is missing
		"servers where score != 1":               1, // absent differs from everything
		"servers where score = 1":                0,
		"attestations where age_days > 0":        0, // unparsable time is absent
	} {
		if got := run(t, g, q); len(got) != want {
			t.Errorf("%s: got %q", q, got)
		}
	}
	// A node without its property struct reads as empty, never panics.
	bare := graph.New()
	for _, k := range []graph.Kind{graph.KindAgent, graph.KindServer, graph.KindTool, graph.KindIdentity, graph.KindSource, graph.KindAttestation} {
		bare.Nodes = append(bare.Nodes, graph.Node{ID: string(k) + ":1", Kind: k, Label: string(k)})
	}
	for k, fs := range Fields() {
		for _, f := range fs {
			q := k + "s where " + k + "." + f
			if k == "identity" {
				q = "identities where identity." + f
			}
			_ = run(t, bare, q)
		}
	}
}
