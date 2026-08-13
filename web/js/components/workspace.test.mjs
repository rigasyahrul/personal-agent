import test from 'node:test'
import assert from 'node:assert/strict'
import {changedPaths, workspaceRows} from './workspace.mjs'

test('workspaceRows escapes labels and marks changed files', () => {
  const html = workspaceRows(
    [{path: 'drafts', kind: 'directory'}, {path: 'drafts/<note>.txt', kind: 'file', size: 5}],
    new Set(['drafts/<note>.txt']),
  )
  assert.match(html, /drafts\/&lt;note&gt;\.txt/)
  assert.match(html, /data-path="drafts\/&lt;note&gt;\.txt"/)
  assert.match(html, /workspace-entry--changed/)
  assert.doesNotMatch(html, /<note>/)
})

test('workspaceRows allowlists kinds used in CSS classes', () => {
  const html = workspaceRows([{path: 'safe.txt', kind: 'file bad" onclick="alert(1)'}], new Set())
  assert.doesNotMatch(html, /onclick|workspace-entry--file bad/)
  assert.match(html, /workspace-entry--file/)
})

test('changedPaths returns only agent tool mutations', () => {
  const messages = [
    {role: 'user', changed_path: 'ignored.txt'},
    {role: 'tool', changed_path: 'draft.md'},
    {role: 'tool', content: '{"changed_path":"notes/raw.txt"}'},
    {role: 'assistant', content: 'done'},
  ]
  assert.deepEqual([...changedPaths(messages)], ['draft.md', 'notes/raw.txt'])
})
