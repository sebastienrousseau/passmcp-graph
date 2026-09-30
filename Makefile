# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only

.PHONY: all build test test-race coverage coverage-json vet lint format spdx-check readme-check \
        fixture-update trace trace-check trace-refresh help name-guard completions versions

# Every gate CI runs that needs no network, in the order the cheap ones fail
# first.
all: format vet lint spdx-check readme-check name-guard versions test

# VERSION is what `passmcp-graph version` prints; a packager sets it to the
# release being built. Release archives get it from goreleaser.
VERSION ?= dev
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-X satellion.com/passmcp-graph/internal/cli.Version=$(VERSION)" \
	  -o build/passmcp-graph ./cmd/passmcp-graph

# Shell completions, generated from the command table by the binary itself,
# into build/completions. bash is syntax-checked here; zsh and fish when
# present.
completions: build
	mkdir -p build/completions
	build/passmcp-graph completion bash > build/completions/passmcp-graph.bash
	build/passmcp-graph completion zsh > build/completions/_passmcp-graph
	build/passmcp-graph completion fish > build/completions/passmcp-graph.fish
	bash -n build/completions/passmcp-graph.bash
	if command -v zsh >/dev/null; then zsh -n build/completions/_passmcp-graph; fi
	if command -v fish >/dev/null; then fish --no-execute build/completions/passmcp-graph.fish; fi

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on -count=1 ./...

# The gate is 85% statement coverage in every package with statements,
# except cmd/passmcp-graph; ci.yml says why.
coverage:
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# The shields.io endpoint document behind the README's coverage badge:
# statement coverage across the module, as CI measures it. The Manual
# workflow publishes it with GitHub Pages as coverage.json.
coverage-json: coverage
	@mkdir -p build
	go run ./scripts/coveragebadge -profile coverage.out -exclude /cmd/passmcp-graph/main.go > build/coverage.json
	@cat build/coverage.json

# Every file that names the version names the newest CHANGELOG release, and
# passmcp-reporting and passmcp's trace are required at that same release.
versions:
	scripts/verify-release-versions.sh "v$$(grep -Eo '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | tr -d '#[] ')"

vet:
	go vet ./...

lint:
	golangci-lint run ./...

format:
	gofmt -l -w .

# The README follows the portfolio template: headings in order, no
# unresolved {{VARIABLES}} (AGENTS.md §7.3).
readme-check:
	scripts/readme-check.sh

spdx-check:
	go run ./scripts/spdx_sweep.go

# Regenerate testdata/fixture/graph.json after a deliberate change to the
# fixture builders. The query fixture's expected paths are hand-written and
# are not touched.
fixture-update:
	go test ./internal/fixture -run TestThePublishedFixtureIsCurrent -update

# The acceptance-criteria trace. This repository's user stories are
# filed in passmcp's issues and name it
# ("**Repository:** sebastienrousseau/passmcp-graph"), so the trace is passmcp's own
# tool, run at a pinned version: a closed story with an untested
# criterion fails here, and passmcp's stories are left to passmcp.
TRACE := go run satellion.com/passmcp/scripts/trace@v0.0.3 -repo sebastienrousseau/passmcp-graph

trace:
	$(TRACE)

trace-check:
	$(TRACE) -check -run -report-dir build/trace

trace-refresh:
	$(TRACE) -refresh

help:
	@printf '%s\n' "targets: all build completions test test-race coverage coverage-json vet lint format" \
	  "         spdx-check readme-check name-guard versions fixture-update trace trace-check trace-refresh" \
	  "GNUmakefile: install uninstall install-smoke (PREFIX, DESTDIR)"

# A retired product name may not appear anywhere in the tree
# (scripts/name-guard.sh).
name-guard:
	./scripts/name-guard.sh
