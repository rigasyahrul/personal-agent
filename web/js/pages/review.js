import{reviewQueue,rateReviewItem,suspendReviewItem}from'../api.js'
import{reviewHash,validReviewScope}from'../router.js'

const button=(text)=>{const el=document.createElement('button');el.type='button';el.textContent=text;return el}
const generations=new WeakMap()
export async function renderReview(root,{projectId,scope,isCurrent=()=>true,api={reviewQueue,rateReviewItem,suspendReviewItem},now=()=>performance.now(),uuid=()=>crypto.randomUUID()}={}){
  const generation=(generations.get(root)||0)+1;generations.set(root,generation)
  const current=()=>isCurrent()&&generations.get(root)===generation
  const fallback=projectId?`project:${projectId}`:'all'
  if(scope===null||scope===undefined){location.hash=reviewHash(fallback,projectId);return}
  if(!validReviewScope(scope,projectId))throw new Error('Invalid review scope')
  const data=await api.reviewQueue(scope);if(!current())return
  const page=document.createElement('section'),heading=document.createElement('div'),title=document.createElement('h2');title.textContent='Review';heading.className='page-heading';heading.append(title);page.append(heading)
  const scopes=document.createElement('nav');scopes.className='scope-chip';scopes.setAttribute('aria-label','Review scope')
  const choices=projectId?[[`project:${projectId}`,'This project'],['all','All projects']]:[['all','All projects']]
  for(const [value,label]of choices){const choice=button(label);choice.disabled=value===scope;choice.onclick=()=>{if(value!==scope)location.hash=reviewHash(value,projectId)};scopes.append(choice)}page.append(scopes)
  if(data.caught_up){const empty=document.createElement('p');empty.className='caught-up';empty.textContent=`Caught up in ${scope==='all'?'all projects':'this project'}.`;page.append(empty);root.replaceChildren(page);return}
  for(const item of data.items||[]){const started=now(),card=document.createElement('article');card.className='review-card';const prompt=document.createElement('h3');prompt.textContent=item.prompt;card.append(prompt)
    if(item.kind==='bite'){const answer=document.createElement('p');answer.className='answer';answer.hidden=true;answer.textContent=item.answer??'';const reveal=button('Reveal answer');reveal.onclick=()=>{answer.hidden=false;reveal.hidden=true};card.append(reveal,answer)}else{const link=document.createElement('a');link.className='button';link.textContent='Open current note';link.href=`#/projects/${encodeURIComponent(item.project_id)}/notes/${encodeURIComponent(item.note_id)}`;card.append(link)}
    const actions=document.createElement('div');actions.className='ratings';const error=document.createElement('p');error.className='error';error.setAttribute('role','alert')
    let actionPending=false;const act=async fn=>{if(actionPending||!current())return;actionPending=true;[...actions.children].forEach(x=>x.disabled=true);try{await fn();if(current()){const refreshGeneration=generation+1;try{await renderReview(root,{projectId,scope,isCurrent,api,now,uuid})}catch(reason){if(isCurrent()&&generations.get(root)===refreshGeneration&&page.parentNode===root){generations.set(root,generation);throw reason}}}}catch(reason){if(current()){error.textContent=reason.message;[...actions.children].forEach(x=>x.disabled=false);actionPending=false}}}
    for(const rating of['again','hard','good','easy']){const rate=button(rating);rate.onclick=()=>act(()=>api.rateReviewItem(item.id,{rating,request_key:uuid(),row_version:item.row_version,duration_ms:Math.max(0,Math.round(now()-started))}));actions.append(rate)}const suspend=button('Suspend');suspend.onclick=()=>act(()=>api.suspendReviewItem(item.id));actions.append(suspend);card.append(actions,error);page.append(card)
  }root.replaceChildren(page)
}
