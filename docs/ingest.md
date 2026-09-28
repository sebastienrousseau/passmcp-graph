<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Ingesting evidence

`passmcp-graph ingest PATH...` reads each file named, and every `*.json`
file under each directory named. It recognises three kinds of input by
their shape, not their name, and skips anything else with a line on
stderr. Ingest is idempotent: the same evidence twice leaves the graph
byte for byte the same.

## What each input supplies

| Input | Recognised by | Supplies |
| :--- | :--- | :--- |
| passmcp attestation (in-toto statement) | `_type` and `predicateType` | The server, its score, grade, failing checks and their severities, and an attestation node linked by `attested_by` |
| passmcp JSON report (`passmcp check --output json`) | `passmcp`, `target` and `phases` | The server's name and version, its `auth` (`none`, `oauth`, `bearer` or `unknown`), its tools and their annotations, and the identity passmcp authenticated as, with its scopes |
| MCP client configuration | a `mcpServers`, `servers`, `context_servers` or `mcp.servers` object | An agent per configuration file, and a `uses` edge to each configured server, citing the file and key |

A server is identified the way passmcp's attestations identify it: by
transport and endpoint. A remote server's endpoint is its URL; a
server started as a program has transport `stdio` and the command line as
its endpoint. So a configuration entry and an attestation about the same
server meet on one node.

An attestation from another passmcp predicate, such as an A2A Agent Card
statement, is skipped. An attestation alone sets `auth` only to `none`,
and only when the run used no credentials and was not blocked; the other
values come from reports.

### Newer evidence wins

A server's score, grade and failing checks come from its newest
attestation, by `ranAt`. Ingesting an older one afterwards adds its
attestation node but leaves the verdict alone. A newer report replaces
the server's tool list: a tool it no longer lists stops being exposed.

### Clients

The agent's client is inferred from the file's path and key:
`claude-desktop` (a path containing "claude"), `cursor`, `zed`
(`context_servers`), `vscode` (`servers` or `mcp.servers`), or
`mcp-client`.

## What is never stored

A client configuration is where people keep credentials, so ingest reads
as little of it as it can:

- `env` and `headers` are never read.
- A URL loses its user information, and any query parameter whose name
  mentions a token, key, secret, password, signature, auth or credential
  has its value replaced with `<redacted>`.
- In a command line, the value after a flag such as `--api-key` or
  `--token`, the value in `--token=...`, and any argument shaped like a
  token (`sk-`, `ghp_`, `github_pat_`, `xox?-`, `AKIA`, `glpat-`,
  `Bearer` and a space, or 32 or more letters, digits, `_` and `-`) become
  `<redacted>`.

Redaction changes the endpoint, so a server configured with a secret in
its URL meets its attestation only when the attestation recorded the same
redacted form.
