<script>
  import { tick } from "svelte";
  import { flip } from "svelte/animate";
  import ReportDocument from "./ReportDocument.svelte";
  import {
    exportFormats,
    exportFilename,
    moveKey,
    orderedKeys,
    standaloneHtml,
  } from "./export-model.js";
  let { report, open = $bindable(false) } = $props();
  let dialog;
  let outputFormat = $state("pdf");
  let excluded = $state([]);
  let excludedMetrics = $state([]);
  let metricOrder = $state([]);
  let dragging = $state();
  let metricList = $state();
  let environment = $state(true);
  let sources = $state(true);
  let printing = $state(false);
  let previewWidth = $state(0);
  let paper = $state();
  const approaches = $derived([
    ...new Set(
      report.metrics.flatMap((m) => m.chartRows.map((row) => row.name)),
    ),
  ]);
  const defaultOrder = $derived(report.metrics.map((m) => m.key));
  const orderedMetrics = $derived(
    orderedKeys(defaultOrder, metricOrder).map((key) =>
      report.metrics.find((m) => m.key === key),
    ),
  );
  const reordered = $derived(
    orderedMetrics.some((m, i) => m.key !== defaultOrder[i]),
  );
  const include = $derived({
    metrics: orderedMetrics
      .map((m) => m.key)
      .filter((key) => !excludedMetrics.includes(key)),
    approaches: approaches.filter((name) => !excluded.includes(name)),
    environment,
    sources,
  });
  const toggled = (list, item, checked) =>
    checked ? list.filter((other) => other !== item) : [...list, item];
  const move = (key, index) =>
    (metricOrder = moveKey(
      orderedMetrics.map((m) => m.key),
      key,
      index,
    ));
  const reduceMotion =
    typeof matchMedia === "function" &&
    matchMedia("(prefers-reduced-motion: reduce)").matches;

  function startDrag(event, key) {
    if (event.button !== 0) return;
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    dragging = key;
  }
  // The dragged row lands after every other row whose midpoint is above the pointer.
  function drag(event) {
    if (!dragging) return;
    const index = [...metricList.children].filter((row) => {
      if (row.dataset.key === dragging) return false;
      const box = row.getBoundingClientRect();
      return box.top + box.height / 2 < event.clientY;
    }).length;
    if (orderedMetrics[index]?.key !== dragging) move(dragging, index);
  }
  function nudge(event, key, index) {
    const step = { ArrowUp: -1, ArrowDown: 1 }[event.key];
    if (!step) return;
    event.preventDefault();
    move(key, index + step);
  }

  // A4 width at 96 dpi, scaled down to fit the preview pane.
  const zoom = $derived(previewWidth ? Math.min(1, (previewWidth - 32) / 794) : 1);

  $effect(() => {
    if (open && !dialog.open) dialog.showModal();
    else if (!open && dialog.open) dialog.close();
  });

  async function print() {
    printing = true;
    open = false;
    await tick();
    await document.fonts.ready;
    await new Promise((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(resolve)),
    );
    window.print();
  }
  function download() {
    const copy = paper.cloneNode(true);
    copy.removeAttribute("style");
    copy.className = "export-paper";
    const styles = [...document.querySelectorAll("style")].map(
      (style) => style.textContent,
    );
    const url = URL.createObjectURL(
      new Blob([standaloneHtml(report.title, styles, copy.outerHTML)], {
        type: "text/html;charset=utf-8",
      }),
    );
    const link = document.createElement("a");
    link.href = url;
    link.download = exportFilename(report.title, "html");
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    open = false;
  }
</script>

<svelte:window onafterprint={() => (printing = false)} />
<dialog
  class="export-dialog"
  bind:this={dialog}
  aria-labelledby="export-title"
  onclose={() => (open = false)}
>
  {#if open}
    <div class="export-layout">
      <div class="export-options">
        <h2 id="export-title">Export</h2>
        <div class="field">
          <span class="field-label" id="export-format">Format</span>
          <div class="segmented" role="group" aria-labelledby="export-format">
            {#each exportFormats as option}<button
                type="button"
                class:active={outputFormat === option.value}
                aria-pressed={outputFormat === option.value}
                onclick={() => (outputFormat = option.value)}
                >{option.label}</button
              >{/each}
          </div>
        </div>
        <fieldset>
          <legend
            >Metrics{#if reordered}<button
                type="button"
                class="reset"
                onclick={() => (metricOrder = [])}>Reset order</button
              >{/if}</legend
          >
          <ol class="metric-list" bind:this={metricList}>
            {#each orderedMetrics as m, i (m.key)}
              <li
                class="metric-row"
                class:dragging={dragging === m.key}
                data-key={m.key}
                animate:flip={{ duration: reduceMotion ? 0 : 150 }}
              >
                <button
                  type="button"
                  class="handle"
                  aria-label="Move {m.title}"
                  aria-keyshortcuts="ArrowUp ArrowDown"
                  title="Drag or use arrow keys to reorder"
                  onpointerdown={(event) => startDrag(event, m.key)}
                  onpointermove={drag}
                  onpointerup={() => (dragging = undefined)}
                  onpointercancel={() => (dragging = undefined)}
                  onkeydown={(event) => nudge(event, m.key, i)}
                  ><svg viewBox="0 0 10 16" aria-hidden="true"
                    ><circle cx="3" cy="3" r="1.3" /><circle
                      cx="7"
                      cy="3"
                      r="1.3"
                    /><circle cx="3" cy="8" r="1.3" /><circle
                      cx="7"
                      cy="8"
                      r="1.3"
                    /><circle cx="3" cy="13" r="1.3" /><circle
                      cx="7"
                      cy="13"
                      r="1.3"
                    /></svg
                  ></button
                ><label class="check"
                  ><input
                    type="checkbox"
                    checked={!excludedMetrics.includes(m.key)}
                    onchange={(event) =>
                      (excludedMetrics = toggled(
                        excludedMetrics,
                        m.key,
                        event.currentTarget.checked,
                      ))}
                  /><span>{m.label} <small>{m.unit}</small></span></label
                >
              </li>
            {/each}
          </ol>
        </fieldset>
        <fieldset>
          <legend>Chart</legend>
          {#if approaches.length}
            <div class="checks scroll">
              {#each approaches as name}
                <label class="check"
                  ><input
                    type="checkbox"
                    checked={!excluded.includes(name)}
                    onchange={(event) =>
                      (excluded = toggled(
                        excluded,
                        name,
                        event.currentTarget.checked,
                      ))}
                  /><span>{name}</span></label
                >
              {/each}
            </div>
          {:else}<p class="none">No measured values.</p>{/if}
        </fieldset>
        <fieldset>
          <legend>Appendix</legend>
          <div class="checks">
            <label class="check"
              ><input type="checkbox" bind:checked={environment} /><span
                >Environment</span
              ></label
            >
            <label class="check"
              ><input type="checkbox" bind:checked={sources} /><span
                >Source code</span
              ></label
            >
          </div>
        </fieldset>
        <div class="export-actions">
          <button type="button" onclick={() => (open = false)}>Cancel</button>
          {#if outputFormat === "pdf"}
            <button
              type="button"
              class="primary"
              disabled={!include.metrics.length}
              onclick={print}>Print</button
            >
          {:else}
            <button
              type="button"
              class="primary"
              disabled={!include.metrics.length}
              onclick={download}>Download</button
            >
          {/if}
        </div>
      </div>
      <div
        class="export-preview"
        bind:clientWidth={previewWidth}
        aria-label="Preview"
      >
        <div class="paper" bind:this={paper} style:zoom>
          <ReportDocument {report} {include} />
        </div>
      </div>
    </div>
  {/if}
</dialog>
{#if printing}
  <div class="print-document">
    <ReportDocument {report} {include} />
  </div>
{/if}

<style>
  .export-dialog {
    width: min(1120px, calc(100vw - 32px));
    max-width: none;
    max-height: none;
    height: min(860px, calc(100vh - 32px));
    padding: 0;
    border: 1px solid #e0e6ee;
    border-radius: 10px;
    overflow: visible;
    color: #172033;
  }
  .export-dialog::backdrop {
    background: #17203366;
  }
  .export-layout {
    display: grid;
    grid-template-columns: 240px 1fr;
    height: 100%;
  }
  .export-options {
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 24px;
    overflow-y: auto;
    border-right: 1px solid #edf0f5;
  }
  .field,
  fieldset {
    display: grid;
    gap: 8px;
    margin: 0;
    padding: 0;
    border: 0;
    min-width: 0;
  }
  .field-label,
  legend {
    padding: 0;
    margin-bottom: 8px;
    font-size: 11px;
    font-weight: 650;
    color: #5f6c80;
  }
  .segmented {
    display: flex;
    gap: 2px;
    padding: 3px;
    background: #e9edf4;
    border-radius: 7px;
  }
  .segmented button {
    flex: 1;
    background: none;
    border-color: transparent;
    padding: 7px 12px;
  }
  .segmented .active {
    background: #fff;
    box-shadow: 0 1px 3px #17203314;
    font-weight: 600;
  }
  .checks {
    display: grid;
    gap: 2px;
  }
  .scroll {
    max-height: 200px;
    overflow-y: auto;
  }
  .check {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 32px;
    font-size: 13px;
    font-weight: 500;
    color: #25324a;
    cursor: pointer;
    overflow-wrap: anywhere;
  }
  .check input {
    width: 16px;
    height: 16px;
    margin: 0;
    flex: none;
    accent-color: #2563eb;
  }
  legend {
    display: flex;
    width: 100%;
    align-items: baseline;
    justify-content: space-between;
  }
  .reset {
    padding: 0;
    border: 0;
    background: none;
    font-size: 11px;
    font-weight: 600;
    color: #2563eb;
  }
  .metric-list {
    display: grid;
    gap: 2px;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .metric-row {
    display: flex;
    align-items: center;
    gap: 2px;
    border-radius: 6px;
    background: #fff;
  }
  .metric-row.dragging {
    position: relative;
    z-index: 1;
    background: #eef3ff;
    box-shadow: 0 2px 8px #17203326;
  }
  .metric-row .check {
    flex: 1;
    min-width: 0;
  }
  .metric-row small {
    font-size: 11px;
    font-weight: 500;
    color: #8a96a8;
    white-space: nowrap;
  }
  .handle {
    display: grid;
    place-items: center;
    flex: none;
    width: 20px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: none;
    color: #9aa5b6;
    cursor: grab;
    touch-action: none;
  }
  .handle:hover,
  .handle:focus-visible {
    color: #25324a;
    background: #e9edf4;
  }
  .dragging .handle {
    cursor: grabbing;
    color: #2563eb;
  }
  .handle svg {
    width: 10px;
    height: 16px;
    fill: currentColor;
  }
  .none {
    margin: 0;
    font-size: 12px;
    color: #65748b;
  }
  h2 {
    font-size: 16px;
    margin: 0 0 4px;
  }
  .export-actions {
    display: flex;
    gap: 8px;
    margin-top: auto;
  }
  .export-actions button {
    flex: 1;
    min-height: 40px;
  }
  .primary {
    background: #2563eb;
    border-color: #2563eb;
    color: #fff;
    font-weight: 600;
  }
  .primary:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .export-preview {
    overflow: auto;
    background: #e9edf4;
    padding: 16px;
    border-radius: 0 10px 10px 0;
  }
  .paper {
    width: 210mm;
    margin: 0 auto;
  }
  /* One A4 sheet per printed page group so the preview shows the page breaks. */
  .paper :global(.doc-page) {
    min-height: 297mm;
    margin-bottom: 16px;
    padding: 12mm;
    background: #fff;
    box-shadow: 0 1px 4px #17203326;
  }
  @media (max-width: 600px) {
    .export-dialog {
      width: calc(100vw - 16px);
      height: calc(100vh - 16px);
    }
    .export-layout {
      grid-template-columns: 1fr;
      grid-template-rows: auto 1fr;
    }
    .export-options {
      flex-direction: row;
      flex-wrap: wrap;
      gap: 12px;
      padding: 16px;
      border-right: none;
      border-bottom: 1px solid #edf0f5;
    }
    h2,
    .field,
    .export-actions {
      width: 100%;
    }
    fieldset {
      width: calc(50% - 6px);
    }
    fieldset:has(.metric-list) {
      width: 100%;
    }
    .scroll {
      max-height: 132px;
    }
    .handle {
      width: 28px;
      height: 44px;
    }
    .check {
      min-height: 44px;
    }
    .segmented button {
      min-height: 40px;
    }
    .export-actions button {
      min-height: 44px;
    }
    .export-preview {
      border-radius: 0 0 10px 10px;
    }
  }
</style>
