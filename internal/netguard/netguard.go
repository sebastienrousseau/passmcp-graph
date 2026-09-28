// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package netguard is --offline: it replaces the process's default HTTP
// transport with one that refuses every request, so nothing reached through
// net/http — this tool's code or a library's — can leave the machine.
//
// passmcp-graph has no connectors yet, so none of its commands make a request
// in the first place; the guard is what makes that a promise rather than an
// observation.
package netguard

import (
	"errors"
	"net/http"
)

// ErrOffline is returned for every request made while the guard is on.
var ErrOffline = errors.New("offline: passmcp-graph was asked to contact nothing")

type deny struct{}

// RoundTrip refuses the request.
func (deny) RoundTrip(*http.Request) (*http.Response, error) { return nil, ErrOffline }

// Deny installs the guard and returns a function that removes it.
func Deny() (restore func()) {
	prev := http.DefaultTransport
	http.DefaultTransport = deny{}
	return func() { http.DefaultTransport = prev }
}
