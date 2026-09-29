<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Architecture

How passmcp-graph turns evidence into answers, for contributors. The user's
view is the [manual](index.md); the decisions behind this shape are in
[`adr/`](adr/README.md).

## The flow

```text
attestations, reports,       ingest          .passmcp-graph/graph.json
client configurations  ──────────────────▶  (passmcp-reporting's graph format)
                                                   │
                  ┌──────────────┬─────────────────┼──────────────┐
                  ▼              ▼                 ▼              ▼
               analyze         query            check          export
          (risks, privilege) (paths)      (forbidden paths)  (json, graphml,
                                                               cypher)
```

Every command but `ingest` and `import` reads the store and writes nothing
back. What `analyze` derives is recomputed from the graph on every run, so
correcting the evidence corrects the answer.

## Packages

| Package | Does | Reads | Writes |
| :--- | :--- | :--- | :--- |
| `cmd/passmcp-graph` | Hands `os.Args`, the standard streams and the clock to `cli.Run` and exits with its status | — | — |
| `internal/cli` | Global flags, the command table, output formats, exit statuses, shell completions | the store, policy files | stdout (results), stderr (diagnostics) |
| `internal/ingest` | Recognises each input and adds its facts to the graph | files and directories the operator names | the graph in memory |
| `internal/analyze` | Inherited critical findings, over-privileged identities | the graph | — |
| `internal/query` | Parses and evaluates the query language | the graph | — |
| `internal/policy` | Parses a policy file and runs each forbidden-path query | the graph | — |
| `internal/export` | JSON, GraphML and Cypher writers; JSON import | the graph, an export file | stdout |
| `internal/schema` | Validates a JSON export against the published graph schema | — | — |
| `internal/netguard` | `--offline`: replaces the default HTTP transport with one that refuses | — | — |
| `internal/fixture` | Builds the evidence the tests and the published fixture use | — | `testdata/fixture/graph.json` on `make fixture-update` |

The data model — node and edge kinds, `graph.ServerID`, `Load` and `Save` —
is the `graph` package of
[passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting),
Apache-2.0, so other tools can read the same store.

## What each input supplies

| Input | Supplies |
| :--- | :--- |
| passmcp attestation | A server's score, grade and failing checks, and the attestation node (digest, predicate type, `ranAt`, file) |
| `passmcp check --output json` report | The server's tools and their annotations, how it admits clients, and the identity passmcp authenticated as, with its scopes |
| Claude Desktop, Cursor, VS Code or Zed configuration | Agents, and the servers each uses; server names, URLs and commands only |

All three name a server by `graph.ServerID(transport, endpoint)`, so the
same server from three inputs is one node
([ADR 0003](adr/0003-server-identity-is-the-attestation-subject.md)).

## Invariants and where they are tested

| Invariant | Test |
| :--- | :--- |
| No secret from a configuration reaches the store | `TestConfigsBecomeAgentsAndNoSecretIsStored` (`internal/ingest`) |
| No command makes a network request, with or without `--offline` | `TestNoCommandContactsTheNetwork` (`internal/cli`) |
| Only `internal/netguard` imports `net` or `net/http` | `TestOnlyTheOfflineGuardImportsTheNetwork` (`internal/cli`) |
| Ingesting the same evidence twice changes nothing | `TestAttestationsBecomeServersAndIngestingTwiceChangesNothing` (`internal/ingest`) |
| A JSON export validates and re-imports without loss | `TestExportsValidateAndReimportWithoutLoss` (`internal/export`) |
| The published fixture answers every published query | `TestQueriesOverThePublishedFixtureReturnExactlyTheExpectedPaths` (`internal/query`) |

## Output and exit statuses

Results go to stdout in the selected format (`text` or `json`, or an
export format); diagnostics go to stderr. The exit status is 0 on success,
1 when `check` finds a forbidden path, and 2 on any error.
