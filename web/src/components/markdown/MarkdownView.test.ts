// web/src/components/markdown/MarkdownView.test.ts
import { cleanup, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import MarkdownView from './MarkdownView.svelte'

vi.mock('mermaid', () => {
  return {
    default: {
      initialize: vi.fn(),
      run: vi.fn(async ({ nodes }: { nodes: ArrayLike<Element> }) => {
        for (const node of Array.from(nodes)) {
          const text = node.textContent ?? ''
          if (text.includes('not valid mermaid') || text.includes('INVALID')) {
            throw new Error('Parse error')
          }
          // Simulate successful diagram render
          node.innerHTML = '<svg data-testid="mermaid-svg"></svg>'
        }
      }),
    },
  }
})

afterEach(cleanup)

describe('MarkdownView', () => {
  it('renders heading text from markdown source', () => {
    render(MarkdownView, { props: { source: '# Hello World\n\nSome paragraph.' } })
    expect(screen.getByRole('heading', { level: 1, name: 'Hello World' })).toBeInTheDocument()
    expect(screen.getByText('Some paragraph.')).toBeInTheDocument()
  })

  it('does not throw when source has invalid mermaid and shows fallback', async () => {
    const source = ['# Diagram', '', '```mermaid', 'not valid mermaid syntax INVALID', '```'].join(
      '\n',
    )

    expect(() => {
      render(MarkdownView, { props: { source } })
    }).not.toThrow()

    await waitFor(() => {
      const fallback = document.querySelector('pre.mermaid-fallback')
      expect(fallback).toBeTruthy()
      expect(fallback?.textContent).toContain('not valid mermaid')
    })
  })
})
