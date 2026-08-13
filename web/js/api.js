export class APIError extends Error{constructor(status,message){super(message);this.status=status}}
function cookie(name){return document.cookie.split('; ').find(value=>value.startsWith(`${name}=`))?.slice(name.length+1)}
export async function request(path,options={}){const headers={Accept:'application/json',...(options.headers||{})};const method=options.method||'GET';if(options.body!==undefined){headers['Content-Type']='application/json'}if(!['GET','HEAD'].includes(method)){const csrf=cookie('pa_csrf');if(csrf)headers['X-CSRF-Token']=decodeURIComponent(csrf)}const response=await fetch(path,{...options,method,headers});if(!response.ok)throw new APIError(response.status,(await response.text()).trim()||`Request failed (${response.status})`);if(response.status===204)return null;return response.json()}
export function get(path){return request(path)}
export function mutate(path,method,body){return request(path,{method,body:JSON.stringify(body)})}
