<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, ApiError } from '$lib/api';

  let username = '';
  let password = '';
  let submitting = false;
  let error = '';

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    submitting = true;
    try {
      const me = await api.login(username.trim(), password);
      goto(me.is_admin ? '/admin' : '/');
    } catch (e) {
      if (e instanceof ApiError) error = e.message;
      else error = '登录失败';
    } finally {
      submitting = false;
    }
  }
</script>

<div class="min-h-screen flex items-center justify-center px-6">
  <form
    class="card w-full max-w-sm p-7 space-y-5"
    on:submit={submit}
    autocomplete="off"
  >
    <div class="flex flex-col items-center gap-3 pb-2">
      <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-accent-400 to-accent-600 grid place-items-center">
        <svg viewBox="0 0 24 24" width="23" height="23" fill="var(--bg)" aria-hidden="true">
          <path d="M1.5 20 L8.5 6.5 L12.5 15 L15.5 10.5 L22.5 20 Z" />
        </svg>
      </div>
      <h1 class="text-lg font-semibold tracking-tight">XG fund</h1>
      <p class="text-xs text-ink-400">朋友资金托管账本</p>
    </div>

    <div>
      <label class="label block mb-1.5">用户名</label>
      <input class="input font-sans" type="text" bind:value={username} required autocomplete="username" />
    </div>
    <div>
      <label class="label block mb-1.5">密码</label>
      <input class="input font-sans" type="password" bind:value={password} required autocomplete="current-password" />
    </div>

    {#if error}
      <div class="text-sm text-loss-400 bg-loss-500/10 ring-1 ring-loss-500/20 rounded-lg px-3 py-2">
        {error}
      </div>
    {/if}

    <button class="btn-primary w-full" type="submit" disabled={submitting}>
      {submitting ? '登录中…' : '登录'}
    </button>
  </form>
</div>
