<script>
  import Brand from "./Brand.svelte";
  import RepositoryLink from "./RepositoryLink.svelte";
  let { runs } = $props();
  let query = $state("");
  const filtered = $derived(
    runs.filter((run) =>
      `${run.project} ${run.run} ${run.runtimes.join(" ")}`
        .toLowerCase()
        .includes(query.trim().toLowerCase()),
    ),
  );
</script>

<svelte:head><title>jmark · Results</title></svelte:head>
<header>
  <Brand />
  <RepositoryLink />
</header>
<main>
  <div class="heading">
    <h1>Results</h1>
    <span class="count"
      >{filtered.length} {filtered.length === 1 ? "run" : "runs"}</span
    >
  </div>
  {#if runs.length}
    <label class="index-search"
      >Search<input
        type="search"
        placeholder="Project, runtime, or run"
        bind:value={query}
      /></label
    >
    <section class="panel results">
      <div class="table-wrap">
        <table>
          <thead
            ><tr
              ><th>Project</th><th>Run</th><th>Runtimes</th><th>Captured</th
              ></tr
            ></thead
          >
          <tbody
            >{#each filtered as run}<tr>
                <td><a class="result-link" href={run.href}>{run.project}</a></td
                >
                <td><a class="run-link" href={run.href}>{run.run}</a></td>
                <td>{run.runtimes.join(" · ")}</td><td
                  >{run.captured_at || "—"}</td
                >
              </tr>{/each}</tbody
          >
        </table>
      </div>
    </section>
    {#if !filtered.length}<p class="empty">No matching results.</p>{/if}
  {:else}<p class="empty">No public results.</p>{/if}
</main>
