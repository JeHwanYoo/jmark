import { mount } from "svelte";
import MetricsGuide from "./MetricsGuide.svelte";
import App from "./App.svelte";
import Index from "./Index.svelte";
import "highlight.js/styles/github.css";
import "@fontsource-variable/inter/wght.css";
import "@fontsource-variable/inter/wght-italic.css";
import "@fontsource-variable/jetbrains-mono/wght.css";
import "@fontsource-variable/jetbrains-mono/wght-italic.css";
import "./style.css";
const target = document.getElementById("app");
try {
  const index = document.getElementById("jmark-index-data");
  const data = document.getElementById("jmark-data");
  if (location.hash.startsWith("#metrics")) {
    const rows = data ? JSON.parse(data.textContent) : [];
    const catalog = JSON.parse(
      document.getElementById("jmark-metrics")?.textContent || "null",
    );
    mount(MetricsGuide, { target, props: { rows, catalog } });
  } else if (index) {
    mount(Index, { target, props: { runs: JSON.parse(index.textContent) } });
  } else {
    const rows = JSON.parse(data.textContent);
    const sources = JSON.parse(
      document.getElementById("jmark-sources")?.textContent || '""',
    );
    mount(App, {
      target,
      props: { rows, sources, indexHref: target.dataset.indexHref || "" },
    });
  }
} catch {
  target.textContent = "Unable to load results.";
}
