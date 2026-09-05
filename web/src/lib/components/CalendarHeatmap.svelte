<script lang="ts">
  import { ChevronLeft, ChevronRight } from "lucide-svelte";
  import type { DayPnL } from "$lib/api";
  import { fmtSignedUSDT, fmtUSDT, pnlClass } from "$lib/format";
  export let days: DayPnL[] = [];
  const today = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Shanghai",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
  let selectedMonth = today.slice(0, 7);
  let selectedDay = today;
  $: byDate = new Map(days.map((d) => [d.date, d]));
  $: earliest = days.length
    ? [...days].sort((a, b) => a.date.localeCompare(b.date))[0].date.slice(0, 7)
    : today.slice(0, 7);
  $: monthsDays = days.filter((d) => d.date.startsWith(selectedMonth));
  $: monthNet = monthsDays.reduce((s, d) => s + d.net, 0);
  $: selected = byDate.get(selectedDay);
  $: cells = (() => {
    const [year, month] = selectedMonth.split("-").map(Number);
    const offset = (new Date(year, month - 1, 1).getDay() + 6) % 7;
    const count = new Date(year, month, 0).getDate();
    return Array.from({ length: 42 }, (_, i) => {
      const n = i - offset + 1;
      return n >= 1 && n <= count
        ? { n, key: `${selectedMonth}-${String(n).padStart(2, "0")}` }
        : null;
    });
  })();
  function moveMonth(amount: number) {
    const [y, m] = selectedMonth.split("-").map(Number);
    const date = new Date(y, m - 1 + amount, 1);
    selectedMonth = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
    selectedDay =
      days
        .filter((d) => d.date.startsWith(selectedMonth))
        .map((d) => d.date)
        .sort()
        .at(-1) ?? `${selectedMonth}-01`;
  }
  function shortAmount(value: number) {
    return Math.abs(value) >= 1000
      ? `${value >= 0 ? "+" : "−"}${(Math.abs(value) / 1000).toFixed(1)}k`
      : fmtSignedUSDT(value, 0);
  }
</script>

<section class="section" aria-label="每日盈亏日历">
  <div class="section-head">
    <h2>每日盈亏</h2>
    <div class="month-navigation">
      <button
        class="icon-button"
        aria-label="上个月"
        disabled={selectedMonth <= earliest}
        on:click={() => moveMonth(-1)}><ChevronLeft size={15} /></button
      ><span class="number">{selectedMonth.replace("-", " / ")}</span><button
        class="icon-button"
        aria-label="下个月"
        disabled={selectedMonth >= today.slice(0, 7)}
        on:click={() => moveMonth(1)}><ChevronRight size={15} /></button
      >
    </div>
  </div>
  <div class="calendar-summary">
    <span class={"number " + pnlClass(monthNet)}>{fmtSignedUSDT(monthNet)}</span
    ><span
      >USDT <span class="ml-2"
        >{monthsDays.filter((d) => d.fills > 0).length} 个交易日</span
      ></span
    >
  </div>
  <div class="calendar-grid">
    <div class="weekdays">
      {#each ["一", "二", "三", "四", "五", "六", "日"] as weekday}<span
          >{weekday}</span
        >{/each}
    </div>
    <div class="calendar-cells">
      {#each cells as cell}{#if cell}{@const record = byDate.get(
            cell.key,
          )}<button
            class="day-cell"
            class:positive={record && record.net > 0}
            class:negative={record && record.net < 0}
            class:selected={selectedDay === cell.key}
            disabled={cell.key > today}
            aria-pressed={selectedDay === cell.key}
            aria-label={`${cell.key}，${record ? `净收益 ${fmtSignedUSDT(record.net)} USDT，${record.fills} 笔成交` : "无成交记录"}`}
            on:click={() => (selectedDay = cell.key)}
            ><span class="day-number">{cell.n}</span><span
              class="day-pnl number"
              >{record ? shortAmount(record.net) : "—"}</span
            ></button
          >{:else}<div class="day-placeholder"></div>{/if}{/each}
    </div>
  </div>
  <div class="calendar-detail" aria-live="polite">
    <span class="number">{selectedDay}</span>{#if selected}<span
        >净 <span class={"number " + pnlClass(selected.net)}
          >{fmtSignedUSDT(selected.net)}</span
        >
        · 费 <span class="number">{fmtUSDT(selected.commission)}</span> · {selected.fills}
        笔</span
      >{:else}<span>无成交记录</span>{/if}
  </div>
</section>

<style>
  .month-navigation {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: var(--mid);
  }
  .month-navigation :global(.icon-button) {
    width: 27px;
    height: 30px;
  }
  .calendar-summary {
    display: flex;
    align-items: baseline;
    gap: 10px;
    margin: 0 0 20px;
  }
  .calendar-summary > .number {
    font-size: 20px;
  }
  .calendar-summary > span:last-child {
    font-size: 10px;
    color: var(--lo);
  }
  .weekdays,
  .calendar-cells {
    display: grid;
    grid-template-columns: repeat(7, minmax(0, 1fr));
    gap: 4px;
  }
  .weekdays {
    margin-bottom: 8px;
    text-align: center;
    color: var(--lo);
    font-size: 10px;
  }
  .day-cell,
  .day-placeholder {
    height: 39px;
    min-width: 0;
  }
  .day-cell {
    border: 1px solid transparent;
    border-radius: 3px;
    background: var(--panel);
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: 3px;
  }
  .day-cell.positive {
    background: #afd4bf0f;
    color: var(--pos);
  }
  .day-cell.negative {
    background: #e58d8912;
    color: var(--neg);
  }
  .day-cell.selected {
    border-color: var(--mid);
  }
  .day-cell:hover:not(:disabled) {
    border-color: var(--lo);
  }
  .day-cell:disabled {
    opacity: 0.3;
  }
  .day-number {
    color: var(--lo);
    font-size: 10px;
  }
  .day-pnl {
    font-size: 9px;
  }
  .calendar-detail {
    min-height: 43px;
    padding-top: 13px;
    display: flex;
    flex-wrap: wrap;
    gap: 5px 10px;
    align-content: start;
    font-size: 10px;
    color: var(--lo);
  }
</style>
