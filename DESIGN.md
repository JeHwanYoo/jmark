# jmark Design Standard

This document defines the product and implementation standards for the benchmark
report UI. Apply it to changes under `internal/report/ui/` and report generation
under `internal/report/`.

## Purpose

Help users compare measured implementations under explicit, comparable conditions.
The primary view is one comparison chart with a shared numeric axis. Users choose
the categorical axis and series; exact values and measurement context remain
available in the results table and Environment section.

## Information hierarchy

1. Project and result count.
2. Workload, metric, runtime, run, sorting, and scale controls.
3. One comparison chart, with Y-axis and series selectors and a visible X-axis unit.
4. Exact result table.
5. Collapsed Environment details.

Keep filters and axis controls visible. Do not split approaches into independent
charts with independently scaled axes. Do not introduce decorative dashboards,
hero copy, or summary cards that compete with the measured results.

## Comparison rules

- X is the numeric benchmark metric, with its unit always visible.
- Y and Series can use Approach, Runtime, or Run. Do not use the same dimension
  for both. Offer Run only when multiple runs exist.
- Default to Approach on Y and Runtime as Series. Switching Y must select a valid
  Series if the previous selection conflicts.
- All displayed measurements share one numeric scale. Linear bars start at zero.
  Log scale uses points and is available only for strictly positive values.
- Keep unselected varying dimensions in category labels so measurements remain
  distinguishable. Omit constant run identifiers from chart labels.
- Separate project, input parameters, warmup, iteration count, and sample count
  combinations into workloads. Provide an independent Metric selector, isolating
  incompatible metric IDs, units, methods and measurement scopes. Never combine incompatible units or inputs.
- Preserve duplicate measurements and missing combinations. Do not silently
  average, sum, overwrite, or substitute zero for missing results.
- Assign stable series colors across filtering. Use labels and exact values in
  addition to color. Tooltips retain full names when an axis label is truncated.
- Keep the table consistent with the active filters. Show runtime and host metadata
  per run in Environment; do not imply that different machines are identical.

## Visual language

Use the bundled Inter Variable font for UI text and chart labels, with tabular
numerals. Use bundled JetBrains Mono Variable for units, workload parameters, and
source code. Both use the SIL Open Font License 1.1; include their license notices
in standalone reports. Fall back to system fonts for unsupported glyphs. No remote
font assets.

| Role | Value |
| --- | --- |
| Page background | `#f5f7fa` |
| Surface | `#ffffff` |
| Primary text | `#172033` |
| Secondary text | `#65748b` |
| Border | `#e0e6ee` |
| Accent / focus | `#2563eb` |
| Series palette | `#2563eb`, `#0d9488`, `#d97706`, `#9333ea`, `#db2777`, `#475569` |

Use a 4px spacing rhythm, typically 12–16px between controls and 20–24px inside
panels. Limit content width to 1208px including gutters. Use 6–10px corner radii.
Reserve shadows for floating menus and focused controls. Chart grid lines should
be quieter than the data marks.

## Searchable dropdowns

Use `SearchSelect.svelte` for report selectors; do not introduce native selects
or separate ad hoc dropdown implementations.

- A persistent label names each field; the closed control shows its selection.
- Opening focuses the search input and displays all options. Search is a trimmed,
  case-insensitive substring match on the visible option label.
- Typing filters options without changing the committed selection. Show a check
  on the selected option and a distinct highlight on the keyboard-active option.
- Arrow keys navigate; Home/End jump; Enter commits; Escape dismisses without
  changing the selection; Tab closes and continues normal focus navigation.
- Ignore navigation shortcuts during IME composition. Keep the active option in
  view. Clicking an option commits; clicking outside dismisses.
- Use the combobox/listbox/option accessibility pattern, associated labels,
  `aria-expanded`, `aria-controls`, `aria-activedescendant`, and `aria-selected`.
- Show `No matches` for an empty search. Preserve the existing selection.
- Keep menus within the viewport, above charts, with bounded scrolling. Long
  option labels wrap instead of being permanently truncated.

## Responsive behavior and accessibility

At widths of 600px or less, controls wrap into two columns; workload occupies a
full row. Use 44px touch targets and 16px search-input text on mobile. The page
must not scroll horizontally; wide data tables may scroll within their panel.

Charts resize with their container. Reduce or hide overlapping ticks on narrow
screens. Retain readable category labels and full values in the table. Provide
visible keyboard focus, semantic headings, labeled controls, and a text table
alongside the chart. Never communicate selection or meaning through color alone.

## UI copy

Use short functional labels: Workload, Runtime, Run, Sort, Y axis, Series,
Linear, Log, Results, Environment, Export. Keep the UI language consistent (English).
Show only useful empty/error states and measurement context. Do not insert agent
prompts, implementation notes, marketing language, or paragraphs describing what
the interface already makes apparent. Put build instructions in documentation.

## Implementation and delivery

- Svelte owns state and markup. ECharts owns chart rendering. Keep data grouping
  and chart options in `model.js`, lifecycle in `Chart.svelte`, and reusable
  selection behavior in `SearchSelect.svelte`.
- Treat result fields as text; never interpolate them into HTML. Escape embedded
  JSON and HTML titles. Use ECharts rich-text tooltips rather than HTML tooltips.
- Observe container resizing and dispose charts and observers on unmount.
- Deliver self-contained HTML with embedded JS and CSS. No CDN or runtime server
  requirement. Existing report files are snapshots and do not update automatically.
- `just ui-build` rebuilds embedded assets. `just bench` uses those assets and does
  not run the frontend build. Keep source, lockfile, and `ui/dist/` in sync.

## Verification

Run `just ui-build` and `go test ./...` after implementation changes. Check real
single-run and multi-run reports in a browser. Verify shared scaling, both axis
choices, series grouping, workload isolation, filtering, duplicates, and log mode.
Exercise dropdown search, no matches, keyboard selection, Escape, Tab, outside
click, and focus retention. Inspect desktop and 390px layouts with long labels.
Use synthetic data only as clearly identified fixtures; do not present it as a
measured benchmark. The verification recipe is
`.cursor/skills/verify-jmark/features/report.md`.

## Public index and source code

The public index lists project, run, runtimes, and capture time. A search input
filters the list. Each run links to its report; the report header provides a `←`
link labelled “Back to index”. Navigation uses relative paths and does not depend
on browser history.

Source code follows the result and environment sections. Render the saved Markdown
snapshot with syntax highlighting, readable monospace text, and horizontal scrolling
inside code blocks. Show “Source snapshot unavailable”
when a saved run has no snapshot. Current working files must not replace missing
historical code. Raw Markdown HTML and remote images are disabled.

## Export

One Export button beside the project heading is the only output entry point; do
not add print or download buttons to individual sections. It opens a modal with
Format (PDF / Print or Markdown), Chart, Environment and Source code selectors and a
live A4 preview. Results are always included. Exports follow the active filters,
metric, scale, axis and series. PDF output is a compact A4 portrait report: charts
split by category so each part fits a page, table rows and environment entries never
break across pages, and source code wraps instead of scrolling. Markdown output
omits the chart.

## Metric presentation

Units, measurement scope, method and preferred direction come from saved metric
definitions. Time values stored in s/op display in µs/op; bytes display in MiB;
ratios display as percentages. Unknown units display without conversion. Value
sorting places larger values first for higher-is-better definitions and smaller
values first otherwise. Missing values follow measured values and retain reasons
in the table. They never become zero bars. Measurement details show phase-level
statistics and samples; no per-operation latency percentiles are inferred.

## Metric help

Place a labelled (?) link next to the Metric selector. Open the selected metric's
explanation in a separate tab, preserving the original report. The guide shares
its definitions and measurement prose with `METRICS.md`, offers all built-in
metrics, and works without a network connection. Hash navigation keeps help inside
the standalone report file.

Use labelled examples distinct from measured data. Animate the quantity being
explained and provide play, pause, replay and timeline controls. Play once, pause
when the document is hidden, and start with a still frame for reduced-motion
preferences. Keep descriptions and controls usable at 320px and 390px widths.
