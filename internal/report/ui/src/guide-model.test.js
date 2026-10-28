import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import {
  metricSections,
  guideSections,
  metricStatus,
  statusLabel,
  supported,
  guideMetric,
  guideHref,
  animationKind,
} from "./guide-model.js";
const document = readFileSync(
  new URL("../../../../METRICS.md", import.meta.url),
  "utf8",
);
test("every documented metric has a definition, measurement and animation", () => {
  const sections = metricSections(document);
  assert.equal(sections.length, 13);
  for (const section of sections) {
    assert.ok(section.summary.length > 10);
    assert.match(section.body, /Measurement:/);
    assert.ok(animationKind(section.id));
  }
  assert.doesNotMatch(sections.at(-1).body, /결과를 읽을 때/);
});
test("help links round trip and tolerate malformed hashes", () => {
  assert.equal(guideMetric(guideHref("peak_rss")), "peak_rss");
  assert.equal(guideMetric("#metrics"), "wall_time");
  assert.equal(guideMetric("#metrics/%invalid"), "wall_time");
  assert.equal(
    guideMetric(guideHref("probe.custom.counter")),
    "probe.custom.counter",
  );
});
const catalog = {
  metrics: {
    wall_time: { label: "Wall time", readings: ["time_wall_clock"] },
    user_time: { label: "User CPU time", readings: ["time_user"] },
    peak_rss: { label: "Peak RSS", readings: ["memory_peak_resident"] },
    ipc: { label: "IPC", readings: ["instructions", "cpu_cycles"] },
    cache_misses: { label: "Cache misses", readings: ["cache_misses"] },
    branch_misses: { label: "Branch misses", readings: ["branch_misses"] },
  },
  readings: [
    { id: "time_wall_clock", platforms: ["darwin", "linux", "windows"] },
    { id: "time_user", platforms: ["darwin", "linux", "windows"] },
    { id: "memory_peak_resident", platforms: ["darwin", "linux"] },
    { id: "instructions", platforms: ["darwin", "linux"] },
    { id: "cpu_cycles", platforms: ["darwin", "linux"] },
    { id: "cache_misses", platforms: ["linux"] },
    { id: "branch_misses", platforms: ["linux"] },
  ],
};
const observed = (os, ...keys) => ({
  host: { os },
  observations: Object.fromEntries(keys.map((key) => [key, {}])),
});
test("metric status separates measured, requested, available, and unsupported", () => {
  const mac = observed("Darwin", "time_wall_clock", "time_user", "instructions", "cpu_cycles");
  const rows = [
    { ...mac, metric: "wall_time", status: "ok" },
    { ...mac, metric: "cache_misses", status: "unavailable", reason: "unsupported_on_platform" },
  ];
  assert.deepEqual(metricStatus("wall_time", rows, catalog), { kind: "measured" });
  assert.equal(metricStatus("cache_misses", rows, catalog).reason, "unsupported_on_platform");
  assert.deepEqual(metricStatus("user_time", rows, catalog), { kind: "available" });
  assert.deepEqual(metricStatus("ipc", rows, catalog), { kind: "available" });
  assert.deepEqual(metricStatus("branch_misses", rows, catalog), {
    kind: "unsupported",
    os: "darwin",
    readings: ["branch_misses"],
  });
  assert.equal(statusLabel(metricStatus("branch_misses", rows, catalog)), "Not supported on macOS");
});
test("supported counters missing on Linux are reported as withheld", () => {
  const rows = [{ ...observed("Linux", "time_wall_clock"), metric: "wall_time", status: "ok" }];
  assert.deepEqual(metricStatus("branch_misses", rows, catalog), {
    kind: "withheld",
    readings: ["branch_misses"],
  });
  assert.equal(supported(catalog, "peak_rss", "windows"), false);
  assert.equal(supported(catalog, "probe.custom", "linux"), undefined);
});
test("guide groups metrics in the report before the rest", () => {
  const sections = metricSections(document);
  const rows = [
    { metric: "wall_time", label: "Wall time", status: "ok" },
    { metric: "peak_rss", label: "Peak RSS", status: "ok" },
    { metric: "probe.custom.counter", label: "Custom counter", status: "ok" },
  ];
  const { inReport, other } = guideSections(sections, rows, catalog);
  assert.deepEqual(
    inReport.map((section) => section.id),
    ["wall_time", "peak_rss", "probe.custom.counter"],
  );
  assert.equal(inReport.length + other.length, sections.length + 1);
  assert.equal(other.some((section) => section.id === "wall_time"), false);
  const empty = guideSections(sections, [], catalog);
  assert.equal(empty.inReport.length, 0);
  assert.equal(empty.other.length, sections.length);
});
