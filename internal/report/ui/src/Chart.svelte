<script>
  import { onMount } from "svelte";
  import { init, use } from "echarts/core";
  import { BarChart, ScatterChart } from "echarts/charts";
  import { GridComponent, TooltipComponent } from "echarts/components";
  import { SVGRenderer } from "echarts/renderers";
  import { chartOption, chartData } from "./model.js";
  use([BarChart, ScatterChart, GridComponent, TooltipComponent, SVGRenderer]);
  let { rows, scale, axis, series, domain, bounds = undefined, print = false } = $props();
  const layout = $derived(chartData(rows, axis, series));
  const height = $derived(
    Math.min(print ? 600 : Infinity, Math.max(
      280,
      layout.categories.length * (layout.seriesNames.length * 32 + 24) + 80,
    )),
  );
  let element;
  let chart = $state.raw();
  onMount(() => {
    chart = init(element, null, { renderer: "svg" });
    document.fonts.ready.then(() => chart?.resize());
    const observer = new ResizeObserver(() => chart?.resize());
    observer.observe(element);
    return () => {
      observer.disconnect();
      chart.dispose();
      chart = undefined;
    };
  });
  $effect(() => {
    if (chart) {
      const fontFamily = getComputedStyle(element).getPropertyValue("--font-ui").trim();
      const option = chartOption(rows, scale, axis, series, domain);
      if (bounds) Object.assign(option.xAxis, bounds);
      option.textStyle = { fontFamily };
      option.tooltip.textStyle = { fontFamily };
      chart.setOption(option, true);
    }
  });
</script>

<div
  class="chart"
  bind:this={element}
  style:height={`${height}px`}
  role="img"
  aria-label="Benchmark comparison chart"
></div>
