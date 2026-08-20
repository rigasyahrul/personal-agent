<script lang="ts">
  let { title, description, actionLabel, onaction }: { title: string; description: string; actionLabel: string; onaction: () => void } = $props()
</script>
<section class="rounded-lg border border-dashed border-slate-300 bg-white px-6 py-10 text-center">
  <h2 class="text-base font-semibold text-slate-950">{title}</h2>
  <p class="mx-auto mt-2 max-w-md text-sm text-slate-600">{description}</p>
  <button class="btn btn--primary mt-5" type="button" onclick={onaction}>{actionLabel}</button>
</section>
