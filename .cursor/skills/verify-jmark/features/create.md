# Create project

Create scaffolds `benches/<id>/` with a TODO strategy, a default `bench.toml`, and template sources for each named runtime's family. It is idempotent and never overwrites.

## Sub-features

- `create-scaffold` writes `AGENTS.md`, `bench.toml`, and `<family>/_input.<ext>` plus `approach-a`/`approach-b`.
- `create-idempotent` re-running only fills missing files.
- `create-cleanup` removes the project without deleting verification evidence.

## How to get to it (user POV)

- Run `just create <id> <runtime>...` with a fresh id, e.g. `just create verify-$RUN_ID node py`.
- Confirm with `just list` and optionally `just bench <id> --runtime node`.
- Remove with `./.cursor/skills/verify-jmark/helpers/cleanup-project.sh <id>`.

## Driving it with just

Preconditions:

- `just doctor` is OK.
- Id `verify-$RUN_ID` does not already exist.

- **Create.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/create" just create "verify-$RUN_ID" node py`. Exit `0`. `benches/verify-$RUN_ID/` contains `AGENTS.md`, `bench.toml`, `js/{_input,approach-a,approach-b}.mjs`, `py/{_input,approach-a,approach-b}.py`.
- **List membership.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/create-list" just list`. Stdout includes `verify-$RUN_ID`.
- **Optional bench.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/create-bench" just bench "verify-$RUN_ID" --runtime node`. Exit `0` with one NDJSON line (`runtime=node`).
- **Cleanup.** Run `./.cursor/skills/verify-jmark/helpers/cleanup-project.sh "verify-$RUN_ID"`. `benches/verify-$RUN_ID` is gone; `just list` no longer includes it; `$EV/**` still exists.

## Gotchas

- `node` and `bun` share the `js` family, so `create <id> node bun` makes only `js/`.
- An unknown runtime fails before anything is written.
- Do not commit leftover `verify-*` projects.
