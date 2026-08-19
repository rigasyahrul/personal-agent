<!-- web/src/routes/VaultsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import SearchField from '../components/SearchField.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import VaultCard from '../components/VaultCard.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Vault } from '../lib/api/types'
  import { filterByQuery } from '../lib/catalog'
  import { navigate } from '../lib/router'
  let vaults = $state<Vault[]>([]), counts = $state<Record<string, number>>({}), query = $state(''), loading = $state(true), creating = $state(false), saving = $state(false), name = $state(''), error = $state('')
  let visible = $derived(filterByQuery(vaults, query))
  onMount(async () => {
    try {
      const [listed, home] = await Promise.all([
        api.get<Vault[]>('/api/v1/vaults'),
        api.get<HomeResponse>('/api/v1/home'),
      ])
      vaults = listed
      counts = home.projects.reduce<Record<string, number>>((all, p) => {
        if (p.vault_id) all[p.vault_id] = (all[p.vault_id] ?? 0) + 1
        return all
      }, {})
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not load vaults.'
    } finally {
      loading = false
    }
  })
  async function createVault() {
    const clean = name.trim()
    if (!clean) return
    saving = true
    error = ''
    try {
      const vault = await api.post<Vault>('/api/v1/vaults', { name: clean })
      navigate(`#/vaults/${encodeURIComponent(vault.id)}`)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not create vault.'
    } finally {
      saving = false
    }
  }
</script>
<svelte:head><title>Vaults · Personal Agent</title></svelte:head>
<div class="space-y-6">
  <header class="flex flex-wrap items-end justify-between gap-4">
    <div><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold">Vaults</h1></div>
    <button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => creating = true}>New vault</button>
  </header>
  <SearchField bind:value={query} label="Search vaults" />
  {#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if creating}
    <form class="flex max-w-lg gap-2 rounded-xl border bg-white p-4" onsubmit={(e) => { e.preventDefault(); createVault() }}>
      <label class="flex-1">
        <span class="text-sm font-medium">Vault name</span>
        <input class="mt-1 w-full rounded-md border px-3 py-2" bind:value={name} />
      </label>
      <button disabled={saving || !name.trim()} class="self-end rounded-md bg-indigo-600 px-4 py-2 text-sm text-white" type="submit">Create vault</button>
    </form>
  {/if}
  {#if loading}
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else if visible.length}
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {#each visible as vault (vault.id)}
        <VaultCard {vault} projectCount={counts[vault.id] ?? 0} onclick={() => navigate(`#/vaults/${encodeURIComponent(vault.id)}`)} />
      {/each}
    </div>
  {:else if query.trim()}
    <EmptyState title="No matching vaults" description="Try a different vault name." actionLabel="Clear search" onaction={() => query = ''} />
  {:else}
    <EmptyState title="No vaults yet" description="Create a vault to organize related projects." actionLabel="New vault" onaction={() => creating = true} />
  {/if}
</div>
