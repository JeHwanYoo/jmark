# Doctor catalog health

Doctor tells the user whether the host deps, the runtime/family registry, configured probe hooks, and every project's layout are ready to drive. Runtime toolchains are reported, never required.

## Sub-features

- `doctor-ok` reports found host deps, `registry ok`, and every project with its runtimes.
- `doctor-runtimes` prints `doctor: runtime <id> (<family>) found|missing: <cmd>` as info only.
- `doctor-guards` (optional negative check) fails on layout errors: unknown subdir in a project, mismatched approach sets, clock/JSON use matching `families/<f>/forbid.re`, a reserved probe name, or a runner that references `probes/`.

## How to get to it (user POV)

- Run `just doctor` from the repo root.

## Driving it with just

Preconditions:

- Repo root contains `justfile` and `cmd/jmark` / `just`.
- `just`, `go`, `bash`, `hyperfine` installed.

- **Health check.** Run `./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/doctor" just doctor`. Exit `0`. Stdout contains `doctor: registry ok`, `doctor: project ok sum-large-doubles (runtimes: bun node py)`, and `doctor: OK`.
- **Guard (optional).** In a `verify-$RUN_ID` project, append `const t = performance.now();` to a `js/` approach and run `just doctor`. Exit non-zero; stderr names the file and line. Clean up afterwards.
- **Proof.** Keep `$EV/doctor.stdout.txt` and `$EV/doctor.exit.txt` showing `0` and `doctor: OK`.

## Gotchas

- Doctor is read-only; a failing doctor is not fixed by re-running bench.
- A missing `node`/`bun`/`python3` does not fail doctor; it only shows `missing:` for that runtime.
- The forbid patterns also match comments; write "no JSON output" style comments, not `JSON.` text.

- Result golden fixtures belong to `go test ./internal/result`; doctor does not load test data.
