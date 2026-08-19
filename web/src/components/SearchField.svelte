<script lang="ts">
  let { value = $bindable(''), label = 'Search' }: { value?: string; label?: string } = $props()
</script>
<label class="relative block w-full sm:max-w-xs">
  <span class="sr-only">{label}</span>
  <input type="search" {value} oninput={(event) => value = event.currentTarget.value} aria-label={label} placeholder={label} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200" />
</label>
