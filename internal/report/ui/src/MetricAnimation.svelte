<script>
  import { onMount } from "svelte";
  import { animationKind, memorySamples, missIndices } from "./guide-model.js";
  let { id } = $props();
  const kind = $derived(animationKind(id));
  const RUN = 5000;
  const HOLD = 1500;
  const FADE = 400;
  let progress = $state(0);
  let fading = $state(false);
  onMount(() => {
    const preference = matchMedia("(prefers-reduced-motion: reduce)");
    let frame;
    let start;
    const loop = (now) => {
      start ??= now;
      const elapsed = (now - start) % (RUN + HOLD);
      progress = Math.min(1, elapsed / RUN);
      fading = elapsed > RUN + HOLD - FADE;
      frame = requestAnimationFrame(loop);
    };
    const sync = () => {
      cancelAnimationFrame(frame);
      start = undefined;
      fading = false;
      if (preference.matches) progress = 1;
      else if (!document.hidden) frame = requestAnimationFrame(loop);
    };
    sync();
    preference.addEventListener("change", sync);
    document.addEventListener("visibilitychange", sync);
    return () => {
      cancelAnimationFrame(frame);
      preference.removeEventListener("change", sync);
      document.removeEventListener("visibilitychange", sync);
    };
  });
  const steps = $derived(Math.floor(progress * 10));
  const accesses = $derived(Math.floor(progress * 20));
  const misses = $derived(missIndices.filter((i) => i < accesses).length);
  const memoryIndex = $derived(Math.min(9, Math.floor(progress * 9)));
  const peak = $derived(Math.max(...memorySamples.slice(0, memoryIndex + 1)));
  const activeTime = $derived(
    id === "wall_time"
      ? progress * 100
      : id === "user_time"
        ? Math.min(progress * 100, 60)
        : id === "system_time"
          ? Math.min(15, Math.max(0, progress * 100 - 60))
          : Math.min(progress * 100, 75),
  );
  const timeLabel = $derived(
    id === "wall_time"
      ? "Elapsed time"
      : id === "cpu_time"
        ? "CPU time"
        : id === "user_time"
          ? "User CPU"
          : "System CPU",
  );
</script>

<section class="metric-demo" aria-label="Metric illustration">
  <p class="demo-caption">Illustrative example. These are not benchmark results.</p>
  <div class="demo-stage" class:fading>
  {#if kind === "time"}
    <div class="demo-reading">
      <span>{timeLabel}</span><strong
        >{Math.round(activeTime)} <small>ms</small></strong
      >
    </div>
    <div
      class="timeline"
      role="img"
      aria-label="Of 100 ms, 60 ms is user time, 15 ms is system time, and 25 ms is waiting. CPU time is 75 ms."
    >
      <div class="time-track">
        <span class="user" style:opacity={id === "system_time" ? 0.2 : 1}
        ></span>
        <span class="kernel" style:opacity={id === "user_time" ? 0.2 : 1}
        ></span>
        <span class="waiting" style:opacity={id === "wall_time" ? 1 : 0.2}
        ></span>
        <em style:left={`${progress * 100}%`}></em>
        <i style:left={`${progress * 100}%`}></i>
      </div>
      <div class="time-labels">
        <span>User<br />60 ms</span><span>System<br />15 ms</span><span
          >Wait<br />25 ms</span
        >
      </div>
      <div class="time-endpoints">
        <span>Start · 0 ms</span><span>End · 100 ms</span>
      </div>
    </div>
    <p class="demo-note">
      {id === "wall_time"
        ? "Wall time includes execution and waiting."
        : "This example runs one thread sequentially. CPU time excludes waiting."}
    </p>
  {:else if kind === "memory"}
    <div class="demo-reading">
      <span>Peak observed usage</span><strong>{peak} <small>MiB</small></strong>
    </div>
    <div class="memory-axis">
      <span>Memory usage · MiB</span><span>120</span>
    </div>
    <svg
      class="memory-plot"
      viewBox="0 0 360 145"
      role="img"
      aria-label="Memory usage rises and falls, reaching 120 MiB. The peak remains 120 MiB as usage falls."
    >
      <defs
        ><clipPath id="memory-reveal"
          ><rect x="0" y="0" width={360 * progress} height="145" /></clipPath
        ></defs
      >
      <line x1="0" x2="360" y1="140" y2="140" stroke="#d8e0ea" />
      <polyline
        points={memorySamples.map((v, i) => `${i * 40},${140 - v}`).join(" ")}
        fill="none"
        stroke="#dce3ee"
        stroke-width="3"
      />
      <polyline
        clip-path="url(#memory-reveal)"
        points={memorySamples.map((v, i) => `${i * 40},${140 - v}`).join(" ")}
        fill="none"
        stroke="#2563eb"
        stroke-width="3"
      />
      <line
        x1="0"
        x2="360"
        y1={140 - peak}
        y2={140 - peak}
        stroke="#b16a08"
        stroke-dasharray="5 5"
        stroke-width="2"
      />
    </svg>
    <div class="time-endpoints">
      <span>0 · Start</span><span>End</span>
    </div>
    <p class="demo-note">
      The blue line shows usage; the dotted line shows the highest value observed so far.
    </p>
  {:else if kind === "throughput"}
    <div class="demo-reading">
      <span>Example throughput</span><strong>10 <small>op/s</small></strong>
    </div>
    <div
      class="work-grid"
      role="img"
      aria-label="Ten operations complete in one second. At 100 ms per operation, throughput is 10 op/s."
    >
      {#each Array(10) as _, i}<span class:done={i < steps}
          >{i < steps ? "✓" : "·"}</span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>Completed {steps} / 10 op</span><span>{progress.toFixed(1)} / 1.0 s</span
      >
    </div>
    <p class="demo-note">
      At 0.1 seconds per operation, throughput is 10 op/s. One operation is one call over the full input.
    </p>
  {:else if kind === "cpu"}
    <div class="demo-reading">
      <span
        >{id === "ipc"
          ? "Example IPC"
          : id === "instructions"
            ? "Instructions completed"
            : "Cycles consumed"}</span
      ><strong
        >{id === "ipc" ? "2" : id === "instructions" ? steps * 2 : steps}<small
          >{id === "ipc"
            ? " instructions/cycle"
            : id === "instructions"
              ? " instructions"
              : " cycles"}</small
        ></strong
      >
    </div>
    <div
      class="work-grid cpu-grid"
      role="img"
      aria-label="Twenty instructions complete in ten cycles, giving an IPC of 2."
    >
      {#each Array(10) as _, i}<span class:done={i < steps}
          ><b>··</b><b>│</b></span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>{steps * 2} instructions</span><span>{steps} cycles</span>
    </div>
    <p class="demo-note">
      Each cell represents one cycle and two completed instructions. IPC depends on both counts.
    </p>
  {:else if kind === "branch"}
    <div class="demo-reading">
      <span>Branch misses</span><strong
        >{[3, 8].filter((i) => i < steps).length} <small>misses</small></strong
      >
    </div>
    <div
      class="work-grid"
      role="img"
      aria-label="Two out of ten branch predictions are wrong."
    >
      {#each Array(10) as _, i}<span
          class:done={i < steps}
          class:miss={i < steps && [3, 8].includes(i)}
          >{i >= steps ? "·" : [3, 8].includes(i) ? "×" : "✓"}</span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>✓ Correct</span><span>× Miss</span>
    </div>
    <p class="demo-note">
      A miss occurs when the predicted branch direction is wrong. jmark stores misses per operation.
    </p>
  {:else}
    <div class="demo-reading">
      <span
        >{id === "cache_miss_rate"
          ? "Example miss rate"
          : id === "cache_misses"
            ? "Cache misses"
            : "Cache references"}</span
      ><strong
        >{id === "cache_miss_rate"
          ? "20"
          : id === "cache_misses"
            ? misses
            : accesses}<small>{id === "cache_miss_rate" ? "%" : " count"}</small
        ></strong
      >
    </div>
    <div
      class="work-grid cache-grid"
      role="img"
      aria-label="Four of twenty cache accesses miss, for a miss rate of 20%."
    >
      {#each Array(20) as _, i}<span
          class:done={i < accesses}
          class:miss={i < accesses && missIndices.includes(i)}
          >{i >= accesses ? "·" : missIndices.includes(i) ? "×" : "✓"}</span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>{accesses} accesses · {misses} misses</span><span>✓ hit · × miss</span
      >
    </div>
    <p class="demo-note">
      Each cell is one cache access. Four misses out of twenty accesses give a rate of 20%.
    </p>
  {/if}
  </div>
</section>

<style>
  .metric-demo {
    margin-top: 24px;
    border-block: 1px solid #e0e6ee;
    padding: 24px 0 16px;
  }
  .demo-caption,
  .demo-note {
    color: #65748b;
    font-size: 13px;
    line-height: 1.7;
  }
  .demo-caption {
    margin: 0 0 22px;
  }
  .demo-stage {
    transition: opacity 0.4s ease;
  }
  .demo-stage.fading {
    opacity: 0;
  }
  .demo-reading {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 12px;
    margin-bottom: 28px;
    font-size: 14px;
  }
  .demo-reading strong {
    font-size: 32px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
  small {
    font: 12px var(--font-code);
    color: #65748b;
  }
  .time-track {
    display: grid;
    grid-template-columns: 60fr 15fr 25fr;
    position: relative;
    height: 32px;
  }
  .user {
    background: #2563eb;
  }
  .kernel {
    background: #0d9488;
  }
  .waiting {
    background: repeating-linear-gradient(
      45deg,
      #c8d0dc,
      #c8d0dc 4px,
      #e8ecf2 4px,
      #e8ecf2 8px
    );
  }
  .time-track em {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    background: rgb(255 255 255 / 0.7);
  }
  .time-track i {
    position: absolute;
    height: 46px;
    top: -7px;
    width: 2px;
    background: #172033;
  }
  .time-labels {
    display: grid;
    grid-template-columns: 60fr 15fr 25fr;
    margin-top: 14px;
    font-size: 12px;
    line-height: 1.7;
  }
  .time-endpoints,
  .memory-axis {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    color: #65748b;
    font-size: 12px;
    margin: 16px 0;
  }
  .work-grid {
    display: grid;
    grid-template-columns: repeat(10, minmax(0, 1fr));
    gap: 6px;
  }
  .work-grid span {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 42px;
    background: #edf0f4;
    color: #7b8797;
    border: 1px solid #dce3ee;
    border-radius: 4px;
    transition:
      background-color 0.25s ease,
      border-color 0.25s ease,
      color 0.25s ease;
  }
  .work-grid .done {
    animation: settle 0.3s ease-out;
    background: #e8f1ff;
    border-color: #2563eb;
    color: #2563eb;
  }
  .work-grid .miss {
    background: #fff2df;
    border-color: #b16a08;
    color: #9b5800;
    border-style: dashed;
  }
  .cpu-grid span {
    flex-direction: column;
    line-height: 1.1;
  }
  .cpu-grid b {
    font-weight: 500;
  }
  .memory-plot {
    width: 100%;
    height: auto;
    display: block;
  }
  .demo-note {
    min-height: 44px;
    margin: 22px 0 0;
  }
  @keyframes settle {
    from {
      transform: scale(0.85);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .demo-stage,
    .work-grid span {
      transition: none;
      animation: none;
    }
  }
  @media (max-width: 600px) {
    .demo-reading strong {
      font-size: 26px;
    }
    .work-grid {
      gap: 4px;
    }
    .work-grid span {
      min-height: 32px;
    }
    .time-labels {
      grid-template-columns: 50fr 25fr 25fr;
    }
  }
</style>
