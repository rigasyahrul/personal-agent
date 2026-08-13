const copies=['Promoting…','Promote failed — Retry','Note saved; cards pending…','Cards failed — Retry cards','Ready']
export function operationBadge(operation,onRetryCards){
  const el=document.createElement('div');el.className=`status-badge status-${operation.publication_status||''}`;el.setAttribute('role','status')
  el.textContent=copies.includes(operation.badge)?operation.badge:'Ready'
  if(operation.retry_cards===true&&operation.pending_id&&typeof onRetryCards==='function'){
    const button=document.createElement('button');button.type='button';button.textContent='Retry cards';button.addEventListener('click',()=>onRetryCards(operation,button));el.append(' ',button)
  }
  return el
}
