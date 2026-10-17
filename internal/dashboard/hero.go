// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

// Hero is the overview's figures: configured tool paths, the share of them
// by declared behaviour, and how the servers divide by status. The status
// distribution is drawn apart from the topology, so no chart implies that
// clients, paths and servers are one conserved quantity.
type Hero struct {
	// Paths is every configured client-to-tool path. Changing, Destructive
	// and Unannotated are the paths whose tool is not declared read-only,
	// is potentially destructive, or declares nothing, each out of Paths.
	Paths                              int
	Changing, Destructive, Unannotated Ratio
	// ReadOnly and Mutating are the paths whose tool declares read-only,
	// or declares it is not; with Unannotated they add up to Paths.
	ReadOnly, Mutating int
	// DeclaredDestructive and PresumedDestructive split Destructive.
	DeclaredDestructive, PresumedDestructive int
	// Attention is how many servers are critical, unattested or stale.
	Attention int
	Groups    []HeroGroup
}

// HeroGroup is one server status and its share of the servers. Key
// follows the count in the ring's key: "1 with stale evidence".
type HeroGroup struct {
	Status, Label, Note, Key string
	Count                    int
	Share                    Ratio
}

// heroGroups are the statuses in precedence order: a server counts once,
// in the first that applies.
var heroGroups = []struct{ status, label, note, key string }{
	{"critical", "Critical findings", "a critical check failed", "with a failed critical check"},
	{"unattested", "No evidence", "no attestation read", "with no evidence"},
	{"stale", "Stale evidence", "older than the threshold", "with stale evidence"},
	{"ok", "No failing server checks", "in fresh evidence", "with no failing server checks"},
}

func buildHero(s *Summary) Hero {
	h := Hero{Groups: make([]HeroGroup, 0, len(heroGroups))}
	for _, a := range s.Agents {
		h.Paths += a.Tools()
		h.Changing.Num += a.Writable
		h.Destructive.Num += a.Destructive
		h.Unannotated.Num += a.Unannotated
		h.ReadOnly += a.ReadOnly
		h.Mutating += a.Mutating()
		h.DeclaredDestructive += a.DeclaredDestructive
		h.PresumedDestructive += a.PresumedDestructive
	}
	h.Changing.Den, h.Destructive.Den, h.Unannotated.Den = h.Paths, h.Paths, h.Paths
	counts := map[string]int{}
	for _, sv := range s.Servers {
		counts[sv.Status]++
	}
	h.Attention = counts["critical"] + counts["unattested"] + counts["stale"]
	for _, g := range heroGroups {
		h.Groups = append(h.Groups, HeroGroup{Status: g.status, Label: g.label, Note: g.note, Key: g.key,
			Count: counts[g.status], Share: Ratio{counts[g.status], s.Totals.Servers}})
	}
	return h
}

// spread is the i-th of n points evenly between lo and hi, centred when
// there is one.
func spread(i, n int, lo, hi float64) float64 {
	if n <= 1 {
		return (lo + hi) / 2
	}
	return lo + (hi-lo)*float64(i)/float64(n-1)
}
