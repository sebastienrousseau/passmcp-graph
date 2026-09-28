// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package policy checks a graph against a policy: a list of paths that must
// not exist, each written as a query. A CI job runs `passmcp-graph check` and
// fails on any path the policy forbids.
//
//	version: 1
//	forbid:
//	  - name: unattested
//	    reason: an agent uses a server nobody has attested
//	    query: agents -> servers where not attested
//	  - name: stale
//	    reason: the server's attestation is older than 30 days
//	    query: agents -> servers where attestation_age_days > 30
package policy

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
	"satellion.com/passmcp-graph/internal/query"
	"satellion.com/passmcp-reporting/graph"
)

// Policy is a parsed policy file.
type Policy struct {
	Version int    `yaml:"version"`
	Forbid  []Rule `yaml:"forbid"`
}

// Rule forbids every path its query returns.
type Rule struct {
	Name   string `yaml:"name"`
	Reason string `yaml:"reason"`
	Query  string `yaml:"query"`

	parsed *query.Query
}

// Violation is one forbidden path that exists.
type Violation struct {
	Rule   string     `json:"rule"`
	Reason string     `json:"reason,omitempty"`
	Path   query.Path `json:"path"`
}

// Parse reads a policy, refusing unknown keys and any rule whose query does
// not parse: a policy that silently checks less than it says is the failure
// a gate exists to prevent.
func Parse(b []byte) (*Policy, error) {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var p Policy
	if err := dec.Decode(&p); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("policy: %w", err)
	}
	if p.Version != 1 {
		return nil, fmt.Errorf("policy: version is %d, want 1", p.Version)
	}
	if len(p.Forbid) == 0 {
		return nil, fmt.Errorf("policy: no forbid rules, so it would pass anything")
	}
	seen := map[string]bool{}
	for i := range p.Forbid {
		r := &p.Forbid[i]
		if r.Name == "" || seen[r.Name] {
			return nil, fmt.Errorf("policy: rule %d needs a unique name", i+1)
		}
		seen[r.Name] = true
		q, err := query.Parse(r.Query)
		if err != nil {
			return nil, fmt.Errorf("policy: rule %q: %w", r.Name, err)
		}
		r.parsed = q
	}
	return &p, nil
}

// Check returns every forbidden path in g, in rule order.
func (p *Policy) Check(g *graph.Graph, ctx query.Context) ([]Violation, error) {
	var out []Violation
	for _, r := range p.Forbid {
		paths, err := query.Eval(g, r.parsed, ctx)
		if err != nil {
			return nil, fmt.Errorf("policy: rule %q: %w", r.Name, err)
		}
		for _, path := range paths {
			out = append(out, Violation{Rule: r.Name, Reason: r.Reason, Path: path})
		}
	}
	return out, nil
}
