<script>
  import Brand from "./Brand.svelte";
  import RepositoryLink from "./RepositoryLink.svelte";
  import WorkloadMetadata from "./WorkloadMetadata.svelte";
  import Sources from "./Sources.svelte";
  import Chart from "./Chart.svelte";
  import SearchSelect from "./SearchSelect.svelte";
  import {
    colors,
    conditions,
    conditionKey,
    selectRows,
    format,
    unit,
    dimensions,
  } from "./model.js";
  let { rows, sources = "", indexHref = "" } = $props();
  const choices = $derived(conditions(rows));
  let selected = $state("");
  let selectedRuntime = $state("");
  let run = $state("");
  let scale = $state("linear");
  let sort = $state("value");
  const condition = $derived(selected || choices[0]?.key);
  const filtered = $derived(selectRows(rows, condition, selectedRuntime, run));
  const runtimes = $derived([...new Set(rows.map((r) => r.runtime))]);
  const runs = $derived([...new Set(rows.map((r) => r.run).filter(Boolean))]);
  let axis = $state("name");
  let series = $state("runtime");
  const axisOptions = $derived(
    dimensions.filter((d) => d.value !== "run" || runs.length > 1),
  );
  const seriesOptions = $derived(axisOptions.filter((d) => d.value !== axis));
  const activeSeries = $derived(
    seriesOptions.some((d) => d.value === series)
      ? series
      : seriesOptions[0].value,
  );
  const domain = $derived([
    ...new Set(rows.map((r) => r[activeSeries] ?? "—")),
  ]);
  const legend = $derived([
    ...new Set(filtered.map((r) => r[activeSeries] ?? "—")),
  ]);
  const ordered = $derived(
    [...filtered].sort((a, b) =>
      sort === "value" ? a.value - b.value : a.runtime.localeCompare(b.runtime),
    ),
  );
  const canLog = $derived(
    filtered.length > 0 && filtered.every((r) => r.value > 0),
  );
  const activeScale = $derived(canLog ? scale : "linear");
  const environments = $derived([
    ...new Map(
      filtered.map((r) => [
        JSON.stringify([
          r.run,
          r.runtime,
          r.runtime_version,
          r.host,
          r.captured_at,
        ]),
        r,
      ]),
    ).values(),
  ]);
</script>

<svelte:head
  ><title>jmark · {filtered[0]?.project ?? rows[0]?.project ?? "Results"}</title
  ></svelte:head
>
<header>
  <div class="brand-group">
    {#if indexHref}<a
        class="back-link"
        href={indexHref}
        aria-label="Back to index"
        title="Back to index">←</a
      >{/if}<Brand />
  </div>
  <RepositoryLink />
</header>
<main>
  <div class="heading">
    <div>
      <p class="eyebrow">
        {runs.length > 1 ? `${runs.length} runs` : "Benchmark"}
      </p>
      <h1>{filtered[0]?.project ?? rows[0]?.project ?? "Results"}</h1>
    </div>
  </div>
  {#if rows.length}
    <div class="filters">
      {#if choices.length > 1}
        <SearchSelect
          id="workload"
          label="Workload"
          wide
          options={choices.map((choice, i) => ({
            value: i === 0 ? "" : choice.key,
            label: choice.label,
          }))}
          bind:value={selected}
        />
      {/if}
      <SearchSelect
        id="runtime"
        label="Runtime"
        options={[
          { value: "", label: "All runtimes" },
          ...runtimes.map((runtime) => ({ value: runtime, label: runtime })),
        ]}
        bind:value={selectedRuntime}
      />
      {#if runs.length > 1}
        <SearchSelect
          id="run"
          label="Run"
          options={[
            { value: "", label: "All runs" },
            ...runs.map((id) => ({ value: id, label: id })),
          ]}
          bind:value={run}
        />
      {/if}
      <SearchSelect
        id="sort"
        label="Sort"
        options={[
          { value: "value", label: "Value" },
          { value: "runtime", label: "Runtime" },
        ]}
        bind:value={sort}
      />
      <div class="scale" role="group" aria-label="Scale">
        <button
          class:active={activeScale === "linear"}
          aria-pressed={activeScale === "linear"}
          onclick={() => (scale = "linear")}>Linear</button
        ><button
          class:active={activeScale === "log"}
          aria-pressed={activeScale === "log"}
          disabled={!canLog}
          onclick={() => (scale = "log")}>Log</button
        >
      </div>
    </div>
    <WorkloadMetadata row={rows.find((row) => conditionKey(row) === condition)} />
    {#if filtered.length}
      <details class="panel report-section comparison" open>
        <summary>Charts</summary>
        <div class="comparison-toolbar">
          <SearchSelect
            id="axis"
            label="Y axis"
            options={axisOptions}
            bind:value={axis}
          />
          <SearchSelect
            id="series"
            label="Series"
            options={seriesOptions}
            bind:value={() => activeSeries, (next) => (series = next)}
          />
          <span class="axis-unit">X axis · {unit(filtered[0])}</span>
        </div>
        <div class="legend">
          {#each legend as name}<span
              ><i
                style:background={colors[domain.indexOf(name) % colors.length]}
              ></i>{name}</span
            >{/each}
        </div>
        <Chart
          rows={ordered}
          scale={activeScale}
          {axis}
          series={activeSeries}
          {domain}
        />
      </details>
      <details class="panel report-section results" open>
        <summary>Results</summary>
        <div class="table-wrap">
          <table>
            <thead
              ><tr
                >{#if runs.length}<th>Run</th>{/if}<th>Runtime</th><th
                  >Approach</th
                ><th class="number">{unit(filtered[0])}</th><th
                  class="number">Total ms</th
                ><th class="number">Iterations</th></tr
              ></thead
            ><tbody
              >{#each ordered as row}<tr
                  >{#if runs.length}<td>{row.run ?? "—"}</td>{/if}<td
                    >{row.runtime}</td
                  ><td>{row.name}</td><td class="number value"
                    >{format(row.value)}</td
                  ><td class="number">{format(row.total_ms)}</td><td
                    class="number">{format(row.iters)}</td
                  ></tr
                >{/each}</tbody
            >
          </table>
        </div>
      </details>
      <details class="panel report-section environment">
        <summary>Environment</summary>{#each environments as row}<div
            class="env-row"
          >
            <strong>{row.runtime}{row.run ? ` · ${row.run}` : ""}</strong>
            <p>{row.runtime_version ?? "—"}</p>
            <p>
              {[row.host?.os, row.host?.arch, row.host?.cpu]
                .filter(Boolean)
                .join(" · ")}
            </p>
            <p>
              {row.captured_at ?? "—"} · warmup={row.warmup ??
                "—"} · iters={row.iters_meta ?? "—"} · runs={row.runs ?? "—"}
            </p>
          </div>{/each}
      </details>
    {:else}<p class="empty">
        No matching results. <button
          onclick={() => {
            selectedRuntime = "";
            run = "";
          }}>Reset filters</button
        >
      </p>{/if}
  {:else}<p class="empty">No results.</p>{/if}
  <Sources markdown={sources} />
</main>
