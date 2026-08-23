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

// Path wikilinks: [[target|alias]] — same matcher as internal/knowledge/wikilink.go
const wikilinkRE = /^\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]/

md.inline.ruler.before('link', 'wikilink', (state, silent) => {
  if (state.src.charCodeAt(state.pos) !== 0x5b /* [ */) return false
  const slice = state.src.slice(state.pos, state.posMax)
  const match = wikilinkRE.exec(slice)
  if (!match) return false

  const raw = match[1].trim()
  const normalized = normalizeWikilinkTarget(raw)
  if (!normalized) return false

  const alias = (match[2] ?? '').trim()
  if (!silent) {
    const token = state.push('wikilink', 'a', 0)
    token.attrSet('href', '#')
    token.attrSet('class', 'wikilink')
    token.attrSet('data-path', normalized)
    token.content = alias || raw
    token.markup = '[['
  }

  state.pos += match[0].length
  return true
})

md.renderer.rules.wikilink = (tokens, idx) => {
  const token = tokens[idx]
  const path = escapeAttr(token.attrGet('data-path') ?? '')
  const text = escapeHtml(token.content)
  return `<a class="wikilink" href="#" data-path="${path}">${text}</a>`
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

function escapeAttr(value: string): string {
  return escapeHtml(value).replaceAll("'", '&#39;')
}

/** Match server NormalizeWikilinkTarget: .md suffix, bare AGENTS/SOUL/SYSTEM, skip .. / empty / abs. */
function normalizeWikilinkTarget(target: string): string | null {
  target = target.trim()
  if (!target) return null
  if (target.includes('\0')) return null
  if (target.includes('..')) return null
  if (target.startsWith('/') || target.startsWith('\\') || /^[a-zA-Z]:[\\/]/.test(target)) {
    return null
  }
  if (target === 'AGENTS' || target === 'SOUL' || target === 'SYSTEM') {
    return `${target}.md`
  }
  if (!target.endsWith('.md')) {
    target += '.md'
  }
  return target
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
    ALLOW_DATA_ATTR: false,
    ADD_ATTR: ['target', 'data-path', 'class'],
  })
}
