<script>
  import { tick } from "svelte";
  let { id, label, options, value = $bindable(""), wide = false } = $props();
  let open = $state(false);
  let query = $state("");
  let active = $state(0);
  let root;
  let input;
  const selected = $derived(options.find((option) => option.value === value));
  const matches = $derived(
    options.filter((option) =>
      option.label
        .toLocaleLowerCase()
        .includes(query.trim().toLocaleLowerCase()),
    ),
  );
  const activeId = $derived(
    open && matches[active] ? `${id}-option-${active}` : undefined,
  );

  function close() {
    open = false;
    query = "";
  }
  function show() {
    if (open) return;
    query = "";
    active = Math.max(
      0,
      options.findIndex((option) => option.value === value),
    );
    open = true;
    reveal();
  }
  async function reveal() {
    await tick();
    root
      ?.querySelector('[data-active="true"]')
      ?.scrollIntoView({ block: "nearest" });
  }
  function choose(option) {
    value = option.value;
    input.focus();
    close();
  }
  function keydown(event) {
    if (event.isComposing) return;
    if (event.key === "Escape") {
      if (open) event.preventDefault();
      close();
      return;
    }
    if (event.key === "Tab") {
      close();
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      if (!open) show();
      else if (matches[active]) choose(matches[active]);
      return;
    }
    if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
      if (!open && ["Home", "End"].includes(event.key)) return;
      event.preventDefault();
      if (!open) {
        show();
        return;
      }
      if (event.key === "Home") active = 0;
      else if (event.key === "End") active = Math.max(0, matches.length - 1);
      else
        active = Math.max(
          0,
          Math.min(
            matches.length - 1,
            active + (event.key === "ArrowDown" ? 1 : -1),
          ),
        );
      reveal();
    }
  }
</script>

<svelte:window
  onpointerdown={(event) => {
    if (root && !root.contains(event.target)) close();
  }}
/>
<div
  class="search-select"
  class:wide
  bind:this={root}
  onfocusout={(event) => {
    if (!root.contains(event.relatedTarget)) close();
  }}
>
  <label for={id}>{label}</label>
  <div class="control" class:expanded={open}>
    <input
      {id}
      bind:this={input}
      role="combobox"
      aria-autocomplete="list"
      aria-expanded={open}
      aria-controls={`${id}-list`}
      aria-activedescendant={activeId}
      autocomplete="off"
      spellcheck="false"
      value={open ? query : (selected?.label ?? "")}
      placeholder={open ? `Search ${label.toLowerCase()}…` : "Select…"}
      onfocus={show}
      onclick={show}
      onkeydown={keydown}
      oninput={(event) => {
        query = event.currentTarget.value;
        open = true;
        active = 0;
      }}
    />
    <button
      type="button"
      class="toggle"
      tabindex="-1"
      aria-label={`Toggle ${label.toLowerCase()} options`}
      aria-expanded={open}
      aria-controls={`${id}-list`}
      onpointerdown={(event) => event.preventDefault()}
      onclick={() => {
        if (open) close();
        else {
          input.focus();
          show();
        }
      }}
    >
      <svg
        width="16"
        height="16"
        viewBox="0 0 16 16"
        fill="none"
        aria-hidden="true"
        ><path
          d="m4 6 4 4 4-4"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        /></svg
      >
    </button>
  </div>
  {#if open}
    <div class="popover">
      <div id={`${id}-list`} role="listbox" aria-label={label} class="options">
        {#each matches as option, index (option.value)}
          <button
            type="button"
            id={`${id}-option-${index}`}
            role="option"
            aria-selected={option.value === value}
            tabindex="-1"
            data-active={index === active}
            class="option"
            onpointerdown={(event) => event.preventDefault()}
            onclick={() => choose(option)}
          >
            <span>{option.label}</span>
            {#if option.value === value}<svg
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                aria-hidden="true"
                ><path
                  d="m3 8 3 3 7-7"
                  stroke="currentColor"
                  stroke-width="1.6"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                /></svg
              >{/if}
          </button>
        {/each}
      </div>
      {#if !matches.length}<p role="status" class="no-options">
          No matches
        </p>{/if}
    </div>
  {/if}
</div>

<style>
  .search-select {
    position: relative;
    width: 180px;
    max-width: 100%;
    min-width: 0;
  }
  .wide {
    width: 100%;
  }
  label {
    display: block;
    margin-bottom: 7px;
  }
  .control {
    display: flex;
    align-items: center;
    background: #fff;
    border: 1px solid #d8dfe9;
    border-radius: 7px;
    transition: border-color 120ms;
  }
  .control:hover {
    border-color: #a5b4cb;
  }
  .control:focus-within {
    border-color: #2563eb;
    box-shadow: 0 0 0 1px #2563eb;
  }
  input {
    width: 100%;
    min-width: 0;
    height: 38px;
    padding: 9px 0 9px 12px;
    border: 0;
    outline: none;
    border-radius: inherit;
    background: transparent;
    font: inherit;
    font-size: 12px;
    color: #25324a;
    text-overflow: ellipsis;
  }
  input::placeholder {
    color: #5f6c80;
  }
  .toggle {
    display: grid;
    place-items: center;
    padding: 0;
    width: 34px;
    height: 38px;
    flex: none;
    border: 0;
    background: transparent;
    color: #69768a;
  }
  .expanded .toggle svg {
    transform: rotate(180deg);
  }
  .popover {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    width: max(100%, 240px);
    max-width: min(100%, calc(100vw - 28px));
    z-index: 20;
    padding: 5px;
    background: #fff;
    border: 1px solid #dce3ee;
    border-radius: 9px;
    box-shadow:
      0 12px 32px #17203318,
      0 3px 8px #17203308;
  }
  .options {
    max-height: min(280px, 40vh);
    overflow-y: auto;
    overscroll-behavior: contain;
  }
  .option {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    width: 100%;
    min-height: 38px;
    padding: 9px 10px;
    border: 0;
    border-radius: 5px;
    text-align: left;
    background: transparent;
    line-height: 1.5;
  }
  .option span {
    overflow-wrap: anywhere;
    min-width: 0;
  }
  .option svg {
    flex: none;
    color: #2563eb;
  }
  .option[data-active="true"],
  .option:hover {
    background: #f0f5ff;
    color: #1e4cb5;
  }
  .option[aria-selected="true"] {
    font-weight: 600;
  }
  .no-options {
    padding: 12px 10px;
    margin: 0;
    font-size: 12px;
    color: #69768a;
  }
  @media (max-width: 600px) {
    .search-select {
      width: calc(50% - 6px);
    }
    .wide {
      width: 100%;
    }
    input {
      height: 44px;
      font-size: 16px;
    }
    .toggle {
      height: 44px;
    }
    .option {
      min-height: 44px;
    }
  }
</style>
