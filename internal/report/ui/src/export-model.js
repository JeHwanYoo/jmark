import { chartData, format, unit } from "./model.js";

export const exportFormats = [
  { value: "pdf", label: "PDF / Print" },
  { value: "markdown", label: "Markdown" },
];
export const inclusion = [
  { value: "include", label: "Include" },
  { value: "omit", label: "Omit" },
];

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

const cell = (value) =>
  String(value ?? "—")
    .replace(/\\/g, "\\\\")
    .replace(/\|/g, "\\|")
    .replace(/\s+/g, " ");

export function reportMarkdown(report, include) {
  const showRun = report.rows.some((r) => r.run);
  const header = [
    ...(showRun ? ["Run"] : []),
    "Runtime",
    "Approach",
    unit(report.rows[0]),
    "Status",
    "Iterations",
  ];
  const align = header.map((_, i) =>
    i === header.length - 3 || i === header.length - 1 ? "--:" : "---",
  );
  const lines = [
    `# ${report.title}`,
    "",
    report.workload,
    "",
    report.metric,
    "",
    "## Results",
    "",
    `| ${header.map(cell).join(" | ")} |`,
    `| ${align.join(" | ")} |`,
    ...report.rows.map(
      (r) =>
        `| ${[
          ...(showRun ? [r.run] : []),
          r.runtime,
          r.name,
          format(r.value),
          statusText(r),
          format(r.iters_meta),
        ]
          .map(cell)
          .join(" | ")} |`,
    ),
    "",
  ];
  if (include.environment) {
    lines.push("## Environment", "");
    for (const row of report.environments) {
      const env = environment(row);
      lines.push(
        `- **${cell(env.title)}** — ${[env.version, env.host, env.settings].filter(Boolean).map(cell).join(" · ")}`,
      );
    }
    lines.push("");
  }
  if (include.sources && report.sources) {
    lines.push("## Source code", "", report.sources.trimEnd(), "");
  }
  return lines.join("\n");
}

export const markdownFilename = (title) =>
  `${String(title || "report").replace(/[^\w.-]+/g, "-")}.md`;
