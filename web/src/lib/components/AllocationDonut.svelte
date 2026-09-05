<script context="module" lang="ts">
  export type DonutItem = {
    id: string;
    label: string;
    value: number;
    color: string;
  };
</script>

<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { arc, pie, type PieArcDatum } from "d3-shape";
  export let items: DonutItem[] = [];
  export let label: string;
  export let centerLabel: string;
  export let centerValue: string;
  export let centerDetail = "";
  export let activeId: string | null = null;
  export let selectedId: string | null = null;
  const dispatch = createEventDispatcher<{
    inspect: string | null;
    select: string;
  }>();
  const layout = pie<DonutItem>()
    .sort(null)
    .value((item) => Math.max(0, item.value))
    .padAngle(0.025);
  const shape = arc<PieArcDatum<DonutItem>>()
    .innerRadius(59)
    .outerRadius(79)
    .cornerRadius(2);
  $: slices = layout(
    items.filter((item) => Number.isFinite(item.value) && item.value > 0),
  );
  function selectKey(event: KeyboardEvent, id: string) {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      dispatch("select", id);
    }
  }
</script>

<div class="donut">
  <svg viewBox="0 0 180 180" role="group" aria-label={label}>
    <circle
      cx="90"
      cy="90"
      r="69"
      fill="none"
      stroke="var(--panel2)"
      stroke-width="20"
    />
    <g transform="translate(90 90)">
      {#each slices as slice (slice.data.id)}
        <path
          d={shape(slice) ?? ""}
          fill={slice.data.color}
          class:muted={activeId !== null && activeId !== slice.data.id}
          class:active={activeId === slice.data.id}
          role="button"
          tabindex="0"
          aria-label={slice.data.label}
          aria-pressed={selectedId === slice.data.id}
          on:pointerenter={() => dispatch("inspect", slice.data.id)}
          on:pointerleave={() => dispatch("inspect", null)}
          on:focus={() => dispatch("inspect", slice.data.id)}
          on:blur={() => dispatch("inspect", null)}
          on:click={() => dispatch("select", slice.data.id)}
          on:keydown={(event) => selectKey(event, slice.data.id)}
        >
          <title>{slice.data.label}</title>
        </path>
      {/each}
    </g>
  </svg>
  <div class="donut-center" aria-live="polite">
    <span>{centerLabel}</span><strong class="number">{centerValue}</strong
    >{#if centerDetail}<small>{centerDetail}</small>{/if}
  </div>
</div>

<style>
  .donut {
    position: relative;
    width: 100%;
    max-width: 180px;
    aspect-ratio: 1;
    margin-inline: auto;
  }
  svg {
    display: block;
    width: 100%;
    height: 100%;
    overflow: visible;
  }
  path {
    cursor: pointer;
    transition:
      opacity 160ms ease,
      filter 160ms ease;
  }
  path.muted {
    opacity: 0.35;
  }
  path.active {
    filter: brightness(1.12);
  }
  path:focus-visible {
    outline: none;
    stroke: var(--hi);
    stroke-width: 2;
  }
  .donut-center {
    position: absolute;
    inset: 26% 18%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 5px;
    pointer-events: none;
    text-align: center;
  }
  .donut-center span {
    color: var(--mid);
    font-size: 10px;
    max-width: 100%;
    overflow-wrap: anywhere;
  }
  .donut-center strong {
    font-size: 17px;
    font-weight: 500;
    line-height: 1.2;
    max-width: 100%;
    overflow-wrap: anywhere;
  }
  .donut-center small {
    font-size: 9px;
    color: var(--lo);
  }
  @media (min-width: 901px) and (max-width: 1100px) {
    .donut-center strong {
      font-size: 15px;
    }
    .donut-center {
      gap: 3px;
    }
    .donut-center span {
      font-size: 9px;
    }
  }
</style>
