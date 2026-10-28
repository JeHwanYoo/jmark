import { test } from "node:test";
import assert from "node:assert/strict";
import {
  chartPages,
  reportMarkdown,
  markdownFilename,
} from "./export-model.js";

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
const report = {
  title: "demo",
  workload: "demo · n=100",
  metric: "Wall time · µs/op",
  rows: [row, { ...row, name: "a|b", status: "skip", reason: "unsupported", value: NaN }],
  environments: [row],
  sources: "## loop.js\n\n```javascript\nrun()\n```\n",
};

test("markdown keeps results and escapes table cells", () => {
  const md = reportMarkdown(report, { environment: false, sources: false });
  assert.match(md, /^# demo\n/);
  assert.match(md, /\| Runtime \| Approach \| µs\/op \| Status \| Iterations \|/);
  assert.match(md, /\| node \| a\\\|b \| — \| unsupported \| 20 \|/);
  assert.doesNotMatch(md, /## Environment|## Source code/);
});

test("markdown sections follow the chosen inclusions", () => {
  const md = reportMarkdown(report, { environment: true, sources: true });
  assert.match(md, /## Environment\n\n- \*\*node\*\* — v22 · darwin · arm64/);
  assert.match(md, /## Source code\n\n## loop\.js/);
});

test("chart pages split categories without dropping rows", () => {
  const rows = Array.from({ length: 30 }, (_, i) => ({ ...row, name: `a${i}` }));
  const pages = chartPages(rows, "name", "runtime");
  assert.ok(pages.length > 1);
  assert.equal(pages.flat().length, rows.length);
});

test("markdown filename is filesystem safe", () => {
  assert.equal(markdownFilename("sum large/doubles"), "sum-large-doubles.md");
});
