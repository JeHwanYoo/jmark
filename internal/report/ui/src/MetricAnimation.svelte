<script>
  import { onMount } from "svelte";
  import { animationKind, memorySamples, missIndices } from "./guide-model.js";
  let { id } = $props();
  const kind = $derived(animationKind(id));
  let progress = $state(0);
  let playing = $state(false);
  let reduced = $state(false);
  let frame;
  let last;
  const tick = (now) => {
    if (!playing) return;
    if (last != null) progress = Math.min(1, progress + (now - last) / 6000);
    last = now;
    if (progress >= 1) {
      playing = false;
      last = undefined;
    } else frame = requestAnimationFrame(tick);
  };
  function play() {
    if (progress >= 1) progress = 0;
    playing = true;
    last = undefined;
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(tick);
  }
  function pause() {
    playing = false;
    cancelAnimationFrame(frame);
    last = undefined;
  }
  function replay() {
    pause();
    progress = 0;
    play();
  }
  onMount(() => {
    const preference = matchMedia("(prefers-reduced-motion: reduce)");
    const changed = () => {
      reduced = preference.matches;
      if (reduced) {
        pause();
        progress = 1;
      }
    };
    changed();
    if (!reduced) play();
    const hidden = () => {
      if (document.hidden) pause();
    };
    preference.addEventListener("change", changed);
    document.addEventListener("visibilitychange", hidden);
    return () => {
      pause();
      preference.removeEventListener("change", changed);
      document.removeEventListener("visibilitychange", hidden);
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
      ? "경과 시간"
      : id === "cpu_time"
        ? "CPU 사용 시간"
        : id === "user_time"
          ? "User CPU"
          : "System CPU",
  );
</script>

<section class="metric-demo" aria-label="지표 설명 애니메이션">
  <p class="demo-caption">설명용 예시 · 실제 벤치마크 결과와는 별개입니다.</p>
  {#if kind === "time"}
    <div class="demo-reading">
      <span>{timeLabel}</span><strong
        >{Math.round(activeTime)} <small>ms</small></strong
      >
    </div>
    <div
      class="timeline"
      role="img"
      aria-label="100ms 중 user 실행 60ms, 커널 실행 15ms, 대기 25ms. CPU 사용 시간은 75ms입니다."
    >
      <div class="time-track">
        <span class="user" style:opacity={id === "system_time" ? 0.2 : 1}
        ></span>
        <span class="kernel" style:opacity={id === "user_time" ? 0.2 : 1}
        ></span>
        <span class="waiting" style:opacity={id === "wall_time" ? 1 : 0.2}
        ></span>
        <i style:left={`${progress * 100}%`}></i>
      </div>
      <div class="time-labels">
        <span>User<br />60 ms</span><span>System<br />15 ms</span><span
          >Wait<br />25 ms</span
        >
      </div>
      <div class="time-endpoints">
        <span>시작 · 0 ms</span><span>완료 · 100 ms</span>
      </div>
    </div>
    <p class="demo-note">
      {id === "wall_time"
        ? "실행 중인 시간과 기다리는 시간을 모두 셉니다."
        : "이 예시는 한 스레드가 순서대로 실행됩니다. CPU 시간은 대기를 제외한 실행 시간입니다."}
    </p>
  {:else if kind === "memory"}
    <div class="demo-reading">
      <span>관측한 최대 점유</span><strong>{peak} <small>MiB</small></strong>
    </div>
    <div class="memory-axis">
      <span>메모리 점유 · MiB</span><span>120</span>
    </div>
    <svg
      class="memory-plot"
      viewBox="0 0 360 145"
      role="img"
      aria-label="메모리 점유가 오르내리다가 120MiB에 도달합니다. 이후 점유가 줄어도 peak는 120MiB로 남습니다."
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
      <span>0 · 실행 시작</span><span>실행 완료</span>
    </div>
    <p class="demo-note">
      파란 선은 점유 변화, 점선은 지금까지 관측한 최대값입니다. 점유가 줄어도
      peak는 내려가지 않습니다.
    </p>
  {:else if kind === "throughput"}
    <div class="demo-reading">
      <span>이 예시의 처리량</span><strong>10 <small>op/s</small></strong>
    </div>
    <div
      class="work-grid"
      role="img"
      aria-label="1초 동안 작업 10개가 완료됩니다. 작업 하나당 100ms라면 처리량은 10op/s입니다."
    >
      {#each Array(10) as _, i}<span class:done={i < steps}
          >{i < steps ? "✓" : "·"}</span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>완료 {steps} / 10 op</span><span>{progress.toFixed(1)} / 1.0 s</span
      >
    </div>
    <p class="demo-note">
      1 op에 0.1초가 걸리면 1초에 10 op를 처리합니다. 작업 하나는 전체 입력을
      처리하는 호출 한 번입니다.
    </p>
  {:else if kind === "cpu"}
    <div class="demo-reading">
      <span
        >{id === "ipc"
          ? "이 예시의 IPC"
          : id === "instructions"
            ? "완료한 명령어"
            : "소비한 cycle"}</span
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
      aria-label="cycle 10개에 명령어 20개를 완료합니다. cycle 하나당 명령어 두 개이므로 IPC는 2입니다."
    >
      {#each Array(10) as _, i}<span class:done={i < steps}
          ><b>··</b><b>│</b></span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>명령어 {steps * 2}개</span><span>cycle {steps}개</span>
    </div>
    <p class="demo-note">
      각 칸은 cycle 하나와 완료한 명령어 두 개를 나타냅니다. 같은 명령어 수도
      소비한 cycle 수에 따라 IPC가 달라집니다.
    </p>
  {:else if kind === "branch"}
    <div class="demo-reading">
      <span>분기 예측 실패</span><strong
        >{[3, 8].filter((i) => i < steps).length} <small>misses</small></strong
      >
    </div>
    <div
      class="work-grid"
      role="img"
      aria-label="분기 10개 중 두 번의 예측이 실제 실행 방향과 다릅니다."
    >
      {#each Array(10) as _, i}<span
          class:done={i < steps}
          class:miss={i < steps && [3, 8].includes(i)}
          >{i >= steps ? "·" : [3, 8].includes(i) ? "×" : "✓"}</span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>✓ 예측 일치</span><span>× 예측 실패</span>
    </div>
    <p class="demo-note">
      실제 분기 방향이 예측과 다르면 miss를 셉니다. jmark에는 op당 실패 횟수를
      저장합니다.
    </p>
  {:else}
    <div class="demo-reading">
      <span
        >{id === "cache_miss_rate"
          ? "이 예시의 miss 비율"
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
      aria-label="캐시 접근 20회 중 4회가 miss입니다. 전체 예시의 miss 비율은 20%입니다."
    >
      {#each Array(20) as _, i}<span
          class:done={i < accesses}
          class:miss={i < accesses && missIndices.includes(i)}
          >{i >= accesses ? "·" : missIndices.includes(i) ? "×" : "✓"}</span
        >{/each}
    </div>
    <div class="time-endpoints">
      <span>접근 {accesses}회 · miss {misses}회</span><span>✓ hit · × miss</span
      >
    </div>
    <p class="demo-note">
      각 칸은 캐시 접근 한 번입니다. 전체 20회 중 miss가 4회이므로 비율은 4 / 20
      = 20%입니다.
    </p>
  {/if}
  <div class="demo-controls">
    <button type="button" onclick={() => (playing ? pause() : play())}
      >{playing ? "일시 정지" : "재생"}</button
    >
    <button type="button" onclick={replay}>다시 보기</button>
    <label
      >재생 위치<input
        type="range"
        min="0"
        max="1"
        step="0.01"
        bind:value={progress}
        oninput={pause}
      /></label
    >
  </div>
  <p class="motion-status" aria-live="polite">
    {reduced
      ? "동작 줄이기 설정이 켜져 있어 정지 화면으로 시작합니다."
      : playing
        ? "예시 재생 중"
        : progress === 1
          ? "예시 재생 완료"
          : "예시 일시 정지"}
  </p>
</section>

<style>
  .metric-demo {
    margin-top: 24px;
    border-block: 1px solid #e0e6ee;
    padding: 24px 0 16px;
  }
  .demo-caption,
  .demo-note,
  .motion-status {
    color: #65748b;
    font-size: 13px;
    line-height: 1.7;
  }
  .demo-caption {
    margin: 0 0 22px;
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
  }
  .work-grid .done {
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
    margin: 22px 0;
  }
  .demo-controls {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }
  .demo-controls label {
    flex: 1;
    min-width: 120px;
  }
  .demo-controls input {
    width: 100%;
    accent-color: #2563eb;
  }
  .motion-status {
    font-size: 12px;
    margin-bottom: 0;
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
