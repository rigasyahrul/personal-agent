## Vault projects (claude-2)

The vault Projects route uses a compact, name-first list rather than fat entity cards. Each row gives the project name visual priority as the hero text. It may include one muted metadata line—or reveal secondary metadata on hover—and may end with a chevron. The whole row is the navigation target and opens that project's hub.

The page header contains the vault-name eyebrow, the title **Projects**, and a **New project** action. New project opens a modal rather than inserting a form into the page. When the vault has no projects, show a purposeful empty state with a single primary action that opens the same modal.

Reference intent: match `claude-2` in hierarchy, row density, whitespace, and restrained secondary chrome; do not reinterpret the reference as the existing entity-card pattern.

## Open session (Amp + Grok rail)

Opening a session—either by sending the hub quick-start prompt or clicking a session row—replaces the hub main content with the session surface while preserving the right rail as a continuous part of the project workspace.

### Main pane (Amp)

The session header contains:

- **Back**, returning to the project hub
- Session title
- Model chip
- Current run status
- No duplicate Files control when the right rail is open; Files belongs to the rail

The tab strip always contains an unclosable **Agent** tab. Opening workspace files adds closable file tabs labelled by basename. Selecting an already-open file focuses its existing tab. Keep at most eight file tabs; when a ninth distinct file is opened, evict the least-recently-used file tab. The Agent tab does not count toward the limit.

#### Agent tab body

- Messages occupy the scrollable region.
- User messages are end-aligned bubbles.
- Assistant responses are bare Markdown on the canvas: no assistant bubble and no “Assistant” label.
- Every assistant response container has a small, focusable copy icon. Activating it copies the full plain-text content of that reply and briefly changes to or announces **Copied** feedback.
- A dense Amp-style composer is sticky at the **bottom** of the Agent pane. It consists of a multi-line textarea and **Send** action, without a large “Message” label or stacked form-card chrome.
- The composer is disabled while a run is active.
- Polling must patch messages, status, and disabled state without remounting the composer. Its DOM identity, draft, focus, selection, and scroll continuity must survive polls.
- When a File tab is active, hide rather than destroy the composer and Agent body state. Returning to Agent restores the draft and prior state.

#### File tab body

The file toolbar provides **Preview | Source**, with Preview selected by default. Content is read-only in both modes. When the file is promotable, show **Save to source**, wired to the existing `PromoteDialog`. Loading and read errors remain local to the tab.

Clicking a file in the right-rail Files tree opens or focuses that file's tab in the main pane; the rail itself does not become a second preview pane.

### Right rail (Grok)

The project-hub rail continues into the open-session view and is open by default. Its header exposes two modes:

- **Memory** — render the approved field design now; persistence and backing APIs come later.
- **Files** — show an Amp-style hierarchical directory tree. Clicking a file opens or focuses its main-pane tab.

The application left navigation is unchanged and only collapses when the user explicitly controls it. Session entry must not auto-collapse it.

### Open-session wireframe

```text
┌─ App shell ─────────────────────────────────────────────────────────────────┐
│ Left nav │ Back · Session title · [model] · run status                     │
│ unchanged├───────────────────────────────────────┬──────────────────────────┤
│          │ [Agent] [notes.md ×] [plan.md ×]     │ [Memory] [Files]         │
│          ├───────────────────────────────────────┤ Right rail (default open)│
│          │                                       │                          │
│          │  Assistant markdown on canvas   [⧉] │ Memory fields             │
│          │                                       │          or              │
│          │                     ┌─ user bubble ┐  │ Files directory tree     │
│          │                     └──────────────┘  │                          │
│          │                                       │                          │
│          │        SCROLLABLE MESSAGE AREA        │                          │
│          │                                       │                          │
│          ├───────────────────────────────────────┤                          │
│          │ ┌─ BOTTOM COMPOSER (sticky) ────────┐│                          │
│          │ │ Multi-line draft             Send ││                          │
│          │ └────────────────────────────────────┘│                          │
└──────────┴───────────────────────────────────────┴──────────────────────────┘
```

## Modals

Use one shared modal primitive for create and options flows. It must provide a backdrop, Escape dismissal, focus containment, focus return to the invoking control, and clear primary and secondary actions. Backdrop dismissal may be supported where it cannot discard an in-flight submission; validation and submission errors stay inside the modal.

The primitive supports:

- **New project**, invoked globally or within a vault, with the invoking context supplying the destination vault/defaults
- **New vault**
- An optional session **More options** modal for model and `workspace_files`; the project-hub quick-start path otherwise uses defaults
- Existing **PromoteDialog**, which remains the dedicated promotion confirmation flow rather than being replaced

Create forms must not expand inline into list pages or the project hub.

## Delivery phases

| Phase | Deliverable |
|---|---|
| **A — Shell** | Establish benchmark-faithful shell geometry, compact navigation, main/rail layout, responsive behavior, and shared tokens. |
| **B1 — Modals** | Build the shared modal primitive and move New vault/New project creation into it; retain PromoteDialog and add optional session options. |
| **B2 — Hub** | Deliver the project hub quick-start prompt, session rows, and default-open Memory/Files rail. |
| **B3 — Open session** | Deliver Amp tabs, message canvas, assistant copy, stable bottom composer, file tabs, and continuous Grok rail. |
| **B4 — Vault list** | Deliver the `claude-2` name-first vault project rows, navigation, empty state, and modal create entry. |
| **B5 — Vibe-pass + skill gate** | Compare named screens against all supplied benchmark screenshots at representative viewport sizes, fix fidelity gaps, and update the frontend craft process gate. |

Memory/soul persistence and related APIs are a later phase. The present delivery may render the field design but must not imply persistence that does not exist.

## Benchmark acceptance (hard done gate)

The redesign is not done merely because it uses shared tokens or passes component tests. Completion requires a screenshot-based fidelity review against each named reference.

| Reference | Measurable acceptance check |
|---|---|
| `claude` | App navigation is compact and no navigation row/control exceeds **44px** in height; visual hierarchy and whitespace are demonstrably comparable in side-by-side screenshots. |
| `claude-2` | Vault Projects renders project-name-first list rows—not fat cards—with at most one muted metadata line or hover metadata; create actions open modals. |
| `grok` | The project hub places the quick-start prompt first with recent session rows directly below it; the right rail is open by default. |
| `grok-2` | The default-open right rail exposes clearly selected **Memory | Files** modes and maintains the reference's narrow auxiliary-rail hierarchy rather than a second main panel. |
| `amp` | Open session always has an **Agent** tab and a sticky **bottom composer** visible on Agent; file tabs open from the tree; assistant responses are bare Markdown and each has a keyboard-focusable full-reply copy action with Copied feedback. |

Additional hard checks:

- [ ] App left navigation remains user-controlled and is never auto-collapsed by hub/session navigation.
- [ ] Hub screenshot shows prompt followed by session rows, not a detached create form or card grid.
- [ ] Open-session screenshot explicitly shows the composer at the bottom of the Agent pane.
- [ ] Composer DOM identity and draft survive polling and Agent/File tab switches.
- [ ] File-tab cap is eight with least-recently-used eviction.
- [ ] Vault empty state and all New vault/New project entry points invoke modals.
- [ ] Desktop and narrow-viewport screenshots are compared to the named references; material spacing, density, hierarchy, or control-placement differences are fixed before sign-off.

## Skill / process change

Update `frontend-ui-craft` so that when a user supplies or names benchmark screenshots, the skill requires explicit screenshot-fidelity acceptance criteria and a side-by-side browser vibe-pass against every named reference. The completion report must identify each reference checked and any intentional deviations. Shared tokens, semantic components, automated tests, and the absence of scaffold styling remain necessary, but **tokens alone are not evidence that the UI matches the benchmark**. A blocked screenshot comparison is reported as blocked, never passed.

## Supersession

This section extends `2026-08-20-session-focus-layout-design.md` for the benchmark-led redesign and supersedes that document where their session decisions conflict. In particular, it supersedes the prior default-closed, independently toggled Files bar with a **default-open continuous Grok rail** containing Memory and Files; moves file access into that rail; and adds the per-assistant-response copy action. It also makes the bottom composer's presence and poll-safe non-remount behavior an explicit benchmark gate.

The earlier document remains authoritative for compatible implementation details such as Agent/file tab behavior, the eight-tab LRU cap, read-only Preview/Source, promotability through `PromoteDialog`, safe Markdown rendering, run-disabled sending, and preserved Agent state. Its unrelated session-list and files-bar presentation decisions yield to this benchmark redesign wherever the assembled design specifies otherwise.
