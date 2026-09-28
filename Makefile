# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only

.PHONY: all build test test-race coverage vet lint format spdx-check readme-check \
        fixture-update trace trace-check trace-refresh help name-guard

# Every gate CI runs that needs no network, in the order the cheap ones fail
# first.
all: format vet lint spdx-check readme-check test

build:
	CGO_ENABLED=0 go build -trimpath -o build/passmcp-graph ./cmd/passmcp-graph

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on -count=1 ./...

# The gate is 85% statement coverage in every package with statements,
# except cmd/passmcp-graph; ci.yml says why.
coverage:
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

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
TRACE := go run satellion.com/passmcp/scripts/trace@v0.0.1 -repo sebastienrousseau/passmcp-graph

trace:
	$(TRACE)

trace-check:
	$(TRACE) -check -run -report-dir build/trace

trace-refresh:
	$(TRACE) -refresh

help:
	@printf '%s\n' "targets: all build test test-race coverage vet lint format spdx-check readme-check fixture-update trace trace-check trace-refresh"

# The project was renamed to passmcp: the old name may appear only in the
# provenance line (scripts/name-guard.sh).
name-guard:
	./scripts/name-guard.sh
