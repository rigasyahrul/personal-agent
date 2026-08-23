<!-- web/src/routes/SettingsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import BackupSection from '../components/settings/BackupSection.svelte'
  import InstructionEditor from '../components/settings/InstructionEditor.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api'
  import type { Settings } from '../lib/api/types'

  let settings = $state<Settings | null>(null)
  let loading = $state(true)
  let error = $state('')

  async function load() {
    loading = true
    error = ''
    try {
      settings = await api.getSettings()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not load settings.'
    } finally {
      loading = false
    }
  }

  onMount(() => {
    void load()
  })

  function handleSettingsChange(next: Settings) {
    settings = next
  }
</script>

<svelte:head><title>Settings · Personal Agent</title></svelte:head>

<div class="mx-auto max-w-2xl page-stack">
  <header class="page-header">
    <div><h1>Settings</h1></div>
  </header>

  {#if loading}
    <div class="space-y-3">
      <Skeleton class="h-40" />
      <Skeleton class="h-56" />
    </div>
  {:else if error}
    <p role="alert" class="alert alert--error">{error}</p>
    <button type="button" class="btn btn--secondary" onclick={() => load()}>Retry</button>
  {:else if settings}
    <section class="settings-main panel panel--pad form-stack">
      <h2 class="text-lg font-semibold text-slate-950" style="margin:0">Defaults</h2>
      <dl class="grid gap-3 sm:grid-cols-2">
        <div>
          <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">Timezone</dt>
          <dd class="mt-1 text-sm font-medium text-slate-900">{settings.timezone}</dd>
        </div>
        <div>
          <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">Default provider</dt>
          <dd class="mt-1 text-sm font-medium text-slate-900">{settings.default_provider || 'Not set'}</dd>
        </div>
        <div>
          <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">Default model</dt>
          <dd class="mt-1 text-sm font-medium text-slate-900">{settings.default_model_id || 'Not set'}</dd>
        </div>
      </dl>
    </section>

    <InstructionEditor scope="global" />

    <BackupSection {settings} onsettingschange={handleSettingsChange} />
  {/if}
</div>
