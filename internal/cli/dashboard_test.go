// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-graph/internal/fixture"
)

// invokeOpening runs a command whose --open is recorded, not performed.
func invokeOpening(open func(string) error, args ...string) result {
	var out, errb bytes.Buffer
	code := Run(args, Env{Stdout: &out, Stderr: &errb, Now: func() time.Time { return fixture.Now }, Open: open})
	return result{code, out.String(), errb.String()}
}

func TestDashboardWritesToOut(t *testing.T) {
	store, dir := seed(t)
	out := filepath.Join(dir, "estate.html")
	r := invoke("--store", store, "dashboard", "--out", out)
	if r.code != ExitOK || r.stdout != "" || !strings.Contains(r.stderr, "wrote "+out) {
		t.Fatalf("%+v", r)
	}
	b, err := os.ReadFile(out)
	if err != nil || !strings.HasPrefix(string(b), "<!doctype html>") {
		t.Fatalf("the page is not in --out: %v", err)
	}
	if fi, _ := os.Stat(out); fi.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("the page is readable by others: %v", fi.Mode())
	}
	if r := invoke("--store", store, "dashboard", "--out", out, "--format", "json"); r.code != ExitOK {
		t.Fatalf("json to --out: %+v", r)
	}
	if b, _ := os.ReadFile(out); !strings.HasPrefix(string(b), "{") {
		t.Fatal("--format json --out did not write JSON")
	}
	if r := invoke("--store", store, "dashboard", "--out", filepath.Join(dir, "no", "such", "dir.html")); r.code != ExitError {
		t.Fatalf("an unwritable --out must fail: %+v", r)
	}
}

func TestDashboardOpen(t *testing.T) {
	store, dir := seed(t)
	var opened []string
	record := func(p string) error { opened = append(opened, p); return nil }

	out := filepath.Join(dir, "estate.html")
	if r := invokeOpening(record, "--store", store, "dashboard", "--out", out, "--open"); r.code != ExitOK || len(opened) != 1 || opened[0] != out {
		t.Fatalf("--out --open: %+v, opened %v", r, opened)
	}
	r := invokeOpening(record, "--store", store, "dashboard", "--open")
	if r.code != ExitOK || len(opened) != 2 || r.stdout != "" {
		t.Fatalf("--open alone: %+v, opened %v", r, opened)
	}
	defer func() { _ = os.Remove(opened[1]) }()
	if b, err := os.ReadFile(opened[1]); err != nil || !strings.Contains(string(b), "claude-desktop") {
		t.Fatalf("--open alone did not open the page: %v", err)
	}
	fail := func(string) error { return errors.New("no display") }
	if r := invokeOpening(fail, "--store", store, "dashboard", "--out", out, "--open"); r.code != ExitError || !strings.Contains(r.stderr, "no display") {
		t.Fatalf("a failed open must be reported: %+v", r)
	}
	if r := invoke("--store", store, "dashboard", "--open", "--format", "json"); r.code != ExitError || len(opened) != 2 {
		t.Fatalf("--open with json must be refused: %+v", r)
	}
}

func TestOpenerPerOS(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  "open /tmp/p.html",
		"linux":   "xdg-open /tmp/p.html",
		"freebsd": "xdg-open /tmp/p.html",
		"windows": "rundll32 url.dll,FileProtocolHandler /tmp/p.html",
	} {
		name, args := opener(goos, "/tmp/p.html")
		if got := strings.Join(append([]string{name}, args...), " "); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}

// --share-safe replaces endpoints and local paths in both formats.
func TestDashboardShareSafe(t *testing.T) {
	store, _ := seed(t)
	for _, f := range []string{"html", "json"} {
		r := invoke("--store", store, "dashboard", "--share-safe", "--format", f)
		if r.code != ExitOK || strings.Contains(r.stdout, "/home/dev/") || strings.Contains(r.stdout, "crm.example") ||
			!strings.Contains(r.stdout, "http-server-1") {
			t.Errorf("%s: code %d, stderr %q", f, r.code, r.stderr)
		}
	}
}
