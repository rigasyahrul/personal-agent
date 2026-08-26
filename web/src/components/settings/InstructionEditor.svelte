<!-- web/src/components/settings/InstructionEditor.svelte -->
<script lang="ts">
  import { APIError, api } from '../../lib/api'
  import type { InstructionName } from '../../lib/api'
  import Skeleton from '../Skeleton.svelte'

  const NAMES = ['soul', 'system', 'agents'] as const
  const LABELS: Record<InstructionName, string> = {
    soul: 'SOUL',
    system: 'SYSTEM',
    agents: 'AGENTS',
  }
  const PLACEHOLDERS: Record<InstructionName, string> = {
    soul: 'Identity, values, and voice for this scope.',
    system: 'How the agent should behave in this scope.',
    agents: 'Standing rules. Keep them short. Leave ## Memory pointing at [[memory/lessons]].',
  }

  let {
    scope,
    projectId,
    variant = 'page',
  }: {
    scope: 'global' | 'project'
    projectId?: string
    variant?: 'page' | 'rail'
  } = $props()

  let name = $state<InstructionName>('soul')
  let content = $state('')
  let loaded = $state('')
  let loading = $state(true)
  let saving = $state(false)
  let error = $state('')
  let savedMsg = $state('')
  let seq = 0

  const dirty = $derived(content !== loaded)
  const canSave = $derived(!loading && !saving && dirty)

  async function load(next: InstructionName) {
    const token = ++seq
    loading = true
    error = ''
    savedMsg = ''
    try {
      const file =
        scope === 'project' && projectId
          ? await api.getProjectInstruction(projectId, next)
          : await api.getGlobalInstruction(next)
      if (token !== seq) return
      content = file?.content ?? ''
      loaded = file?.content ?? ''
    } catch (cause) {
      if (token !== seq) return
      if (cause instanceof APIError && cause.status === 404) {
        content = ''
        loaded = ''
        error = ''
      } else {
        content = ''
        loaded = ''
        error = cause instanceof Error ? cause.message : 'Could not load instruction.'
      }
    } finally {
      if (token === seq) loading = false
    }
  }

  $effect(() => {
    void scope
    void projectId
    void name
    void load(name)
  })

  function select(next: InstructionName) {
    if (next === name) return
    name = next
  }

  async function save() {
    if (!canSave) return
    saving = true
    error = ''
    savedMsg = ''
    try {
      const file =
        scope === 'project' && projectId
          ? await api.putProjectInstruction(projectId, name, content)
          : await api.putGlobalInstruction(name, content)
      loaded = file?.content ?? content
      content = file?.content ?? content
      savedMsg = 'Saved.'
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not save instruction.'
    } finally {
      saving = false
    }
  }
</script>

<section
  class="instruction-editor {variant === 'rail' ? 'instruction-editor--rail' : 'panel panel--pad'}"
  aria-label="Instructions"
>
  {#if variant !== 'rail'}
    <header class="instruction-editor__head">
      <h2 class="instruction-editor__title">Instructions</h2>
    </header>
  {/if}

  <div class="instruction-editor__tabs" role="tablist" aria-label="Instruction files">
    {#each NAMES as next (next)}
      <button
        type="button"
        role="tab"
        class="instruction-editor__tab {name === next ? 'instruction-editor__tab--active' : ''}"
        aria-selected={name === next}
        tabindex={name === next ? 0 : -1}
        onclick={() => select(next)}
      >{LABELS[next]}</button>
    {/each}
  </div>

  {#if loading}
    <div class="space-y-2" aria-busy="true">
      <Skeleton class="h-6" />
      <Skeleton class="h-24" />
    </div>
  {:else}
    {#if error}
      <p role="alert" class="alert alert--error">{error}</p>
      <button type="button" class="btn btn--secondary" onclick={() => void load(name)}>Retry</button>
    {/if}

    <label class="instruction-editor__field">
      {#if variant !== 'rail'}
        {LABELS[name]}
      {/if}
      <textarea
        class="field-textarea"
        aria-label={LABELS[name]}
        placeholder={PLACEHOLDERS[name]}
        bind:value={content}
        rows="10"
        disabled={saving}
      ></textarea>
    </label>

    <div class="instruction-editor__actions">
      <button
        type="button"
        class="btn btn--primary"
        disabled={!canSave}
        onclick={() => void save()}
      >Save</button>
      {#if savedMsg}
        <p class="instruction-editor__status" aria-live="polite">{savedMsg}</p>
      {/if}
    </div>
  {/if}
</section>
