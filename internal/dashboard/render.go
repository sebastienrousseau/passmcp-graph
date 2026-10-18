// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"math"
	"strconv"
	"strings"
)

//go:embed dashboard.html
var page string

// stylesheet and script are the page's CSS and JavaScript: inline in the
// single file, or written beside the page by dashboard --assets. One copy
// of each serves both, so the two modes cannot drift apart.
//
// The stylesheet is Skeletonic Stylus's core, byte for byte as its 3.0.0
// release ships it, then the page's own: the Control Centre tokens and
// components in dashboard.css, and the investigation panels, widths,
// motion, colour modes and print in dashboard-adapt.css. Skeletonic
// declares everything in @layer skeletonic.*, and the page's rules are
// unlayered, so the page wins every overlap whatever their order.
var stylesheet = skeletonicNotice + skeletonicCSS + "\n" + baseCSS + adaptCSS

// skeletonicCSS is css/core/skeletonic.min.css from
// @sebastienrousseau/skeletonic-stylus 3.0.0, unmodified; skeletonicSHA256
// pins it, and REUSE.toml records its licence.
//
//go:embed third_party/skeletonic/skeletonic.min.css
var skeletonicCSS string

// skeletonicSHA256 is the SHA-256 of the vendored Skeletonic core.
const skeletonicSHA256 = "116dcdcf51009485eb2dbb6f5c773856f437d972ef277b31e2ee4ef587e1fb09"

// skeletonicNotice carries Skeletonic's copyright and licence into every
// copy of the stylesheet, which the minified file itself does not.
const skeletonicNotice = "/*! Skeletonic Stylus 3.0.0 core (css/core/skeletonic.min.css), unmodified. " +
	"Copyright (c) 2023 - 2026 Skeletonic CSS. MIT OR Apache-2.0. https://skeletonic.com */\n"

//go:embed dashboard.css
var baseCSS string

//go:embed dashboard-adapt.css
var adaptCSS string

//go:embed dashboard.js
var script string

// Asset is a file a linked page loads from its own directory.
type Asset struct {
	Name, Body string
}

// Assets are the files RenderLinked's page links, by relative URL: its
// stylesheet and its script.
func Assets() []Asset {
	return []Asset{{"dashboard.css", stylesheet}, {"dashboard.js", script}}
}

// logoFile is passmcp's flame, the mark the family shares. The page inlines
// it, as the header mark and as a data: favicon, so it still loads nothing.
//
//go:embed logo.svg
var logoFile string

// logo is the SVG element alone, without the file's licence comments.
var logo = logoFile[strings.Index(logoFile, "<svg"):]

var tmpl = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"join": strings.Join,
	// #nosec G203 -- the embedded stylesheet above, not input
	"stylesheet": func() template.CSS { return template.CSS(stylesheet) },
	// #nosec G203 -- the embedded script above, not input
	"script": func() template.JS { return template.JS(script) },
	// #nosec G203 -- logo is the embedded file above, not input
	"logo": func() template.HTML { return template.HTML(logo) },
	// #nosec G203 -- a data: URL of the embedded logo, not input
	"favicon": func() template.URL {
		return template.URL("data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(logo)))
	},
	"steps": func(p []string) string { return strings.Join(p, " -> ") },
	"initial": func(s string) string {
		for _, r := range s {
			return strings.ToUpper(string(r))
		}
		return "?"
	},
	"add": func(a, b float64) float64 { return a + b },
	"plural": func(n int, one, many string) string {
		if n == 1 {
			return "1 " + one
		}
		return fmt.Sprintf("%d %s", n, many)
	},
	"commandNote": func() string { return CommandNote },
	"command":     Command,
	"serverIcon":  serverIcon,
	"clientIcon":  clientIcon,
	"display":     displayName,
	// Ring arcs share a 100-unit circumference, a small gap between them.
	"arcLen": func(v float64) string { return strconv.FormatFloat(math.Max(v-1.5, 0.6), 'f', 2, 64) },
	"arcGap": func(v float64) string { return strconv.FormatFloat(100-math.Max(v-1.5, 0.6), 'f', 2, 64) },
	"arcOff": func(off float64) string { return strconv.FormatFloat(-off, 'f', 2, 64) },
	"sev": func(rows []AttentionRow, sev string) int {
		n := 0
		for _, r := range rows {
			if r.Severity == sev {
				n++
			}
		}
		return n
	},
	"repro": func(client string) string { return ReproQuery(client, nil, nil) },
	"sum": func(xs ...int) int {
		t := 0
		for _, x := range xs {
			t += x
		}
		return t
	},
	"or0": func(s string) string {
		if s == "" {
			return "–"
		}
		return s
	},
}).Parse(page))

// view is what the template reads: the summary, and whether the page
// links its assets rather than inlining them.
type view struct {
	*Summary
	Linked bool
}

// Render writes s as one HTML page. html/template escapes every value, so a
// label read from a configuration file cannot inject markup.
func Render(w io.Writer, s *Summary) error {
	return tmpl.Execute(w, view{Summary: s})
}

// RenderLinked writes s as an HTML page that loads Assets from its own
// directory instead of inlining them, for a host whose
// Content-Security-Policy refuses inline script and style. Its own policy
// allows 'self' for both, and nothing inline.
func RenderLinked(w io.Writer, s *Summary) error {
	return tmpl.Execute(w, view{Summary: s, Linked: true})
}
