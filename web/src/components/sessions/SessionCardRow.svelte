<script lang="ts">
  let {
    title,
    meta = '',
    onclick,
    href,
    variant = 'card',
    dateLabel = '',
    onrename,
    ondelete,
  }: {
    title: string
    meta?: string
    onclick?: () => void
    href?: string
    /** card = fat catalog card; list = hub history row */
    variant?: 'card' | 'list'
    dateLabel?: string
    onrename?: () => void
    ondelete?: () => void
  } = $props()

  let menuOpen = $state(false)

  function toggleMenu(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    menuOpen = !menuOpen
  }

  function closeMenu() {
    menuOpen = false
  }

  function handleRename(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    menuOpen = false
    onrename?.()
  }

  function handleDelete(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    menuOpen = false
    ondelete?.()
  }

  function onWindowPointerDown(e: PointerEvent) {
    if (!menuOpen) return
    const t = e.target as Node | null
    const wraps = document.querySelectorAll('.session-row__menu-wrap')
    let inside = false
    wraps.forEach((el) => {
      if (t && el.contains(t)) inside = true
    })
    if (!inside) closeMenu()
  }
</script>

<svelte:window onpointerdown={onWindowPointerDown} />

{#if variant === 'list'}
  <div class="session-row">
    <button type="button" class="session-row__main" {onclick}>
      <span class="session-row__icon" aria-hidden="true">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
      </span>
      <span class="session-row__title">{title}</span>
      {#if dateLabel}
        <span class="session-row__date">{dateLabel}</span>
      {/if}
    </button>
    {#if onrename || ondelete}
      <div class="session-row__menu-wrap">
        <button
          type="button"
          class="session-row__menu-btn"
          aria-label="Session actions"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onclick={toggleMenu}
        >
          <span aria-hidden="true">⋯</span>
        </button>
        {#if menuOpen}
          <div class="session-row__menu" role="menu">
            {#if onrename}
              <button type="button" role="menuitem" class="session-row__menu-item" onclick={handleRename}
              >Rename</button>
            {/if}
            {#if ondelete}
              <button
                type="button"
                role="menuitem"
                class="session-row__menu-item session-row__menu-item--danger"
                onclick={handleDelete}
              >Delete</button>
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{:else if href}
  <a class="session-card" {href} {onclick}>
    <h3 class="session-card__title">{title}</h3>
    {#if meta}
      <p class="session-card__meta">{meta}</p>
    {/if}
  </a>
{:else}
  <button type="button" class="session-card" {onclick}>
    <h3 class="session-card__title">{title}</h3>
    {#if meta}
      <p class="session-card__meta">{meta}</p>
    {/if}
  </button>
{/if}
