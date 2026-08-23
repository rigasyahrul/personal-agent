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

  it('knowledge compound search instruction surfaces use theme tokens, not one-off hex', () => {
    expect(css).toMatch(/--fg\s*:/);
    const blocks =
      css.match(
        /\.(compound-[\w-]+|backlinks[\w-]*|knowledge-search[\w-]*|instruction-editor[\w-]*)\s*\{[^}]*\}/g,
      ) ?? [];
    expect(blocks.length).toBeGreaterThan(10);
    const hex = /#[0-9a-fA-F]{3,8}\b/;
    expect(blocks.filter((block) => hex.test(block))).toEqual([]);
    expect(css).toMatch(/\.compound-card__title\s*\{[^}]*color:\s*var\(--fg\)/s);
    expect(css).toMatch(/\.knowledge-search__hit:hover\s*\{[^}]*background:\s*var\(/s);
    expect(css).toMatch(/\.instruction-editor__tab--active\s*\{[^}]*color:\s*var\(--accent\)/s);
    expect(css).toMatch(/\.backlinks__item\s*\{[^}]*color:\s*var\(--accent\)/s);
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
      '.session-composer__card',
      '.session-composer__model',
      '.session-composer__send',
      '.session-chat-column',
      '.message-copy',
      '.content-canvas--session-focus',
    ]) {
      expect(css).toContain(token);
    }
  });

  it('aligns session header height with rail iconbar (48px chrome row)', () => {
    expect(css).toMatch(/\.rail-iconbar\s*\{[^}]*height:\s*48px/s);
    expect(css).toMatch(/\.session-focus__header\s*\{[^}]*height:\s*48px/s);
    expect(css).toMatch(/\.session-focus__header\s*\{[^}]*min-height:\s*48px/s);
  });

  it('gives session tabs and message thread breathing room', () => {
    expect(css).toMatch(/\.session-tabs\s*\{[^}]*padding:\s*8px\s+20px\s+0/s);
    expect(css).toMatch(/\.session-tab\s*\{[^}]*min-height:\s*40px/s);
    expect(css).toMatch(/\.session-focus__messages\s*\{[^}]*padding:\s*36px\s+0\s+24px/s);
    expect(css).toMatch(/\.session-chat-column\s*\{[^}]*padding-top:\s*8px/s);
    expect(css).toMatch(/\.message-thread\s*\{[^}]*gap:\s*22px/s);
    expect(css).toMatch(/\.message-bubble\s*\{[^}]*padding:\s*14px\s+18px/s);
  });

  it('places assistant copy as footer icon with date chrome', () => {
    expect(css).toContain('.message-assistant__footer');
    expect(css).toContain('.message-assistant__date');
    expect(css).toMatch(/\.message-copy\s*\{[^}]*width:\s*28px/s);
    expect(css).toContain('.message-copy__icon');
    expect(css).toMatch(/\.message-assistant__date\[data-tooltip\]::after/);
  });

  it('collapsed rail restore control uses 48px chrome row without extra top pad', () => {
    expect(css).toContain('.project-rail--collapsed');
    expect(css).toContain('.rail-collapsed-chrome');
    expect(css).toMatch(
      /\.project-workspace\[data-rail=["']collapsed["']\]\s+\.project-workspace__rail\s*\{[^}]*padding-top:\s*0/s,
    );
    expect(css).toMatch(/\.rail-collapsed-chrome\s*\{[^}]*height:\s*48px/s);
    expect(css).toMatch(/\.rail-collapsed-chrome\s*\{[^}]*border-bottom:\s*1px\s+solid/s);
  });

  it('declares project-workspace responsive breakpoints', () => {
    expect(css).toMatch(/@media\s*\(max-width:\s*1280px\)/);
    expect(css).toMatch(/@media\s*\(max-width:\s*1100px\)/);
    expect(css).toMatch(/@media\s*\(max-width:\s*960px\)/);
    // Overlay rail on narrow open mode so chat keeps width
    expect(css).toMatch(
      /@media\s*\(max-width:\s*960px\)[\s\S]*?\.project-workspace\[data-rail=['"]open['"]\]\s+\.project-workspace__rail\s*\{[^}]*position:\s*fixed/s,
    );
  });

  it('user bubbles use soft lavender not solid accent fill (Claude chat)', () => {
    const userBubble = css.match(/\.message-bubble--user\s*\{[^}]*\}/);
    expect(userBubble?.[0]).toBeTruthy();
    // Soft surface + dark text (layout-chat.png), not brand-blue pill
    expect(userBubble![0]).toMatch(/background:\s*(#ede9fe|#eef2ff|#f3e8ff)/i);
    expect(userBubble![0]).not.toMatch(/background:\s*var\(--accent\)/);
    expect(css).toMatch(/\.message-bubble--user p\s*\{[^}]*color:\s*#18181b/s);
  });

  it('declares hub/rail workspace tokens', () => {
    for (const token of [
      '.project-workspace',
      '.project-workspace__main',
      '.project-workspace__rail',
      '.project-rail',
      '.rail-iconbar',
      '.rail-icon',
      '.rail-icon--active',
      '.rail-panel',
      '.rail-memory-preview',
      '.hub-start',
      '.hub-start__title',
      '.hub-composer',
      '.hub-session-list',
      '.hub-session-list__label',
      '.session-row',
      '.session-row__icon',
      '.session-row__title',
      '.session-row__date',
      '.session-row__menu',
      '.content-canvas--project-workspace',
    ]) {
      expect(css).toContain(token);
    }

    expect(css).toMatch(/\.project-workspace\[data-rail=['"]open['"]\]/);
    expect(css).toMatch(/\.project-workspace\[data-rail=['"]expanded['"]\]/);
    expect(css).toMatch(/\.project-workspace\[data-rail=['"]collapsed['"]\]/);
    // Single column when expanded: main is display:none and leaves the grid;
    // a two-track "0 1fr" would assign the rail the zero-width first track.
    expect(css).toMatch(
      /data-rail=['"]expanded['"][^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s*;/s,
    );
    expect(css).not.toMatch(
      /data-rail=['"]expanded['"][^}]*grid-template-columns:\s*0\s+minmax\(0,\s*1fr\)/s,
    );
    expect(css).toMatch(
      /\.project-workspace\[data-rail=['"]expanded['"]\]\s+\.project-workspace__main\s*\{[^}]*display:\s*none/s,
    );
    expect(css).toMatch(/data-rail=['"]collapsed['"][^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s+48px/s);
  });

  it('locks project hub height so chat scrollbar sits on the main edge by the rail', () => {
    // Viewport-locked shell only when hub workspace is present
    expect(css).toMatch(
      /\.app-shell:has\(\.project-workspace\)\s*\{[^}]*height:\s*100vh/s,
    );
    expect(css).toMatch(
      /\.app-shell:has\(\.project-workspace\)\s*\{[^}]*overflow:\s*hidden/s,
    );
    expect(css).toMatch(
      /\.content-canvas--project-workspace\s*\{[^}]*overflow:\s*hidden/s,
    );
    expect(css).toMatch(/\.project-workspace\s*\{[^}]*height:\s*100%/s);
    expect(css).toMatch(/\.project-workspace__rail\s*\{[^}]*overflow:\s*hidden/s);
    // Rail panel scrolls only when content overflows
    expect(css).toMatch(/\.rail-panel\s*\{[^}]*overflow-y:\s*auto/s);
    // Message list is the chat scroller (full main width → scrollbar next to rail)
    expect(css).toMatch(
      /\.session-focus__messages\s*\{[^}]*overflow-y:\s*auto/s,
    );
    expect(css).toMatch(
      /\.session-chat-column\s*\{[^}]*max-width:\s*none/s,
    );
    // Conversation column stays ~44rem centered inside the full-bleed scroller
    expect(css).toMatch(
      /\.session-focus__messages\s*>\s*\.message-row\s*\{[^}]*max-width:\s*44rem/s,
    );
    // Config instructions fill the rail and scroll with content
    expect(css).toContain('.rail-panel--config');
    expect(css).toMatch(
      /\.rail-config-field__input\s*\{[^}]*overflow-y:\s*auto/s,
    );
    expect(css).toMatch(
      /\.rail-config-field__input\s*\{[^}]*resize:\s*none/s,
    );
  });

  it('hub project title is 1.5rem', () => {
    expect(css).toMatch(/\.hub-header__title\s*\{[^}]*font-size:\s*1\.5rem/s);
  });

  it('hub start and session list are full width; session rows have hover', () => {
    expect(css).toMatch(/\.hub-start\s*\{[^}]*max-width:\s*none/s);
    expect(css).toMatch(/\.hub-session-list\s*\{[^}]*max-width:\s*none/s);
    expect(css).toMatch(/\.session-row:hover\s*\{[^}]*background:/s);
  });

  it('hub composer card is full width with 20px radius', () => {
    // Must appear after .session-composer__card and force full width
    const hubOverride = css.match(
      /\.hub-composer \.session-composer__card[\s\S]*?max-width:\s*none\s*!important/,
    );
    expect(hubOverride).toBeTruthy();
    expect(css).toMatch(
      /\.hub-composer \.session-composer__card[\s\S]*?border-radius:\s*20px/,
    );
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
