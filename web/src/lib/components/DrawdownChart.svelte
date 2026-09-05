<script lang="ts">
  import type { EquityPoint } from "$lib/api";
  import { fmtPct, despike } from "$lib/format";
  import TimeSeriesChart from "./TimeSeriesChart.svelte";
  export let points: EquityPoint[] = [];
  export let height = 165;
  $: nav = despike(points.map((p) => p.nav));
  $: values = (() => {
    let peak = -Infinity;
    return nav.map((v) => {
      peak = Math.max(peak, v);
      return peak > 0 ? v / peak - 1 : 0;
    });
  })();
  $: maxDD = Math.min(0, ...values);
  $: currentDD = values.at(-1) ?? 0;
</script>

<section class="section" aria-label="水下回撤">
  <div class="section-head">
    <h2>水下回撤</h2>
    <div class="flex flex-wrap gap-5 text-xs text-ink-400">
      <span
        >最大 <span class="number neg ml-2"
          >{values.length >= 2 ? fmtPct(maxDD) : "—"}</span
        ></span
      ><span
        >当前 <span class="number neg ml-2"
          >{values.length >= 2 ? fmtPct(currentDD) : "—"}</span
        ></span
      >
    </div>
  </div>
  <TimeSeriesChart
    times={points.map((p) => p.taken_at)}
    series={[{ label: "回撤", color: "var(--neg)", values }]}
    {height}
    area
    domain={{ min: maxDD < 0 ? maxDD * 1.15 : -0.01, max: 0 }}
    label="历史回撤，按左右方向键查看"
  />
</section>
