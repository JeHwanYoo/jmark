import { chartData, displayRow, metricKey, orderRows, unit } from "./model.js";

// One printed section per metric, filtered and ordered like the on-screen chart.
export function metricSections(rows, choices, sort, scale) {
  return choices.map((choice) => {
    const matching = rows.filter((row) => metricKey(row) === choice.key);
    const sectionRows = orderRows(matching.map(displayRow), sort);
    const chartRows = sectionRows.filter(
      (row) => row.status === "ok" && Number.isFinite(row.value),
    );
    // Convert the unit even when the metric has no measured value.
    const label = unit(displayRow({ ...matching[0], value: 0 }));
    return {
      key: choice.key,
      label: choice.label,
      title: `${choice.label} · ${label}`,
      unit: label,
      rows: sectionRows,
      chartRows,
      scale:
        chartRows.length && chartRows.every((row) => row.value > 0)
          ? scale
          : "linear",
    };
  });
}

// One shared value axis across every printed chart page.
export function chartBounds(rows, scale) {
  const values = rows.map((row) => row.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  if (!values.length || min === max) return undefined;
  return {
    min: scale === "log" ? min : Math.min(0, min),
    max: scale === "log" ? max : Math.max(0, max),
  };
}

// Split chart rows by category so each printed chart fits on one A4 page.
export function chartPages(rows, axis, series, budget = 520) {
  const layout = chartData(rows, axis, series);
  const perPage = Math.max(
    1,
    Math.floor(budget / (layout.seriesNames.length * 32 + 24)),
  );
  const pages = [];
  for (let start = 0; start < layout.categories.length; start += perPage) {
    pages.push(
      layout.points
        .filter((p) => p.category >= start && p.category < start + perPage)
        .map((p) => p.row),
    );
  }
  return pages;
}

// Saved keys keep their order; keys not yet ordered follow in default order.
export const orderedKeys = (keys, order) => [
  ...order.filter((key) => keys.includes(key)),
  ...keys.filter((key) => !order.includes(key)),
];

export function moveKey(keys, key, index) {
  const rest = keys.filter((other) => other !== key);
  rest.splice(Math.max(0, Math.min(index, rest.length)), 0, key);
  return rest;
}

export const statusText = (row) =>
  row.status === "ok" ? "Measured" : (row.reason ?? row.status ?? "—");

export function environment(row) {
  return {
    title: `${row.runtime}${row.run ? ` · ${row.run}` : ""}`,
    version: row.runtime_version ?? "—",
    host: [row.host?.os, row.host?.arch, row.host?.cpu]
      .filter(Boolean)
      .join(" · "),
    settings: `${row.captured_at ?? "—"} · warmup=${row.warmup ?? "—"} · iters=${row.iters_meta ?? "—"} · runs=${row.runs ?? "—"}`,
  };
}

export const exportFormats = [
  { value: "pdf", label: "PDF" },
  { value: "html", label: "HTML" },
];

export const exportFilename = (title, extension) =>
  `${String(title || "report").replace(/[^\w.-]+/g, "-")}.${extension}`;

const escapeHtml = (text) =>
  String(text).replace(
    /[&<>"]/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c],
  );

// The exported page reuses the report's own styles so it matches the preview,
// and keeps an A4 viewport so phones zoom out instead of reflowing the page.
export function standaloneHtml(title, styles, body) {
  return `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=840"><title>${escapeHtml(title)}</title>
<style>${styles.join("\n").replace(/<\/style/gi, "<\\/style")}</style>
<style>body{padding:16px 0}.export-paper{width:210mm;margin:0 auto}.export-paper .doc-page{min-height:297mm;margin-bottom:16px;padding:12mm;background:#fff;box-shadow:0 1px 4px #17203326}@page{size:A4 portrait;margin:0 12mm}@media print{:root{background:#fff}body{padding:0}.export-paper{width:auto}.export-paper .doc-page{min-height:0;margin:0;padding:12mm 0;box-shadow:none;box-decoration-break:clone;-webkit-box-decoration-break:clone}}</style>
</head>
<body>${body}</body>
</html>
`;
}
