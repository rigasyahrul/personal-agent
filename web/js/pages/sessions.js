const esc = value => String(value ?? '').replace(/[&<>"']/g, char => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[char]))
const pathID = value => encodeURIComponent(value)

export function createSessionsPage({
  root,
  api,
  projectID,
  randomUUID = () => globalThis.crypto.randomUUID(),
  setInterval = globalThis.setInterval.bind(globalThis),
  clearInterval = globalThis.clearInterval.bind(globalThis),
  isCurrent = () => true,
}) {
  let session = null
  let timer = null
  let sending = false
  let error = ''
  let destroyed = false
  let chatGeneration = 0

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
    const create = models.length ? `<form data-new class="sessions-create"><label>Title<input name="title" required maxlength="200"></label><label>Model<select name="model" required>${models.map((model, index) => `<option value="${index}">${esc(model.provider)}:${esc(model.model_id)}</option>`).join('')}</select></label><label class="sessions-grant"><input type="checkbox" name="workspace_files"> Allow workspace files</label><button>Create session</button><p class="error" role="alert"></p></form>` : '<section class="sessions-setup"><p>Configure a model before creating a session.</p><a class="button" href="#/settings">Open settings</a></section>'
    root.innerHTML = `<div class="page-heading"><h2>Sessions</h2></div>${create}<ul class="sessions-list">${sessions.map(item => `<li><button type="button" data-session="${esc(item.id)}"><span>${esc(item.title)}</span><span class="model-badge">${esc(item.provider)}:${esc(item.model_id)}</span></button></li>`).join('')}</ul>`
    const form = root.querySelector('form[data-new]')
    if (form) form.onsubmit = async event => {
      event.preventDefault()
      const index = Number(form.elements.model.value)
      const model = models[index]
      if (!model) return
      const created = await api(`/api/v1/projects/${pathID(projectID)}/sessions`, {method: 'POST', body: {home: 'project', title: form.elements.title.value, provider: model.provider, model_id: model.model_id, model_parameters: {}, tool_grants: {workspace_files: Boolean(form.elements.workspace_files.checked)}}})
      if (current(generation)) await openChat(created)
    }
    root.querySelectorAll('[data-session]').forEach(button => { button.onclick = () => openChat(sessions.find(item => item.id === button.dataset.session)) })
  }

  function renderChat(messages, run, preserveDraft = true) {
    const draft = preserveDraft ? root.querySelector('[name=message]')?.value || '' : ''
    root.innerHTML = `<section class="sessions-chat"><button type="button" data-back>Sessions</button><div class="page-heading"><h2>${esc(session.title)}</h2><span class="model-badge">${esc(session.provider)}:${esc(session.model_id)}</span></div><ol class="messages">${[...messages].sort((a, b) => a.sequence - b.sequence).map(message => `<li class="message message-${esc(message.role)}"><strong>${esc(message.role)}</strong><p>${esc(message.content)}</p></li>`).join('')}</ol><p class="run-status">${run ? `Run: ${esc(run.status)}` : 'Idle'}</p><p class="error" role="alert">${esc(error)}</p><form data-chat><label>Message<textarea name="message" required></textarea></label><button ${sending || run ? 'disabled' : ''}>Send</button></form></section>`
    const input = root.querySelector('[name=message]')
    if (input) input.value = draft
    root.querySelector('[data-back]')?.addEventListener('click', list)
    root.querySelector('form[data-chat]').onsubmit = send
  }

  async function poll() {
    if (!session || destroyed || !isCurrent()) return
    const generation = chatGeneration
    const id = session.id
    try {
      const [messages, run] = await Promise.all([
        api(`/api/v1/sessions/${pathID(id)}/messages`),
        api(`/api/v1/sessions/${pathID(id)}/runs/current`),
      ])
      if (current(generation) && session?.id === id) renderChat(messages || [], run)
    } catch (pollError) {
      if (current(generation) && session?.id === id) {
        error = pollError.message
        renderChat([], null)
      }
    }
  }

  async function send(event) {
    event.preventDefault()
    if (sending || !session) return
    const content = event.currentTarget?.elements?.message?.value ?? root.querySelector('[name=message]').value
    sending = true
    error = ''
    const id = session.id
    const key = randomUUID()
    try {
      await api(`/api/v1/sessions/${pathID(id)}/messages`, {method: 'POST', body: {content, request_key: key}})
    } catch (sendError) {
      error = sendError.message
    } finally {
      sending = false
      await poll()
    }
  }

  async function openChat(value) {
    stopPolling()
    destroyed = false
    session = value
    error = ''
    ++chatGeneration
    await poll()
    if (session === value && !destroyed && isCurrent()) timer = setInterval(poll, 1500)
  }

  function stopPolling() { if (timer !== null) clearInterval(timer); timer = null }
  function destroy() { destroyed = true; ++chatGeneration; stopPolling(); session = null }
  return {list, openChat, poll, destroy}
}
