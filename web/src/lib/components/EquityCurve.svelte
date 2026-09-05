<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { despike } from "$lib/format";
  import type { EquityPoint, IndexPoint } from "$lib/api";
  import { RANGES, type RangeKey } from "$lib/ranges";
  import TimeSeriesChart from "./TimeSeriesChart.svelte";
  export let points: EquityPoint[] = [];
  export let qqq: IndexPoint[] = [];
  export let spy: IndexPoint[] = [];
  export let height = 235;
  export let range: RangeKey = "30d";
  export let loading = false;
  const dispatch = createEventDispatcher<{ range: RangeKey }>();
  function interp(pts: IndexPoint[], t: number): number {
    if (!pts.length) return NaN;
    if (t <= pts[0].t) return pts[0].close;
    if (t >= pts[pts.length - 1].t) return pts[pts.length - 1].close;
    let a = 0,
      b = pts.length - 1;
    while (b - a > 1) {
      const m = Math.floor((a + b) / 2);
      if (pts[m].t < t) a = m;
      else b = m;
    }
    return (
      pts[a].close +
      ((t - pts[a].t) / (pts[b].t - pts[a].t || 1)) *
        (pts[b].close - pts[a].close)
    );
  }
  $: nav = despike(points.map((p) => p.nav));
  $: times = points.map((p) => p.taken_at);
  $: series = (() => {
    if (points.length < 2 || nav[0] <= 0) return [];
    const rows = [
      {
        label: "XG fund",
        color: "var(--pos)",
        values: nav.map((v) => v / nav[0] - 1),
      },
    ];
    for (const benchmark of [
      { pts: qqq, label: "纳指 100", color: "var(--benchmark-a)" },
      { pts: spy, label: "标普 500", color: "var(--benchmark-b)" },
    ]) {
      const base = interp(benchmark.pts, times[0]);
      if (base > 0)
        rows.push({
          label: benchmark.label,
          color: benchmark.color,
          values: times.map((t) => interp(benchmark.pts, t) / base - 1),
        });
    }
    return rows;
  })();
</script>

<section class="section" aria-label="收益走势">
  <div class="section-head">
    <h2>收益走势</h2>
    <div class="range-control" role="group" aria-label="收益时间范围">
      {#each RANGES as r}<button
          class={r.key === range ? "seg-on" : "seg-off"}
          aria-pressed={r.key === range}
          disabled={loading}
          on:click={() => dispatch("range", r.key)}>{r.label}</button
        >{/each}
    </div>
  </div>
  <TimeSeriesChart
    {times}
    {series}
    {height}
    {loading}
    area
    label="收益走势，按左右方向键查看历史快照"
  />
  <div class="muted-note mt-4">
    区间收益 · NAV 归一化{#if qqq.length === 0 || spy.length === 0}<span
        class="ml-3">部分基准数据暂不可用</span
      >{/if}
  </div>
</section>

<style>
  .range-control {
    display: flex;
    flex-wrap: wrap;
    gap: 2px;
  }
  .range-control :global(button) {
    font-size: 11px;
    padding-inline: 9px;
  }
</style>
