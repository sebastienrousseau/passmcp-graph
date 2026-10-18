// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"strings"
	"testing"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-graph/internal/policy"
)

// styleAttr is a style attribute on any element, which a
// Content-Security-Policy without style-src 'unsafe-inline' blocks.
var styleAttr = regexp.MustCompile(`(?i)<[^>]*\sstyle\s*=`)

// The single-file page sets no style attribute either, so a stricter
// policy than its own, applied by a host, blocks nothing it styles.
func TestTheSingleFilePageHasNoStyleAttribute(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, build(t, fixture.Graph(), []policy.Violation{})); err != nil {
		t.Fatal(err)
	}
	if m := styleAttr.FindString(b.String()); m != "" {
		t.Fatalf("the page sets a style attribute: %q", m)
	}
}

// renderBoth renders the fixture as one file and as a page that links its
// assets.
func renderBoth(t *testing.T) (single, linked string) {
	t.Helper()
	s := build(t, fixture.Graph(), []policy.Violation{})
	var a, b bytes.Buffer
	if err := Render(&a, s); err != nil {
		t.Fatal(err)
	}
	if err := RenderLinked(&b, s); err != nil {
		t.Fatal(err)
	}
	return a.String(), b.String()
}

// The linked page carries no inline style or script, so a policy of
// style-src 'self' and script-src 'self' blocks nothing on it. Its one
// inline script element is the export, a data block the browser never runs.
func TestTheLinkedPageIsStrictCSPClean(t *testing.T) {
	_, page := renderBoth(t)
	if m := styleAttr.FindString(page); m != "" {
		t.Errorf("the page sets a style attribute: %q", m)
	}
	if strings.Contains(page, "<style") {
		t.Error("the page has a <style> element")
	}
	scripts := regexp.MustCompile(`<script([^>]*)>`).FindAllStringSubmatch(page, -1)
	if len(scripts) != 2 {
		t.Fatalf("want the linked script and the export, got %d script element(s)", len(scripts))
	}
	for _, m := range scripts {
		if m[1] != ` src="dashboard.js"` && m[1] != ` type="application/json" id="export"` {
			t.Errorf("an unexpected script element: <script%s>", m[1])
		}
	}
	if !strings.Contains(page, `<script src="dashboard.js"></script>`) ||
		!strings.Contains(page, `<link rel="stylesheet" href="dashboard.css">`) {
		t.Error("the page does not link its assets relatively")
	}
	csp := `content="default-src 'none'; style-src 'self'; script-src 'self'; img-src data:; base-uri 'none'; form-action 'none'"`
	if !strings.Contains(page, csp) || strings.Contains(page, "unsafe-inline") {
		t.Error("the page lacks its strict Content-Security-Policy")
	}
	// Besides data: and fragments, the page references its two assets
	// and nothing else.
	for _, m := range regexp.MustCompile(`(?i)\s(?:src|href|xlink:href)="([^"]*)"`).FindAllStringSubmatch(page, -1) {
		switch ref := m[1]; {
		case strings.HasPrefix(ref, "data:"), strings.HasPrefix(ref, "#"), ref == "dashboard.css", ref == "dashboard.js":
		default:
			t.Errorf("the page references %q", ref)
		}
	}
}

// The assets are exactly the stylesheet and script the single file inlines,
// so the two modes cannot drift apart.
func TestTheAssetsAreWhatTheSingleFileInlines(t *testing.T) {
	single, linked := renderBoth(t)
	as := Assets()
	if len(as) != 2 || as[0].Name != "dashboard.css" || as[1].Name != "dashboard.js" {
		t.Fatalf("Assets() = %d asset(s)", len(as))
	}
	if !strings.Contains(single, "<style>"+as[0].Body+"</style>") {
		t.Error("the single file does not inline dashboard.css")
	}
	if !strings.Contains(single, "<script>"+as[1].Body+"</script>") {
		t.Error("the single file does not inline dashboard.js")
	}
	for _, a := range as {
		if strings.Contains(linked, a.Body) {
			t.Errorf("the linked page inlines %s", a.Name)
		}
		if strings.Contains(a.Body, "{{") || strings.Contains(strings.ToLower(a.Body), "</script") || strings.Contains(strings.ToLower(a.Body), "</style") {
			t.Errorf("%s holds template syntax or a closing tag", a.Name)
		}
	}
	// Outside the head, the two pages are the same markup.
	body := func(p string) string { return p[strings.Index(p, "</head>"):] }
	if body(single) != body(linked) {
		t.Error("the two modes render different bodies")
	}
}

// Skeletonic Stylus's core is vendored byte for byte as its 3.0.0 release
// ships it, and the stylesheet both modes serve opens with it, after the
// notice that carries its copyright and licence.
func TestSkeletonicIsVendoredUnmodified(t *testing.T) {
	onDisk, err := os.ReadFile("third_party/skeletonic/skeletonic.min.css")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(onDisk)
	if got := hex.EncodeToString(sum[:]); got != skeletonicSHA256 || string(onDisk) != skeletonicCSS {
		t.Fatalf("vendored Skeletonic is %s, want %s", got, skeletonicSHA256)
	}
	if !strings.HasPrefix(Assets()[0].Body, skeletonicNotice+skeletonicCSS+"\n") {
		t.Error("the stylesheet does not open with the Skeletonic core")
	}
	if !strings.Contains(skeletonicNotice, "MIT OR Apache-2.0") || !strings.Contains(skeletonicNotice, "Skeletonic CSS") {
		t.Errorf("the notice lacks the copyright or licence: %q", skeletonicNotice)
	}
}
