# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# The README demo: passmcp-graph importing the published fixture graph
# (testdata/fixture/graph.json), naming the inherited risk and the
# over-privileged identity in it, and answering two path queries. Rendered
# by `make demo`, which builds the binary and runs this in a scratch
# directory under build/demo, so the graph store it writes is thrown away.
Output .github/demo.gif
Require passmcp-graph
Set Shell "bash"
Set FontSize 13
Set Width 1200
Set Height 290
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "PS1='$ '; cd build/demo/work; clear" Enter
Show

Type "passmcp-graph import testdata/fixture/graph.json" Sleep 400ms Enter
Sleep 1500ms
Type "passmcp-graph analyze" Sleep 400ms Enter
Sleep 2s
Type `passmcp-graph query "agents -> tools where destructive and server.auth = none"` Sleep 400ms Enter
Sleep 2s
Type `passmcp-graph query "agents -> servers where not attested"` Sleep 400ms Enter
Sleep 4s
