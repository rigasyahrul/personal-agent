import test from 'node:test'
import assert from 'node:assert/strict'
import {createSessionsPage} from './sessions.js'
import {parseRoute} from '../router.js'

class Root {
  constructor() { this.innerHTML = '' }
  set innerHTML(value) {
    this.html = value
    this.chatForm = value.includes('data-chat') ? {elements: {message: {value: this.chatForm?.elements.message.value || ''}}} : null
    this.newForm = value.includes('data-new') ? {elements: {title: {value: ''}, model: {value: '0'}, workspace_files: {checked: false}}} : null
  }
  get innerHTML() { return this.html }
  get textContent() { return this.html.replace(/<[^>]*>/g, ' ').replaceAll('&lt;', '<').replaceAll('&gt;', '>').replaceAll('&amp;', '&') }
  querySelector(selector) {
    if (selector === 'form[data-chat]') return this.chatForm
    if (selector === 'form[data-new]') return this.newForm
    if (selector === '[name=message]') return this.chatForm?.elements.message
    return null
  }
  querySelectorAll() { return [] }
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
