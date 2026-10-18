<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Changelog

All notable changes to passmcp-graph are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions are
[Semantic Versioning](https://semver.org/) shaped; before 1.0 every
release moves the patch digit.

## [0.0.8] - 2026-10-18

The family's eighth release. `dashboard` gains `--assets`, for a web
server whose Content-Security-Policy refuses inline code, and is rebuilt
on the Control Centre theme; the other commands, the query language, the
policy file, the graph format and the exit statuses are unchanged from
0.0.7.

### Added

- **`dashboard --assets`** writes the page's stylesheet and script beside
  `--out` as `dashboard.css` and `dashboard.js`, and the page links them,
  for a web server whose Content-Security-Policy refuses inline code.
  Under such a header, as satellion.com sends, the single-file page
  showed as unstyled shapes with no behaviour. The page's own policy
  allows `'self'` instead of `'unsafe-inline'`; the default single file
  is unchanged. See
  [Hosting behind a strict Content-Security-Policy](docs/dashboard.md#hosting-behind-a-strict-content-security-policy).

### Changed

- **In lockstep with passmcp 0.0.8**: passmcp-reporting is required at
  v0.0.8, and passmcp's trace runs at v0.0.8. passmcp-reporting's `graph`
  package is unchanged from v0.0.7, so the graph format and stored graphs
  are unaffected.
- **The dashboard is built on Skeletonic Stylus and the Control Centre
  theme.** Its stylesheet now opens with Skeletonic Stylus 3.0.0's core,
  embedded unmodified (`internal/dashboard/third_party`, MIT OR
  Apache-2.0, recorded in `REUSE.toml`), and the page follows the
  Control Centre theme's tokens and layout: a sidebar that becomes a menu
  under the top bar below 1024 pixels (replacing the phone tab bar), the
  critical-finding banner, the status ring with the clients and servers
  joined to it by flows in their status colour, KPI tiles, severity
  pills, scrollable table regions and the mitigation steps. A client's
  entry takes the worst status among the servers it is configured to
  reach and names it. Every figure, filter, link, export and theme
  setting works as before, in the single file and with `--assets`, and
  no element carries a style attribute. The colours are the theme's,
  and a test now also checks each severity pill's text against its tint
  and that the system and switched dark modes share one palette.
- The 0.0.1 to 0.0.4 headings in this file use a hyphen between version
  and date, like the later ones, instead of an em dash, and
  `DEVELOPMENT.md`'s release steps name that heading form.
- The README's capability table says the dashboard was released in
  0.0.7, and shell completions and release archives in 0.0.2; those
  rows still said "Not yet released".

### Fixed

- **The dashboard sets no `style` attribute.** Its template carried 40,
  some repeated per row, for colours, margins, animation order and the
  score bars, and a
  Content-Security-Policy without `style-src 'unsafe-inline'` blocks
  every one. Classes, `data-i` order read by the script, and SVG bars
  replace them, in both the single file and `--assets` output, and the
  page looks the same.
- **`dashboard` names a server one way.** The top issue, finding and
  evidence-gap titles, the mitigation steps and the links to a server
  used its endpoint (or, with `--share-safe`, a placeholder such as
  `http-server-1`) while the command centre used the name it reports, so
  "crm" in the ring was `https://crm.example/mcp` in the banner. They now
  all use the reported name, or the endpoint when there is none; in
  `--format json`, `server.name` in findings, evidence gaps, paths and
  grants changes the same way. The server's details panel, the evidence
  table and the topology still show the endpoint, and search still finds
  a server's items by it.
- **The ring's key reads with its counts**: "1 with stale evidence"
  instead of "1 Stale evidence".
- **A server card's status line ends in an ellipsis** when it does not
  fit, instead of being cut off mid-word, as "No failing server checks"
  was at 1440 pixels wide; the full status is in its tooltip.

## [0.0.7] - 2026-10-17

The family's seventh release. One command is added, `dashboard`, and
`ingest` follows passmcp-reporting's subject-digest rule for sha512; the
other commands, the query language, the policy file, the graph format and
the exit statuses are unchanged from 0.0.5. passmcp 0.0.6 had no
passmcp-graph release, so this member goes from 0.0.5 to 0.0.7.

### Added

- **`dashboard`** writes the graph as one self-contained HTML page for
  an operator to read or share, opening on a command centre: clients and
  servers either side of a ring of configured tool paths, the totals,
  attention items by severity, a mitigation path for the first item, and
  quick actions to save a PDF report or download the share-safe JSON. It
  leads with the item to look at first,
  with its evidence age, the client configured to reach the server, and
  links to the evidence and the affected path; then the totals; the
  configured tool paths split into declared read-only, declared mutating
  and unknown; the servers by status; an exact, animated client -> server
  -> tool topology with a path table as its text equivalent; attention
  items, as findings and evidence gaps; and details panels for every
  server, client, identity and finding. Search and filters scope every
  widget alike, every count says whether it covers the whole snapshot or
  the current selection, and the page prints the `query` command that
  returns the paths it shows, tested against the query engine for every
  filter combination. `--open` shows it in the default
  browser, `--out FILE` keeps it, `--share-safe` replaces endpoints,
  identities, local paths and token-shaped strings, `--stale-days` sets
  the staleness threshold (default 30) and `--format json` prints the
  summary, with each headline metric's unit and denominator. It follows
  the system's light or dark mode with a switch to override it, reflows
  down to 320 CSS pixels, honours reduced motion and forced colours, and
  meets WCAG 2.2 AA contrast in both themes. The page loads nothing, and
  its Content-Security-Policy has the browser enforce that. The exit
  status is 0 even when the policy is violated; `check` stays the gate.

### Changed

- **In lockstep with passmcp 0.0.7**: passmcp-reporting is required at
  v0.0.7, and passmcp's trace runs at v0.0.7.
- **`ingest` accepts a sha512 subject digest.** passmcp-reporting 0.0.6
  changed how an attestation's subject is validated
  ([passmcp-reporting#10](https://github.com/sebastienrousseau/passmcp-reporting/issues/10)):
  a statement whose subject carries only a sha512 digest is now ingested,
  and one whose sha512 digest does not match its server is now refused
  even when its sha256 digest does. A server's node ID is still its sha256
  digest, so graphs already stored keep their IDs.
- **The manual carries passmcp's flame** as its logo and favicon, as
  passmcp's own manual does; the dashboard page inlines the same mark.
- **GitHub Actions updated**: `github/codeql-action` 4.38.3,
  `actions/upload-artifact` 7.0.2 and `ossf/scorecard-action` 2.5.0
  ([#8](https://github.com/sebastienrousseau/passmcp-graph/pull/8)).

### Fixed

- **The manual lists the 0.0.5 release notes.** 0.0.5 added
  `docs/releases/v0.0.5.md` but not its entry in the mkdocs navigation.
- **`CITATION.cff` gives 0.0.5's release date**, 2026-10-12; the 0.0.5
  release moved the version and left the date at 0.0.4's.

### Security

- **Build with Go 1.26.9.** The `go` directive moves from 1.26.8, which
  `govulncheck` reports as reachable by five standard-library advisories
  in `net/http` and its HTTP/2 server (GO-2026-6603, GO-2026-6611,
  GO-2026-6612, GO-2026-6613 and GO-2026-6617), all fixed in 1.26.9.
  Release binaries are built with this toolchain.

## [0.0.5] - 2026-10-12

### Fixed

- **The release page is published on a tag push again.** The workflow
  passed `${GITHUB_REF_passmcp-graph}` as the tag, which the shell reads
  as an unset variable with the default `graph`, so v0.0.4's page step
  failed looking for `docs/releases/graph.md`. It now passes
  `${GITHUB_REF_NAME}`.

### Changed

- **In lockstep with passmcp 0.0.5**: passmcp-reporting is required at
  v0.0.5, and passmcp's trace runs at v0.0.5.

## [0.0.4] - 2026-09-30

The family's fourth release. The commands, the query language, the policy
file, the graph format and the exit statuses are unchanged from 0.0.3;
this member moves with the family and gains a README demo.

### Added

- **A README demo**, rendered from `.github/demo.tape` by `make demo`:
  importing a fixture graph, the analysis it finds (an inherited critical
  finding and an over-privileged agent), and two queries answered offline.

### Changed

- **Release pages are published in the family layout** by the release
  workflow itself (Highlights, What's Changed, Checksums, Full Changelog),
  so no page is rewritten by hand after a release.
- **In lockstep with passmcp 0.0.4**: passmcp-reporting is required at
  v0.0.4, and passmcp's trace runs at v0.0.4; neither changes what this
  repository reads or writes.

## [0.0.3] - 2026-09-30

The family's third release. The commands, the query language, the policy
file, the graph format and the exit statuses are unchanged from 0.0.2;
this member moves with the family and gains fuzz testing.

### Added

- **Fuzz targets.** `FuzzParse` checks that any query either fails with
  an error or parses to a tree that survives being written back out and
  parsed again, and that evaluating it never panics.
  `FuzzConfigNeverStoresASecret` places a fuzzer-chosen secret everywhere
  a client configuration can carry a credential and checks it never
  reaches the graph.

### Changed

- **passmcp-reporting is required at v0.0.3**, and passmcp's trace runs
  at v0.0.3; neither changes what this repository reads or writes.

## [0.0.2] - 2026-09-29

The family's second release. The 0.0.1 commands, the query language, the
policy file, the graph format and the exit statuses are unchanged; what is
new is the `completion` command and how passmcp-graph is installed and
released.

### Added

- **Shell completions.** `passmcp-graph completion bash|zsh|fish` prints a
  completion script generated from the command table, and `make install`
  installs all three with the binary under `PREFIX` and `DESTDIR`
  (`GNUmakefile`).
- **Release archives.** A tag-triggered Release workflow builds archives
  for Linux, macOS and Windows on amd64 and arm64 with goreleaser, signs
  `checksums.txt` with cosign keyless and attaches SLSA build provenance.
- **Repository standard.** OpenSSF Scorecard, a coverage badge published
  with the manual, `make versions` checking that every version-bearing
  file names the same release, and the family's community, development
  and architecture documents.

### Changed

- **passmcp-reporting is required at v0.0.2**, the family's lockstep
  version, instead of a pre-release commit.
- **Complexity ceilings tightened** to cyclomatic 10, cognitive 15 and 60
  lines per function; `query.Eval`, `check` and the licence-header sweep
  were split to meet them, with no change in behaviour.

## [0.0.1] - 2026-09-29

The first release.

### Added

- **A local graph of agents, MCP servers, tools and identities**, built from
  passmcp's attestations, reports and MCP client configurations: `ingest`,
  `analyze`, `query`, `check`, `export` and `import`, all offline, with a
  policy gate for CI. The store holds no secret, however a configuration
  carried it.
