# Run bench

Bench times each approach of a project from outside the runtime with hyperfine (`T(W+N) − T(W)`), then `internal/result` prints one NDJSON line per runtime. Output is all-or-nothing.

## Sub-features

- `bench-all` runs every available runtime of the project (`bun`, `node`, `py` for `sum-large-doubles`).
- `bench-one` runs a single runtime (`node`, `py`).
- `bench-bad-runtime` rejects an unknown runtime (`rust`).
- `bench-all-or-nothing` prints nothing when any requested runtime fails.
- `bench-sink-guard` fails an approach whose sink is not proportional to iters (input mutation).

## How to get to it (user POV)

- Run `just bench sum-large-doubles`.
- Run `just bench sum-large-doubles --runtime node` and `just bench sum-large-doubles --runtime py`.
- Run `just bench sum-large-doubles --runtime rust`, expecting failure.

## Driving it with just

Preconditions:

- `just doctor` is OK.
- The runtime's toolchain shows `found` in `just runtimes`.

- **All runtimes.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/bench-all" just bench sum-large-doubles`. Exit `0`. Stdout has one line per available runtime. Each line has `"metric":"us_per_op"`, results named `for-loop` and `map-filter-reduce` with equal `sink` (1950000), and `meta.clock` `hyperfine`.
- **One runtime.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/bench-node" just bench sum-large-doubles --runtime node`. Exactly one line with `"runtime":"node"`. Same for `py`.
- **Bad runtime.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/bench-bad-runtime" just bench sum-large-doubles --runtime rust`. Exit non-zero; stderr says `unknown runtime 'rust'`; stdout empty.
- **All-or-nothing (optional).** In a `verify-$RUN_ID` project, make `py/approach-b.py` raise, then run `just bench verify-$RUN_ID`. Exit non-zero and stdout is empty even though js runtimes finished.
- **Proof.** Every stdout line parses with `jq` and carries the schema's required fields.

## Gotchas

- `sum-large-doubles` py takes about 35 s (10 runs × 2 iteration counts × 2 approaches); the full run is about a minute.
- µs/op magnitudes differ across runtimes; compare structure and sink, not absolute `value`.
- Stdout is NDJSON only; progress (`→ <id> <runtime> <approach>`) and skips go to stderr.
- A missing toolchain is skipped with `skip: runtime … unavailable` when no runtime is named, but fails when that runtime is named.
