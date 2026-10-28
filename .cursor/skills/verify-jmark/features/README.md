# jmark verification map

Maintained source for verifying user-facing jmark CLI behavior. Read this index, then the matching feature file.

## Baseline preconditions

- Repo root is the `jmark` checkout (directory containing `justfile`, `cmd/jmark/`, `runtimes/`, `families/`).
- `just`, `go`, `bash`, and `hyperfine` are on `PATH`. `node`, `bun`, `python3` are needed only for the runtimes being driven.
- `just doctor` exits 0 with `doctor: OK`.
- Mutating drives use a unique id `verify-<run-id>`; never leave it behind after cleanup.
- Evidence goes under `.cursor/skills/verify-jmark/artifacts/<run-id>/` and must survive cleanup.

## Driving conventions

- Start from a clean doctor-OK catalog unless a feature says otherwise.
- Treat every `just` invocation as literal; keep ids and runtime args unchanged.
- Capture via `helpers/capture.sh`.
- Remove only projects this run created; never delete `sum-large-doubles` or evidence.

## Proof and skip reporting

- Proof includes command, stdout, stderr, and exit code.
- Bench proof: each stdout line is one JSON object with `project`, `runtime`, `schema_version=2` and metric definitions, non-empty `results` (`name`, `metrics`, `observations`, `sink`), and `meta` with collector and host metadata; workload contains parameters and counts.
- Create proof includes on-disk template files plus `just list` membership, then successful cleanup.
- Record feature id and entry point with every artifact.
- Report a runtime whose toolchain is missing as skipped with the missing command; do not call it verified.

## Feature entry contract

Each feature file: H1 + one paragraph, then H2s `Sub-features`, `How to get to it (user POV)`, `Driving it with just`, `Gotchas`.

## Features

- [Doctor catalog health](./doctor.md) — host deps, registry, project layout, probe availability.
- [List projects](./list.md) — project ids from `benches/*/AGENTS.md`.
- [List runtimes](./runtimes.md) — runtime ids, family, toolchain availability.
- [Run bench](./bench.md) — external hyperfine timing, NDJSON per runtime, all-or-nothing.
- [Create project](./create.md) — scaffold per family, idempotent, then cleanup.

- [Report UI](./report.md) — standalone reports, filters, chart scales, and responsive layout.

- [Public results](./public-results.md) — private defaults, visibility changes, Pages output, and source snapshots.
