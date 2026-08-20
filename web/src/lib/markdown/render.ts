import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: false,
  typographer: false,
})

// Safe external links only: http(s)/mailto + noopener noreferrer
const defaultLinkOpen =
  md.renderer.rules.link_open ??
  ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))

md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const href = token.attrGet('href') ?? ''
  if (!isSafeHref(href)) {
    token.attrSet('href', '')
  }
  const rel = token.attrGet('rel')
  const extra = 'noopener noreferrer'
  token.attrSet('rel', rel ? `${rel} ${extra}` : extra)
  token.attrSet('target', '_blank')
  return defaultLinkOpen(tokens, idx, options, env, self)
}

function isSafeHref(href: string): boolean {
  const trimmed = href.trim()
  if (!trimmed) return false
  // Allow relative paths and fragments
  if (trimmed.startsWith('/') || trimmed.startsWith('#') || trimmed.startsWith('./') || trimmed.startsWith('../')) {
    return true
  }
  try {
    const url = new URL(trimmed, 'https://example.invalid')
    return url.protocol === 'http:' || url.protocol === 'https:' || url.protocol === 'mailto:'
  } catch {
    return false
  }
}

/**
 * Render markdown source to sanitized HTML safe for {@html} injection.
 * Mermaid fences are left as recognizable code blocks (language-mermaid)
 * for the view layer to draw — this helper does not run Mermaid.
 */
export function renderMarkdownToSafeHtml(source: string): string {
  const raw = md.render(source ?? '')
  return DOMPurify.sanitize(raw, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target'],
  })
}
