import { test } from "node:test";
import assert from "node:assert/strict";
import { conditions, selectRows, chartOption, chartData } from "./model.js";
const base = {
  project: "demo",
  metric: "us_per_op",
  n: 100,
  warmup: 1,
  iters_meta: 20,
  runs: 3,
  runtime: "node",
  name: "loop",
  value: 2,
};
test("incompatible workloads never share a chart", () => {
  const rows = [
    base,
    { ...base, n: 200 },
    { ...base, project: "other" },
    { ...base, metric: "bytes" },
    { ...base, warmup: 2 },
    { ...base, iters_meta: 40 },
    { ...base, runs: 10 },
  ];
  const groups = conditions(rows);
  assert.equal(groups.length, 7);
  assert.deepEqual(selectRows(rows, groups[0].key, "", ""), [base]);
});
test("filters retain duplicate measurements and distinguish runs", () => {
  const rows = [
    { ...base, run: "one" },
    { ...base, run: "two" },
    { ...base, run: "two" },
  ];
  assert.equal(
    selectRows(rows, conditions(rows)[0].key, "node", "two").length,
    2,
  );
  assert.equal(selectRows(rows, conditions(rows)[0].key, "py", "").length, 0);
});
test("log scale uses points; zero falls back without losing values", () => {
  const option = chartOption([base], "log");
  assert.equal(option.xAxis.type, "log");
  assert.equal(option.series[0].type, "scatter");
  const zero = chartOption([{ ...base, value: 0 }], "log");
  assert.equal(zero.xAxis.type, "value");
  assert.equal(zero.series[0].data[0].value[0], 0);
  assert.equal(zero.tooltip.renderMode, "richText");
});

test("approaches share one axis and can pivot by runtime", () => {
  const rows = [
    base,
    { ...base, name: "map", value: 200 },
    { ...base, runtime: "bun", value: 1 },
  ];
  const option = chartOption(rows, "linear");
  assert.deepEqual(option.yAxis.data, ["loop", "map"]);
  assert.equal(option.series.length, 2);
  assert.equal(option.series.flatMap((s) => s.data).length, 3);
  assert.equal(option.series[0].data[1].value[0], 200);
  const pivot = chartOption(rows, "linear", "runtime", "name");
  assert.deepEqual(pivot.yAxis.data, ["node", "bun"]);
  assert.deepEqual(
    pivot.series.map((s) => s.name),
    ["loop", "map"],
  );
});
test("remaining dimensions and duplicates are preserved without constant run noise", () => {
  const one = chartData([{ ...base, run: "long/run/id" }]);
  assert.deepEqual(one.categories, ["loop"]);
  const rows = [
    { ...base, run: "one" },
    { ...base, run: "two" },
    { ...base, run: "two" },
  ];
  const data = chartData(rows);
  assert.deepEqual(data.categories, [
    "loop · one",
    "loop · two",
    "loop · two · #2",
  ]);
  assert.equal(data.points.length, 3);
  const pivot = chartOption(rows, "linear", "run", "name");
  assert.equal(pivot.series.flatMap((s) => s.data).length, 3);
});
