<!-- web/src/routes/VaultReviewPage.svelte -->
<script lang="ts">
  import ReviewRunner from '../components/review/ReviewRunner.svelte'
  import { api } from '../lib/api'
  import type { HomeResponse, ReviewQueue } from '../lib/api/types'
  import { filterVaultProjects } from '../lib/vault-scope'
  import { filterQueueByProjectIds } from '../lib/review/vault-filter'

  let {
    vaultId,
    vaultName = 'Vault',
  }: {
    vaultId: string
    vaultName?: string
  } = $props()

  async function loadQueue(): Promise<ReviewQueue> {
    const [home, full] = await Promise.all([
      api.get<HomeResponse>('/api/v1/home'),
      api.getReviewQueue('all'),
    ])
    const projects = filterVaultProjects(home.projects, vaultId)
    const ids = new Set(projects.map((project) => project.id))
    return filterQueueByProjectIds(full, ids)
  }
</script>

<svelte:head><title>Review · {vaultName}</title></svelte:head>

<div class="space-y-6">
  <header>
    <p class="text-sm text-slate-500">{vaultName}</p>
    <h1 class="text-2xl font-semibold">Review</h1>
  </header>

  <ReviewRunner scope="all" showScopeChips={false} {loadQueue} />
</div>
