// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package cli is passmcp-graph's command line. Stdout carries the selected
// output; diagnostics go to stderr.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"satellion.com/passmcp-graph/internal/analyze"
	"satellion.com/passmcp-graph/internal/export"
	"satellion.com/passmcp-graph/internal/ingest"
	"satellion.com/passmcp-graph/internal/netguard"
	"satellion.com/passmcp-graph/internal/policy"
	"satellion.com/passmcp-graph/internal/query"
	"satellion.com/passmcp-reporting/graph"
)

// Exit statuses: 0 success, 1 a check found forbidden paths, 2 a usage or
// input error.
const (
	ExitOK        = 0
	ExitViolation = 1
	ExitError     = 2
)

// Version is set at build time with -ldflags.
var Version = "dev"

// Env is what a command runs against.
type Env struct {
	Stdout, Stderr io.Writer
	Now            func() time.Time
}

const usage = `passmcp-graph builds a local graph of agents, MCP servers, tools and identities
from passmcp's evidence, and answers questions about it.

Usage:
  passmcp-graph [--store DIR] [--offline] <command> [arguments]

Commands:
  ingest PATH...                 read attestations, passmcp JSON reports and MCP client configurations
  analyze [--format text|json]   inherited risks and over-privileged identities
  query QUERY [--format text|json]
  check --policy FILE [--format text|json]
  export --format json|graphml|cypher
  import FILE                    read a graph exported as JSON
  completion bash|zsh|fish       print a shell completion script
  version

Global flags:
  --store DIR   where the graph is kept (default .passmcp-graph)
  --offline     refuse every network request (passmcp-graph makes none today)
`

// Run executes one invocation and returns its exit status.
func Run(args []string, env Env) int {
	fs, store, offline := newGlobalFlags(env)
	fs.Usage = func() { say(env.Stderr, "%s", usage) }
	if err := fs.Parse(args); err != nil {
		return ExitError
	}
	if *offline {
		defer netguard.Deny()()
	}
	rest := fs.Args()
	if len(rest) == 0 {
		say(env.Stderr, "%s", usage)
		return ExitError
	}
	cmd, ok := commands[rest[0]]
	if !ok {
		say(env.Stderr, "passmcp-graph: unknown command %q\n\n%s", rest[0], usage)
		return ExitError
	}
	code, err := cmd(*store, rest[1:], env)
	if err != nil {
		say(env.Stderr, "passmcp-graph: %v\n", err)
		return ExitError
	}
	return code
}

// newGlobalFlags defines the flags that come before the command.
func newGlobalFlags(env Env) (fs *flag.FlagSet, store *string, offline *bool) {
	fs = flag.NewFlagSet("passmcp-graph", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	store = fs.String("store", ".passmcp-graph", "")
	offline = fs.Bool("offline", false, "")
	return fs, store, offline
}

type command func(store string, args []string, env Env) (int, error)

var commands map[string]command

func init() {
	commands = map[string]command{
		"ingest": runIngest, "analyze": runAnalyze, "query": runQuery, "check": runCheck,
		"export": runExport, "import": runImport, "completion": runCompletion,
		"version": func(_ string, _ []string, env Env) (int, error) {
			sayln(env.Stdout, "passmcp-graph "+Version)
			return ExitOK, nil
		},
	}
}

// subFlags parses a subcommand's flags, allowing flags after positional
// arguments, and returns the positionals.
func subFlags(name string, args []string, env Env, define func(*flag.FlagSet)) ([]string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	define(fs)
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return pos, nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// say and sayln write output and diagnostics. A failed write to stdout or
// stderr has nowhere left to be reported, so the error is discarded
// deliberately rather than by omission.
func say(w io.Writer, format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) }

func sayln(w io.Writer, s string) { _, _ = fmt.Fprintln(w, s) }

func format(f string) error {
	if f != "text" && f != "json" {
		return fmt.Errorf("--format is %q, want text or json", f)
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func runIngest(store string, args []string, env Env) (int, error) {
	paths, err := subFlags("ingest", args, env, func(*flag.FlagSet) {})
	if err != nil {
		return ExitError, err
	}
	if len(paths) == 0 {
		return ExitError, errors.New("ingest needs at least one file or directory")
	}
	g, err := graph.Load(store)
	if err != nil {
		return ExitError, err
	}
	res, err := ingest.Paths(g, paths)
	if err != nil {
		return ExitError, err
	}
	if err := graph.Save(store, g); err != nil {
		return ExitError, err
	}
	for _, s := range res.Skipped {
		sayln(env.Stderr, "skipped "+s)
	}
	say(env.Stdout, "ingested %d attestation(s), %d report(s), %d configuration(s); the graph has %d nodes and %d edges\n",
		res.Read["attestation"], res.Read["report"], res.Read["config"], len(g.Nodes), len(g.Edges))
	return ExitOK, nil
}

func runAnalyze(store string, args []string, env Env) (int, error) {
	var f string
	if _, err := subFlags("analyze", args, env, func(fs *flag.FlagSet) { fs.StringVar(&f, "format", "text", "") }); err != nil {
		return ExitError, err
	}
	if err := format(f); err != nil {
		return ExitError, err
	}
	g, err := graph.Load(store)
	if err != nil {
		return ExitError, err
	}
	risks, flags := analyze.Risks(g), analyze.OverPrivileged(g)
	if f == "json" {
		return ExitOK, writeJSON(env.Stdout, map[string]any{"risks": risks, "overPrivileged": flags})
	}
	for _, r := range risks {
		say(env.Stdout, "risk: %s inherits %s (%s) via %s\n", r.Agent, r.Finding, r.Severity, strings.Join(r.Path, " -> "))
	}
	for _, fl := range flags {
		say(env.Stdout, "%s: %s on %s: %s\n", fl.Flag, fl.Identity, fl.Server, fl.Reason)
	}
	if len(risks)+len(flags) == 0 {
		sayln(env.Stdout, "no inherited risk and no over-privileged identity")
	}
	return ExitOK, nil
}

func runQuery(store string, args []string, env Env) (int, error) {
	var f string
	pos, err := subFlags("query", args, env, func(fs *flag.FlagSet) { fs.StringVar(&f, "format", "text", "") })
	if err != nil {
		return ExitError, err
	}
	if err := format(f); err != nil {
		return ExitError, err
	}
	if len(pos) != 1 {
		return ExitError, errors.New("query takes one quoted query")
	}
	q, err := query.Parse(pos[0])
	if err != nil {
		return ExitError, err
	}
	g, err := graph.Load(store)
	if err != nil {
		return ExitError, err
	}
	paths, err := query.Eval(g, q, query.Context{Now: env.Now()})
	if err != nil {
		return ExitError, err
	}
	if f == "json" {
		if paths == nil {
			paths = []query.Path{}
		}
		return ExitOK, writeJSON(env.Stdout, paths)
	}
	for _, p := range paths {
		sayln(env.Stdout, p.String())
	}
	say(env.Stderr, "%d path(s)\n", len(paths))
	return ExitOK, nil
}

func runCheck(store string, args []string, env Env) (int, error) {
	var f, file string
	if _, err := subFlags("check", args, env, func(fs *flag.FlagSet) {
		fs.StringVar(&f, "format", "text", "")
		fs.StringVar(&file, "policy", "", "")
	}); err != nil {
		return ExitError, err
	}
	if err := format(f); err != nil {
		return ExitError, err
	}
	if file == "" {
		return ExitError, errors.New("check needs --policy FILE")
	}
	vs, err := violations(store, file, env)
	if err != nil {
		return ExitError, err
	}
	if err := writeViolations(f, vs, env); err != nil {
		return ExitError, err
	}
	if len(vs) > 0 {
		say(env.Stderr, "%d forbidden path(s)\n", len(vs))
		return ExitViolation, nil
	}
	sayln(env.Stderr, "no forbidden path")
	return ExitOK, nil
}

// violations evaluates the policy file against the stored graph.
func violations(store, file string, env Env) ([]policy.Violation, error) {
	raw, err := os.ReadFile(file) // #nosec G304 -- the operator names the policy file
	if err != nil {
		return nil, err
	}
	p, err := policy.Parse(raw)
	if err != nil {
		return nil, err
	}
	g, err := graph.Load(store)
	if err != nil {
		return nil, err
	}
	return p.Check(g, query.Context{Now: env.Now()})
}

// writeViolations prints the violations to stdout in the selected format;
// JSON is always an array, empty when the policy holds.
func writeViolations(f string, vs []policy.Violation, env Env) error {
	if f == "json" {
		if vs == nil {
			vs = []policy.Violation{}
		}
		return writeJSON(env.Stdout, vs)
	}
	for _, v := range vs {
		say(env.Stdout, "forbidden (%s): %s\n", v.Rule, v.Path.String())
	}
	return nil
}

func runExport(store string, args []string, env Env) (int, error) {
	var f string
	if _, err := subFlags("export", args, env, func(fs *flag.FlagSet) { fs.StringVar(&f, "format", "json", "") }); err != nil {
		return ExitError, err
	}
	g, err := graph.Load(store)
	if err != nil {
		return ExitError, err
	}
	return ExitOK, export.Write(env.Stdout, g, f)
}

func runImport(store string, args []string, env Env) (int, error) {
	pos, err := subFlags("import", args, env, func(*flag.FlagSet) {})
	if err != nil {
		return ExitError, err
	}
	if len(pos) != 1 {
		return ExitError, errors.New("import takes one JSON file")
	}
	raw, err := os.ReadFile(pos[0]) // #nosec G304 -- the operator names the file to import
	if err != nil {
		return ExitError, err
	}
	in, err := export.Import(raw)
	if err != nil {
		return ExitError, err
	}
	g, err := graph.Load(store)
	if err != nil {
		return ExitError, err
	}
	g.Merge(in)
	if err := graph.Save(store, g); err != nil {
		return ExitError, err
	}
	say(env.Stdout, "imported %d nodes and %d edges\n", len(in.Nodes), len(in.Edges))
	return ExitOK, nil
}
