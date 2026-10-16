// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package dashboard

import (
	"fmt"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// The share-safe profile replaces what identifies an operator's estate
// before anything is rendered: server endpoints and commands, identity
// labels, client IDs and issuers, configuration directories, and anything
// left that looks like a local path, an email address or a token. What
// remains is what the page needs to be read: client products, tool names,
// check IDs, the names servers report, grades and counts.
var (
	localPath = regexp.MustCompile(`(?:/Users/|/home/|/root/|[A-Za-z]:\\Users\\)[^\s"'<>]*`)
	email     = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	// token matches common secret prefixes, JWTs, and long opaque runs. A
	// 40 or 64 hex-digit run is a commit or SHA-256 digest, kept as
	// evidence.
	token     = regexp.MustCompile(`(?:sk|pk|rk)-[A-Za-z0-9_-]{12,}|gh[pousr]_[A-Za-z0-9]{20,}|xox[abprs]-[A-Za-z0-9-]{10,}|AKIA[A-Z0-9]{16}|eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}|[A-Za-z0-9+/_=-]{32,}`)
	hexDigest = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)
)

// shareSafe rewrites every string in s, then lays the topology out again
// so no label clipped before redaction survives.
func shareSafe(s *Summary) {
	r := redactor(s)
	walk(reflect.ValueOf(s), r)
	s.ShareSafe = true
	addTags(s)
	s.Topology = buildTopology(s)
}

// redactor maps each identifying value to a stable placeholder, longest
// value first so an endpoint is replaced before a shorter label inside it.
func redactor(s *Summary) func(string) string {
	m := map[string]string{}
	for i, sv := range s.Servers {
		m[sv.Endpoint] = fmt.Sprintf("%s-server-%d", orDefault(sv.Transport, "mcp"), i+1)
	}
	for i, id := range s.Identities {
		m[id.Label] = fmt.Sprintf("identity-%d", i+1)
		m[id.ClientID] = fmt.Sprintf("identity-%d", i+1)
		m[id.Issuer] = fmt.Sprintf("issuer-%d", i+1)
	}
	for _, a := range s.Agents {
		if a.ConfigFile != "" {
			m[a.ConfigFile] = path.Base(strings.ReplaceAll(a.ConfigFile, `\`, "/"))
		}
	}
	delete(m, "")
	values := make([]string, 0, len(m))
	for v := range m {
		values = append(values, v)
	}
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	res := make([]*regexp.Regexp, len(values))
	for i, v := range values {
		res[i] = regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(v) + `($|[^A-Za-z0-9_])`)
	}
	return func(str string) string {
		for i, re := range res {
			repl := "${1}" + strings.ReplaceAll(m[values[i]], "$", "$$") + "${2}"
			// Twice, because adjacent matches share a boundary character.
			str = re.ReplaceAllString(re.ReplaceAllString(str, repl), repl)
		}
		return scrub(str)
	}
}

// scrub replaces what no mapping named but still looks private.
func scrub(s string) string {
	s = localPath.ReplaceAllString(s, "<local path>")
	s = email.ReplaceAllString(s, "<email>")
	return token.ReplaceAllStringFunc(s, func(t string) string {
		if hexDigest.MatchString(t) {
			return t
		}
		return "<redacted>"
	})
}

// walk applies f to every settable string reachable from v.
func walk(v reflect.Value, f func(string) string) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			walk(v.Elem(), f)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				walk(v.Field(i), f)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			walk(v.Index(i), f)
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(f(v.String()))
		}
	}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
