<script lang="ts">
  import type { Stats } from "$lib/api";
  import { fmtPct, fmtSignedUSDT, fmtDuration, pnlClass } from "$lib/format";
  export let stats: Stats | null = null;
  export let window = 0;
</script>

<div class="metric-strip">
  <div class="metric">
    <div class="metric-label">胜率</div>
    <div class="metric-number">{stats ? fmtPct(stats.win_rate, 1) : "—"}</div>
    <div class="metric-note">
      {stats
        ? `${stats.wins} 胜 / ${stats.losses} 负 / ${stats.total} 笔`
        : "数据暂不可用"}
    </div>
  </div>
  <div class="metric">
    <div class="metric-label">已实现净收益 · USDT</div>
    <div class={"metric-number " + pnlClass(stats?.total_pnl ?? 0)}>
      {stats ? fmtSignedUSDT(stats.total_pnl) : "—"}
    </div>
    <div class="metric-note">
      扣除手续费 · 最近 {Math.min(stats?.total ?? 0, window)} 笔
    </div>
  </div>
  <div class="metric">
    <div class="metric-label">平均盈利 / 平均亏损</div>
    <div class="metric-number">
      {stats && stats.win_loss_ratio > 0
        ? stats.win_loss_ratio.toFixed(2)
        : "—"}
    </div>
    <div class="metric-note">
      <span class="number"
        >{stats ? fmtSignedUSDT(stats.avg_win_usdt, 0) : "—"}</span
      >
      /
      <span class="number"
        >{stats ? fmtSignedUSDT(stats.avg_loss_usdt, 0) : "—"}</span
      >
    </div>
  </div>
  <div class="metric">
    <div class="metric-label">平均持仓时长</div>
    <div class="metric-number">
      {stats ? fmtDuration(stats.avg_hold_hours * 3600_000) : "—"}
    </div>
    <div class="metric-note">
      中位 {stats ? fmtDuration(stats.median_hold_hours * 3600_000) : "—"}
    </div>
  </div>
</div>
