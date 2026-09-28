<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# passmcp-graph

passmcp-graph builds a local graph of which agents use which MCP servers,
which tools those servers expose, and which identities reach them, from
[passmcp](https://github.com/sebastienrousseau/passmcp)'s evidence and your
MCP client configurations. It answers questions about that graph offline
and gates on policy in CI.

```bash
passmcp-graph ingest attestations/ reports/ claude_desktop_config.json
passmcp-graph analyze
passmcp-graph query "agents -> tools where destructive and server.auth = none"
passmcp-graph check --policy policy.yaml
passmcp-graph export --format graphml > estate.graphml
```

| Command | Does |
| :--- | :--- |
| `ingest PATH...` | Reads files, or every `*.json` under a directory; see [Ingesting evidence](ingest.md) |
| `analyze [--format text\|json]` | Inherited risks and over-privileged identities |
| `query QUERY [--format text\|json]` | Paths that match; see [Query language](query-language.md) |
| `check --policy FILE [--format text\|json]` | Forbidden paths; see [Policies](policy.md) |
| `export --format json\|graphml\|cypher` | The whole graph, to stdout |
| `import FILE` | Merges a JSON export into the store, after validating it |
| `version` | Prints the version |

Global flags come before the command: `--store DIR` (default
`.passmcp-graph`) and `--offline`. Results go to stdout and diagnostics to
stderr. The exit status is 0 on success, 1 when `check` finds a
forbidden path, and 2 on any error.

## Risk inheritance

A server's failing finding of severity `critical`, from its latest
attestation, is inherited by every agent that uses the server. `analyze`
reports each as the agent, the finding and the path. When a newer
attestation no longer fails the check, the risk is gone.

## Over-privilege

An identity is over-privileged on a server when it holds a scope naming
`write`, `admin`, `delete`, `manage` or `owner` as a whole word (or `*`)
and every tool the server exposes is annotated `readOnlyHint: true`. A
server with no known tools flags nothing. Which tools an agent actually
calls is not recorded, so the judgement is against what the server
exposes.
