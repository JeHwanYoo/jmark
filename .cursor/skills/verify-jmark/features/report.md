# Report UI

Verify the embedded Svelte/ECharts report through real saved benchmark results.

## Sub-features

Single and comparison reports, searchable filters, shared chart axes, environment metadata, and responsive layout.

## How to get to it (user POV)

Use `just bench <id> --format html` or `just open`.

## Driving it with just

After changing `internal/report/ui/src/`, run `just ui-build` and `go test ./...`.
Keep the generated `ui/dist/` files with the source changes; Go embeds these files.

Run `just bench sum-large-doubles --runtime node --format html` and open the printed
HTML path. Confirm the table values match `result.ndjson`, both chart scales work,
and Environment shows runtime and workload metadata. HTML must work offline.

Use `just open` to compare saved runs. Check Runtime and Run filters, a combination
with no matching results, and Reset filters. Runs with different projects, metrics,
input sizes, warmup, iterations, or sample counts must appear as separate workloads.
Check the report at desktop and 390px widths; tables may scroll horizontally.

## Gotchas

Regression checks cover HTML/script escaping, malformed NDJSON, distinct artifact
folders, workload isolation, duplicate rows, and zero-valued log-axis fallback.

Verify one chart with a shared numeric axis. Search the Y axis and Series selectors,
pivot Approach/Runtime/Run, and confirm remaining dimensions and duplicate rows
remain distinct. Exercise Arrow keys, Home/End, Enter, Escape, Tab, no matches,
and outside dismissal in SearchSelect. Follow the root DESIGN.md standards.
