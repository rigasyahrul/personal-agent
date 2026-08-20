<script lang="ts">
  let {
    title,
    meta,
    onclick,
    href,
  }: {
    title: string
    meta: string
    onclick?: () => void
    href?: string
  } = $props()
</script>

{#if href}
  <a class="session-card" {href} {onclick}>
    <h3 class="session-card__title">{title}</h3>
    <p class="session-card__meta">{meta}</p>
  </a>
{:else}
  <button type="button" class="session-card" {onclick}>
    <h3 class="session-card__title">{title}</h3>
    <p class="session-card__meta">{meta}</p>
  </button>
{/if}
