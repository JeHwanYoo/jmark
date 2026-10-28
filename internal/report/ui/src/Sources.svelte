<script>
  import { renderMarkdown } from "./markdown.js";
  let { markdown = "" } = $props();
  const html = $derived(renderMarkdown(markdown));
  function download() {
    const url = URL.createObjectURL(
      new Blob([markdown], { type: "text/markdown;charset=utf-8" }),
    );
    const link = document.createElement("a");
    link.href = url;
    link.download = "sources.md";
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
</script>

<details class="panel report-section sources">
  <summary>Source code</summary>
  <div class="sources-heading">
    {#if markdown}<button onclick={download}>Download Markdown</button>{/if}
  </div>
  {#if markdown}<div class="markdown">{@html html}</div>{:else}<p class="muted">
      Source snapshot unavailable.
    </p>{/if}
</details>
