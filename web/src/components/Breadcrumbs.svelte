<!-- web/src/components/Breadcrumbs.svelte -->
<script lang="ts">
  import type { Project } from '../lib/api/types'
  import { isUnfiled } from '../lib/catalog'

  let {
    project,
    leaf,
  }: {
    project: Project
    leaf?: string
  } = $props()

  const vaulted = $derived(!isUnfiled(project) && Boolean(project.vault_id))
  const vaultHref = $derived(
    vaulted ? `#/vaults/${encodeURIComponent(project.vault_id!)}` : '',
  )
  const projectHref = $derived(`#/projects/${encodeURIComponent(project.id)}`)
</script>

<nav aria-label="Breadcrumb" class="text-sm text-slate-600">
  <ol class="flex flex-wrap items-center gap-1">
    {#if vaulted}
      <li class="flex items-center gap-1">
        <a class="link-accent" href="#/vaults">Vaults</a>
        <span aria-hidden="true" class="text-slate-400">/</span>
      </li>
      <li class="flex min-w-0 items-center gap-1">
        <a
          class="link-accent max-w-[12rem] truncate"
          href={vaultHref}
          title={project.vault_name ?? 'Vault'}
        >{project.vault_name ?? 'Vault'}</a>
        <span aria-hidden="true" class="text-slate-400">/</span>
      </li>
    {:else}
      <li class="flex items-center gap-1">
        <a class="link-accent" href="#/projects">Projects</a>
        <span aria-hidden="true" class="text-slate-400">/</span>
      </li>
    {/if}

    <li class="flex min-w-0 items-center gap-1">
      {#if leaf}
        <a
          class="link-accent max-w-[12rem] truncate"
          href={projectHref}
          title={project.name}
        >{project.name}</a>
        <span aria-hidden="true" class="text-slate-400">/</span>
      {:else}
        <span class="max-w-[16rem] truncate font-medium text-slate-900" aria-current="page" title={project.name}
        >{project.name}</span>
      {/if}
    </li>

    {#if leaf}
      <li class="min-w-0">
        <span class="max-w-[12rem] truncate font-medium text-slate-900" aria-current="page">{leaf}</span>
      </li>
    {/if}
  </ol>
</nav>
