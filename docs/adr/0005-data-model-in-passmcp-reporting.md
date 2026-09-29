<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# 0005: The data model and store format live in passmcp-reporting

**Status:** Accepted

## Context

Gateways, registries and GRC tools should be able to read the graph
without taking a GPL-3.0-only dependency on passmcp-graph, and the graph's
JSON format needs one versioned schema that export and import both check.

## Decision

Node and edge kinds, `graph.ServerID`, the store (`graph.json`, loaded and
saved by `graph.Load` and `graph.Save`) and the JSON Schema for
`https://satellion.com/graph/v1` are the `graph` package of
[passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting),
under Apache-2.0. passmcp-graph requires it at the family's lockstep
version.

## Consequences

- A new node or edge kind is proposed in passmcp-reporting first.
- The format changes only by a new version URL, so a consumer never reads
  a changed format under an old name.
- `scripts/verify-release-versions.sh` fails when `go.mod` requires
  passmcp-reporting at a version other than this repository's.
