## benches/sum-large-doubles/bench.toml

```toml
n = 100000
warmup = 20
iters = 500
runs = 10
```

## benches/sum-large-doubles/js/\_input.mjs

```javascript
export function makeInput(n) {
  return Array.from({ length: n }, (_, i) => (i % 20) + 1);
}
```

## benches/sum-large-doubles/js/for-loop.mjs

```javascript
export function run(xs) {
  const length = xs.length;
  let sum = 0;
  for (let i = 0; i < length; i++) {
    const value = xs[i] * 2;
    if (value > 10) sum += value;
  }
  return sum;
}
```

## benches/sum-large-doubles/js/map-filter-reduce.mjs

```javascript
export function run(xs) {
  return xs
    .map((x) => x * 2)
    .filter((x) => x > 10)
    .reduce((sum, x) => sum + x, 0);
}
```

## benches/sum-large-doubles/py/\_input.py

```python
def make_input(n: int) -> list[int]:
    return [(i % 20) + 1 for i in range(n)]
```

## benches/sum-large-doubles/py/for-loop.py

```python
def run(xs: list[int]) -> int:
    total = 0
    for v in xs:
        value = v * 2
        if value > 10:
            total += value
    return total
```

## benches/sum-large-doubles/py/map-filter-reduce.py

```python
def run(xs: list[int]) -> int:
    return sum(x for x in (v * 2 for v in xs) if x > 10)
```

## families/js/runner.mjs

```javascript
// Loop driver shared by every js-family runtime. Never reads a clock; the host times this process.
// argv: <input> <approach> <n> <iters> [<probe_out> <hook>...]  (probe args only in the untimed probe pass)
import { writeFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

const [inputPath, approachPath, n, iters, probeOut, ...hookPaths] = process.argv.slice(2);
const load = (path) => import(pathToFileURL(path).href);

const { makeInput } = await load(inputPath);
const { run } = await load(approachPath);
const count = Number(iters);
const data = makeInput(Number(n));

const hooks = probeOut ? await Promise.all(hookPaths.map(load)) : [];
const before = hooks.map((h) => h.snapshot());
let sink = 0;
for (let i = 0; i < count; i++) sink += run(data);
const after = hooks.map((h) => h.snapshot());

process.stdout.write(`${sink}\n`);
if (probeOut) {
  const lines = after.flatMap((end, i) =>
    Object.keys(end).map((k) => `${k}\t${end[k] - (before[i][k] ?? 0)}\n`),
  );
  writeFileSync(probeOut, lines.join(""));
}
```

## families/py/runner.py

```python
# Loop driver shared by every py-family runtime. Never reads a clock; the host times this process.
# argv: <input> <approach> <n> <iters> [<probe_out> <hook>...]  (probe args only in the untimed probe pass)
import importlib.util
import sys

sys.dont_write_bytecode = True


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main():
    input_path, approach_path, n, iters = sys.argv[1:5]
    probe_out, hook_paths = (sys.argv[5], sys.argv[6:]) if len(sys.argv) > 5 else (None, [])
    make_input = load(input_path, "_jmark_input").make_input
    run = load(approach_path, "_jmark_approach").run
    count = int(iters)
    data = make_input(int(n))

    hooks = [load(p, f"_jmark_hook_{i}") for i, p in enumerate(hook_paths)] if probe_out else []
    before = [h.snapshot() for h in hooks]
    sink = 0
    for _ in range(count):
        sink += run(data)
    after = [h.snapshot() for h in hooks]

    sys.stdout.write(f"{sink}\n")
    sys.stdout.flush()
    if probe_out:
        with open(probe_out, "w") as out:
            for start, end in zip(before, after):
                for k, v in end.items():
                    out.write(f"{k}\t{v - start.get(k, 0)}\n")


main()
```

## runtimes/bun.toml

```toml
family = "js"
needs = "bun"
run = "bun {runner} {input} {approach} {n} {iters} {probe}"
```

## runtimes/node.toml

```toml
family = "js"
needs = "node"
run = "node {runner} {input} {approach} {n} {iters} {probe}"
```

## runtimes/py.toml

```toml
family = "py"
needs = "python3"
run = "python3 {runner} {input} {approach} {n} {iters} {probe}"
```

