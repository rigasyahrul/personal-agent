import test from 'node:test'
import assert from 'node:assert/strict'
import {renderReview} from './review.js'
import {TestDocument,TestElement,findText} from '../test-dom.mjs'

globalThis.document=new TestDocument();globalThis.location={hash:''}
const queue=(items,caught_up=false)=>({reviewQueue:async()=>({items,caught_up})})

test('renderReview renders explicit scopes, caught-up, bite reveal, and whole link',async()=>{
  const root=new TestElement('main')
  await renderReview(root,{projectId:'p',scope:'project:p',api:queue([],true)})
  assert.equal(findText(root,'This project').disabled,true);assert.ok(findText(root,'Caught up in this project.'))
  await renderReview(root,{projectId:'p',scope:'project:p',api:queue([{id:'b',kind:'bite',prompt:'Q',answer:'A',row_version:1},{id:'w',kind:'whole',prompt:'W',project_id:'p',note_id:'n'}])})
  const reveal=findText(root,'Reveal answer'),answer=findText(root,'A');assert.equal(answer.hidden,true);reveal.click();assert.equal(answer.hidden,false);assert.equal(reveal.hidden,true)
  assert.equal(findText(root,'Open current note').href,'#/projects/p/notes/n')
  findText(root,'All projects').click();assert.equal(location.hash,'#/projects/p/review?scope=all')
})

test('renderReview production actions send exact rate duration and suspend, and show errors',async()=>{
  const root=new TestElement('main'),calls=[];let time=100
  const api={reviewQueue:async()=>({items:[{id:'i',kind:'bite',prompt:'Q',answer:'A',row_version:7}]}),rateReviewItem:async(...args)=>{calls.push(['rate',...args]);throw new Error('rate failed')},suspendReviewItem:async(...args)=>calls.push(['suspend',...args])}
  await renderReview(root,{scope:'all',api,now:()=>time,uuid:()=> 'key'})
  time=142;await findText(root,'good').click();assert.deepEqual(calls[0],['rate','i',{rating:'good',request_key:'key',row_version:7,duration_ms:42}]);assert.equal(findText(root,'rate failed').getAttribute('role'),'alert')
  await findText(root,'Suspend').click();assert.deepEqual(calls[1],['suspend','i'])
})

test('stale overlapping queue and action results cannot overwrite newer render',async()=>{
  let resolveOld;const old=new Promise(resolve=>resolveOld=resolve),root=new TestElement('main')
  const first=renderReview(root,{scope:'all',api:{reviewQueue:()=>old}})
  await renderReview(root,{scope:'all',api:queue([],true)})
  resolveOld({items:[{id:'old',kind:'bite',prompt:'OLD',answer:'A'}]});await first
  assert.equal(findText(root,'OLD'),undefined);assert.ok(findText(root,'Caught up in all projects.'))
})

test('all-project review never emits project undefined',async()=>{
  const root=new TestElement('main');await renderReview(root,{scope:'all',api:queue([],true)})
  assert.equal([...root.walk()].some(node=>Object.hasOwn(node,'href')&&String(node.href).includes('undefined')),false)
})
