// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"context"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"
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
	probe, err := os.ReadFile("testdata/probe.html")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "probe.html"), probe, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, chrome, "--headless=new", "--disable-gpu", "--allow-file-access-from-files",
		"--force-prefers-reduced-motion", "--user-data-dir="+filepath.Join(dir, "profile"), "--virtual-time-budget=8000",
		"--dump-dom", "file://"+filepath.Join(dir, "probe.html")).Output()
	if err != nil {
		t.Fatalf("chrome: %v", err)
	}
	m := probeOut.FindSubmatch(out)
	if m == nil {
		t.Fatalf("the probe printed nothing:\n%s", out)
	}
	return html.UnescapeString(string(m[1]))
}
