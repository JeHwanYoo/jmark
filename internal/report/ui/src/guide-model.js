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
