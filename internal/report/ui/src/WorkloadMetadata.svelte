<script>
  import { parameters, parameterValue, unit } from "./model.js";
  let { row } = $props();
</script>

<dl class="workload-metadata" aria-label="Benchmark parameters and measurement">
  {#each parameters(row) as [name, value]}
    <div><dt>{name}</dt><dd>{parameterValue(value)}</dd></div>
  {/each}
  <div><dt>Metric</dt><dd>{row?.metric ?? "—"}</dd></div>
  <div><dt>Unit</dt><dd>{unit(row)}</dd></div>
</dl>

<style>
  .workload-metadata {
    display: flex;
    flex-wrap: wrap;
    gap: 12px 28px;
    margin: 0 0 24px;
    font-size: 12px;
  }
  .workload-metadata > div { display: flex; gap: 8px; align-items: baseline; }
  dt { color: #65748b; }
  dd { margin: 0; color: #172033; overflow-wrap: anywhere; font-family: var(--font-code); }
</style>
