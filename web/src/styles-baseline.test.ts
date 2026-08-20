// web/src/styles-baseline.test.ts
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const here = dirname(fileURLToPath(import.meta.url));
const css = readFileSync(join(here, 'app.css'), 'utf8');
const html = readFileSync(join(here, 'app.html'), 'utf8');

describe('visual baseline', () => {
  it('loads Inter once and declares required theme surfaces', () => {
    expect(css).toContain("@import '@fontsource/inter/variable.css'");
    for (const token of ['--canvas', '--panel', '--sidebar', '--border', '--accent', '--danger']) {
      expect(css).toContain(token);
    }
    expect(html).toContain('<meta name="viewport" content="width=device-width, initial-scale=1" />');
  });

  it('has focus and mobile shell rules', () => {
    expect(css).toContain(':focus-visible');
    expect(css).toContain('@media (max-width: 767px)');
    expect(css).not.toMatch(/linear-gradient|radial-gradient|backdrop-filter/);
  });

  it('declares full-surface craft primitives', () => {
    for (const token of [
      '.panel',
      '.form-stack',
      '.field-input',
      '.scope-chip',
      '.list-panel',
      '.link-accent',
      '.catalog-grid',
      '.alert--error',
    ]) {
      expect(css).toContain(token);
    }
  });
});
