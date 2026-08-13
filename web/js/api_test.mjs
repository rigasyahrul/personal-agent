import assert from 'node:assert/strict'
import {APIError, get, mutate} from './api.js'

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
