<!-- web/src/components/settings/BackupSection.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import { api } from '../../lib/api'
  import type { BackupRun, Settings } from '../../lib/api/types'

  let {
    settings,
    onsettingschange,
  }: {
    settings: Settings
    onsettingschange?: (next: Settings) => void
  } = $props()

  let scheduleOverride = $state<string | null>(null)
  let savingSchedule = $state(false)
  let runningBackup = $state(false)
  let scheduleMsg = $state('')
  let backupMsg = $state('')
  let history = $state<BackupRun[]>([])
  let historyError = $state('')

  const schedule = $derived(
    scheduleOverride ?? settings.backup_schedule ?? settings.backup?.schedule ?? 'off',
  )
  const sinkConfigured = $derived(settings.backup?.sink_configured === true)
  const lastSuccess = $derived(
    settings.backup?.last_success ?? settings.last_success ?? null,
  )
  const lastFailure = $derived(
    settings.backup?.last_failure ?? settings.last_failure ?? null,
  )
  const showFailure = $derived(
    Boolean(
      lastFailure?.completed_at &&
        (!lastSuccess?.completed_at ||
          String(lastFailure.completed_at) > String(lastSuccess.completed_at)),
    ),
  )

  $effect(() => {
    // Reset local override when parent settings change.
    void settings.backup_schedule
    void settings.backup?.schedule
    scheduleOverride = null
  })

  async function loadHistory() {
    historyError = ''
    try {
      const data = await api.listBackups()
      history = data.backups ?? []
    } catch (cause) {
      historyError = cause instanceof Error ? cause.message : 'Could not load backups.'
    }
  }

  onMount(() => {
    void loadHistory()
  })

  async function onScheduleChange(event: Event) {
    const value = (event.currentTarget as HTMLSelectElement).value
    scheduleOverride = value
    savingSchedule = true
    scheduleMsg = 'Saving schedule…'
    backupMsg = ''
    try {
      const next = await api.updateSettings({
        timezone: settings.timezone,
        default_provider: settings.default_provider || '',
        default_model_id: settings.default_model_id || '',
        backup_schedule: value,
      })
      onsettingschange?.(next)
      scheduleMsg = 'Schedule saved.'
    } catch (cause) {
      scheduleMsg = cause instanceof Error ? cause.message : 'Failed to save schedule'
    } finally {
      savingSchedule = false
    }
  }

  async function runBackup() {
    runningBackup = true
    backupMsg = 'Running backup…'
    scheduleMsg = ''
    try {
      await api.createBackup()
      await loadHistory()
      // Refresh settings summary (last success/failure) via parent callback if available
      try {
        const refreshed = await api.getSettings()
        onsettingschange?.(refreshed)
      } catch {
        /* history already refreshed */
      }
      backupMsg = 'Backup completed.'
    } catch (cause) {
      backupMsg = cause instanceof Error ? cause.message : 'Backup failed'
    } finally {
      runningBackup = false
    }
  }
</script>

<section class="settings-backup space-y-4 rounded-xl border border-slate-200 bg-white p-5">
  <h2 class="text-lg font-semibold text-slate-950">Backup</h2>

  <p class="text-sm text-slate-700">
    {#if lastSuccess?.completed_at}
      Last successful backup: {lastSuccess.completed_at}
    {:else}
      Never backed up
    {/if}
  </p>

  {#if showFailure && lastFailure}
    <p class="error text-sm text-red-700">
      Last attempt failed: {lastFailure.error || 'unknown error'}
    </p>
  {/if}

  <p class="muted text-sm text-slate-500">
    Remote sink configured: {sinkConfigured ? 'yes' : 'no'}
  </p>

  <label class="block text-sm font-medium text-slate-800">
    Schedule
    <select
      id="backup-schedule"
      class="mt-1 block w-full max-w-xs rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
      aria-label="Schedule"
      value={schedule}
      disabled={savingSchedule}
      onchange={onScheduleChange}
    >
      <option value="off">Off</option>
      <option value="daily">Daily</option>
    </select>
  </label>
  <p class="text-sm text-slate-600" aria-live="polite">{scheduleMsg}</p>

  <div class="flex flex-wrap items-center gap-3">
    <button
      type="button"
      id="backup-now"
      class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
      disabled={runningBackup}
      onclick={() => runBackup()}
    >Backup now</button>
    <p class="text-sm text-slate-600" aria-live="polite">{backupMsg}</p>
  </div>

  <div class="space-y-2">
    <h3 class="text-sm font-semibold text-slate-800">History</h3>
    {#if historyError}
      <p class="text-sm text-red-700" role="alert">{historyError}</p>
    {:else if history.length === 0}
      <p class="text-sm text-slate-500">No backups yet.</p>
    {:else}
      <ul class="divide-y divide-slate-100 rounded-md border border-slate-200">
        {#each history as run (run.id)}
          <li class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm">
            <span class="font-medium capitalize text-slate-800">{run.status}</span>
            <span class="text-slate-500">{run.completed_at || run.started_at || run.id}</span>
            {#if run.error}
              <span class="w-full text-red-700">{run.error}</span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</section>
