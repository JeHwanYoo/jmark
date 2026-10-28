import { test } from "node:test";
import assert from "node:assert/strict";
import { conditions, selectRows, chartOption, chartData } from "./model.js";
const base = {
  project: "demo",
  metric: "wall_time",
 unit: "s/op",
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

    { ...base, warmup: 2 },
    { ...base, iters_meta: 40 },
    { ...base, runs: 10 },
  ];
  const groups = conditions(rows);
  assert.equal(groups.length, 6);
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

test("workload metadata supports arbitrary parameters and units", () => {
  const first = { ...base, metric: "throughput", unit: "items/s", parameters: { size: 64, distribution: "uniform" } };
  const reordered = { ...first, parameters: { distribution: "uniform", size: 64 } };
  const larger = { ...first, parameters: { size: 128, distribution: "uniform" } };
  const otherUnit = { ...first, unit: "MB/s" };
  const groups = conditions([first, reordered, larger, otherUnit]);
  assert.equal(groups.length, 2);
  assert.deepEqual(selectRows([first, reordered, larger], groups[0].key, "", ""), [first, reordered]);
  assert.match(groups[0].label, /size=64/);
  assert.doesNotMatch(groups[0].label, /n=100/);
  assert.equal(chartOption([first], "linear").xAxis.name, "items/s");
  assert.match(chartOption([first], "linear").tooltip.formatter({data: {row: first}}), /items\/s/);
});

test("metric definitions isolate units, methods and scopes independently of workloads", async () => {
 const {metrics,displayRow,orderRows}=await import("./model.js");
 const rows=[{...base,metric:"custom",unit:"score",method:"absolute",scope:"process",direction:"higher",value:2},{...base,metric:"custom",unit:"score",method:"absolute",scope:"process",direction:"higher",value:5}];
 assert.equal(metrics(rows).length,1);
 assert.equal(metrics([...rows,{...rows[0],scope:"operation"}]).length,2);
 assert.equal(orderRows(rows,"value")[0].value,5);
 assert.equal(displayRow({...base,unit:"B",value:1048576}).value,1);
 assert.equal(displayRow({...base,unit:"s/op",value:0.000002}).value,2);
 assert.equal(displayRow(rows[0]).unit,"score");
});
