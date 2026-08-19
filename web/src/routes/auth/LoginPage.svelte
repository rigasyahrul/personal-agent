<!-- web/src/routes/auth/LoginPage.svelte -->
<script lang="ts">
  import { mutate } from '../../lib/api/client';
  import AuthCard from './AuthCard.svelte';

  let {
    submit = mutate,
    onComplete,
  }: {
    submit?: typeof mutate;
    onComplete: () => void | Promise<void>;
  } = $props();

  let password = $state('');
  let error = $state('');
  let pending = $state(false);

  async function handleSubmit() {
    pending = true;
    error = '';
    try {
      await submit('/api/v1/auth/login', 'POST', { password });
      await onComplete();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Sign in failed';
      pending = false;
    }
  }
</script>

<AuthCard>
  <h1>Sign in</h1>
  <form
    aria-label="Sign in"
    onsubmit={(event) => {
      event.preventDefault();
      void handleSubmit();
    }}
  >
    <label>
      Password
      <input bind:value={password} type="password" required autocomplete="current-password" />
    </label>
    {#if error}<p role="alert">{error}</p>{/if}
    <button type="submit" disabled={pending}>Continue</button>
  </form>
</AuthCard>
