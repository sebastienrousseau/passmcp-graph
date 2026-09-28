<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Contributing

passmcp-graph reads files where people keep credentials and answers
questions that decide whether a deployment ships. Keep both in mind for
every change: nothing secret may reach the store, and no answer may claim
more than the evidence shows.

## Getting started

1. Fork and clone the repository.
2. Install **Go** at the version the `go` directive in `go.mod` names,
   **make**, and [golangci-lint](https://golangci-lint.run) v2.
3. Create a branch from `main` and open the pull request against `main`.
   Every workflow filters on `pull_request: branches: [main]`, so a PR
   aimed elsewhere runs no CI.
4. Make the change.
5. Verify:

   ```bash
   make            # format, vet, lint, headers, README, tests
   make test-race
   ```

## Commits

**Sign your commits cryptographically and add a DCO sign-off trailer.**
Both are required and both are enforced: signing proves who authored the
commit; the sign-off (`git commit -s`) certifies the
[Developer Certificate of Origin](https://developercertificate.org).
Merge commits are exempt from the DCO check.

Use [Conventional Commits](https://www.conventionalcommits.org/) with an
imperative subject: `feat(query): compare list fields by membership`,
not `Added lists`.

## What a change needs

- **A test that fails without it.** Build graphs with
  `internal/fixture`; never point a test at the network.
- **85% statement coverage in every package** but `cmd/passmcp-graph`.
- **The published fixture kept honest.** A change to what a query
  returns updates `testdata/fixture/queries.json` by hand; a change to
  the fixture builders is followed by `make fixture-update`.
- **An entry under `## [Unreleased]` in `CHANGELOG.md`.**
- **A reason in the commit for any new dependency.** The module has two.

## Pull request checklist

- [ ] `make` and `make test-race` pass
- [ ] The change is covered by a test that fails without it
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`
- [ ] Commits are signed and carry a DCO sign-off
