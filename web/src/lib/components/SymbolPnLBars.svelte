<script lang="ts">
  import type { SymbolPnL } from '$lib/api';
  import { fmtSignedUSDT, fmtPct } from '$lib/format';

  export let rows: SymbolPnL[] = [];
  /** How many rows are visible before the block scrolls internally. */
  export let maxRows = 10;

  const ROW_PX = 44; // measured row height incl. gap; 10 rows + a peek of the next

  $: maxAbs = rows.reduce((m, r) => Math.max(m, Math.abs(r.total_pnl)), 0) || 1;
  $: scroll = rows.length > maxRows;
</script>

<div class="card p-4 sm:p-5">
  <div class="text-[13px] font-bold mb-3.5">按标的盈亏</div>
  {#if rows.length === 0}
    <div class="py-10 text-center text-ink-500 text-sm">没有已平仓数据</div>
  {:else}
    <div class={'flex flex-col gap-3 ' + (scroll ? 'overflow-y-auto -mr-1.5 pr-1.5' : '')}
      style={scroll ? `max-height:${maxRows * ROW_PX}px` : ''}>
      {#each rows as r}
        {@const isWin = r.total_pnl >= 0}
        <div>
          <div class="flex items-baseline justify-between mb-1.5">
            <div class="flex items-baseline gap-2">
              <span class="text-[13px] font-bold">{r.symbol.replace('USDT', '')}</span>
              <span class="text-[10px] text-ink-500 font-mono whitespace-nowrap">{r.trades} 笔 · 胜 {fmtPct(r.win_rate, 0)}</span>
            </div>
            <span class={'font-mono text-[13px] font-semibold ' + (isWin ? 'pos' : 'neg')}>{fmtSignedUSDT(r.total_pnl, 2)}</span>
          </div>
          <div class="h-1.5 rounded-full overflow-hidden" style="background:oklch(0.26 0.008 240)">
            <div class="h-full rounded-full" style="width:{((Math.abs(r.total_pnl) / maxAbs) * 100).toFixed(1)}%;background:{isWin ? 'oklch(0.78 0.115 168)' : 'oklch(0.70 0.155 24)'}"></div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
