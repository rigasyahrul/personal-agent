# 01 — Overview

## What Personal Agent is

A **browser UI + single Go API** you run yourself. One owner account. All durable state lives under one data directory (`PA_DATA_DIR`, default `./data`): SQLite, project source files, session workspaces, and local backup bundles.

Optional **OpenAI-compatible** HTTP is used for chat and bite generation. Optional **S3-compatible** storage is for **backup upload only** — the app works fully without a bucket.

## Mental model

```text
Projects
  └── Source notes (.md library you own)
  └── Sessions (chat + private workspace)
        └── Promote (“Save to source”) → source note + optional review
  └── Review (due cards: whole notes or bites)
Backup (snapshot of DB + files)
```

| Concept | Meaning |
|---------|---------|
| **Project** | A learning container: notes, sessions, scoped review |
| **Source note** | Durable Markdown under the project source tree |
| **Session** | One model, optional workspace file tools, chat history |
| **Promote** | Copy a workspace `.md` into project source (no clobber) |
| **Direct create** | Write a source `.md` from the Notes UI without a session |
| **Review** | Spaced repetition (`sm2-lite-v1`): Again / Hard / Good / Easy |
| **Backup** | Sealed directory bundle of DB + files (+ optional S3) |

## What you will do most days

1. Open a **project**.  
2. Either **chat** in a session (and promote good drafts) or **publish** a Markdown file under Notes.  
3. Clear **due review** (Home → Review, or project → Review).  
4. Occasionally **Backup now** (or leave Daily schedule on).

## Non-goals (v1)

Do not expect:

- Multi-user or multi-tenant auth  
- Arbitrary shell / host filesystem for the agent  
- Full-text search or a vault browser  
- Safe public HTTP without HTTPS  

## Navigation in the app

Top nav (after login):

- **Home** — project cards, new project, global review  
- **Review** — due items (`scope=all`)  
- **Settings** — timezone display, backup schedule, Backup now  

Header also shows **storage ready / unavailable** from `/health`.

Hash routes (examples):

| Route | Screen |
|-------|--------|
| `#/home` | Projects |
| `#/projects/{id}` | Project overview |
| `#/projects/{id}/notes` | Source tree + create |
| `#/projects/{id}/sessions` | Sessions / chat |
| `#/projects/{id}/review?scope=project:{id}` | Project review |
| `#/review?scope=all` | All-projects review |
| `#settings` | Settings |

## Next

→ [02 — Install and first run](02-install-and-first-run.md)
