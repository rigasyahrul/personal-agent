// web/src/lib/router.test.ts
import { describe, expect, it } from 'vitest';
import { parseRoute, routeToHash, type AppRoute } from './router';

const cases: Array<[string, AppRoute]> = [
  ['#/home', { name: 'home' }],
  ['#/projects', { name: 'projects' }],
  ['#/vaults', { name: 'vaults' }],
  ['#/vaults/health', { name: 'vault-home', vaultId: 'health' }],
  ['#/vaults/health/projects', { name: 'vault-projects', vaultId: 'health' }],
  ['#/vaults/health/sessions', { name: 'vault-sessions', vaultId: 'health' }],
  ['#/vaults/health/review', { name: 'vault-review', vaultId: 'health' }],
  ['#/projects/p1', { name: 'project', projectId: 'p1' }],
  ['#/projects/p1/notes', { name: 'notes', projectId: 'p1' }],
  ['#/projects/p1/notes/n1', { name: 'note', projectId: 'p1', noteId: 'n1' }],
  ['#/projects/p1/sessions', { name: 'sessions', projectId: 'p1' }],
  ['#/projects/p1/review', { name: 'project-review', projectId: 'p1' }],
  ['#/review', { name: 'review', scope: null }],
  ['#/review?scope=all', { name: 'review', scope: 'all' }],
  ['#/settings', { name: 'settings' }],
];

describe('hash router', () => {
  it.each(cases)('parses and serializes %s', (hash, route) => {
    expect(parseRoute(hash)).toEqual(route);
    expect(routeToHash(route)).toBe(hash);
  });

  it('decodes path and query values and encodes them on output', () => {
    const route: AppRoute = { name: 'note', projectId: 'project one', noteId: 'a/b' };
    expect(parseRoute(routeToHash(route))).toEqual(route);
    expect(parseRoute('#/review?scope=due%20today')).toEqual({ name: 'review', scope: 'due today' });
  });

  it.each(['', '#', '#/', '#/unknown', '#/vaults/v/unknown', '#settings'])(
    'falls back or supports a legacy hash: %s',
    (hash) => {
      expect(parseRoute(hash)).toEqual(hash === '#settings' ? { name: 'settings' } : { name: 'home' });
    },
  );
});
