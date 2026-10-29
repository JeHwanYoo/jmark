<script>
  import { guideHref } from "./guide-model.js";
  let { metric = "wall_time" } = $props();
</script>

<a
  class="metric-help"
  href={guideHref(metric)}
  target="_blank"
  rel="noopener"
  aria-label="Explain this metric (opens a new tab)"
  title="Explain this metric">?</a
>

<style>
  .metric-help {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid #cad4e2;
    border-radius: 50%;
    color: #526078;
    background: #fff;
    font-size: 15px;
    text-decoration: none;
    flex: none;
    margin-bottom: 6px;
  }
  .metric-help:hover {
    color: #2563eb;
    border-color: #2563eb;
  }
  .metric-help:focus-visible {
    outline: 3px solid #2563eb;
    outline-offset: 3px;
  }
  @media (pointer: coarse) {
    .metric-help {
      width: 44px;
      height: 44px;
      margin-bottom: 0;
    }
  }
</style>
