<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { despike, fmtPct, pnlClass } from "$lib/format";
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
        label: "我的收益",
        color: "var(--pos)",
        values: nav.map((v) => v / nav[0] - 1),
      },
    ];
    for (const benchmark of [
      { pts: qqq, label: "QQQ", color: "var(--benchmark-a)" },
      { pts: spy, label: "SPY", color: "var(--benchmark-b)" },
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
  function comparisonsAt(index: number) {
    const mine = series[0]?.values[index];
    return [
      { label: "QQQ", short: "QQQ" },
      { label: "SPY", short: "SPY" },
    ].map((benchmark) => {
      const value = series.find((row) => row.label === benchmark.label)?.values[
        index
      ];
      const delta = mine - (value ?? NaN);
      // Match the displayed precision so rounding cannot say "跑赢 0.00%".
      const rounded = Number((delta * 100).toFixed(2)) / 100;
      return { ...benchmark, delta: Number.isFinite(rounded) ? rounded : null };
    });
  }
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
    let:index
  >
    <div
      class="benchmark-comparison"
      role="group"
      aria-label="相对基准收益"
      aria-busy={loading}
    >
      {#each comparisonsAt(index) as comparison}
        <span
          class={comparison.delta === null
            ? "text-ink-400"
            : pnlClass(comparison.delta)}
          title={`${comparison.label}：所选区间收益率之差，单位为百分点`}
        >
          {#if comparison.delta === null}{comparison.short} 暂无数据
          {:else if comparison.delta === 0}与 {comparison.short} 持平
            <span class="number">0.00%</span>
          {:else}{comparison.delta > 0 ? "跑赢" : "跑输"}
            {comparison.short}
            <span class="number">{fmtPct(Math.abs(comparison.delta))}</span>
          {/if}
        </span>
      {/each}
    </div>
  </TimeSeriesChart>
</section>

<style>
  .benchmark-comparison {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 20px;
    margin-top: 16px;
    min-height: 20px;
    font-size: 12px;
  }
  .benchmark-comparison > span {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
  .benchmark-comparison[aria-busy="true"] {
    opacity: 0.45;
  }
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
