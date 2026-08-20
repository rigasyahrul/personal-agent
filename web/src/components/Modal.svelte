<!-- web/src/components/Modal.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import type { Snippet } from 'svelte'

  let {
    open = false,
    title,
    onclose,
    children,
  }: {
    open?: boolean
    title: string
    onclose?: () => void
    children: Snippet
  } = $props()

  let dialogEl = $state<HTMLDialogElement | null>(null)

  $effect(() => {
    if (open) {
      queueMicrotask(() => {
        try {
          // jsdom may lack showModal; callers/tests polyfill, but never throw uncaught.
          if (dialogEl && typeof dialogEl.showModal === 'function') {
            dialogEl.showModal()
          } else if (dialogEl) {
            dialogEl.setAttribute('open', '')
          }
        } catch {
          /* ignore */
        }
      })
    } else {
      try {
        if (dialogEl?.open) dialogEl.close()
      } catch {
        /* ignore */
      }
    }
  })

  onMount(() => {
    return () => {
      try {
        if (dialogEl?.open) dialogEl.close()
      } catch {
        /* ignore */
      }
    }
  })

  function handleClose() {
    onclose?.()
  }
</script>

<dialog
  bind:this={dialogEl}
  class="modal"
  onclose={handleClose}
>
  <div class="modal__chrome">
    <h2 class="modal__title">{title}</h2>
    <div class="modal__body">
      {@render children()}
    </div>
  </div>
</dialog>
