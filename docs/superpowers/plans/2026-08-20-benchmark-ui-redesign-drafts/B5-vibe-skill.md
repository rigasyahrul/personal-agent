# Phase B5 — Craft skill gate + vibe-pass (draft)

> Tasks 11–12. Spec §10–11.

### Task 11: frontend-ui-craft benchmark fidelity gate

**Files:**
- Modify: `.agents/skills/frontend-ui-craft/SKILL.md`
- Modify: `.agents/skills/frontend-ui-craft/reference/craft.md`
- Optional: `.agents/skills/frontend-ui-craft/baseline-red.md` note

**Add to SKILL.md (Mandatory loop / Red flags):**

| Red flag | Why |
|----------|-----|
| User named benchmark screenshots but agent only checked tokens/classes | Tokens ≠ fidelity |
| Claimed vibe-pass without side-by-side vs each named ref | Guessing |
| Blocked browser treated as passed | Blocked ≠ passed |

**Add Positive recipe item:** When refs are named (`claude.png`, `amp.png`, etc.), completion report must list each ref + structural checks (layout regions, not pixel-perfect).

**craft.md section "Benchmark fidelity":**
- Require short fidelity criteria table in screen spec when refs exist
- Side-by-side: open product URL + view ref images
- personal-agent benchmark redesign refs: `.amp/in/artifacts/{claude,claude-2,grok,grok-2,amp}.png` or repo root

- [ ] **Step 1: Edit skill files** (no test required beyond grep/self-check that wording exists)
- [ ] **Step 2: Commit** `docs(skills): require benchmark screenshot fidelity in frontend-ui-craft`

---

### Task 12: Full vibe-pass + dist + checklist

**Files:** none required beyond rebuild artifacts; fix any gaps found in prior tasks.

**Commands:**
```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test
npm run build
# confirm dist hashes
grep -E 'assets/.*\.(js|css)' dist/index.html
# if Go serves :8080, restart or ensure dist copied as project expects
```

**Browser checklist (password login if needed):**

| Check | Pass? |
|-------|-------|
| Nav item height ≤ 44px @ 1440×900 | |
| Hub: “How can I help you today?” + composer + sessions **below** | |
| No metric/destination card grid on hub | |
| No “New session” button | |
| Right rail default open; Memory \| Files tabs | |
| Memory shows non-persistent helper; no fake Save success | |
| Files tree or empty copy | |
| Open session: Agent tab + sticky **bottom** composer | |
| Assistant message has Copy control | |
| Composer has no fat “Message” label soup | |
| Vault projects: name-first rows | |
| New project / New vault open `role=dialog` | |
| `#/projects/:id/sessions` shows hub | |
| SessionChat.focus still green in CI | |

Compare against: `claude.png`, `claude-2.png`, `grok.png`, `grok-2.png`, `amp.png`.

- [ ] **Step 1: Full web test suite green**
- [ ] **Step 2: Build dist; cache-bust vibe-pass each surface**
- [ ] **Step 3: Fix any fidelity gaps (loop to earlier tasks)**
- [ ] **Step 4: Commit remaining + push only if user allows**

**Done B5:** skill updated; vibe-pass evidence recorded in final worker summary (URL + what checked).
