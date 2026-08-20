# Phase B3 — Open session Amp + Grok rail (draft)

> Tasks 7–9. Spec §7. Supersedes session-focus design where rail/copy/composer conflict.

**Goal:** Open session in hub main = Amp tabs + sticky bottom composer + assistant copy; continuous ProjectRail; Back returns to hub start.

---

### Task 7: SessionChat continuous rail + Files → tabs

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/components/sessions/SessionChat.test.ts`
- May deprecate default-closed-only UX of SessionFilesBar as the sole files UI; prefer embedding `ProjectRail` **from hub** (Task 5 already mounts rail outside SessionChat).

**Architecture decision (locked for implementers):**
- **Rail lives on ProjectHubPage**, not inside SessionChat — so rail stays mounted across hub ↔ session without remount.
- SessionChat props gain: `onOpenFile` not required if Files tree is in hub rail; hub passes `onOpenFile` into ProjectRail that calls into SessionChat via bindable callback or small store.

**Chosen wiring:**
```ts
// ProjectHubPage
let openFileHandler = $state<(path: string) => void>(() => {})
// SessionChat exposes onMount register: onReady={{ openFile: (p) => ... }}
// Simpler: SessionChat accepts optional `externalOpenPath` bindable or:
let fileToOpen = $state<string | null>(null)
<ProjectRail onOpenFile={(p) => { fileToOpen = p }} />
<SessionChat bind:openPath={fileToOpen} ... />
```

Or SessionChat keeps internal files bar **in addition** when not in hub — but hub always provides rail. Spec: rail continuous. **Minimal path:**

1. When SessionChat is used inside hub, pass `embeddedInHub={true}` → hide internal files toggle/bar; hub ProjectRail drives `openPath`.
2. SessionChat `$effect` on `openPath` → open file tab (existing file tab logic).

**Tests:**
```ts
it('opens file tab when openPath prop is set', async () => {
  // render SessionChat with workspace enabled, set openPath to 'notes/a.md'
  // expect tab with name a.md
})
it('when embeddedInHub, does not show Show files toggle', () => {
  // queryByRole button /show files/i null
})
```

- [ ] **Step 1: Failing tests**
- [ ] **Step 2: Run**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/components/sessions/SessionChat.test.ts src/components/sessions/SessionChat.focus.test.ts
```

- [ ] **Step 3: Implement openPath + embeddedInHub**
- [ ] **Step 4: Pass (focus test still green)**
- [ ] **Step 5: Commit** `feat(web): session file tabs open from hub ProjectRail`

---

### Task 8: Dense bottom composer + assistant copy

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte` (Agent tab markup only; keep form ancestry)
- Modify: `web/src/app.css` — `.session-composer`, `.message-copy`
- Modify: `web/src/components/sessions/SessionChat.test.ts`
- Gate: `SessionChat.focus.test.ts` must remain green

**Composer chrome:**
- Remove visible "Message" label soup; use `aria-label="Message"` on textarea
- Classes: `session-composer` sticky bottom; `field-textarea` + `btn btn--primary` Send
- Still a single stable `<form>` wrapping textarea+button (focus test depends on this)

**Assistant copy:**
```svelte
{#if message.role === 'assistant' || message.role === 'model'}
  <div class="message-row message-row--assistant">
    <div class="message-prose">...</div>
    <button
      type="button"
      class="message-copy"
      aria-label="Copy response"
      onclick={() => copyAssistant(message.content)}
    >...</button>
  </div>
{/if}
```

```ts
async function copyAssistant(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedSeq = message.sequence // show "Copied" briefly
  } catch { /* ignore */ }
}
```

Polyfill clipboard in tests if needed:
```ts
Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
```

**Tests:**
```ts
it('copy control copies assistant plain text', async () => {
  // mock messages with assistant content
  await fireEvent.click(screen.getByRole('button', { name: 'Copy response' }))
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith('Hi — how can I help you today?')
})
it('composer has no visible Message label text node soup', () => {
  expect(screen.queryByText('Message', { selector: 'span.font-medium' })).toBeNull()
  expect(screen.getByLabelText('Message')).toBeInTheDocument()
})
```

- [ ] **Step 1: Failing tests**
- [ ] **Step 2: Run SessionChat tests + focus test**
- [ ] **Step 3: Implement CSS + markup (do not remount form)**
- [ ] **Step 4: Pass**
- [ ] **Step 5: Commit** `feat(web): dense session composer and assistant copy`

---

### Task 9: Hub embeds SessionChat; Back restores start stack

**Files:**
- Modify: `web/src/routes/ProjectHubPage.svelte` (if not complete in Task 5)
- Modify: `web/src/routes/ProjectHubPage.test.ts`

**Behaviors:**
- `activeSession` set → main shows SessionChat with `embeddedInHub` + `onclose` clears activeSession and reloads sessions
- ProjectRail stays mounted (sibling, not child of SessionChat)
- Back button in SessionChat calls onclose

**Tests:**
```ts
it('clicking a session row shows chat and Back returns to prompt', async () => {
  // mock sessions, click row, find Back, click, find How can I help you today
})
```

- [ ] **Step 1–5: TDD + commit** `feat(web): open session inside project hub canvas`

**Done B3:** Amp session + bottom composer + copy + rail continuous + focus tests green.
