<script lang="ts">
  import { fmtDate, fmtSignedPct } from "$lib/format";
  export let times: number[] = [];
  export let series: { label: string; color: string; values: number[] }[] = [];
  export let height = 240;
  export let label = "历史收益";
  export let loading = false;
  export let area = false;
  export let domain: { min: number; max: number } | null = null;
  let width = 640;
  let inspected: number | null = null;
  $: valid = times.length >= 2 && series.length > 0;
  $: index = Math.max(
    0,
    Math.min(inspected ?? times.length - 1, times.length - 1),
  );
  $: values = series.flatMap((s) => s.values).filter(Number.isFinite);
  $: low = Math.min(0, ...values);
  $: high = Math.max(0, ...values);
  $: padding = (high - low) * 0.12 || 0.01;
  $: min = domain?.min ?? low - padding;
  $: max = domain?.max ?? high + padding;
  $: plotWidth = Math.max(1, width - 56);
  $: plotHeight = height - 40;
  $: ticks = Array.from({ length: 5 }, (_, i) => max - ((max - min) * i) / 4);
  $: x = (t: number) =>
    4 +
    ((t - times[0]) / (times[times.length - 1] - times[0] || 1)) * plotWidth;
  $: y = (v: number) => 10 + ((max - v) / (max - min || 1)) * plotHeight;
  $: line = (s: (typeof series)[number]) =>
    s.values
      .map(
        (v, i) =>
          `${i ? "L" : "M"}${x(times[i]).toFixed(2)},${y(v).toFixed(2)}`,
      )
      .join(" ");
  $: readout = valid
    ? `${fmtDate(times[index], true)}，${series.map((s) => `${s.label} ${fmtSignedPct(s.values[index])}`).join("，")}`
    : "数据不足";
  function inspect(event: PointerEvent) {
    if (!valid) return;
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
    const target =
      times[0] +
      Math.max(0, Math.min(1, (event.clientX - rect.left - 4) / plotWidth)) *
        (times[times.length - 1] - times[0]);
    let left = 0,
      right = times.length - 1;
    while (left < right) {
      const middle = Math.floor((left + right) / 2);
      if (times[middle] < target) left = middle + 1;
      else right = middle;
    }
    inspected =
      left > 0 && target - times[left - 1] < times[left] - target
        ? left - 1
        : left;
  }
  function keydown(event: KeyboardEvent) {
    if (
      !valid ||
      !["ArrowLeft", "ArrowRight", "Home", "End", "Escape"].includes(event.key)
    )
      return;
    event.preventDefault();
    if (event.key === "Escape") inspected = null;
    else if (event.key === "Home") inspected = 0;
    else if (event.key === "End") inspected = times.length - 1;
    else
      inspected = Math.max(
        0,
        Math.min(
          times.length - 1,
          index + (event.key === "ArrowLeft" ? -1 : 1),
        ),
      );
  }
</script>

<div class="chart" aria-busy={loading}>
  <div class="legend">
    {#each series as s}<div class="legend-item">
        <div class="legend-label">
          <span style:background={s.color}></span>{s.label}
        </div>
        <div class="number legend-value" style:color={s.color}>
          {valid ? fmtSignedPct(s.values[index]) : "—"}
        </div>
      </div>{/each}
  </div>
  <div class="chart-inspection-time number">
    {valid ? fmtDate(times[index], true) : "暂无快照"}
  </div>
  {#if valid}
    <div
      class="plot"
      bind:clientWidth={width}
      style:opacity={loading ? "0.45" : "1"}
      tabindex="0"
      role="slider"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={times.length - 1}
      aria-valuenow={index}
      aria-valuetext={readout}
      on:keydown={keydown}
      on:pointerdown={inspect}
      on:pointermove={inspect}
      on:pointerleave={() => (inspected = null)}
      on:blur={() => (inspected = null)}
    >
      <svg
        width="100%"
        {height}
        viewBox={`0 0 ${width} ${height}`}
        role="img"
        aria-label={label}
      >
        {#each ticks as tick}<line
            x1="4"
            x2={plotWidth + 4}
            y1={y(tick)}
            y2={y(tick)}
            stroke="var(--line)"
            stroke-width=".7"
            stroke-dasharray="2 4"
          /><text
            x={width - 2}
            y={y(tick) + 3}
            text-anchor="end"
            fill="var(--lo)"
            font-size="10">{(tick * 100).toFixed(1)}%</text
          >{/each}
        {#each series as s, i}
          {#if area && i === 0}<path
              d={`${line(s)} L${x(times[times.length - 1])},${y(0)} L${x(times[0])},${y(0)} Z`}
              fill={s.color}
              opacity=".07"
            />{/if}
          <path
            d={line(s)}
            fill="none"
            stroke={s.color}
            stroke-width={i === 0 ? 2 : 1.4}
            stroke-linejoin="round"
          />
        {/each}
        {#if inspected !== null}<line
            x1={x(times[index])}
            x2={x(times[index])}
            y1="6"
            y2={height - 27}
            stroke="var(--mid)"
            stroke-width=".8"
            stroke-dasharray="3 4"
          />{/if}
        {#each series as s}<circle
            cx={x(times[index])}
            cy={y(s.values[index])}
            r={inspected === null ? 2.5 : 3.5}
            fill={s.color}
            stroke="var(--bg)"
            stroke-width="2"
          />{/each}
        {#each [0, Math.floor((times.length - 1) / 2), times.length - 1] as n, i}<text
            x={x(times[n])}
            y={height - 3}
            text-anchor={i === 0 ? "start" : i === 2 ? "end" : "middle"}
            fill="var(--lo)"
            font-size="10">{fmtDate(times[n]).slice(5)}</text
          >{/each}
      </svg>
    </div>
  {:else}<div class="empty-state" style:min-height={`${height}px`}>
      还没有足够的快照数据
    </div>{/if}
</div>

<style>
  .legend {
    display: flex;
    flex-wrap: wrap;
    gap: 12px 30px;
  }
  .legend-item {
    min-width: 84px;
  }
  .legend-label {
    color: var(--mid);
    font-size: 11px;
    display: flex;
    align-items: center;
    gap: 7px;
  }
  .legend-label span {
    width: 11px;
    height: 2px;
  }
  .legend-value {
    font-size: 19px;
    margin-top: 7px;
    font-weight: 500;
  }
  .chart-inspection-time {
    color: var(--lo);
    font-size: 10px;
    min-height: 32px;
    padding-top: 12px;
    margin-bottom: 8px;
  }
  .plot {
    touch-action: pan-y;
    transition: opacity 0.18s;
    border-radius: 3px;
  }
  svg {
    display: block;
    overflow: visible;
  }
  @media (max-width: 600px) {
    .legend {
      gap: 12px 20px;
    }
    .legend-item {
      min-width: 72px;
    }
    .legend-value {
      font-size: 17px;
    }
  }
</style>
