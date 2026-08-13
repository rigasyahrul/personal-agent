import{get}from'../api.js';
export async function render(root){const home=await get('/api/v1/home');root.innerHTML=home.projects.length?'<h2>Projects</h2>':`<h2>Your learning home</h2><p class="muted">No projects yet. Create your first project to begin collecting notes and sessions.</p><button disabled>Create project (next phase)</button>`}
