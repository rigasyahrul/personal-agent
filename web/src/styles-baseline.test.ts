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
      '.wikilink',
      '.catalog-grid',
      '.alert--error',
      '.btn--primary',
      '.entity-card',
      '.metric-card',
      '.modal', // benchmark B1 shared dialog primitive
      '.name-list',
      '.name-row',
      '.name-row__title',
      '.name-row__meta',
      '.compound-card',
      '.compound-item',
      '.backlinks',
      '.backlinks__item',
      '.knowledge-search',
      '.knowledge-search__hit',
      '.instruction-editor',
      '.instruction-editor__tab',
    ]) {
      expect(css).toContain(token);
    }
  });

  it('declares session-focus layout tokens', () => {
    for (const token of [
      '.session-focus',
      '.session-tabs',
      '.session-tab',
      '.session-split',
      '.session-files',
      '.session-card',
      '.message-prose',
      '.session-composer',
      '.session-compound',
      '.message-copy',
      '.content-canvas--session-focus',
    ]) {
      expect(css).toContain(token);
    }
  });

  it('declares hub/rail workspace tokens', () => {
    for (const token of [
      '.project-workspace',
      '.project-workspace__main',
      '.project-workspace__rail',
      '.rail-tabs',
      '.rail-tab',
      '.rail-tab--active',
      '.rail-panel',
      '.rail-memory-preview',
      '.hub-start',
      '.hub-start__title',
      '.hub-composer',
      '.hub-session-list',
      '.content-canvas--project-workspace',
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

  it('packs sidebar nav rows without stretch (benchmark Phase A density)', () => {
    // Expanded sidebar chrome: 220–240px width, ~12×10 padding
    expect(css).toMatch(/\.sidebar\s*\{[^}]*width:\s*(220|240|230|225|235)px/s);
    expect(css).toMatch(/\.sidebar\s*\{[^}]*padding:\s*12px\s+10px/s);

    // Nav grows to push collapse down, but rows pack to start (no stretch fill)
    const navBlock = css.match(/\.sidebar nav\s*\{[^}]*\}/);
    expect(navBlock?.[0]).toBeTruthy();
    expect(navBlock![0]).toMatch(/flex:\s*1/);
    expect(navBlock![0]).toMatch(/display:\s*grid/);
    expect(navBlock![0]).toMatch(/align-content:\s*start/);
    // Default grid stretch is the bug; explicit stretch must not reappear on nav
    expect(navBlock![0]).not.toMatch(/align-content:\s*stretch/);

    // Row min-height stays compact (36–40px)
    expect(css).toMatch(
      /\.sidebar nav a,\s*\n\s*\.sidebar__disabled\s*\{[^}]*min-height:\s*(36|37|38|39|40)px/s,
    );
  });
});
