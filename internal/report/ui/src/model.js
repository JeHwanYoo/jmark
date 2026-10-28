export const colors = [
  "#2563eb",
  "#0d9488",
  "#d97706",
  "#9333ea",
  "#db2777",
  "#475569",
];
export const unit = (row) => row?.unit || "—";
export function displayRow(row) {
 if (!Number.isFinite(row.value)) return row;
 let factor=1, label=row.unit;
 if(row.unit==="s/op") {factor=1e6;label="µs/op";}
 if(row.unit==="B") {factor=1/1048576;label="MiB";}
 if(row.unit==="ratio") {factor=100;label="%";}
 return {...row,value:row.value*factor,unit:label};
}
export const metricKey = r => JSON.stringify([r.metric,r.unit,r.method,r.scope]);
export const metrics = rows => [...new Map(rows.map(r=>[metricKey(r),{key:metricKey(r),label:r.label || r.metric}])).values()];
export function parameters(row) {
  const values = row?.parameters ?? (row?.n != null ? { n: row.n } : {});
  return Object.entries(values).sort(([a], [b]) => a.localeCompare(b));
}
export const parameterValue = (value) =>
  typeof value === "object" ? JSON.stringify(value) : String(value);

export const format = (value) =>
  Number.isFinite(value)
    ? new Intl.NumberFormat("en", { maximumSignificantDigits: 5 }).format(value)
    : "—";
export const conditionKey = (r) =>
  JSON.stringify([r.project, parameters(r), r.warmup, r.iters_meta, r.runs]);
export function conditions(rows) {
  return [...new Map(rows.map((r) => [conditionKey(r), r])).entries()].map(
    ([key, r]) => ({
      key,
      label: `${r.project}${parameters(r).map(([key, value]) => ` · ${key}=${parameterValue(value)}`).join("")} · warmup=${r.warmup ?? "—"} · iters=${r.iters_meta ?? "—"} · runs=${r.runs ?? "—"}`,
    }),
  );
}
export function selectRows(rows, condition, runtime, run) {
  return rows.filter(
    (r) =>
      conditionKey(r) === condition &&
      (!runtime || r.runtime === runtime) &&
      (!run || r.run === run),
  );
}
export const dimensions = [
  { value: "name", label: "Approach" },
  { value: "runtime", label: "Runtime" },
  { value: "run", label: "Run" },
];
export function chartData(rows, axis = "name", series = "runtime") {
  const remaining = dimensions
    .map((d) => d.value)
    .filter(
      (key) =>
        key !== axis &&
        key !== series &&
        new Set(rows.map((r) => r[key])).size > 1,
    );
  const categories = [];
  const categoryIndex = new Map();
  const seriesNames = [...new Set(rows.map((r) => r[series] ?? "—"))];
  const counts = new Map();
  const points = rows.map((row) => {
    const parts = [
      row[axis] ?? "—",
      ...remaining.map((key) => row[key] ?? "—"),
    ];
    const identity = JSON.stringify([parts, row[series]]);
    const occurrence = counts.get(identity) ?? 0;
    counts.set(identity, occurrence + 1);
    const key = JSON.stringify([parts, occurrence]);
    if (!categoryIndex.has(key)) {
      categoryIndex.set(key, categories.length);
      categories.push(
        parts.join(" · ") + (occurrence ? ` · #${occurrence + 1}` : ""),
      );
    }
    return { row, category: categoryIndex.get(key) };
  });
  return { categories, seriesNames, points };
}
export function chartOption(
  rows,
  scale,
  axis = "name",
  series = "runtime",
  domain = [],
) {
  const { categories, seriesNames, points } = chartData(rows, axis, series);
  const logarithmic = scale === "log" && rows.every((r) => r.value > 0);
  return {
    animation: false,
    grid: { left: 16, right: 80, top: 30, bottom: 35, containLabel: true },
    tooltip: {
      trigger: "item",
      renderMode: "richText",
      formatter: (p) =>
        `${[p.data.row.name, p.data.row.runtime, p.data.row.run].filter(Boolean).join(" · ")}\n${format(p.data.row.value)} ${unit(p.data.row)}`,
    },
    xAxis: {
      type: logarithmic ? "log" : "value",
      name: unit(rows[0]),
      nameLocation: "end",
      splitNumber: 4,
      axisLabel: { hideOverlap: true },
      splitLine: { lineStyle: { color: "#edf0f4" } },
    },
    yAxis: {
      type: "category",
      inverse: true,
      data: categories,
      axisTick: { show: false },
      axisLine: { show: false },
      axisLabel: { width: 130, overflow: "truncate" },
    },
    series: seriesNames.map((name, index) => ({
      name,
      type: logarithmic ? "scatter" : "bar",
      symbolSize: 11,
      symbolOffset: [
        0,
        logarithmic ? (index - (seriesNames.length - 1) / 2) * 24 : 0,
      ],
      barMaxWidth: 22,
      barGap: "30%",
      itemStyle: {
        color:
          colors[
            Math.max(0, (domain.length ? domain : seriesNames).indexOf(name)) %
              colors.length
          ],
      },
      label: {
        show: true,
        position: "right",
        formatter: (p) => format(p.value[0]),
        color: "#334155",
      },
      data: points
        .filter((p) => (p.row[series] ?? "—") === name)
        .map((p) => ({ value: [p.row.value, p.category], row: p.row })),
    })),
  };
}

export const orderRows = (rows,sort) => [...rows].sort((a,b)=> {
 if(sort!=="value") return a.runtime.localeCompare(b.runtime);
 if(!Number.isFinite(a.value)) return Number.isFinite(b.value)?1:0;
 if(!Number.isFinite(b.value)) return -1;
 return (a.direction==="higher"?-1:1)*(a.value-b.value);
});
