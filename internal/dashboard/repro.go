// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"fmt"
	"strings"
)

// Every filter on the page has an equivalent in the query language, so the
// page can print a command that returns the paths it shows. The fragments
// are written here, the page joins them exactly as ReproQuery does, and a
// test runs ReproQuery's output through the query engine for every
// combination and compares it with the paths the filters keep.

// reproBase is the query every path-view command starts from.
const reproBase = "agents -> tools"

// statusQuery is each server-status tag in the query language.
func statusQuery(staleDays int) map[string]string {
	return map[string]string{
		"st-critical":   "server.critical",
		"st-unattested": "not server.attested",
		"st-stale":      fmt.Sprintf("server.attestation_age_days > %d", staleDays),
		"st-ok":         fmt.Sprintf("(server.attested and not server.critical and server.attestation_age_days <= %d)", staleDays),
	}
}

// behaviourQuery is each tool-behaviour tag in the query language.
var behaviourQuery = map[string]string{
	"destr": "destructive",
	"unann": "not annotated",
}

// clientQuery selects one client's paths by its configuration file, or is
// empty when the file cannot be quoted in the query language.
func clientQuery(a AgentRow) string {
	if a.ConfigFile == "" || strings.ContainsAny(a.ConfigFile, `'"`) {
		return ""
	}
	return fmt.Sprintf("agent.config = '%s'", a.ConfigFile)
}

// ReproQuery joins a client fragment and the status and behaviour
// fragments the way the page does: statuses are alternatives, behaviours
// are alternatives, and the groups must all hold.
func ReproQuery(client string, statuses, behaviours []string) string {
	var groups []string
	if client != "" {
		groups = append(groups, client)
	}
	for _, alts := range [][]string{statuses, behaviours} {
		switch len(alts) {
		case 0:
		case 1:
			groups = append(groups, alts[0])
		default:
			groups = append(groups, "("+strings.Join(alts, " or ")+")")
		}
	}
	if len(groups) == 0 {
		return reproBase
	}
	return reproBase + " where " + strings.Join(groups, " and ")
}

// Command wraps a query for the shell, as the page prints it.
func Command(q string) string { return `passmcp-graph query "` + q + `"` }
