<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Benchmarks

What a command costs, measured end to end: process start to exit, against
the published fixture in `testdata/fixture/` (16 nodes, 15 edges). At this
size, process start dominates, so these numbers say how quickly a CI step
or an editor integration gets an answer, not how the query engine scales.

## Method

```bash
make build
store=$(mktemp -d)
build/passmcp-graph --store "$store" import testdata/fixture/graph.json
hyperfine -N --runs 50 --warmup 3 \
  "build/passmcp-graph --store $store query 'agents -> tools where destructive'" \
  "build/passmcp-graph --store $store analyze"
```

The machine was running other builds during the 2026-09-29 run (load
average 46), so the spread is wide; the minimum is the better guide to
what the command itself costs.

## Results

| Scenario | Median | Min | Mean ± σ | Environment |
| :--- | ---: | ---: | ---: | :--- |
| `query "agents -> tools where destructive"` | 13.9 ms | 9.6 ms | 15.9 ms ± 4.9 ms | Apple A18 Pro, Go 1.27.1, 2026-09-29, load average 46 |
| `analyze` | 20.7 ms | 10.3 ms | 26.5 ms ± 18.8 ms | same |

An earlier measurement, recorded in the 0.0.1 README as the median of 50
runs of the whole process, gave 50.6 ms for the query and 37.4 ms for
`analyze` (Apple A18 Pro, Go 1.27.1, 2026-09-27); its load was not
recorded.

No benchmark is run in CI; these figures are re-measured by hand when a
change could move them.
