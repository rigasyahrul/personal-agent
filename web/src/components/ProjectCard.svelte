<script lang="ts">
  import type { Project } from '../lib/api/types'
  import Badge from './Badge.svelte'
  let { project, onclick }: { project: Project; onclick: () => void } = $props()
</script>
<button type="button" {onclick} aria-label={`Open ${project.name}`} class="w-full rounded-xl border border-slate-200 bg-white p-5 text-left hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500">
  <div class="flex items-start justify-between gap-3"><h2 class="font-semibold text-slate-950">{project.name}</h2>{#if project.vault_id && project.vault_name}<Badge text={project.vault_name} />{/if}</div>
  <div class="mt-5 flex flex-wrap gap-3 text-xs text-slate-500">
    <span>{project.note_count} {project.note_count === 1 ? 'note' : 'notes'}</span>
    {#if project.session_count !== undefined}<span>{project.session_count} {project.session_count === 1 ? 'session' : 'sessions'}</span>{/if}
    {#if project.due_count !== undefined}<span>{project.due_count} due</span>{/if}
  </div>
</button>
