import{get,request}from'../api.js';

export async function render(root,isCurrent=()=>true){
  const settings=await get('/api/v1/settings');
  if(!isCurrent())return;
  root.innerHTML=renderSettings(settings);
  wire(root,isCurrent);
}

function renderSettings(settings){
  const backup=settings.backup||{};
  const lastSuccess=backup.last_success||settings.last_success||null;
  const lastFailure=backup.last_failure||settings.last_failure||null;
  const schedule=settings.backup_schedule||backup.schedule||'off';
  const sink=backup.sink_configured===true;
  let statusHTML='Never backed up';
  if(lastSuccess&&lastSuccess.completed_at){
    statusHTML=`Last successful backup: ${escapeHTML(lastSuccess.completed_at)}`;
  }
  let failHTML='';
  if(lastFailure&&lastFailure.completed_at){
    const failNewer=!lastSuccess||String(lastFailure.completed_at)>String(lastSuccess.completed_at);
    if(failNewer){
      failHTML=`<p class="error">Last attempt failed: ${escapeHTML(lastFailure.error||'unknown error')}</p>`;
    }
  }
  return `<h2>Settings</h2>
<section class="settings-main">
  <dl>
    <dt>Timezone</dt><dd>${escapeHTML(settings.timezone)}</dd>
    <dt>Default provider</dt><dd>${escapeHTML(settings.default_provider||'Not set')}</dd>
    <dt>Default model</dt><dd>${escapeHTML(settings.default_model_id||'Not set')}</dd>
  </dl>
</section>
<section class="settings-backup">
  <h3>Backup</h3>
  <p>${statusHTML}</p>
  ${failHTML}
  <p class="muted">Remote sink configured: ${sink?'yes':'no'}</p>
  <label>Schedule
    <select id="backup-schedule">
      <option value="off"${schedule==='off'?' selected':''}>Off</option>
      <option value="daily"${schedule==='daily'?' selected':''}>Daily</option>
    </select>
  </label>
  <button type="button" id="backup-now">Backup now</button>
  <p id="backup-msg" class="muted" aria-live="polite"></p>
</section>`;
}

function wire(root,isCurrent){
  const scheduleEl=root.querySelector('#backup-schedule');
  const btn=root.querySelector('#backup-now');
  const msg=root.querySelector('#backup-msg');
  scheduleEl?.addEventListener('change',async()=>{
    msg.textContent='Saving schedule…';
    try{
      const current=await get('/api/v1/settings');
      await request('/api/v1/settings',{method:'PUT',body:{
        timezone:current.timezone,
        default_provider:current.default_provider||'',
        default_model_id:current.default_model_id||'',
        backup_schedule:scheduleEl.value
      }});
      msg.textContent='Schedule saved.';
    }catch(err){
      msg.textContent=err.message||'Failed to save schedule';
    }
  });
  btn?.addEventListener('click',async()=>{
    btn.disabled=true;
    msg.textContent='Running backup…';
    try{
      await request('/api/v1/backups',{method:'POST',body:{}});
      const settings=await get('/api/v1/settings');
      if(!isCurrent())return;
      root.innerHTML=renderSettings(settings);
      wire(root,isCurrent);
      root.querySelector('#backup-msg').textContent='Backup completed.';
    }catch(err){
      msg.textContent=err.message||'Backup failed';
      btn.disabled=false;
    }
  });
}

function escapeHTML(value){
  const span=document.createElement('span');
  span.textContent=value==null?'':String(value);
  return span.innerHTML;
}
