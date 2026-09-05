<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { RefreshCw, CircleAlert } from "lucide-svelte";
  export let loading = false;
  export let error = "";
  const dispatch = createEventDispatcher<{ retry: void }>();
</script>

{#if loading}
  <div role="status" aria-label="正在加载数据" class="space-y-7 py-2">
    <div class="skeleton h-7 w-36"></div>
    <div
      class="grid grid-cols-2 lg:grid-cols-4 gap-6 py-5 border-y border-ink-800"
    >
      {#each [1, 2, 3, 4] as i}<div class="space-y-4">
          <div class="skeleton h-3 w-20"></div>
          <div class="skeleton h-7 w-4/5"></div>
        </div>{/each}
    </div>
    <div class="skeleton h-64 w-full"></div>
  </div>
{:else if error}
  <div class="alert" role="alert">
    <div class="flex gap-3 items-center min-w-0">
      <CircleAlert size={18} class="flex-none" /><span class="break-words"
        >{error}</span
      >
    </div>
    <button class="btn-ghost" on:click={() => dispatch("retry")}
      ><RefreshCw size={14} />重试</button
    >
  </div>
{/if}
