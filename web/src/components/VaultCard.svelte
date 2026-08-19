<script lang="ts">
  import type { Vault } from '../lib/api/types'
  let { vault, projectCount, onclick }: { vault: Vault; projectCount: number; onclick: () => void } = $props()
</script>
<button type="button" {onclick} aria-label={`Enter ${vault.name} vault`} class="w-full rounded-xl border border-slate-200 bg-white p-5 text-left hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500">
  <h2 class="font-semibold text-slate-950">{vault.name}</h2>
  <p class="mt-5 text-xs text-slate-500">{projectCount} {projectCount === 1 ? 'project' : 'projects'}</p>
</button>
