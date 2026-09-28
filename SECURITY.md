<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Security Policy

passmcp-graph reads MCP client configurations, which is where people keep
credentials, and evidence written by servers nobody has vetted. Its
security posture is about three things: it stores no secret it reads, it
makes no network request, and it treats every input as untrusted.

## Reporting a Vulnerability

Report security issues through [GitHub's private vulnerability reporting](https://github.com/sebastienrousseau/passmcp-graph/security/advisories/new). Do not open a public issue.

You will receive an acknowledgement within **72 hours**. A confirmed
vulnerability is fixed and released within **90 days** of the report, or
sooner when a fix is straightforward; if the window cannot be met you will
be told why and given a revised date.

A way to make a credential from a configuration reach the store or any
output, or to make passmcp-graph open a network connection, is a
vulnerability.

## Supported Versions

Only the latest release is supported.

## Security Measures

Each item names the test that enforces it, so the claim can be checked
rather than taken on trust.

- **No secrets stored.** `env` and `headers` are never read; URL user
  information is dropped; secret-named query parameters, flag values and
  token-shaped arguments are replaced with `<redacted>`.
  `TestConfigsBecomeAgentsAndNoSecretIsStored` in
  `internal/ingest/ingest_test.go` seeds nine kinds of secret and searches
  the saved store for each.
- **No network.** `TestNoCommandContactsTheNetwork` in
  `internal/cli/cli_test.go` runs every command, with and without
  `--offline`, while recording the default HTTP transport and a proxy
  listener every proxy variable points at, and requires zero of both.
  `TestOnlyTheOfflineGuardImportsTheNetwork` fails when any package the
  binary links, other than the offline guard, imports `net` or `net/http`.
- **Imports are validated first.** `import` checks a file against the
  graph schema and the data model's own validation before reading any of
  it. `TestExportEscapesAndRefuses` in `internal/export/export_test.go`.
- **Exports are escaped.** GraphML text is XML-escaped and Cypher strings
  are quoted, so a tool name chosen by a server cannot break out of
  either. `TestExportEscapesAndRefuses`.
- **No telemetry.**
- **Supply chain.** Two direct dependencies, `passmcp-reporting` (the data
  model) and `go.yaml.in/yaml/v3` (the policy file). CI runs
  `govulncheck` on every push and CodeQL on every pull request and weekly.
