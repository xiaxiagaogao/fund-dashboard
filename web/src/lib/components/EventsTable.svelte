<script lang="ts">
  import { fmtUSDT, fmtShares, fmtDate } from '$lib/format';
  import type { CashEvent } from '$lib/api';

  export let events: CashEvent[] = [];
</script>

<div class="card overflow-hidden">
  <div class="px-5 py-4 border-b border-ink-800/80 flex items-center justify-between">
    <div>
      <div class="label">我的入金 / 赎回</div>
      <div class="stat-sub text-ink-400 mt-1">按事件时间升序 · 不可修改</div>
    </div>
    <a href="/api/me/export.csv" class="btn-ghost text-xs px-3 py-1.5">导出 CSV</a>
  </div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead class="text-ink-400 text-xs uppercase tracking-wider">
        <tr>
          <th class="text-left py-2.5 px-5 font-medium">时间</th>
          <th class="text-left py-2.5 px-3 font-medium">类型</th>
          <th class="text-right py-2.5 px-3 font-medium">金额 (USDT)</th>
          <th class="text-right py-2.5 px-3 font-medium">事件 NAV</th>
          <th class="text-right py-2.5 px-3 font-medium">份额变动</th>
          <th class="text-right py-2.5 px-5 font-medium">累计份额</th>
        </tr>
      </thead>
      <tbody>
        {#each events as e}
          <tr class="table-row-hover border-t border-ink-800/60">
            <td class="py-3 px-5 font-mono text-ink-300">{fmtDate(e.occurred_at, true)}</td>
            <td class="py-3 px-3">
              <span class={e.type === 'deposit' ? 'pill-pos' : 'pill-neg'}>
                {e.type === 'deposit' ? '入金' : '赎回'}
              </span>
            </td>
            <td class="py-3 px-3 text-right font-mono tabular-nums">{fmtUSDT(e.amount_usdt, 4)}</td>
            <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-300">{e.nav_at_event.toFixed(6)}</td>
            <td class={'py-3 px-3 text-right font-mono tabular-nums ' + (e.shares_delta > 0 ? 'pos' : 'neg')}>
              {(e.shares_delta > 0 ? '+' : '') + fmtShares(e.shares_delta, 4)}
            </td>
            <td class="py-3 px-5 text-right font-mono tabular-nums text-ink-50">{fmtShares(e.shares_after, 4)}</td>
          </tr>
        {:else}
          <tr>
            <td colspan="6" class="py-8 text-center text-ink-500 text-sm">还没有入金记录</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
