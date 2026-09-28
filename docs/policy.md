<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Policies

A policy lists paths that must not exist. `passmcp-graph check --policy
FILE` runs every rule's query against the graph and reports each match.

```yaml
version: 1
forbid:
  - name: unattested
    reason: an agent uses a server nobody has attested
    query: agents -> servers where not attested
  - name: stale
    reason: the attestation is older than 30 days
    query: agents -> servers where attestation_age_days > 30
  - name: open-destructive
    query: agents -> tools where destructive and server.auth = none
```

| Key | Required | Meaning |
| :--- | :--- | :--- |
| `version` | yes | Must be `1` |
| `forbid` | yes, non-empty | The rules |
| `forbid[].name` | yes, unique | Printed with each violation |
| `forbid[].reason` | no | Carried into the JSON output |
| `forbid[].query` | yes | A query in the [query language](query-language.md) |

An unknown key is an error, so a misspelt rule cannot pass silently. A
query that does not parse, or names a field its path does not have, is an
error too.

## Exit status

| Status | Meaning |
| :--- | :--- |
| 0 | No forbidden path |
| 1 | At least one; each is printed as `forbidden (<rule>): <path>` |
| 2 | The policy or the store could not be read |

`--format json` prints the violations as an array of objects with
`rule`, `reason` and `path` (an array of `{id, kind, label}` steps),
which is `[]` when there are none.

## In CI

```bash
passmcp-graph --offline ingest attestations/ mcp-configs/
passmcp-graph --offline check --policy policy.yaml
```
