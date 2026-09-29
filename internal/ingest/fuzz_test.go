// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package ingest

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"satellion.com/passmcp-reporting/graph"
)

// secretShape is what the fuzzer's secret is limited to: long enough to be
// distinctive and made of characters that neither JSON nor a URL escapes,
// so a leak shows up byte for byte in the marshalled graph.
var secretShape = regexp.MustCompile(`^[A-Za-z0-9._~-]{8,64}$`)

// FuzzConfigNeverStoresASecret places one secret everywhere a client
// configuration can carry a credential (env, headers, the value after a
// secret flag, a --flag=value assignment, a URL's user information and a
// secret-named query parameter), next to a server name and command the
// fuzzer also chooses. The graph must stay valid, and the secret must occur
// in it no more often than it does when every one of those places holds a
// placeholder instead: any extra occurrence is a leak. It is the SG-02 guard
// with the secret's shape left to the fuzzer.
func FuzzConfigNeverStoresASecret(f *testing.F) {
	f.Add("shell", "npx", "sk-live-ENVSECRET123")
	f.Add("zed", "/usr/bin/tool", "ghp_barePATsecret0000")
	f.Add("", "", "abcdefghijklmnopqrstuvwxyz0123456789ABCD")
	f.Add("a.b", "tool --verbose", "-leading-dash")
	f.Add("x", "cmd", "--password")
	f.Add("stdio", "http", "with~tilde.and.dots")
	f.Fuzz(func(t *testing.T, name, command, secret string) {
		if !secretShape.MatchString(secret) {
			t.Skip("secret has a character JSON or a URL would escape")
		}
		placeholder := strings.Repeat("Q", len(secret))
		if secret == placeholder {
			placeholder = strings.Repeat("Z", len(secret))
		}
		got := ingested(t, secretConfig(t, name, command, secret))
		want := ingested(t, secretConfig(t, name, command, placeholder))
		if bytes.Count(got, []byte(secret)) > bytes.Count(want, []byte(secret)) {
			t.Fatalf("secret %q reached the graph:\n%s", secret, got)
		}
	})
}

// ingested reads a client configuration into a fresh graph and returns the
// graph marshalled, failing unless it validates.
func ingested(t *testing.T, config []byte) []byte {
	t.Helper()
	g := graph.New()
	if err := configFile(g, "claude_desktop_config.json", config); err != nil {
		t.Fatal(err)
	}
	out, err := g.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := graph.Parse(out); err != nil {
		t.Fatalf("ingested graph does not validate: %v\n%s", err, out)
	}
	return out
}

// secretConfig builds a Claude Desktop configuration with a remote and a
// local server, each holding secret in every place a credential can go.
func secretConfig(t *testing.T, name, command, secret string) []byte {
	t.Helper()
	servers := map[string]any{
		"remote": map[string]any{
			"url":     "https://operator:" + secret + "@crm.example/mcp?api_key=" + secret + "&team=blue",
			"headers": map[string]string{"Authorization": "Bearer " + secret},
		},
		name: map[string]any{
			"command": command,
			"args":    []string{"--api-key", secret, "--token=" + secret},
			"env":     map[string]string{"OPENAI_API_KEY": secret},
		},
	}
	b, err := json.Marshal(map[string]any{"mcpServers": servers})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
