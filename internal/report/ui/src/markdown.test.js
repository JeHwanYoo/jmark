import test from "node:test";
import assert from "node:assert/strict";
import { renderMarkdown } from "./markdown.js";

test("source fences highlight code and escape markup", () => {
  const html = renderMarkdown(
    '## Source\n\n```javascript\nconst value = "<script>alert(1)</script>";\n```',
  );
  assert.match(html, /hljs-keyword/);
  assert.match(html, /&lt;script&gt;/);
  assert.doesNotMatch(html, /<script>/);
});
test("markdown cannot inject HTML, executable links, or remote images", () => {
  const html = renderMarkdown(
    "<img src=x onerror=alert(1)>\n\n[click](javascript:alert(1))\n\n![remote](https://example.com/a.png)",
  );
  assert.doesNotMatch(html, /<img|href="javascript:/);
});
test("unrecognized syntax stays readable", () => {
  const html = renderMarkdown("```new-runtime\nvalue < limit\n```");
  assert.match(html, /value &lt; limit/);
});
