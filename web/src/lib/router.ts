// web/src/lib/router.ts
export type AppRoute =
  | { name: 'home' }
  | { name: 'projects' }
  | { name: 'vaults' }
  | { name: 'vault-home'; vaultId: string }
  | { name: 'vault-projects'; vaultId: string }
  | { name: 'vault-sessions'; vaultId: string }
  | { name: 'vault-review'; vaultId: string }
  | { name: 'project'; projectId: string }
  | { name: 'notes'; projectId: string }
  | { name: 'note'; projectId: string; noteId: string }
  | { name: 'sessions'; projectId: string }
  | { name: 'project-review'; projectId: string }
  | { name: 'review'; scope: string | null }
  | { name: 'settings' };

const part = (value: string) => decodeURIComponent(value);
const encoded = (value: string) => encodeURIComponent(value);

export function parseRoute(hash: string): AppRoute {
  if (hash === '#settings') return { name: 'settings' };
  const raw = hash.startsWith('#') ? hash.slice(1) : hash;
  const [pathname, query = ''] = raw.split('?', 2);
  const segments = pathname.split('/').filter(Boolean).map(part);

  if (segments[0] === 'home' && segments.length === 1) return { name: 'home' };
  if (segments[0] === 'projects' && segments.length === 1) return { name: 'projects' };
  if (segments[0] === 'vaults' && segments.length === 1) return { name: 'vaults' };
  if (segments[0] === 'settings' && segments.length === 1) return { name: 'settings' };
  if (segments[0] === 'review' && segments.length === 1) {
    return { name: 'review', scope: new URLSearchParams(query).get('scope') };
  }
  if (segments[0] === 'vaults' && segments[1] && segments.length === 2) {
    return { name: 'vault-home', vaultId: segments[1] };
  }
  if (segments[0] === 'vaults' && segments[1] && segments.length === 3) {
    const name = { projects: 'vault-projects', sessions: 'vault-sessions', review: 'vault-review' }[
      segments[2]
    ] as 'vault-projects' | 'vault-sessions' | 'vault-review' | undefined;
    if (name) return { name, vaultId: segments[1] };
  }
  if (segments[0] === 'projects' && segments[1]) {
    const projectId = segments[1];
    if (segments.length === 2) return { name: 'project', projectId };
    if (segments[2] === 'notes' && segments.length === 3) return { name: 'notes', projectId };
    if (segments[2] === 'notes' && segments[3] && segments.length === 4) {
      return { name: 'note', projectId, noteId: segments[3] };
    }
    if (segments[2] === 'sessions' && segments.length === 3) return { name: 'sessions', projectId };
    if (segments[2] === 'review' && segments.length === 3) return { name: 'project-review', projectId };
  }
  return { name: 'home' };
}

export function routeToHash(route: AppRoute): string {
  switch (route.name) {
    case 'home': return '#/home';
    case 'projects': return '#/projects';
    case 'vaults': return '#/vaults';
    case 'vault-home': return `#/vaults/${encoded(route.vaultId)}`;
    case 'vault-projects': return `#/vaults/${encoded(route.vaultId)}/projects`;
    case 'vault-sessions': return `#/vaults/${encoded(route.vaultId)}/sessions`;
    case 'vault-review': return `#/vaults/${encoded(route.vaultId)}/review`;
    case 'project': return `#/projects/${encoded(route.projectId)}`;
    case 'notes': return `#/projects/${encoded(route.projectId)}/notes`;
    case 'note': return `#/projects/${encoded(route.projectId)}/notes/${encoded(route.noteId)}`;
    case 'sessions': return `#/projects/${encoded(route.projectId)}/sessions`;
    case 'project-review': return `#/projects/${encoded(route.projectId)}/review`;
    case 'review': return route.scope === null ? '#/review' : `#/review?scope=${encoded(route.scope)}`;
    case 'settings': return '#/settings';
  }
}
