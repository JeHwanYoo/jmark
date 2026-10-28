import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import {
  metricSections,
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
    assert.match(section.body, /측정:/);
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
