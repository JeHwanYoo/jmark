// Loop driver shared by every js-family runtime. Never reads a clock; the host times this process.
// argv: <input> <approach> <n> <iters> [<probe_out> <hook>...]  (probe args only in the untimed probe pass)
import { basename } from "node:path";
import { writeFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

const [inputPath, approachPath, n, iters, probeOut, ...hookPaths] = process.argv.slice(2);
const load = (path) => import(pathToFileURL(path).href);

const { makeInput } = await load(inputPath);
const { run } = await load(approachPath);
const count = Number(iters);
const data = makeInput(Number(n));

const hooks = probeOut ? await Promise.all(hookPaths.map(load)) : [];
const descriptions = hooks.map(h=>h.describe());
const before = hooks.map((h) => h.snapshot());
let sink = 0;
for (let i = 0; i < count; i++) sink += run(data);
const after = hooks.map((h) => h.snapshot());

process.stdout.write(`${sink}\n`);
if (probeOut) {
  const lines = after.flatMap((end, i) =>
    Object.keys(end).map((k) => `${basename(hookPaths[i]).split(".")[0]}.${k}\t${descriptions[i][k].unit}\t${descriptions[i][k].direction}\t${end[k] - (before[i][k] ?? 0)}\n`),
  );
  writeFileSync(probeOut, lines.join(""));
}
