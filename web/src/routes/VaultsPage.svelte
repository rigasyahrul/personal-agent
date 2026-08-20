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
<div class="page-stack">
  <header class="page-header">
    <div><h1>Vaults</h1></div>
    <div class="page-header__actions">
      <button type="button" class="btn btn--primary" onclick={() => creating = true}>New vault</button>
    </div>
  </header>
  <SearchField bind:value={query} label="Search vaults" />
  {#if error}<p role="alert" class="alert alert--error">{error}</p>{/if}
  {#if creating}
    <form class="panel form-inline" onsubmit={(e) => { e.preventDefault(); createVault() }}>
      <label>
        Vault name
        <input class="field-input" bind:value={name} aria-label="Vault name" />
      </label>
      <button disabled={saving || !name.trim()} class="btn btn--primary" type="submit">Create vault</button>
    </form>
  {/if}
  {#if loading}
    <div class="catalog-grid" aria-busy="true"><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else if visible.length}
    <div class="catalog-grid">
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
