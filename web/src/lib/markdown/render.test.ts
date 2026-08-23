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

  it('renders title-mask wikilink with normalized data-path', () => {
    const html = renderMarkdownToSafeHtml('See [[memory/a|Title]] for the lesson.')
    const link = parseWikilink(html)
    expect(link).not.toBeNull()
    expect(link?.textContent).toBe('Title')
    expect(link?.getAttribute('class')).toBe('wikilink')
    expect(link?.getAttribute('data-path')).toBe('memory/a.md')
  })

  it('renders bare wikilink with path text and appends .md to data-path', () => {
    const html = renderMarkdownToSafeHtml('Open [[memory/a]] next.')
    const link = parseWikilink(html)
    expect(link).not.toBeNull()
    expect(link?.textContent).toBe('memory/a')
    expect(link?.getAttribute('data-path')).toBe('memory/a.md')
  })

  it('maps bare AGENTS to AGENTS.md and keeps existing .md suffix', () => {
    const agents = parseWikilink(renderMarkdownToSafeHtml('[[AGENTS]]'))
    expect(agents?.textContent).toBe('AGENTS')
    expect(agents?.getAttribute('data-path')).toBe('AGENTS.md')

    const sourced = parseWikilink(renderMarkdownToSafeHtml('[[source/x.md]]'))
    expect(sourced?.textContent).toBe('source/x.md')
    expect(sourced?.getAttribute('data-path')).toBe('source/x.md')
  })

  it('skips empty and parent-traversal wikilink targets', () => {
    const html = renderMarkdownToSafeHtml('keep [[memory/a|Title]] skip [[../x]] skip [[]] skip [[/abs]]')
    const paths = [...parseHtml(html).querySelectorAll('a.wikilink')].map((el) =>
      el.getAttribute('data-path'),
    )
    expect(paths).toEqual(['memory/a.md'])
    expect(html).toContain('[[../x]]')
    expect(html).toContain('[[]]')
  })

  it('keeps data-path and class through DOMPurify and does not navigate externally', () => {
    const html = renderMarkdownToSafeHtml('[[memory/a|Title]]')
    const link = parseWikilink(html)
    expect(link?.getAttribute('data-path')).toBe('memory/a.md')
    expect(link?.classList.contains('wikilink')).toBe(true)
    const href = link?.getAttribute('href') ?? ''
    expect(href === '#' || href === 'javascript:void(0)' || href === 'javascript:void(0);').toBe(true)
    expect(link?.getAttribute('target')).not.toBe('_blank')
  })

  it('does not inject raw HTML from a hostile wikilink target', () => {
    const html = renderMarkdownToSafeHtml('[[<script>alert(1)</script>]]')
    expect(html.toLowerCase()).not.toContain('<script')
    const doc = parseHtml(html)
    expect(doc.querySelector('script')).toBeNull()
    const link = doc.querySelector('a.wikilink')
    if (link) {
      expect(link.getAttribute('data-path') ?? '').not.toMatch(/<script/i)
    }
  })
})

function parseHtml(html: string): Document {
  return new DOMParser().parseFromString(html, 'text/html')
}

function parseWikilink(html: string): HTMLAnchorElement | null {
  return parseHtml(html).querySelector('a.wikilink')
}
