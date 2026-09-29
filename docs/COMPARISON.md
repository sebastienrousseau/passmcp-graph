<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Comparison

passmcp alone answers "is this server safe"; a general graph database
answers anything you load into it. passmcp-graph answers the questions in
between — which agents reach what, under which identities — from passmcp's
own evidence, with no service to run.

| Approach | Reads passmcp's evidence directly | Needs a running service | Policy gate with an exit status | Holds agents and identities | Network access |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **passmcp-graph** | yes | no | yes: `check --policy`, exit 1 on a violation | yes | none |
| `passmcp check`, one server at a time | yes | no | per server | no agents | the server under test |
| Neo4j or another graph database, loaded from `passmcp-graph export --format cypher` | through the export | yes | write it yourself | through the export | the database's |

## Evidence for each cell

- **Reads passmcp's evidence directly.** `ingest` recognises passmcp
  attestations and `passmcp check --output json` reports
  ([Ingesting evidence](ingest.md)); a file it does not recognise is
  skipped and named.
- **No running service.** The store is one JSON file,
  `.passmcp-graph/graph.json`, read and written by the command that runs.
- **Policy gate.** `check --policy FILE` exits 1 and names every forbidden
  path, 0 when there is none, and 2 on an error ([Policies](policy.md)).
- **Agents and identities.** Agents come from client configurations and
  identities from passmcp reports; `passmcp check` evaluates one server and
  does not record which agents use it.
- **Network.** `TestNoCommandContactsTheNetwork` requires zero requests
  from every command ([ADR 0001](adr/0001-no-network.md)).

A graph database is the better choice for ad hoc exploration across data
passmcp does not produce; `export --format cypher` and `--format graphml`
exist so the two can be combined.
