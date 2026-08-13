export function parseRoute(hash=''){
  if(hash==='#settings'||hash==='#/settings')return{name:'settings'}
  const parts=hash.replace(/^#\/?/,'').split('/').filter(Boolean)
  if(parts[0]==='projects'&&parts[1]&&parts[2]==='notes'&&parts[3])return{name:'note',projectID:decodeURIComponent(parts[1]),noteID:decodeURIComponent(parts[3])}
  if(parts[0]==='projects'&&parts[1]&&parts[2]==='notes')return{name:'notes',projectID:decodeURIComponent(parts[1])}
  if(parts[0]==='projects'&&parts[1])return{name:'project',projectID:decodeURIComponent(parts[1])}
  return{name:'home'}
}
export function route(){return parseRoute(globalThis.location?.hash)}
