# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# The Unix install contract. GNU make reads this file before Makefile, so
# every developer target is pulled in from there and this file adds only
# install, uninstall and the staged smoke test packagers rely on.
#
# PREFIX defaults to /usr/local per the FHS; DESTDIR stages the tree
# elsewhere without changing the paths compiled into it, which is how
# distribution packages are built. For a home-directory install:
#   make install PREFIX=$$HOME/.local
#
# passmcp-graph generates its shell completions from its command table; it
# generates no manual page, so none is installed.

include Makefile

PREFIX ?= /usr/local
DESTDIR ?=
BINDIR = $(DESTDIR)$(PREFIX)/bin
DOCDIR = $(DESTDIR)$(PREFIX)/share/doc/passmcp-graph
BASHCOMPDIR = $(DESTDIR)$(PREFIX)/share/bash-completion/completions
ZSHCOMPDIR = $(DESTDIR)$(PREFIX)/share/zsh/site-functions
FISHCOMPDIR = $(DESTDIR)$(PREFIX)/share/fish/vendor_completions.d

.PHONY: install uninstall install-smoke

install: build completions
	install -d $(BINDIR)
	install -m 0755 build/passmcp-graph $(BINDIR)/passmcp-graph
	install -d $(BASHCOMPDIR) $(ZSHCOMPDIR) $(FISHCOMPDIR)
	install -m 0644 build/completions/passmcp-graph.bash $(BASHCOMPDIR)/passmcp-graph
	install -m 0644 build/completions/_passmcp-graph $(ZSHCOMPDIR)/_passmcp-graph
	install -m 0644 build/completions/passmcp-graph.fish $(FISHCOMPDIR)/passmcp-graph.fish
	install -d $(DOCDIR)
	install -m 0644 README.md CHANGELOG.md LICENSE SECURITY.md $(DOCDIR)/

uninstall:
	rm -f $(BINDIR)/passmcp-graph
	rm -f $(BASHCOMPDIR)/passmcp-graph
	rm -f $(ZSHCOMPDIR)/_passmcp-graph
	rm -f $(FISHCOMPDIR)/passmcp-graph.fish
	rm -rf $(DOCDIR)

# Stage an install under a temporary DESTDIR, check every file lands where
# the FHS says and the binary runs, then uninstall and check nothing is
# left behind.
install-smoke:
	@set -e; stage=$$(mktemp -d); trap 'rm -rf "$$stage"' EXIT; \
	$(MAKE) --no-print-directory install DESTDIR="$$stage" PREFIX=/usr >/dev/null; \
	for f in usr/bin/passmcp-graph \
	         usr/share/bash-completion/completions/passmcp-graph \
	         usr/share/zsh/site-functions/_passmcp-graph \
	         usr/share/fish/vendor_completions.d/passmcp-graph.fish \
	         usr/share/doc/passmcp-graph/README.md \
	         usr/share/doc/passmcp-graph/LICENSE; do \
	  test -f "$$stage/$$f" || { echo "install-smoke: missing $$f" >&2; exit 1; }; \
	done; \
	test -x "$$stage/usr/bin/passmcp-graph" || { echo "install-smoke: the binary is not executable" >&2; exit 1; }; \
	"$$stage/usr/bin/passmcp-graph" version >/dev/null; \
	$(MAKE) --no-print-directory uninstall DESTDIR="$$stage" PREFIX=/usr >/dev/null; \
	left=$$(find "$$stage" -type f); \
	[ -z "$$left" ] || { echo "install-smoke: uninstall left $$left" >&2; exit 1; }; \
	echo "install-smoke: install and uninstall are correct under DESTDIR"
