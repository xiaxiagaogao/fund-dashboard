<script lang="ts">
  import type { Allocation } from '$lib/api';
  import { fmtUSDT, fmtPct } from '$lib/format';

  export let alloc: Allocation | null = null;
  export let loading = false;

  const R = 70;
  const C = 2 * Math.PI * R;
  // 9-step teal→cyan ramp so the top 9 holdings each get a distinct shade.
  const TEAL = [
    'oklch(0.88 0.10 165)', 'oklch(0.83 0.115 169)', 'oklch(0.78 0.115 173)',
    'oklch(0.73 0.115 177)', 'oklch(0.68 0.11 181)', 'oklch(0.63 0.11 185)',
    'oklch(0.58 0.10 189)', 'oklch(0.53 0.10 193)', 'oklch(0.48 0.09 197)'
  ];
  const SHORT = 'oklch(0.70 0.155 24)';

  type Slice = { label?: string; side?: string; pct: number; color: string; dash: string; offset: string };

  // Gap of 4 user units between slices, rounded caps — straight from the design.
  function donut(items: { label?: string; side?: string; pct: number; color: string }[]): Slice[] {
    let acc = 0;
    return items.map((s) => {
      const len = Math.max(0, s.pct * C - 4);
      const o: Slice = { ...s, dash: `${len.toFixed(2)} ${(C - len).toFixed(2)}`, offset: (-acc * C).toFixed(2) };
      acc += s.pct;
      return o;
    });
  }

  $: capital = alloc
    ? donut([
        { pct: alloc.equity > 0 ? Math.max(0, alloc.margin_used) / alloc.equity : 0, color: 'oklch(0.80 0.115 168)' },
        { pct: alloc.equity > 0 ? Math.max(0, alloc.free_cash) / alloc.equity : 1, color: 'oklch(0.34 0.012 240)' }
      ])
    : [];

  // Notional by symbol: show up to 10 individually; beyond that, top 9 + 其他.
  $: notionalItems = (() => {
    const ps = alloc?.positions ?? [];
    let ti = 0;
    const colorFor = (side: string) => (side === 'SHORT' ? SHORT : TEAL[ti++ % TEAL.length]);
    if (ps.length <= 10) return ps.map((p) => ({ label: p.symbol.replace('USDT', ''), side: p.side, pct: p.pct, color: colorFor(p.side) }));
    const head = ps.slice(0, 9).map((p) => ({ label: p.symbol.replace('USDT', ''), side: p.side, pct: p.pct, color: colorFor(p.side) }));
    const tail = ps.slice(9);
    return [...head, { label: `其他 ${tail.length}`, side: '', pct: tail.reduce((s, p) => s + p.pct, 0), color: 'oklch(0.40 0.012 240)' }];
  })();
  $: notional = donut(notionalItems);

  $: isFlat = !!alloc && (alloc.positions?.length ?? 0) === 0;
</script>

<div class="card p-4 sm:p-5">
  <div class="text-[13px] font-bold mb-3.5">当前持仓</div>

  {#if loading && !alloc}
    <div class="py-10 text-center text-ink-500 text-sm">加载中…</div>
  {:else if !alloc}
    <div class="py-10 text-center text-ink-500 text-sm">持仓数据暂不可用</div>
  {:else}
    <div class="flex gap-2 justify-around mb-4">
      <!-- 资金配置 -->
      <div class="relative w-[130px] h-[130px] flex-none">
        <svg viewBox="0 0 180 180" class="w-[130px] h-[130px]" style="transform:rotate(-90deg)">
          <circle cx="90" cy="90" r={R} fill="none" stroke="oklch(0.24 0.008 240)" stroke-width="16" />
          {#each capital as s}
            <circle cx="90" cy="90" r={R} fill="none" stroke={s.color} stroke-width="16" stroke-linecap="round"
              stroke-dasharray={s.dash} stroke-dashoffset={s.offset} />
          {/each}
        </svg>
        <div class="absolute inset-0 flex flex-col items-center justify-center">
          <div class="text-[9px] text-ink-500 tracking-wider">杠杆</div>
          <div class="font-mono text-[22px] font-semibold leading-tight">{alloc.leverage.toFixed(2)}×</div>
          <div class="text-[9px] text-ink-300 font-mono">占 {alloc.equity > 0 ? fmtPct(alloc.margin_used / alloc.equity, 0) : '0%'}</div>
        </div>
      </div>
      <!-- 持仓分布(名义) -->
      <div class="relative w-[130px] h-[130px] flex-none">
        <svg viewBox="0 0 180 180" class="w-[130px] h-[130px]" style="transform:rotate(-90deg)">
          <circle cx="90" cy="90" r={R} fill="none" stroke="oklch(0.24 0.008 240)" stroke-width="16" />
          {#each notional as s}
            <circle cx="90" cy="90" r={R} fill="none" stroke={s.color} stroke-width="16" stroke-linecap="round"
              stroke-dasharray={s.dash} stroke-dashoffset={s.offset} />
          {/each}
        </svg>
        <div class="absolute inset-0 flex flex-col items-center justify-center">
          <div class="text-[9px] text-ink-500 tracking-wider">名义敞口</div>
          {#if isFlat}
            <div class="text-sm text-ink-400 mt-1">空仓</div>
          {:else}
            <div class="font-mono text-base font-semibold leading-tight">{fmtUSDT(alloc.notional, 0)}</div>
          {/if}
        </div>
      </div>
    </div>

    {#if !isFlat}
      <div class="flex flex-col gap-2">
        {#each notionalItems as l}
          <div class="flex items-center gap-2 text-xs">
            <span class="w-2 h-2 rounded-sm flex-none" style="background:{l.color}"></span>
            <span class="text-ink-300">{l.label}</span>
            {#if l.side}<span class="text-[9px] text-ink-500 border border-white/[0.08] rounded px-1">{l.side === 'LONG' ? '多' : '空'}</span>{/if}
            <span class="ml-auto font-mono text-ink-50">{fmtPct(l.pct, 1)}</span>
          </div>
        {/each}
      </div>
    {/if}

    <div class="mt-3.5 pt-3 border-t border-white/[0.06] text-[11px] text-ink-500 font-mono">
      净值 {fmtUSDT(alloc.equity, 0)} · 名义 {fmtUSDT(alloc.notional, 0)} USDT
    </div>
  {/if}
</div>
