<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Architecture Decision Records

Decisions about this repository that will be questioned later, with the
reasoning that produced them. Records are immutable once merged; a
decision that changes gets a new record superseding the old one.

Decisions about the checks, the score and the attestation format are
passmcp's, in [passmcp's ADRs](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/README.md).
This directory records what is decided here. The records below were made
with the 0.0.1 release and written down afterwards, from the package
documentation, [AGENTS.md](https://github.com/sebastienrousseau/passmcp-graph/blob/main/AGENTS.md)
and the tests that enforce them.

| # | Decision | Status |
| :--- | :--- | :--- |
| [0001](0001-no-network.md) | passmcp-graph makes no network request, and `--offline` enforces it | Accepted |
| [0002](0002-no-secret-reaches-the-store.md) | Client configurations are read for names, URLs and commands only | Accepted |
| [0003](0003-server-identity-is-the-attestation-subject.md) | A server's node is identified by passmcp's attestation subject | Accepted |
| [0004](0004-newer-evidence-wins.md) | Newer evidence replaces older, and older never replaces newer | Accepted |
| [0005](0005-data-model-in-passmcp-reporting.md) | The data model and store format live in passmcp-reporting | Accepted |
