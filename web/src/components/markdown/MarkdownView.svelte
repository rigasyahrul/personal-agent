<!-- web/src/components/markdown/MarkdownView.svelte -->
<script lang="ts">
  import { renderMarkdownToSafeHtml } from '../../lib/markdown/render'

  let { source }: { source: string } = $props()

  let rootEl: HTMLElement | undefined = $state()
  let html = $derived(renderMarkdownToSafeHtml(source ?? ''))

  let mermaidReady: Promise<typeof import('mermaid')> | null = null

  function loadMermaid() {
    if (!mermaidReady) {
      mermaidReady = import('mermaid').then((mod) => {
        mod.default.initialize({
          startOnLoad: false,
          securityLevel: 'strict',
        })
        return mod
      })
    }
    return mermaidReady
  }

  function showFallback(target: Element, text: string) {
    const pre = document.createElement('pre')
    pre.className = 'mermaid-fallback'
    pre.textContent = text
    target.replaceWith(pre)
  }

  async function renderMermaid(root: HTMLElement) {
    const blocks = Array.from(root.querySelectorAll('code.language-mermaid')) as HTMLElement[]
    if (blocks.length === 0) return

    let mermaidMod: typeof import('mermaid')
    try {
      mermaidMod = await loadMermaid()
    } catch {
      for (const code of blocks) {
        showFallback(code.closest('pre') ?? code, code.textContent ?? '')
      }
      return
    }

    const mermaid = mermaidMod.default

    for (const code of blocks) {
      // Skip if this node was already replaced (stale query after re-render)
      if (!code.isConnected) continue

      const text = code.textContent ?? ''
      const host = document.createElement('div')
      host.className = 'mermaid'
      host.textContent = text
      const replaceTarget = code.closest('pre') ?? code
      replaceTarget.replaceWith(host)

      try {
        await mermaid.run({ nodes: [host] })
      } catch {
        if (host.isConnected) {
          showFallback(host, text)
        } else {
          const pre = document.createElement('pre')
          pre.className = 'mermaid-fallback'
          pre.textContent = text
          root.appendChild(pre)
        }
      }
    }
  }

  // Re-run when source/html changes and root is mounted
  $effect(() => {
    void html
    const el = rootEl
    if (!el) return

    const id = requestAnimationFrame(() => {
      void renderMermaid(el)
    })
    return () => cancelAnimationFrame(id)
  })
</script>

<div class="markdown-view prose prose-slate max-w-none" bind:this={rootEl}>
  {@html html}
</div>
