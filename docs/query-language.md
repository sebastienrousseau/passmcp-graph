<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Query language

A query is a path of node kinds joined by `->`, optionally filtered by a
condition. The result is every path in the graph that matches, one per
line, each node named by its label.

```text
query      = path [ "where" expr ]
path       = kind { "->" kind }
kind       = "agents" | "servers" | "tools" | "identities" | "sources" | "attestations"
expr       = term { "or" term }
term       = factor { "and" factor }
factor     = "not" factor | "(" expr ")" | comparison
comparison = field [ op value ]
field      = [ kindname "." ] name
op         = "=" | "!=" | "<" | "<=" | ">" | ">=" | "contains"
value      = word | number | quoted string
```

Keywords are case-insensitive, and singular kinds (`agent`, `server`)
work as well as plural ones. `and` binds tighter than `or`.

## Paths

| From | To | Relation |
| :--- | :--- | :--- |
| agents | servers | the agent's configuration uses the server |
| servers | tools | the server exposes the tool |
| identities | servers | the identity is authorised on the server, with scopes |
| servers | sources | the server was discovered by the source |
| servers | attestations | the server is attested by the attestation |

Every relation can be walked in either direction. Two kinds with no
direct relation are joined through the server between them, which then
appears in the result: `agents -> tools` is agent, server, tool.
`servers -> servers` has no relation and is an error.

passmcp-graph does not create source nodes itself; they arrive in a graph
given to `import`.

## Fields

A field is named bare when exactly one kind on the path has it, and as
`kind.field` otherwise (`server.name`, `tool.name`). A field with no
operator is tested for truth: a true boolean, a non-zero number, a
non-empty string or list.

| Kind | Field | Type | Meaning |
| :--- | :--- | :--- | :--- |
| agent | `name` | string | The agent's label |
| agent | `client` | string | `claude-desktop`, `cursor`, `zed`, `vscode` or `mcp-client` |
| agent | `config` | string | The configuration file it came from |
| server | `endpoint` | string | URL, or the command line for stdio |
| server | `name`, `version` | string | As the server reported them |
| server | `transport` | string | `http` or `stdio` |
| server | `auth` | string | `none`, `oauth`, `bearer` or `unknown` |
| server | `grade` | string | passmcp's grade, A to F |
| server | `score` | number | passmcp's score out of 100 |
| server | `attested` | boolean | An attestation is on record |
| server | `attestation_age_days` | number | Days since the newest attestation ran |
| server | `failing` | list | Ids of the failing checks |
| server | `critical` | boolean | A failing check is critical |
| tool | `name` | string | The tool's name |
| tool | `readonly` | boolean | Annotated `readOnlyHint: true` |
| tool | `destructive` | boolean | Not read-only, and `destructiveHint` absent or true, as MCP defaults |
| tool | `annotated` | boolean | Declares `readOnlyHint` at all |
| identity | `issuer` | string | The authorisation server |
| identity | `client` | string | The client id |
| identity | `scopes` | list | Scopes held on the path's server |
| source | `type`, `ref` | string | Where the server was discovered |
| attestation | `digest` | string | The statement's SHA-256 |
| attestation | `ran_at` | string | When the run happened |
| attestation | `age_days` | number | Days since it ran |

## Comparisons

- Strings compare without regard to case; `contains` is a substring test.
- Numbers compare numerically; a value that is not a number never matches.
- Booleans take `=` and `!=` with `true` or `false`.
- Lists: `=` and `contains` test membership, `!=` its absence.
- A field with no value — a server never attested has no score — matches
  only `!=`.

## Examples

Each of these is in the published fixture,
[`testdata/fixture/queries.json`](https://github.com/sebastienrousseau/passmcp-graph/blob/main/testdata/fixture/queries.json),
with its expected answer, and the tests require every one.

```text
agents -> tools where destructive and server.auth = none
agents -> servers where not attested
agents -> servers where attestation_age_days > 30
servers where critical
identities -> servers where scopes contains admin
agents -> identities where identity.client = 'reader-agent'
servers -> attestations where grade = F or score >= 95
```
