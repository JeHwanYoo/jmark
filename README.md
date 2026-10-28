# jmark

Run the same micro-benchmark across runtimes and compare time, memory, throughput, and optional hardware counters.

## Requirements

macOS or Linux with:

- [just](https://github.com/casey/just)
- [Go](https://go.dev) 1.26.4+
- [hyperfine](https://github.com/sharkdp/hyperfine) 2.0.0+
- [mise](https://mise.jdx.dev/installing-mise.html)

Benchmark runtimes are installed automatically through mise.

## Quick start

```bash
just doctor              # check tools and every project's layout
just list                # project ids
just bench               # choose project, runtime, and output format
just open                # pick past runs → compare charts in the browser
```

## Run benchmarks

```bash
just bench <id> [--runtime <runtime>] [--format stream|ndjson|html]
```

Pass an ID in scripts; omit it in a terminal to pick the project, runtime, and format. Supplied options skip their pickers. By default, jmark measures all available runtimes supported by the project. `--runtime` selects one for measurement; mise still installs all configured tools. Results stream to stdout as JSON lines. Progress and skipped runtimes go to stderr, so `just bench <id> | jq` works.

`ndjson` and `html` save to `artifacts/private/<id>/<timestamp>/` and print the path. The default `stream` format saves nothing and does not appear in `just open`.

```bash
just open [<id>]         # Tab to multi-select; Enter = compare, Ctrl-X = delete
just delete [<id>]       # Tab to multi-select; confirm with y
```

## Metrics

Default metrics are `wall_time`, `cpu_time`, `peak_rss`, and `throughput`.
Set `metrics = "wall_time peak_rss"` in `bench.toml`, or override it for one run:

```bash
just bench sum-large-doubles --metrics wall_time,peak_rss
just bench sum-large-doubles --metrics instructions,cpu_cycles,ipc
```

CLI selection replaces project selection; project selection replaces defaults.
Additional IDs: `user_time`, `system_time`, `cache_references`, `cache_misses`,
`branch_misses`, and `cache_miss_rate`. Hardware support depends on OS, CPU and
permissions. Missing metrics remain visible with an unavailable reason.

Reports have separate Workload and Metric selectors. The (?) next to Metric opens
a guide in a separate tab with definitions and animated examples for all built-in
metrics. The guide works offline and reads its text from `METRICS.md`. Peak RSS includes runtime
startup and input generation; it is the median of process peak measurements,
never bytes allocated per operation. One operation processes the entire input.
See [METRICS.md](METRICS.md) for definitions, extension contracts and research.
Results require schema version 2; previous saved formats are unsupported.

## Publish results

Private runs are Git-ignored; public runs in `artifacts/public/<project>/<run>/` can be committed. The picker shows both.

```bash
just public                       # choose private runs to make public
just public <project>/<run>        # move one run to public
just private <project>/<run>       # move it back to private
just site                         # build .jmark/site/index.html
```

These commands move run directories intact and leave Git operations to you. Commit public additions and deletions with `git add -A artifacts/public`. Moving a run to private preserves its Git history.

The Pages workflow publishes public results on pushes to the default branch. In GitHub Settings → Pages → Build and deployment, select GitHub Actions. `just site` builds locally without running benchmarks or needing their runtimes.

Saved runs include `sources.md` with inputs, approaches, runners, selected probes, and execution configuration. Reports display this snapshot and offer it as a Markdown download. Source changes during measurement abort the run. Older runs without snapshots leave sources unavailable.

## Add a benchmark

```bash
just runtimes                       # registered runtimes and current PATH availability
just create <id> <runtime>...       # scaffolds benches/<id>/
```

Write a strategy of ≤200 characters in `benches/<id>/AGENTS.md` (see the root `AGENTS.md`). Set sizes in `bench.toml`. In each family folder, add one pure-logic file per approach, with no clocks or JSON.

## Add a runtime

Add `runtimes/<runtime>.toml`. Version capture runs the first `needs` command with `--version`; set `version` to override it. New families also need a runner and template in `families/<family>/`. Add the tool version to `mise.toml` and refresh `mise.lock` before benchmarking.

## Runtime versions

mise resolves and installs the versions in `mise.toml`, recording downloads and checksums in `mise.lock`.

```bash
just runtimes lock               # refresh mise.lock for configured runtimes
just runtimes lock node py       # refresh Node and Python entries
just runtimes install            # install locked runtimes in the project cache
```

Commit `mise.toml` and `mise.lock` together. To change versions, edit `mise.toml` and run `just runtimes lock`. Runtime ID `py` maps to mise's `python`.

`just bench` and `just runtimes lock|install` use mise from `PATH`, without shell activation. Each bench invocation installs locked runtimes in a fresh temporary directory before measurement, runs through `mise exec --locked`, then removes the directory on exit. Downloads stay cached in `.jmark/`.

`just runtimes` checks `PATH` without installing tools. `just runtimes install` keeps an installation in `.jmark/mise-data/` without changing your shell PATH. Python and Node source compilation are disabled; installation fails if the platform has no locked download. Manage third-party application dependencies separately.

## Development

[ARCHITECTURE.md](ARCHITECTURE.md) describes the structure and data flow; [DESIGN.md](DESIGN.md) covers report UI standards.

Reports use Svelte and ECharts. After editing `internal/report/ui/src/` or `METRICS.md`, run `just ui-build` with Node.js and npm. Commit the lockfile and generated `internal/report/ui/dist/` assets; host tests verify their manifest against sources and assets. The report favicon is embedded from `internal/report/ui/src/assets/favicon.ico`; its matching vector source is `icon.svg`. `bench` and `open` use the embedded UI without rebuilding it. Reports are standalone HTML and need no server or network.

`go test ./...` runs host and report fixture tests. `go test -tags live ./internal/measure` uses installed timer and runtime tools. Use `go test -race ./...` to check concurrent execution. `just doctor` checks tools, registry, project layouts, and probes.
