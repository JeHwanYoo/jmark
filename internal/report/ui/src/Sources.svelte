<script>
  import { renderMarkdown } from "./markdown.js";
  let { markdown = "" } = $props();
  const html = $derived(renderMarkdown(markdown));
</script>

<details class="panel report-section sources">
  <summary>Source code</summary>
  {#if markdown}<div class="markdown">{@html html}</div>{:else}<p class="muted">
      Source snapshot unavailable.
    </p>{/if}
</details>
