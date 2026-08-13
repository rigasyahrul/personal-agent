const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, char => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[char]))

export function changedPaths(messages = []) {
  const paths = new Set()
  for (const message of messages) {
    if (message?.role !== 'tool') continue
    let path = message.changed_path
    if (!path && typeof message.content === 'string') {
      try { path = JSON.parse(message.content)?.changed_path } catch { path = '' }
    }
    if (typeof path === 'string' && path) paths.add(path)
  }
  return paths
}

export function workspaceRows(entries = [], changed = new Set()) {
  return entries.map(entry => {
    const path = escapeHTML(entry?.path)
    const kind = entry?.kind === 'directory' ? 'directory' : 'file'
    const changedClass = changed.has(entry?.path) ? ' workspace-entry--changed' : ''
    const disabled = kind === 'directory' ? ' disabled' : ''
    return `<button type="button" class="workspace-entry workspace-entry--${kind}${changedClass}" data-path="${path}"${disabled}>${path}</button>`
  }).join('')
}

export async function renderWorkspacePanel({container, sessionID, messages, api, isCurrent = () => true, onFileSelected = () => {}}) {
  try {
    const tree = await api.workspaceTree(sessionID)
    if (!isCurrent()) return
    container.innerHTML = `<section class="workspace-panel"><h2>Workspace files</h2><div class="workspace-tree">${workspaceRows(tree?.entries, changedPaths(messages))}</div><pre class="workspace-preview" aria-live="polite">Select a file</pre></section>`
    const panel = container.querySelector('.workspace-panel')
    container.querySelectorAll('.workspace-entry--file').forEach(button => button.addEventListener('click', async () => {
      const preview = panel?.querySelector('.workspace-preview') || container.querySelector('.workspace-preview')
      try {
        const file = await api.workspaceFile(sessionID, button.dataset.path)
        if (isCurrent() && container.querySelector('.workspace-panel') === panel) { preview.textContent = file?.content ?? ''; onFileSelected({path:button.dataset.path,kind:'file'}) }
      } catch {
        if (isCurrent() && container.querySelector('.workspace-panel') === panel) preview.textContent = 'Unable to read file.'
      }
    }))
  } catch {
    if (isCurrent()) container.innerHTML = '<section class="workspace-panel"><h2>Workspace files</h2><p class="error" role="alert">Unable to load workspace.</p></section>'
  }
}
