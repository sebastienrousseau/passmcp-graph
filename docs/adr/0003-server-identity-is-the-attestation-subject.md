<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# 0003: A server's node is identified by passmcp's attestation subject

**Status:** Accepted

## Context

The same server arrives from three inputs: an attestation, a JSON report
and the client configurations that use it. Unless all three name it the
same way, the graph holds three disconnected nodes and every path through
the server is lost.

## Decision

A server is `graph.ServerID(transport, endpoint)` from passmcp-reporting,
the digest passmcp uses as an attestation's subject. Every input computes
it from the transport and endpoint it carries; no input normalises
endpoints its own way.

## Consequences

- Evidence about one server meets on one node whichever input supplied
  it.
- A change to endpoint normalisation belongs in passmcp-reporting, where
  passmcp and passmcp-graph both pick it up, never in one ingest path.
