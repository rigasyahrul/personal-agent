import test from 'node:test'
import assert from 'node:assert/strict'
import {createSessionsPage} from './sessions.js'
import {parseRoute} from '../router.js'

class Root {
  constructor() { this.renders = []; this.innerHTML = '' }
  set innerHTML(value) {
    this.renders.push(value)
    this.html = value
    this.chatForm = value.includes('data-chat') ? {elements: {message: {value: this.chatForm?.elements.message.value || ''}}} : null
    this.newForm = value.includes('data-new') ? {elements: {title: {value: ''}, model: {value: '0'}, workspace_files: {checked: false}}} : null
    this.back = value.includes('data-back') ? {
      listeners: new Map(),
      addEventListener: (type, handler) => { this.back.listeners.set(type, handler) },
    } : null
    this.sendButton = value.includes('data-chat') ? {disabled: /<button disabled>Send<\/button>/.test(value)} : null
    this.workspace = value.includes('data-workspace-panel') ? {} : null
    this.sessionButtons = [...value.matchAll(/data-session="([^"]+)"/g)].map(match => ({dataset: {session: match[1]}}))
  }
  get innerHTML() { return this.html }
  get textContent() { return this.html.replace(/<[^>]*>/g, ' ').replaceAll('&lt;', '<').replaceAll('&gt;', '>').replaceAll('&amp;', '&') }
  querySelector(selector) {
    if (selector === 'form[data-chat]') return this.chatForm
    if (selector === 'form[data-new]') return this.newForm
    if (selector === '[name=message]') return this.chatForm?.elements.message
    if (selector === '[data-back]') return this.back
    if (selector === 'form[data-chat] button') return this.sendButton
    if (selector === '[data-workspace-panel]') return this.workspace
    return null
  }
  querySelectorAll(selector) { return selector === '[data-session]' ? this.sessionButtons : [] }
}

const deferred = () => {
  let resolve, reject
  const promise = new Promise((yes, no) => { resolve = yes; reject = no })
  return {promise, resolve, reject}
}

test('chat submit posts once with one stable key while polling and preserves history and draft after failure', async () => {
  const calls = []
  let interval
  const api = async (path, options = {}) => {
    calls.push([path, options])
    if (path.endsWith('/messages') && options.method === 'POST') throw new Error('AI unavailable <detail>')
    if (path.endsWith('/messages')) return [{sequence: 1, role: 'user', content: '<b>kept</b>'}]
    if (path.endsWith('/runs/current')) return null
    return []
  }
  const root = new Root()
  const page = createSessionsPage({root, api, projectID: 'p1', randomUUID: () => 'stable-key', setInterval: fn => { interval = fn; return 7 }, clearInterval() {}})
  await page.openChat({id: 's1', title: '<Chat>', provider: 'openai', model_id: 'm:latest'})
  root.querySelector('[name=message]').value = 'hello draft'
  const submit = root.querySelector('form[data-chat]').onsubmit({preventDefault() {}})
  await Promise.all([submit, root.querySelector('form[data-chat]').onsubmit({preventDefault() {}})])
  await interval(); await page.poll()
  const posts = calls.filter(([path, options]) => path.endsWith('/messages') && options.method === 'POST')
  assert.equal(posts.length, 1)
  assert.deepEqual(posts[0][1].body, {content: 'hello draft', request_key: 'stable-key'})
  assert.match(root.innerHTML, /&lt;b&gt;kept&lt;\/b&gt;/)
  assert.doesNotMatch(root.innerHTML, /<b>kept<\/b>/)
  assert.match(root.textContent, /AI unavailable/)
  assert.equal(root.querySelector('[name=message]').value, 'hello draft')
})

test('list uses configured model selection and explicit grants, and empty models show setup only', async () => {
  const calls = []
  const root = new Root()
  const models = [{provider: 'vendor', model_id: 'model:version'}]
  const api = async (path, options = {}) => {
    calls.push([path, options])
    if (path === '/api/v1/models') return {models}
    if (options.method === 'POST') return {id: 's1', title: 'New', ...models[0]}
    return [{id: 's0', title: '<old>', ...models[0]}]
  }
  const page = createSessionsPage({root, api, projectID: 'p1', setInterval: () => 1, clearInterval() {}})
  await page.list()
  assert.match(root.innerHTML, /<select[^>]+name="model"/)
  assert.doesNotMatch(root.innerHTML, /name="provider"|name="model_id"/)
  assert.match(root.innerHTML, /&lt;old&gt;/)
  root.newForm.elements.title.value = 'New'
  root.newForm.elements.model.value = '0'
  root.newForm.elements.workspace_files.checked = false
  await root.newForm.onsubmit({preventDefault() {}})
  const post = calls.find(([, options]) => options.method === 'POST')
  assert.deepEqual(post[1].body, {home: 'project', title: 'New', provider: 'vendor', model_id: 'model:version', model_parameters: {}, tool_grants: {workspace_files: false}})

  const emptyRoot = new Root()
  const empty = createSessionsPage({root: emptyRoot, api: async path => path === '/api/v1/models' ? {models: []} : [], projectID: 'p1'})
  await empty.list()
  assert.equal(emptyRoot.querySelector('form[data-new]'), null)
  assert.match(emptyRoot.textContent, /configure.*model/i)
})

test('sessions route parses exactly and destroy clears polling timer', async () => {
  assert.deepEqual(parseRoute('#/projects/a%2Fb/sessions'), {name: 'sessions', projectID: 'a/b'})
  assert.deepEqual(parseRoute('#/projects/p/sessions/extra'), {name: 'home'})
  let cleared = 0
  const root = new Root()
  const page = createSessionsPage({root, api: async path => path.endsWith('/messages') ? [] : null, projectID: 'p', setInterval: () => 19, clearInterval: id => { assert.equal(id, 19); cleared++ }})
  await page.openChat({id: 's', title: 'S', provider: 'p', model_id: 'm'})
  page.destroy()
  assert.equal(cleared, 1)
})

test('polls serialize and coalesce, and an old session cannot overwrite the new chat', async () => {
  const pending = []
  const api = path => { const item = deferred(); pending.push({path, ...item}); return item.promise }
  const root = new Root()
  const page = createSessionsPage({root, api, projectID: 'p', setInterval: () => 1, clearInterval() {}})
  const firstOpen = page.openChat({id: 'old', title: 'Old', provider: 'p', model_id: 'm'})
  const extraA = page.poll(), extraB = page.poll()
  assert.equal(pending.length, 2)
  pending[0].resolve([{sequence: 1, role: 'user', content: 'old history'}]); pending[1].resolve(null)
  await Promise.resolve(); await Promise.resolve()
  assert.equal(pending.length, 4, 'many requests during a poll produce one follow-up pair')
  pending[2].resolve([]); pending[3].resolve(null)
  await Promise.all([firstOpen, extraA, extraB])

  const stale = page.poll()
  assert.equal(pending.length, 6)
  const newOpen = page.openChat({id: 'new', title: 'New', provider: 'p', model_id: 'm'})
  pending[4].resolve([{sequence: 2, role: 'assistant', content: 'stale old'}]); pending[5].resolve(null)
  await Promise.resolve(); await Promise.resolve()
  const newest = pending.slice(-2)
  newest[0].resolve([{sequence: 1, role: 'user', content: 'new history'}]); newest[1].resolve(null)
  await Promise.all([stale, newOpen])
  assert.match(root.textContent, /New.*new history/s)
  assert.doesNotMatch(root.textContent, /stale old/)
})

test('overlapping opens install only one timer and destroy clears it', async () => {
  let installed = 0, cleared = 0
  const page = createSessionsPage({root: new Root(), api: async path => path.endsWith('/messages') ? [] : null, projectID: 'p', setInterval: () => ++installed, clearInterval: () => cleared++})
  await Promise.all([
    page.openChat({id: 'same', title: 'Same', provider: 'p', model_id: 'm'}),
    page.openChat({id: 'same', title: 'Same', provider: 'p', model_id: 'm'}),
  ])
  assert.equal(installed, 1)
  page.destroy()
  assert.equal(cleared, 1)
})

test('poll failure retains cached chat history and reports the error', async () => {
  let fail = false
  const api = async path => {
    if (fail) throw new Error('network down')
    return path.endsWith('/messages') ? [{sequence: 1, role: 'user', content: 'remember me'}] : {status: 'running'}
  }
  const root = new Root(), page = createSessionsPage({root, api, projectID: 'p', setInterval: () => 1, clearInterval() {}})
  await page.openChat({id: 's', title: 'S', provider: 'p', model_id: 'm'})
  fail = true; await page.poll()
  assert.match(root.textContent, /remember me/)
  assert.match(root.textContent, /network down/)
  assert.match(root.innerHTML, /class="run-status" role="status" aria-live="polite"/)
  assert.match(root.textContent, /Run: running/)
})

test('workspace is tools-on only and refreshes with newly polled tool messages', async () => {
  const renders = []
  let messages = []
  const workspaceAPI = {workspaceTree() {}, workspaceFile() {}}
  const renderWorkspace = async options => { renders.push(options) }
  const api = async path => path.endsWith('/messages') ? messages : null
  const root = new Root()
  const page = createSessionsPage({root, api, workspaceAPI, renderWorkspace, projectID: 'p', setInterval: () => 1, clearInterval() {}})

  await page.openChat({id: 'off', title: 'Off', provider: 'p', model_id: 'm', tool_grants_json: '{"workspace_files":false}'})
  assert.equal(renders.length, 0)
  assert.doesNotMatch(root.innerHTML, /data-workspace-panel/)

  await page.openChat({id: 'on', title: 'On', provider: 'p', model_id: 'm', tool_grants: {workspace_files: true}})
  assert.match(root.innerHTML, /data-workspace-panel/)
  assert.equal(renders.at(-1).sessionID, 'on')
  messages = [{role: 'tool', changed_path: 'new.txt'}]
  await page.poll()
  assert.deepEqual(renders.at(-1).messages, messages)
})

test('malformed persisted grants default workspace off', async () => {
  let renders = 0
  const page = createSessionsPage({root: new Root(), api: async path => path.endsWith('/messages') ? [] : null, workspaceAPI: {}, renderWorkspace: async () => { renders++ }, projectID: 'p', setInterval: () => 1, clearInterval() {}})
  await page.openChat({id: 's', title: 'S', provider: 'p', model_id: 'm', tool_grants_json: '{bad'})
  assert.equal(renders, 0)
})

test('pending and failed send disables submit and retains draft, history, and one key', async () => {
  const post = deferred(), calls = []
  const api = (path, options = {}) => {
    calls.push([path, options])
    if (options.method === 'POST') return post.promise
    return Promise.resolve(path.endsWith('/messages') ? [{sequence: 1, role: 'user', content: 'cached'}] : null)
  }
  const root = new Root(), page = createSessionsPage({root, api, projectID: 'p', randomUUID: () => 'one-key', setInterval: () => 1, clearInterval() {}})
  await page.openChat({id: 's', title: 'S', provider: 'p', model_id: 'm'})
  root.chatForm.elements.message.value = 'keep draft'
  const sending = root.chatForm.onsubmit({preventDefault() {}})
  assert.equal(root.querySelector('form[data-chat] button').disabled, true)
  await root.chatForm.onsubmit({preventDefault() {}})
  post.reject(new Error('send failed')); await sending
  assert.equal(calls.filter(([, options]) => options.method === 'POST').length, 1)
  assert.equal(calls.find(([, options]) => options.method === 'POST')[1].body.request_key, 'one-key')
  assert.equal(root.querySelector('[name=message]').value, 'keep draft')
  assert.match(root.textContent, /cached.*send failed/s)
})

test('pending send cannot leak into a newly opened session', async () => {
  const post = deferred()
  const api = (path, options = {}) => options.method === 'POST' ? post.promise : Promise.resolve(path.includes('/new/') && path.endsWith('/messages') ? [{sequence: 1, role: 'user', content: 'new only'}] : path.endsWith('/messages') ? [{sequence: 1, role: 'user', content: 'old'}] : null)
  const root = new Root(), page = createSessionsPage({root, api, projectID: 'p', setInterval: () => 1, clearInterval() {}})
  await page.openChat({id: 'old', title: 'Old', provider: 'p', model_id: 'm'})
  root.chatForm.elements.message.value = 'old draft'
  const sending = root.chatForm.onsubmit({preventDefault() {}})
  await page.openChat({id: 'new', title: 'New', provider: 'p', model_id: 'm'})
  post.reject(new Error('old failure')); await sending
  assert.match(root.textContent, /New.*new only/s)
  assert.doesNotMatch(root.textContent, /old failure|old draft/)
})

test('an older button open rejection cannot replace a newer successful chat', async () => {
  const root = new Root()
  let page, newerOpen, intervals = 0
  const api = async path => {
    if (path === '/api/v1/models') return {models: [{provider: 'p', model_id: 'm'}]}
    if (path === '/api/v1/projects/p/sessions') return [{id: 'old', title: 'Old', provider: 'p', model_id: 'm'}]
    if (path.includes('/new/') && path.endsWith('/messages')) return [{sequence: 1, role: 'user', content: 'new history'}]
    if (path.endsWith('/messages')) return []
    if (path.endsWith('/runs/current')) return null
    throw new Error(`unexpected request: ${path}`)
  }
  page = createSessionsPage({
    root,
    api,
    projectID: 'p',
    setInterval: () => {
      if (++intervals === 1) {
        newerOpen = page.openChat({id: 'new', title: 'New', provider: 'p', model_id: 'm'})
        throw new Error('stale cannot open')
      }
      return intervals
    },
    clearInterval() {},
  })
  await page.list()

  root.sessionButtons[0].onclick()
  await new Promise(resolve => setImmediate(resolve))
  await newerOpen
  await new Promise(resolve => setImmediate(resolve))

  assert.match(root.textContent, /New.*new history/s)
  assert.equal(root.renders.some(html => /Sessions.*stale cannot open/s.test(html)), false)
})

test('real create and navigation handlers consume failures and show errors inline', async () => {
  const unhandled = []
  const listener = reason => unhandled.push(reason)
  process.on('unhandledRejection', listener)
  try {
    const root = new Root()
    let cannotList = false
    let cannotCreate = true
    let cannotOpen = true
    let intervalAttempt = deferred()
    const api = async (path, options = {}) => {
      if (options.method === 'POST') {
        if (cannotCreate) throw new Error('cannot create')
        return {id: 'created', title: 'Created', provider: 'p', model_id: 'm'}
      }
      if (cannotList && (path === '/api/v1/models' || path === '/api/v1/projects/p/sessions')) throw new Error('cannot list')
      if (path === '/api/v1/models') return {models: [{provider: 'p', model_id: 'm'}]}
      if (path === '/api/v1/projects/p/sessions') return [{id: 's', title: 'Session', provider: 'p', model_id: 'm'}]
      if (path.endsWith('/messages')) return []
      if (path.endsWith('/runs/current')) return null
      throw new Error(`unexpected request: ${path}`)
    }
    const page = createSessionsPage({
      root,
      api,
      projectID: 'p',
      setInterval: () => {
        intervalAttempt.resolve()
        if (cannotOpen) throw new Error('cannot open')
        return 1
      },
      clearInterval() {},
    })
    await page.list()

    root.newForm.onsubmit({preventDefault() {}})
    await new Promise(resolve => setImmediate(resolve))
    assert.match(root.textContent, /cannot create/)

    cannotCreate = false
    intervalAttempt = deferred()
    root.newForm.onsubmit({preventDefault() {}})
    await intervalAttempt.promise
    await new Promise(resolve => setImmediate(resolve))
    assert.match(root.textContent, /Sessions.*cannot open/s)

    intervalAttempt = deferred()
    root.sessionButtons[0].onclick()
    await intervalAttempt.promise
    await new Promise(resolve => setImmediate(resolve))
    assert.match(root.textContent, /Sessions.*cannot open/s)
    assert.equal(root.sessionButtons.length, 1, 'open failure re-renders the sessions list')
    assert.deepEqual(unhandled, [])

    cannotOpen = false
    intervalAttempt = deferred()
    root.sessionButtons[0].onclick()
    await intervalAttempt.promise
    await new Promise(resolve => setImmediate(resolve))
    assert.ok(root.back, 'a successful open renders a back element')
    const backClick = root.back.listeners.get('click')
    assert.equal(typeof backClick, 'function', 'production registered the back click listener')

    cannotList = true
    backClick({preventDefault() {}})
    await new Promise(resolve => setImmediate(resolve))
    assert.match(root.textContent, /cannot list/)
    assert.deepEqual(unhandled, [])
  } finally {
    process.off('unhandledRejection', listener)
  }
})
