# List projects

List shows every benchmark project id, derived from `benches/*/AGENTS.md`. There is no index file.

## Sub-features

- `list-ids` prints one id per line, sorted.

## How to get to it (user POV)

- Run `just list` from the repo root.

## Driving it with just

Preconditions:

- `just doctor` is OK.
- `benches/sum-large-doubles/AGENTS.md` exists.

- **List.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/list" just list`. Exit `0`. Stdout contains a line exactly `sum-large-doubles`.
- **Proof.** `$EV/list.stdout.txt` includes `sum-large-doubles` and `$EV/list.exit.txt` is `0`.

## Gotchas

- A directory without `AGENTS.md` is not a project and is not listed.
