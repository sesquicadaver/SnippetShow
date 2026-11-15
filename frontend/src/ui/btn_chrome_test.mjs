/**
 * SS-42: one primary gold; no btn-accent; toolbar must not mute primary.
 * Run: node --test frontend/src/ui/btn_chrome_test.mjs
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { describe, it } from 'node:test';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '../..');
const css = readFileSync(join(root, 'src/styles/main.css'), 'utf8');
const wiki = readFileSync(join(root, 'src/ui/wiki.js'), 'utf8');
const builder = readFileSync(join(root, 'src/ui/builder.js'), 'utf8');
const toolbar = readFileSync(join(root, 'src/ui/toolbar.js'), 'utf8');

describe('button chrome (SS-42)', () => {
  it('removes btn-accent from CSS and UI markup', () => {
    assert.equal(css.includes('btn-accent'), false);
    assert.equal(wiki.includes('btn-accent'), false);
    assert.equal(builder.includes('btn-accent'), false);
    assert.equal(toolbar.includes('btn-accent'), false);
  });

  it('keeps a single primary gold token and toolbar override', () => {
    assert.match(css, /\.btn-primary\s*\{/);
    assert.match(css, /\.toolbar\s+\.btn\.btn-primary\s*\{/);
    assert.match(css, /\.toolbar\s+\.btn\.btn-danger\s*\{/);
  });

  it('marks Update Index and Save-to-DB as btn-primary', () => {
    assert.match(toolbar, /class="btn btn-primary"[^>]*data-action="update"/);
    assert.match(builder, /class="btn btn-primary"[^>]*data-action="script-save"/);
  });
});
