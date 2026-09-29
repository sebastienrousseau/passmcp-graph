// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package cli

import (
	"flag"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionListsEveryCommandAndFlag(t *testing.T) {
	fs, _, _ := newGlobalFlags(Env{Stderr: io.Discard})
	var defined []string
	fs.VisitAll(func(f *flag.Flag) { defined = append(defined, "--"+f.Name) })
	if strings.Join(defined, " ") != "--offline --store" {
		t.Fatalf("Run defines %v; update globalFlags and the zsh and fish scripts", defined)
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		r := invoke("completion", shell)
		if r.code != ExitOK || r.stderr != "" {
			t.Fatalf("completion %s: code %d, stderr %q", shell, r.code, r.stderr)
		}
		for name := range commands {
			if !strings.Contains(r.stdout, name) {
				t.Errorf("completion %s does not offer %q", shell, name)
			}
		}
		for _, f := range globalFlags {
			if !strings.Contains(r.stdout, strings.TrimPrefix(f, "--")) {
				t.Errorf("completion %s does not offer %s", shell, f)
			}
		}
	}
}

func TestCompletionRefusesAnUnknownShell(t *testing.T) {
	for _, args := range [][]string{{"completion"}, {"completion", "tcsh"}, {"completion", "bash", "zsh"}} {
		r := invoke(args...)
		if r.code != ExitError || r.stdout != "" || !strings.Contains(r.stderr, "completion") {
			t.Errorf("%v: code %d, stdout %q, stderr %q", args, r.code, r.stdout, r.stderr)
		}
	}
}

// Each script is syntax-checked by its own shell when that shell is
// installed; bash is on every CI runner but Windows.
func TestCompletionScriptsParse(t *testing.T) {
	checks := map[string][]string{"bash": {"bash", "-n"}, "zsh": {"zsh", "-n"}, "fish": {"fish", "--no-execute"}}
	for shell, argv := range checks {
		bin, err := exec.LookPath(argv[0])
		if err != nil {
			t.Logf("%s not installed; skipping its syntax check", argv[0])
			continue
		}
		p := put(t, t.TempDir(), "completion."+shell, invoke("completion", shell).stdout)
		out, err := exec.Command(bin, append(argv[1:], filepath.Clean(p))...).CombinedOutput() // #nosec G204 -- fixed shells, a temp file
		if err != nil {
			t.Errorf("%s rejects its completion script: %v\n%s", shell, err, out)
		}
	}
}
