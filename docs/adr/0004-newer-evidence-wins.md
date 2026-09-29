<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# 0004: Newer evidence replaces older, and older never replaces newer

**Status:** Accepted

## Context

`ingest` is run again as evidence arrives, and attestation directories are
not ordered: a run can read last month's attestation after this week's.

## Decision

A server's verdict comes from its newest attestation, ordered by the
attestation's `ranAt`. An attestation at least as new as the stored one
replaces it; an older one is recorded but does not replace the verdict.
Ingesting the same evidence twice changes nothing.

## Consequences

- The order in which files are ingested does not change the answer.
- A fixed server stops passing its risk to agents on the next ingest of
  an attestation that no longer fails the check.
- `TestAttestationsBecomeServersAndIngestingTwiceChangesNothing` and
  `TestCriticalFindingsAreInheritedAndFixingThemRemovesTheRisk` enforce
  both properties.
