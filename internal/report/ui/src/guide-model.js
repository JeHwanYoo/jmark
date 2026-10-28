export function metricSections(document) {
  const pattern = /^## (.+) \(`([a-z_]+)`\)\s*$/gm;
  const matches = [...document.matchAll(pattern)];
  return matches.map((match, index) => {
    let body = document.slice(
      match.index + match[0].length,
      matches[index + 1]?.index ?? document.length,
    );
    body = body.split(/^## /m)[0].trim();
    const [summary, ...paragraphs] = body.split(/\n\s*\n/);
    return {
      id: match[2],
      label: match[1],
      summary: summary.replace(/\n/g, " "),
      body: paragraphs.join("\n\n"),
    };
  });
}
export const platforms = ["darwin", "linux", "windows"];
export const platformName = (os) =>
  ({ darwin: "macOS", linux: "Linux", windows: "Windows" })[os] ?? os;
export function hostPlatform(host) {
  const os = String(host?.os ?? "").toLowerCase();
  if (/^(mingw|msys|cygwin|windows)/.test(os)) return "windows";
  return os || undefined;
}
export function supported(catalog, id, os) {
  const readings = catalog?.metrics?.[id]?.readings;
  if (!readings || !os) return undefined;
  return readings.every((reading) =>
    catalog.readings.find((r) => r.id === reading)?.platforms.includes(os),
  );
}
// Collected readings come from row observations: hyperfine exports every reading it gathered.
export function metricStatus(id, rows, catalog) {
  const own = rows.filter((row) => row.metric === id);
  if (own.some((row) => row.status === "ok")) return { kind: "measured" };
  if (own.length) return { kind: "unavailable", reason: own[0].reason };
  const readings = catalog?.metrics?.[id]?.readings;
  if (!readings || !rows.length) return { kind: "unknown" };
  const collected = new Set(rows.flatMap((row) => Object.keys(row.observations ?? {})));
  const missing = readings.filter((reading) => !collected.has(reading));
  if (!missing.length) return { kind: "available" };
  const os = hostPlatform(rows[0].host);
  if (supported(catalog, id, os) === false)
    return { kind: "unsupported", os, readings: missing };
  return { kind: "withheld", readings: missing };
}
const reasons = {
  unsupported_on_platform: "not supported on this OS",
  not_reported_by_collector: "not reported on this host",
  below_noise_floor: "below measurement resolution",
  zero_denominator: "zero denominator",
};
export function statusLabel(status) {
  switch (status.kind) {
    case "measured":
      return "Measured";
    case "unavailable":
      return `Unavailable · ${reasons[status.reason] ?? status.reason ?? "unknown reason"}`;
    case "available":
      return "Available, not requested";
    case "unsupported":
      return `Not supported on ${platformName(status.os)}`;
    case "withheld":
      return "Not reported on this host";
    default:
      return "";
  }
}
export function guideSections(sections, rows = [], catalog) {
  const measured = new Map(rows.map((row) => [row.metric, row.label || row.metric]));
  const custom = [...measured]
    .filter(([id]) => !sections.some((section) => section.id === id))
    .map(([id, label]) => ({ id, label }));
  const all = [...sections, ...custom].map((section) => ({
    ...section,
    status: metricStatus(section.id, rows, catalog),
  }));
  if (!rows.length) return { inReport: [], other: all };
  return {
    inReport: all.filter((section) => measured.has(section.id)),
    other: all.filter((section) => !measured.has(section.id)),
  };
}
export const guideHref = (id) =>
  `#metrics/${encodeURIComponent(id || "wall_time")}`;
export function guideMetric(hash) {
  try {
    return decodeURIComponent(hash.replace(/^#metrics\/?/, "")) || "wall_time";
  } catch {
    return "wall_time";
  }
}
export function animationKind(id) {
  if (["wall_time", "cpu_time", "user_time", "system_time"].includes(id))
    return "time";
  if (id === "peak_rss") return "memory";
  if (id === "throughput") return "throughput";
  if (["instructions", "cpu_cycles", "ipc"].includes(id)) return "cpu";
  if (id === "branch_misses") return "branch";
  return "cache";
}
export const memorySamples = [20, 38, 70, 58, 95, 120, 82, 56, 72, 45];
export const missIndices = [2, 7, 12, 17];
