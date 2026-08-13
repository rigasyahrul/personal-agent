import test from 'node:test'
import assert from 'node:assert/strict'
import {projectCard} from './home.js'
import {projectOverview} from './project.js'
import {bodyWithinByteLimit, directPayload, nextPublicationKey, noteViewer, publicationDestination, treeRows} from './notes.js'
import {parseRoute} from '../router.js'

test('project card includes counts and vault badge', () => {
  const html = projectCard({id: 'p1', name: 'Go', vault_name: 'Learning', note_count: 2, session_count: 0, due_count: 1})
  assert.match(html, /Go/)
  assert.match(html, /Learning/)
  assert.match(html, /2 notes/)
  assert.match(html, /1 due/)
})

test('tree links notes by id, never by path', () => {
  const html = treeRows('p1', [{kind: 'note', path: 'guide/a.md', note_id: 'n1'}])
  assert.match(html, /notes\/n1/)
  assert.doesNotMatch(html, /notes\/guide/)
})

test('direct create preserves locked request fields', () => {
  assert.deepEqual(directPayload('guide/a.md', '# A', 'whole'), {relative_path: 'guide/a.md', body: '# A', review_mode: 'whole'})
})

test('router prioritizes note detail and decodes encoded segments', () => {
  assert.deepEqual(parseRoute('#/projects/a%2Fb/notes/n%201'), {name: 'note', projectID: 'a/b', noteID: 'n 1'})
  assert.deepEqual(parseRoute('#/projects/p1/notes'), {name: 'notes', projectID: 'p1'})
  assert.deepEqual(parseRoute('#/projects/p1'), {name: 'project', projectID: 'p1'})
  assert.deepEqual(parseRoute('#settings'), {name: 'settings'})
})

test('router falls back for malformed encoding and surplus segments', () => {
  assert.deepEqual(parseRoute('#/projects/%E0%A4%A'), {name: 'home'})
  assert.deepEqual(parseRoute('#/projects/p1/notes/n1/extra'), {name: 'home'})
  assert.deepEqual(parseRoute('#/projects/p1/extra'), {name: 'home'})
})

test('dynamic project, path, and note values are escaped', () => {
  const attack = `"><img src=x onerror='alert(1)'>`
  assert.doesNotMatch(projectCard({id: attack, name: attack, vault_name: attack}), /<img/)
  assert.doesNotMatch(projectOverview({id: attack, name: attack}), /<img/)
  assert.doesNotMatch(treeRows(attack, [{kind: 'folder', path: attack}]), /<img/)
  assert.doesNotMatch(noteViewer({relative_path: attack, body: attack}), /<img/)
  assert.match(noteViewer({relative_path: 'x.md', body: attack}), /&lt;img/)
})

test('publication retries reuse a key only for the exact payload', () => {
  const first = nextPublicationKey(null, directPayload('a.md', '# A', 'whole'), () => 'key-1')
  const retry = nextPublicationKey(first, directPayload('a.md', '# A', 'whole'), () => 'key-2')
  const changed = nextPublicationKey(retry, directPayload('a.md', '# B', 'whole'), () => 'key-3')
  assert.equal(retry.key, 'key-1')
  assert.equal(changed.key, 'key-3')
})

test('body byte validation accepts the boundary and rejects multibyte overflow', () => {
  assert.equal(bodyWithinByteLimit('a'.repeat(1048576)), true)
  assert.equal(bodyWithinByteLimit(`${'a'.repeat(1048575)}é`), false)
})

test('publication destination requires a note id', () => {
  assert.equal(publicationDestination('p1', {note_id: `n'\"><x>`}), '#/projects/p1/notes/n%27%22%3E%3Cx%3E')
  assert.equal(publicationDestination('p1', {status: 'completed'}), null)
})

test('render generation gate rejects stale work', async () => {
  const {renderGenerationIsCurrent} = await import('../app.js')
  let current = 1
  let rendered = ''
  const oldRender = Promise.resolve().then(() => {
    if (renderGenerationIsCurrent(1, current)) rendered = 'old'
  })
  current = 2
  if (renderGenerationIsCurrent(2, current)) rendered = 'new'
  await oldRender
  assert.equal(rendered, 'new')
})

test('direct note form constraints are present', async () => {
  const source = await import('node:fs/promises').then(fs => fs.readFile(new URL('./notes.js', import.meta.url), 'utf8'))
  assert.match(source, /maxlength="512"/)
  assert.match(source, /maxlength="1048576"/)
  assert.match(source, /accept="\.md"/)
})
