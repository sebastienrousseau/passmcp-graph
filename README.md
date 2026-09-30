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
  <a href="https://github.com/sebastienrousseau/passmcp-graph/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/sebastienrousseau/passmcp-graph/ci.yml?branch=main&style=for-the-badge&logo=github&label=Build" alt="Build" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-graph/blob/main/DEVELOPMENT.md#coverage"><img src="https://img.shields.io/endpoint?url=https%3A%2F%2Fsebastienrousseau.com%2Fpassmcp-graph%2Fcoverage.json&style=for-the-badge&logo=codecov&logoColor=white" alt="Coverage" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-graph/releases"><img src="https://img.shields.io/github/v/release/sebastienrousseau/passmcp-graph?style=for-the-badge&color=fc8d62&logo=github&label=Release" alt="Release" /></a>
  <a href="https://pkg.go.dev/satellion.com/passmcp-graph"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=for-the-badge&labelColor=555555&logo=go&logoColor=white" alt="Docs" /></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-graph"><img src="https://img.shields.io/ossf-scorecard/github.com/sebastienrousseau/passmcp-graph?style=for-the-badge&label=OpenSSF%20Scorecard&logo=openssf" alt="OpenSSF Scorecard" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL--3.0--only-blue.svg?style=for-the-badge" alt="License: GPL-3.0-only" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-graph/blob/main/DEVELOPMENT.md#requirements"><img src="https://img.shields.io/badge/go-1.26.8%2B-93450a.svg?style=for-the-badge&logo=go" alt="Go 1.26.8+" /></a>
</p>

<p align="center">
  <img src=".github/demo.gif" alt="passmcp-graph importing the published fixture graph, naming an inherited critical risk and an over-privileged identity, and answering two path queries" width="100%" />
</p>

---

## Contents

**Getting started**

- [Install](#install) — `go install`, `make install`, or build from source
- [Requirements](#requirements) — the Go floor, and passmcp's evidence to feed it
- [Quick Start](#quick-start) — ingest, ask a question, gate on a policy

**The passmcp-graph ecosystem**

- [The passmcp-graph ecosystem](#the-passmcp-graph-ecosystem) — `passmcp`, `passmcp-reporting`, `passmcp-server`, `passmcp-action`, `passmcp-graph`, `passmcp-registry`, `passmcp-lsp`, `passmcp-census`, `satellion.com`

**Library reference**

- [Capabilities at a glance](#capabilities-at-a-glance) — the current surface by theme
- [Ecosystem comparison](#ecosystem-comparison) — short matrix; full table at [`docs/COMPARISON.md`](docs/COMPARISON.md)
- [Benchmarks](#benchmarks) — headline numbers; full table at [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md)
- [Features](#features) — inherited risk, over-privilege, the query language, export
- [Configuration](#configuration) — two global flags
- [Examples](#examples) — the published fixture's queries

**Operational**

- [When not to use passmcp-graph](#when-not-to-use-passmcp-graph) — limitations
- [Development](#development) — make targets, the install contract, CI
- [Security](#security) — what is never stored, and no network
- [Documentation](#documentation) — all reference docs
- [Stability guarantees](#stability-guarantees) — the graph format, the query language, exit statuses
- [License](#license)

---

## Install

### As a Go program

```bash
go install satellion.com/passmcp-graph/cmd/passmcp-graph@v0.0.4
```

### From source, with `make install`

```bash
git clone https://github.com/sebastienrousseau/passmcp-graph.git
cd passmcp-graph
make install PREFIX="$HOME/.local"
```

`make install` builds the binary and its bash, zsh and fish completions and
installs them under `PREFIX` (default `/usr/local`), staged under `DESTDIR`
when a packager sets it; `make uninstall` removes them. The
[GNUmakefile](GNUmakefile) holds the contract, and CI checks the staged tree
on every push. `make build` alone writes `build/passmcp-graph`.

### Release archives

0.0.1 was released as source, with no archives attached. From the next
tag, the [Release workflow](.github/workflows/release.yml) attaches archives
for Linux, macOS and Windows on amd64 and arm64, with a signed
`checksums.txt` and SLSA build provenance, to the
[releases page](https://github.com/sebastienrousseau/passmcp-graph/releases).

---

## Requirements

| Requirement | Floor | Enforced by |
| :--- | :--- | :--- |
| Go (building from source) | 1.26.8, the `go` directive in [`go.mod`](go.mod) | CI tests on that version and on latest stable, on Linux, macOS and Windows |
| Evidence | passmcp attestations or `passmcp check --output json` reports, from [passmcp](https://github.com/sebastienrousseau/passmcp) 0.0.1 | `ingest` skips, and names, a file it does not recognise |
| Network | none | a test runs every command with egress recorded and requires zero requests |

The Go floor is raised only when a release needs a language feature or a
standard-library security fix, on a patch release like everything else
pre-1.0, and the changelog says so. [DEVELOPMENT.md](DEVELOPMENT.md#requirements)
lists the tools for each local gate.

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

Every component is released at **0.0.4** and moves in lockstep: one version across the family, released together ([docs/ecosystem.md](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)).

| Component | Purpose | Use case |
| :--- | :--- | :--- |
| [passmcp](https://github.com/sebastienrousseau/passmcp) | The MCP server diagnostic: checks in nine phases, every finding tied to the request that showed it, signed attestations | Test a server before your agents trust it, and gate it in CI |
| [passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting) | The attestation format, its JSON Schemas and offline verifier, the graph model, and the agentgateway processor | Verify an attestation in a gateway, registry or pipeline |
| [passmcp-server](https://github.com/sebastienrousseau/passmcp-server) | passmcp's diagnostics as read-only MCP tools | Evaluate a server, or check an attestation, from inside the agent |
| [passmcp-action](https://github.com/sebastienrousseau/passmcp-action) | passmcp in GitHub Actions and GitLab CI, the image pinned by digest | Fail a build on the findings you choose |
| [passmcp-graph](https://github.com/sebastienrousseau/passmcp-graph) | A local graph of agents, servers, tools and identities built from attestations | Find inherited risk and over-privilege, and gate on policy |
| [passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry) | A signed public scorecard of the MCP Registry's remote servers | Check a public server's standing before connecting to it |
| [passmcp-lsp](https://github.com/sebastienrousseau/passmcp-lsp) | A language server for MCP artefacts, with check-id hover from the guidance catalogue | Catch mistakes in server.json, tool schemas and client configuration while editing |
| [passmcp-census](https://github.com/sebastienrousseau/passmcp-census) | The published reliability census: dataset, methodology, disclosure log and reproduction command | Cite ecosystem-wide reliability figures, and reproduce them |
| [satellion.com](https://github.com/sebastienrousseau/satellion.github.io) | The website, the Go module paths and the format URIs | Read the manual, and resolve `satellion.com/...` imports |

passmcp evaluates one server at a time. passmcp-graph joins those
evaluations to the clients that use the servers and the identities that
reach them, so a question about the fleet has an answer. It reads the graph
format from passmcp-reporting at the same version; `make versions` checks
that `go.mod` requires it there.

---

## Capabilities at a glance

| Area | Capability | Status |
| :--- | :--- | :--- |
| Ingest | passmcp attestations, passmcp JSON reports, and Claude Desktop, Cursor, VS Code and Zed configurations, from files or directories | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-graph/releases/tag/v0.0.1) |
| Analyse | Inherited critical findings, and over-privileged identities | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-graph/releases/tag/v0.0.1) |
| Query | Paths of node kinds joined by `->`, filtered by `where` | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-graph/releases/tag/v0.0.1) |
| Policy | `check --policy` with forbidden-path rules; exit 1 on a violation | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-graph/releases/tag/v0.0.1) |
| Export | `json` (schema-valid, re-imports without loss), `graphml`, `cypher`; `import` of a JSON export | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-graph/releases/tag/v0.0.1) |
| Network | none; `--offline` refuses every request as well | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-graph/releases/tag/v0.0.1) |
| Shell completions | `completion bash\|zsh\|fish`, installed by `make install` | [Not yet released](https://github.com/sebastienrousseau/passmcp-graph/blob/main/CHANGELOG.md#002) |
| Release archives | Linux, macOS and Windows on amd64 and arm64, signed checksums, SLSA provenance | [Not yet released](https://github.com/sebastienrousseau/passmcp-graph/blob/main/CHANGELOG.md#002) |

---

## Ecosystem comparison

passmcp alone answers "is this server safe"; a general graph database answers
anything you load into it. passmcp-graph answers the questions in between,
from passmcp's own evidence, with no server to run.

| Project | Reads passmcp's evidence directly | Needs a running service | Policy gate with exit status |
| :--- | :---: | :---: | :---: |
| **passmcp-graph** | yes | no | yes |
| `passmcp check` per server | yes — one at a time | no | per server |
| Neo4j or another graph database, loaded from `export --format cypher` | through the export | yes | write it yourself |

See [`docs/COMPARISON.md`](docs/COMPARISON.md) for the evidence and complete matrix.

---

## Benchmarks

Median of 50 runs of the whole process, start to exit, measured with
[hyperfine](https://github.com/sharkdp/hyperfine) against the published
fixture (16 nodes, 15 edges) on a machine running other builds at the
time. Process start dominates at this size.

| Scenario | Result | Environment |
| :--- | ---: | :--- |
| `query "agents -> tools where destructive"` | 13.9 ms median, 9.6 ms min | Apple A18 Pro, Go 1.27.1, 2026-09-29, load average 46 |
| `analyze` | 20.7 ms median, 10.3 ms min | same |

See [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md) for methodology and full results.

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

Shell completions come from the command table, so they list every command:

```bash
passmcp-graph completion bash > ~/.local/share/bash-completion/completions/passmcp-graph
passmcp-graph completion zsh > "${fpath[1]}/_passmcp-graph"
passmcp-graph completion fish > ~/.config/fish/completions/passmcp-graph.fish
```

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
- **You need pre-built binaries of 0.0.1.** It was released as source;
  install it with `go install` or `make install`. Archives are attached
  from the next release on.

---

## Development

```bash
make                 # format, vet, lint, headers, README, name guard, versions, tests
make test-race       # race detector, shuffled order
make coverage        # the 85% per-package gate CI runs
make coverage-json   # build/coverage.json, the document behind the badge
make versions        # every version-bearing file names the same release
make install-smoke   # install and uninstall under a staged DESTDIR
make trace-check     # every acceptance criterion has a passing test (network)
make fixture-update  # regenerate testdata/fixture/graph.json
```

Every gate CI runs has a local form; [DEVELOPMENT.md](DEVELOPMENT.md) maps
them. Every acceptance criterion from
[passmcp#4](https://github.com/sebastienrousseau/passmcp/issues/4) has a test
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
- CI runs `govulncheck` on every push, CodeQL on every push and pull
  request and weekly, and OpenSSF Scorecard on every push to main and
  weekly.

Report vulnerabilities according to [`SECURITY.md`](SECURITY.md).

---

## Documentation

The four entry points, identical across every repo in the family:

- **[User Manual](https://sebastienrousseau.com/passmcp-graph/)** — ingest, the query language and policies
- **[API reference](https://pkg.go.dev/satellion.com/passmcp-graph)** — this module's packages
- **[Developer docs](DEVELOPMENT.md)** — toolchain, task map, reproducing every CI gate locally
- **[Ecosystem map](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)** — the family, the published artefacts, the lockstep version rule

| Document | Covers |
| :--- | :--- |
| [`docs/index.md`](docs/index.md) | The manual's front page: every command |
| [`docs/ingest.md`](docs/ingest.md) | Which input supplies what, and what is never stored |
| [`docs/query-language.md`](docs/query-language.md) | The grammar, every field, and examples |
| [`docs/policy.md`](docs/policy.md) | The policy file and `check` |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | The packages, the flow, and the invariants with their tests |
| [`docs/adr/`](docs/adr/README.md) | Decision records for this repository |
| [`docs/COMPARISON.md`](docs/COMPARISON.md) | passmcp-graph beside passmcp alone and a graph database |
| [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md) | What a command costs, and how it was measured |
| [`docs/releases/`](docs/releases/v0.0.1.md) | Release highlights, one file per release |
| [`SECURITY.md`](SECURITY.md) | Disclosure policy and what is guaranteed |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Signed-commit and DCO policy, what a change needs |
| [`CHANGELOG.md`](CHANGELOG.md) | Per-release notes |

---

## Stability guarantees

passmcp-graph is pre-1.0 and versions move by 0.0.1, in lockstep with the
rest of the family. Every capability above was released in 0.0.1 as
experimental except `--offline` and the no-network guarantee, which are
stable. The JSON export follows
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
