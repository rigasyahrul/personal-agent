import test from 'node:test'
import assert from 'node:assert/strict'
import {changedPaths, renderWorkspacePanel, workspaceRows} from './workspace.mjs'

const deferred = () => {
  let resolve, reject
  const promise = new Promise((yes, no) => { resolve = yes; reject = no })
  return {promise, resolve, reject}
}

const decodeHTML = value => value.replaceAll('&quot;', '"').replaceAll('&#39;', "'").replaceAll('&lt;', '<').replaceAll('&gt;', '>').replaceAll('&amp;', '&')

class WorkspaceContainer {
  constructor() { this.html = ''; this.panel = null; this.buttons = []; this.previewWrites = [] }
  set innerHTML(value) {
    this.html = value
    this.panel = value.includes('class="workspace-panel"') ? {preview: null, querySelector: selector => selector === '.workspace-preview' ? this.panel.preview : null} : null
    const previewMarkup = value.match(/<pre class="workspace-preview"([^>]*)>([^<]*)<\/pre>/)
    this.panel && (this.panel.preview = previewMarkup ? {
      tagName: 'PRE',
      attributes: new Map([...previewMarkup[1].matchAll(/([\w-]+)="([^"]*)"/g)].map(match => [match[1], decodeHTML(match[2])])),
      value: decodeHTML(previewMarkup[2]),
      set textContent(content) { this.value = String(content); this.writes.push(String(content)) },
      get textContent() { return this.value },
      getAttribute(name) { return this.attributes.get(name) ?? null },
      writes: this.previewWrites,
    } : null)
    this.buttons = [...value.matchAll(/class="[^"]*workspace-entry--file[^"]*" data-path="([^"]*)"/g)].map(match => ({
      dataset: {path: decodeHTML(match[1])},
      listeners: new Map(),
      addEventListener(type, handler) { this.listeners.set(type, handler) },
    }))
  }
  get innerHTML() { return this.html }
  querySelector(selector) {
    if (selector === '.workspace-panel') return this.panel
    if (selector === '.workspace-preview') return this.panel?.preview || null
    return null
  }
  querySelectorAll(selector) { return selector === '.workspace-entry--file' ? this.buttons : [] }
}

test('workspaceRows escapes labels and marks changed files', () => {
  const html = workspaceRows(
    [{path: 'drafts', kind: 'directory'}, {path: 'drafts/<note>.txt', kind: 'file', size: 5}],
    new Set(['drafts/<note>.txt']),
  )
  assert.match(html, /drafts\/&lt;note&gt;\.txt/)
  assert.match(html, /data-path="drafts\/&lt;note&gt;\.txt"/)
  assert.match(html, /workspace-entry--changed/)
  assert.doesNotMatch(html, /<note>/)
})

test('workspaceRows allowlists kinds used in CSS classes', () => {
  const html = workspaceRows([{path: 'safe.txt', kind: 'file bad" onclick="alert(1)'}], new Set())
  assert.doesNotMatch(html, /onclick|workspace-entry--file bad/)
  assert.match(html, /workspace-entry--file/)
})

test('changedPaths returns only agent tool mutations', () => {
  const messages = [
    {role: 'user', changed_path: 'ignored.txt'},
    {role: 'tool', changed_path: 'draft.md'},
    {role: 'tool', content: '{"changed_path":"notes/raw.txt"}'},
    {role: 'assistant', content: 'done'},
  ]
  assert.deepEqual([...changedPaths(messages)], ['draft.md', 'notes/raw.txt'])
})

test('renderWorkspacePanel renders escaped tree markup and binds real file clicks', async () => {
  const container = new WorkspaceContainer()
  const calls = []
  const api = {
    workspaceTree: async () => ({entries: [{path: 'notes/<draft>&".md', kind: 'file'}]}),
    workspaceFile: async (sessionID, path) => { calls.push([sessionID, path]); return {content: '<script>not markup</script>'} },
  }

  await renderWorkspacePanel({container, sessionID: 'session/id', messages: [], api})
  assert.match(container.innerHTML, /data-path="notes\/&lt;draft&gt;&amp;&quot;\.md"/)
  assert.doesNotMatch(container.innerHTML, /notes\/<draft>/)
  await container.buttons[0].listeners.get('click')()
  assert.deepEqual(calls, [['session/id', 'notes/<draft>&".md']])
  assert.deepEqual(container.previewWrites, ['<script>not markup</script>'])
  assert.equal(container.querySelector('.workspace-preview').textContent, '<script>not markup</script>')
})

test('renderWorkspacePanel reports a selected regular file after preview succeeds', async () => {
  const container = new WorkspaceContainer(), selected = []
  await renderWorkspacePanel({container, sessionID: 's', messages: [], onFileSelected: entry => selected.push(entry), api: {
    workspaceTree: async () => ({entries: [{path: 'draft.md', kind: 'file'}]}),
    workspaceFile: async () => ({content: '# Draft'}),
  }})
  await container.buttons[0].listeners.get('click')()
  assert.deepEqual(selected, [{path: 'draft.md', kind: 'file'}])
})

test('renderWorkspacePanel contains tree and file rejections with accessible concise errors', async () => {
  const treeContainer = new WorkspaceContainer()
  await renderWorkspacePanel({container: treeContainer, sessionID: 's', messages: [], api: {workspaceTree: async () => { throw new Error('secret tree detail') }}})
  assert.match(treeContainer.innerHTML, /role="alert"/)
  assert.match(treeContainer.innerHTML, /Unable to load workspace\./)
  assert.doesNotMatch(treeContainer.innerHTML, /secret tree detail/)

  const fileContainer = new WorkspaceContainer()
  await renderWorkspacePanel({container: fileContainer, sessionID: 's', messages: [], api: {
    workspaceTree: async () => ({entries: [{path: 'bad.txt', kind: 'file'}]}),
    workspaceFile: async () => { throw new Error('secret file detail') },
  }})
  await assert.doesNotReject(fileContainer.buttons[0].listeners.get('click')())
  const errorPreview = fileContainer.querySelector('.workspace-preview')
  assert.equal(errorPreview.tagName, 'PRE')
  assert.equal(errorPreview.getAttribute('aria-live'), 'polite')
  assert.equal(errorPreview.textContent, 'Unable to read file.')
})

test('stale tree rejection cannot overwrite replacement content', async () => {
  const staleTree = deferred()
  const treeContainer = new WorkspaceContainer()
  let current = true
  const treeRender = renderWorkspacePanel({container: treeContainer, sessionID: 'old', messages: [], api: {workspaceTree: () => staleTree.promise}, isCurrent: () => current})
  current = false
  treeContainer.innerHTML = '<section class="replacement">new session</section>'
  staleTree.reject(new Error('stale tree failure'))
  await assert.doesNotReject(treeRender)
  assert.equal(treeContainer.innerHTML, '<section class="replacement">new session</section>')
})

test('stale file rejection cannot overwrite replacement content', async () => {
  const staleFile = deferred()
  const fileContainer = new WorkspaceContainer()
  await renderWorkspacePanel({container: fileContainer, sessionID: 'old', messages: [], api: {
    workspaceTree: async () => ({entries: [{path: 'old.txt', kind: 'file'}]}),
    workspaceFile: () => staleFile.promise,
  }})
  const click = fileContainer.buttons[0].listeners.get('click')()
  fileContainer.innerHTML = '<section class="workspace-panel"><pre class="workspace-preview">replacement</pre></section>'
  staleFile.reject(new Error('stale file failure'))
  await assert.doesNotReject(click)
  assert.deepEqual(fileContainer.previewWrites, [])
  assert.equal(fileContainer.querySelector('.workspace-preview').textContent, 'replacement')
})
