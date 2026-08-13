import assert from 'node:assert/strict'
import {APIError, api, get, mutate, workspaceFile, workspaceTree} from './api.js'

globalThis.document = {cookie: ''}
globalThis.fetch = async () => new Response(null, {status: 201})

const result = await mutate('/api/v1/setup/bootstrap', 'POST', {
  token: 'bootstrap-token',
  password: 'long-enough-password',
})

assert.equal(result, null, 'an empty successful response should return null')
console.log('PASS api request accepts an empty successful 201 response')

globalThis.fetch = async () => new Response(null, {status: 204})
assert.equal(await mutate('/api/v1/auth/logout', 'POST'), null)
console.log('PASS api request accepts a bodyless 204 response')

globalThis.fetch = async () => new Response('{"ready":true}', {status: 200})
assert.deepEqual(await get('/health'), {ready: true})
console.log('PASS api request parses a successful JSON response')

globalThis.fetch = async () => new Response('bad token', {status: 403})
await assert.rejects(get('/api/v1/auth/me'), error => {
  assert.ok(error instanceof APIError)
  assert.equal(error.status, 403)
  assert.equal(error.message, 'bad token')
  return true
})
console.log('PASS api request preserves failure status and body')

globalThis.fetch = async () => new Response('{"error":"provider_unavailable","detail":"secret upstream response"}', {status: 502})
await assert.rejects(get('/api/v1/sessions/s/messages'), error => {
  assert.equal(error.message, 'provider unavailable')
  assert.doesNotMatch(error.message, /secret/)
  return true
})
console.log('PASS api request presents safe JSON error codes')

let requested
globalThis.fetch = async (path, options) => {
  requested = {path, options}
  return new Response('{"id":"p1"}', {status: 201})
}
assert.deepEqual(await api('/projects', {method: 'POST', body: {name: 'Go'}}), {id: 'p1'})
assert.equal(requested.path, '/api/v1/projects')
assert.equal(requested.options.body, '{"name":"Go"}')
console.log('PASS api prefixes project paths and serializes object bodies')

const paths = []
globalThis.fetch = async path => {
  paths.push(path)
  return new Response('{"entries":[]}', {status: 200})
}
await workspaceTree('session /?#%')
globalThis.fetch = async path => {
  paths.push(path)
  return new Response('{"content":""}', {status: 200})
}
await workspaceFile('session /?#%', 'folder/<note> &?#%.txt')
assert.deepEqual(paths, [
  '/api/v1/sessions/session%20%2F%3F%23%25/workspace/tree',
  '/api/v1/sessions/session%20%2F%3F%23%25/workspace/file?path=folder%2F%3Cnote%3E%20%26%3F%23%25.txt',
])
console.log('PASS workspace API paths encode session IDs and logical file paths')
