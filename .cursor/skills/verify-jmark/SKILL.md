---
name: verify-jmark
description: "Drive the jmark CLI catalog (just doctor/list/runtimes/create/bench) the way a user would — prove the host times pure-logic sources externally and emits schema-valid NDJSON. Use when proving jmark behavior, after host/runtimes/families/schema changes, or when dogfooding create-verification-skill against this repo."
---

# verify-jmark

jmark is a short-lived CLI catalog (no server). Launch means ensure deps and cwd; each drive is an isolated `just` invocation from the repo root.

Evidence root (survives cleanup): `.cursor/skills/verify-jmark/artifacts/<run-id>/`

## Launch

From the repo root (`jmark/`):

```bash
command -v just && command -v go && command -v hyperfine && command -v mise
just doctor
```

Ready when `just doctor` prints `doctor: OK` and exits 0. Runtime toolchains (`node`, `bun`, `python3`) are optional; `just runtimes` shows which are found. There is no background process to keep alive.

Teardown: none for the catalog itself. For drives that create a project, remove its directory (see Cleanup). Never delete evidence under `artifacts/`.

## Doctor

```bash
just doctor
```

Require: exit 0, `doctor: OK`, `doctor: registry ok`, and `doctor: project ok <id> (runtimes: …)` for every project. If doctor fails, fix the registry or project layout before any Drive.

## Drive

Work from the repo root. Prefer the helpers in `helpers/`. Capture stdout/stderr/exit code for every command into the run's evidence dir.

Stable handles:

- Commands: `just doctor`, `just list`, `just runtimes`, `just create <id> <runtime>...`, `just bench <id> [--runtime <runtime>] [--format stream|ndjson|html]`
- Host: `cmd/jmark` (entry), `internal/{benchmark,catalog,measure,result,source,artifact,report}`
- Registry: `runtimes/*.toml`, `families/<f>/{runner.*,forbid.re,template/}`
- Schema: `schema/result.schema.json`; host enforces via `internal/result.CheckSchema` on every emit. `go test ./internal/result` checks generated and expected golden results against the schema and each other; fixtures live in `internal/result/testdata/golden/`

Isolation: pick a unique project id per mutating drive, e.g. `verify-$RUN_ID`. Do not mutate `sum-large-doubles` beyond running it.

## Evidence

For each feature drive, store under `artifacts/<run-id>/`:

- `*.cmd.txt` — exact command
- `*.stdout.txt` / `*.stderr.txt` — captured streams
- `*.exit.txt` — exit code
- For bench: `*.result.json` — NDJSON stdout after success

Proof standards:

- Exercise real user paths (`just …`), not by importing host functions.
- Capture the action and the resulting state (files after `create`, NDJSON lines after `bench`).
- Side effects: after `create`, `AGENTS.md`, `bench.toml`, and `<family>/` templates exist and `just list` includes the id; after cleanup those are gone while evidence remains.

## Cleanup

```bash
./.cursor/skills/verify-jmark/helpers/cleanup-project.sh <id>
```

It deletes `benches/<id>/` only. The measurement host removes its own execution workspace on completion or failure. Do not `killall`/`pkill` by name. No long-lived processes are started by this skill.

## Helpers

All helpers are executable; run from repo root.

```bash
RUN_ID=$(date +%Y%m%d-%H%M%S)
EV=.cursor/skills/verify-jmark/artifacts/$RUN_ID
mkdir -p "$EV"

./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/doctor" just doctor
./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/list" just list
./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/runtimes" just runtimes
./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/bench-node" just bench sum-large-doubles --runtime node
./.cursor/skills/verify-jmark/helpers/cleanup-project.sh verify-example   # after a create drive
```

- `helpers/capture.sh <prefix> <cmd…>` — writes `prefix.cmd.txt`, `.stdout.txt`, `.stderr.txt`, `.exit.txt`; copies stdout to `prefix.result.json` when the command is a successful `just bench`.
- `helpers/cleanup-project.sh <id>` — deletes the verification project under `benches/<id>/`.

## Feature map

See [`features/README.md`](features/README.md). Drive the mapped features from that index; one feature is enough for a smoke proof, all features for a full pass.
