<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api, type FriendRow, type NofxFill, ApiError } from '$lib/api';
  import { fmtUSDT, fmtSignedUSDT, fmtDate, fmtShares } from '$lib/format';

  type LedgerRow = {
    id: number; username: string; name: string;
    type: 'deposit' | 'withdraw'; amount_usdt: number; occurred_at: number;
    nav_at_event: number; shares_delta: number;
    source: 'manual' | 'binance_transfer'; note: string;
  };

  let friends: FriendRow[] = [];
  let fills: NofxFill[] = [];
  let ledger: LedgerRow[] = [];
  let loading = true;
  let error = '';
  let notice = '';
  // Diagnostic: surface which step is currently in flight so a stuck loading
  // screen tells us *where* it stuck. Cheap to keep around; tiny.
  let loadStep = '初始化';

  // form: create friend
  let cf = { name: '', username: '', password: '', is_admin: false };
  let cfBusy = false;

  // form: cash event
  let ce = {
    username: '',
    type: 'deposit' as 'deposit' | 'withdraw',
    amount_usdt: 0,
    note: '',
    manual_nav: 0,
    occurred_at_ms: 0,
    skip_bootstrap_check: false
  };
  let ceBusy = false;
  let ceResult: { nav_at_event: number; shares_delta: number; equity_at_evt: number; manual_nav: boolean } | null = null;

  // form: snapshot
  let snapBusy = false;
  let snapResult: { taken_at: number; total_equity: number; nav: number } | null = null;

  async function load() {
    try {
      loadStep = '1/3 检查身份';
      const me = await api.me();
      if (!me.is_admin) {
        loadStep = '非 admin，跳转';
        goto('/');
        return;
      }
      loadStep = '2/3 加载朋友列表';
      friends = await api.admin.listFriends();
      loadStep = '3/3 加载入金流水';
      ledger = await api.admin.listCashEvents(200);
      // nofx fills 端点已经移除（404 OK），try/catch 静默吞下
      try {
        fills = await api.admin.nofxFills(20);
      } catch (e) {
        fills = [];
      }
      loadStep = '完成';
    } catch (e) {
      error = e instanceof Error ? e.message : '加载失败';
      // Also keep the step so we know which await threw.
      console.error('admin load failed at step:', loadStep, e);
    } finally {
      loading = false;
    }
  }

  async function submitCreateFriend(e: SubmitEvent) {
    e.preventDefault();
    cfBusy = true;
    notice = '';
    try {
      await api.admin.createFriend(cf);
      cf = { name: '', username: '', password: '', is_admin: false };
      friends = await api.admin.listFriends();
      notice = '已创建朋友账号';
    } catch (err) {
      notice = (err instanceof ApiError ? err.message : '创建失败');
    } finally {
      cfBusy = false;
    }
  }

  async function submitCashEvent(e: SubmitEvent) {
    e.preventDefault();
    ceBusy = true;
    notice = '';
    ceResult = null;
    try {
      const payload: any = {
        username: ce.username,
        type: ce.type,
        amount_usdt: Number(ce.amount_usdt),
        note: ce.note || undefined
      };
      if (ce.manual_nav > 0) payload.manual_nav = Number(ce.manual_nav);
      if (ce.occurred_at_ms > 0) payload.occurred_at_ms = ce.occurred_at_ms;
      if (ce.skip_bootstrap_check) payload.skip_bootstrap_check = true;

      ceResult = await api.admin.cashEvent(payload);
      notice = '入金/赎回已记录';
      // Refresh the ledger below so the new row shows up immediately.
      ledger = await api.admin.listCashEvents(200);
    } catch (err) {
      notice = err instanceof ApiError ? err.message : '记录失败';
    } finally {
      ceBusy = false;
    }
  }

  async function takeSnapshot() {
    snapBusy = true;
    notice = '';
    snapResult = null;
    try {
      snapResult = await api.admin.snapshot();
      notice = '已生成 snapshot';
    } catch (err) {
      notice = err instanceof ApiError ? err.message : '失败';
    } finally {
      snapBusy = false;
    }
  }

  onMount(load);
</script>

{#if loading}
  <div class="text-ink-400 text-sm flex items-center gap-2">
    <span class="inline-block w-1.5 h-1.5 rounded-full bg-accent-500 animate-pulse"></span>
    加载中… <span class="font-mono text-ink-500">[{loadStep}]</span>
  </div>
{:else if error}
  <div class="card p-6 text-loss-400">{error}</div>
{:else}
  <div class="mb-6 flex items-baseline justify-between">
    <h2 class="text-xl font-semibold tracking-tight">Admin</h2>
    {#if notice}
      <div class="text-sm text-ink-300">{notice}</div>
    {/if}
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-6">
    <!-- Cash event form -->
    <form on:submit={submitCashEvent} class="card p-5 space-y-3 lg:col-span-2">
      <div>
        <div class="label">记录入金 / 赎回</div>
        <div class="stat-sub text-ink-400 mt-1">默认走 Binance 自动算 NAV；手填 NAV 走人工 override</div>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <label class="block">
          <span class="label block mb-1.5">朋友账号</span>
          <select class="select" bind:value={ce.username} required>
            <option value="" disabled>选择…</option>
            {#each friends as f}
              <option value={f.username}>{f.name} · {f.username}{f.is_admin ? ' (admin)' : ''}</option>
            {/each}
          </select>
        </label>
        <label class="block">
          <span class="label block mb-1.5">类型</span>
          <select class="select" bind:value={ce.type}>
            <option value="deposit">入金 (deposit)</option>
            <option value="withdraw">赎回 (withdraw)</option>
          </select>
        </label>
        <label class="block">
          <span class="label block mb-1.5">金额 USDT</span>
          <input class="input" type="number" step="0.0001" min="0" bind:value={ce.amount_usdt} required />
        </label>
        <label class="block">
          <span class="label block mb-1.5">手填 NAV（可选）</span>
          <input class="input" type="number" step="0.000001" min="0" placeholder="0 = 走 Binance 自动" bind:value={ce.manual_nav} />
        </label>
        <label class="block sm:col-span-2">
          <span class="label block mb-1.5">备注（可选）</span>
          <input class="input font-sans" type="text" bind:value={ce.note} placeholder="如 bootstrap / 朋友 A 第二次入金" />
        </label>
      </div>
      <label class="flex items-center gap-2 text-xs text-ink-400">
        <input type="checkbox" bind:checked={ce.skip_bootstrap_check} />
        跳过 bootstrap 1% 偏离检查（首笔入金可用）
      </label>
      <div class="flex items-center justify-between pt-2">
        <button class="btn-primary" type="submit" disabled={ceBusy}>{ceBusy ? '记录中…' : '记录'}</button>
        {#if ceResult}
          <div class="text-xs font-mono text-ink-300">
            NAV={ceResult.nav_at_event.toFixed(6)} · Δshares={ceResult.shares_delta.toFixed(4)}
            {#if ceResult.manual_nav}· <span class="text-ink-400">manual</span>{/if}
          </div>
        {/if}
      </div>
    </form>

    <!-- Snapshot button + add friend -->
    <div class="space-y-4">
      <div class="card p-5">
        <div class="label">立即拍 NAV 快照</div>
        <p class="stat-sub text-ink-400 mt-1 mb-3">额外触发一次（hourly cron 之外）</p>
        <button class="btn-ghost w-full" on:click={takeSnapshot} disabled={snapBusy}>
          {snapBusy ? '拍摄中…' : '现在拍'}
        </button>
        {#if snapResult}
          <div class="mt-3 text-xs font-mono text-ink-300">
            <div>equity: {fmtUSDT(snapResult.total_equity, 2)} USDT</div>
            <div>NAV: {snapResult.nav.toFixed(6)}</div>
            <div>{fmtDate(snapResult.taken_at, true)}</div>
          </div>
        {/if}
      </div>

      <form on:submit={submitCreateFriend} class="card p-5 space-y-3">
        <div class="label">新建朋友账号</div>
        <input class="input font-sans" placeholder="显示名" bind:value={cf.name} required />
        <input class="input font-sans" placeholder="用户名" bind:value={cf.username} required />
        <input class="input font-sans" type="password" placeholder="初始密码 (≥8 字符)" bind:value={cf.password} required />
        <label class="flex items-center gap-2 text-xs text-ink-400">
          <input type="checkbox" bind:checked={cf.is_admin} /> 设为 admin
        </label>
        <button class="btn-primary w-full" type="submit" disabled={cfBusy}>{cfBusy ? '创建中…' : '创建'}</button>
      </form>
    </div>
  </div>

  <!-- Pool ledger — all friends' deposits/withdrawals, newest first -->
  <div class="card overflow-hidden mb-6">
    <div class="px-5 py-4 border-b border-ink-800/80 flex items-baseline justify-between">
      <div>
        <div class="label">全员入金 / 赎回流水</div>
        <div class="stat-sub text-ink-400 mt-1">所有成员的 cash event · 按事件时间倒序 · append-only</div>
      </div>
      <div class="text-xs text-ink-500">共 {ledger.length} 条</div>
    </div>
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-ink-400 text-xs uppercase tracking-wider">
          <tr>
            <th class="text-left py-2.5 px-5 font-medium">时间</th>
            <th class="text-left py-2.5 px-3 font-medium">成员</th>
            <th class="text-left py-2.5 px-3 font-medium">类型</th>
            <th class="text-right py-2.5 px-3 font-medium">金额 (USDT)</th>
            <th class="text-right py-2.5 px-3 font-medium">事件 NAV</th>
            <th class="text-right py-2.5 px-3 font-medium">份额变动</th>
            <th class="text-left py-2.5 px-3 font-medium">来源</th>
            <th class="text-left py-2.5 px-5 font-medium">备注</th>
          </tr>
        </thead>
        <tbody>
          {#each ledger as r}
            <tr class="table-row-hover border-t border-ink-800/60">
              <td class="py-3 px-5 font-mono text-ink-300 whitespace-nowrap">{fmtDate(r.occurred_at, true)}</td>
              <td class="py-3 px-3">
                <div class="text-ink-50">{r.name}</div>
                <div class="text-[11px] text-ink-500 font-mono">{r.username}</div>
              </td>
              <td class="py-3 px-3">
                <span class={r.type === 'deposit' ? 'pill-pos' : 'pill-neg'}>
                  {r.type === 'deposit' ? '入金' : '赎回'}
                </span>
              </td>
              <td class="py-3 px-3 text-right font-mono tabular-nums">{fmtUSDT(r.amount_usdt, 4)}</td>
              <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-300">{r.nav_at_event.toFixed(6)}</td>
              <td class={'py-3 px-3 text-right font-mono tabular-nums ' + (r.shares_delta > 0 ? 'pos' : 'neg')}>
                {(r.shares_delta > 0 ? '+' : '') + fmtShares(r.shares_delta, 4)}
              </td>
              <td class="py-3 px-3 text-xs text-ink-400">{r.source}</td>
              <td class="py-3 px-5 text-xs text-ink-300 max-w-[200px] truncate" title={r.note}>{r.note || '—'}</td>
            </tr>
          {:else}
            <tr><td colspan="8" class="py-8 text-center text-ink-500 text-sm">还没有任何 cash event</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>

  <!-- nofx fills -->
  <div class="card overflow-hidden mb-6">
    <div class="px-5 py-4 border-b border-ink-800/80 flex items-baseline justify-between">
      <div>
        <div class="label">nofx 最近成交</div>
        <div class="stat-sub text-ink-400 mt-1">同 Binance 账户的交易 fills · 区分 bot/manual · 朋友看不见</div>
      </div>
      {#if fills.length === 0}
        <div class="text-xs text-ink-500">未挂 nofx 数据库（NOFX_DB_PATH）</div>
      {/if}
    </div>
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-ink-400 text-xs uppercase tracking-wider">
          <tr>
            <th class="text-left py-2.5 px-5 font-medium">时间</th>
            <th class="text-left py-2.5 px-3 font-medium">来源</th>
            <th class="text-left py-2.5 px-3 font-medium">Symbol</th>
            <th class="text-left py-2.5 px-3 font-medium">动作</th>
            <th class="text-right py-2.5 px-3 font-medium">价</th>
            <th class="text-right py-2.5 px-3 font-medium">数量</th>
            <th class="text-right py-2.5 px-3 font-medium">名义 USDT</th>
            <th class="text-right py-2.5 px-5 font-medium">已实现 PnL</th>
          </tr>
        </thead>
        <tbody>
          {#each fills as f}
            <tr class="table-row-hover border-t border-ink-800/60">
              <td class="py-3 px-5 font-mono text-ink-300 whitespace-nowrap">{fmtDate(f.created_at, true)}</td>
              <td class="py-3 px-3"><span class={f.source === 'bot' ? 'pill-pos' : 'pill-neutral'}>{f.source}</span></td>
              <td class="py-3 px-3 font-mono text-ink-200">{f.symbol}</td>
              <td class="py-3 px-3">
                <span class="text-ink-300 text-xs">{f.side}</span>
                {#if f.order_action}<span class="text-ink-500 text-xs ml-1">· {f.order_action}</span>{/if}
              </td>
              <td class="py-3 px-3 text-right font-mono tabular-nums">{f.price.toFixed(4)}</td>
              <td class="py-3 px-3 text-right font-mono tabular-nums">{f.quantity.toFixed(4)}</td>
              <td class="py-3 px-3 text-right font-mono tabular-nums">{fmtUSDT(f.quote_quantity, 2)}</td>
              <td class={'py-3 px-5 text-right font-mono tabular-nums ' + (f.realized_pnl > 0 ? 'pos' : f.realized_pnl < 0 ? 'neg' : 'text-ink-400')}>
                {fmtSignedUSDT(f.realized_pnl, 4)}
              </td>
            </tr>
          {:else}
            <tr><td colspan="8" class="py-8 text-center text-ink-500 text-sm">无 fills</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}
