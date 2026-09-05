<script lang="ts">
  import { goto } from "$app/navigation";
  import {
    ChartNoAxesCombined,
    ArrowRight,
    Eye,
    EyeOff,
    LockKeyhole,
  } from "lucide-svelte";
  import { api, ApiError } from "$lib/api";
  import { session } from "$lib/session";
  let username = "",
    password = "",
    error = "";
  let submitting = false,
    showPassword = false;
  async function submit(event: SubmitEvent) {
    event.preventDefault();
    error = "";
    submitting = true;
    try {
      $session = await api.login(username.trim(), password);
      await goto("/");
    } catch (e) {
      error = e instanceof ApiError ? e.message : "登录失败，请稍后重试";
    } finally {
      submitting = false;
    }
  }
</script>

<div class="login-page">
  <header>
    <a href="/" class="login-brand"
      ><ChartNoAxesCombined size={23} strokeWidth={1.5} /><span
        >XG <span class="text-ink-400 font-normal">fund</span></span
      ></a
    ><span class="text-xs text-ink-400">基金账户</span>
  </header>
  <main class="login-main">
    <div class="login-content">
      <div class="login-symbol">
        <LockKeyhole size={22} strokeWidth={1.5} />
      </div>
      <h1>登录 XG fund</h1>
      {#if import.meta.env.VITE_PREVIEW}<p class="preview-message">
          本地演示 · 任意用户名和密码进入管理员视图；用户名 member
          进入成员视图。
        </p>{/if}
      <form on:submit={submit} class="login-form">
        <div>
          <label for="username" class="label block mb-2">用户名</label><input
            id="username"
            class="input"
            type="text"
            bind:value={username}
            required
            autocomplete="username"
            autocapitalize="none"
            spellcheck="false"
          />
        </div>
        <div>
          <label for="password" class="label block mb-2">密码</label>
          <div class="password-field">
            <input
              id="password"
              class="input"
              type={showPassword ? "text" : "password"}
              value={password}
              on:input={(e) => (password = e.currentTarget.value)}
              required
              autocomplete="current-password"
            /><button
              type="button"
              class="icon-button"
              aria-label={showPassword ? "隐藏密码" : "显示密码"}
              aria-pressed={showPassword}
              on:click={() => (showPassword = !showPassword)}
              >{#if showPassword}<EyeOff size={17} />{:else}<Eye
                  size={17}
                />{/if}</button
            >
          </div>
        </div>
        {#if error}<div class="alert" role="alert">{error}</div>{/if}<button
          class="btn-primary w-full justify-between"
          type="submit"
          disabled={submitting}
          ><span>{submitting ? "正在登录" : "登录账户"}</span><ArrowRight
            size={16}
          /></button
        >
      </form>
    </div>
  </main>
  <footer><span>XG fund</span><span>账户访问受限</span></footer>
</div>

<style>
  .login-page {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
  header,
  footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 26px 40px;
    gap: 16px;
  }
  header {
    border-bottom: 1px solid var(--line);
  }
  .login-brand {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 22px;
    font-weight: 650;
  }
  .login-brand :global(svg) {
    color: var(--pos);
  }
  .login-main {
    flex: 1;
    display: grid;
    place-items: center;
    padding: 64px 24px 90px;
  }
  .login-content {
    width: 100%;
    max-width: 340px;
  }
  .login-symbol {
    color: var(--pos);
    margin-bottom: 24px;
  }
  .login-form {
    display: grid;
    gap: 22px;
    margin-top: 34px;
  }
  .password-field {
    position: relative;
  }
  .password-field :global(input) {
    padding-right: 46px;
  }
  .password-field :global(button) {
    position: absolute;
    top: 2px;
    right: 3px;
  }
  .login-form > :global(button) {
    min-height: 46px;
    margin-top: 5px;
  }
  footer {
    color: var(--lo);
    font-size: 11px;
    border-top: 1px solid var(--line);
  }
  .preview-message {
    margin-top: 18px;
    font-size: 11px;
    line-height: 1.7;
    color: var(--benchmark-b);
  }
  @media (max-width: 600px) {
    header,
    footer {
      padding: 22px 24px;
    }
    .login-main {
      padding-block: 54px 70px;
    }
  }
</style>
