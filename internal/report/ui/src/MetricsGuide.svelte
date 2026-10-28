<script>
  import { onMount } from "svelte";
  import Brand from "./Brand.svelte";
  import RepositoryLink from "./RepositoryLink.svelte";
  import MetricAnimation from "./MetricAnimation.svelte";
  import document from "../../../../METRICS.md";
  import {
    metricSections,
    guideSections,
    guideMetric,
    guideHref,
    hostPlatform,
    platformName,
    platforms,
    statusLabel,
    supported,
  } from "./guide-model.js";
  import { renderMarkdown } from "./markdown.js";
  let { rows = [], catalog = null } = $props();
  const groups = $derived(guideSections(metricSections(document), rows, catalog));
  const sections = $derived([...groups.inReport, ...groups.other]);
  let id = $state(guideMetric(location.hash));
  const selected = $derived(sections.find((section) => section.id === id));
  const os = $derived(hostPlatform(rows[0]?.host));
  const readings = $derived(catalog?.metrics?.[id]?.readings ?? []);
  const notes = {
    measured: "This report includes measured values for this metric.",
    unavailable: "This report requested the metric, but no value could be computed.",
    available: "This host reported every reading the metric needs. Request it with --metrics to include it.",
    unsupported: "hyperfine cannot report the required readings on this operating system.",
    withheld: "The operating system supports the readings, but this host did not report them. Check counter permissions or virtualization.",
  };
  const intro = document.split(/^## /m)[0].replace(/^# Metrics\s*/, "");
  onMount(() => {
    const update = () => {
      id = guideMetric(location.hash);
    };
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  });
</script>

<svelte:head
  ><title>jmark · {selected?.label ?? id} · Metrics</title></svelte:head
>
<header>
  <div class="brand-group">
    <a
      class="back-link"
      href={location.href.split("#")[0]}
      aria-label="Back to results"
      title="Back to results"><span aria-hidden="true">←</span> Results</a
    ><Brand />
  </div>
  <RepositoryLink />
</header>
<main class="metrics-guide">
  <nav class="metric-directory" aria-label="Metrics">
    {#if groups.inReport.length}
      <h2>In this report</h2>
      <div class="metric-links">
        {#each groups.inReport as section}
          <a href={guideHref(section.id)} aria-current={section.id === id ? "page" : undefined}
            >{section.label}{#if section.status.kind === "unavailable"}<small>Unavailable</small>{/if}</a
          >
        {/each}
      </div>
      <details class="other-metrics" open={groups.other.some((section) => section.id === id)}>
        <summary>Not in this report <span>{groups.other.length}</span></summary>
        <div class="metric-links">
          {#each groups.other as section}
            <a href={guideHref(section.id)} aria-current={section.id === id ? "page" : undefined}
              >{section.label}<small>{statusLabel(section.status)}</small></a
            >
          {/each}
        </div>
      </details>
    {:else}
      <h2>Metrics</h2>
      <div class="metric-links">
        {#each groups.other as section}
          <a href={guideHref(section.id)} aria-current={section.id === id ? "page" : undefined}>{section.label}</a>
        {/each}
      </div>
    {/if}
  </nav>
  <article class="guide-content">
    <p class="eyebrow">Metric guide</p>
    <h1>{selected?.label ?? id}</h1>
    {#if selected && selected.status.kind !== "unknown"}
      <p class="metric-status" data-kind={selected.status.kind}>
        <strong>{statusLabel(selected.status)}</strong>
        <span>{notes[selected.status.kind]}</span>
      </p>
    {/if}
    {#if selected?.summary}
      <p class="guide-summary">{selected.summary}</p>
      <div class="guide-measurement">{@html renderMarkdown(selected.body)}</div>
      {#if readings.length}
        <dl class="metric-support">
          <div>
            <dt>Readings</dt>
            <dd>{#each readings as reading}<code>{reading}</code>{/each}</dd>
          </div>
          <div>
            <dt>Platforms</dt>
            <dd>
              {#each platforms as platform}<span
                  class:supported={supported(catalog, id, platform)}
                  class:host={platform === os}
                  >{supported(catalog, id, platform) ? "✓" : "—"} {platformName(platform)}{platform === os ? " · this host" : ""}</span
                >{/each}
            </dd>
          </div>
        </dl>
      {/if}
      {#key id}<MetricAnimation {id} />{/key}
      <details class="guide-method">
        <summary>How measurements work</summary>
        <div>{@html renderMarkdown(intro)}</div>
      </details>
    {:else}
      <p class="guide-summary">This is a custom metric. See its definition in the report for its unit and measurement scope.</p>
    {/if}
  </article>
</main>

<style>
  .back-link { display: inline-flex; gap: 6px; width: auto; height: auto; padding: 8px 12px; font-size: 13px; font-weight: 600; }
  .back-link span { font-size: 18px; line-height: 1; }
  .metrics-guide { max-width: 1100px; display: grid; grid-template-columns: 210px minmax(0, 1fr); gap: 56px; align-items: start; }
  .metric-directory { position: sticky; top: 24px; display: flex; flex-direction: column; gap: 2px; }
  .metric-directory h2 { margin: 0 0 12px; padding: 0 12px; color: #65748b; font-size: 12px; text-transform: uppercase; letter-spacing: .06em; }
  .metric-links { display: flex; flex-direction: column; gap: 2px; }
  .metric-directory a { display: flex; flex-direction: column; gap: 2px; padding: 9px 12px; border-radius: 7px; color: #526078; font-size: 14px; text-decoration: none; }
  .metric-directory a small { color: #7b8797; font-size: 12px; font-weight: 400; }
  .other-metrics { margin-top: 16px; }
  .other-metrics summary { cursor: pointer; padding: 9px 12px; color: #65748b; font-size: 12px; text-transform: uppercase; letter-spacing: .06em; font-weight: 600; }
  .other-metrics summary span { margin-left: 4px; font-variant-numeric: tabular-nums; }
  .metric-directory a:hover { background: #e9edf4; color: #172033; }
  .metric-directory a[aria-current="page"] { background: #e4edff; color: #1d4ed8; font-weight: 600; }
  .guide-content { min-width: 0; max-width: 720px; }
  .metric-status { display: flex; flex-direction: column; gap: 4px; margin: -8px 0 24px; padding: 12px 14px; border-left: 3px solid #2563eb; background: #f1f5ff; font-size: 14px; line-height: 1.6; color: #27334a; }
  .metric-status strong { font-size: 13px; }
  .metric-status[data-kind="unavailable"], .metric-status[data-kind="unsupported"], .metric-status[data-kind="withheld"] { border-color: #b16a08; background: #fff7eb; }
  .metric-status[data-kind="available"] { border-color: #0d9488; background: #effaf8; }
  .metric-support { display: grid; gap: 10px; margin: 22px 0 0; font-size: 13px; }
  .metric-support div { display: grid; grid-template-columns: 90px minmax(0, 1fr); gap: 12px; align-items: baseline; }
  .metric-support dt { color: #65748b; }
  .metric-support dd { display: flex; flex-wrap: wrap; gap: 6px 14px; margin: 0; color: #7b8797; }
  .metric-support code { font-family: var(--font-code); font-size: 12px; }
  .metric-support .supported { color: #27334a; }
  .metric-support .host { font-weight: 600; }
  .guide-content h1 { margin-bottom: 24px; }
  .guide-summary { font-size: 19px; line-height: 1.7; margin: 0 0 22px; color: #27334a; }
  .guide-measurement, .guide-method { font-size: 14px; line-height: 1.8; }
  .guide-measurement { color: #526078; }
  .guide-method { margin: 26px 0; }
  .guide-method summary { cursor: pointer; color: #526078; padding: 10px 0; }
  .guide-method :global(table) { display: block; overflow-x: auto; margin: 12px 0; }
  .guide-method :global(th), .guide-method :global(td) { padding: 8px 12px; }
  .guide-method :global(code), .guide-measurement :global(code) { font-family: var(--font-code); font-size: 12px; overflow-wrap: anywhere; }
  @media (max-width: 700px) {
    .metrics-guide { display: flex; flex-direction: column; gap: 24px; }
    .metric-directory { position: static; width: 100%; }
    .metric-directory h2 { margin-bottom: 6px; }
    .metric-links { flex-direction: row; overflow-x: auto; padding-bottom: 6px; }
    .other-metrics .metric-links { flex-wrap: wrap; overflow-x: visible; }
    .other-metrics { margin-top: 8px; }
    .metric-directory a { white-space: nowrap; }
    .metric-support div { grid-template-columns: 1fr; gap: 4px; }
    .guide-summary { font-size: 17px; }
  }
</style>
