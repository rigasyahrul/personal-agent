# Sessions chat focus + Docker live-reload

**Date:** 2026-08-19  
**Tags:** ui, spa, docker, make


**Task:** Fix message textarea losing focus ~500ms after click on sessions chat; make local Docker pick up API+web edits without rebuild thrash.

**Wrong / mistakes:**
1. Assumed “I edited `web/js/…` so localhost:8080 has the fix.” Port 8080 was **Docker** with a **baked** image (`COPY web /app/web`). Host file ≠ served file. First “fix” never reached the browser.
2. Assumed skipping unchanged poll re-renders was enough. Any poll that still called `root.innerHTML = …` (new messages, run status, send disabled) **destroyed** the focused textarea. User felt ~500ms loss = poll RTT, not the 1.5s timer.
3. Put a permanent `../web:/app/web` mount on **production** compose — wrong layer. Prod should stay image-baked; live mounts belong only in a **dev override**.
4. Told the user to pass two compose `-f` flags without a one-command path. Dev file is an override (ports/env/`pa-data` live in the base file) — document that, but expose **`make docker-dev`**.

**What worked:**
1. **Focus:** keep the composer node alive across polls — `patchChat()` updates `ol.messages`, `.run-status`, alert text, and Send `disabled` in place; full `renderChat()` only when the chat shell is missing or session switches. Restore focus/selection only if a full rebuild is unavoidable. Regression: `web/js/pages/sessions.test.js` (“polling does not steal message focus…”).
2. **Prove serve path before claiming UI fixed:** `curl -s http://127.0.0.1:8080/js/pages/sessions.js | wc -c` (or `rg patchChat`) vs host file; `docker exec … wc -c /app/web/…` or `/src/web/…`; `lsof -iTCP:8080 -sTCP:LISTEN`.
3. **Dev compose:** `deploy/docker-compose.dev.yml` overrides the same `personal-agent` service with `Dockerfile.dev` + `air` + `..:/src` + module/build caches. Base prod compose has **no** host source mounts (enforced in `deploy/deploy_test.go`).
4. **UX:** `make docker-dev` / `docker-dev-down` / `docker-dev-logs`; README + `docs/ops/deploy.md` explain override vs prod.

**Rule (next agent):**
- SPA pages that poll must **not** replace focused form controls via `innerHTML` on every tick. Prefer in-place DOM patches; gate full shell rebuilds; add a focus/selection regression test.
- Before claiming a frontend fix works against “localhost”, verify **which process owns the port** and that the **bytes served** include the change.
- Production Compose stays baked. Live API+web reload = **dev override** (`docker-compose.dev.yml` + `air`), not mounts on the prod file. Day-to-day command: **`make docker-dev`** (needs `deploy/.env`).
- New Makefile public targets: `##` help text, `.PHONY`, and the Development `print-help-section` list.

**Codified into:**
- `web/js/pages/sessions.js` (`patchChat` / `paintChat` / skip full rebuild)
- `web/js/pages/sessions.test.js` (focus + in-place update)
- `deploy/Dockerfile.dev`, `deploy/air.toml`, `deploy/docker-compose.dev.yml`
- `deploy/docker-compose.yml` (prod clean), `deploy/deploy_test.go` (prod forbids source mounts; dev requires `..:/src` + air)
- `Makefile` (`docker-dev*`), `README.md`, `docs/ops/deploy.md`
- `.gitignore` (`/tmp/` for air output)
- Standing bullets in `AGENTS.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a01972-ac51-727e-be42-738fa7156a3c

---
