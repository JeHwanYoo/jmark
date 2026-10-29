import { test } from "node:test";
import assert from "node:assert/strict";
import {
  chartPages,
  environment,
  statusText,
  standaloneHtml,
  exportFilename,
  chartBounds,
  metricSections,
  moveKey,
  orderedKeys,
} from "./export-model.js";
import { metrics } from "./model.js";

const row = {
  runtime: "node",
  name: "loop",
  unit: "µs/op",
  value: 2,
  status: "ok",
  iters_meta: 20,
  runtime_version: "v22",
  host: { os: "darwin", arch: "arm64" },
  captured_at: "2026-10-28",
  warmup: 1,
  runs: 3,
};

test("chart pages split categories without dropping rows", () => {
  const rows = Array.from({ length: 30 }, (_, i) => ({ ...row, name: `a${i}` }));
  const pages = chartPages(rows, "name", "runtime");
  assert.ok(pages.length > 1);
  assert.equal(pages.flat().length, rows.length);
});

test("status keeps the reason for unmeasured rows", () => {
  assert.equal(statusText(row), "Measured");
  assert.equal(statusText({ status: "skip", reason: "unsupported" }), "unsupported");
});

test("environment joins host fields and run settings", () => {
  const env = environment({ ...row, run: "r1" });
  assert.equal(env.title, "node · r1");
  assert.equal(env.host, "darwin · arm64");
  assert.equal(env.settings, "2026-10-28 · warmup=1 · iters=20 · runs=3");
});

test("standalone html escapes the title and closes embedded styles", () => {
  const html = standaloneHtml("a<b>", ["x{}</style><script>"], "<main></main>");
  assert.match(html, /<title>a&lt;b&gt;<\/title>/);
  assert.doesNotMatch(html, /<\/style><script>/);
  assert.match(html, /<body><main><\/main><\/body>/);
});

test("export filename is filesystem safe", () => {
  assert.equal(exportFilename("sum large/doubles", "html"), "sum-large-doubles.html");
});

test("chart bounds start linear bars at zero and skip flat data", () => {
  assert.deepEqual(chartBounds([{ value: 2 }, { value: 5 }], "linear"), { min: 0, max: 5 });
  assert.deepEqual(chartBounds([{ value: 2 }, { value: 5 }], "log"), { min: 2, max: 5 });
  assert.equal(chartBounds([{ value: 2 }], "linear"), undefined);
});

test("metric sections follow choices and keep log scale only for positive values", () => {
  const rows = [
    { ...row, metric: "wall_time", unit: "s/op", value: 2e-6 },
    { ...row, metric: "system_time", unit: "s/op", value: null, status: "unavailable" },
  ];
  const key = (metric) => JSON.stringify([metric, "s/op", undefined, undefined]);
  const sections = metricSections(
    rows,
    [
      { key: key("system_time"), label: "System CPU time" },
      { key: key("wall_time"), label: "Wall time" },
    ],
    "value",
    "log",
  );
  assert.deepEqual(
    sections.map((s) => [s.title, s.rows.length, s.chartRows.length, s.scale]),
    [
      ["System CPU time · µs/op", 1, 0, "linear"],
      ["Wall time · µs/op", 1, 1, "log"],
    ],
  );
  assert.equal(sections[1].rows[0].value, 2);
});

test("metrics list wall time first in a fixed order and unknown metrics last", () => {
  const ids = ["custom", "ipc", "peak_rss", "throughput", "wall_time", "cpu_time"];
  const rows = ids.map((metric) => ({ ...row, metric }));
  assert.deepEqual(
    metrics(rows).map((m) => m.id),
    ["wall_time", "throughput", "cpu_time", "peak_rss", "ipc", "custom"],
  );
});

test("custom metric order survives new keys and clamps moves", () => {
  assert.deepEqual(orderedKeys(["a", "b", "c"], ["c", "x", "a"]), ["c", "a", "b"]);
  assert.deepEqual(moveKey(["a", "b", "c"], "a", 2), ["b", "c", "a"]);
  assert.deepEqual(moveKey(["a", "b", "c"], "c", -1), ["c", "a", "b"]);
  assert.deepEqual(moveKey(["a", "b", "c"], "a", 9), ["b", "c", "a"]);
});
