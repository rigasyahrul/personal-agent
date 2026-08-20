// web/src/components/Modal.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ModalHarness from './ModalHarness.svelte'

afterEach(cleanup)

describe('Modal', () => {
  beforeEach(() => {
    // jsdom dialog polyfill (same as PromoteDialog.test.ts)
    if (!HTMLDialogElement.prototype.showModal) {
      HTMLDialogElement.prototype.showModal = function showModal() {
        this.setAttribute('open', '')
      }
      HTMLDialogElement.prototype.close = function close() {
        this.removeAttribute('open')
      }
    }
  })

  it('does not expose a dialog when closed', () => {
    render(ModalHarness, { props: { open: false, title: 'New project' } })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'New project' })).not.toBeInTheDocument()
  })

  it('opens a native dialog with title and children when open', async () => {
    render(ModalHarness, { props: { open: true, title: 'New project' } })
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(dialog.tagName).toBe('DIALOG')
    expect(dialog).toHaveClass('modal')
    expect(screen.getByRole('heading', { name: 'New project' })).toBeInTheDocument()
    expect(screen.getByText('Harness body')).toBeInTheDocument()
  })

  it('calls onclose from Cancel', async () => {
    const onclose = vi.fn()
    render(ModalHarness, { props: { open: true, title: 'New vault', onclose } })
    await screen.findByRole('dialog')
    await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onclose).toHaveBeenCalledTimes(1)
  })
})
