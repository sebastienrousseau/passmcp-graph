// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"fmt"
	"strings"

	"satellion.com/passmcp-graph/internal/policy"
)

// Finding is one finding occurrence as its details panel shows it: a
// readable title over the raw rule or check ID, the path, where the
// evidence comes from, what is not known, and a read-only next step.
type Finding struct {
	Anchor      string `json:"anchor"`
	Kind        string `json:"kind"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Rule        string `json:"rule"`
	Path        string `json:"path"`
	Evidence    string `json:"evidence"`
	Uncertainty string `json:"uncertainty"`
	Next        string `json:"next"`
	Server      Ref    `json:"server"`
	Clients     []Ref  `json:"clients"`
	Tags        string `json:"-"`
	Text        string `json:"-"`
}

const configuredReach = "Reach is configured, not observed: no tool call is recorded."

func buildFindings(s *Summary) []Finding {
	byEndpoint, byName := map[string]ServerRow{}, map[string]Ref{}
	for _, sv := range s.Servers {
		byEndpoint[sv.Endpoint] = sv
	}
	for _, a := range s.Agents {
		byName[a.Name] = Ref{a.Name, a.Anchor}
	}
	out := []Finding{}
	for _, r := range s.Risks {
		sv := byEndpoint[r.Server]
		out = append(out, Finding{Kind: "inherited-critical", Severity: "critical",
			Title: fmt.Sprintf("A critical check failed on %s, inherited by %s", nameOr(sv, r.Server), r.Agent),
			Rule:  r.Finding, Path: strings.Join(r.Path, " -> "), Evidence: attestationEvidence(sv),
			Uncertainty: "The check result is passmcp's, from the server's latest attestation. " + configuredReach,
			Next:        `passmcp-graph query "agents -> servers where critical"`,
			Server:      serverRef(sv), Clients: []Ref{byName[r.Agent]}})
	}
	for _, v := range s.Violations {
		out = append(out, violationFinding(v, byEndpoint, byName))
	}
	for _, f := range s.OverPrivileged {
		sv := byEndpoint[f.Server]
		out = append(out, Finding{Kind: "over-privileged", Severity: "warning",
			Title: fmt.Sprintf("Identity %s holds more than %s's tools need", f.Identity, nameOr(sv, f.Server)),
			Rule:  f.Flag, Path: f.Identity + " -> " + f.Server,
			Evidence:    "Scopes " + strings.Join(f.Scopes, ", ") + " on a server whose every tool declares readOnlyHint true.",
			Uncertainty: "Read-only is what the server declares about its tools, not observed behaviour; the scopes are as recorded at ingest.",
			Next:        scopeQuery(f.Scopes), Server: serverRef(sv), Clients: sv.Clients})
	}
	for i := range out {
		out[i].Anchor = fmt.Sprintf("fnd-%d", i)
		out[i].Clients = nonNil(out[i].Clients)
	}
	return out
}

func violationFinding(v policy.Violation, byEndpoint map[string]ServerRow, byName map[string]Ref) Finding {
	f := Finding{Kind: "policy", Severity: "critical", Rule: v.Rule, Path: v.Path.String(),
		Title:       fmt.Sprintf("Policy rule %s forbids this path", v.Rule),
		Evidence:    orDefault(v.Reason, "A forbid rule in the policy file passed with --policy matched this path."),
		Uncertainty: "The path is derived from configuration and evidence. " + configuredReach,
		Next:        "passmcp-graph check --policy FILE", Clients: []Ref{}}
	for _, st := range v.Path {
		if sv, ok := byEndpoint[st.Label]; ok && st.Kind == "server" {
			f.Server = serverRef(sv)
		}
		if c, ok := byName[st.Label]; ok && st.Kind == "agent" {
			f.Clients = append(f.Clients, c)
		}
	}
	return f
}

// serverRef links to a server under the name the rest of the page gives
// it; see displayName.
func serverRef(sv ServerRow) Ref { return Ref{displayName(sv), sv.Anchor} }

// nameOr is a server's display name, or label when the server is not in
// the summary.
func nameOr(sv ServerRow, label string) string { return orDefault(displayName(sv), label) }

func attestationEvidence(sv ServerRow) string {
	if sv.Attestation == "" {
		return "No attestation on record."
	}
	return fmt.Sprintf("Attestation %s, run at %s.", shortDigest(sv.Attestation), orDefault(sv.RanAt, "an unknown time"))
}

func shortDigest(d string) string {
	if len(d) > 12 {
		return d[:12] + "…"
	}
	return d
}

// scopeQuery finds every identity holding the first wide scope, when it
// can be quoted in the query language.
func scopeQuery(scopes []string) string {
	if len(scopes) == 0 || strings.ContainsAny(scopes[0], `'"`) {
		return "passmcp-graph analyze"
	}
	return fmt.Sprintf(`passmcp-graph query "identities -> servers where scopes contains '%s'"`, scopes[0])
}
