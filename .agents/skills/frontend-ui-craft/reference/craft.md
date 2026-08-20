# Frontend UI craft — reference

Load when polishing UI, when red flags fire, or when the thin recipe in `SKILL.md` is not enough. Not a full design system.

## Anti-AI-slop patterns

### Scaffold chrome

| Smell | Prefer |
|-------|--------|
| `•` or emoji as nav icons | Consistent icon set (SVG/icon component), same optical size |
| Collapse control as raw `‹`/`›` text with no alignment | Labeled button (`aria-label`), aligned to brand/nav column |
| "Storage status unavailable" as a lonely floating pill | Integrate into top bar hierarchy; muted when unknown, not alarming by default |
| Disabled nav item, no reason | Hide, or enable with destination, or tooltip/title explaining why |

### Default stack soup

Agents default to: Inter + indigo-600 buttons + slate-500 labels + `rounded-xl` white cards + `space-y-6/8` + gray canvas.

That stack is fine as a **baseline** if hierarchy and density are intentional. It becomes slop when:

- Every surface uses the same radius, border, and padding (metrics = CTAs = empty states)
- Primary and secondary buttons only differ by fill vs border with mismatched padding
- Every page repeats the same eyebrow + H1 + two stat cards + one content card pattern without purpose

**Move:** vary weight (type size/weight, border strength, fill vs outline), not random colors. One accent. Semantic success/warning/danger only for status.

### Desert layout

| Smell | Prefer |
|-------|--------|
| Content band at top, 60%+ empty viewport | Tighter vertical rhythm; optional secondary sections; max-width canvas that doesn't stretch metrics into billboards |
| One narrow card under full-width stat row | Shared grid columns (e.g. 12-col mental model); align card edges |
| Huge padding "to look clean" | Medium-dense dashboard: 16–24px page padding, 12–16px card padding unless content needs air |

### Weak hierarchy

- **Metrics vs actions:** stats can be quieter (smaller type, less chrome); clickable destinations need clearer affordance (hover, focus, chevron/label "Open").
- **One primary action per view** when possible; secondary actions visually quieter.
- **Eyebrows** ("Global desk") — keep only if they disambiguate context (global vs vault); drop if they only repeat the H1.

### Interaction hostility

- Polled UIs: never tear down focused inputs (see `AGENTS.md` sessions rule).
- Dialogs: Esc closes; focus returns to opener.
- Forms: inline errors near fields; page-level alert only for hard fail.

## Screenshot / snapshot review prompts

When viewing a screenshot or a11y snapshot, answer explicitly:

1. What is the **one** primary action on this screen?
2. Can I see **where I am** in the app (nav current, breadcrumbs)?
3. Is empty space **intentional** or leftover?
4. Do cards of different jobs **look different**?
5. Would a tired user know what to click first in under 2 seconds?
6. Any **scaffold** leftovers (bullets, orphan glyphs, lorem, "unavailable" noise)?

If any answer is weak → not done.

## Vibe-user script (naive human)

For the surface you changed, spend one short pass as a confused user:

1. Land on the URL cold — is the title/purpose obvious?
2. Tab once or twice — is focus visible?
3. Hit the primary action — does something clear happen?
4. Force empty or error if you can (empty search, bad submit) — is the state kind and useful?
5. If chat/composer: type, wait across a poll interval — focus still there?

Report what felt broken in plain language; fix or file before "done."

## Personal-agent chrome notes

**Baseline (pre-polish 2026-08-20)** — fixed in shell/home + full-surface craft commits; do not reintroduce:

- Sidebar nav prefix `•` instead of icons → SVG `nav-icon` set
- Collapse control as orphan `‹` under nav → labeled `.sidebar__collapse`
- Top bar health pill “Storage status unavailable” → muted/ok/warn from `/health`
- Home/hub: metric ≡ destination weight → `metric-card` vs `destination-card` / `entity-card`
- Global Sessions disabled with no explanation → title + `aria-description`
- Catalogs/review/settings/sessions: raw `bg-indigo-600` + “Global desk” eyebrows → shared `btn--*` + page-header

**Still enforce when editing any screen:**

- Prefer `web/src/app.css` primitives over Tailwind indigo/scaffold soup
- Rebuild `web/dist` + cache-bust before claiming vibe-pass on Go-served `:8080`
- SessionChat: classes only; never remount composer on poll

Product IA remains  
`docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`.  
Token source of truth: `web/src/app.css`.

## Disposable UI drafts

If after a vibe-pass the layout is still generic soup:

1. Re-state the screen spec in one tighter paragraph (hierarchy + primary action + states).
2. Regenerate the component/section rather than ten micro-patches.
3. Re-run vibe-pass.

Code is cheap; confusing chrome is expensive.

## Benchmark fidelity (named screenshots)

When the user names or supplies reference images (e.g. `claude.png`, `grok.png`, `amp.png`):

1. Freeze a **fidelity table**: region → required structure (not pixel-perfect).
2. Open the real product URL **and** view each ref (local paths preferred).
3. Side-by-side check every named ref before claiming done.
4. Completion report: list each ref + pass/fail structural notes + intentional deviations.
5. **Tokens / green tests alone do not pass** this gate.

Personal-agent benchmark redesign (2026-08-20):

| Ref | Structure |
|-----|-----------|
| Shell | Nav rows ≤44px; `.sidebar nav` packs with `align-content: start` |
| `claude.png` | Project hub: “How can I help you today?” + composer; session rows **below**; no left session column; no metric/destination grid |
| `claude-2.png` | Vault projects: name-first rows; create via modal |
| `grok.png` / `grok-2.png` | Right rail default open; **Memory \| Files** header tabs |
| `amp.png` | Agent + file tabs; sticky **bottom** composer; assistant copy control |

Spec/plan: `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md`, `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md`.

## What this reference is not

- Not a component library API
- Not permission to expand product scope
- Not a substitute for opening the real app
- Not a license to treat token polish as screenshot fidelity
