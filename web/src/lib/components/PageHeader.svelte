<script lang="ts">
  export let title: string;
  export let detail = "";
</script>

<header class="page-heading">
  <div class="min-w-0">
    <h1>{title}</h1>
    {#if detail}<p class="text-xs text-ink-400 mt-2 leading-relaxed">
        {detail}
      </p>{/if}
  </div>
  <div class="flex items-center gap-3 flex-wrap"><slot /></div>
</header>

<style>
  .page-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 18px;
    margin-bottom: 26px;
  }
</style>
