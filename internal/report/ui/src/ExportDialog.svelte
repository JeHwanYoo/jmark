<script>
  import { tick } from "svelte";
  import SearchSelect from "./SearchSelect.svelte";
  import ReportDocument from "./ReportDocument.svelte";
  import { renderMarkdown } from "./markdown.js";
  import {
    exportFormats,
    inclusion,
    reportMarkdown,
    markdownFilename,
  } from "./export-model.js";
  let { report, open = $bindable(false) } = $props();
  let dialog;
  let outputFormat = $state("pdf");
  let chart = $state("include");
  let environment = $state("include");
  let sources = $state("omit");
  let printing = $state(false);
  let previewWidth = $state(0);
  const include = $derived({
    chart: outputFormat === "pdf" && chart === "include",
    environment: environment === "include",
    sources: sources === "include",
  });
  const markdown = $derived(open ? reportMarkdown(report, include) : "");
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
    const url = URL.createObjectURL(
      new Blob([markdown], { type: "text/markdown;charset=utf-8" }),
    );
    const link = document.createElement("a");
    link.href = url;
    link.download = markdownFilename(report.title);
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
        <SearchSelect
          id="export-format"
          label="Format"
          options={exportFormats}
          bind:value={outputFormat}
        />
        {#if outputFormat === "pdf"}
          <SearchSelect
            id="export-chart"
            label="Chart"
            options={inclusion}
            bind:value={chart}
          />
        {/if}
        <SearchSelect
          id="export-environment"
          label="Environment"
          options={inclusion}
          bind:value={environment}
        />
        <SearchSelect
          id="export-sources"
          label="Source code"
          options={inclusion}
          bind:value={sources}
        />
        <div class="export-actions">
          <button type="button" onclick={() => (open = false)}>Cancel</button>
          {#if outputFormat === "pdf"}
            <button type="button" class="primary" onclick={print}>Print</button>
          {:else}
            <button type="button" class="primary" onclick={download}
              >Download</button
            >
          {/if}
        </div>
      </div>
      <div
        class="export-preview"
        bind:clientWidth={previewWidth}
        aria-label="Preview"
      >
        <div class="paper" style:zoom>
          {#if outputFormat === "pdf"}
            <ReportDocument {report} {include} />
          {:else}
            <div class="markdown">{@html renderMarkdown(markdown)}</div>
          {/if}
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
    border-right: 1px solid #edf0f5;
  }
  .export-options :global(.search-select) {
    width: 100%;
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
  .export-preview {
    overflow: auto;
    background: #e9edf4;
    padding: 16px;
    border-radius: 0 10px 10px 0;
  }
  .paper {
    width: 210mm;
    min-height: 297mm;
    margin: 0 auto;
    padding: 12mm;
    background: #fff;
    box-shadow: 0 1px 4px #17203326;
  }
  .paper .markdown {
    font-size: 10pt;
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
    .export-options :global(.search-select) {
      width: calc(50% - 6px);
    }
    h2,
    .export-actions {
      width: 100%;
    }
    .export-actions button {
      min-height: 44px;
    }
    .export-preview {
      border-radius: 0 0 10px 10px;
    }
  }
</style>
