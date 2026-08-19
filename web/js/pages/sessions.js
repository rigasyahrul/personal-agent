import {renderWorkspacePanel} from '../components/workspace.mjs'
import {workspaceTree, workspaceFile, promoteSession, operationStatus, retryReviewPending} from '../api.js'
import {operationBadge} from '../components/status-badges.js'

const esc = value => String(value ?? '').replace(/[&<>"']/g, char => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[char]))
const pathID = value => encodeURIComponent(value)

function workspaceEnabled(session) {
  if (session?.tool_grants && typeof session.tool_grants === 'object') return session.tool_grants.workspace_files === true
  if (typeof session?.tool_grants_json !== 'string') return false
  try { return JSON.parse(session.tool_grants_json)?.workspace_files === true } catch { return false }
}
export const isPromotableWorkspaceFile = entry => entry?.kind === 'file' && entry.path.endsWith('.md')
export function nextPromoteAttempt(previous,payload,uuid){const fingerprint=JSON.stringify(payload);return previous?.fingerprint===fingerprint?previous:{fingerprint,key:uuid(),payload}}
const operationStorageKey=sessionID=>`personal-agent:v1:promote-operations:${sessionID}`

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
  promote = promoteSession,
  getOperation = operationStatus,
  retryPending = retryReviewPending,
  storage,
  document = globalThis.document,
  dialogHost,
}) {
  if (storage === undefined) { try { storage = globalThis.localStorage } catch { storage = null } }
  if (!dialogHost) dialogHost = document?.body
  let session = null
  let timer = null
  let sending = false
  let error = ''
  let operationError = ''
  let destroyed = false
  let chatGeneration = 0
  let messages = []
  let run = null
  let pollPromise = null
  let pollQueued = false
  let sendToken = null
  let pollFailed = false
  let selectedFile = null, promoteAttempt = null, operations = [], operationResults = new Map(), promoteDialog = null
  let operationPollPromise = null, operationPollQueued = false
  let renderedSessionID = null
  const retryingPending = new Set()

  function current(generation) { return !destroyed && isCurrent() && generation === chatGeneration }

  async function list() {
    stopPolling()
    closePromoteDialog()
    session = null
    renderedSessionID = null
    operationError = ''
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

  function messagesEqual(left, right) {
    if (left === right) return true
    if (!left || !right || left.length !== right.length) return false
    for (let index = 0; index < left.length; index++) {
      const a = left[index], b = right[index]
      if (a?.sequence !== b?.sequence || a?.role !== b?.role || a?.content !== b?.content) return false
    }
    return true
  }

  function runStatusOf(value) { return value?.status ?? null }
  function messageListHTML() {
    return [...messages].sort((a, b) => a.sequence - b.sequence).map(message => `<li class="message message-${esc(message.role)}"><strong>${esc(message.role)}</strong><p>${esc(message.content)}</p></li>`).join('')
  }
  function runStatusText() { return run ? `Run: ${run.status}` : 'Idle' }
  function sendDisabled() { return Boolean(sending || run) }

  function captureComposer(preserveDraft) {
    const input = root.querySelector('[name=message]')
    if (!input) return {draft: '', hadFocus: false, selectionStart: null, selectionEnd: null}
    const hadFocus = document?.activeElement === input
    return {
      draft: preserveDraft ? (input.value || '') : '',
      hadFocus,
      selectionStart: hadFocus ? input.selectionStart ?? null : null,
      selectionEnd: hadFocus ? input.selectionEnd ?? null : null,
    }
  }

  function restoreComposer(input, composer) {
    if (!input || !composer) return
    input.value = composer.draft
    if (!composer.hadFocus) return
    input.focus?.()
    if (composer.selectionStart == null) return
    try {
      if (typeof input.setSelectionRange === 'function') input.setSelectionRange(composer.selectionStart, composer.selectionEnd ?? composer.selectionStart)
      else {
        input.selectionStart = composer.selectionStart
        input.selectionEnd = composer.selectionEnd ?? composer.selectionStart
      }
    } catch { /* selection APIs are best-effort across hosts */ }
  }

  // Patch live chat chrome without replacing the message textarea (keeps focus/caret).
  function patchChat(options = {}) {
    const form = root.querySelector('form[data-chat]')
    if (!form || renderedSessionID !== session?.id) return false
    const list = root.querySelector('ol.messages')
    if (list) list.innerHTML = messageListHTML()
    const status = root.querySelector('.run-status')
    if (status) status.textContent = runStatusText()
    updateChatAlert()
    const button = root.querySelector('form[data-chat] button')
    if (button) button.disabled = sendDisabled()
    if (options.clearDraft) {
      const input = root.querySelector('[name=message]')
      if (input) input.value = ''
    }
    return true
  }

  function renderChat(preserveDraft = true) {
    const composer = captureComposer(preserveDraft)
    const workspace = workspaceEnabled(session) ? '<aside data-workspace-panel></aside>' : ''
    root.innerHTML = `<div class="session-layout"><section class="sessions-chat"><button type="button" data-back>Sessions</button><div class="page-heading"><h2>${esc(session.title)}</h2><span class="model-badge">${esc(session.provider)}:${esc(session.model_id)}</span></div><div data-operation-statuses></div><ol class="messages">${messageListHTML()}</ol><p class="run-status" role="status" aria-live="polite">${esc(runStatusText())}</p><p class="error" role="alert" data-chat-alert>${esc(chatErrorText())}</p><form data-chat><label>Message<textarea name="message" required></textarea></label><button ${sendDisabled() ? 'disabled' : ''}>Send</button></form></section>${workspace}</div>`
    renderedSessionID = session?.id ?? null
    restoreComposer(root.querySelector('[name=message]'), composer)
    root.querySelector('[data-back]')?.addEventListener('click', () => { void list().catch(listError => { if (!destroyed && isCurrent()) root.innerHTML = `<p class="error" role="alert">${esc(listError.message)}</p>` }) })
    root.querySelector('form[data-chat]').onsubmit = send
    renderOperations()
  }

  function chatErrorText(){return [operationError,error].filter(Boolean).join(' — ')}
  function updateChatAlert(){const alert=root.querySelector('[data-chat-alert]');if(alert)alert.textContent=chatErrorText()}
  function renderOperations(){const host=root.querySelector('[data-operation-statuses]');if(!host||!document?.createElement)return;host.replaceChildren(...operations.map(id=>operationResults.has(id)?operationBadge(operationResults.get(id),op=>retryCards(op),{retryDisabled:retryingPending.has(operationResults.get(id).pending_id)}):document.createTextNode('Promoting…')))}
  function paintChat(preserveDraft = true, options = {}) {
    if (root.querySelector('form[data-chat]') && renderedSessionID === session?.id) {
      if (options.clearDraft || !preserveDraft) patchChat({clearDraft: true})
      else patchChat()
      return
    }
    renderChat(preserveDraft)
  }
  function saveOperations(){try{storage?.setItem(operationStorageKey(session.id),JSON.stringify(operations))}catch{}}
  async function pollOperations(){if(!session||destroyed)return;operationPollQueued=true;if(operationPollPromise)return operationPollPromise;operationPollPromise=(async()=>{while(operationPollQueued){operationPollQueued=false;const id=session?.id,generation=chatGeneration;if(!id)continue;const active=operations.filter(operationID=>{const value=operationResults.get(operationID);return !value||!['Ready','Promote failed — Retry','Cards failed — Retry cards'].includes(value.badge)});let failed=false,nextOperationError='';await Promise.all(active.map(async operationID=>{try{const value=await getOperation(operationID);if(current(generation)&&session?.id===id)operationResults.set(operationID,value)}catch(reason){if(current(generation)&&session?.id===id){nextOperationError=reason.message;failed=true}}}));if(current(generation)&&session?.id===id){operationError=failed?nextOperationError:'';updateChatAlert();renderOperations()}}})().finally(()=>{operationPollPromise=null});return operationPollPromise}
  async function retryCards(op){const pendingID=op?.pending_id;if(!op?.retry_cards||!pendingID||retryingPending.has(pendingID))return;const generation=chatGeneration,id=session?.id;retryingPending.add(pendingID);renderOperations();try{await retryPending(pendingID);if(current(generation)&&session?.id===id)operationResults.delete(op.operation_id)}catch(reason){if(current(generation)&&session?.id===id){error=reason.message;paintChat()}}finally{retryingPending.delete(pendingID);if(current(generation)&&session?.id===id){renderOperations();await pollOperations()}}}

  function closePromoteDialog(){const dialog=promoteDialog;promoteDialog=null;if(!dialog)return;try{if(dialog.open)dialog.close()}catch{}dialog.remove?.()}

  async function openPromoteModal(){if(!isPromotableWorkspaceFile(selectedFile)||!session)return;const sourceFile=Object.freeze({...selectedFile}),sessionID=session.id,generation=chatGeneration;let projectName=projectID;try{projectName=(await api(`/api/v1/projects/${pathID(projectID)}`))?.name||projectID}catch{}if(!current(generation)||session?.id!==sessionID||!isPromotableWorkspaceFile(sourceFile))return
    closePromoteDialog();const dialog=document.createElement('dialog');promoteDialog=dialog;dialog.className='promote-dialog';const form=document.createElement('form');form.method='dialog';const heading=document.createElement('h2');heading.textContent='Save to source';const project=document.createElement('p');project.textContent=`Project: ${projectName}`;const targetLabel=document.createElement('label');targetLabel.textContent='Target path';const target=document.createElement('input');target.name='target_relative_path';target.required=true;target.value=sourceFile.path;targetLabel.append(target);const modes=document.createElement('fieldset'),legend=document.createElement('legend');legend.textContent='Review mode';modes.append(legend);for(const value of['none','whole','bites']){const label=document.createElement('label'),radio=document.createElement('input');radio.type='radio';radio.name='review_mode';radio.value=value;radio.checked=value==='none';label.append(radio,document.createTextNode(value));modes.append(label)}const inline=document.createElement('p');inline.className='error';inline.setAttribute('role','alert');const submit=document.createElement('button');submit.type='submit';submit.textContent='Save';const cancel=document.createElement('button');cancel.type='button';cancel.textContent='Cancel';cancel.onclick=closePromoteDialog;dialog.addEventListener?.('close',()=>{if(promoteDialog===dialog){promoteDialog=null;dialog.remove?.()}});form.append(heading,project,targetLabel,modes,inline,submit,cancel);dialog.append(form);dialogHost?.append(dialog)
    form.onsubmit=async event=>{event.preventDefault();if(!current(generation)||session?.id!==sessionID||promoteDialog!==dialog)return;const targetPath=target.value.trim();if(!targetPath||!targetPath.endsWith('.md')){inline.textContent='Target path must end in .md';return}const reviewMode=form.querySelector('[name=review_mode]:checked').value,payload={workspace_path:sourceFile.path,target_relative_path:targetPath,review_mode:reviewMode};promoteAttempt=nextPromoteAttempt(promoteAttempt,payload,randomUUID);submit.disabled=true;inline.textContent='';try{const result=await promote(sessionID,promoteAttempt.payload,promoteAttempt.key);if(!current(generation)||session?.id!==sessionID||promoteDialog!==dialog)return;if(!result?.operation_id)throw new Error('Promotion did not return an operation ID');if(!operations.includes(result.operation_id))operations.push(result.operation_id);saveOperations();promoteAttempt=null;closePromoteDialog();await pollOperations()}catch(reason){if(current(generation)&&session?.id===sessionID&&promoteDialog===dialog){inline.textContent=reason.message;submit.disabled=false}}};dialog.showModal()
  }

  async function refreshWorkspace(generation, id) {
    if (!workspaceEnabled(session)) return
    const container = root.querySelector('[data-workspace-panel]')
    if (!container) return
    const showSave=(entry,panel=container.querySelector('.workspace-panel'))=>{if(!isPromotableWorkspaceFile(entry)||!panel||panel.querySelector?.('[data-promote]'))return;const save=document.createElement('button');save.type='button';save.dataset.promote='';save.textContent='Save to source';save.onclick=()=>void openPromoteModal();panel.append(save)}
    await renderWorkspace({container, sessionID: id, messages, api: workspaceAPI, isCurrent: () => current(generation) && session?.id === id,onTree:(entries,panel)=>{if(selectedFile){const same=entries.find(entry=>entry.path===selectedFile.path&&isPromotableWorkspaceFile(entry));selectedFile=same||null;if(same)showSave(same,panel)}},onFileSelected:entry=>{selectedFile=entry;if(isPromotableWorkspaceFile(entry))showSave(entry)}})
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
            const nextList = nextMessages || []
            const messagesChanged = !messagesEqual(messages, nextList)
            const runChanged = runStatusOf(run) !== runStatusOf(nextRun)
            const clearingError = pollFailed && Boolean(error)
            const needsShell = !root.querySelector('form[data-chat]') || renderedSessionID !== id
            const desiredDisabled = Boolean(sending || nextRun)
            const sendStateChanged = Boolean(root.querySelector('form[data-chat]')) && renderedSessionID === id && Boolean(root.querySelector('form[data-chat] button')?.disabled) !== desiredDisabled
            messages = nextList
            run = nextRun
            if (pollFailed) error = ''
            pollFailed = false
            if (needsShell) renderChat()
            else if (messagesChanged || runChanged || sendStateChanged || clearingError) patchChat()
            void pollOperations()
            if (workspaceEnabled(session) && (needsShell || messagesChanged)) await refreshWorkspace(generation, id)
          }
        } catch (pollError) {
          if (current(generation) && session?.id === id) {
            error = pollError.message
            pollFailed = true
            paintChat()
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
    paintChat()
    try {
      await api(`/api/v1/sessions/${pathID(id)}/messages`, {method: 'POST', body: {content, request_key: key}})
      if (sendToken === token && current(generation) && session?.id === id) paintChat(false, {clearDraft: true})
    } catch (sendError) {
      if (sendToken === token && current(generation) && session?.id === id) {
        error = sendError.message
        pollFailed = false
      }
    } finally {
      if (sendToken === token && current(generation) && session?.id === id) {
        sending = false
        sendToken = null
        paintChat()
        await poll()
      }
    }
  }

  async function openChat(value) {
    stopPolling()
    closePromoteDialog()
    destroyed = false
    session = value
    renderedSessionID = null
    error = ''
    operationError = ''
    pollFailed = false
    messages = []
    run = null
    sending = false
    sendToken = null
    selectedFile=null;promoteAttempt=null;operationResults=new Map();retryingPending.clear();operationPollQueued=false;try{const stored=JSON.parse(storage?.getItem(operationStorageKey(value.id))||'[]');operations=Array.isArray(stored)?stored.filter(x=>typeof x==='string'):[]}catch{operations=[]}
    const generation = ++chatGeneration
    await poll()
    if (current(generation) && session?.id === value.id && timer === null) timer = setInterval(() => { void poll().catch(() => {}) }, 1500)
  }

  function stopPolling() { if (timer !== null) clearInterval(timer); timer = null }
  function destroy() { destroyed = true; ++chatGeneration; stopPolling(); closePromoteDialog();retryingPending.clear();operationPollQueued=false;error='';operationError='';session = null; renderedSessionID = null }
  return {list, openChat, poll, destroy}
}
