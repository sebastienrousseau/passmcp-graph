<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Changelog

All notable changes to passmcp-graph are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions are
[Semantic Versioning](https://semver.org/) shaped; before 1.0 every
release moves the patch digit.

## [0.0.1] — 2026-09-29

The first release.

### Added

- **A local graph of agents, MCP servers, tools and identities**, built from
  passmcp's attestations, reports and MCP client configurations: `ingest`,
  `analyze`, `query`, `check`, `export` and `import`, all offline, with a
  policy gate for CI. The store holds no secret, however a configuration
  carried it.
