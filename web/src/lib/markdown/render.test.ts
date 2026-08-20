import { describe, expect, it } from 'vitest'
import { renderMarkdownToSafeHtml } from './render'

describe('renderMarkdownToSafeHtml', () => {
  it('renders headings and lists', () => {
    const html = renderMarkdownToSafeHtml('# Title\n\n- a\n- b')
    expect(html).toContain('<h1>')
    expect(html).toContain('<li>')
  })
  it('strips script tags', () => {
    const html = renderMarkdownToSafeHtml('ok <script>alert(1)</script>')
    expect(html.toLowerCase()).not.toContain('<script')
  })
  it('keeps fenced code', () => {
    const html = renderMarkdownToSafeHtml('```js\nconst x = 1\n```')
    expect(html).toContain('<code')
  })
  it('marks mermaid fences for the view layer', () => {
    const html = renderMarkdownToSafeHtml('```mermaid\ngraph TD; A-->B\n```')
    expect(html.includes('language-mermaid') || html.includes('data-mermaid')).toBe(true)
  })
})
