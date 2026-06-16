<script lang="ts">
  import type { Allocation } from '$lib/api';
  import { fmtUSDT, fmtPct } from '$lib/format';

  export let alloc: Allocation | null = null;
  export let loading = false;

  // Categorical palette for the notional donut — distinct on the dark theme.
  const PALETTE = ['#a3e635', '#60a5fa', '#f59e0b', '#22d3ee', '#c084fc', '#fb7185'];
  const OTHER = '#71717a';

  type Seg = { label: string; value: number; color: string; pct: number; dash: string; offset: number };

  // segments turns weighted items into stroke-dasharray donut segments.
  // Circle r=15.915 → circumference≈100, so each segment's dash length is its
  // percentage. offset=25 starts the first segment at 12 o'clock.
  function segments(items: { label: string; value: number; color: string }[]): Seg[] {
    const total = items.reduce((s, it) => s + it.value, 0);
    let acc = 0;
    return items.map((it) => {
      const pct = total > 0 ? (it.value / total) * 100 : 0;
      const seg: Seg = { ...it, pct, dash: `${pct.toFixed(2)} ${(100 - pct).toFixed(2)}`, offset: 25 - acc };
      acc += pct;
      return seg;
    });
  }

  // Capital: margin posted vs idle cash.
  $: capitalItems = alloc
    ? [
        { label: '保证金占用', value: Math.max(alloc.margin_used, 0), color: '#a3e635' },
        { label: '闲置现金', value: Math.max(alloc.free_cash, 0), color: '#3f3f46' }
      ]
    : [];
  $: capitalSegs = segments(capitalItems);

  // Notional by symbol, tail grouped into 其他 beyond 6 slices.
  $: notionalItems = (() => {
    const ps = alloc?.positions ?? [];
    if (ps.length <= 6) return ps.map((p, i) => ({ label: p.symbol, value: p.notional, color: PALETTE[i % PALETTE.length], side: p.side, pct: p.pct }));
    const head = ps.slice(0, 5).map((p, i) => ({ label: p.symbol, value: p.notional, color: PALETTE[i], side: p.side, pct: p.pct }));
    const tail = ps.slice(5);
    const tailVal = tail.reduce((s, p) => s + p.notional, 0);
    const tailPct = tail.reduce((s, p) => s + p.pct, 0);
    return [...head, { label: `其他 ${tail.length}`, value: tailVal, color: OTHER, side: '', pct: tailPct }];
  })();
  $: notionalSegs = segments(notionalItems);

  $: isFlat = !!alloc && (alloc.positions?.length ?? 0) === 0;
</script>

<div class="card p-5">
  <div class="label mb-4">当前持仓</div>

  {#if loading && !alloc}
    <div class="py-10 text-center text-ink-500 text-sm">加载中…</div>
  {:else if !alloc}
    <div class="py-10 text-center text-ink-500 text-sm">持仓数据暂不可用</div>
  {:else}
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
      <!-- 资金配置 -->
      <div class="flex flex-col items-center">
        <div class="text-[11px] text-ink-400 mb-3">资金配置</div>
        <div class="relative w-[128px] h-[128px]">
          <svg viewBox="0 0 42 42" class="w-full h-full -rotate-0">
            <circle cx="21" cy="21" r="15.915" fill="none" stroke="#27272a" stroke-width="4.5" />
            {#each capitalSegs as s}
              <circle cx="21" cy="21" r="15.915" fill="none" stroke={s.color} stroke-width="4.5"
                stroke-dasharray={s.dash} stroke-dashoffset={s.offset} />
            {/each}
          </svg>
          <div class="absolute inset-0 flex flex-col items-center justify-center">
            <span class="font-mono text-2xl text-ink-50 leading-none">{alloc.leverage.toFixed(2)}x</span>
            <span class="text-[10px] text-ink-500 mt-1">全仓杠杆</span>
          </div>
        </div>
        <div class="mt-3 space-y-1.5">
          {#each capitalSegs as s}
            <div class="flex items-center gap-2 text-xs text-ink-300">
              <span class="w-2.5 h-2.5 rounded-sm" style="background:{s.color}"></span>
              <span>{s.label}</span>
              <span class="font-mono text-ink-400 ml-auto pl-3">{fmtPct(s.pct / 100, 0)}</span>
            </div>
          {/each}
        </div>
      </div>

      <!-- 持仓分布(名义) -->
      <div class="flex flex-col items-center">
        <div class="text-[11px] text-ink-400 mb-3">持仓分布 · 名义</div>
        <div class="relative w-[128px] h-[128px]">
          <svg viewBox="0 0 42 42" class="w-full h-full">
            <circle cx="21" cy="21" r="15.915" fill="none" stroke="#27272a" stroke-width="4.5" />
            {#each notionalSegs as s}
              <circle cx="21" cy="21" r="15.915" fill="none" stroke={s.color} stroke-width="4.5"
                stroke-dasharray={s.dash} stroke-dashoffset={s.offset} />
            {/each}
          </svg>
          <div class="absolute inset-0 flex flex-col items-center justify-center">
            {#if isFlat}
              <span class="text-xs text-ink-400">空仓</span>
            {:else}
              <span class="font-mono text-xl text-ink-50 leading-none">{alloc.positions.length}</span>
              <span class="text-[10px] text-ink-500 mt-1">个持仓</span>
            {/if}
          </div>
        </div>
        {#if !isFlat}
          <div class="mt-3 space-y-1.5 w-full max-w-[180px]">
            {#each notionalItems as it}
              <div class="flex items-center gap-2 text-xs text-ink-300">
                <span class="w-2.5 h-2.5 rounded-sm shrink-0" style="background:{it.color}"></span>
                <span class="font-mono truncate">{it.label}</span>
                {#if it.side}<span class={'text-[10px] ' + (it.side === 'LONG' ? 'pos' : 'neg')}>{it.side === 'LONG' ? '多' : '空'}</span>{/if}
                <span class="font-mono text-ink-400 ml-auto pl-2">{fmtPct(it.pct, 0)}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <div class="mt-4 pt-3 border-t border-ink-800/60 flex justify-between text-[11px] text-ink-500 font-mono">
      <span>净值 {fmtUSDT(alloc.equity, 0)} · 名义 {fmtUSDT(alloc.notional, 0)} USDT</span>
    </div>
  {/if}
</div>
