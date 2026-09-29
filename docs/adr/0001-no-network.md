<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# 0001: passmcp-graph makes no network request, and `--offline` enforces it

**Status:** Accepted

## Context

passmcp-graph reads MCP client configurations, where people keep
credentials, and evidence about servers nobody has vetted. A tool that
holds both and can open a connection is a way for either to leave the
machine. Operators also run it in CI, where egress is often restricted.

## Decision

No command makes a network request. Only `internal/netguard` may import
`net` or `net/http`; `--offline` replaces the process's default HTTP
transport with one that refuses every request, so a library that reached
for the network would fail rather than succeed.

## Consequences

- The graph holds only what the operator gives it. Discovering servers,
  or reading an identity provider or a SaaS inventory, is out of scope
  until a connector is designed; a connector is a decision for the
  maintainer, recorded in a new ADR, not a drive-by import.
- `TestNoCommandContactsTheNetwork` records the default transport and a
  proxy listener while every command runs, and requires zero requests;
  `TestOnlyTheOfflineGuardImportsTheNetwork` fails on any other package
  that imports the network.
