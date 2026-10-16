<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Dashboard

`passmcp-graph dashboard` writes the whole graph as one HTML page: what each
agent can reach, how each MCP server was last attested, and what the
analysis and an optional policy found. The page is a file, not a server. It
loads no script, font or stylesheet from anywhere, so it opens offline, can
be attached to a ticket or a pull request, and can be archived as evidence
of the estate on the day it was generated.

```bash
passmcp-graph dashboard --open                     # write a temporary page and open it
passmcp-graph dashboard --out estate.html          # keep it, to attach or archive
passmcp-graph dashboard --policy policy.yaml --stale-days 14 --out estate.html
passmcp-graph dashboard > estate.html              # or to stdout, for a pipeline
passmcp-graph dashboard --format json | jq .totals
```

`--open` hands the file to the system's opener (`open` on macOS,
`xdg-open` on Linux and the BSDs, the file handler on Windows), which shows
it in the default browser. That starts a local program; it is not a network
request, and the page itself fetches nothing. The page names your
configuration files, so `--out` and the temporary file are created readable
by you alone.

| Flag | Default | Meaning |
| :--- | :--- | :--- |
| `--out FILE` | stdout | Write the page to `FILE` instead; the path is printed on stderr |
| `--open` | off | Open the page in the default browser; without `--out`, it is written to a temporary file first |
| `--share-safe` | off | Replace identifying details before writing; see [Sharing the page](#sharing-the-page) |
| `--policy FILE` | none | Also check a [policy](policy.md) and list its violations on the page |
| `--stale-days N` | `30` | An attestation older than `N` days counts as stale |
| `--format html\|json` | `html` | The page, or the summary it is built from |

The exit status is 0 whenever the page is written, violations or not: the
dashboard is a report, and `check` is the gate. Run both in CI when you
want a red build and a page to read.

## What the page shows

The page reads top to bottom from mode and scope, to the item to look at
first, to the summary, to the relationships and findings, to the full
evidence.

| Section | Derived from |
| :--- | :--- |
| Look at this first | The first attention item: an inherited critical finding, a policy violation, an over-privileged identity, then a server with missing or stale evidence. It names the client configured to reach the server and the evidence age, links to the evidence and the affected path, and offers a command that re-reads the local graph |
| Totals | Servers with diagnostic evidence and the mean available score; configured clients and identities; tools and how many are potentially destructive; attention items, as findings and evidence gaps |
| Overview | The configured tool paths split into declared read-only, declared mutating and unknown; how many are potentially destructive, declared or presumed; and the servers by status, each counted once |
| Topology | Every configured client -> server -> tool relationship, exactly, with a path table as its text equivalent |
| Attention items | Findings from `analyze` and `check --policy`, and evidence gaps: servers with no attestation or a stale one |
| Servers, identities, clients | One bubble per server, its area proportional to its tools; each identity and the clients that reach its servers; each client and its reach |
| Paths, server evidence | Every path, and every server's figures, as tables |
| Details | One panel per server, client and identity |

A server's status is its own diagnostics and evidence: critical findings,
no evidence, stale evidence, or no failing server checks. Identity and
policy findings are attention items of their own, so a server with no
failing checks can still carry one; its row and its panel count them as
related findings.

## Search, filters and details

The search box and the status and tool chips filter the topology, the
attention items, the bubbles, the clients, the identities and both tables
together; choosing a client in the topology, or "Show affected path" in
the banner, narrows them to that client's paths. A line under the filters
says what is shown, as "2 of 6 paths", and that the totals and overview
count the whole snapshot. The filter state is kept in the browser's
history entry, not the address, so reloading or going back restores it
and nothing typed into the search reaches a URL.

Every server, client, identity, finding and evidence gap has a details
panel. A link anywhere on the page opens it; its address, such as
`#srv-2`, is an index, not a name, and reopens it on reload. Each finding
shows its path, where its evidence comes from, what is not known, and a
read-only next step.

## Sharing the page

`--share-safe` replaces what identifies the estate before anything is
written: server endpoints and commands become `http-server-1` or
`stdio-server-2`, identity labels, client IDs and issuers become
`identity-1` and `issuer-1`, configuration paths keep only their file
name, and anything left that looks like a local path, an email address or
a token is replaced. Client products, tool names, check IDs, the names
servers report, grades, scores and attestation digests remain, so the
page can still be read. It applies to both formats, and the page says it
is share-safe. Commands on a share-safe page name the placeholders, so
they reproduce nothing until run against your own store.

Redaction removes the values passmcp-graph knows to be identifying and
the shapes it recognises; read a share-safe page before sending it.

## What the numbers count

A client is a configuration that names servers, not a running agent, and
reach is configured, not observed: no tool call is recorded. Each
headline figure is defined once, with its unit and denominator, and
`--format json` prints the definitions under `metrics`:

| Metric | Unit | Counts |
| :--- | :--- | :--- |
| `tools` | tool | Distinct per server: the same name on two servers is two tools |
| `paths` | path | Client -> server -> tool paths derived from configuration |
| `paths.changing` | path, of `paths` | Paths whose tool lacks `readOnlyHint: true`, unannotated tools included |
| `paths.destructive` | path, of `paths` | Paths to potentially destructive tools: `destructiveHint: true`, or no `destructiveHint` on a tool not declared read-only |
| `paths.destructive.declared` | path, of `paths` | Paths whose tool declares `destructiveHint: true` |
| `paths.unannotated` | path, of `paths` | Paths whose tool declares no `readOnlyHint` |
| `servers.evidence` | server, of `servers` | Servers with at least one attestation, whatever its age |
| `servers.fresh` | server, of `servers` | Servers whose latest attestation is within `--stale-days` |
| `score.mean` | score | Mean over scored servers only; the page names how many were scored |

Percentages have one decimal and drop a trailing `.0`: 4 of 6 is 66.7%,
3 of 5 is 60%, and a zero denominator is "n/a".

A tool's behaviour is what its server declares: read-only, mutating, or
unknown when it declares no `readOnlyHint`. A tool is potentially
destructive when it declares `destructiveHint: true` (declared), or is not
declared read-only and declares no `destructiveHint`, which the MCP
specification reads as possibly destructive (presumed). A tool with no
annotations at all is therefore both unknown and presumed destructive;
the page names both. Annotations are what a server says about its tools,
not observed behaviour.

Each server counts once in the overview, in the first status that
applies. Staleness is also tracked on its own, so a server that is both
critical and stale shows both in the bubbles and the tables. A server
bubble's area, not its radius, is proportional to the tools it exposes.

Every figure comes from the same query and analysis code as `query`,
`analyze` and `check`, so the page cannot disagree with them.

## Themes, motion and accessibility

The page follows the system's light or dark mode; the System, Light and
Dark switch in its top bar overrides it and is remembered in the browser.
It lays out for phones with a bottom tab bar and reflows at 320 CSS
pixels, with the topology and the tables scrolling inside their own
panels. Flows, counters and bars animate, and every animation stops when
the system asks for reduced motion. Its text and status colours meet
WCAG 2.2 AA contrast in both themes, a test computes it from the page's
own colours, and forced-colours mode keeps every distinction. Every
figure is in the markup, so the page reads the same with scripts off, and
it prints without the controls.

Labels on the page are read from client configurations and attestations;
every one is escaped, so a server or agent name cannot inject markup. A
Content-Security-Policy in the page has the browser refuse any network
load, so the "loads nothing" guarantee does not rest on the markup alone.
