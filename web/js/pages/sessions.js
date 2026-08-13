import {renderWorkspacePanel} from '../components/workspace.mjs'
import {workspaceTree, workspaceFile} from '../api.js'

const esc = value => String(value ?? '').replace(/[&<>"']/g, char => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[char]))
const pathID = value => encodeURIComponent(value)

function workspaceEnabled(session) {
  if (session?.tool_grants && typeof session.tool_grants === 'object') return session.tool_grants.workspace_files === true
  if (typeof session?.tool_grants_json !== 'string') return false
  try { return JSON.parse(session.tool_grants_json)?.workspace_files === true } catch { return false }
}

export function createSessionsPage({
  root,
  api,
  projectID,
  randomUUID = () => globalThis.crypto.randomUUID(),
  setInterval = globalThis.setInterval.bind(globalThis),
  clearInterval = globalThis.clearInterval.bind(globalThis),
  isCurrent = () => true,
  workspaceAPI = {workspaceTree, workspaceFile},
  renderWorkspace = renderWorkspacePanel,
}) {
  let session = null
  let timer = null
  let sending = false
  let error = ''
  let destroyed = false
  let chatGeneration = 0
  let messages = []
  let run = null
  let pollPromise = null
  let pollQueued = false
  let sendToken = null
  let pollFailed = false

  function current(generation) { return !destroyed && isCurrent() && generation === chatGeneration }

  async function list() {
    stopPolling()
    session = null
    const generation = ++chatGeneration
    const [configured, sessions] = await Promise.all([
      api('/api/v1/models'),
      api(`/api/v1/projects/${pathID(projectID)}/sessions`),
    ])
    if (!current(generation)) return
    const models = configured?.models || []
    let listError = ''
    const openFromList = async selected => {
      const expectedGeneration = chatGeneration + 1
      const selectedID = selected.id
      try {
        await openChat(selected)
      } catch (openError) {
        if (current(expectedGeneration) && session?.id === selectedID) {
          listError = openError.message
          renderList()
        }
      }
    }
    const renderList = () => {
      const create = models.length ? `<form data-new class="sessions-create"><label>Title<input name="title" required maxlength="200"></label><label>Model<select name="model" required>${models.map((model, index) => `<option value="${index}">${esc(model.provider)}:${esc(model.model_id)}</option>`).join('')}</select></label><label class="sessions-grant"><input type="checkbox" name="workspace_files"> Allow workspace files</label><button>Create session</button><p class="error" role="alert">${esc(listError)}</p></form>` : '<section class="sessions-setup"><p>Configure a model before creating a session.</p><a class="button" href="#/settings">Open settings</a></section>'
      root.innerHTML = `<div class="page-heading"><h2>Sessions</h2></div>${create}<ul class="sessions-list">${sessions.map(item => `<li><button type="button" data-session="${esc(item.id)}"><span>${esc(item.title)}</span><span class="model-badge">${esc(item.provider)}:${esc(item.model_id)}</span></button></li>`).join('')}</ul>`
      const form = root.querySelector('form[data-new]')
      if (form) form.onsubmit = event => {
        event.preventDefault()
        const index = Number(form.elements.model.value)
        const model = models[index]
        if (!model) return
        void api(`/api/v1/projects/${pathID(projectID)}/sessions`, {method: 'POST', body: {home: 'project', title: form.elements.title.value, provider: model.provider, model_id: model.model_id, model_parameters: {}, tool_grants: {workspace_files: Boolean(form.elements.workspace_files.checked)}}})
          .then(created => { if (current(generation)) void openFromList(created) })
          .catch(createError => { if (current(generation)) { listError = createError.message; renderList() } })
      }
      root.querySelectorAll('[data-session]').forEach(button => { button.onclick = () => { void openFromList(sessions.find(item => item.id === button.dataset.session)) } })
    }
    renderList()
  }

  function renderChat(preserveDraft = true) {
    const draft = preserveDraft ? root.querySelector('[name=message]')?.value || '' : ''
    const workspace = workspaceEnabled(session) ? '<aside data-workspace-panel></aside>' : ''
    root.innerHTML = `<div class="session-layout"><section class="sessions-chat"><button type="button" data-back>Sessions</button><div class="page-heading"><h2>${esc(session.title)}</h2><span class="model-badge">${esc(session.provider)}:${esc(session.model_id)}</span></div><ol class="messages">${[...messages].sort((a, b) => a.sequence - b.sequence).map(message => `<li class="message message-${esc(message.role)}"><strong>${esc(message.role)}</strong><p>${esc(message.content)}</p></li>`).join('')}</ol><p class="run-status" role="status" aria-live="polite">${run ? `Run: ${esc(run.status)}` : 'Idle'}</p><p class="error" role="alert">${esc(error)}</p><form data-chat><label>Message<textarea name="message" required></textarea></label><button ${sending || run ? 'disabled' : ''}>Send</button></form></section>${workspace}</div>`
    const input = root.querySelector('[name=message]')
    if (input) input.value = draft
    root.querySelector('[data-back]')?.addEventListener('click', () => { void list().catch(listError => { if (!destroyed && isCurrent()) root.innerHTML = `<p class="error" role="alert">${esc(listError.message)}</p>` }) })
    root.querySelector('form[data-chat]').onsubmit = send
  }

  async function refreshWorkspace(generation, id) {
    if (!workspaceEnabled(session)) return
    const container = root.querySelector('[data-workspace-panel]')
    if (!container) return
    await renderWorkspace({container, sessionID: id, messages, api: workspaceAPI, isCurrent: () => current(generation) && session?.id === id})
  }

  async function poll() {
    if (!session || destroyed || !isCurrent()) return
    pollQueued = true
    if (pollPromise) return pollPromise
    pollPromise = (async () => {
      while (pollQueued) {
        pollQueued = false
        const generation = chatGeneration
        const id = session?.id
        if (!id) continue
        try {
          const [nextMessages, nextRun] = await Promise.all([
            api(`/api/v1/sessions/${pathID(id)}/messages`),
            api(`/api/v1/sessions/${pathID(id)}/runs/current`),
          ])
          if (current(generation) && session?.id === id) {
            messages = nextMessages || []
            run = nextRun
            if (pollFailed) error = ''
            pollFailed = false
            renderChat()
            if (workspaceEnabled(session)) await refreshWorkspace(generation, id)
          }
        } catch (pollError) {
          if (current(generation) && session?.id === id) {
            error = pollError.message
            pollFailed = true
            renderChat()
          }
        }
      }
    })().finally(() => { pollPromise = null })
    return pollPromise
  }

  async function send(event) {
    event.preventDefault()
    if (sending || !session) return
    const content = event.currentTarget?.elements?.message?.value ?? root.querySelector('[name=message]').value
    sending = true
    error = ''
    pollFailed = false
    const id = session.id
    const generation = chatGeneration
    const token = {}
    sendToken = token
    const key = randomUUID()
    renderChat()
    try {
      await api(`/api/v1/sessions/${pathID(id)}/messages`, {method: 'POST', body: {content, request_key: key}})
      if (sendToken === token && current(generation) && session?.id === id) renderChat(false)
    } catch (sendError) {
      if (sendToken === token && current(generation) && session?.id === id) {
        error = sendError.message
        pollFailed = false
      }
    } finally {
      if (sendToken === token && current(generation) && session?.id === id) {
        sending = false
        sendToken = null
        renderChat()
        await poll()
      }
    }
  }

  async function openChat(value) {
    stopPolling()
    destroyed = false
    session = value
    error = ''
    pollFailed = false
    messages = []
    run = null
    sending = false
    sendToken = null
    const generation = ++chatGeneration
    await poll()
    if (current(generation) && session?.id === value.id && timer === null) timer = setInterval(() => { void poll().catch(() => {}) }, 1500)
  }

  function stopPolling() { if (timer !== null) clearInterval(timer); timer = null }
  function destroy() { destroyed = true; ++chatGeneration; stopPolling(); session = null }
  return {list, openChat, poll, destroy}
}
