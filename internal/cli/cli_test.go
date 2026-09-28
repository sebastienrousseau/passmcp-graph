// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package cli

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-reporting/graph"
)

type result struct {
	code           int
	stdout, stderr string
}

func invoke(args ...string) result {
	var out, errb bytes.Buffer
	code := Run(args, Env{Stdout: &out, Stderr: &errb, Now: func() time.Time { return fixture.Now }})
	return result{code, out.String(), errb.String()}
}

func seed(t *testing.T) (store, dir string) {
	t.Helper()
	dir = t.TempDir()
	store = filepath.Join(dir, "store")
	if err := graph.Save(store, fixture.Graph()); err != nil {
		t.Fatal(err)
	}
	return store, dir
}

func put(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEveryCommandWorksEndToEnd(t *testing.T) {
	store, dir := seed(t)
	att := put(t, dir, "a.json", string(fixture.Statement("http", "https://new.example/mcp", fixture.Now, 80, "B", "none")))

	if r := invoke("--store", store, "ingest", att, filepath.Join(dir, "missing-is-not-here.txt")); r.code != ExitError {
		t.Fatalf("a missing path must fail: %+v", r)
	}
	junk := put(t, dir, "junk.json", `{"x":1}`)
	if r := invoke("--store", store, "ingest", att, junk); r.code != ExitOK || !strings.Contains(r.stdout, "ingested 1 attestation(s)") || !strings.Contains(r.stderr, "skipped") {
		t.Fatalf("ingest: %+v", r)
	}
	if r := invoke("--store", store, "analyze"); r.code != ExitOK || !strings.Contains(r.stdout, "risk: claude-desktop inherits auth.unauthenticated_tools") ||
		!strings.Contains(r.stdout, "over-privileged: ops-agent") {
		t.Fatalf("analyze: %+v", r)
	}
	if r := invoke("--store", store, "analyze", "--format", "json"); r.code != ExitOK || !json.Valid([]byte(r.stdout)) {
		t.Fatalf("analyze json: %+v", r)
	}
	q := "agents -> tools where destructive and server.auth = none"
	if r := invoke("--store", store, "query", q); r.code != ExitOK || strings.TrimSpace(r.stdout) != "claude-desktop -> https://crm.example/mcp -> delete_record" {
		t.Fatalf("query: %+v", r)
	}
	if r := invoke("--store", store, "query", "servers where grade = Z", "--format", "json"); r.code != ExitOK || strings.TrimSpace(r.stdout) != "[]" {
		t.Fatalf("query json: %+v", r)
	}
	for _, f := range []string{"json", "graphml", "cypher"} {
		if r := invoke("--store", store, "export", "--format", f); r.code != ExitOK || r.stdout == "" {
			t.Fatalf("export %s: %+v", f, r)
		}
	}
	exported := invoke("--store", store, "export").stdout
	file := put(t, dir, "export.json", exported)
	other := filepath.Join(dir, "other")
	if r := invoke("--store", other, "import", file); r.code != ExitOK {
		t.Fatalf("import: %+v", r)
	}
	if invoke("--store", other, "export").stdout != exported {
		t.Fatal("import then export lost something")
	}
	if r := invoke("version"); r.code != ExitOK || !strings.HasPrefix(r.stdout, "passmcp-graph ") {
		t.Fatalf("version: %+v", r)
	}
}

// The policy gate, driven through the command line: forbidden paths exit 1
// and are named; a clean graph exits 0.
func TestCheckExitsNonZeroOnForbiddenPaths(t *testing.T) {
	store, dir := seed(t)
	strict := put(t, dir, "p.yaml", "version: 1\nforbid:\n  - name: unattested\n    query: agents -> servers where not attested\n")
	r := invoke("--store", store, "check", "--policy", strict)
	if r.code != ExitViolation || !strings.Contains(r.stdout, "forbidden (unattested): claude-desktop -> npx @example/shell-mcp") {
		t.Fatalf("check: %+v", r)
	}
	if r := invoke("--store", store, "check", "--policy", strict, "--format", "json"); r.code != ExitViolation || !json.Valid([]byte(r.stdout)) {
		t.Fatalf("check json: %+v", r)
	}
	clean := put(t, dir, "c.yaml", "version: 1\nforbid:\n  - name: z\n    query: servers where grade = Z\n")
	if r := invoke("--store", store, "check", "--policy", clean); r.code != ExitOK || !strings.Contains(r.stderr, "no forbidden path") {
		t.Fatalf("clean: %+v", r)
	}
	if r := invoke("--store", store, "check", "--policy", clean, "--format", "json"); r.code != ExitOK || strings.TrimSpace(r.stdout) != "[]" {
		t.Fatalf("clean json: %+v", r)
	}
}

func TestUsageAndInputErrors(t *testing.T) {
	store, dir := seed(t)
	bad := put(t, dir, "bad.json", "{")
	badPolicy := put(t, dir, "bad.yaml", "version: 9\n")
	fieldPolicy := put(t, dir, "field.yaml", "version: 1\nforbid:\n  - name: a\n    query: agents where destructive\n")
	notDir := put(t, dir, "file", "x")
	for _, args := range [][]string{
		{}, {"--nope"}, {"frobnicate"},
		{"--store", store, "ingest"}, {"--store", store, "ingest", "--x"},
		{"--store", notDir, "ingest", bad},
		{"--store", store, "analyze", "--format", "xml"}, {"--store", store, "analyze", "--x"},
		{"--store", notDir, "analyze"},
		{"--store", store, "query"}, {"--store", store, "query", "a", "b"}, {"--store", store, "query", "widgets"},
		{"--store", store, "query", "agents", "--format", "xml"}, {"--store", store, "query", "--x"},
		{"--store", store, "query", "agents where destructive"}, {"--store", notDir, "query", "agents"},
		{"--store", store, "check"}, {"--store", store, "check", "--policy", filepath.Join(dir, "none.yaml")},
		{"--store", store, "check", "--policy", badPolicy}, {"--store", store, "check", "--policy", fieldPolicy},
		{"--store", store, "check", "--format", "xml", "--policy", badPolicy}, {"--store", store, "check", "--x"},
		{"--store", notDir, "check", "--policy", fieldPolicy},
		{"--store", store, "export", "--format", "dot"}, {"--store", store, "export", "--x"}, {"--store", notDir, "export"},
		{"--store", store, "import"}, {"--store", store, "import", filepath.Join(dir, "none.json")},
		{"--store", store, "import", bad}, {"--store", store, "import", "--x"},
	} {
		if r := invoke(args...); r.code != ExitError {
			t.Errorf("%v: want exit %d, got %+v", args, ExitError, r)
		}
	}
	good := invoke("--store", store, "export").stdout
	file := put(t, dir, "good.json", good)
	if r := invoke("--store", notDir, "import", file); r.code != ExitError {
		t.Errorf("import into an unusable store: %+v", r)
	}
	att := put(t, dir, "att.json", string(fixture.Statement("http", "https://z.example", fixture.Now, 1, "F", "none")))
	if r := invoke("--store", filepath.Join(notDir, "sub"), "ingest", att); r.code != ExitError {
		t.Errorf("ingest into an unusable store: %+v", r)
	}
	empty := filepath.Join(dir, "empty")
	if r := invoke("--store", empty, "analyze"); r.code != ExitOK || !strings.Contains(r.stdout, "no inherited risk") {
		t.Errorf("an empty store analyses clean: %+v", r)
	}
}

// recorder counts every request made through net/http's default transport.
type recorder struct{ n atomic.Int64 }

func (r *recorder) RoundTrip(*http.Request) (*http.Response, error) {
	r.n.Add(1)
	return nil, http.ErrUseLastResponse
}

// AC: SG-07
// With its network egress recorded, every passmcp-graph command contacts
// nothing — there are no connectors configured — and with --offline it
// contacts nothing at all. Two recordings: the default HTTP transport, and a
// local proxy listener every proxy variable points at. A structural test
// below backs them: nothing in the binary but the offline guard imports a
// networking package.
func TestNoCommandContactsTheNetwork(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var accepted atomic.Int64
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			_ = c.Close()
		}
	}()
	defer func() { _ = ln.Close() }()
	proxy := "http://" + ln.Addr().String()
	for _, v := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(v, proxy)
	}
	rec := &recorder{}
	prev := http.DefaultTransport
	http.DefaultTransport = rec
	defer func() { http.DefaultTransport = prev }()

	store, dir := seed(t)
	att := put(t, dir, "a.json", string(fixture.Statement("http", "https://new.example/mcp", fixture.Now, 80, "B", "none")))
	cfg := put(t, dir, "claude_desktop_config.json", `{"mcpServers": {"x": {"url": "https://remote.example/mcp"}}}`)
	pol := put(t, dir, "p.yaml", "version: 1\nforbid:\n  - name: a\n    query: agents -> servers where not attested\n")
	exp := put(t, dir, "g.json", invoke("--store", store, "export").stdout)
	commands := [][]string{
		{"ingest", att, cfg}, {"analyze"}, {"query", "agents -> tools"}, {"check", "--policy", pol},
		{"export", "--format", "graphml"}, {"import", exp}, {"version"},
	}
	for _, offline := range []bool{false, true} {
		for _, c := range commands {
			args := []string{"--store", store}
			if offline {
				args = append(args, "--offline")
			}
			if r := invoke(append(args, c...)...); r.code == ExitError {
				t.Fatalf("%v: %+v", c, r)
			}
		}
	}
	if http.DefaultTransport != rec {
		t.Fatal("--offline did not restore the transport")
	}
	if rec.n.Load() != 0 || accepted.Load() != 0 {
		t.Fatalf("passmcp-graph made %d request(s) and %d connection(s)", rec.n.Load(), accepted.Load())
	}
}

// AC: SG-07
// Structurally: of every package the binary links, only the offline guard
// imports net or net/http outside the standard library itself, so no command
// can reach the network without that showing up here.
func TestOnlyTheOfflineGuardImportsTheNetwork(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}|{{.Standard}}|{{join .Imports \",\"}}", "../../cmd/passmcp-graph").Output()
	if err != nil {
		t.Skipf("go list unavailable: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 || parts[1] == "true" {
			continue
		}
		for _, imp := range strings.Split(parts[2], ",") {
			if imp == "net" || imp == "net/http" || strings.HasPrefix(imp, "net/http/") {
				if parts[0] != "satellion.com/passmcp-graph/internal/netguard" {
					t.Errorf("%s imports %s", parts[0], imp)
				}
			}
		}
	}
}
