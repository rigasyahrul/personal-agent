// web/src/components/Breadcrumbs.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import Breadcrumbs from './Breadcrumbs.svelte'

afterEach(cleanup)

const vaultedProject = {
  id: 'p1',
  name: 'Sleep',
  vault_id: 'v1',
  vault_name: 'HEALTH',
  note_count: 0,
}

const unfiledProject = {
  id: 'p2',
  name: 'Inbox',
  note_count: 0,
}

describe('Breadcrumbs', () => {
  it('links a vaulted project back through its vault', () => {
    render(Breadcrumbs, { props: { project: vaultedProject, leaf: 'Sessions' } })
    expect(screen.getByRole('link', { name: 'Vaults' })).toHaveAttribute('href', '#/vaults')
    expect(screen.getByRole('link', { name: 'HEALTH' })).toHaveAttribute('href', '#/vaults/v1')
    expect(screen.getByRole('link', { name: 'Sleep' })).toHaveAttribute('href', '#/projects/p1')
    expect(screen.getByText('Sessions')).toHaveAttribute('aria-current', 'page')
  })

  it('links an unfiled project through global projects', () => {
    render(Breadcrumbs, { props: { project: unfiledProject, leaf: 'Notes' } })
    expect(screen.getByRole('link', { name: 'Projects' })).toHaveAttribute('href', '#/projects')
    expect(screen.getByRole('link', { name: 'Inbox' })).toHaveAttribute('href', '#/projects/p2')
    expect(screen.getByText('Notes')).toHaveAttribute('aria-current', 'page')
    expect(screen.queryByText('Vaults')).not.toBeInTheDocument()
  })

  it('marks the project as current when there is no leaf', () => {
    render(Breadcrumbs, { props: { project: vaultedProject } })
    expect(screen.getByText('Sleep')).toHaveAttribute('aria-current', 'page')
  })

  it('invokes onProjectClick when project crumb is clicked with a leaf', async () => {
    const onProjectClick = vi.fn()
    render(Breadcrumbs, {
      props: { project: vaultedProject, leaf: 'Chat', onProjectClick },
    })
    await fireEvent.click(screen.getByRole('link', { name: 'Sleep' }))
    expect(onProjectClick).toHaveBeenCalledTimes(1)
  })
})
