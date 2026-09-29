<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Governance

passmcp-graph is one repository in the passmcp family and is governed the
way passmcp is. This document says what is specific to this repository and
points at passmcp's for the rest.

## Roles

**Maintainer:** Sebastien Rousseau (<sebastian.rousseau@gmail.com>,
GitHub `@sebastienrousseau`), with commit access, responsible for what the
graph reads, what its answers claim, its releases and its security
response.

**Contributor:** anyone who opens an issue or a pull request. Mechanics
are in [CONTRIBUTING.md](CONTRIBUTING.md).

## What is decided here, and what is not

This repository decides **what the graph reads and what it answers**: the
inputs `ingest` recognises and which of their fields it keeps, the query
language, the policy file, the analyses and the exit statuses. A change
that widens what is read from a client configuration, or that adds a
network connection, is recorded in [docs/adr/](docs/adr/README.md) before
it lands.

It does not decide **the data model** or **the verdicts**. Node and edge
kinds and the graph format are passmcp-reporting's; which checks passmcp
runs and how it scores them are passmcp's, recorded in passmcp's ADRs.

## Version and release

The family moves in lockstep: this repository is released at the same
version as passmcp, and requires passmcp-reporting at that version
(`make versions` checks it). Releases are signed tags cut by the
Maintainer; see [DEVELOPMENT.md](DEVELOPMENT.md#release-model).

## Continuity

The single-Maintainer model is a real bus-factor risk, stated rather than
hidden. The succession procedure — hand-off, community fork after six
months of unresponsiveness, and compromise response — is passmcp's, in
[passmcp's GOVERNANCE.md](https://github.com/sebastienrousseau/passmcp/blob/main/GOVERNANCE.md),
and applies to this repository as one of the family. GPL-3.0-only lets
anyone fork under the same licence without further permission.

The family manifest does not yet record a criterion for archiving this
repository; that is owed by the Maintainer.

## Changes to this document

Through the usual pull request process.
