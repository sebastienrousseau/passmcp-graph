// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-graph/internal/policy"
)

// probeWants are the lines testdata/probe.html prints for the fixture when
// the page's script works: filters, search, selection, fragments, themes,
// the export download and reflow at 320 CSS pixels.
var probeWants = []string{
	"hash srv-1 open: true",
	"pick cursor: paths 2, servers 2",
	"2 of 6 paths · 2 of 4 attention items (1 finding, 1 evidence gap)",
	"attention: 2 attention items in this selection: 1 finding · 1 evidence gap. 4 attention items in the whole snapshot.",
	"selection pill: true cursor",
	"topology filtered, hidden: 12 of 12",
	"topology with context, hidden: 0",
	"repro label: Reproduce this view:",
	"search crm: servers 1, paths 2, findings 1",
	"search repro label: Closest command:",
	"status stale: servers 1",
	"unannotated: paths 3",
	"cleared: paths 6, servers 4, attention 4, hidden topology 0 | Showing the whole snapshot.",
	"all columns: true",
	"back shown true, ← Back to A critical check failed",
	"after back: hash #fnd-0, focus in fnd-0 true",
	"320px: scrollWidth 320 clientWidth 320; server status clipped without ellipsis: 0, at 1440px 0",
	"flows drawn: 18",
	"severity low: rows 2",
	"ctrl-k focuses search: true",
	"themes: light dark system",
	"hero pick: cursor, pressed true",
	"export: blobs 1, CSP violations 0",
}

// strictCSP is the policy a host such as satellion.com sends as a header,
// without its analytics hosts. A file:// page gets no header, so the probe
// puts it first in the page's head as a meta element, which the browser
// enforces the same way for everything after it.
const strictCSP = `<meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:">`

// cspWatch records every Content-Security-Policy violation in the page, for
// the probe to print. It is loaded before anything the page has.
const cspWatch = `window.cspViolations = [];
document.addEventListener("securitypolicyviolation", function (e) {
  window.cspViolations.push(e.violatedDirective + " " + (e.blockedURI || "inline"));
});
`

// renderForProbe writes the page into dir as dashboard.html, linked to its
// assets or as one file, with head inserted at the top of its head.
func renderForProbe(t *testing.T, dir string, linked bool, head string) {
	t.Helper()
	s := build(t, fixture.Graph(), []policy.Violation{})
	var b bytes.Buffer
	render := Render
	if linked {
		render = RenderLinked
		for _, a := range Assets() {
			if err := os.WriteFile(filepath.Join(dir, a.Name), []byte(a.Body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := render(&b, s); err != nil {
		t.Fatal(err)
	}
	page := strings.Replace(b.String(), "<head>", "<head>\n"+head, 1)
	if err := os.WriteFile(filepath.Join(dir, "dashboard.html"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cspwatch.js"), []byte(cspWatch), 0o600); err != nil {
		t.Fatal(err)
	}
}

var cspLine = regexp.MustCompile(`csp violations: (.*) at 1280px, (.*) at 320px`)

// probeCSP renders the page, opens it in the probe, and returns what the
// probe printed and the violations it counted in each of its two frames.
func probeCSP(t *testing.T, chrome string, linked bool, head string) (got string, wide, narrow string) {
	t.Helper()
	dir := t.TempDir()
	renderForProbe(t, dir, linked, head)
	got = probeDir(t, chrome, dir)
	m := cspLine.FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("the probe printed no violation count:\n%s", got)
	}
	return got, m[1], m[2]
}

// The page passes the whole probe under a strict Content-Security-Policy
// when it links its assets (dashboard --assets), and under its own policy
// as one file, with no violation in either; the single file under the
// strict policy is blocked, which shows the count is real.
func TestUnderAStrictContentSecurityPolicy(t *testing.T) {
	chrome := os.Getenv("PASSMCP_GRAPH_BROWSER")
	if chrome == "" {
		t.Skip("set PASSMCP_GRAPH_BROWSER to a Chrome binary to run the browser check")
	}
	for _, c := range []struct {
		name   string
		linked bool
		head   string
	}{
		{"linked, strict policy", true, strictCSP + "\n<script src=\"cspwatch.js\"></script>"},
		{"single file, its own policy", false, "<script>" + cspWatch + "</script>"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, wide, narrow := probeCSP(t, chrome, c.linked, c.head)
			t.Logf("violations: %s at 1280px, %s at 320px", wide, narrow)
			if wide != "0" || narrow != "0" {
				t.Errorf("violations: %s at 1280px, %s at 320px", wide, narrow)
			}
			for _, want := range probeWants {
				if !strings.Contains(got, want) {
					t.Errorf("browser check lacks %q in:\n%s", want, got)
				}
			}
		})
	}
	t.Run("single file, strict policy", func(t *testing.T) {
		dir := t.TempDir()
		renderForProbe(t, dir, false, strictCSP+"\n<script src=\"cspwatch.js\"></script>")
		got := probeDir(t, chrome, dir)
		m := regexp.MustCompile(`csp violations at load: (\d+) `).FindStringSubmatch(got)
		t.Logf("%s", regexp.MustCompile(`csp violations at load: .*`).FindString(got))
		if m == nil || m[1] == "0" || strings.Contains(got, probeWants[0]) {
			t.Errorf("the strict policy blocked nothing on the single file:\n%s", got)
		}
	})
}
