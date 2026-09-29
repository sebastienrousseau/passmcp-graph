<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Working on passmcp-graph as an AI agent

Invariants for AI-assisted contributions: the constraints a plausible
change can break for a reason the code does not state.

## Hard gates

| Gate | Command |
| --- | --- |
| 85% statement coverage, every package but `cmd/passmcp-graph` | `make coverage` |
| Race detector, randomised order | `make test-race` |
| Lint at zero findings, on darwin and linux | `make lint`, `GOOS=linux make lint` |
| SPDX header on every source file | `make spdx-check` |
| README follows the portfolio template, badges and ecosystem table | `make readme-check` |
| Complexity: cyclomatic 10, cognitive 15, 60 lines per function | `make lint` (gocyclo, gocognit, funlen) |
| Every version-bearing file names the same release | `make versions` |
| Install contract under `PREFIX` and `DESTDIR` | `make install-smoke` |
| Every acceptance criterion has a passing test | `make trace-check` |
| The retired product name appears nowhere | `make name-guard` |

[DEVELOPMENT.md](DEVELOPMENT.md) maps every CI job to its local command.

## Commits

- Every commit is cryptographically signed and carries a DCO
  `Signed-off-by` trailer. An agent's shell usually cannot reach the
  maintainer's ssh-agent; hand the commits over as a script.
- Conventional Commits for the subject line.
- Never rewrite published history.

## Versioning

- Pre-1.0, every release moves the patch digit by one.
- The version lives in the newest `## [x.y.z]` heading in `CHANGELOG.md`.
  `cli.Version` is injected at build time through `-ldflags`; never
  hard-code one.
- The family moves in lockstep: passmcp-reporting in `go.mod`, the trace
  tool in `Makefile`, the install lines, `CITATION.cff` and the README's
  ecosystem sentence all name the same release. `make versions` fails
  otherwise.

## Things that are load-bearing

- **No secret reaches the store.** Ingest reads a configuration's server
  names, URLs and commands and nothing else. A change that reads `env` or
  `headers`, or stops redacting an argument, is the most damaging change
  available here. The `SG-02` test is the guard; extend it with any new
  secret shape.
- **No network.** Only `internal/netguard` may import `net` or
  `net/http`; a test enforces it. A connector that needs the network is a
  design decision for the maintainer, not a drive-by import.
- **Server identity is passmcp's.** A server is `graph.ServerID(transport,
  endpoint)`, the attestation subject digest, so evidence from different
  inputs meets on one node. Do not normalise endpoints differently in one
  input.
- **Newer evidence wins, older never does.** An older attestation must
  not replace a newer verdict; ordering is by `ranAt`.
- **Stdout carries results; diagnostics go to stderr.** This keeps
  `--format json` pipeable.
- **Exit statuses are an interface.** 0 clean, 1 policy violation, 2
  error. CI pipelines depend on the distinction.
- **The fixture is published.** `testdata/fixture/queries.json` holds
  expected answers by hand; do not regenerate it to make a test pass.

## Scope

- The data model belongs to `passmcp-reporting`'s `graph` package
  (Apache-2.0). A new node or edge kind goes there first.
- Do not add a dependency without saying why in the commit.
- Do not add a CI gate that does not currently pass.
