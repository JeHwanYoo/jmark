# Metrics

One operation in jmark is one call to `run(input)` over the full input.

Time and cumulative counters compare two executions with different iteration counts. `W` is the baseline count and `N` is the additional count. Each execution is measured several times:

`ΔX = median(X(W + N)) - median(X(W))`

### Collector readings

Every built-in metric is computed from raw readings reported by hyperfine. A metric is available when all of its readings are. Support follows hyperfine 2.0. Permissions or virtualization can still withhold a supported counter; `just doctor` reports what the current host collects.

| Reading | Used by | macOS | Linux | Windows |
| --- | --- | --- | --- | --- |
| `time_wall_clock` | `wall_time`, `throughput` | ✓ | ✓ | ✓ |
| `time_cpu` | `cpu_time` | ✓ | ✓ | ✓ |
| `time_user` | `user_time` | ✓ | ✓ | ✓ |
| `time_system` | `system_time` | ✓ | ✓ | ✓ |
| `memory_peak_resident` | `peak_rss` | ✓ | ✓ | — |
| `cpu_cycles` | `cpu_cycles`, `ipc` | ✓ | ✓ | — |
| `instructions` | `instructions`, `ipc` | ✓ | ✓ | — |
| `cache_references` | `cache_references`, `cache_miss_rate` | — | ✓ | — |
| `cache_misses` | `cache_misses`, `cache_miss_rate` | — | ✓ | — |
| `branch_misses` | `branch_misses` | — | ✓ | — |

Linux hardware counters need a 64-bit x86_64, AArch64, RISC-V, POWER, or LoongArch build.

## Wall time (`wall_time`)

Elapsed time until the operation completes, including CPU time and waiting.

Measurement: `Δwall / N`. Stored in `s/op` and displayed in `µs/op`. Lower is faster.

## CPU time (`cpu_time`)

The sum of CPU time used by the process's threads. It can exceed wall time when threads run on multiple cores.

Measurement: `Δcpu / N`. User and system time are added for each execution before calculating the difference between medians. Unit: `s/op`.

## User CPU time (`user_time`)

CPU time spent running application and library code.

Measurement: `Δuser / N`. Unit: `s/op`.

## System CPU time (`system_time`)

CPU time spent running kernel code on behalf of the process, including system operations such as reading files.

Measurement: `Δsystem / N`. Unit: `s/op`.

## Peak RSS (`peak_rss`)

The highest physical memory usage during execution, including runtime startup and input creation.

Measurement: `median(peak RSS(W + N))`. Stored in `B` and displayed in `MiB`. The baseline is not subtracted and the value is not divided by iterations. This is different from total allocated memory.

## Throughput (`throughput`)

The number of operations completed per second. Higher values mean more operations in the same time.

Measurement: `1 / wall_time`. Unit: `op/s`. jmark calculates this as the reciprocal of wall time per operation. Input size `n` is not interpreted as items per second.

## Instructions (`instructions`)

The number of instructions completed by the CPU per operation.

Measurement: `Δinstructions / N`. Unit: `instructions/op`. Availability depends on the CPU, operating system, and profiling permissions.

## CPU cycles (`cpu_cycles`)

The number of CPU cycles consumed per operation.

Measurement: `Δcycles / N`. Unit: `cycles/op`. The relationship to elapsed time varies with CPU frequency and execution environment.

## IPC (`ipc`)

The number of instructions completed per CPU cycle.

Measurement: `Δinstructions / Δcycles`. Unit: `instructions/cycle`. Higher IPC alone does not mean different code is faster.

## Cache references (`cache_references`)

The number of cache accesses counted by the CPU's generic cache event. The accesses counted vary by CPU.

Measurement: `Δcache_references / N`. Unit: `references/op`. Neither higher nor lower counts are inherently better.

## Cache misses (`cache_misses`)

The number of cache misses counted by the CPU's generic cache event.

Measurement: `Δcache_misses / N`. Unit: `misses/op`. Event definitions can vary by CPU, so compare results from the same environment.

## Cache miss rate (`cache_miss_rate`)

The share of counted cache accesses that missed.

Measurement: `Δcache_misses / Δcache_references`. Stored as a `ratio` and displayed as `%`. Calculated when both counters are valid and references are positive.

## Branch misses (`branch_misses`)

The number of branch directions the CPU predicted incorrectly.

Measurement: `Δbranch_misses / N`. Unit: `misses/op`. jmark does not calculate a rate against total branches.

## Reading results

A time difference of zero or less, or a negative counter difference, is treated as below measurement resolution. Requested metrics that cannot be computed remain `unavailable`: `unsupported_on_platform` when the operating system cannot report a reading, and `not_reported_by_collector` when a supported reading was withheld. Valid zero counter values are displayed.

Raw samples and statistics describe process executions. They are not per-operation p95 or p99 latencies. Peak RSS is the largest process peak, including waited-for child processes; it is not the sum of memory used simultaneously by the process tree.

Hardware counter scope varies by platform. On Linux, counters exclude kernel execution and include child processes. On macOS, cycles and instructions include kernel execution and exclude child processes.

[Probe and statistics details](docs/metrics-implementation.md) describe the extension protocol and storage. The [hyperfine metric definitions](https://github.com/sharkdp/hyperfine/blob/v2.0.0/README.md#choosing-metrics-and-units) describe platform limitations.
