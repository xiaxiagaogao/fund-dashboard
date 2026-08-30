<script lang="ts">
  import type { Position } from '$lib/api';
  import { fmtDuration, fmtSignedUSDT, fmtUSDT } from '$lib/format';

  export let positions: Position[] = [];
  /** How many rows are visible before the block scrolls internally. */
  export let maxRows = 10;

  const ROW_PX = 60; // measured row height; 10 rows + a peek of the next

  // Net is what actually reached the balance: Binance books realized PnL gross,
  // fees are charged separately and run large relative to these trade sizes.
  // Showing gross made losing trades look like wins.
  $: totalFees = positions.reduce((s, p) => s + (p.commission ?? 0), 0);
  $: totalNet = positions.reduce((s, p) => s + p.realized_pnl - (p.commission ?? 0), 0);
  $: scroll = positions.length > maxRows;

  const net = (p: { realized_pnl: number; commission?: number }) =>
    p.realized_pnl - (p.commission ?? 0);

  function fmtPx(v: number): string {
    const dp = v < 10 ? 3 : v < 1000 ? 1 : 0;
    return v.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp });
  }
</script>

<div class="card p-4 sm:p-5">
  <div class="flex items-baseline justify-between mb-2.5">
    <div class="text-[13px] font-bold">平仓记录</div>
    <div class="text-[10px] text-ink-500 font-mono">
      {positions.length} 笔 · 净 {fmtSignedUSDT(totalNet, 2)} · 费 {fmtUSDT(totalFees, 2)}
    </div>
  </div>

  {#if positions.length === 0}
    <div class="py-10 text-center text-ink-500 text-sm">还没有已平仓的交易</div>
  {:else}
    <div class={'flex flex-col ' + (scroll ? 'overflow-y-auto -mr-1.5 pr-1.5' : '')}
      style={scroll ? `max-height:${maxRows * ROW_PX}px` : ''}>
      {#each positions as p}
        {@const isLong = p.side === 'LONG' || p.side === 'BUY'}
        <div class="flex items-center gap-2.5 py-2.5 border-b border-white/[0.04] last:border-0">
          <div class="min-w-0">
            <div class="flex items-center gap-1.5">
              <span class="text-[13px] font-bold">{p.symbol.replace('USDT', '')}</span>
              <span class={'text-[9px] rounded px-1 border ' + (isLong ? 'text-accent-400 border-accent-500/30' : 'text-loss-400 border-loss-500/30')}>
                {isLong ? '多' : '空'}
              </span>
            </div>
            <div class="text-[10px] text-ink-500 font-mono mt-1 whitespace-nowrap">
              {fmtPx(p.entry_price)} → {fmtPx(p.exit_price ?? 0)}
            </div>
          </div>
          <div class="ml-auto text-right">
            <div class={'font-mono text-[13px] font-semibold ' + (net(p) >= 0 ? 'pos' : 'neg')}>
              {fmtSignedUSDT(net(p), 2)}
            </div>
            <div class="text-[10px] text-ink-500 font-mono mt-1 whitespace-nowrap">
              {fmtDuration((p.exit_time ?? 0) - p.entry_time)}
              {#if p.commission}· 费 {fmtUSDT(p.commission, 2)}{/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
