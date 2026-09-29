#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# Fail unless README.md follows the portfolio README template (AGENTS.md
# §7.3): a centred plain-text h1, the template's second-level headings in
# the template's order, and no unresolved {{UPPER_SNAKE_CASE}} variable
# outside code. The heading list is the template's; update both together.
# It also holds the family's shared parts in place: the seven badges in
# their order and style, a link to every component in the ecosystem
# table, and no vague status ("Shipping", "Planned") in a table.
#
#   scripts/readme-check.sh [README.md]
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
readme="${1:-README.md}"
[ -f "${readme}" ] || { echo "readme-check: ${readme} not found" >&2; exit 1; }

project=$(sed -n 's|^<h1 align="center">\([^<]*\)</h1>$|\1|p' "${readme}" | head -1)
[ -n "${project}" ] || { echo "readme-check: no centred plain-text <h1 align=\"center\">name</h1>" >&2; exit 1; }

expected=$(cat <<LIST
Contents
Install
Requirements
Quick Start
The ${project} ecosystem
Capabilities at a glance
Ecosystem comparison
Benchmarks
Features
Configuration
Examples
When not to use ${project}
Development
Security
Documentation
Stability guarantees
License
LIST
)

# Second-level headings and template tokens, both outside fenced code.
outside_code=$(awk '/^[[:space:]]*(```|~~~)/ { fence = !fence; next } !fence' "${readme}")
actual=$(sed -n 's/^## //p' <<<"${outside_code}")
if [ "${actual}" != "${expected}" ]; then
  echo "readme-check: second-level headings differ from the template:" >&2
  diff <(echo "${expected}") <(echo "${actual}") >&2 || true
  exit 1
fi

# Inline code is content too: strip it before looking for variables.
# shellcheck disable=SC2001,SC2016 # a regex substitution with literal backticks
tokens=$(sed 's/`[^`]*`//g' <<<"${outside_code}" | grep -oE '\{\{ *[A-Z][A-Z0-9_]* *\}\}' || true)
[ -z "${tokens}" ] || { echo "readme-check: unresolved template variables: ${tokens}" >&2; exit 1; }

# The badge row: seven shields.io badges before the Contents, in the
# family's order, every one in the for-the-badge style.
head_block=$(sed '/^## Contents/q' "${readme}")
alts=$(grep -o '<img src="https://img.shields.io[^>]*alt="[^"]*"' <<<"${head_block}" | sed 's/.*alt="\([^"]*\)"/\1/' | cut -d: -f1 | tr '\n' '|')
want="Build|Coverage|Release|Docs|OpenSSF Scorecard|License|"
case "${alts}" in
  "${want}"*) ;;
  *) echo "readme-check: badges are '${alts}', want '${want}<toolchain>|'" >&2; exit 1 ;;
esac
[ "$(grep -o '<img src="https://img.shields.io' <<<"${head_block}" | wc -l | tr -d ' ')" = 7 ] ||
  { echo "readme-check: the badge row must hold exactly seven badges" >&2; exit 1; }
if grep -o '<img src="https://img.shields.io[^"]*"' <<<"${head_block}" | grep -v 'style=for-the-badge'; then
  echo "readme-check: every badge must use style=for-the-badge" >&2; exit 1
fi

# The ecosystem table links every component of the family.
for repo in passmcp passmcp-reporting passmcp-server passmcp-action passmcp-graph \
            passmcp-registry passmcp-lsp passmcp-census; do
  grep -Fq "| [${repo}](https://github.com/sebastienrousseau/${repo}) |" "${readme}" ||
    { echo "readme-check: the ecosystem table does not link ${repo}" >&2; exit 1; }
done
grep -Fq "| [satellion.com](https://github.com/sebastienrousseau/satellion.github.io) |" "${readme}" ||
  { echo "readme-check: the ecosystem table does not link satellion.com" >&2; exit 1; }

# A status says which release a capability is in, or that it is not yet
# released; a vague word in a table cell says neither.
if grep -nE '\| *(Shipping|Shipped|Available|Planned) *\|' <<<"${outside_code}"; then
  echo "readme-check: a table uses a vague status; say 'Released in X.Y.Z' or 'Not yet released'" >&2; exit 1
fi

echo "readme-check: ${readme} follows the template (${project})"
