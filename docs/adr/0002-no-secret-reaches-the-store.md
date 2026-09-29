<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# 0002: Client configurations are read for names, URLs and commands only

**Status:** Accepted

## Context

Claude Desktop, Cursor, VS Code and Zed configurations carry tokens in
`env`, in `headers`, in URL user information, in query parameters and in
command-line arguments. The graph needs to know which agent uses which
server, not how it authenticates.

## Decision

Ingest reads a configured server's name, URL and command and nothing else.
`env` and `headers` are absent from the type it decodes into, so they are
never read. URL user information is dropped; secret-named query parameters,
flag values and token-shaped arguments are replaced with `<redacted>`.

## Consequences

- The store can be shared, committed or exported without leaking a
  credential from the configurations it was built from.
- A change that reads `env` or `headers`, or stops redacting an argument,
  is the most damaging change available here.
  `TestConfigsBecomeAgentsAndNoSecretIsStored` seeds nine kinds of secret
  and searches the saved store for each; a new secret shape extends it.
