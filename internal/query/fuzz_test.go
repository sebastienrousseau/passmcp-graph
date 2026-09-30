// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package query

import (
	"reflect"
	"strings"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
)

// FuzzParse feeds the query language arbitrary text, which arrives from the
// command line and from policy files. A query either fails with an error or
// parses to a tree that survives being written back out and parsed again;
// evaluating it over the fixture graph never panics.
func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"agents -> tools where destructive and server.auth = none",
		"tools where readonly = false or not (server.grade < \"C\")",
		"servers where name contains 'mail' and attestation.age > 30",
		"identities -> servers where scopes contains \"repo:write\"",
		"agents -> servers -> tools where not not readonly",
		"tools where (a or b) and (c or d)",
		"tools where x != 'it''s'",
		"agents -> ",
		"tools where",
		"tools where a = ",
		"tools where \"unterminated",
		"sources -x-> servers",
		"tools where a-b->c",
		"attestations where ((((grade))))",
	} {
		f.Add(s)
	}
	g := fixture.Graph()
	f.Fuzz(func(t *testing.T, s string) {
		q, err := Parse(s)
		if err != nil {
			return
		}
		if len(q.Kinds) == 0 {
			t.Fatalf("%q parsed with no node kind", s)
		}
		text, ok := format(q)
		if ok {
			again, err := Parse(text)
			if err != nil {
				t.Fatalf("%q parsed, but its rendering %q does not: %v", s, text, err)
			}
			if !reflect.DeepEqual(q, again) {
				t.Fatalf("%q and its rendering %q parse differently:\n%#v\n%#v", s, text, q, again)
			}
		}
		if len(q.Kinds) <= 4 {
			_, _ = Eval(g, q, Context{Now: fixture.Now})
		}
	})
}

// format writes a query back out in a form Parse reads to the same tree,
// with every operand quoted and every connective parenthesised. It reports
// false when an operand holds both quote characters and cannot be written.
func format(q *Query) (string, bool) {
	s := strings.Join(q.Kinds, " -> ")
	if q.Where == nil {
		return s, true
	}
	w, ok := formatExpr(q.Where)
	return s + " where " + w, ok
}

func formatExpr(e Expr) (string, bool) {
	switch e := e.(type) {
	case And:
		return formatBinary(e.L, "and", e.R)
	case Or:
		return formatBinary(e.L, "or", e.R)
	case Not:
		x, ok := formatExpr(e.X)
		return "not (" + x + ")", ok
	case Cmp:
		return formatCmp(e)
	}
	return "", false
}

func formatBinary(l Expr, op string, r Expr) (string, bool) {
	ls, lok := formatExpr(l)
	rs, rok := formatExpr(r)
	return "(" + ls + ") " + op + " (" + rs + ")", lok && rok
}

func formatCmp(c Cmp) (string, bool) {
	field := c.Field
	if c.Kind != "" {
		field = c.Kind + "." + field
	}
	f, ok := quote(field)
	if c.Op == "" {
		return f, ok
	}
	v, vok := quote(c.Value)
	return f + " " + c.Op + " " + v, ok && vok
}

func quote(s string) (string, bool) {
	switch {
	case !strings.Contains(s, `"`):
		return `"` + s + `"`, true
	case !strings.Contains(s, `'`):
		return `'` + s + `'`, true
	}
	return "", false
}
