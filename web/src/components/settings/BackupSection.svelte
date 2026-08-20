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

<section class="settings-backup panel panel--pad form-stack">
  <h2 class="text-lg font-semibold text-slate-950" style="margin:0">Backup</h2>

  <p class="text-sm text-slate-700" style="margin:0">
    {#if lastSuccess?.completed_at}
      Last successful backup: {lastSuccess.completed_at}
    {:else}
      Never backed up
    {/if}
  </p>

  {#if showFailure && lastFailure}
    <p class="error alert alert--error">
      Last attempt failed: {lastFailure.error || 'unknown error'}
    </p>
  {/if}

  <p class="muted text-sm text-slate-500" style="margin:0">
    Remote sink configured: {sinkConfigured ? 'yes' : 'no'}
  </p>

  <label>
    Schedule
    <select
      id="backup-schedule"
      class="field-select"
      aria-label="Schedule"
      value={schedule}
      disabled={savingSchedule}
      onchange={onScheduleChange}
    >
      <option value="off">Off</option>
      <option value="daily">Daily</option>
    </select>
  </label>
  <p class="text-sm text-slate-600" aria-live="polite" style="margin:0">{scheduleMsg}</p>

  <div class="flex flex-wrap items-center gap-3">
    <button
      type="button"
      id="backup-now"
      class="btn btn--primary"
      disabled={runningBackup}
      onclick={() => runBackup()}
    >Backup now</button>
    <p class="text-sm text-slate-600" aria-live="polite" style="margin:0">{backupMsg}</p>
  </div>

  <div class="form-stack">
    <h3 class="text-sm font-semibold text-slate-800" style="margin:0">History</h3>
    {#if historyError}
      <p class="text-sm text-red-700" role="alert">{historyError}</p>
    {:else if history.length === 0}
      <p class="text-sm text-slate-500" style="margin:0">No backups yet.</p>
    {:else}
      <ul class="list-panel">
        {#each history as run (run.id)}
          <li class="list-row" style="cursor:default">
            <span class="font-medium capitalize text-slate-800">{run.status}</span>
            <span class="text-slate-500 text-sm">{run.completed_at || run.started_at || run.id}</span>
            {#if run.error}
              <span class="w-full text-red-700 text-sm">{run.error}</span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</section>
