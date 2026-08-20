// web/src/routes/ProjectSessionsPage.test.ts
// Legacy ProjectSessionsPage is unused: App routes `sessions` → ProjectHubPage.
// Keep this file as a pointer so the suite stays green; hub behavior lives in
// ProjectHubPage.test.ts and App.test.ts (legacy sessions hash).
import { describe, expect, it } from 'vitest'
import { parseRoute } from '../lib/router'

describe('ProjectSessionsPage (legacy)', () => {
  it('still parses #/projects/:id/sessions as the sessions route name', () => {
    expect(parseRoute('#/projects/p1/sessions')).toEqual({
      name: 'sessions',
      projectId: 'p1',
    })
  })
})
