// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package export

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-reporting/graph"
)

// AC: SG-08
// JSON export validates against the published, versioned schema and
// re-imports without loss; GraphML and Cypher export the same nodes and
// edges for existing graph tools.
func TestExportsValidateAndReimportWithoutLoss(t *testing.T) {
	g := fixture.Graph()
	var j bytes.Buffer
	if err := Write(&j, g, "json"); err != nil {
		t.Fatal(err)
	}
	problems, err := Validate(j.Bytes())
	if err != nil || len(problems) != 0 {
		t.Fatalf("schema problems %v %v", problems, err)
	}
	if !strings.Contains(j.String(), graph.Version) {
		t.Fatal("the export does not name its format version")
	}
	back, err := Import(j.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	again, _ := back.Marshal()
	if !bytes.Equal(again, j.Bytes()) {
		t.Fatal("re-import lost or changed something")
	}

	var ml bytes.Buffer
	if err := Write(&ml, g, "graphml"); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Graph struct {
			Nodes []struct {
				ID   string `xml:"id,attr"`
				Data []struct {
					Key   string `xml:"key,attr"`
					Value string `xml:",chardata"`
				} `xml:"data"`
			} `xml:"node"`
			Edges []struct {
				Source string `xml:"source,attr"`
			} `xml:"edge"`
		} `xml:"graph"`
	}
	if err := xml.Unmarshal(ml.Bytes(), &doc); err != nil {
		t.Fatalf("GraphML is not XML: %v", err)
	}
	if len(doc.Graph.Nodes) != len(g.Nodes) || len(doc.Graph.Edges) != len(g.Edges) {
		t.Fatalf("GraphML has %d nodes and %d edges, want %d and %d", len(doc.Graph.Nodes), len(doc.Graph.Edges), len(g.Nodes), len(g.Edges))
	}

	var cy bytes.Buffer
	if err := Write(&cy, g, "cypher"); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(cy.String(), "\nMERGE (n:") + boolInt(strings.HasPrefix(cy.String(), "MERGE (n:")); n != len(g.Nodes) {
		t.Fatalf("Cypher merges %d nodes, want %d", n, len(g.Nodes))
	}
	if n := strings.Count(cy.String(), "MERGE (a)-["); n != len(g.Edges) {
		t.Fatalf("Cypher merges %d edges, want %d", n, len(g.Edges))
	}
	for _, want := range []string{":Server", ":Tool", ":Agent", ":Identity", ":Attestation", "[r:USES]", "SET r.scopes"} {
		if !strings.Contains(cy.String(), want) {
			t.Errorf("Cypher lacks %s", want)
		}
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestExportEscapesAndRefuses(t *testing.T) {
	if err := Write(&bytes.Buffer{}, fixture.Graph(), "dot"); err == nil {
		t.Fatal("an unknown format must be refused")
	}
	if got := quote("it's a \\ path\nnext\r"); got != `'it\'s a \\ path\nnext\r'` {
		t.Errorf("quote %s", got)
	}
	if escape("<a&b>") != "&lt;a&amp;b&gt;" || label("") != "Node" {
		t.Error("escape or label")
	}
	for _, bad := range []string{`{`, `{"version":"x","nodes":[],"edges":[]}`, `{"version":"https://satellion.com/graph/v1","nodes":[{"id":"a","kind":"server","label":"a"}],"edges":[]}`} {
		if _, err := Import([]byte(bad)); err == nil {
			t.Errorf("%s: want an import error", bad)
		}
	}
	src := graph.New()
	src.Upsert(graph.Node{ID: graph.SourceID("targets", "t.txt"), Kind: graph.KindSource, Label: "t.txt", Source: &graph.SourceProps{Type: "targets", Ref: "t.txt"}})
	var ml bytes.Buffer
	if err := Write(&ml, src, "graphml"); err != nil || !strings.Contains(ml.String(), "targets") {
		t.Fatalf("source attributes missing: %v", err)
	}
}
