<script lang="ts">
  let { title, description, actionLabel, onaction }: { title: string; description: string; actionLabel: string; onaction: () => void } = $props()
</script>
<section class="rounded-xl border border-dashed border-slate-300 bg-white px-6 py-12 text-center">
  <h2 class="text-base font-semibold text-slate-950">{title}</h2>
  <p class="mx-auto mt-2 max-w-md text-sm text-slate-600">{description}</p>
  <button class="mt-5 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2" type="button" onclick={onaction}>{actionLabel}</button>
</section>
