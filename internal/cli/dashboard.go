// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"satellion.com/passmcp-graph/internal/dashboard"
	"satellion.com/passmcp-graph/internal/policy"
	"satellion.com/passmcp-reporting/graph"
)

type dashboardFlags struct {
	format, policy, out string
	stale               int
	open, shareSafe     bool
	// assets writes the page's stylesheet and script beside --out and
	// links them, for a host whose Content-Security-Policy refuses inline
	// code.
	assets bool
}

func parseDashboard(args []string, env Env) (dashboardFlags, error) {
	var fl dashboardFlags
	if _, err := subFlags("dashboard", args, env, func(fs *flag.FlagSet) {
		fs.StringVar(&fl.format, "format", "html", "")
		fs.StringVar(&fl.policy, "policy", "", "")
		fs.StringVar(&fl.out, "out", "", "")
		fs.IntVar(&fl.stale, "stale-days", 30, "")
		fs.BoolVar(&fl.open, "open", false, "")
		fs.BoolVar(&fl.shareSafe, "share-safe", false, "")
		fs.BoolVar(&fl.assets, "assets", false, "")
	}); err != nil {
		return fl, err
	}
	switch {
	case fl.format != "html" && fl.format != "json":
		return fl, fmt.Errorf("--format is %q, want html or json", fl.format)
	case fl.stale < 0:
		return fl, fmt.Errorf("--stale-days is %d, want 0 or more", fl.stale)
	case fl.open && fl.format != "html":
		return fl, errors.New("--open shows the HTML page; drop --format json")
	}
	return fl, checkAssets(fl)
}

// checkAssets refuses --assets where it cannot apply: only the HTML page
// links a stylesheet and a script, they are written beside --out, and
// neither may replace the page itself.
func checkAssets(fl dashboardFlags) error {
	switch {
	case !fl.assets:
		return nil
	case fl.format != "html":
		return errors.New("--assets writes the HTML page and its stylesheet and script; drop --format json")
	case fl.out == "":
		return errors.New("--assets needs --out: the stylesheet and script are written beside the page")
	}
	for _, a := range dashboard.Assets() {
		if strings.EqualFold(filepath.Base(fl.out), a.Name) {
			return fmt.Errorf("--out would be overwritten by an asset: --assets writes %s beside the page", a.Name)
		}
	}
	return nil
}

// runDashboard writes the page to stdout, or to --out. With --open and no
// --out it writes a temporary file, then hands the file to the system's
// opener: a local program, not a network request.
func runDashboard(store string, args []string, env Env) (int, error) {
	fl, err := parseDashboard(args, env)
	if err != nil {
		return ExitError, err
	}
	s, err := summarise(store, fl, env)
	if err != nil {
		return ExitError, err
	}
	if fl.out == "" && !fl.open {
		return ExitOK, writeSummary(env.Stdout, fl.format, s)
	}
	path, err := writeFile(fl, s)
	if err != nil {
		return ExitError, err
	}
	reportWritten(env, fl, path)
	if !fl.open {
		return ExitOK, nil
	}
	open := env.Open
	if open == nil {
		open = openWithSystem
	}
	if err := open(path); err != nil {
		return ExitError, fmt.Errorf("opening %s: %w", path, err)
	}
	return ExitOK, nil
}

// reportWritten names on stderr each file the command wrote.
func reportWritten(env Env, fl dashboardFlags, path string) {
	say(env.Stderr, "wrote %s\n", path)
	if fl.assets {
		for _, a := range dashboard.Assets() {
			say(env.Stderr, "wrote %s\n", filepath.Join(filepath.Dir(path), a.Name))
		}
	}
}

func writeSummary(w io.Writer, f string, s *dashboard.Summary) error {
	if f == "json" {
		return writeJSON(w, s)
	}
	return dashboard.Render(w, s)
}

// writeFile writes the summary to --out, or to a new temporary file, readable
// by the operator alone: the page names their configuration files. With
// --assets, the page links its stylesheet and script, written beside it
// the same way.
func writeFile(fl dashboardFlags, s *dashboard.Summary) (string, error) {
	var f *os.File
	var err error
	if fl.out == "" {
		f, err = os.CreateTemp("", "passmcp-graph-dashboard-*.html")
	} else {
		f, err = createPrivate(fl.out)
	}
	if err != nil {
		return "", err
	}
	werr := writePage(f, fl, s)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil && fl.assets {
		werr = writeAssets(filepath.Dir(f.Name()))
	}
	return f.Name(), werr
}

func writePage(w io.Writer, fl dashboardFlags, s *dashboard.Summary) error {
	if fl.assets {
		return dashboard.RenderLinked(w, s)
	}
	return writeSummary(w, fl.format, s)
}

// createPrivate creates or truncates path, readable by the operator alone.
func createPrivate(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- the operator names the output file
}

// writeAssets writes the linked page's stylesheet and script into dir, the
// page's own directory, under their fixed names: nowhere else.
func writeAssets(dir string) error {
	for _, a := range dashboard.Assets() {
		f, err := createPrivate(filepath.Join(dir, a.Name))
		if err != nil {
			return err
		}
		_, werr := io.WriteString(f, a.Body)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return werr
		}
	}
	return nil
}

// summarise builds the dashboard summary, checking the policy file when one
// is named. A violation is shown on the page, not reported as an exit status:
// the page is a report, and `check` is the gate.
func summarise(store string, fl dashboardFlags, env Env) (*dashboard.Summary, error) {
	o := dashboard.Options{Now: env.Now(), StaleDays: fl.stale, Version: Version, ShareSafe: fl.shareSafe}
	if fl.policy != "" {
		vs, err := violations(store, fl.policy, env)
		if err != nil {
			return nil, err
		}
		o.Violations = append([]policy.Violation{}, vs...)
	}
	g, err := graph.Load(store)
	if err != nil {
		return nil, err
	}
	return dashboard.Build(g, o)
}

// opener is the program each OS uses to show a file in its default
// application.
func opener(goos, path string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{path}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", path}
	}
	return "xdg-open", []string{path}
}

// openWithSystem starts the system's opener on path and does not wait for
// the browser.
func openWithSystem(path string) error {
	name, args := opener(runtime.GOOS, path)
	return exec.CommandContext(context.Background(), name, args...).Start() // #nosec G204 -- a fixed opener on a file this command wrote
}
