<!-- web/src/routes/auth/BootstrapPage.svelte -->
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

  let token = $state('');
  let password = $state('');
  let error = $state('');
  let pending = $state(false);

  async function handleSubmit() {
    pending = true;
    error = '';
    try {
      await submit('/api/v1/setup/bootstrap', 'POST', { token, password });
      await onComplete();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Setup failed';
      pending = false;
    }
  }
</script>

<AuthCard>
  <h1>Set up your owner account</h1>
  <form
    aria-label="Set up owner account"
    onsubmit={(event) => {
      event.preventDefault();
      void handleSubmit();
    }}
  >
    <label>
      Bootstrap token
      <input bind:value={token} required autocomplete="off" />
    </label>
    <label>
      Password
      <input bind:value={password} type="password" minlength="12" required autocomplete="new-password" />
    </label>
    {#if error}<p role="alert">{error}</p>{/if}
    <button type="submit" disabled={pending}>Continue</button>
  </form>
</AuthCard>
