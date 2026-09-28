// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package ingest

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"satellion.com/passmcp-reporting/graph"
)

// A client configuration names MCP servers under one of these keys:
// mcpServers (Claude Desktop, Cursor and most clients), servers (VS Code's
// mcp.json), mcp.servers (VS Code's settings.json) and context_servers (Zed).
func isConfig(probe map[string]json.RawMessage) bool {
	_, servers := serverSection(probe)
	return servers != nil
}

// serverSection returns the key path and the raw server map of a client
// configuration, or nil when the file has none.
func serverSection(probe map[string]json.RawMessage) (string, map[string]json.RawMessage) {
	for _, key := range []string{"mcpServers", "servers", "context_servers"} {
		if m := asMap(probe[key]); m != nil {
			return key, m
		}
	}
	if mcp := asMap(probe["mcp"]); mcp != nil {
		if m := asMap(mcp["servers"]); m != nil {
			return "mcp.servers", m
		}
	}
	return "", nil
}

func asMap(raw json.RawMessage) map[string]json.RawMessage {
	if raw == nil {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// entry is a configured server. Env and headers are deliberately absent:
// their values are credentials more often than not, so they are never read
// into the graph at all.
type entry struct {
	Command json.RawMessage `json:"command"`
	Args    []string        `json:"args"`
	URL     string          `json:"url"`
	Server  string          `json:"serverUrl"`
	Type    string          `json:"type"`
}

// configFile ingests a client configuration: one agent for the file, linked
// by uses to each server it configures.
func configFile(g *graph.Graph, path string, raw []byte) error {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return err
	}
	key, servers := serverSection(probe)
	client := clientOf(path, key)
	aid := graph.AgentID(client, path, client)
	g.Upsert(graph.Node{ID: aid, Kind: graph.KindAgent, Label: client + " (" + filepath.Base(path) + ")",
		Agent: &graph.AgentProps{Client: client, ConfigFile: path}})
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		transport, endpoint, ok := target(servers[name])
		if !ok {
			continue
		}
		sid := graph.ServerID(transport, endpoint)
		g.Upsert(graph.Node{ID: sid, Kind: graph.KindServer, Label: endpoint,
			Server: &graph.ServerProps{Transport: transport, Endpoint: endpoint}})
		g.Link(graph.Edge{From: aid, To: sid, Kind: graph.Uses, Ref: &graph.ConfigRef{File: path, Key: key + "." + name}})
	}
	return nil
}

// clientOf names the client product from the file and the key it used.
func clientOf(path, key string) string {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(p, "claude"):
		return "claude-desktop"
	case strings.Contains(p, "cursor"):
		return "cursor"
	case key == "context_servers" || strings.Contains(p, "zed"):
		return "zed"
	case key == "servers" || key == "mcp.servers" || strings.Contains(p, "vscode"):
		return "vscode"
	}
	return "mcp-client"
}

// target derives the transport and endpoint passmcp would record for a
// configured server: the URL for a remote one, the command line for a child
// process. Secrets that a configuration puts in either are redacted first.
func target(raw json.RawMessage) (string, string, bool) {
	var e entry
	if json.Unmarshal(raw, &e) != nil {
		return "", "", false
	}
	if u := firstNonEmpty(e.URL, e.Server); u != "" {
		return "http", redactURL(u), true
	}
	cmd, args := command(e)
	if cmd == "" {
		return "", "", false
	}
	return "stdio", strings.Join(append([]string{cmd}, redactArgs(args)...), " "), true
}

// command reads the command, which Zed nests as {"path", "args"}.
func command(e entry) (string, []string) {
	var s string
	if json.Unmarshal(e.Command, &s) == nil {
		return s, e.Args
	}
	var nested struct {
		Path string   `json:"path"`
		Args []string `json:"args"`
	}
	if json.Unmarshal(e.Command, &nested) == nil && nested.Path != "" {
		return nested.Path, nested.Args
	}
	return "", nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// Redacted replaces a secret that would otherwise be copied into the graph.
const Redacted = "<redacted>"

var (
	secretFlag   = regexp.MustCompile(`(?i)^--?(api[-_]?key|token|access[-_]?token|secret|password|passwd|auth|authorization|key|pat|bearer)$`)
	secretAssign = regexp.MustCompile(`(?i)^(--?(?:api[-_]?key|token|access[-_]?token|secret|password|passwd|auth|authorization|key|pat|bearer)=).+`)
	secretValue  = regexp.MustCompile(`^(sk-|ghp_|gho_|github_pat_|xox[abpr]-|AKIA|glpat-|Bearer\s)|^[A-Za-z0-9_\-]{32,}$`)
	secretParam  = regexp.MustCompile(`(?i)(token|key|secret|password|passwd|sig|signature|auth|credential)`)
)

// redactArgs masks arguments that carry a secret: the value after a flag such
// as --api-key, a --token=value assignment, and a value that looks like a
// token on its own.
func redactArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		switch {
		case i > 0 && secretFlag.MatchString(args[i-1]):
			out[i] = Redacted
		case secretAssign.MatchString(a):
			out[i] = secretAssign.ReplaceAllString(a, "${1}"+Redacted)
		case secretValue.MatchString(a):
			out[i] = Redacted
		default:
			out[i] = a
		}
	}
	return out
}

// redactURL drops credentials from a URL: its user information and the value
// of any query parameter named like a secret.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return Redacted
	}
	u.User = nil
	q := u.Query()
	changed := false
	for k := range q {
		if secretParam.MatchString(k) {
			q.Set(k, Redacted)
			changed = true
		}
	}
	if changed {
		u.RawQuery = q.Encode()
	}
	return u.String()
}
