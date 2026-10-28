# Metrics

jmark schema version 2 stores independent metric definitions and results. IDs do
not encode units. Add a metric definition in `internal/benchmark` and its reduction
in `internal/result`; a new data source implements `measure.Collector`. Reports
consume the persisted definition without a metric-specific component.

## Built-in definitions

| ID | Stored unit | Calculation | Direction |
| --- | --- | --- | --- |
| wall_time | s/op | Differential median / iterations | lower |
| cpu_time | s/op | Differential total CPU median / iterations | lower |
| user_time, system_time | s/op | Differential CPU-mode median / iterations | lower |
| peak_rss | B | Median workload process peak | lower |
| throughput | op/s | Inverse wall_time | higher |
| instructions, cpu_cycles | instructions/op, cycles/op | Differential counter median / iterations | lower |
| cache_references | references/op | Differential counter median / iterations | neutral |
| cache_misses, branch_misses | misses/op | Differential counter median / iterations | lower |
| ipc | instructions/cycle | Differential instructions / differential cycles | neutral |
| cache_miss_rate | ratio | Differential misses / differential references | lower |

One operation calls an approach once over the entire input. An input size `n`
is not implicitly a processed-items or processed-bytes declaration. Hardware
counter scope is platform-specific and separates comparisons between operating
systems. Linux counters exclude kernel execution and include descendants; macOS
cycles/instructions include kernel execution and exclude descendants. Generic
cache events can vary between CPUs; environment metadata remains essential.

Peak RSS includes startup, input generation and runtime memory. Hyperfine reports
the largest per-process peak including waited-for children, not simultaneous
process-tree memory. Linux very small processes may inherit a peak from the
measurement parent. RSS is neither total allocated bytes nor retained heap.
See [hyperfine's definitions](https://github.com/sharkdp/hyperfine/blob/v2.0.0/README.md#choosing-metrics-and-units).

Missing readings produce `status: unavailable` with a reason. Nonpositive time
estimates and negative counter estimates are below the noise floor; valid zero
counters remain zero. Ratio denominators must be positive. Source process errors
and malformed data fail the run. Selecting only RSS does not require a positive
wall-time difference.

Raw baseline and workload samples retain separate count, mean, median, sample
standard deviation, min, max and CV (stddev/mean). Standard deviation is null for
one sample; CV is null when undefined. These distributions are process executions,
not individual operations. They do not imply operation p95/p99 or a confidence
interval for a difference of medians.

## Probe extension contract

Allocation and GC probes remain opt-in, read-only family hooks, separate from the
time pass. A hook exports `describe()` returning a map from counter name to
`{unit, direction}`, and `snapshot()` returning cumulative numeric counters.
Allowed directions are `lower`, `higher`, and `neutral`; units describe raw counters
(e.g. `B`, `count`, `s`). Runners emit four tab-separated fields per counter:
`hook.counter`, raw unit, direction, snapshot delta. The host namespaces metric IDs
as `probe.hook.counter`, subtracts baseline/workload medians and normalizes per op.
Descriptors must stay identical across samples. A null-workload bias check remains
required for runtime probe claims. Gauge/heap-retention measurements should use a
collector with explicitly different aggregation semantics.

## Research and priorities

| Source | Relevant capabilities | jmark decision |
| --- | --- | --- |
| [hyperfine 2](https://github.com/sharkdp/hyperfine/releases/tag/v2.0.0) | CPU time, RSS, instructions, cycles, cache/branch misses, raw samples | Default collector; portable metrics by default, diagnostic outputs selected explicitly |
| [Google Benchmark](https://github.com/google/benchmark/blob/main/docs/user_guide.md) | Time, CPU, throughput, custom counters, memory manager, repeated-run statistics | Generic metric descriptors and phase statistics |
| [Go testing](https://pkg.go.dev/testing#BenchmarkResult) | ns/op, B/op, allocs/op, custom metrics | Allocation bytes/count are next-priority hooks |
| [JMH profilers](https://github.com/openjdk/jmh/blob/master/jmh-samples/src/main/java/org/openjdk/jmh/samples/JMHSample_35_Profilers.java) | Allocation rate and normalized allocation, GC count/time | GC hooks are next-priority, runtime-specific work |
| [BenchmarkDotNet](https://benchmarkdotnet.org/articles/configs/diagnosers.html) | Allocation, GC, threading, contention and hardware diagnostics | Keep runtime-dependent semantics explicit |
| [Criterion.rs](https://bheisler.github.io/criterion.rs/book/user_guide/command_line_output.html) | Bytes/elements throughput, bootstrap confidence intervals | Preserve samples for future statistical analysis |
| [pyperf](https://pyperf.readthedocs.io/en/stable/index.html) | Distribution, instability checks, significance analysis, memory tracking | Distinguish measurement quality from benchmark quantities |

Next: allocation B/op, allocs/op and GC counts/time where read-only counters exist.
Heap snapshot differences must not be relabeled as total allocations. Later work:
bootstrap intervals, declared items/bytes throughput, per-operation latency sampling,
contention, I/O and energy collectors. These require explicit work definitions or
new measurement methods and are not inferred from the current batch runner.
