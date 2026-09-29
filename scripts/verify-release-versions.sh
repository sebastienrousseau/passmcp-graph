#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# Fail unless every place that names the version being released names it:
# the CHANGELOG heading, the release highlights, CITATION.cff, the install
# snippets in the README and docs, the version the README's ecosystem
# section states, no status link to a later release, the
# passmcp-reporting release go.mod requires, and the passmcp release the
# acceptance-criteria trace runs. The family moves in lockstep, so every
# one of them is the same version.
#
#   scripts/verify-release-versions.sh v0.0.1
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
tag="${1:-${GITHUB_REF_NAME:-}}"
[ -n "${tag}" ] || { echo "usage: $0 vX.Y.Z" >&2; exit 2; }
ver="${tag#v}"
fail=0
bad() { echo "$*" >&2; fail=1; }

grep -Eq "^## \[${ver}\]" CHANGELOG.md || bad "CHANGELOG.md has no '## [${ver}]' heading"
[ -f "docs/releases/v${ver}.md" ] || bad "docs/releases/v${ver}.md is missing"
grep -Eq "^version: \"?${ver}\"?\$" CITATION.cff || bad "CITATION.cff does not say version ${ver}"

# Every go install line is pinned to this release; at least one must be
# there, so a path that stops matching cannot pass silently.
installs=$(grep -Eoh 'satellion\.com/passmcp[a-z-]*/cmd/passmcp[a-z-]*@[^ `]+' README.md docs/*.md || true)
if [ -z "${installs}" ]; then
  bad "README.md names no go install line to check"
elif grep -v "@v${ver}\$" <<<"${installs}"; then
  bad "the docs pin a go install version other than v${ver}"
fi

grep -Fq "Every component is released at **${ver}**" README.md ||
  bad "README.md's ecosystem section does not state ${ver}"
# A status link names the release a capability shipped in, which may be
# an earlier one; it must never name a release after this one.
while read -r linked; do
  [ -n "${linked}" ] || continue
  newest=$(printf '%s\n%s\n' "${linked}" "${ver}" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)
  [ "${newest}" = "${ver}" ] || bad "README.md links release v${linked}, after v${ver}"
done < <(grep -Eo 'passmcp-graph/releases/tag/v[0-9]+\.[0-9]+\.[0-9]+' README.md | sed 's|.*/v||' | sort -u)

grep -Eq "satellion\.com/passmcp-reporting v${ver}\$" go.mod ||
  bad "go.mod does not require satellion.com/passmcp-reporting v${ver}"
grep -Fq "satellion.com/passmcp/scripts/trace@v${ver} " Makefile ||
  bad "Makefile does not run passmcp's trace at v${ver}"

[ "${fail}" -eq 0 ] && echo "release versions agree on ${ver}"
exit "${fail}"
