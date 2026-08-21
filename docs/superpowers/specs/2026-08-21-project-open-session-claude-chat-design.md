# Design: Project open-session Claude chat canvas

**Status:** Approved (user 2026-08-21)  
**Date:** 2026-08-21  
**Scope:** `#/projects/:id` **open-session main canvas only**  
**Refs:** `layout-chat.png` (target), `current-layout.png` (failure)  
**Approach:** 1 — Claude chat canvas inside project; right Memory|Files rail unchanged  

---

## 1. Why

Prior benchmark passes shipped Amp tokens (tabs, sticky composer, copy) but the **open session main** still feels empty AI slop:

| Failure | Evidence (`current-layout.png`) |
|---------|----------------------------------|
| Blank composer | Textarea has **no placeholder** — looks broken |
| Full-bleed thread | Messages hug the left of a huge empty pane |
| Harsh user bubble | Solid accent blue pill, not soft Claude lavender |
| Form-soup composer | Labeled-field energy: tall textarea + trailing **Send** row |
| Heavy header | Back + title + model chip + run status compete with the thread |

User contract: match `layout-chat.png` for this surface. Skills/tokens alone did not fix UX hierarchy.

---

## 2. Goals / non-goals

### Goals

1. **Centered conversation column** (~680–720px) in the main canvas.
2. **Composer always shows `Reply…`** when draft is empty (visible placeholder).
3. **Claude-style composer card**: rounded surface, multi-line field, model chip left, primary send control right — not a bare form stack.
4. **User messages**: soft lavender bubble, dark text, right-aligned.
5. **Assistant**: bare markdown on canvas + existing copy control.
6. **Quiet header**: Back + title primary; model/run secondary (smaller/muted).
7. **Keep** Agent/file tabs, poll-safe composer ancestry, rail, focus regression.

### Non-goals (this pass)

- Hub start state (“How can I help…”) restyle  
- Right rail Memory|Files redesign or hide  
- Other routes (Notes, vault list, global Home)  
- Memory persist API  
- Dark mode  

---

## 3. Layout (open session main)

```text
┌─ App sidebar ─┬─ Main canvas ─────────────────────────┬─ Rail (unchanged) ─┐
│               │ Back · title          model · run quiet│ Memory | Files     │
│               │ [Agent] [files…] (quiet tabs)          │                    │
│               │                                        │                    │
│               │     ┌─ thread column ~720px ─────┐     │                    │
│               │     │ assistant md …       [Copy] │     │                    │
│               │     │              [user lavender] │     │                    │
│               │     └────────────────────────────┘     │                    │
│               │     ┌─ composer card ────────────┐     │                    │
│               │     │ Reply…                     │     │                    │
│               │     │ [model chip]            [↑] │     │                    │
│               │     └────────────────────────────┘     │                    │
└───────────────┴────────────────────────────────────────┴────────────────────┘
```

When a **file tab** is active: same as today (Preview/Source body); composer stays mounted but hidden (`hidden`/`inert`) — no ancestry destroy.

---

## 4. Component / token changes

| Piece | Change |
|-------|--------|
| `SessionChat.svelte` | `placeholder="Reply…"` on composer textarea; composer markup → card + model chip + send; header hierarchy quieter |
| `app.css` | Centered thread column; soft user bubble; composer card (no harsh full-width form bar); denser message rhythm |
| Tests | Assert placeholder; assert Claude chat tokens; keep `SessionChat.focus.test.ts` (label still accessible, e.g. `aria-label="Message"` or “Reply”) |

Prefer shared tokens in `app.css` over one-off indigo/scaffold classes.

### Composer interaction (unchanged behavior)

- Empty/whitespace: no send  
- Run active / sending: disable send  
- Poll: patch messages in place; never remount form  
- Enter: existing behavior retained (submit on form; no surprise Ctrl-only change unless already present)

### Colors (user bubble)

- Background: soft lavender (`#ede9fe` or equivalent token)  
- Text: near-black (`#18181b` / zinc-900)  
- **Not** solid brand blue fill with white text  

---

## 5. Acceptance (done gate)

1. Open project session at `#/projects/:id` (session open).  
2. Empty composer shows visible **`Reply…`**.  
3. Side-by-side vs `layout-chat.png`: centered column, lavender user bubbles, card composer, quiet header.  
4. Rail still present (Approach 1).  
5. `SessionChat` unit + focus tests green; styles baseline includes new tokens if added.  
6. Rebuild `web/dist` and vibe-pass with cache-bust (Go serves dist).  

---

## 6. Failure analysis for skills

`frontend-ui-craft` already requires benchmark fidelity, but prior work optimized for Amp/Grok chrome. This pass treats **`layout-chat.png` as the sole visual contract for open-session main** and prioritizes placeholder + column hierarchy over token checklist completeness.
