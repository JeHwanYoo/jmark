# Benchmark strategy (AGENTS.md)

Each project keeps its strategy in `<bench-project-id>/AGENTS.md`.

## Rules

- Max **200 characters** (Unicode length, spaces included).
- Language-agnostic: state *what* to compare and *how to measure*, not runtime-specific APIs.
- One short paragraph. No headings, lists, or code.

## What to include

1. **Subject** — which implementations / approaches are under test
2. **Metric** — e.g. wall time per op, throughput, memory
3. **Input shape** — size and distribution that matter for fairness
4. **Fairness note** (optional) — same work, same data, same warmup/iters intent

## What to omit

- Full setup steps, dependency pins, or CLI flags (those live in `bench.toml`, `runtimes/`, or just recipes)
- Marketing claims or predicted winners
- Language-specific jargon that blocks porting the same idea elsewhere

## Sources

Each project keeps pure logic per family in `<family>/`: `_input.<ext>` builds the input from `n`, and each other file is one approach exporting `run(input)` → number. No comments, clocks, warmup/iter protocol, probes, or JSON in those files (algorithm loops inside `run` are fine). The host times a shared runner from outside and builds the result matching `schema/result.schema.json`. Workload params live in `bench.toml`. Alloc/GC metrics come from opt-in `probes` (read-only hooks in `families/<f>/probes/`), never from approach code. Every family must define the same approaches. New stacks go in `runtimes/` + `families/`.

## Example

```text
Compare map/filter/reduce vs a single for-loop for sum(x*2 where x*2>10). Metric: µs/op. Input: 1e5 ints in 1..20. Same data, warmup, and iteration count for every language port.
```

(Character count must stay ≤ 200 in the project file.)

Agent verify recipes live in `.cursor/skills/verify-jmark/`. UX and host changes belong in `cmd/jmark` (packages under `internal/`).
