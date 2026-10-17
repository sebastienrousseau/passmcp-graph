// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed dashboard.html
var page string

// logoFile is passmcp's flame, the mark the family shares. The page inlines
// it, as the header mark and as a data: favicon, so it still loads nothing.
//
//go:embed logo.svg
var logoFile string

// logo is the SVG element alone, without the file's licence comments.
var logo = logoFile[strings.Index(logoFile, "<svg"):]

var tmpl = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"join": strings.Join,
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
	"repro":       func(client string) string { return ReproQuery(client, nil, nil) },
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

// Render writes s as one HTML page. html/template escapes every value, so a
// label read from a configuration file cannot inject markup.
func Render(w io.Writer, s *Summary) error {
	return tmpl.Execute(w, s)
}
