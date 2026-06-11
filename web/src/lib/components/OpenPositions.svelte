<script lang="ts">
  import type { Position } from '$lib/api';
  import { fmtDate, fmtDuration, fmtUSDT, fmtSignedUSDT, fmtSignedPct } from '$lib/format';

  export let positions: Position[] = [];

  // Live "now" so age ticks without re-fetching the page.
  let now = Date.now();
  if (typeof window !== 'undefined') {
    setInterval(() => (now = Date.now()), 30_000);
  }

  $: notional = positions.reduce((s, p) => s + Math.abs((p.mark_price ?? p.entry_price) * p.quantity), 0);
  $: totalUnrealized = positions.reduce((s, p) => s + (p.unrealized_pnl ?? 0), 0);
</script>

<div class="card overflow-hidden">
  <div class="px-5 py-4 border-b border-ink-800/80 flex items-baseline justify-between gap-4">
    <div>
      <div class="label">当前持仓</div>
      <div class="stat-sub text-ink-400 mt-1">Binance 实时 mark price · 未实现 PnL</div>
    </div>
    <div class="text-right">
      <div class="font-mono tabular-nums text-ink-200 text-sm">{positions.length} 个 · 名义 {fmtUSDT(notional, 0)} USDT</div>
      <div class={'font-mono tabular-nums text-sm mt-0.5 ' + (totalUnrealized > 0 ? 'pos' : totalUnrealized < 0 ? 'neg' : 'text-ink-300')}>
        浮动 {fmtSignedUSDT(totalUnrealized, 2)}
      </div>
    </div>
  </div>
  {#if positions.length === 0}
    <div class="py-12 text-center text-ink-500 text-sm">手上没单</div>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-ink-400 text-xs uppercase tracking-wider">
          <tr>
            <th class="text-left py-2.5 px-5 font-medium">Symbol</th>
            <th class="text-left py-2.5 px-3 font-medium">方向</th>
            <th class="text-right py-2.5 px-3 font-medium">入场</th>
            <th class="text-right py-2.5 px-3 font-medium">Mark</th>
            <th class="text-right py-2.5 px-3 font-medium">价格变动</th>
            <th class="text-right py-2.5 px-3 font-medium">数量</th>
            <th class="text-right py-2.5 px-3 font-medium">未实现 PnL</th>
            <th class="text-right py-2.5 px-5 font-medium">已持仓</th>
          </tr>
        </thead>
        <tbody>
          {#each positions as p}
            {@const isLong = p.side === 'LONG' || p.side === 'BUY'}
            {@const mark = p.mark_price ?? p.entry_price}
            {@const pricePct = p.entry_price > 0 ? (mark - p.entry_price) / p.entry_price : 0}
            {@const effectivePct = isLong ? pricePct : -pricePct}
            <tr class="table-row-hover border-t border-ink-800/60">
              <td class="py-3 px-5 font-mono font-medium text-ink-50">{p.symbol.replace('USDT','')}</td>
              <td class="py-3 px-3">
                <span class={isLong ? 'pill-pos' : 'pill-neg'}>{isLong ? '多' : '空'}</span>
              </td>
              <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-200">{p.entry_price.toFixed(4)}</td>
              <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-100">{mark.toFixed(4)}</td>
              <td class={'py-3 px-3 text-right font-mono tabular-nums ' + (effectivePct > 0 ? 'pos' : effectivePct < 0 ? 'neg' : 'text-ink-300')}>
                {fmtSignedPct(effectivePct)}
              </td>
              <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-200">{p.quantity.toFixed(4)}</td>
              <td class={'py-3 px-3 text-right font-mono tabular-nums font-medium ' + ((p.unrealized_pnl ?? 0) > 0 ? 'pos' : (p.unrealized_pnl ?? 0) < 0 ? 'neg' : 'text-ink-300')}>
                {p.unrealized_pnl !== undefined ? fmtSignedUSDT(p.unrealized_pnl, 4) : '—'}
              </td>
              <td class="py-3 px-5 text-right">
                <div class="font-mono tabular-nums text-ink-100">{p.entry_time ? fmtDuration(now - p.entry_time) : '—'}</div>
                <div class="text-[11px] font-mono text-ink-500">{p.entry_time ? fmtDate(p.entry_time, true) : ''}</div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
