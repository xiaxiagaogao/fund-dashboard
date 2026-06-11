<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { api, type Me, ApiError } from '$lib/api';

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

  $: showShell = !!me && $page.url.pathname !== '/login';
</script>

{#if loading}
  <div class="min-h-screen flex items-center justify-center text-ink-400 text-sm">加载中…</div>
{:else if showShell && me}
  <div class="min-h-screen">
    <header class="sticky top-0 z-10 backdrop-blur bg-ink-950/80 border-b border-ink-800/80">
      <div class="max-w-6xl mx-auto px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-7 h-7 rounded-lg bg-gradient-to-br from-accent-400 to-accent-600 ring-1 ring-accent-500/40
                      grid place-items-center text-ink-950 font-bold text-xs">F</div>
          <div>
            <div class="text-sm font-semibold tracking-tight">Fund Dashboard</div>
            <div class="text-[11px] text-ink-400">{me.name}{me.is_admin ? ' · admin' : ''}</div>
          </div>
        </div>
        <nav class="flex items-center gap-2 text-sm">
          <a
            href="/"
            class={'px-3 py-1.5 rounded-lg ' +
              ($page.url.pathname === '/' ? 'bg-ink-800 text-ink-50' : 'text-ink-300 hover:text-ink-100')}
            >我的</a
          >
          {#if me.is_admin}
            <a
              href="/admin"
              class={'px-3 py-1.5 rounded-lg ' +
                ($page.url.pathname.startsWith('/admin') ? 'bg-ink-800 text-ink-50' : 'text-ink-300 hover:text-ink-100')}
              >Admin</a
            >
          {/if}
          <button class="btn-ghost text-xs px-2.5 py-1.5" on:click={logout}>退出</button>
        </nav>
      </div>
    </header>
    <main class="max-w-6xl mx-auto px-6 py-8">
      <slot />
    </main>
    <footer class="max-w-6xl mx-auto px-6 py-8 text-[11px] text-ink-500">
      所有数据本地存储 · NAV 单位法 · {new Date().getFullYear()}
    </footer>
  </div>
{:else}
  <slot />
{/if}
