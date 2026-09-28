// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package query implements passmcp-graph's query language: a path of node kinds
// joined by "->", filtered by an optional condition.
//
//	query      = path [ "where" expr ]
//	path       = kind { "->" kind }
//	kind       = "agents" | "servers" | "tools" | "identities" | "sources" | "attestations"
//	expr       = term { "or" term }
//	term       = factor { "and" factor }
//	factor     = "not" factor | "(" expr ")" | comparison
//	comparison = field [ op value ]
//	field      = [ kindname "." ] name
//	op         = "=" | "!=" | "<" | "<=" | ">" | ">=" | "contains"
//	value      = word | number | quoted string
//
// Two kinds with no direct relationship are joined through the server between
// them, so "agents -> tools" walks agent -uses-> server -exposes-> tool and
// the server is part of the result.
package query

import (
	"fmt"
	"strings"
	"unicode"
)

// Query is a parsed query.
type Query struct {
	// Kinds is the path as written, singular: "agent", "server" and so on.
	Kinds []string
	// Where is the condition, or nil.
	Where Expr
}

// Expr is a condition node.
type Expr interface{ expr() }

// And is true when both sides are.
type And struct{ L, R Expr }

// Or is true when either side is.
type Or struct{ L, R Expr }

// Not negates its operand.
type Not struct{ X Expr }

// Cmp compares a field with a value; Op empty means the field's truthiness.
type Cmp struct {
	Kind, Field, Op, Value string
}

func (And) expr() {}
func (Or) expr()  {}
func (Not) expr() {}
func (Cmp) expr() {}

// kinds maps the words a query may use to the node kind's singular name.
var kinds = map[string]string{
	"agent": "agent", "agents": "agent", "server": "server", "servers": "server",
	"tool": "tool", "tools": "tool", "identity": "identity", "identities": "identity",
	"source": "source", "sources": "source", "attestation": "attestation", "attestations": "attestation",
}

// Parse parses a query.
func Parse(s string) (*Query, error) {
	toks, err := lex(s)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	q := &Query{}
	for {
		t := p.next()
		k, ok := kinds[strings.ToLower(t)]
		if !ok {
			return nil, fmt.Errorf("query: %q is not a node kind (agents, servers, tools, identities, sources, attestations)", t)
		}
		q.Kinds = append(q.Kinds, k)
		if p.peek() != "->" {
			break
		}
		p.next()
	}
	switch strings.ToLower(p.peek()) {
	case "":
		return q, nil
	case "where":
		p.next()
		q.Where, err = p.or()
		if err != nil {
			return nil, err
		}
		if p.peek() != "" {
			return nil, fmt.Errorf("query: unexpected %q", p.peek())
		}
		return q, nil
	}
	return nil, fmt.Errorf("query: expected \"->\" or \"where\", found %q", p.peek())
}

type parser struct {
	toks []string
	i    int
}

func (p *parser) peek() string {
	if p.i < len(p.toks) {
		return p.toks[p.i]
	}
	return ""
}

func (p *parser) next() string {
	t := p.peek()
	p.i++
	return t
}

func (p *parser) or() (Expr, error) {
	l, err := p.and()
	for err == nil && strings.EqualFold(p.peek(), "or") {
		p.next()
		var r Expr
		r, err = p.and()
		l = Or{l, r}
	}
	return l, err
}

func (p *parser) and() (Expr, error) {
	l, err := p.factor()
	for err == nil && strings.EqualFold(p.peek(), "and") {
		p.next()
		var r Expr
		r, err = p.factor()
		l = And{l, r}
	}
	return l, err
}

func (p *parser) factor() (Expr, error) {
	switch t := p.next(); {
	case strings.EqualFold(t, "not"):
		x, err := p.factor()
		return Not{x}, err
	case t == "(":
		x, err := p.or()
		if err == nil && p.next() != ")" {
			err = fmt.Errorf("query: missing \")\"")
		}
		return x, err
	case t == "" || isOp(t) || t == ")":
		return nil, fmt.Errorf("query: expected a field, found %q", t)
	default:
		return p.comparison(t)
	}
}

func (p *parser) comparison(field string) (Expr, error) {
	c := Cmp{Field: strings.ToLower(field)}
	if k, f, ok := strings.Cut(c.Field, "."); ok {
		kind, known := kinds[k]
		if !known {
			return nil, fmt.Errorf("query: %q is not a node kind", k)
		}
		c.Kind, c.Field = kind, f
	}
	if !isOp(p.peek()) {
		return c, nil
	}
	c.Op = strings.ToLower(p.next())
	v := p.next()
	if v == "" || isOp(v) || v == "(" || v == ")" {
		return nil, fmt.Errorf("query: %s needs a value", c.Op)
	}
	c.Value = v
	return c, nil
}

func isOp(t string) bool {
	switch strings.ToLower(t) {
	case "=", "!=", "<", "<=", ">", ">=", "contains":
		return true
	}
	return false
}

// lex splits a query into tokens: words, quoted strings (quotes removed),
// "->", comparison operators and parentheses.
func lex(s string) ([]string, error) {
	var toks []string
	rs := []rune(s)
	for i := 0; i < len(rs); {
		var tok string
		var err error
		switch r := rs[i]; {
		case unicode.IsSpace(r):
			i++
			continue
		case r == '(' || r == ')':
			tok, i = string(r), i+1
		case r == '"' || r == '\'':
			tok, i, err = lexQuoted(rs, i)
		case strings.ContainsRune("-<>=!", r):
			tok, i = lexOperator(rs, i)
		default:
			tok, i = lexWord(rs, i)
		}
		if err != nil {
			return nil, err
		}
		toks = append(toks, tok)
	}
	return toks, nil
}

// lexQuoted reads a string quoted with the rune at i.
func lexQuoted(rs []rune, i int) (string, int, error) {
	j := i + 1
	for j < len(rs) && rs[j] != rs[i] {
		j++
	}
	if j == len(rs) {
		return "", 0, fmt.Errorf("query: unterminated string")
	}
	return string(rs[i+1 : j]), j + 1, nil
}

// lexOperator reads "->" or a comparison operator.
func lexOperator(rs []rune, i int) (string, int) {
	j := i + 1
	for j < len(rs) && strings.ContainsRune("-<>=", rs[j]) {
		j++
	}
	return string(rs[i:j]), j
}

// lexWord reads a bare word, which may contain "-" but ends at "->".
func lexWord(rs []rune, i int) (string, int) {
	j := i
	for j < len(rs) && !endsWord(rs, j) {
		j++
	}
	return string(rs[i:j]), j
}

func endsWord(rs []rune, j int) bool {
	r := rs[j]
	return unicode.IsSpace(r) || strings.ContainsRune("()<>=!\"'", r) || (r == '-' && j+1 < len(rs) && rs[j+1] == '>')
}
