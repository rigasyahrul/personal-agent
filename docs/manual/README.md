# Personal Agent — Owner handbook

**Edition:** 2026-08-20 (Svelte UI + first-class vaults)  
**Audience:** The single owner who installs, runs, and uses the app day to day.

This handbook matches the **shipped** product on `main`: Svelte 5 SPA (`web/` → `web/dist`), hash routes, global vs vault sidebar, unfiled projects, and sessions always per-project.

## Read in order

| # | Chapter | Contents |
|---|---------|----------|
| 1 | [Overview](01-overview.md) | Mental model, vaults, sidebar, routes |
| 2 | [Install and first run](02-install-and-first-run.md) | Binary, Docker, docker-dev, bootstrap |
| 3 | [Daily use](03-daily-use.md) | Home, projects, vaults, notes, chat, promote, review |
| 4 | [Settings and backup](04-settings-and-backup.md) | Schedule, Backup now, S3, restore habits |
| 5 | [Troubleshooting](05-troubleshooting.md) | Common failures and fixes |
| 6 | [Reference](06-reference.md) | Routes, env, make targets, invariants |

## Related engineering docs

| Doc | When to open it |
|-----|-----------------|
| [Root README](../../README.md) | Clone, build, test, docker-dev one-liners |
| [Deploy / ops](../ops/deploy.md) | Production Compose, TLS, volumes, HMR details |
| [Superpowers index](../superpowers/README.md) | Specs, plans, execution boards (history) |

## What changed in this edition

- Replaced top-nav “Home / Review / Settings only” docs with **sidebar + vault context**
- Documented **unfiled vs vaulted** projects and immutable `vault_id` at create
- Documented **hash routes** and production path **`web/dist`**
- Documented **Node 22**, `make web-build` / `make web-test`, and `make docker-dev` HMR
- Removed references to the deleted legacy static UI

If something in this handbook disagrees with a frozen design note under `docs/superpowers/`, **trust the running app and this handbook**.
