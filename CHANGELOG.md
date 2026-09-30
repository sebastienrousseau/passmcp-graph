<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Changelog

All notable changes to passmcp-graph are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions are
[Semantic Versioning](https://semver.org/) shaped; before 1.0 every
release moves the patch digit.

## [0.0.4] — 2026-09-30

The family's fourth release. The commands, the query language, the policy
file, the graph format and the exit statuses are unchanged from 0.0.3;
this member moves with the family and gains a README demo.

### Added

- **A README demo**, rendered from `.github/demo.tape` by `make demo`:
  importing a fixture graph, the analysis it finds (an inherited critical
  finding and an over-privileged agent), and two queries answered offline.

### Changed

- **Release pages are published in the family layout** by the release
  workflow itself (Highlights, What's Changed, Checksums, Full Changelog),
  so no page is rewritten by hand after a release.
- **In lockstep with passmcp 0.0.4**: passmcp-reporting is required at
  v0.0.4, and passmcp's trace runs at v0.0.4; neither changes what this
  repository reads or writes.

## [0.0.3] — 2026-09-30

The family's third release. The commands, the query language, the policy
file, the graph format and the exit statuses are unchanged from 0.0.2;
this member moves with the family and gains fuzz testing.

### Added

- **Fuzz targets.** `FuzzParse` checks that any query either fails with
  an error or parses to a tree that survives being written back out and
  parsed again, and that evaluating it never panics.
  `FuzzConfigNeverStoresASecret` places a fuzzer-chosen secret everywhere
  a client configuration can carry a credential and checks it never
  reaches the graph.

### Changed

- **passmcp-reporting is required at v0.0.3**, and passmcp's trace runs
  at v0.0.3; neither changes what this repository reads or writes.

## [0.0.2] — 2026-09-29

The family's second release. The 0.0.1 commands, the query language, the
policy file, the graph format and the exit statuses are unchanged; what is
new is the `completion` command and how passmcp-graph is installed and
released.

### Added

- **Shell completions.** `passmcp-graph completion bash|zsh|fish` prints a
  completion script generated from the command table, and `make install`
  installs all three with the binary under `PREFIX` and `DESTDIR`
  (`GNUmakefile`).
- **Release archives.** A tag-triggered Release workflow builds archives
  for Linux, macOS and Windows on amd64 and arm64 with goreleaser, signs
  `checksums.txt` with cosign keyless and attaches SLSA build provenance.
- **Repository standard.** OpenSSF Scorecard, a coverage badge published
  with the manual, `make versions` checking that every version-bearing
  file names the same release, and the family's community, development
  and architecture documents.

### Changed

- **passmcp-reporting is required at v0.0.2**, the family's lockstep
  version, instead of a pre-release commit.
- **Complexity ceilings tightened** to cyclomatic 10, cognitive 15 and 60
  lines per function; `query.Eval`, `check` and the licence-header sweep
  were split to meet them, with no change in behaviour.

## [0.0.1] — 2026-09-29

The first release.

### Added

- **A local graph of agents, MCP servers, tools and identities**, built from
  passmcp's attestations, reports and MCP client configurations: `ingest`,
  `analyze`, `query`, `check`, `export` and `import`, all offline, with a
  policy gate for CI. The store holds no secret, however a configuration
  carried it.
