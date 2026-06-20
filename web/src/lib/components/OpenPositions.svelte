<script lang="ts">
  import type { Position } from '$lib/api';
  import { fmtDuration, fmtUSDT, fmtSignedUSDT, fmtSignedPct } from '$lib/format';

  export let positions: Position[] = [];

  // Live "now" so age ticks without re-fetching the page.
  let now = Date.now();
  if (typeof window !== 'undefined') {
    setInterval(() => (now = Date.now()), 30_000);
  }

  $: notional = positions.reduce((s, p) => s + Math.abs((p.mark_price ?? p.entry_price) * p.quantity), 0);
  $: totalUnrealized = positions.reduce((s, p) => s + (p.unrealized_pnl ?? 0), 0);

  function fmtPx(v: number): string {
    const dp = v < 10 ? 4 : v < 1000 ? 2 : 1;
    return v.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp });
  }
</script>

<div class="card p-4 sm:p-5">
  <div class="flex items-baseline justify-between gap-4 mb-3">
    <div class="text-[13px] font-bold">当前持仓明细</div>
    <div class="text-right">
      <div class="font-mono text-[11px] text-ink-400">{positions.length} 个 · 名义 {fmtUSDT(notional, 0)} USDT</div>
      <div class={'font-mono text-[11px] mt-0.5 ' + (totalUnrealized > 0 ? 'pos' : totalUnrealized < 0 ? 'neg' : 'text-ink-300')}>
        浮动 {fmtSignedUSDT(totalUnrealized, 2)}
      </div>
    </div>
  </div>

  {#if positions.length === 0}
    <div class="py-10 text-center text-ink-500 text-sm">手上没单</div>
  {:else}
    <div class="flex flex-col">
      {#each positions as p}
        {@const isLong = p.side === 'LONG' || p.side === 'BUY'}
        {@const mark = p.mark_price ?? p.entry_price}
        {@const pricePct = p.entry_price > 0 ? (mark - p.entry_price) / p.entry_price : 0}
        {@const effectivePct = isLong ? pricePct : -pricePct}
        <div class="flex items-center gap-2.5 py-2.5 border-b border-white/[0.04] last:border-0">
          <div class="min-w-0">
            <div class="flex items-center gap-1.5">
              <span class="text-[13px] font-bold">{p.symbol.replace('USDT', '')}</span>
              <span class={'text-[9px] rounded px-1 border ' + (isLong ? 'text-accent-400 border-accent-500/30' : 'text-loss-400 border-loss-500/30')}>
                {isLong ? '多' : '空'}
              </span>
            </div>
            <div class="text-[10px] text-ink-500 font-mono mt-1 whitespace-nowrap">
              {fmtPx(p.entry_price)} → {fmtPx(mark)} · {p.entry_time ? fmtDuration(now - p.entry_time) : '—'}
            </div>
          </div>
          <div class="ml-auto text-right">
            <div class={'font-mono text-[13px] font-semibold ' + ((p.unrealized_pnl ?? 0) >= 0 ? 'pos' : 'neg')}>
              {p.unrealized_pnl !== undefined ? fmtSignedUSDT(p.unrealized_pnl, 2) : '—'}
            </div>
            <div class={'text-[10px] font-mono mt-1 ' + (effectivePct >= 0 ? 'pos' : 'neg')}>{fmtSignedPct(effectivePct)}</div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
