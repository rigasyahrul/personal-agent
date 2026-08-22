# Draft B — P1 Compound (Tasks 20–35)

> Assembled into final plan. Header Canonical contracts win.

---

### Task 20: Compound proposal store — create pending

**Files:**
- Create: `internal/store/compound.go`
- Create: `internal/store/compound_test.go`

**Interfaces:**
```go
type CompoundItem struct {
  Kind string `json:"kind"` // agents_patch|memory_detail|lessons_index_row
  Path string `json:"path"`
  Action string `json:"action"` // create|update
  Title string `json:"title,omitempty"`
  Content string `json:"content"`
  ContentSHA256 string `json:"content_sha256"`
}

type CreateProposalInput struct {
  SessionID string
  RequestKey string
  // Scope, ProjectID, VaultID MUST be filled by handler from session row only — never from client body
  Scope domain.CompoundScope
  ProjectID, VaultID string
  Items []CompoundItem
  Now time.Time
}

func (s *CompoundStore) CreatePending(ctx context.Context, in CreateProposalInput) (domain.CompoundProposal, error)
// Idempotent: same session_id+request_key+same items fingerprint → return existing
// Different fingerprint → ErrConflict
```

Validation via shared `ValidateCompoundItems` (Canonical LOCKED):
- Run on **CreatePending**, on **Decide(approve) final items** (including human edits), and again in **PublishApproved** before any write
- kind/path/action allowed for scope (vault cannot agents_patch)
- path allowlist ONLY `AGENTS.md` or `memory/**` — NEVER `source/**`, `.agents/**`, SOUL/SYSTEM
- memory_detail path regex `^memory/[0-9]{8}-[0-9]{4}-[a-z0-9-]+\.md$`
- if any memory_detail → require lessons_index_row path `memory/lessons.md`
- `ValidateKnowledgeRelPath` on every path (not promote `ValidateRelPath`)
- content sha256 must match sha256(content); max 256KiB/item; max 20 items
- Decide when already terminal → idempotent return; do not re-publish
- [ ] **Step 1: Failing tests** — happy path; vault rejects agents_patch; detail without index row rejects; path escape rejects.

- [ ] **Step 2–4:** implement, pass, commit `feat(store): compound proposal create pending`

---

### Task 21: Compound decide + timestamps

**Files:**
- Modify: `internal/store/compound.go`

**Interfaces:**
```go
type DecideInput struct {
  ProposalID string
  RequestKey string // decide idempotency optional separate key — use body request_key UNIQUE per decide op OR reuse: LOCKED use decide_request_key column optional; simpler: idempotent decide by proposal status
  Decision string // approve|reject
  Items []CompoundItem // optional edits replacing items_json when approve
  Now time.Time
}

func (s *CompoundStore) Decide(ctx context.Context, in DecideInput) (domain.CompoundProposal, error)
// reject: status=rejected, decided_at=now, finished_at=now
// approve: status=approved, decided_at=now, finished_at still null until publish completes
func (s *CompoundStore) MarkFinished(ctx context.Context, id string, status string, errMsg string, now time.Time) error
// status approved→ stays approved with finished_at; or failed with error+finished_at
func (s *CompoundStore) Get(ctx, id string) (domain.CompoundProposal, error)
func (s *CompoundStore) GetBySessionRequest(ctx, sessionID, requestKey string) (domain.CompoundProposal, error)
```

- [ ] Tests: reject sets both timestamps; approve sets decided_at only; MarkFinished sets finished_at.

- [ ] Commit: `feat(store): compound decide and finished_at`

---

### Task 22: Compound publish writer

**Files:**
- Create: `internal/compound/publish.go`
- Create: `internal/compound/publish_test.go`

**Interfaces:**
```go
type Publisher struct {
  DataDir string
  DB *sql.DB
  // knowledge upsert + optional link reparse hook
  Notes *store.KnowledgeStore // or InstructionStore + MemoryWriter
  Clock clock.Clock
  Barrier mutBarrier
}

func (p *Publisher) PublishApproved(ctx context.Context, proposal domain.CompoundProposal) error
// For each item: atomic write under scope root; upsert knowledge_notes; update lessons/agents
// Preserve AGENTS Memory block: if agents_patch content strips ## Memory pointer, reject before write (ValidateAgentsMemoryPointer)
// On full success caller MarkFinished approved; on error MarkFinished failed
```

Use temp+rename same volume patterns from `internal/publish`, but **knowledge FS strategy from Canonical**:
- Memory writes: open `MemoryDir(scopeRoot)` sub-root (or knowledge opener) — **never** `ValidateRelPath` on `memory/...` under project root via stock fsroot.
- AGENTS: instruction atomic write under scope root.
- **Forbid** loosening promote `ValidateRelPath` reserved memory/soul.
- All-or-nothing multi-item publish; Decide CAS `WHERE status='pending'`.
- [ ] Tests: approve agents_patch writes file; strips Memory block → error; memory detail + lessons row both land.

- [ ] Commit: `feat(compound): publish approved proposal items to disk`

---

### Task 23: Load compounding skill for session scope

**Files:**
- Create: `internal/agent/compound_skill.go`
- Test: `internal/agent/compound_skill_test.go`

```go
func LoadCompoundingSkill(dataDir string, home layout.SessionHome, vaultID, projectID string) (string, string, error)
// returns (markdown, sourcePathOr"embedded", err)
// missing/empty file → embedded default
```

- [ ] Commit: `feat(agent): load scoped compounding skill with embed fallback`

---

### Task 24: HTTP POST start compound (proposal from model — phase split)

**Design lock for P1:** Two substeps in product:

**24a UI/API path without LLM (testable):** client may POST items directly for tests/dev.  
**24b Agent path:** Runner mode `compound` loads skill + user_context and expects model to return JSON items (parse from assistant message fenced json).

**Files:**
- Create: `internal/httpapi/compound_handlers.go`
- Tests

```
POST /api/v1/sessions/{id}/compound
{ "request_key": "...", "user_context": "optional", "items": [ ... optional prebuilt ...] }
```

If `items` present → validate+CreatePending (no model).  
If `items` absent → start compound agent run (Task 25) OR return 501 until 25 done — **LOCKED:** implement items-present path first in this task; Task 25 adds generation.

- [ ] Test: POST items → 200 proposal pending.

- [ ] Commit: `feat(api): create compound proposal from items`

---

### Task 25: Compound generation via Runner

**Files:**
- Modify: `internal/agent/runner.go` or new `internal/agent/compound_run.go`
- HTTP: when items omitted, admit a run with system= skill + instructions to emit JSON array of items only; parse; CreatePending; return proposal id.

**Interfaces:**
```go
func ParseCompoundItemsFromAssistant(content string) ([]store.CompoundItem, error)
// extract first ```json ... ``` or raw JSON array
```

- [ ] Test with fake provider returning fixed JSON → proposal rows created.

- [ ] Commit: `feat(agent): generate compound proposal items from model`

---

### Task 26: HTTP GET proposal + decide

**Files:**
- Modify: `compound_handlers.go`

```
GET  /api/v1/sessions/{id}/compound/{proposal_id}
POST /api/v1/sessions/{id}/compound/{proposal_id}/decide
{ "request_key": "...", "decision": "approve"|"reject", "items": [optional edits] }
```

Decide approve → Publisher.PublishApproved → MarkFinished.  
All under auth+CSRF.

- [ ] Tests: reject; approve writes AGENTS on disk; wrong session 404.

- [ ] Commit: `feat(api): get and decide compound proposals`

---

### Task 27: Frontend API client types

**Files:**
- Modify: `web/src/lib/api/types.ts`
- Create: `web/src/lib/api/compound.ts`
- Test: unit parse types if applicable

```ts
export type CompoundItem = { kind: string; path: string; action: string; title?: string; content: string; content_sha256: string }
export type CompoundProposal = { id: string; status: string; items: CompoundItem[]; created_at: string; decided_at?: string; finished_at?: string }
export function createCompound(sessionId: string, body: {...}): Promise<CompoundProposal>
export function decideCompound(sessionId: string, proposalId: string, body: {...}): Promise<CompoundProposal>
```

- [ ] Commit: `feat(web): compound API client`

---

### Task 28: `CompoundReviewCard.svelte`

**Files:**
- Create: `web/src/components/sessions/CompoundReviewCard.svelte`
- Create: `web/src/components/sessions/CompoundReviewCard.test.ts`
- Styles: `web/src/app.css` — `.compound-card`, `.compound-item`, etc. tokens

**Props:**
```ts
proposal: CompoundProposal
onconfirm: (decision: 'approve'|'reject', items: CompoundItem[]) => void
oncancel?: () => void
busy?: boolean
```

UI: list items with path/kind; textarea edit content; approve/reject buttons; confirm.

- [ ] Test: renders paths; edit updates local items; approve callback.

- [ ] Commit: `feat(web): CompoundReviewCard for human gate`

---

### Task 29: Wire Compound control into SessionChat

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `SessionChat.focus.test.ts` — ensure compound UI does not remount composer

- [ ] Add **Compound** button (explicit only); on click POST compound with user_context=last messages summary or empty; show CompoundReviewCard when pending.
- [ ] On approve/reject call decide endpoint; toast/error.
- [ ] Focus regression test still PASS.

- [ ] Commit: `feat(web): explicit Compound action in session chat`

---

### Task 30: Vault/global compound scope enforcement tests

**Files:**
- `internal/store/compound_test.go` + API tests

- [ ] Project session cannot write vault memory paths outside project root.
- [ ] Vault session CreatePending accepts memory_detail under vault; rejects agents_patch.
- [ ] Global session writes global AGENTS/memory only.

- [ ] Commit: `test: compound scope enforcement`

---

### Task 31: AGENTS Memory pointer preservation helper

**Files:**
- Create: `internal/compound/agents_pointer.go`
- Test

```go
const AgentsMemoryMarker = "## Memory"
func EnsureAgentsMemoryPointer(content string) (string, error)
// if missing Memory section, append Canonical block; if present keep
func ValidateAgentsMemoryPointer(content string) error
// require [[memory/lessons
```

- [ ] Used by publish + optional Put instruction.

- [ ] Commit: `feat(compound): ensure AGENTS memory lessons pointer`

---

### Task 32: Replace ProjectRail fake memory textarea (read-only summary)

**Files:**
- Modify: `web/src/components/ProjectRail.svelte`
- Modify: `ProjectRail.test.ts`
- API: GET project knowledge read `memory/lessons.md` or instructions

- [ ] Remove non-persistent bind:value memory dump.
- [ ] Show lessons index preview (first N lines) + link “Open memory”.
- [ ] Empty state when no lessons.

- [ ] Commit: `fix(web): ProjectRail shows real memory index not fake textarea`

---

### Task 33: Compound timestamps metric helper (optional UI)

**Files:**
- Create: `web/src/lib/compoundMetrics.ts`
- Test: time-to-finish ms from created_at/finished_at ISO strings

- [ ] Commit: `feat(web): compound time-to-finish helper`

---

### Task 34: Integration test Go — full compound approve path

**Files:**
- Create: `internal/httpapi/compound_handlers_test.go` end-to-end with temp data dir

Flow: create project+session → POST items → decide approve → read AGENTS from disk contains rule + Memory pointer → proposal finished_at set.

- [ ] Commit: `test(api): compound approve end-to-end`

---

### Task 35: P1 verification gate

```bash
go test ./internal/compound/ ./internal/store/ ./internal/agent/ ./internal/httpapi/ -count=1
export PATH="$HOME/.local/node-v22/bin:/usr/bin:$PATH"
npm --prefix web test -- CompoundReviewCard SessionChat ProjectRail
```

- [ ] Expected PASS  
- [ ] Commit empty or changelog: `test: P1 compound verification gate`

DRAFT_B_COMPLETE
