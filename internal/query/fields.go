// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package query

import (
	"strconv"
	"strings"
	"time"

	"satellion.com/passmcp-reporting/graph"
)

// Context is what fields need beyond the graph.
type Context struct {
	// Now is the time attestation ages are measured against.
	Now time.Time
}

// value is a field's value: a string, a number, a boolean or a list, or
// absent.
type value struct {
	s       string
	n       float64
	b       bool
	list    []string
	kind    byte // 's', 'n', 'b', 'l'
	present bool
}

func str(s string) value    { return value{s: s, kind: 's', present: s != ""} }
func num(n float64) value   { return value{n: n, kind: 'n', present: true} }
func boolean(b bool) value  { return value{b: b, kind: 'b', present: true} }
func list(l []string) value { return value{list: l, kind: 'l', present: len(l) > 0} }

// fields lists every field a query can name, by kind.
var fields = map[string]map[string]func(b *binding, n graph.Node) value{
	"agent": {
		"name":   func(_ *binding, n graph.Node) value { return str(n.Label) },
		"client": func(_ *binding, n graph.Node) value { return str(agentProps(n).Client) },
		"config": func(_ *binding, n graph.Node) value { return str(agentProps(n).ConfigFile) },
	},
	"server": {
		"endpoint":             func(_ *binding, n graph.Node) value { return str(serverProps(n).Endpoint) },
		"name":                 func(_ *binding, n graph.Node) value { return str(serverProps(n).Name) },
		"version":              func(_ *binding, n graph.Node) value { return str(serverProps(n).Version) },
		"transport":            func(_ *binding, n graph.Node) value { return str(serverProps(n).Transport) },
		"auth":                 func(_ *binding, n graph.Node) value { return str(serverProps(n).Auth) },
		"grade":                func(_ *binding, n graph.Node) value { return str(serverProps(n).Grade) },
		"score":                serverScore,
		"attested":             func(_ *binding, n graph.Node) value { return boolean(serverProps(n).Attestation != "") },
		"attestation_age_days": attestationAge,
		"failing":              serverFailing,
		"critical":             serverCritical,
	},
	"tool": {
		"name":        func(_ *binding, n graph.Node) value { return str(toolProps(n).Name) },
		"readonly":    func(_ *binding, n graph.Node) value { return boolean(readOnly(toolProps(n))) },
		"destructive": func(_ *binding, n graph.Node) value { return boolean(destructive(toolProps(n))) },
		"annotated":   func(_ *binding, n graph.Node) value { return boolean(toolProps(n).ReadOnlyHint != nil) },
	},
	"identity": {
		"issuer": func(_ *binding, n graph.Node) value { return str(identityProps(n).Issuer) },
		"client": func(_ *binding, n graph.Node) value { return str(identityProps(n).ClientID) },
		"scopes": identityScopes,
	},
	"source": {
		"type": func(_ *binding, n graph.Node) value { return str(sourceProps(n).Type) },
		"ref":  func(_ *binding, n graph.Node) value { return str(sourceProps(n).Ref) },
	},
	"attestation": {
		"digest":   func(_ *binding, n graph.Node) value { return str(attProps(n).Digest) },
		"ran_at":   func(_ *binding, n graph.Node) value { return str(attProps(n).RanAt) },
		"age_days": func(b *binding, n graph.Node) value { return ageOf(b, attProps(n).RanAt) },
	},
}

func hasField(kind, field string) bool {
	_, ok := fields[kind][field]
	return ok
}

// Fields returns the fields each kind offers, for documentation and errors.
func Fields() map[string][]string {
	out := map[string][]string{}
	for k, fs := range fields {
		for f := range fs {
			out[k] = append(out[k], f)
		}
	}
	return out
}

func agentProps(n graph.Node) graph.AgentProps {
	if n.Agent == nil {
		return graph.AgentProps{}
	}
	return *n.Agent
}

func serverProps(n graph.Node) graph.ServerProps {
	if n.Server == nil {
		return graph.ServerProps{}
	}
	return *n.Server
}

func toolProps(n graph.Node) graph.ToolProps {
	if n.Tool == nil {
		return graph.ToolProps{}
	}
	return *n.Tool
}

func identityProps(n graph.Node) graph.IdentityProps {
	if n.Identity == nil {
		return graph.IdentityProps{}
	}
	return *n.Identity
}

func sourceProps(n graph.Node) graph.SourceProps {
	if n.Source == nil {
		return graph.SourceProps{}
	}
	return *n.Source
}

func attProps(n graph.Node) graph.AttestationProps {
	if n.Attestation == nil {
		return graph.AttestationProps{}
	}
	return *n.Attestation
}

func readOnly(t graph.ToolProps) bool { return t.ReadOnlyHint != nil && *t.ReadOnlyHint }

// destructive follows the MCP specification's defaults: a tool that does not
// declare itself read-only may change things, and one that does not declare
// destructiveHint false may destroy them.
func destructive(t graph.ToolProps) bool {
	if readOnly(t) {
		return false
	}
	return t.DestructiveHint == nil || *t.DestructiveHint
}

func serverScore(_ *binding, n graph.Node) value {
	if s := serverProps(n).Score; s != nil {
		return num(*s)
	}
	return value{kind: 'n'}
}

func serverFailing(_ *binding, n graph.Node) value {
	var ids []string
	for _, f := range serverProps(n).Failing {
		ids = append(ids, f.ID)
	}
	return list(ids)
}

func serverCritical(_ *binding, n graph.Node) value {
	for _, f := range serverProps(n).Failing {
		if strings.EqualFold(f.Severity, "critical") {
			return boolean(true)
		}
	}
	return boolean(false)
}

// attestationAge is the age in days of the server's latest attestation;
// absent when it has none.
func attestationAge(b *binding, n graph.Node) value {
	d := serverProps(n).Attestation
	if d == "" {
		return value{kind: 'n'}
	}
	a, ok := b.g.Node(graph.AttestationID(d))
	if !ok {
		return value{kind: 'n'}
	}
	return ageOf(b, attProps(a).RanAt)
}

func ageOf(b *binding, ranAt string) value {
	t, err := time.Parse(time.RFC3339, ranAt)
	if err != nil {
		return value{kind: 'n'}
	}
	return num(b.ctx.Now.Sub(t).Hours() / 24)
}

// identityScopes are the scopes the identity holds on the path's server, or
// on every server when the path has none.
func identityScopes(b *binding, n graph.Node) value {
	server := b.node("server")
	var scopes []string
	for _, e := range b.g.EdgesFrom(n.ID, graph.Authorizes) {
		if server == nil || e.To == server.ID {
			scopes = append(scopes, e.Scopes...)
		}
	}
	return list(scopes)
}

// binding is one candidate path under evaluation.
type binding struct {
	g     *graph.Graph
	nodes []graph.Node
	kinds []string
	ctx   Context
}

// node returns the path's node of kind k, the last one when there are
// several.
func (b *binding) node(k string) *graph.Node {
	for i := len(b.kinds) - 1; i >= 0; i-- {
		if b.kinds[i] == k {
			return &b.nodes[i]
		}
	}
	return nil
}

func (b *binding) eval(e Expr) bool {
	switch x := e.(type) {
	case And:
		return b.eval(x.L) && b.eval(x.R)
	case Or:
		return b.eval(x.L) || b.eval(x.R)
	case Not:
		return !b.eval(x.X)
	case Cmp:
		return b.compare(x)
	}
	return false
}

func (b *binding) compare(c Cmp) bool {
	kind, _ := resolve(c, b.kinds)
	v := fields[kind][c.Field](b, *b.node(kind))
	if c.Op == "" {
		return truthy(v)
	}
	if !v.present {
		// An absent value equals nothing and differs from everything.
		return c.Op == "!="
	}
	return match(v, c.Op, c.Value)
}

func truthy(v value) bool {
	switch v.kind {
	case 'b':
		return v.b
	case 'n':
		return v.present && v.n != 0
	}
	return v.present
}

func match(v value, op, want string) bool {
	switch v.kind {
	case 'n':
		w, err := strconv.ParseFloat(want, 64)
		return err == nil && numeric(v.n, op, w)
	case 'b':
		w, err := strconv.ParseBool(want)
		return err == nil && ((op == "=") == (v.b == w)) && (op == "=" || op == "!=")
	case 'l':
		return listMatch(v.list, op, want)
	}
	return stringMatch(v.s, op, want)
}

func numeric(a float64, op string, b float64) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	}
	return false
}

func stringMatch(s, op, want string) bool {
	switch op {
	case "=":
		return strings.EqualFold(s, want)
	case "!=":
		return !strings.EqualFold(s, want)
	case "contains":
		return strings.Contains(strings.ToLower(s), strings.ToLower(want))
	}
	return false
}

func listMatch(l []string, op, want string) bool {
	has := false
	for _, s := range l {
		if strings.EqualFold(s, want) {
			has = true
		}
	}
	switch op {
	case "contains", "=":
		return has
	case "!=":
		return !has
	}
	return false
}
