<script>
  import { onMount } from "svelte";
  import Brand from "./Brand.svelte";
  import RepositoryLink from "./RepositoryLink.svelte";
  import SearchSelect from "./SearchSelect.svelte";
  import MetricAnimation from "./MetricAnimation.svelte";
  import document from "../../../../METRICS.md";
  import { metricSections, guideMetric, guideHref } from "./guide-model.js";
  import { renderMarkdown } from "./markdown.js";
  const sections = metricSections(document);
  let id = $state(guideMetric(location.hash));
  const selected = $derived(sections.find((section) => section.id === id));
  const intro = document.split(/^## /m)[0].replace(/^# Metrics\s*/, "");
  function select(next) {
    id = next;
    location.hash = guideHref(next).slice(1);
  }
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
      title="Back to results">←</a
    ><Brand />
  </div>
  <RepositoryLink />
</header>
<main class="metrics-guide">
  <p class="eyebrow">Metric guide</p>
  <div class="guide-heading">
    <h1>{selected?.label ?? id}</h1>
    <SearchSelect
      id="guide-metric"
      label="Metric"
      options={sections.map((section) => ({
        value: section.id,
        label: section.label,
      }))}
      bind:value={() => id, select}
    />
  </div>
  {#if selected}
    <p class="guide-summary">{selected.summary}</p>
    <div class="guide-measurement">{@html renderMarkdown(selected.body)}</div>
    {#key id}<MetricAnimation {id} />{/key}
    <details class="guide-method">
      <summary>공통 측정 방식</summary>
      <div>{@html renderMarkdown(intro)}</div>
    </details>
  {:else}
    <p class="guide-summary">
      실행에서 제공한 사용자 정의 지표입니다. 단위와 측정 범위는 보고서에 저장된
      정의를 확인하세요.
    </p>
  {/if}
  <nav class="metric-directory" aria-label="All metrics">
    <h2>전체 지표</h2>
    <div>
      {#each sections as section}<a
          href={guideHref(section.id)}
          aria-current={section.id === id ? "page" : undefined}
          >{section.label}</a
        >{/each}
    </div>
  </nav>
</main>

<style>
  .metrics-guide {
    max-width: 920px;
  }
  .guide-heading {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 24px;
    margin-bottom: 24px;
  }
  .guide-summary {
    max-width: 680px;
    font-size: 19px;
    line-height: 1.8;
    margin: 0 0 22px;
    color: #27334a;
  }
  .guide-measurement,
  .guide-method {
    font-size: 14px;
    line-height: 1.8;
  }
  .guide-measurement {
    color: #526078;
  }
  .guide-method {
    margin: 26px 0;
  }
  .guide-method summary {
    cursor: pointer;
    color: #526078;
    padding: 10px 0;
  }
  .guide-method :global(code),
  .guide-measurement :global(code) {
    font-family: var(--font-code);
    font-size: 12px;
    overflow-wrap: anywhere;
  }
  .metric-directory {
    margin-top: 36px;
  }
  .metric-directory h2 {
    font-size: 15px;
    font-weight: 600;
  }
  .metric-directory div {
    display: flex;
    gap: 10px 20px;
    flex-wrap: wrap;
  }
  .metric-directory a {
    color: #526078;
    font-size: 13px;
    line-height: 2.5;
  }
  .metric-directory a[aria-current="page"] {
    color: #2563eb;
    font-weight: 600;
  }
  @media (max-width: 600px) {
    .guide-summary {
      font-size: 17px;
    }
    .guide-heading {
      align-items: flex-start;
    }
  }
</style>
