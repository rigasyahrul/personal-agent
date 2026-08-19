import test from 'node:test'
import assert from 'node:assert/strict'
import {createSessionsPage} from './sessions.js'
import {parseRoute} from '../router.js'
import {TestDocument,TestElement,findText} from '../test-dom.mjs'

globalThis.document = new TestDocument()

class Root {
  constructor() { this.renders = []; this.innerHTML = ''; this.activeMessage = null; this.messageList = null; this.runStatus = null }
  set innerHTML(value) {
    this.renders.push(value)
    this.html = value
    const previous = this.chatForm?.elements?.message
    const message = value.includes('data-chat') ? {
      value: previous?.value || '',
      selectionStart: 0,
      selectionEnd: 0,
      focus: () => { this.activeMessage = message },
    } : null
    if (this.activeMessage && this.activeMessage !== message) this.activeMessage = null
    this.chatForm = message ? {elements: {message}} : null
    this.newForm = value.includes('data-new') ? {elements: {title: {value: ''}, model: {value: '0'}, workspace_files: {checked: false}}} : null
    this.back = value.includes('data-back') ? {
      listeners: new Map(),
      addEventListener: (type, handler) => { this.back.listeners.set(type, handler) },
    } : null
    const previousDisabled = this.sendButton?.disabled
    this.sendButton = value.includes('data-chat') ? {
      get disabled() { return this._disabled },
      set disabled(next) {
        this._disabled = Boolean(next)
        const host = this.root
        if (!host?.html) return
        host.html = host.html.replace(/<form data-chat>[\s\S]*?<\/form>/, form => form.replace(/<button[^>]*>Send<\/button>/, `<button${this._disabled ? ' disabled' : ''}>Send</button>`))
      },
      _disabled: previousDisabled ?? /<button disabled>Send<\/button>/.test(value),
      root: this,
    } : null
    this.workspace = value.includes('data-workspace-panel') ? {} : null
    this.operationHost = value.includes('data-operation-statuses') ? new TestElement('div') : null
    this.chatAlert = value.includes('data-chat-alert') ? {
      set textContent(text) {
        this.text = text
        const host = this.root
        if (!host?.html) return
        host.html = host.html.replace(/data-chat-alert[^>]*>[\s\S]*?<\/p>/, match => match.replace(/>[\s\S]*<\/p>/, `>${text}</p>`))
      },
      get textContent() { return this.text || '' },
      text: (value.match(/data-chat-alert[^>]*>([\s\S]*?)<\/p>/) || [,''])[1],
      root: this,
    } : null
    this.messageList = value.includes('class="messages"') ? {
      set innerHTML(html) {
        this.html = html
        const host = this.root
        if (!host) return
        host.html = host.html.replace(/<ol class="messages">[\s\S]*?<\/ol>/, `<ol class="messages">${html}</ol>`)
      },
      get innerHTML() { return this.html || '' },
      html: (value.match(/<ol class="messages">([\s\S]*?)<\/ol>/) || [,''])[1],
      root: this,
    } : null
    this.runStatus = value.includes('run-status') ? {
      set textContent(text) {
        this.text = text
        const host = this.root
        if (!host) return
        host.html = host.html.replace(/class="run-status"[^>]*>[\s\S]*?<\/p>/, match => match.replace(/>[\s\S]*<\/p>/, `>${text}</p>`))
      },
      get textContent() { return this.text || '' },
      text: (value.match(/class="run-status"[^>]*>([\s\S]*?)<\/p>/) || [,''])[1],
      root: this,
    } : null
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
    if (selector === '[data-operation-statuses]') return this.operationHost
    if (selector === '[data-chat-alert]') return this.chatAlert
    if (selector === 'ol.messages') return this.messageList
    if (selector === '.run-status') return this.runStatus
    return null
  }
  querySelectorAll(selector) { return selector === '[data-session]' ? this.sessionButtons : [] }
  contains(node) { return node != null && (node === this || node === this.chatForm?.elements?.message) }
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

test('polling does not steal message focus or selection while typing, and restores them when chat content changes', async () => {
  let messages = [{sequence: 1, role: 'user', content: 'hello'}]
  let run = null
  const api = async path => {
    if (path.endsWith('/messages')) return messages
    if (path.endsWith('/runs/current')) return run
    return null
  }
  const root = new Root()
  const document = {
    get activeElement() { return root.activeMessage },
  }
  const page = createSessionsPage({root, api, projectID: 'p', document, setInterval: () => 1, clearInterval() {}})
  await page.openChat({id: 's', title: 'S', provider: 'p', model_id: 'm'})
  const firstInput = root.querySelector('[name=message]')
  firstInput.value = 'typing here'
  firstInput.selectionStart = 6
  firstInput.selectionEnd = 11
  firstInput.focus()
  const rendersBefore = root.renders.length

  await page.poll()
  assert.equal(root.renders.length, rendersBefore, 'unchanged poll must not rebuild chat DOM')
  assert.equal(root.querySelector('[name=message]'), firstInput)
  assert.equal(document.activeElement, firstInput)
  assert.equal(firstInput.value, 'typing here')
  assert.equal(firstInput.selectionStart, 6)
  assert.equal(firstInput.selectionEnd, 11)

  messages = [...messages, {sequence: 2, role: 'assistant', content: 'reply'}]
  run = {status: 'running'}
  await page.poll()
  const nextInput = root.querySelector('[name=message]')
  assert.equal(root.renders.length, rendersBefore, 'message/run updates must patch in place, not rebuild shell')
  assert.equal(nextInput, firstInput, 'textarea node stays alive across poll updates')
  assert.equal(document.activeElement, firstInput, 'focus stays on the same message field')
  assert.equal(firstInput.value, 'typing here')
  assert.equal(firstInput.selectionStart, 6)
  assert.equal(firstInput.selectionEnd, 11)
  assert.match(root.textContent, /reply/)
  assert.match(root.textContent, /Run: running/)
  assert.equal(root.querySelector('form[data-chat] button').disabled, true)
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

  await page.openChat({id: 'on', title: 'On', provider: 'p', model_id: 'm', tool_grants_json: '{"workspace_files":true}'})
  assert.match(root.innerHTML, /data-workspace-panel/)
  assert.equal(renders.at(-1).sessionID, 'on')
  assert.equal(renders.length, 1, 'positive persisted grants drive the initial panel refresh')
  messages = [{role: 'tool', changed_path: 'new.txt'}]
  await page.poll()
  assert.deepEqual(renders.at(-1).messages, messages)
  assert.equal(renders.length, 2, 'positive persisted grants drive the polling panel refresh')
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

test('unavailable and throwing storage degrade without preventing sessions', async () => {
  const descriptor=Object.getOwnPropertyDescriptor(globalThis,'localStorage')
  Object.defineProperty(globalThis,'localStorage',{configurable:true,get(){throw new Error('blocked')}})
  try{
    const page=createSessionsPage({root:new Root(),api:async path=>path.endsWith('/messages')?[]:null,projectID:'p',setInterval:()=>1,clearInterval(){}})
    await assert.doesNotReject(page.openChat({id:'s',title:'S',provider:'p',model_id:'m'}))
  }finally{descriptor?Object.defineProperty(globalThis,'localStorage',descriptor):delete globalThis.localStorage}
  const storage={getItem(){throw new Error('get blocked')},setItem(){throw new Error('set blocked')}}
  const page=createSessionsPage({root:new Root(),storage,api:async path=>path.endsWith('/messages')?[]:null,projectID:'p',setInterval:()=>1,clearInterval(){}})
  await assert.doesNotReject(page.openChat({id:'s',title:'S',provider:'p',model_id:'m'}))
})

test('restored operation polling is serialized across overlapping chat triggers',async()=>{
  let active=0,maxActive=0,calls=0,release
  const gate=new Promise(resolve=>release=resolve)
  const page=createSessionsPage({root:new Root(),storage:{getItem:()=> '["op"]'},api:async path=>path.endsWith('/messages')?[]:null,projectID:'p',setInterval:()=>1,clearInterval(){},getOperation:async()=>{calls++;active++;maxActive=Math.max(maxActive,active);await gate;active--;return{operation_id:'op',badge:'Ready'}}})
  const opening=page.openChat({id:'s',title:'S',provider:'p',model_id:'m'});await new Promise(resolve=>setImmediate(resolve));const extra=page.poll();release();await Promise.all([opening,extra]);await new Promise(resolve=>setImmediate(resolve))
  assert.ok(calls>=1);assert.equal(maxActive,1)
})

test('promote dialog survives chat rerender with entered state and selection Save, then destroy removes it',async()=>{
  const document=new TestDocument(),host=document.body,root=new Root();let panel,options
  const renderWorkspace=async value=>{options=value;panel=new TestElement('section');panel.querySelector=selector=>selector==='[data-promote]'?panel.children.find(child=>Object.hasOwn(child.dataset,'promote'))||null:null;value.container.querySelector=selector=>selector==='.workspace-panel'?panel:null;value.onTree([{path:'draft.md',kind:'file'}],panel);if(!panel.children.length)value.onFileSelected({path:'draft.md',kind:'file'})}
  const api=async path=>path.includes('/projects/p')?{name:'Project'}:path.endsWith('/messages')?[]:null
  const page=createSessionsPage({root,api,projectID:'p',document,dialogHost:host,renderWorkspace,workspaceAPI:{},setInterval:()=>1,clearInterval(){}})
  await page.openChat({id:'s',title:'S',provider:'p',model_id:'m',tool_grants:{workspace_files:true}})
  panel.children[0].click();await new Promise(resolve=>setImmediate(resolve));const dialog=host.children[0],target=dialog.querySelector('input');target.value='entered.md'
  const mode=dialog.querySelectorAll('input').find(input=>input.name==='review_mode'&&input.value==='bites');mode.checked=true
  await page.poll();assert.equal(host.children[0],dialog);assert.equal(target.value,'entered.md');assert.ok(panel.children.find(child=>child.textContent==='Save to source'))
  page.destroy();assert.equal(host.children.length,0)
})

test('promote submit remains bound to captured file and stable attempt across failure',async()=>{
  const document=new TestDocument(),root=new Root(),calls=[];let workspaceOptions,fail=true,key=0
  const renderWorkspace=async value=>{workspaceOptions=value;const panel=new TestElement('section');panel.querySelector=selector=>selector==='[data-promote]'?panel.children.find(child=>Object.hasOwn(child.dataset,'promote'))||null:null;value.container.querySelector=selector=>selector==='.workspace-panel'?panel:null;value.onFileSelected({path:'draft.md',kind:'file'});value.save=panel.children[0]}
  const page=createSessionsPage({root,document,dialogHost:document.body,projectID:'p',renderWorkspace,workspaceAPI:{},randomUUID:()=>`key-${++key}`,api:async path=>path==='/api/v1/projects/p'?{name:'P'}:path.endsWith('/messages')?[]:null,promote:async(...args)=>{calls.push(args);if(fail)throw new Error('missing file');return{operation_id:'op'}},getOperation:async()=>({operation_id:'op',badge:'Ready'}),setInterval:()=>1,clearInterval(){}})
  await page.openChat({id:'session',title:'S',provider:'p',model_id:'m',tool_grants:{workspace_files:true}})
  workspaceOptions.save.click();await new Promise(resolve=>setImmediate(resolve))
  const dialog=document.body.children[0],form=dialog.querySelector('form'),target=dialog.querySelector('input');target.value='target.md'
  const bites=dialog.querySelectorAll('input').find(input=>input.value==='bites');dialog.querySelectorAll('input').forEach(input=>input.checked=false);bites.checked=true
  workspaceOptions.onFileSelected({path:'other.md',kind:'file'})
  await form.onsubmit({preventDefault(){}})
  assert.deepEqual(calls[0],['session',{workspace_path:'draft.md',target_relative_path:'target.md',review_mode:'bites'},'key-1'])
  assert.equal(findText(dialog,'missing file').getAttribute('role'),'alert');assert.equal(target.value,'target.md')
  fail=false;await form.onsubmit({preventDefault(){}})
  assert.equal(calls[1][2],'key-1');assert.equal(document.body.children.length,0)
})

test('promote dialog actual cancel, native close, back, and session switch handlers clean up stale forms',async()=>{
  const document=new TestDocument(),root=new Root(),promotions=[];let save
  const renderWorkspace=async value=>{const panel=new TestElement('section');panel.querySelector=selector=>selector==='[data-promote]'?panel.children.find(child=>Object.hasOwn(child.dataset,'promote'))||null:null;value.container.querySelector=selector=>selector==='.workspace-panel'?panel:null;value.onFileSelected({path:'draft.md',kind:'file'});save=panel.children[0]}
  const api=async path=>path==='/api/v1/projects/p'?{name:'P'}:path==='/api/v1/models'?{models:[]}:path==='/api/v1/projects/p/sessions'?[]:path.endsWith('/messages')?[]:null
  const page=createSessionsPage({root,document,dialogHost:document.body,projectID:'p',renderWorkspace,workspaceAPI:{},api,promote:async(...args)=>{promotions.push(args);return{operation_id:'op'}},setInterval:()=>1,clearInterval(){}})
  await page.openChat({id:'one',title:'One',provider:'p',model_id:'m',tool_grants:{workspace_files:true}})
  save.click();await new Promise(resolve=>setImmediate(resolve));let dialog=document.body.children[0],stale=dialog.querySelector('form');findText(dialog,'Cancel').click();assert.equal(document.body.children.length,0);await stale.onsubmit({preventDefault(){}});assert.equal(promotions.length,0)
  save.click();await new Promise(resolve=>setImmediate(resolve));dialog=document.body.children[0];dialog.close();assert.equal(document.body.children.length,0);save.click();await new Promise(resolve=>setImmediate(resolve));assert.equal(document.body.children.length,1,'a fresh Save opens exactly one dialog')
  stale=document.body.children[0].querySelector('form');root.back.listeners.get('click')({preventDefault(){}});await new Promise(resolve=>setImmediate(resolve));assert.equal(document.body.children.length,0);await stale.onsubmit({preventDefault(){}});assert.equal(promotions.length,0)
  await page.openChat({id:'one',title:'One',provider:'p',model_id:'m',tool_grants:{workspace_files:true}});save.click();await new Promise(resolve=>setImmediate(resolve));stale=document.body.children[0].querySelector('form');await page.openChat({id:'two',title:'Two',provider:'p',model_id:'m'});assert.equal(document.body.children.length,0);await stale.onsubmit({preventDefault(){}});assert.equal(promotions.length,0)
})

test('retry cards actual rendered clicks deduplicate, survive rerender, recover from failure, and repoll',async()=>{
  const unhandled=[],listener=reason=>unhandled.push(reason);process.on('unhandledRejection',listener)
  try{
    const root=new Root(),retryGates=[deferred(),deferred()],results=[{operation_id:'op',badge:'Cards failed — Retry cards',pending_id:'pending',retry_cards:true},{operation_id:'op',badge:'Ready'}];let retries=0,polls=0
    const page=createSessionsPage({root,document:new TestDocument(),storage:{getItem:()=> '["op"]',setItem(){}},projectID:'p',api:async path=>path.endsWith('/messages')?[]:null,getOperation:async()=>{polls++;return results.shift()||{operation_id:'op',badge:'Ready'}},retryPending:async()=>{const gate=retryGates[retries++];return gate.promise},setInterval:()=>1,clearInterval(){}})
    await page.openChat({id:'s',title:'S',provider:'p',model_id:'m'});await new Promise(resolve=>setImmediate(resolve));let button=findText(root.operationHost,'Retry cards');assert.ok(button)
    button.click();button.click();await new Promise(resolve=>setImmediate(resolve));assert.equal(retries,1);await page.poll();button=findText(root.operationHost,'Retry cards');assert.equal(button.disabled,true)
    retryGates[0].reject(new Error('retry failed'));await new Promise(resolve=>setImmediate(resolve));await new Promise(resolve=>setImmediate(resolve));assert.match(root.textContent,/retry failed/);button=findText(root.operationHost,'Retry cards');assert.equal(button.disabled,false)
    button.click();await new Promise(resolve=>setImmediate(resolve));assert.equal(retries,2);retryGates[1].resolve();await new Promise(resolve=>setImmediate(resolve));await new Promise(resolve=>setImmediate(resolve));assert.ok(polls>=2);assert.equal(findText(root.operationHost,'Retry cards'),undefined);assert.equal(root.chatAlert.textContent,'retry failed','successful operation status preserves the ordinary retry error');assert.deepEqual(unhandled,[])
  }finally{process.off('unhandledRejection',listener)}
})

test('operation polling contains rejection, retries, coalesces one follow-up, and ignores switched session',async()=>{
  const root=new Root(),first=deferred(),second=deferred();let calls=0,fail=true
  const page=createSessionsPage({root,document:new TestDocument(),storage:{getItem:key=>key.endsWith(':old')?'["op"]':'[]'},projectID:'p',api:async path=>path.endsWith('/messages')?[]:null,getOperation:async()=>{calls++;if(calls===1)return first.promise;if(fail){fail=false;throw new Error('operation down')}return second.promise},setInterval:()=>1,clearInterval(){}})
  const opening=page.openChat({id:'old',title:'Old',provider:'p',model_id:'m'});await new Promise(resolve=>setImmediate(resolve));page.poll();page.poll();page.poll();first.resolve({operation_id:'op',badge:'Promoting…'});await opening;await new Promise(resolve=>setImmediate(resolve));assert.equal(calls,2,'many triggers queue only one follow-up operation cycle');await new Promise(resolve=>setImmediate(resolve));assert.match(root.chatAlert.textContent,/operation down/)
  const retry=page.poll();await new Promise(resolve=>setImmediate(resolve));assert.equal(calls,3);const switched=page.openChat({id:'new',title:'New',provider:'p',model_id:'m'});second.resolve({operation_id:'op',badge:'Ready'});await Promise.all([retry,switched]);await new Promise(resolve=>setImmediate(resolve));assert.match(root.textContent,/New/);assert.equal(findText(root.operationHost,'Ready'),undefined);assert.doesNotMatch(root.textContent,/operation down/)
  page.destroy();await new Promise(resolve=>setImmediate(resolve));assert.equal(findText(root.operationHost,'Ready'),undefined)
})

test('operation failure updates the existing alert without replacing a populated workspace, then success clears it and renders the badge',async()=>{
  const unhandled=[],listener=reason=>unhandled.push(reason);process.on('unhandledRejection',listener)
  try{
    const root=new Root(),document=new TestDocument();let save,operationCalls=0
    const renderWorkspace=async value=>{const panel=new TestElement('section');panel.querySelector=selector=>selector==='[data-promote]'?panel.children[0]||null:null;value.container.querySelector=selector=>selector==='.workspace-panel'?panel:null;value.onFileSelected({path:'draft.md',kind:'file'});save=panel.children[0]}
    const page=createSessionsPage({root,document,dialogHost:document.body,storage:{getItem:()=> '["op"]',setItem(){}},projectID:'p',renderWorkspace,workspaceAPI:{},api:async path=>path==='/api/v1/projects/p'?{name:'P'}:path.endsWith('/messages')?[]:null,promote:async()=>({operation_id:'op'}),getOperation:async()=>{operationCalls++;if(operationCalls===1)throw new Error('operation unavailable');return{operation_id:'op',badge:'Ready'}},setInterval:()=>1,clearInterval(){}})
    await page.openChat({id:'s',title:'S',provider:'p',model_id:'m',tool_grants:{workspace_files:true}});await new Promise(resolve=>setImmediate(resolve))
    const workspace=root.workspace,alert=root.chatAlert,renders=root.renders.length
    assert.ok(save);assert.equal(alert.textContent,'operation unavailable');assert.equal(root.workspace,workspace);assert.equal(root.renders.length,renders)
    save.click();await new Promise(resolve=>setImmediate(resolve));await document.body.children[0].querySelector('form').onsubmit({preventDefault(){}})
    assert.equal(root.chatAlert,alert);assert.equal(alert.textContent,'');assert.equal(root.workspace,workspace);assert.equal(root.renders.length,renders);assert.ok(findText(root.operationHost,'Ready'));assert.deepEqual(unhandled,[])
  }finally{process.off('unhandledRejection',listener)}
})

test('changed promote form payload receives a new idempotency key after unchanged retry',async()=>{
  const document=new TestDocument(),root=new Root(),calls=[];let save,key=0
  const renderWorkspace=async value=>{const panel=new TestElement('section');panel.querySelector=selector=>selector==='[data-promote]'?panel.children[0]||null:null;value.container.querySelector=selector=>selector==='.workspace-panel'?panel:null;value.onFileSelected({path:'draft.md',kind:'file'});save=panel.children[0]}
  const page=createSessionsPage({root,document,projectID:'p',renderWorkspace,workspaceAPI:{},randomUUID:()=>`key-${++key}`,api:async path=>path==='/api/v1/projects/p'?{name:'P'}:path.endsWith('/messages')?[]:null,promote:async(...args)=>{calls.push(args);throw new Error('no')},setInterval:()=>1,clearInterval(){}})
  await page.openChat({id:'s',title:'S',provider:'p',model_id:'m',tool_grants:{workspace_files:true}});save.click();await new Promise(resolve=>setImmediate(resolve));const dialog=document.body.children[0],form=dialog.querySelector('form'),target=dialog.querySelector('input');target.value='one.md';await form.onsubmit({preventDefault(){}});await form.onsubmit({preventDefault(){}});target.value='two.md';const whole=dialog.querySelectorAll('input').find(input=>input.value==='whole');dialog.querySelectorAll('input').forEach(input=>input.checked=false);whole.checked=true;await form.onsubmit({preventDefault(){}})
  assert.equal(calls[0][2],'key-1');assert.equal(calls[1][2],'key-1');assert.deepEqual(calls[2],['s',{workspace_path:'draft.md',target_relative_path:'two.md',review_mode:'whole'},'key-2'])
})
