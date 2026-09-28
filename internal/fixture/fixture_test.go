// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package fixture

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"satellion.com/passmcp-reporting/attestation"
	"satellion.com/passmcp-reporting/graph"
)

var update = flag.Bool("update", false, "rewrite testdata/fixture/graph.json from the builder")

// published is the fixture file the documentation and the query tests use.
var published = filepath.Join("..", "..", "testdata", "fixture", "graph.json")

// The published fixture graph is exactly what the builder makes, so a change
// to either shows up as a failing test rather than as documentation that no
// longer matches.
func TestThePublishedFixtureIsCurrent(t *testing.T) {
	want, err := Graph().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.WriteFile(published, want, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(published)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("testdata/fixture/graph.json is stale; run go test ./internal/fixture -update")
	}
	if err := Graph().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStatementsAreValid(t *testing.T) {
	raw := Statement("http", CRM, Now, 50, "D", "none",
		Verdict{"a", "pass", ""}, Verdict{"b", "warn", "minor"}, Verdict{"c", "fail", "critical"},
		Verdict{"d", "skip", ""}, Verdict{"e", "info", ""})
	if _, err := attestation.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if len(Report("http", CRM, Auth{Issuer: Issuer, ClientID: "c", Scope: "x"}, Tool{Name: "t"})) == 0 {
		t.Fatal("empty report")
	}
}

func TestLabel(t *testing.T) {
	g := Graph()
	if Label(g, graph.ServerID("http", CRM)) != CRM || Label(g, "nope") != "<nope>" {
		t.Fatal("label")
	}
	if Now.Location() != time.UTC {
		t.Fatal("fixture time must be UTC")
	}
}
