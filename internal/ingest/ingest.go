// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package ingest reads evidence into a graph: passmcp attestations, passmcp JSON
// reports and MCP client configurations. It only reads files; it opens no
// connection.
//
// Each input supplies different facts, and the graph accumulates them:
//
//   - an attestation supplies a server's score, grade, failing checks and the
//     attestation itself;
//   - a passmcp JSON report supplies the server's tools and their annotations,
//     how the server admits clients, and the identity passmcp authenticated as
//     with its scopes;
//   - a client configuration supplies the agents and which servers they use.
//
// Ingesting the same evidence twice leaves the graph unchanged.
package ingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"satellion.com/passmcp-reporting/graph"
)

// Result summarises one ingestion.
type Result struct {
	// Read counts the files ingested, by what they were.
	Read map[string]int `json:"read"`
	// Skipped names files that were not recognised or not applicable, with
	// the reason.
	Skipped []string `json:"skipped,omitempty"`
}

// Paths ingests every file named, and every .json file under every directory
// named, into g.
func Paths(g *graph.Graph, paths []string) (Result, error) {
	res := Result{Read: map[string]int{}}
	files, err := expand(paths)
	if err != nil {
		return res, err
	}
	for _, f := range files {
		kind, err := File(g, f)
		switch {
		case errors.Is(err, errSkip):
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s: %v", f, err))
		case err != nil:
			return res, fmt.Errorf("%s: %w", f, err)
		default:
			res.Read[kind]++
		}
	}
	return res, nil
}

// expand turns the named paths into a sorted list of files.
func expand(paths []string) ([]string, error) {
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".json") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

// errSkip marks a file that is valid JSON but not evidence this tool reads.
var errSkip = errors.New("not ingested")

// File ingests one file and returns what it was: "attestation", "report" or
// "config".
func File(g *graph.Graph, path string) (string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- the operator names the files to ingest
	if err != nil {
		return "", err
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return "", fmt.Errorf("%w: not a JSON object", errSkip)
	}
	switch {
	case probe["predicateType"] != nil && probe["_type"] != nil:
		return "attestation", attestationFile(g, path, raw)
	case probe["passmcp"] != nil && probe["target"] != nil && probe["phases"] != nil:
		return "report", reportFile(g, raw)
	case isConfig(probe):
		return "config", configFile(g, path, raw)
	}
	return "", fmt.Errorf("%w: neither an attestation, a passmcp report nor an MCP client configuration", errSkip)
}
