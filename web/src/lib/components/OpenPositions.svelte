<script lang="ts">
  import { onMount } from "svelte";
  import { ChevronDown } from "lucide-svelte";
  import type { Position } from "$lib/api";
  import {
    fmtDuration,
    fmtUSDT,
    fmtSignedUSDT,
    fmtSignedPct,
    pnlClass,
  } from "$lib/format";
  export let positions: Position[] = [];
  let now = Date.now();
  onMount(() => {
    const timer = setInterval(() => (now = Date.now()), 30_000);
    return () => clearInterval(timer);
  });
  $: totalUnrealized = positions.reduce(
    (s, p) => s + (p.unrealized_pnl ?? 0),
    0,
  );
  $: totalRealized = positions.reduce(
    (s, p) => s + (p.realized_pnl ?? 0) - (p.commission ?? 0),
    0,
  );
  const net = (p: Position) => (p.realized_pnl ?? 0) - (p.commission ?? 0);
  const fmtPx = (v: number) => fmtUSDT(v, v < 10 ? 4 : 2);
  const change = (p: Position) =>
    p.entry_price > 0
      ? (((p.mark_price ?? p.entry_price) - p.entry_price) / p.entry_price) *
        (p.side === "LONG" || p.side === "BUY" ? 1 : -1)
      : 0;
</script>

<section aria-label="当前持仓明细">
  <div class="section-head">
    <h2>
      当前持仓 <span class="text-xs text-ink-400 font-normal ml-2"
        >{positions.length}</span
      >
    </h2>
    <div class="position-summary">
      <span
        >浮动 <span class={"number " + pnlClass(totalUnrealized)}
          >{fmtSignedUSDT(totalUnrealized)}</span
        ></span
      ><span
        >已实现净收益 <span class={"number " + pnlClass(totalRealized)}
          >{fmtSignedUSDT(totalRealized)}</span
        ></span
      >
    </div>
  </div>
  {#if positions.length === 0}<div class="empty-state">当前没有持仓</div>
  {:else}
    <div class="desktop-positions table-scroll">
      <table class="data-table">
        <thead
          ><tr
            ><th>标的 / 方向</th><th class="text-right">开仓价</th><th
              class="text-right">标记价</th
            ><th class="text-right">持仓数量</th><th class="text-right"
              >浮动盈亏</th
            ><th class="text-right">已实现净收益</th><th class="text-right"
              >持仓时长</th
            ></tr
          ></thead
        ><tbody
          >{#each positions as p}<tr class="table-row-hover"
              ><td
                ><div class="flex items-center gap-3">
                  <strong class="text-[13px] font-semibold"
                    >{p.symbol.replace(/USDT$/, "")}</strong
                  ><span
                    class={p.side === "LONG" || p.side === "BUY"
                      ? "pill-pos"
                      : "pill-neg"}
                    >{p.side === "LONG" || p.side === "BUY" ? "多" : "空"}</span
                  >
                </div></td
              ><td class="number text-right text-xs text-ink-300"
                >{fmtPx(p.entry_price)}</td
              ><td class="number text-right text-xs text-ink-300"
                >{fmtPx(p.mark_price ?? p.entry_price)}</td
              ><td class="number text-right text-xs text-ink-300"
                >{fmtUSDT(p.quantity, 4)}</td
              ><td
                class={"number text-right text-xs " +
                  pnlClass(p.unrealized_pnl ?? 0)}
                >{p.unrealized_pnl === undefined
                  ? "—"
                  : fmtSignedUSDT(p.unrealized_pnl)}
                <div class="text-[10px] mt-1">
                  {fmtSignedPct(change(p))}
                </div></td
              ><td class={"number text-right text-xs " + pnlClass(net(p))}
                >{fmtSignedUSDT(net(p))}</td
              ><td class="text-right text-xs text-ink-400 whitespace-nowrap"
                >{fmtDuration(now - p.entry_time)}</td
              ></tr
            >{/each}</tbody
        >
      </table>
    </div>
    <div class="mobile-positions">
      {#each positions as p}<details>
          <summary
            ><span class="flex items-center gap-2"
              ><strong>{p.symbol.replace(/USDT$/, "")}</strong><span
                class={p.side === "LONG" || p.side === "BUY"
                  ? "pill-pos"
                  : "pill-neg"}
                >{p.side === "LONG" || p.side === "BUY" ? "多" : "空"}</span
              ></span
            ><span class="text-right"
              ><span class={"number " + pnlClass(p.unrealized_pnl ?? 0)}
                >{p.unrealized_pnl === undefined
                  ? "—"
                  : fmtSignedUSDT(p.unrealized_pnl)}</span
              ><span class="block text-ink-400 text-[10px] mt-1"
                >浮动盈亏 · USDT</span
              ></span
            ><ChevronDown size={14} class="disclosure-icon" /></summary
          >
          <dl>
            <div>
              <dt>开仓 / 标记价</dt>
              <dd class="number">
                {fmtPx(p.entry_price)} / {fmtPx(p.mark_price ?? p.entry_price)}
              </dd>
            </div>
            <div>
              <dt>价格变动</dt>
              <dd class={"number " + pnlClass(change(p))}>
                {fmtSignedPct(change(p))}
              </dd>
            </div>
            <div>
              <dt>数量</dt>
              <dd class="number">{fmtUSDT(p.quantity, 4)}</dd>
            </div>
            <div>
              <dt>已实现净收益</dt>
              <dd class={"number " + pnlClass(net(p))}>
                {fmtSignedUSDT(net(p))}
              </dd>
            </div>
            <div>
              <dt>持仓时长</dt>
              <dd>{fmtDuration(now - p.entry_time)}</dd>
            </div>
          </dl>
        </details>{/each}
    </div>
  {/if}
</section>

<style>
  .position-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 24px;
    font-size: 11px;
    color: var(--lo);
  }
  .position-summary .number {
    margin-left: 6px;
  }
  .desktop-positions table {
    min-width: 700px;
  }
  .mobile-positions {
    display: none;
  }
  @media (max-width: 767px) {
    .desktop-positions {
      display: none;
    }
    .mobile-positions {
      display: block;
    }
    details {
      border-top: 1px solid var(--panel2);
    }
    summary {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: center;
      padding: 17px 0;
      font-size: 13px;
    }
    summary :global(.disclosure-icon) { color: var(--lo); flex: none; transition: transform .18s; }
    details[open] summary :global(.disclosure-icon) { transform: rotate(180deg); }
    dl {
      display: grid;
      gap: 11px;
      padding: 0 0 18px;
      font-size: 11px;
    }
    dl div {
      display: flex;
      justify-content: space-between;
      gap: 10px;
    }
    dt {
      color: var(--lo);
    }
  }
</style>
