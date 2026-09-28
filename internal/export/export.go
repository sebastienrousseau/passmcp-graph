// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package export writes a graph for other tools: JSON in the published graph
// format, GraphML for graph editors, and Cypher for property-graph databases.
package export

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"satellion.com/passmcp-graph/internal/schema"
	"satellion.com/passmcp-reporting/graph"
	"satellion.com/passmcp-reporting/spec"
)

// Formats lists the export formats.
var Formats = []string{"json", "graphml", "cypher"}

// Write exports g in format to w.
func Write(w io.Writer, g *graph.Graph, format string) error {
	switch format {
	case "json":
		b, err := g.Marshal()
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	case "graphml":
		return graphML(w, g)
	case "cypher":
		return cypher(w, g)
	}
	return fmt.Errorf("export: unknown format %q (want %s)", format, strings.Join(Formats, ", "))
}

// Validate checks JSON against the published graph schema and the model's own
// rules, returning every problem.
func Validate(b []byte) ([]string, error) {
	v, err := schema.New(spec.GraphSchema)
	if err != nil {
		return nil, err
	}
	problems, err := v.Validate(b)
	if err != nil {
		return nil, err
	}
	if g, err := graph.Parse(b); err != nil {
		problems = append(problems, err.Error())
	} else if err := g.Validate(); err != nil {
		problems = append(problems, err.Error())
	}
	return problems, nil
}

// Import reads JSON in the published graph format.
func Import(b []byte) (*graph.Graph, error) {
	problems, err := Validate(b)
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("import: %s", strings.Join(problems, "; "))
	}
	return graph.Parse(b)
}

// attrs flattens a node's properties to string key/values, for the formats
// that have no nesting.
func attrs(n graph.Node) map[string]string {
	a := map[string]string{"kind": string(n.Kind), "label": n.Label}
	switch {
	case n.Agent != nil:
		a["client"], a["configFile"] = n.Agent.Client, n.Agent.ConfigFile
	case n.Server != nil:
		serverAttrs(a, n.Server)
	case n.Tool != nil:
		a["name"] = n.Tool.Name
		setBool(a, "readOnlyHint", n.Tool.ReadOnlyHint)
		setBool(a, "destructiveHint", n.Tool.DestructiveHint)
	case n.Identity != nil:
		a["issuer"], a["clientId"] = n.Identity.Issuer, n.Identity.ClientID
	case n.Source != nil:
		a["type"], a["ref"] = n.Source.Type, n.Source.Ref
	case n.Attestation != nil:
		a["digest"], a["predicateType"], a["ranAt"] = n.Attestation.Digest, n.Attestation.PredicateType, n.Attestation.RanAt
	}
	for k, v := range a {
		if v == "" {
			delete(a, k)
		}
	}
	return a
}

func serverAttrs(a map[string]string, s *graph.ServerProps) {
	a["transport"], a["endpoint"], a["name"], a["version"] = s.Transport, s.Endpoint, s.Name, s.Version
	a["auth"], a["grade"], a["attestation"] = s.Auth, s.Grade, s.Attestation
	if s.Score != nil {
		a["score"] = strconv.FormatFloat(*s.Score, 'f', -1, 64)
	}
	ids := make([]string, len(s.Failing))
	for i, f := range s.Failing {
		ids[i] = f.ID
	}
	a["failing"] = strings.Join(ids, ",")
}

func setBool(a map[string]string, k string, b *bool) {
	if b != nil {
		a[k] = strconv.FormatBool(*b)
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// graphML writes GraphML: one key per attribute name, then the nodes and the
// edges.
func graphML(w io.Writer, g *graph.Graph) error {
	g.Canonical()
	keys := map[string]bool{}
	nodeAttrs := make([]map[string]string, len(g.Nodes))
	for i, n := range g.Nodes {
		nodeAttrs[i] = attrs(n)
		for k := range nodeAttrs[i] {
			keys[k] = true
		}
	}
	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString(`<graphml xmlns="http://graphml.graphdrawing.org/xmlns">` + "\n")
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		fmt.Fprintf(&b, "  <key id=%q for=\"node\" attr.name=%q attr.type=\"string\"/>\n", k, k)
	}
	b.WriteString("  <key id=\"edgekind\" for=\"edge\" attr.name=\"kind\" attr.type=\"string\"/>\n")
	b.WriteString("  <key id=\"scopes\" for=\"edge\" attr.name=\"scopes\" attr.type=\"string\"/>\n")
	b.WriteString("  <graph id=\"passmcp\" edgedefault=\"directed\">\n")
	for i, n := range g.Nodes {
		fmt.Fprintf(&b, "    <node id=%q>\n", n.ID)
		for _, k := range sortedKeys(nodeAttrs[i]) {
			fmt.Fprintf(&b, "      <data key=%q>%s</data>\n", k, escape(nodeAttrs[i][k]))
		}
		b.WriteString("    </node>\n")
	}
	for _, e := range g.Edges {
		fmt.Fprintf(&b, "    <edge source=%q target=%q>\n      <data key=\"edgekind\">%s</data>\n", e.From, e.To, e.Kind)
		if len(e.Scopes) > 0 {
			fmt.Fprintf(&b, "      <data key=\"scopes\">%s</data>\n", escape(strings.Join(e.Scopes, " ")))
		}
		b.WriteString("    </edge>\n")
	}
	b.WriteString("  </graph>\n</graphml>\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func escape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// cypher writes one MERGE per node and edge, labelled by kind, so a script
// run twice creates nothing new.
func cypher(w io.Writer, g *graph.Graph) error {
	g.Canonical()
	var b strings.Builder
	for _, n := range g.Nodes {
		a := attrs(n)
		fmt.Fprintf(&b, "MERGE (n:%s {id: %s})", label(string(n.Kind)), quote(n.ID))
		if keys := sortedKeys(a); len(keys) > 0 {
			sets := make([]string, len(keys))
			for i, k := range keys {
				sets[i] = "n." + k + " = " + quote(a[k])
			}
			b.WriteString(" SET " + strings.Join(sets, ", "))
		}
		b.WriteString(";\n")
	}
	for _, e := range g.Edges {
		fmt.Fprintf(&b, "MATCH (a {id: %s}), (b {id: %s}) MERGE (a)-[r:%s]->(b)", quote(e.From), quote(e.To), strings.ToUpper(string(e.Kind)))
		if len(e.Scopes) > 0 {
			fmt.Fprintf(&b, " SET r.scopes = %s", quote(strings.Join(e.Scopes, " ")))
		}
		b.WriteString(";\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// label capitalises a kind for a Cypher node label.
func label(k string) string {
	if k == "" {
		return "Node"
	}
	return strings.ToUpper(k[:1]) + k[1:]
}

// quote writes a Cypher string literal.
func quote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`, "\n", `\n`, "\r", `\r`).Replace(s) + "'"
}
