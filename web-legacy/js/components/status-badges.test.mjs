import test from 'node:test'
import assert from 'node:assert/strict'
import {operationBadge} from './status-badges.js'
import {TestDocument,findText} from '../test-dom.mjs'

globalThis.document=new TestDocument()

test('operationBadge renders only the exact five safe copies',()=>{
  for(const copy of ['Promoting…','Promote failed — Retry','Note saved; cards pending…','Cards failed — Retry cards','Ready'])assert.equal(operationBadge({badge:copy}).textContent,copy)
  assert.equal(operationBadge({badge:'<img onerror=bad>'}).textContent,'Ready')
})

test('operationBadge gates retry, supports initial disabled, and consumes rejection',async()=>{
  assert.equal(operationBadge({badge:'Ready',retry_cards:false,pending_id:'p'},()=>{}).querySelector('button'),null)
  const unhandled=[];const listener=reason=>unhandled.push(reason);process.on('unhandledRejection',listener)
  try{
    let calls=0
    const badge=operationBadge({badge:'Cards failed — Retry cards',retry_cards:true,pending_id:'p'},async()=>{calls++;throw new Error('handled')},{retryDisabled:true})
    const button=findText(badge,'Retry cards');assert.equal(button.disabled,true)
    button.disabled=false;button.click();await new Promise(resolve=>setImmediate(resolve))
    assert.equal(calls,1);assert.deepEqual(unhandled,[])
  }finally{process.off('unhandledRejection',listener)}
})
