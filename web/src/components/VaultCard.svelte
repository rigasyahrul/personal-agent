<script lang="ts">
  import type { Vault } from '../lib/api/types'
  let { vault, projectCount, onclick }: { vault: Vault; projectCount: number; onclick: () => void } = $props()
</script>
<button type="button" {onclick} aria-label={`Enter ${vault.name} vault`} class="entity-card">
  <h2 class="entity-card__title">{vault.name}</h2>
  <p class="entity-card__meta">{projectCount} {projectCount === 1 ? 'project' : 'projects'}</p>
</button>
