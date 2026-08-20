<!-- web/src/components/ModalHarness.svelte -->
<script lang="ts">
  import Modal from './Modal.svelte'

  let {
    open = false,
    title = 'Test modal',
    onclose,
  }: {
    open?: boolean
    title?: string
    onclose?: () => void
  } = $props()
</script>

<Modal {open} {title} {onclose}>
  <p>Harness body</p>
  <button type="button" class="btn btn--secondary" onclick={() => onclose?.()}>Cancel</button>
</Modal>
