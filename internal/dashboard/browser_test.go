// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"bytes"
	"context"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"satellion.com/passmcp-graph/internal/fixture"
	"satellion.com/passmcp-graph/internal/policy"
)

var probeOut = regexp.MustCompile(`(?s)<pre id="out">(.*?)</pre>`)

// runProbe renders s beside testdata/probe.html, opens the probe in
// headless Chrome with reduced motion, and returns what the probe printed.
func runProbe(t *testing.T, chrome string, s *Summary) string {
	t.Helper()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "dashboard.html"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Render(f, s); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return probeDir(t, chrome, dir)
}

// probeDir copies testdata/probe.html into dir, which holds the page as
// dashboard.html, opens the probe in headless Chrome with reduced motion,
// and returns what the probe printed.
func probeDir(t *testing.T, chrome, dir string) string {
	t.Helper()
	probe, err := os.ReadFile("testdata/probe.html")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "probe.html"), probe, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, chrome, "--headless=new", "--disable-gpu", "--allow-file-access-from-files",
		"--force-prefers-reduced-motion", "--user-data-dir="+filepath.Join(dir, "profile"), "--virtual-time-budget=8000",
		"--dump-dom", "file://"+filepath.Join(dir, "probe.html"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// Chrome's helper processes can keep the output pipe open after Chrome
	// is killed; stop waiting for them rather than for the whole timeout.
	cmd.WaitDelay = 5 * time.Second
	out, err := cmd.Output()
	m := probeOut.FindSubmatch(out)
	switch {
	case m == nil && err != nil:
		t.Fatalf("chrome: %v\n%s", err, stderr.Bytes())
	case m == nil:
		t.Fatalf("the probe printed nothing:\n%s", out)
	case err != nil:
		// Headless Chrome sometimes dumps the page and then hangs while
		// shutting down until its teardown watchdog exits with status 2.
		// The probe's output is complete by then, so that is not a failure
		// of the page.
		t.Logf("chrome exited after printing the probe: %v", err)
	}
	return html.UnescapeString(string(m[1]))
}

// A Chrome that prints the probe and then fails to shut down, as its
// teardown watchdog does, still yields the probe's output.
func TestProbeSurvivesChromeExitingAfterOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in Chrome is a shell script")
	}
	chrome := filepath.Join(t.TempDir(), "chrome")
	script := "#!/bin/sh\necho '<pre id=\"out\">probe ran &amp; printed</pre>'\nexit 2\n"
	if err := os.WriteFile(chrome, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if got := runProbe(t, chrome, build(t, fixture.Graph(), []policy.Violation{})); got != "probe ran & printed" {
		t.Fatalf("runProbe = %q", got)
	}
}
