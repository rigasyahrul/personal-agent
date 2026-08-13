export function parseRoute(hash=''){
  if(hash==='#settings'||hash==='#/settings')return{name:'settings'}
  const parts=hash.replace(/^#\/?/,'').split('/').filter(Boolean)
  try{
    if(parts.length===4&&parts[0]==='projects'&&parts[1]&&parts[2]==='notes'&&parts[3])return{name:'note',projectID:decodeURIComponent(parts[1]),noteID:decodeURIComponent(parts[3])}
    if(parts.length===3&&parts[0]==='projects'&&parts[1]&&parts[2]==='notes')return{name:'notes',projectID:decodeURIComponent(parts[1])}
    if(parts.length===3&&parts[0]==='projects'&&parts[1]&&parts[2]==='sessions')return{name:'sessions',projectID:decodeURIComponent(parts[1])}
    if(parts.length===2&&parts[0]==='projects'&&parts[1])return{name:'project',projectID:decodeURIComponent(parts[1])}
  }catch{}
  return{name:'home'}
}
export function route(){return parseRoute(globalThis.location?.hash)}
export function navigateIfCurrent(isCurrent,destination,navigate=value=>{globalThis.location.hash=value}){
  if(!isCurrent())return false
  navigate(destination)
  return true
}
