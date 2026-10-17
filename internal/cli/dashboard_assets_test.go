// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// --assets writes the page and its stylesheet and script side by side, each
// readable by the operator alone, and the page links them relatively.
func TestDashboardAssets(t *testing.T) {
	store, dir := seed(t)
	site := filepath.Join(dir, "site")
	if err := os.Mkdir(site, 0o700); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(site, "index.html")
	r := invoke("--store", store, "dashboard", "--share-safe", "--assets", "--out", out)
	if r.code != ExitOK || r.stdout != "" || !strings.Contains(r.stderr, "wrote "+out) {
		t.Fatalf("%+v", r)
	}
	entries, err := os.ReadDir(site)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
		if fi, _ := e.Info(); fi.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
			t.Errorf("%s is readable by others: %v", e.Name(), fi.Mode())
		}
	}
	if got := strings.Join(names, " "); got != "dashboard.css dashboard.js index.html" {
		t.Fatalf("--assets wrote %q", got)
	}
	page, _ := os.ReadFile(out)
	for _, want := range []string{`<link rel="stylesheet" href="dashboard.css">`, `<script src="dashboard.js"></script>`} {
		if !strings.Contains(string(page), want) {
			t.Errorf("the page lacks %s", want)
		}
	}
	css, _ := os.ReadFile(filepath.Join(site, "dashboard.css"))
	js, _ := os.ReadFile(filepath.Join(site, "dashboard.js"))
	if !strings.Contains(string(css), ":root {") || !strings.Contains(string(js), "passmcp-graph-theme") {
		t.Error("the stylesheet or the script is not the page's")
	}
	if regexp.MustCompile(`(?i)<style|\sstyle=`).Match(page) {
		t.Error("the page still carries inline styles")
	}
}

func TestDashboardAssetsRefusals(t *testing.T) {
	store, dir := seed(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--assets"}, "--assets needs --out"},
		{[]string{"--assets", "--open"}, "--assets needs --out"},
		{[]string{"--assets", "--format", "json", "--out", filepath.Join(dir, "d.json")}, "--assets writes the HTML page"},
		{[]string{"--assets", "--out", filepath.Join(dir, "dashboard.js")}, "--out would be overwritten by an asset"},
		{[]string{"--assets", "--out", filepath.Join(dir, "DASHBOARD.CSS")}, "--out would be overwritten by an asset"},
	} {
		r := invoke(append([]string{"--store", store, "dashboard"}, c.args...)...)
		if r.code != ExitError || !strings.Contains(r.stderr, c.want) {
			t.Errorf("%v: %+v, want %q", c.args, r, c.want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "dashboard.css")); err == nil {
		t.Error("a refused --assets wrote a file")
	}
	// The asset that cannot be written is reported, after the page.
	blocked := filepath.Join(dir, "blocked")
	if err := os.MkdirAll(filepath.Join(blocked, "dashboard.css"), 0o700); err != nil {
		t.Fatal(err)
	}
	r := invoke("--store", store, "dashboard", "--assets", "--out", filepath.Join(blocked, "index.html"))
	if r.code != ExitError || !strings.Contains(r.stderr, "dashboard.css") {
		t.Errorf("an unwritable asset must fail: %+v", r)
	}
}
