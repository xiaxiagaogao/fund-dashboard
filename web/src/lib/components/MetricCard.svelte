<script lang="ts">
  export let label: string;
  export let value: string;
  /** Optional small caption under the value */
  export let sub: string | null = null;
  /** Optional pill text (e.g., +12.3%) */
  export let pill: string | null = null;
  /** numeric to color the pill: positive = green, negative = red */
  export let pillSignal: number | null = null;
  /** value-color override: 'pos' | 'neg' | null */
  export let valueSignal: number | null = null;

  function pillClass(n: number | null): string {
    if (n === null) return 'pill-neutral';
    if (Math.abs(n) < 1e-9) return 'pill-neutral';
    return n > 0 ? 'pill-pos' : 'pill-neg';
  }
  function valueClass(n: number | null): string {
    if (n === null) return 'text-ink-50';
    if (Math.abs(n) < 1e-9) return 'text-ink-50';
    return n > 0 ? 'pos' : 'neg';
  }
</script>

<div class="card p-5">
  <div class="label">{label}</div>
  <div class={'stat-value mt-2 ' + valueClass(valueSignal)}>{value}</div>
  <div class="flex items-center justify-between mt-2 min-h-[20px]">
    {#if sub}<div class="stat-sub text-ink-400">{sub}</div>{:else}<span></span>{/if}
    {#if pill}<span class={pillClass(pillSignal)}>{pill}</span>{/if}
  </div>
</div>
