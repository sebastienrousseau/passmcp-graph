<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->
<!--
Thanks for contributing to passmcp-graph.

Branch from `main` and target `main`. passmcp-graph reads files where people
keep credentials and answers questions that decide whether a deployment
ships, so a change to what ingest reads or what a query claims is the
change to describe most carefully.
-->

## What this changes

<!-- One or two sentences. What is different after this merges? -->

## Why

<!-- The problem, not the patch. If it fixes an issue, link it: Fixes #123 -->

## How it was verified

- [ ] `make` passes
- [ ] `make test-race` passes
- [ ] The change is covered by a test that fails without it
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`

## Risk

<!--
Name anything that could let a secret reach the store or an output, make a
network request, or change a query's answer or an exit status.
-->

---

- [ ] Commits are signed and carry a DCO `Signed-off-by` trailer (`git commit -s -S`)
