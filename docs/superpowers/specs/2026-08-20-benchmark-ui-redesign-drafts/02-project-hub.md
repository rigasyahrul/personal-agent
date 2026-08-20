# Project Hub

## Purpose

The project hub at `#/projects/:id` is the canonical place to start work or resume a project session. It keeps the global app sidebar, removes the intermediate session-management layout, and puts the prompt composer and recent sessions in one continuous main canvas.

## Layout

The global app sidebar/navigation remains unchanged. Inside the project content area there are exactly two containers:

1. **Main canvas** — start prompt, composer, session list, and the active Amp session shell.
2. **Right rail** — open by default, with **Memory** and **Files** tabs.

There is no left session-list column. The hub and session surfaces use a fuller available width than the standard 1120px canvas cap, while retaining readable inner line lengths for message content and form fields.

```text
┌──────────────────┬────────────────────────────────────────────────────┬──────────────────────────┐
│ APP SIDEBAR      │ PROJECT HEADER                                     │ Memory | Files           │
│                  │ Project name                    Notes · Review       ├──────────────────────────┤
│ Dashboard        ├────────────────────────────────────────────────────┤ MEMORY                   │
│ Projects         │                                                    │                          │
│ …                │ How can I help you today?                           │ Memory                   │
│                  │                                                    │ ┌──────────────────────┐ │
│                  │ ┌────────────────────────────────────────────────┐ │ │ labeled field        │ │
│                  │ │ Multi-line prompt…                             │ │ └──────────────────────┘ │
│                  │ │                                                │ │                          │
│                  │ │                                      [ Send ]  │ │ Instructions (system)    │
│                  │ └────────────────────────────────────────────────┘ │ ┌──────────────────────┐ │
│                  │                                                    │ │ labeled field        │ │
│                  │ Your sessions                                      │ └──────────────────────┘ │
│                  │ ┌────────────────────────────────────────────────┐ │ Persistence coming later │
│                  │ │ Session title                8 minutes ago     │ │                          │
│                  │ │ provider:model                                 │ │                          │
│                  │ ├────────────────────────────────────────────────┤ │                          │
│                  │ │ Another session              Yesterday         │ │                          │
│                  │ │ provider:model                                 │ │                          │
│                  │ └────────────────────────────────────────────────┘ │                          │
└──────────────────┴────────────────────────────────────────────────────┴──────────────────────────┘

After send or row selection, the main canvas becomes the existing Amp session shell; the app sidebar and right rail remain in place.
```

## Header and Navigation

- Show the project name in the content header.
- Provide quiet text links for **Notes** and **Review** in the header. These are secondary navigation, not cards or primary calls to action.
- Do not show a metric strip, destination-card grid, or a **New session** button anywhere on this surface.
- `#/projects/:id` is the canonical hub URL.
- **Preferred legacy-route behavior:** redirect `#/projects/:id/sessions` to `#/projects/:id` with replace semantics so browser history does not retain the obsolete route. If routing constraints require an alias, it must render the identical hub state rather than the old sessions destination.

## Start Area and Session Creation

The top of the main canvas presents a large heading, **“How can I help you today?”**, followed by a prominent multi-line composer and **Send** action.

Sending a valid prompt performs one user action with this product outcome:

1. Create a session using existing defaults.
2. Submit the composer text as that session's first message.
3. Open the new session's Amp session shell in the main canvas.

The UI must not expose a separate session-creation step. Disable duplicate submission while the create-and-send operation is in flight. Preserve the typed prompt and offer retry if creation or first-message submission fails; do not navigate to an empty or misleadingly successful session state.

## Session List

Session rows sit directly below the start area in the same main-canvas flow, following the Claude-style “Your chats” stack rather than a separate navigation column.

Each row shows:

- Session title as the primary label.
- `provider:model` as secondary metadata.
- Relative time when the API provides a timestamp; omit the time cleanly when it does not.

Clicking a row opens that session's existing Amp session shell in the main canvas. Rows are full-width interactive targets with visible hover and keyboard-focus states.

## Right Rail

The right rail is open by default and persists beside both the hub start state and an opened session. Its header contains two tabs: **Memory** and **Files**.

### Memory

Use Grok-2-style restrained form chrome with two clearly labeled fields:

- **Memory**
- **Instructions (system)**

This phase designs the fields and states only. Persistence is deferred until an API exists. The interface must not present an enabled save action, success toast, saved indicator, or any other claim that edits were persisted. If fields are editable for design evaluation, label them explicitly as unsaved/non-persistent; read-only or disabled presentation is also acceptable if clearer.

### Files

Show an Amp-style hierarchical directory tree with disclosure controls and nested indentation:

- Populate notes/source entries through existing APIs.
- When an active session exposes `workspace_files`, include its workspace hierarchy.
- On the hub before a session is selected, show only project-level files available from existing APIs.
- Switching or opening a session refreshes the workspace portion of the tree without replacing the entire page shell.

## Behavior Table

| Trigger / state | Main canvas | Right rail | URL / navigation |
|---|---|---|---|
| Open project hub | Start heading, composer, then session rows | Open by default; last selected tab may be retained, otherwise Memory | `#/projects/:id` |
| Submit non-empty prompt | Disable repeated send; create session and send first message | Remains mounted and open | Open created session in the hub's main canvas |
| Submit empty/whitespace prompt | No request; keep focus in composer and show concise validation | Unchanged | Unchanged |
| Click session row | Replace start/list content with existing Amp session shell | Remains open; Files may gain `workspace_files` | Open selected session in hub-equivalent state |
| Click Notes or Review | Navigate to the existing project destination | Follow destination shell behavior | Existing Notes or Review route |
| Select Memory tab | Show Memory and Instructions (system) fields | Memory tab active | No route change required |
| Select Files tab | Main canvas unchanged | Show hierarchical project/workspace tree | No route change required |
| Visit legacy sessions URL | Render no legacy session-list destination | Same as hub default | Prefer replace-redirect to `#/projects/:id` |

## Empty States

### No sessions yet

Always retain the full start area. Below it, show the session-section heading and quiet copy such as:

> No sessions yet. Send a message above to start one.

Do not replace the composer with a destination card, oversized illustration, or separate creation CTA.

### No files

In the Files tab, show concise contextual copy:

> No project files available.

If project files exist but no session is active, do not imply workspace files are missing; simply omit the workspace group. If an active session has an explicitly empty `workspace_files` result, its workspace group may state **No workspace files**.

### No stored memory data

Keep both labeled Memory fields visible with neutral empty values or placeholders. Do not imply that persistence is available.

## Loading and Error States

- **Initial hub loading:** keep the stable global shell and two-container geometry; use compact skeletons for the project heading and session rows. The composer may render once project identity/defaults are known.
- **Session-list loading:** show row-shaped skeletons beneath the fully visible start area. Do not block composing unnecessarily.
- **Session-list error:** keep the composer usable and show an inline retry message in the list region. Failure to load history must not look like an empty project.
- **Create/send error:** retain the draft, restore Send, focus the relevant error/composer area, and offer retry. Never show a successful session transition unless both required operations have reached a valid recoverable session state.
- **Opening-session loading:** show loading feedback in the main canvas without unmounting the app sidebar or right rail.
- **Opening-session error:** remain in or return to the hub start/list state with an inline error; do not strand the user in a blank session shell.
- **Memory:** because persistence is out of scope, do not simulate loading, saving, or save success. Surface only real read errors if memory data is fetched.
- **Files:** show tree-shaped loading placeholders. A fetch failure gets an inline retry within the Files tab and must not disrupt the main canvas.
- **Missing timestamps:** omit relative time rather than substituting fabricated recency.

## Out of Scope for the Hub

- Memory and system-instruction persistence APIs or fake save behavior.
- New session defaults, provider/model selection, or a separate session-creation flow.
- A left session-list column or any **New session** button.
- Metric strips, analytics summaries, and destination-card grids.
- Redesigning the internal Amp session shell or changing message execution behavior.
- Creating new Notes, source, or workspace file APIs; the Files tree consumes existing data only.
- Full Notes or Review page redesigns beyond their quiet header links.
- Inventing timestamps or other session metadata absent from the API.
