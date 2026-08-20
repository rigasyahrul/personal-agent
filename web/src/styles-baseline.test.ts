// web/src/styles-baseline.test.ts
import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const here = dirname(fileURLToPath(import.meta.url));
const css = readFileSync(join(here, 'app.css'), 'utf8');
const html = readFileSync(join(here, 'app.html'), 'utf8');

function walkSvelte(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) walkSvelte(path, out);
    else if (entry.name.endsWith('.svelte')) out.push(path);
  }
  return out;
}

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
      '.btn--primary',
      '.entity-card',
      '.metric-card',
    ]) {
      expect(css).toContain(token);
    }
  });

  it('bans scaffold soup leftovers in Svelte markup', () => {
    const files = walkSvelte(here);
    const offenders: string[] = [];
    for (const file of files) {
      const src = readFileSync(file, 'utf8');
      if (src.includes('bg-indigo-600')) offenders.push(`${file}: bg-indigo-600`);
      if (src.includes('Global desk')) offenders.push(`${file}: Global desk`);
      if (src.includes('Storage status unavailable')) {
        offenders.push(`${file}: Storage status unavailable`);
      }
      // Bullet-as-nav-icon (literal • glyph in markup)
      if (/>\s*•\s*</.test(src) || src.includes('>•<')) {
        offenders.push(`${file}: bullet nav glyph`);
      }
    }
    expect(offenders).toEqual([]);
  });
});
