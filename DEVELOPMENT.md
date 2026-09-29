<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Development

The single entry point for working on passmcp-graph: toolchain, how to
reproduce every CI gate locally, and how a release is cut. If a gate fails
in CI and you cannot reproduce it from this file, that is a bug in this
file.

## Requirements

| Tool | Version | Why |
| :--- | :--- | :--- |
| Go | 1.26.8 or later: the `go` directive in `go.mod` | CI tests on that version and on latest stable, on Linux, macOS and Windows; `GOTOOLCHAIN=auto` downloads it |
| make | GNU make | Task runner; `GNUmakefile` adds the install contract |
| golangci-lint | v2.13.2, the version `ci.yml` pins | `make lint` |

Optional, only for the gate that uses it: `markdownlint-cli2`, `codespell`
and `lychee` (the Docs Lint workflow and `pre-commit`), `goreleaser`
(`goreleaser check`), `python3` with `docs/requirements.txt` (the manual),
`zsh` and `fish` (their completion scripts are syntax-checked when
present). `make trace-check` needs the network: it fetches passmcp's trace
tool and reads passmcp's issues.

The Go floor is raised only when a release needs a language feature or a
security fix in the standard library, on a patch release like everything
else pre-1.0, and the changelog says so.

## Reproducing every CI gate

| CI job | Local command |
| :--- | :--- |
| Test (three OSes × two Go versions) | `make test` |
| Race & Shuffled Tests | `make test-race` |
| Coverage Gate (85% per package but `cmd/passmcp-graph`) | `make coverage` |
| Lint | `gofmt -l .` and `make lint`; CI lints on Linux, so also `GOOS=linux make lint` |
| Vulnerability Scan | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |
| Repository Checks | `make trace-check build versions install-smoke`, `go test ./internal/fixture -run TestThePublishedFixtureIsCurrent`, `goreleaser check` |
| Licence Headers | `make spdx-check name-guard` |
| Markdown & Spelling | `make readme-check`, `markdownlint-cli2 '**/*.md'` and `codespell` |
| Link Check | `lychee --offline --include-fragments '**/*.md'` |
| Manual | `pip install --require-hashes -r docs/requirements.txt && mkdocs build --strict` |
| CodeQL, OpenSSF Scorecard | GitHub only |
| DCO check | `git log --format=%B origin/main.. \| grep Signed-off-by` |

`make` with no target runs the gates that need no network, in the order
they fail fastest.

## Coverage

The gate is 85% statement coverage in every package with statements
except `cmd/passmcp-graph`, whose `main` hands `os.Args`, the standard
streams and the clock to `cli.Run` and has no behaviour of its own. A
branch that is hard to reach gets a fixture in `internal/fixture`, never a
test against the network.

`make coverage-json` writes `build/coverage.json`, the shields.io endpoint
document behind the README's coverage badge: statement coverage across the
module, with `cmd/passmcp-graph/main.go` excluded, truncated to one
decimal. The Manual workflow computes the same document on every push to
main and publishes it with GitHub Pages at
<https://sebastienrousseau.com/passmcp-graph/coverage.json>; the number is
never typed by hand.

## Test layout

Each package's tests sit beside it. `internal/fixture` builds valid
attestations, reports and the fixture graph; `testdata/fixture/graph.json`
is what those builders produce (`make fixture-update` regenerates it) and
`testdata/fixture/queries.json` holds hand-written expected answers that
are never regenerated. Tests marked `// AC: SG-0N` prove the acceptance
criteria of [passmcp#4](https://github.com/sebastienrousseau/passmcp/issues/4),
and `make trace-check` fails when a criterion has no passing test.

## Generated artefacts

None are committed. `make build`, `make completions` and `make
coverage-json` write to `build/`; goreleaser writes release archives and
checksums to `dist/`. Both are ignored.

## Release model

The family moves in lockstep: every repository is released at the same
version. A release is cut on a `feat/vX.Y.Z` branch:

1. Move the `## [Unreleased]` entries under a `## [X.Y.Z] — date` heading
   in `CHANGELOG.md`, write `docs/releases/vX.Y.Z.md` with its
   `## Highlights ⭐️`, and move every version-bearing place to X.Y.Z: the
   README's install lines, ecosystem sentence and status links,
   `CITATION.cff`, the passmcp-reporting requirement in `go.mod` and the
   trace version in `Makefile`.
2. `make versions` and `goreleaser check`; optionally run the Release
   workflow's dry run.
3. Push a signed annotated tag `vX.Y.Z` with the message
   `passmcp-graph vX.Y.Z`. The Release workflow checks the tag is on main
   and that every version agrees, builds archives for Linux, macOS and
   Windows on amd64 and arm64, signs `checksums.txt` with cosign keyless
   and attaches SLSA build provenance.
4. Read the tag, the release page and its assets back before calling it
   done.

## Conventions

- Stdout carries results in the selected format; diagnostics go to stderr.
- Every exported identifier is documented.
- Everything read is untrusted input.
