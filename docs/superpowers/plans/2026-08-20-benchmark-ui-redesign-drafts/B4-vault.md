# Phase B4 — Vault projects name-first list (draft)

> Task 10. Spec §6 (`claude-2.png`).

### Task 10: Vault projects name-first rows

**Files:**
- Modify: `web/src/routes/VaultProjectsPage.svelte`
- Modify: `web/src/routes/VaultProjectsPage.test.ts`
- Modify: `web/src/app.css` — `.name-row` list tokens
- Modify: `web/src/styles-baseline.test.ts` — assert `.name-row`
- Optional: Create `web/src/components/NameRow.svelte` if reuse helps; else inline button rows

**Replace:** `catalog-grid` + `ProjectCard` / `entity-card` fat cards  
**With:** vertical list of name-first rows:

```svelte
<ul class="name-list" role="list">
  {#each visible as project (project.id)}
    <li>
      <button type="button" class="name-row" onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)}>
        <span class="name-row__title">{project.name}</span>
        <span class="name-row__meta">{/* optional quiet counts */}</span>
        <span class="name-row__chevron" aria-hidden="true">→</span>
      </button>
    </li>
  {/each}
</ul>
```

**CSS:**
```css
.name-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; }
.name-row {
  display: flex; align-items: center; gap: 12px; width: 100%;
  min-height: 44px; padding: 10px 12px; text-align: left;
  border: 1px solid transparent; border-radius: var(--radius-sm);
  background: transparent; cursor: pointer;
}
.name-row:hover { background: #fafafa; border-color: var(--border); }
.name-row__title { font-weight: 600; flex: 1; min-width: 0; }
.name-row__meta { font-size: 12px; color: var(--muted); }
```

**Keep:** vault eyebrow + Projects h1 + New project → Modal (from Task 3). Empty → modal. Search may stay above list.

**Tests:**
```ts
it('renders name-first rows not entity-card grid', async () => {
  // mock projects
  expect(screen.getByRole('button', { name: /Project 1/i })).toBeInTheDocument()
  expect(document.querySelector('.entity-card')).toBeNull()
  expect(document.querySelector('.name-row')).toBeTruthy()
})
it('New project opens dialog', async () => {
  await fireEvent.click(screen.getByRole('button', { name: /new project/i }))
  expect(screen.getByRole('dialog')).toBeInTheDocument()
})
```

- [ ] **Step 1: Failing tests + baseline `.name-row`**
- [ ] **Step 2: Run**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts src/routes/VaultProjectsPage.test.ts
```

- [ ] **Step 3: Implement list + CSS**
- [ ] **Step 4: Pass**
- [ ] **Step 5: Commit** `feat(web): vault projects name-first list`

**Done B4:** vault list matches claude-2 hierarchy; create still modal.
