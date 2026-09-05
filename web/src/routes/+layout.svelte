<script lang="ts">
  import "../app.css";
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import {
    Activity,
    ChartNoAxesCombined,
    SlidersHorizontal,
    LogOut,
    ArrowUpRight,
  } from "lucide-svelte";
  import { api } from "$lib/api";
  import { session } from "$lib/session";

  let loading = true;
  let exiting = false;
  onMount(async () => {
    try {
      $session = await api.me();
    } catch {
      $session = null;
      if ($page.url.pathname !== "/login") await goto("/login");
    } finally {
      loading = false;
    }
  });
  async function logout() {
    exiting = true;
    try {
      await api.logout();
    } catch {
      exiting = false;
      return;
    }
    $session = null;
    exiting = false;
    await goto("/login");
  }
  $: path = $page.url.pathname;
  $: navItems = [
    {
      href: "/",
      label: $session?.is_admin ? "基金总览" : "我的资产",
      icon: Activity,
    },
    ...($session?.is_admin
      ? [
          { href: "/review", label: "交易复盘", icon: ChartNoAxesCombined },
          { href: "/admin", label: "基金管理", icon: SlidersHorizontal },
        ]
      : []),
  ];
</script>

<svelte:head
  ><title
    >{navItems.find((n) => n.href === path)?.label ?? "登录"} · XG fund</title
  ></svelte:head
>

{#if loading}
  <div class="boot-state" role="status" aria-label="正在加载账户">
    <div class="text-lg font-semibold">XG fund</div>
    <div class="skeleton h-1 w-36 mt-5"></div>
  </div>
{:else if $session && path !== "/login"}
  <a class="skip-link" href="#main">跳到内容</a>
  <div class="app-shell">
    <aside class="sidebar">
      <a class="brand" href="/" aria-label="XG fund 首页"
        ><span class="brand-mark"
          ><ChartNoAxesCombined size={21} strokeWidth={1.6} /></span
        ><span>XG <span class="font-normal text-ink-400">fund</span></span></a
      >
      <nav aria-label="主导航" class="desktop-nav">
        {#each navItems as item}<a
            href={item.href}
            class:active={path === item.href}
            aria-current={path === item.href ? "page" : undefined}
            ><svelte:component
              this={item.icon}
              size={18}
              strokeWidth={1.7}
            /><span>{item.label}</span></a
          >{/each}
      </nav>
      <div class="sidebar-foot">
        <div class="account-row">
          <div class="account-avatar">{$session.name.slice(0, 1)}</div>
          <div class="min-w-0 flex-1">
            <div class="text-xs font-semibold truncate">{$session.name}</div>
            <div class="text-[11px] text-ink-400 mt-1">
              {$session.is_admin ? "基金管理员" : "基金成员"}
            </div>
          </div>
          <button
            class="icon-button"
            on:click={logout}
            disabled={exiting}
            aria-label="退出登录"><LogOut size={16} /></button
          >
        </div>
      </div>
    </aside>
    <div class="main-column">
      <header class="topbar">
        <a class="mobile-brand" href="/"
          >XG <span class="font-normal text-ink-400">fund</span></a
        >
        <div class="desktop-location">
          <span>XG fund</span><span class="text-ink-600">/</span><span
            class="text-ink-200"
            >{navItems.find((n) => n.href === path)?.label}</span
          >
        </div>
        <div class="topbar-right">
          <span class="text-xs text-ink-400">USDT</span><span
            class="topbar-divider"
          ></span><span class="text-xs text-ink-300">{$session.name}</span
          ><button
            class="icon-button mobile-logout"
            on:click={logout}
            disabled={exiting}
            aria-label="退出登录"><LogOut size={16} /></button
          >
        </div>
      </header>
      {#if import.meta.env.VITE_PREVIEW}<div
          class="preview-notice"
          role="status"
        >
          本地预览 · 演示数据 · 操作仅保存在预览内存中
        </div>{/if}
      <main id="main" tabindex="-1"><slot /></main>
      <footer class="page-footer">
        <span>XG fund</span><span
          >NAV 单位法核算 <ArrowUpRight size={12} /></span
        >
      </footer>
    </div>
    {#if navItems.length > 1}<nav aria-label="手机导航" class="mobile-nav">
        {#each navItems as item}<a
            href={item.href}
            class:active={path === item.href}
            aria-current={path === item.href ? "page" : undefined}
            ><svelte:component
              this={item.icon}
              size={20}
              strokeWidth={1.7}
            /><span>{item.label}</span></a
          >{/each}
      </nav>{/if}
  </div>
{:else}<slot />{/if}

<style>
  .boot-state {
    min-height: 100vh;
    display: grid;
    place-content: center;
    justify-items: center;
  }
  .app-shell {
    min-height: 100vh;
  }
  .sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    width: 200px;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--line);
    padding: 28px 16px 18px;
    background: var(--bg);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 9px;
    font-size: 21px;
    font-weight: 700;
  }
  .brand-mark {
    width: 29px;
    height: 29px;
    display: grid;
    place-items: center;
    color: var(--pos);
  }
  .desktop-nav {
    display: grid;
    gap: 7px;
    margin-top: 48px;
  }
  .desktop-nav a {
    display: flex;
    gap: 11px;
    align-items: center;
    padding: 12px;
    border-radius: 5px;
    color: var(--lo);
    font-size: 13px;
    transition:
      color 0.18s,
      background 0.18s;
  }
  .desktop-nav a:hover {
    background: var(--panel);
    color: var(--hi);
  }
  .desktop-nav a.active {
    background: var(--panel2);
    color: var(--hi);
  }
  .desktop-nav a.active :global(svg) {
    color: var(--pos);
  }
  .sidebar-foot {
    margin-top: auto;
    padding-top: 18px;
    border-top: 1px solid var(--line);
  }
  .account-row {
    display: flex;
    gap: 9px;
    align-items: center;
  }
  .account-avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--panel2);
    display: grid;
    place-items: center;
    font-size: 11px;
    color: var(--mid);
    flex: none;
  }
  .account-row :global(.icon-button) {
    width: 28px;
    height: 34px;
  }
  .main-column {
    margin-left: 200px;
  }
  .topbar {
    height: 65px;
    border-bottom: 1px solid var(--line);
    padding: 0 32px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }
  .desktop-location {
    display: flex;
    gap: 18px;
    align-items: center;
    font-size: 11px;
    color: var(--lo);
  }
  .topbar-right {
    display: flex;
    gap: 16px;
    align-items: center;
    min-width: 0;
  }
  .topbar-right > span:last-of-type {
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .topbar-divider {
    height: 13px;
    width: 1px;
    background: var(--line);
  }
  main {
    max-width: 1500px;
    margin: auto;
    padding: 32px 36px 8px;
    outline: none;
  }
  .page-footer {
    max-width: 1500px;
    margin: auto;
    padding: 24px 36px 30px;
    border-top: 1px solid var(--line);
    display: flex;
    justify-content: space-between;
    gap: 16px;
    font-size: 11px;
    color: var(--lo);
  }
  .page-footer span:last-child {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .mobile-brand,
  .mobile-nav,
  .topbar-right :global(.mobile-logout) {
    display: none;
  }
  .preview-notice {
    background: #afd4bf08;
    color: var(--mid);
    border-bottom: 1px solid var(--line);
    padding: 7px 16px;
    text-align: center;
    font-size: 11px;
  }
  .skip-link {
    position: fixed;
    left: 16px;
    top: -60px;
    padding: 10px;
    background: var(--pos);
    color: var(--bg);
    z-index: 50;
  }
  .skip-link:focus {
    top: 12px;
  }
  @media (min-width: 1700px) {
    main {
      padding-top: 40px;
    }
  }
  @media (max-width: 1100px) {
    .sidebar {
      width: 176px;
      padding-inline: 12px;
    }
    .main-column {
      margin-left: 176px;
    }
    main {
      padding-inline: 24px;
    }
  }
  @media (max-width: 767px) {
    .sidebar,
    .desktop-location {
      display: none;
    }
    .main-column {
      margin-left: 0;
      padding-bottom: 70px;
    }
    .topbar {
      padding: 0 18px;
      height: 58px;
    }
    .mobile-brand {
      display: block;
      font-size: 18px;
      font-weight: 700;
    }
    .topbar-right {
      gap: 10px;
    }
    .topbar-right :global(.mobile-logout) {
      display: inline-flex;
      width: 30px;
    }
    main {
      padding: 24px 18px 4px;
    }
    .mobile-nav {
      display: flex;
      position: fixed;
      z-index: 30;
      bottom: 0;
      left: 0;
      right: 0;
      background: var(--bg);
      border-top: 1px solid var(--line);
      padding: 7px 12px max(9px, env(safe-area-inset-bottom));
    }
    .mobile-nav a {
      flex: 1;
      min-height: 46px;
      display: flex;
      flex-direction: column;
      justify-content: center;
      align-items: center;
      gap: 5px;
      font-size: 10px;
      color: var(--lo);
    }
    .mobile-nav a.active {
      color: var(--pos);
    }
    .page-footer {
      padding: 20px 18px 24px;
    }
  }
</style>
