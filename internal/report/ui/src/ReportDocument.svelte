<script>
  import Chart from "./Chart.svelte";
  import { renderMarkdown } from "./markdown.js";
  import { chartPages, environment, statusText } from "./export-model.js";
  import { colors, format, unit } from "./model.js";
  let { report, include } = $props();
  const pages = $derived(
    include.chart
      ? chartPages(report.chartRows, report.chart.axis, report.chart.series)
      : [],
  );
  const showRun = $derived(report.rows.some((r) => r.run));
  const sources = $derived(
    include.sources && report.sources ? renderMarkdown(report.sources) : "",
  );
</script>

<article class="report-document">
  <div class="doc-header">
    <p class="doc-brand">jmark · Benchmark report</p>
    <h1>{report.title}</h1>
    <p class="doc-meta">{report.workload}</p>
    <p class="doc-meta">{report.metric}</p>
  </div>
  {#if pages.length}
    <section class="doc-section">
      <h2>Chart</h2>
      <p class="doc-meta">{report.chart.summary}</p>
      <div class="legend">
        {#each report.chart.legend as name}<span
            ><i
              style:background={colors[
                report.chart.domain.indexOf(name) % colors.length
              ]}
            ></i>{name}</span
          >{/each}
      </div>
      {#each pages as page}
        <div class="doc-chart">
          <Chart
            rows={page}
            scale={report.chart.scale}
            axis={report.chart.axis}
            series={report.chart.series}
            domain={report.chart.domain}
            bounds={report.chart.bounds}
            print
          />
        </div>
      {/each}
    </section>
  {/if}
  <section class="doc-section">
    <h2>Results</h2>
    <table>
      <thead
        ><tr
          >{#if showRun}<th>Run</th>{/if}<th>Runtime</th><th>Approach</th><th
            class="number">{unit(report.rows[0])}</th
          ><th>Status</th><th class="number">Iterations</th></tr
        ></thead
      ><tbody
        >{#each report.rows as row}<tr
            >{#if showRun}<td>{row.run ?? "—"}</td>{/if}<td>{row.runtime}</td
            ><td>{row.name}</td><td class="number">{format(row.value)}</td><td
              >{statusText(row)}</td
            ><td class="number">{format(row.iters_meta)}</td></tr
          >{/each}</tbody
      >
    </table>
  </section>
  {#if include.environment}
    <section class="doc-section">
      <h2>Environment</h2>
      <dl class="doc-env">
        {#each report.environments as row}
          {@const env = environment(row)}
          <div>
            <dt>{env.title}</dt>
            <dd>{[env.version, env.host, env.settings].filter(Boolean).join(" · ")}</dd>
          </div>
        {/each}
      </dl>
    </section>
  {/if}
  {#if include.sources}
    <section class="doc-section doc-sources">
      <h2>Source code</h2>
      {#if sources}<div class="markdown">{@html sources}</div>{:else}<p
          class="doc-meta"
        >
          Source snapshot unavailable.
        </p>{/if}
    </section>
  {/if}
</article>

<style>
  .report-document {
    color: #172033;
    background: #fff;
    font-size: 9.5pt;
    line-height: 1.4;
  }
  .doc-header {
    padding-bottom: 3mm;
    border-bottom: 0.5pt solid #cfd6e0;
  }
  .doc-brand {
    margin: 0 0 2mm;
    font-size: 8pt;
    font-weight: 650;
    color: #526078;
  }
  h1 {
    font-size: 16pt;
    letter-spacing: -0.3pt;
    margin: 0 0 1.5mm;
  }
  h2 {
    font-size: 10.5pt;
    margin: 0 0 2mm;
    break-after: avoid;
  }
  .doc-meta {
    margin: 0 0 1mm;
    font-size: 8pt;
    color: #526078;
    overflow-wrap: anywhere;
  }
  .doc-section {
    margin-top: 5mm;
  }
  .legend {
    margin: 2mm 0 0;
    font-size: 8pt;
    gap: 4px 14px;
  }
  .doc-chart {
    break-inside: avoid;
  }
  table {
    font-size: 8pt;
    white-space: normal;
  }
  th,
  td {
    padding: 1.2mm 2mm;
    border-top: 0.5pt solid #e3e8ef;
    overflow-wrap: anywhere;
  }
  th {
    font-size: 7.5pt;
    background: #f5f7fa;
  }
  tr {
    break-inside: avoid;
  }
  thead {
    display: table-header-group;
  }
  .doc-env {
    margin: 0;
    font-size: 8pt;
  }
  .doc-env div {
    display: grid;
    grid-template-columns: 38mm 1fr;
    gap: 3mm;
    padding: 1.2mm 0;
    border-top: 0.5pt solid #e3e8ef;
    break-inside: avoid;
  }
  dt {
    font-weight: 650;
    overflow-wrap: anywhere;
  }
  dd {
    margin: 0;
    color: #526078;
    overflow-wrap: anywhere;
  }
  .markdown :global(h2) {
    font-size: 8.5pt;
    margin: 4mm 0 1.5mm;
    font-family: var(--font-code);
  }
  .markdown :global(pre) {
    margin: 0;
    padding: 2.5mm;
    font-size: 7.5pt;
    line-height: 1.45;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    background: #f6f8fa;
    border: 0.5pt solid #e2e7ee;
    border-radius: 4px;
  }
</style>
