# List runtimes

`just runtimes` lists the IDs in `runtimes/*.toml` in sorted order. Each tab-separated row contains the runtime ID, source family, and `found` or `missing: <commands>`. Availability means every command in `needs` is on `PATH`.

## Verify

Run `just doctor` first; it must exit 0 and print `doctor: OK`.

```bash
./.cursor/skills/verify-jmark/helpers/capture.sh "$EV/runtimes" just runtimes
```

Check that the command exits 0 and that `$EV/runtimes.stdout.txt` has one row per registered runtime, sorted by ID. The current registry includes `bun` and `node` in family `js`, and `py` in family `py`. `js` is a family, not a runtime ID.

## Runtime installation

`just runtimes lock [<runtime>...]` delegates to `mise lock`. `py` maps to `python`. Check the generated `mise.lock`; selected updates preserve other tools.

`just runtimes install [<runtime>...]` delegates to `mise install --locked`. Installation errors must return a nonzero exit code. Benchmark invocations use fresh temporary installations and keep downloaded artifacts cached.
