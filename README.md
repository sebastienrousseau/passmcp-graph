<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

<p align="center">
  <img src="https://raw.githubusercontent.com/sebastienrousseau/passmcp/main/.github/logo.svg" alt="passmcp-graph logo" width="128" />
</p>

<h1 align="center">passmcp-graph</h1>

<p align="center">
  A local graph of which agents use which MCP servers, which tools those servers expose, and which identities reach them — built from passmcp's attestations, reports and your MCP client configurations, queried offline, and gated by policy in CI.
</p>

<p align="center">
  <a href="https://github.com/sebastienrousseau/passmcp-graph/actions"><img src="https://img.shields.io/github/actions/workflow/status/sebastienrousseau/passmcp-graph/ci.yml?style=for-the-badge&logo=github" alt="Build Status" /></a>
  <a href="https://pkg.go.dev/satellion.com/passmcp-graph"><img src="https://img.shields.io/badge/go.dev-module-fc8d62?style=for-the-badge&logo=go&logoColor=white" alt="Go module" /></a>
  <a href="https://pkg.go.dev/satellion.com/passmcp-graph"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=for-the-badge&logo=go&logoColor=white" alt="Go Reference" /></a>
  <a href="https://scorecard.dev/viewer/?uri=satellion.com/passmcp-graph"><img src="https://img.shields.io/ossf-scorecard/satellion.com/passmcp-graph?style=for-the-badge&label=OpenSSF%20Scorecard&logo=openssf" alt="OpenSSF Scorecard" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-GPL--3.0--only-blue?style=for-the-badge" alt="License: GPL-3.0-only" /></a>
  <a href="#requirements"><img src="https://img.shields.io/github/go-mod/go-version/sebastienrousseau/passmcp-graph?style=for-the-badge&logo=go&logoColor=white&label=Go" alt="Minimum Go version" /></a>
</p>

---

## Contents

**Getting started**

- [Install](#install) — `go install`, or build from source
- [Requirements](#requirements) — the Go floor, and passmcp's evidence to feed it
- [Quick Start](#quick-start) — ingest, ask a question, gate on a policy

**The passmcp-graph ecosystem**

- [The passmcp-graph ecosystem](#the-passmcp-graph-ecosystem) — `passmcp`, `passmcp-reporting`, `passmcp-action`, `passmcp-server`, `passmcp-graph` at a glance

**Reference**

- [Capabilities at a glance](#capabilities-at-a-glance) — the commands and what each reads
- [Ecosystem comparison](#ecosystem-comparison) — beside passmcp alone and a general graph database
- [Benchmarks](#benchmarks) — what a query costs
- [Features](#features) — inherited risk, over-privilege, the query language, export
- [Configuration](#configuration) — two global flags
- [Examples](#examples) — the published fixture's queries

**Operational**

- [When not to use passmcp-graph](#when-not-to-use-passmcp-graph) — limitations
- [Development](#development) — make targets, CI
- [Security](#security) — what is never stored, and no network
- [Documentation](#documentation) — all reference docs
- [Stability guarantees](#stability-guarantees) — the graph format, the query language, exit statuses
- [License](#license)

---

## Install

### As a Go program

```bash
go install satellion.com/passmcp-graph/cmd/passmcp-graph@latest
```

### From source

```bash
git clone https://github.com/sebastienrousseau/passmcp-graph.git
cd passmcp-graph
make build   # writes build/passmcp-graph
```

There are no pre-built binaries or container images yet; see
[When not to use passmcp-graph](#when-not-to-use-passmcp-graph).

---

## Requirements

| Requirement | Floor | Enforced by |
| :--- | :--- | :--- |
| Go (building from source) | the `go` directive in [`go.mod`](go.mod) | CI tests on that version and on latest stable, on Linux, macOS and Windows |
| Evidence | passmcp attestations or `passmcp check --output json` reports, from [passmcp](https://github.com/sebastienrousseau/passmcp) | `ingest` skips, and names, a file it does not recognise |
| Network | none | a test runs every command with egress recorded and requires zero requests |

---

## Quick Start

```bash
passmcp-graph ingest attestations/ reports/ ~/Library/Application\ Support/Claude/claude_desktop_config.json
passmcp-graph analyze
passmcp-graph query "agents -> tools where destructive and server.auth = none"
passmcp-graph check --policy policy.yaml   # exit 1 names every forbidden path
```

`ingest` reads what it is given into `.passmcp-graph/graph.json` and can be
run again as evidence arrives: the same file twice changes nothing, and a
newer attestation replaces an older verdict but an older one never
replaces a newer. `analyze` names every agent that inherits a server's
failing critical finding, and every identity holding write or admin
scopes on a server whose tools are all read-only.

---

## The passmcp-graph ecosystem

passmcp evaluates one server at a time. passmcp-graph joins those evaluations
to the clients that use the servers and the identities that reach them, so
a question about the fleet has an answer.

| Component | Purpose | Use case |
| :--- | :--- | :--- |
| [`passmcp`](https://github.com/sebastienrousseau/passmcp) | The engine, every check, and the CLI, TUI and web surfaces (GPL-3.0-only) | Evaluate a server and write the statement |
| [`passmcp-reporting`](https://github.com/sebastienrousseau/passmcp-reporting) | The attestation format, the graph data model and their schemas (Apache-2.0) | Read either format in your own tooling |
| [`passmcp-action`](https://github.com/sebastienrousseau/passmcp-action) | The GitHub Action and GitLab template (Apache-2.0) | Run passmcp in CI |
| [`passmcp-server`](https://github.com/sebastienrousseau/passmcp-server) | passmcp's diagnostics as read-only MCP tools (GPL-3.0-only) | Evaluate a server from inside an editor |
| **`passmcp-graph`** | The graph of agents, servers, tools and identities (GPL-3.0-only) | Ask which agents reach a destructive, unauthenticated tool |

---

## Capabilities at a glance

| Area | Capability | Status |
| :--- | :--- | :--- |
| Ingest | passmcp attestations, passmcp JSON reports, and Claude Desktop, Cursor, VS Code and Zed configurations, from files or directories | Experimental |
| Analyse | Inherited critical findings, and over-privileged identities | Experimental |
| Query | Paths of node kinds joined by `->`, filtered by `where` | Experimental |
| Policy | `check --policy` with forbidden-path rules; exit 1 on a violation | Experimental |
| Export | `json` (schema-valid, re-imports without loss), `graphml`, `cypher` | Experimental |
| Network | none; `--offline` refuses every request as well | Stable |
| Connectors to identity providers or SaaS inventories | not built | Planned |

---

## Ecosystem comparison

passmcp alone answers "is this server safe"; a general graph database answers
anything you load into it. passmcp-graph answers the questions in between,
from passmcp's own evidence, with no server to run.

| Approach | Reads passmcp's evidence directly | Needs a running service | Policy gate with exit status |
| :--- | :---: | :---: | :---: |
| **passmcp-graph** | yes | no | yes |
| `passmcp check` per server | yes — one at a time | no | per server |
| Neo4j or another graph database, loaded from `export --format cypher` | through the export | yes | write it yourself |

---

## Benchmarks

Each result is the median of 50 runs of the whole process, start to exit,
against the published fixture (16 nodes, 15 edges). Process start
dominates at this size.

| Scenario | Result | Environment |
| :--- | ---: | :--- |
| `query "agents -> tools where destructive"` | 50.6 ms | Apple A18 Pro, Go 1.27.1, 2026-09-27 |
| `analyze` | 37.4 ms | same |

---

## Features

- **Inherited risk.** A server's failing critical finding becomes a risk for
  every agent with a path to it, explained by the finding and the path. The
  next attestation that no longer fails the check removes it.
- **Over-privilege.** An identity holding a scope that names write, admin,
  delete, manage or owner (or `*`) on a server whose every tool is
  annotated read-only is flagged, with the scopes that caused it.
- **A query language with a written grammar.** Paths such as
  `agents -> tools`, joined through the server when two kinds have no
  direct relation, filtered by `and`, `or`, `not` and comparisons. See
  [`docs/query-language.md`](docs/query-language.md).
- **Policy as forbidden paths.** A YAML file of named queries; any match
  is a violation, reported with its rule, reason and path.
- **Export.** JSON in the versioned graph format from `passmcp-reporting`,
  validated against its schema on export and on import; GraphML and
  Cypher for existing graph tools.
- **No secrets stored.** Client configurations are read for their server
  names, URLs and commands only; `env` and `headers` are never read, URL
  credentials and secret-named query parameters are dropped, and
  token-shaped arguments are replaced with `<redacted>`.

---

## Configuration

| Flag | Default | Meaning |
| :--- | :--- | :--- |
| `--store DIR` | `.passmcp-graph` | Directory holding `graph.json` |
| `--offline` | off | Replace the HTTP transport with one that refuses every request |

Policy files are described in [`docs/policy.md`](docs/policy.md). There
are no environment variables and no configuration file.

---

## Examples

The fixture in [`testdata/fixture/`](testdata/fixture) is published with
its expected answers, and the test suite requires every one:

```console
$ passmcp-graph --store s import testdata/fixture/graph.json
imported 16 nodes and 15 edges
$ passmcp-graph --store s query "agents -> tools where destructive and server.auth = none"
claude-desktop -> https://crm.example/mcp -> delete_record
$ passmcp-graph --store s analyze
risk: claude-desktop inherits auth.unauthenticated_tools (critical) via claude-desktop -> https://crm.example/mcp
over-privileged: ops-agent on https://files.example/mcp: holds admin, files:write but every tool the server exposes is read-only
```

[`testdata/fixture/queries.json`](testdata/fixture/queries.json) has eight
more, from stale attestations to identities by client id.

---

## When not to use passmcp-graph

- **You need an inventory you do not already have.** passmcp-graph reads
  evidence you give it. It does not discover servers, query an identity
  provider or call a SaaS API; those connectors do not exist yet.
- **You need which tools an agent actually called.** Configurations say
  which servers an agent can reach, not which tools it used, so a path
  through a server includes every tool that server exposes.
- **You need a multi-user service.** The store is one JSON file with no
  locking; run one `ingest` at a time.
- **You need pre-built binaries.** Install with `go install` until the
  release pipeline exists.

---

## Development

```bash
make            # format, vet, lint, headers, README, tests
make test-race  # race detector, shuffled order
make coverage   # the 85% per-package gate CI runs
make fixture-update  # regenerate testdata/fixture/graph.json
```

Every acceptance criterion from
[passmcp#82](https://github.com/sebastienrousseau/passmcp/issues/4) has a test
marked `// AC: SG-0N`. Tests never use the network.

---

## Security

- Credentials in client configurations are never read or stored; the
  `SG-02` test seeds nine kinds of secret and searches the saved store for
  each.
- passmcp-graph makes no network request. The `SG-07` tests record every
  request through the default transport and every connection to a proxy
  listener, and check that only the offline guard imports a networking
  package.
- Everything read is treated as untrusted: a file given to `import` is
  validated against the published schema before any of it is read.
- No telemetry.

Report vulnerabilities according to [`SECURITY.md`](SECURITY.md).

---

## Documentation

| Document | Covers |
| :--- | :--- |
| [`docs/index.md`](docs/index.md) | The manual's front page |
| [`docs/ingest.md`](docs/ingest.md) | Which input supplies what, and what is never stored |
| [`docs/query-language.md`](docs/query-language.md) | The grammar, every field, and examples |
| [`docs/policy.md`](docs/policy.md) | The policy file and `check` |
| [`SECURITY.md`](SECURITY.md) | Disclosure policy and what is guaranteed |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Signed-commit and DCO policy, what a change needs |
| [`CHANGELOG.md`](CHANGELOG.md) | Per-release notes |

---

## Stability guarantees

passmcp-graph is pre-1.0 and versions move by 0.0.1. The JSON export follows
the graph format `https://satellion.com/graph/v1` from `passmcp-reporting`,
which changes only by a new version URL. The query language, the policy
file (`version: 1`) and the exit statuses (0 clean, 1 policy violation,
2 error) may change before 1.0; every change is in the changelog. The
`internal/` packages are not an API.

---

## License

GPL-3.0-only; see [`LICENSE`](LICENSE). The graph data model it reads and
writes is Apache-2.0, in
[`passmcp-reporting`](https://github.com/sebastienrousseau/passmcp-reporting).
