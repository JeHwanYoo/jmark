import { mount } from "svelte";
import App from "./App.svelte";
import Index from "./Index.svelte";
import "highlight.js/styles/github.css";
import "./style.css";
const target = document.getElementById("app");
try {
  const index = document.getElementById("jmark-index-data");
  if (index) {
    mount(Index, { target, props: { runs: JSON.parse(index.textContent) } });
  } else {
    const rows = JSON.parse(document.getElementById("jmark-data").textContent);
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
