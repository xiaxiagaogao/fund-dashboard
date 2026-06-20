<script lang="ts">
  import type { DayPnL } from '$lib/api';
  import { fmtSignedUSDT } from '$lib/format';

  export let days: DayPnL[] = [];

  type Cell = { date: string; net: number; fills: number; has: boolean } | null;

  // Build Monday-started week columns spanning the first trade day → today.
  $: weeks = (() => {
    if (days.length === 0) return [] as Cell[][];
    const byDate = new Map(days.map((d) => [d.date, d]));
    const first = new Date(days[0].date + 'T00:00:00');
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    // Back up to Monday (getDay: 0=Sun..6=Sat → Monday offset).
    const start = new Date(first);
    const dow = (start.getDay() + 6) % 7; // 0=Mon
    start.setDate(start.getDate() - dow);

    const cols: Cell[][] = [];
    let col: Cell[] = [];
    const cur = new Date(start);
    while (cur <= today) {
      const y = cur.getFullYear();
      const m = String(cur.getMonth() + 1).padStart(2, '0');
      const d = String(cur.getDate()).padStart(2, '0');
      const key = `${y}-${m}-${d}`;
      const rec = byDate.get(key);
      const beforeFirst = cur < first;
      col.push(beforeFirst ? null : rec ? { date: key, net: rec.net, fills: rec.fills, has: true } : { date: key, net: 0, fills: 0, has: false });
      if (col.length === 7) {
        cols.push(col);
        col = [];
      }
      cur.setDate(cur.getDate() + 1);
    }
    if (col.length) {
      while (col.length < 7) col.push(null);
      cols.push(col);
    }
    return cols;
  })();

  $: maxAbs = Math.max(1e-9, ...days.map((d) => Math.abs(d.net)));

  function cellStyle(c: Cell): string {
    if (!c) return 'background:transparent';
    if (!c.has) return 'background:oklch(0.245 0.008 240)'; // in-range, no trades
    if (Math.abs(c.net) < 1e-9) return 'background:oklch(0.30 0.008 240)';
    const k = Math.min(1, Math.abs(c.net) / maxAbs);
    return c.net > 0
      ? `background:oklch(${(0.50 + 0.30 * k).toFixed(3)} ${(0.09 + 0.045 * k).toFixed(3)} 168)`
      : `background:oklch(${(0.48 + 0.18 * k).toFixed(3)} ${(0.10 + 0.05 * k).toFixed(3)} 24)`;
  }
  function cellTitle(c: Cell): string {
    if (!c || !c.has) return c ? `${c.date} · 无交易` : '';
    return `${c.date} · ${fmtSignedUSDT(c.net, 2)} USDT · ${c.fills} 笔`;
  }

  // Month label per column (show when the column's first real day starts a new month).
  $: monthLabels = weeks.map((col) => {
    const firstReal = col.find((c) => c !== null);
    if (!firstReal) return '';
    const d = new Date(firstReal.date + 'T00:00:00');
    return d.getDate() <= 7 ? `${d.getMonth() + 1}月` : '';
  });

  const WD = ['一', '', '三', '', '五', '', '日'];

  $: tradedDays = days.filter((d) => d.fills > 0).length;
  $: greenDays = days.filter((d) => d.net > 0).length;
  $: totalNet = days.reduce((s, d) => s + d.net, 0);
</script>

<div class="card p-5">
  <div class="flex items-start justify-between gap-4 mb-4">
    <div>
      <div class="label">每日盈亏日历</div>
      <div class="stat-sub text-ink-400 mt-1">{tradedDays} 个交易日 · {greenDays} 绿 · 净 {fmtSignedUSDT(totalNet, 0)} USDT</div>
    </div>
    <div class="flex items-center gap-1.5 text-[11px] text-ink-500">
      <span>亏</span>
      <span class="w-3 h-3 rounded-[3px]" style="background:oklch(0.62 0.16 24)"></span>
      <span class="w-3 h-3 rounded-[3px]" style="background:oklch(0.245 0.008 240)"></span>
      <span class="w-3 h-3 rounded-[3px]" style="background:oklch(0.74 0.12 168)"></span>
      <span>盈</span>
    </div>
  </div>

  {#if weeks.length === 0}
    <div class="py-8 text-center text-ink-500 text-sm">还没有成交记录</div>
  {:else}
    <div class="overflow-x-auto">
      <div class="inline-flex gap-[3px] pb-1 ml-7 mb-1">
        {#each monthLabels as ml}
          <div class="w-3 text-[10px] text-ink-500">{ml}</div>
        {/each}
      </div>
      <div class="flex gap-2">
        <div class="flex flex-col gap-[3px] text-[10px] text-ink-500 pr-1">
          {#each WD as w}<div class="h-3 leading-3">{w}</div>{/each}
        </div>
        <div class="inline-flex gap-[3px]">
          {#each weeks as col}
            <div class="flex flex-col gap-[3px]">
              {#each col as cell}
                <div class="w-3 h-3 rounded-[3px]" style={cellStyle(cell)} title={cellTitle(cell)}></div>
              {/each}
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>
