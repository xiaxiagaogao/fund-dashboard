<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { api, type Me } from '$lib/api';

  let me: Me | null = null;
  let loading = true;

  onMount(async () => {
    const publicPath = $page.url.pathname === '/login';
    try {
      me = await api.me();
    } catch (e) {
      me = null;
      if (!publicPath) {
        goto('/login');
        return;
      }
    } finally {
      loading = false;
    }
  });

  async function logout() {
    try {
      await api.logout();
    } catch {}
    me = null;
    goto('/login');
  }

  $: path = $page.url.pathname;
  $: showShell = !!me && path !== '/login';

  // Nav destinations the current user can reach. 复盘 + Admin are operator-only.
  $: navItems = me
    ? [
        { href: '/', label: '我的看板', icon: 'activity' },
        ...(me.is_admin
          ? [
              { href: '/review', label: '复盘分析', icon: 'bars' },
              { href: '/admin', label: '管理', icon: 'cog' }
            ]
          : [])
      ]
    : [];
  $: isActive = (href: string) => (href === '/' ? path === '/' : path.startsWith(href));
</script>

{#if loading}
  <div class="min-h-screen flex items-center justify-center text-ink-400 text-sm">加载中…</div>
{:else if showShell && me}
  <div class="md:flex min-h-screen">
    <!-- Desktop sidebar -->
    <aside class="hidden md:flex w-[236px] flex-none flex-col gap-7 border-r border-white/[0.08] px-4 py-6 sticky top-0 h-screen">
      <div class="flex items-center gap-3 px-1.5">
        <div class="w-[30px] h-[30px] rounded-[9px] flex items-center justify-center flex-none shadow-glow"
          style="background:linear-gradient(140deg,var(--pos),oklch(0.66 0.11 182))">
          <div class="w-[11px] h-[11px] rounded-[3px]" style="background:var(--bg)"></div>
        </div>
        <div>
          <div class="text-sm font-extrabold tracking-tight leading-none">朋友基金</div>
          <div class="text-[10px] text-ink-500 tracking-[0.16em] uppercase mt-1 whitespace-nowrap">NAV · 单位法核算</div>
        </div>
      </div>

      <nav class="flex flex-col gap-1">
        <div class="text-[10px] text-ink-500 tracking-[0.16em] uppercase px-2 pb-2">导航</div>
        {#each navItems as item}
          <a href={item.href}
            class={'flex items-center gap-2.5 px-3 py-2 rounded-xl text-sm font-medium transition-colors ' +
              (isActive(item.href) ? 'bg-ink-800 text-ink-50' : 'text-ink-400 hover:text-ink-100 hover:bg-white/[0.03]')}>
            <span class={'w-1.5 h-1.5 rounded-full flex-none ' + (isActive(item.href) ? '' : 'opacity-40')}
              style={isActive(item.href) ? 'background:var(--pos)' : 'background:currentColor'}></span>
            {item.label}
          </a>
        {/each}
      </nav>

      <div class="mt-auto border-t border-white/[0.08] pt-4 flex flex-col gap-3">
        <div class="flex items-center gap-2.5 text-[11px] text-ink-300">
          <span class="w-1.5 h-1.5 rounded-full animate-pulse" style="background:var(--pos)"></span>
          实时 · 30 分钟快照
        </div>
        <div class="flex items-center gap-2.5">
          <div class="w-[30px] h-[30px] rounded-full bg-ink-800 border border-white/[0.08] flex items-center justify-center text-xs font-bold flex-none">
            {me.name.slice(0, 1)}
          </div>
          <div class="leading-tight min-w-0">
            <div class="text-xs font-semibold truncate">{me.name}</div>
            <div class="text-[10px] text-ink-500">{me.is_admin ? '管理 · 透明账本' : '成员 · 透明账本'}</div>
          </div>
          <button class="ml-auto text-ink-500 hover:text-ink-200 text-xs" on:click={logout} aria-label="退出">退出</button>
        </div>
      </div>
    </aside>

    <!-- Mobile top header -->
    <header class="md:hidden sticky top-0 z-10 backdrop-blur bg-ink-950/80 border-b border-white/[0.08] px-4 py-3 flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <div class="w-[26px] h-[26px] rounded-lg flex items-center justify-center flex-none"
          style="background:linear-gradient(140deg,var(--pos),oklch(0.66 0.11 182))">
          <div class="w-[9px] h-[9px] rounded-[3px]" style="background:var(--bg)"></div>
        </div>
        <div class="text-sm font-extrabold tracking-tight">朋友基金</div>
      </div>
      <button class="text-ink-400 hover:text-ink-100 text-xs" on:click={logout}>退出</button>
    </header>

    <!-- Main -->
    <main class="flex-1 min-w-0 px-4 md:px-8 py-5 md:py-7 pb-24 md:pb-12">
      <div class="max-w-5xl mx-auto">
        <slot />
      </div>
    </main>

    <!-- Mobile bottom tab bar (only when there's more than one destination) -->
    {#if navItems.length > 1}
      <nav class="md:hidden fixed bottom-0 inset-x-0 z-10 flex px-6 pt-2 pb-7 border-t border-white/[0.08]"
        style="background:oklch(0.18 0.008 240 / 0.86);backdrop-filter:blur(16px)">
        {#each navItems as item}
          <a href={item.href}
            class={'flex-1 flex flex-col items-center gap-1 py-1.5 transition-colors ' +
              (isActive(item.href) ? 'text-accent-400' : 'text-ink-500')}>
            {#if item.icon === 'activity'}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 13h4l2 5 4-13 2 8h6" /></svg>
            {:else if item.icon === 'bars'}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="12" width="4" height="8" rx="1" /><rect x="10" y="7" width="4" height="13" rx="1" /><rect x="17" y="3" width="4" height="17" rx="1" /></svg>
            {:else}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" /></svg>
            {/if}
            <span class="text-[10px] font-semibold">{item.label}</span>
          </a>
        {/each}
      </nav>
    {/if}
  </div>
{:else}
  <slot />
{/if}
