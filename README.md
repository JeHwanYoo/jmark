# jmark

Run the same micro-benchmark across runtimes and compare time, memory, throughput, and optional hardware counters.
It began as a personal tool to satisfy my own curiosity about how code performs.

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

Without an ID, a terminal picker asks for whatever options you left out. jmark measures every available runtime of the project unless `--runtime` picks one. The default `stream` format prints JSON lines to stdout and saves nothing; progress goes to stderr, so `just bench <id> | jq` works. `ndjson` and `html` save to `artifacts/private/<id>/<timestamp>/`, print the path, and show up in `just open`.

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

`--metrics` overrides `bench.toml`, which overrides the defaults. Other IDs are
`user_time`, `system_time`, `cache_references`, `cache_misses`, `branch_misses`,
and `cache_miss_rate`. macOS has no cache or branch counters, and Windows has no
memory or hardware counters; `just doctor` shows what this host collects. A metric
that cannot be measured stays in the result with a reason.

In a report, the (?) beside Metric opens a guide to each metric, and Export saves
the report as A4 PDF or standalone HTML. [METRICS.md](METRICS.md) has the full
definitions.

## Publish results

Private runs are Git-ignored. Public runs live in `artifacts/public/<project>/<run>/` and can be committed.

```bash
just public                       # choose private runs to make public
just public <project>/<run>        # move one run to public
just private <project>/<run>       # move it back to private
just site [--base-url <url>]      # build .jmark/site/index.html; a base URL adds canonical links and sitemap.xml
```

These commands only move directories; stage the result with `git add -A artifacts/public`. On each push to the default branch, the Pages workflow publishes public runs (set Settings → Pages → Source to GitHub Actions).

Each saved run keeps a `sources.md` snapshot of the code and configuration it measured, and the report shows it. Editing sources mid-run aborts the run.

## Add a benchmark

```bash
just runtimes                       # registered runtimes and current PATH availability
just create <id> <runtime>...       # scaffolds benches/<id>/
```

Write a strategy of up to 200 characters in `benches/<id>/AGENTS.md` (rules in the root `AGENTS.md`), set sizes in `bench.toml`, and add one pure-logic file per approach in each family folder.

## Add a runtime

Add `runtimes/<runtime>.toml` and the tool version in `mise.toml`, then run `just runtimes lock`. A new family also needs a runner and template in `families/<family>/`.

## Runtime versions

mise resolves and installs the versions in `mise.toml`, recording downloads and checksums in `mise.lock`.

```bash
just runtimes lock               # refresh mise.lock for configured runtimes
just runtimes lock node py       # refresh Node and Python entries
just runtimes install            # install locked runtimes in the project cache
```

Commit `mise.toml` and `mise.lock` together. Runtime ID `py` maps to mise's `python`.

Each bench run installs the locked runtimes into a temporary directory, runs through `mise exec --locked`, and removes the directory afterwards; downloads stay cached in `.jmark/`. Shell activation is not needed and your PATH is untouched. Python and Node are never compiled from source, so a platform without a locked download fails to install.

## Development

[ARCHITECTURE.md](ARCHITECTURE.md) describes the structure and data flow; [DESIGN.md](DESIGN.md) covers report UI standards.

Reports are standalone Svelte and ECharts pages. After editing `internal/report/ui/src/` or `METRICS.md`, run `just ui-build` and commit the generated `internal/report/ui/dist/`; host tests check it against the sources.

`go test ./...` runs host and report tests, `go test -tags live ./internal/measure` uses the installed tools, and `go test -race ./...` checks concurrency.
