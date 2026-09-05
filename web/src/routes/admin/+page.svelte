<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import {
    RefreshCw,
    Plus,
    Camera,
    ArrowDownToLine,
    Users,
    ListFilter,
    Check,
    CircleAlert,
  } from "lucide-svelte";
  import { api, type FriendRow, type RecentFill, type Me } from "$lib/api";
  import {
    fmtUSDT,
    fmtSignedUSDT,
    fmtDate,
    fmtShares,
    pnlClass,
  } from "$lib/format";
  import PageHeader from "$lib/components/PageHeader.svelte";
  import AsyncState from "$lib/components/AsyncState.svelte";
  type LedgerRow = Awaited<ReturnType<typeof api.admin.listCashEvents>>[number];
  type View = "ledger" | "members" | "data";
  const tabs: { key: View; label: string; icon: typeof Users }[] = [
    { key: "ledger", label: "资金流水", icon: ArrowDownToLine },
    { key: "members", label: "成员管理", icon: Users },
    { key: "data", label: "成交与快照", icon: ListFilter },
  ];
  let view: View = "ledger";
  let friends: FriendRow[] = [],
    fills: RecentFill[] = [],
    ledger: LedgerRow[] = [];
  let me: Me | null = null;
  let loading = true,
    refreshing = false,
    cfBusy = false,
    ceBusy = false,
    snapBusy = false;
  let error = "",
    notice = "",
    fillsError = "";
  let noticeError = false;
  let togglingId: number | null = null;
  let cf = { name: "", username: "", password: "", is_admin: false };
  let ce = {
    username: "",
    type: "deposit" as "deposit" | "withdraw",
    amount_usdt: 0,
    note: "",
    manual_nav: 0,
    skip_bootstrap_check: false,
  };
  let ceResult: Awaited<ReturnType<typeof api.admin.cashEvent>> | null = null;
  let snapResult: Awaited<ReturnType<typeof api.admin.snapshot>> | null = null;
  $: netFlow = ledger.reduce(
    (sum, row) => sum + row.amount_usdt * (row.type === "withdraw" ? -1 : 1),
    0,
  );
  function announce(message: string, failed = false) {
    notice = message;
    noticeError = failed;
  }
  async function load(initial = true) {
    if (refreshing) return;
    refreshing = true;
    if (initial) loading = true;
    error = "";
    try {
      me = await api.me();
      if (!me.is_admin) {
        await goto("/");
        return;
      }
      const [f, l] = await Promise.all([
        api.admin.listFriends(),
        api.admin.listCashEvents(200),
      ]);
      friends = f ?? [];
      ledger = l ?? [];
      try {
        fills = (await api.admin.recentFills(20)) ?? [];
        fillsError = "";
      } catch {
        fillsError = "最近成交暂不可用";
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "加载失败";
    } finally {
      loading = false;
      refreshing = false;
    }
  }
  async function submitCreateFriend(event: SubmitEvent) {
    event.preventDefault();
    cfBusy = true;
    notice = "";
    try {
      await api.admin.createFriend(cf);
      cf = { name: "", username: "", password: "", is_admin: false };
      friends = (await api.admin.listFriends()) ?? [];
      announce("成员账号已创建");
    } catch (e) {
      announce(e instanceof Error ? e.message : "创建失败", true);
    } finally {
      cfBusy = false;
    }
  }
  async function toggleActive(friend: FriendRow) {
    togglingId = friend.id;
    notice = "";
    try {
      await api.admin.setFriendActive(friend.id, !friend.active);
      friends = (await api.admin.listFriends()) ?? [];
      announce(`已${friend.active ? "停用" : "恢复"} ${friend.name}`);
    } catch (e) {
      announce(e instanceof Error ? e.message : "操作失败", true);
    } finally {
      togglingId = null;
    }
  }
  async function submitCashEvent(event: SubmitEvent) {
    event.preventDefault();
    ceBusy = true;
    notice = "";
    ceResult = null;
    try {
      ceResult = await api.admin.cashEvent({
        username: ce.username,
        type: ce.type,
        amount_usdt: Number(ce.amount_usdt),
        note: ce.note || undefined,
        ...(ce.manual_nav > 0 ? { manual_nav: Number(ce.manual_nav) } : {}),
        ...(ce.skip_bootstrap_check ? { skip_bootstrap_check: true } : {}),
      });
      announce(`${ce.type === "deposit" ? "入金" : "赎回"}已记录`);
      ledger = (await api.admin.listCashEvents(200)) ?? [];
    } catch (e) {
      announce(e instanceof Error ? e.message : "记录失败", true);
    } finally {
      ceBusy = false;
    }
  }
  async function takeSnapshot() {
    snapBusy = true;
    notice = "";
    snapResult = null;
    try {
      snapResult = await api.admin.snapshot();
      announce("净值快照已生成");
    } catch (e) {
      announce(e instanceof Error ? e.message : "快照失败", true);
    } finally {
      snapBusy = false;
    }
  }
  function tabKey(event: KeyboardEvent) {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const current = tabs.findIndex((t) => t.key === view);
    const index =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? tabs.length - 1
          : (current + (event.key === "ArrowRight" ? 1 : -1) + tabs.length) %
            tabs.length;
    view = tabs[index].key;
    document.getElementById(`tab-${view}`)?.focus();
  }
  onMount(() => load());
</script>

{#if loading || error}<AsyncState {loading} {error} on:retry={() => load()} />
{:else if me?.is_admin}
  <PageHeader title="基金管理" detail="成员账户、资金记录与净值快照"
    ><button
      class="icon-button"
      aria-label="刷新管理数据"
      disabled={refreshing}
      on:click={() => load(false)}
      ><RefreshCw size={16} class={refreshing ? "refreshing" : ""} /></button
    ></PageHeader
  >
  <div class="admin-tabs" role="tablist" aria-label="管理视图">
    {#each tabs as tab}<button
        id={`tab-${tab.key}`}
        type="button"
        role="tab"
        class:active={view === tab.key}
        aria-selected={view === tab.key}
        aria-controls={`panel-${tab.key}`}
        tabindex={view === tab.key ? 0 : -1}
        on:keydown={tabKey}
        on:click={() => (view = tab.key)}
        ><svelte:component this={tab.icon} size={15} />{tab.label}</button
      >{/each}
  </div>
  {#if notice}<div
      class="notice"
      class:failed={noticeError}
      role={noticeError ? "alert" : "status"}
    >
      {#if noticeError}<CircleAlert size={16} />{:else}<Check
          size={16}
        />{/if}<span>{notice}</span>
    </div>{/if}
  {#if view === "ledger"}
    <div tabindex="0"
      id="panel-ledger"
      role="tabpanel"
      aria-labelledby="tab-ledger"
      class="admin-workspace"
    >
      <div class="section">
        <div class="section-head">
          <h2>入金与赎回</h2>
          <span class="section-meta">最近 {ledger.length} 条</span>
        </div>
        <div class="ledger-summary">
          <span>所列流水净额</span><strong class="number"
            >{fmtSignedUSDT(netFlow)} <span>USDT</span></strong
          >
        </div>
        <div class="table-scroll">
          <table class="data-table ledger-table">
            <thead
              ><tr
                ><th>时间 / 成员</th><th>类型</th><th class="text-right"
                  >金额 · USDT</th
                ><th class="text-right">事件 NAV / 份额变动</th><th
                  >备注 / 来源</th
                ></tr
              ></thead
            ><tbody
              >{#each ledger as row}<tr class="table-row-hover"
                  ><td
                    ><div class="text-xs">{row.name}</div>
                    <div class="text-[10px] text-ink-400 number mt-1">
                      {fmtDate(row.occurred_at, true)}
                    </div>
                    <div class="text-[10px] text-ink-400 mt-1">
                      {row.username}
                    </div></td
                  ><td
                    ><span
                      class={row.type === "deposit"
                        ? "pill-pos"
                        : "pill-neutral"}
                      >{row.type === "deposit" ? "入金" : "赎回"}</span
                    ></td
                  ><td class="number text-xs text-right whitespace-nowrap"
                    >{fmtUSDT(row.amount_usdt, 4)}</td
                  ><td class="number text-xs text-right text-ink-300"
                    ><div>{row.nav_at_event.toFixed(6)}</div>
                    <div
                      class={"text-[10px] mt-1 " + pnlClass(row.shares_delta)}
                    >
                      {row.shares_delta > 0 ? "+" : ""}{fmtShares(
                        row.shares_delta,
                        4,
                      )}
                    </div></td
                  ><td class="text-xs text-ink-300"
                    ><div class="note-text" title={row.note}>
                      {row.note || "—"}
                    </div>
                    <div class="text-[10px] text-ink-400 mt-1">
                      {row.source === "manual" ? "手动记录" : "账户划转"}
                    </div></td
                  ></tr
                >{:else}<tr
                  ><td colspan="5"
                    ><div class="empty-state">还没有资金流水</div></td
                  ></tr
                >{/each}</tbody
            >
          </table>
        </div>
      </div>
      <form class="operation-form" on:submit={submitCashEvent}>
        <div class="section-head">
          <h2>记录资金变动</h2>
          <ArrowDownToLine size={16} class="text-ink-400" />
        </div>
        <label
          ><span>成员</span><select
            class="select"
            bind:value={ce.username}
            required
            ><option value="" disabled>选择成员</option
            >{#each friends as f}<option value={f.username}
                >{f.name} · {f.username}</option
              >{/each}</select
          ></label
        >
        <fieldset>
          <legend class="label mb-2">类型</legend>
          <div class="cash-type">
            {#each [{ value: "deposit", label: "入金" }, { value: "withdraw", label: "赎回" }] as type}<label
                class:selected={ce.type === type.value}
                ><input
                  type="radio"
                  name="cash-type"
                  value={type.value}
                  bind:group={ce.type}
                /><span>{type.label}</span></label
              >{/each}
          </div>
        </fieldset>
        <label
          ><span>金额 · USDT</span><input
            class="input number"
            type="number"
            inputmode="decimal"
            step="0.0001"
            min="0.0001"
            bind:value={ce.amount_usdt}
            required
          /></label
        ><label
          ><span>备注 <span class="text-ink-500">可选</span></span><input
            class="input"
            type="text"
            bind:value={ce.note}
            placeholder="资金变动备注"
          /></label
        >
        <details class="advanced-options">
          <summary>高级核算选项</summary>
          <div>
            <label
              ><span>手动指定 NAV <span class="text-ink-500">可选</span></span
              ><input
                class="input number"
                type="number"
                step="0.000001"
                min="0"
                bind:value={ce.manual_nav}
              /></label
            >
            <p class="muted-note">填 0 时使用账户当前净值。</p>
            <label class="checkbox-label"
              ><input
                type="checkbox"
                bind:checked={ce.skip_bootstrap_check}
              /><span>跳过首笔入金的 1% 净值偏离校验</span></label
            >
          </div>
        </details>
        <button class="btn-primary w-full" type="submit" disabled={ceBusy}
          ><Plus size={15} />{ceBusy
            ? "正在记录"
            : `记录${ce.type === "deposit" ? "入金" : "赎回"}`}</button
        >{#if ceResult}<div class="form-result" role="status">
            <div>
              <span>事件 NAV</span><strong class="number"
                >{ceResult.nav_at_event.toFixed(6)}</strong
              >
            </div>
            <div>
              <span>份额变动</span><strong class="number"
                >{fmtShares(ceResult.shares_delta, 4)}</strong
              >
            </div>
            <div class="muted-note">
              {ceResult.manual_nav ? "使用手动指定净值" : "使用账户净值"}
            </div>
          </div>{/if}
      </form>
    </div>
  {:else if view === "members"}
    <div tabindex="0"
      id="panel-members"
      role="tabpanel"
      aria-labelledby="tab-members"
      class="admin-workspace"
    >
      <div class="section">
        <div class="section-head">
          <h2>基金成员</h2>
          <span class="section-meta">{friends.length} 人</span>
        </div>
        <p class="muted-note mb-5">停用后无法登录，已有份额与流水保留。</p>
        <div class="table-scroll">
          <table class="data-table member-table">
            <thead
              ><tr
                ><th>成员 / 账号</th><th>角色</th><th>状态</th><th
                  class="text-right">操作</th
                ></tr
              ></thead
            ><tbody
              >{#each friends as f}<tr class="table-row-hover"
                  ><td
                    ><div class="text-[13px]">{f.name}</div>
                    <div class="text-[11px] text-ink-400 number mt-1">
                      {f.username}
                    </div></td
                  ><td class="text-xs text-ink-300"
                    >{f.is_admin ? "管理员" : "成员"}</td
                  ><td
                    ><span class={f.active ? "pill-pos" : "pill-neutral"}
                      >{f.active ? "启用" : "停用"}</span
                    ></td
                  ><td class="text-right"
                    >{#if f.id === me.id}<span class="text-[11px] text-ink-400"
                        >当前账号</span
                      >{:else}<button
                        class="btn-ghost"
                        disabled={togglingId !== null}
                        on:click={() => toggleActive(f)}
                        >{togglingId === f.id
                          ? "处理中"
                          : f.active
                            ? "停用"
                            : "恢复"}</button
                      >{/if}</td
                  ></tr
                >{/each}</tbody
            >
          </table>
        </div>
      </div>
      <form class="operation-form" on:submit={submitCreateFriend}>
        <div class="section-head">
          <h2>新建成员</h2>
          <Users size={16} class="text-ink-400" />
        </div>
        <label
          ><span>显示名</span><input
            class="input"
            bind:value={cf.name}
            required
          /></label
        ><label
          ><span>用户名</span><input
            class="input"
            bind:value={cf.username}
            required
            autocomplete="off"
            autocapitalize="none"
            spellcheck="false"
          /></label
        ><label
          ><span>初始密码</span><input
            class="input"
            type="password"
            bind:value={cf.password}
            required
            minlength="8"
            autocomplete="new-password"
            placeholder="至少 8 个字符"
          /></label
        ><label class="checkbox-label"
          ><input type="checkbox" bind:checked={cf.is_admin} /><span
            >授予管理员权限</span
          ></label
        ><button class="btn-primary w-full" type="submit" disabled={cfBusy}
          ><Plus size={15} />{cfBusy ? "正在创建" : "创建成员"}</button
        >
      </form>
    </div>
  {:else}
    <div tabindex="0" id="panel-data" role="tabpanel" aria-labelledby="tab-data">
      <div class="snapshot-tool">
        <div>
          <h2>净值快照</h2>
          <p class="muted-note mt-2">记录当前基金权益与单位净值。</p>
        </div>
        <button class="btn-ghost" on:click={takeSnapshot} disabled={snapBusy}
          ><Camera size={15} />{snapBusy ? "正在生成" : "生成快照"}</button
        >
      </div>
      {#if snapResult}<div class="snapshot-result" role="status">
          <span class="number">{fmtDate(snapResult.taken_at, true)}</span><span
            >权益 <strong class="number"
              >{fmtUSDT(snapResult.total_equity)}</strong
            > USDT</span
          ><span
            >NAV <strong class="number">{snapResult.nav.toFixed(6)}</strong
            ></span
          >
        </div>{/if}
      <div class="data-section">
        <div class="section-head">
          <h2>最近成交</h2>
          <span class="section-meta">最近 {fills.length} 条 · Binance</span>
        </div>
        {#if fillsError}<AsyncState
            error={fillsError}
            on:retry={() => load(false)}
          />{:else}<div class="table-scroll">
            <table class="data-table fills-table">
              <thead
                ><tr
                  ><th>时间</th><th>标的</th><th>方向</th><th class="text-right"
                    >价格</th
                  ><th class="text-right">数量</th><th class="text-right"
                    >名义 · USDT</th
                  ><th class="text-right">已实现毛收益</th></tr
                ></thead
              ><tbody
                >{#each fills as f}<tr class="table-row-hover"
                    ><td class="number text-xs text-ink-400 whitespace-nowrap"
                      >{fmtDate(f.fill_time, true)}</td
                    ><td class="text-xs font-semibold">{f.symbol}</td><td
                      ><span class={f.side === "BUY" ? "pill-pos" : "pill-neg"}
                        >{f.side === "BUY" ? "买入" : "卖出"}</span
                      >{#if f.position_side && f.position_side !== "BOTH"}<span
                          class="text-[10px] text-ink-400 ml-1"
                          >{f.position_side === "LONG"
                            ? "多"
                            : f.position_side === "SHORT"
                              ? "空"
                              : f.position_side}</span
                        >{/if}</td
                    ><td class="number text-right text-xs"
                      >{fmtUSDT(f.price, 4)}</td
                    ><td class="number text-right text-xs"
                      >{fmtUSDT(f.qty, 4)}</td
                    ><td class="number text-right text-xs"
                      >{fmtUSDT(f.quote_qty)}</td
                    ><td
                      class={"number text-right text-xs " +
                        pnlClass(f.realized_pnl)}
                      >{fmtSignedUSDT(f.realized_pnl, 4)}</td
                    ></tr
                  >{:else}<tr
                    ><td colspan="7"
                      ><div class="empty-state">暂无成交记录</div></td
                    ></tr
                  >{/each}</tbody
              >
            </table>
          </div>{/if}
      </div>
    </div>
  {/if}
{/if}

<style>
  .admin-tabs {
    display: flex;
    gap: 25px;
    border-bottom: 1px solid var(--line);
  }
  .admin-tabs button {
    color: var(--lo);
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 0 15px;
    border-bottom: 2px solid transparent;
    font-size: 12px;
  }
  .admin-tabs button.active {
    color: var(--hi);
    border-bottom-color: var(--pos);
  }
  .admin-tabs button:hover {
    color: var(--hi);
  }
  .admin-workspace {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 285px;
    gap: 32px;
  }
  .operation-form {
    padding: 26px 0 26px 26px;
    border-left: 1px solid var(--line);
    display: flex;
    flex-direction: column;
    gap: 19px;
    align-self: start;
  }
  .operation-form :global(.section-head) {
    margin-bottom: 1px;
  }
  .operation-form label > span {
    display: block;
    color: var(--mid);
    font-size: 12px;
    margin-bottom: 8px;
  }
  .operation-form .checkbox-label {
    display: flex;
    align-items: flex-start;
    gap: 9px;
  }
  .operation-form .checkbox-label input {
    flex: none;
    margin-top: 2px;
  }
  .operation-form .checkbox-label > span {
    font-size: 11px;
    line-height: 1.7;
    margin: 0;
  }
  .cash-type {
    display: grid;
    grid-template-columns: 1fr 1fr;
    border: 1px solid var(--line);
    border-radius: 5px;
    padding: 3px;
  }
  .cash-type label {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: center;
    min-height: 35px;
    border-radius: 3px;
    cursor: pointer;
  }
  .cash-type label.selected {
    background: var(--panel2);
  }
  .cash-type label > span {
    margin: 0;
    font-size: 12px;
  }
  .cash-type input {
    accent-color: var(--pos);
    width: 12px;
    height: 12px;
  }
  .advanced-options {
    border-top: 1px solid var(--line);
    padding-top: 15px;
  }
  .advanced-options summary {
    color: var(--lo);
    font-size: 11px;
  }
  .advanced-options > div {
    display: grid;
    gap: 14px;
    margin-top: 18px;
  }
  .notice {
    display: flex;
    gap: 10px;
    align-items: center;
    color: var(--pos);
    background: var(--pos-tint);
    padding: 13px 16px;
    margin-top: 20px;
    border-radius: 4px;
    font-size: 12px;
  }
  .notice.failed {
    color: var(--neg);
    background: var(--neg-tint);
  }
  .form-result {
    display: grid;
    gap: 10px;
    border-top: 1px solid var(--line);
    padding-top: 16px;
    font-size: 11px;
  }
  .form-result > div {
    display: flex;
    justify-content: space-between;
    gap: 8px;
    color: var(--mid);
  }
  .form-result strong {
    font-weight: 400;
    color: var(--pos);
    overflow-wrap: anywhere;
  }
  .ledger-summary {
    display: flex;
    gap: 13px;
    align-items: baseline;
    flex-wrap: wrap;
    margin-bottom: 24px;
    color: var(--lo);
    font-size: 11px;
  }
  .ledger-summary strong {
    color: var(--hi);
    font-size: 18px;
    font-weight: 500;
  }
  .ledger-summary strong span {
    color: var(--lo);
    font-size: 10px;
    font-weight: 400;
  }
  .ledger-table {
    min-width: 640px;
  }
  .member-table {
    min-width: 430px;
  }
  .fills-table {
    min-width: 880px;
  }
  .note-text {
    max-width: 140px;
    overflow-wrap: anywhere;
  }
  .snapshot-tool {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    padding: 28px 0;
  }
  .snapshot-result {
    display: flex;
    flex-wrap: wrap;
    gap: 13px 25px;
    font-size: 12px;
    color: var(--mid);
    padding-bottom: 26px;
  }
  .snapshot-result strong {
    font-weight: 500;
    color: var(--hi);
  }
  @media (max-width: 1100px) {
    .admin-workspace {
      grid-template-columns: minmax(0, 1fr);
      gap: 0;
    }
    .operation-form {
      padding: 26px 0;
      border-left: 0;
      border-top: 1px solid var(--line);
      max-width: 480px;
      width: 100%;
    }
  }
  @media (max-width: 600px) {
    .admin-tabs {
      gap: 19px;
    }
    .admin-tabs button {
      font-size: 11px;
      gap: 6px;
    }
    .snapshot-tool {
      align-items: flex-start;
      flex-wrap: wrap;
    }
  }
</style>
