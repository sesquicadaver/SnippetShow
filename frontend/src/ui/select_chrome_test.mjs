/**
 * SS-43 / SS-45: Section / letter filters must use dark app chrome (not system white).
 * SS-45 replaces native <select> with custom toolbar-select under WebKitGTK.
 * Run: node --test frontend/src/ui/select_chrome_test.mjs
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { describe, it } from 'node:test';
import { fileURLToPath } from 'node:url';
import {
  patchToolbarSelect,
  toolbarSelectLabel,
  toolbarSelectMarkup,
} from './toolbar.js';

const rootDir = join(dirname(fileURLToPath(import.meta.url)), '../..');
const css = readFileSync(join(rootDir, 'src/styles/main.css'), 'utf8');
const toolbarSrc = readFileSync(join(rootDir, 'src/ui/toolbar.js'), 'utf8');

describe('toolbar select chrome (SS-45)', () => {
  it('opts into dark color-scheme for the document', () => {
    assert.match(css, /color-scheme:\s*dark/);
  });

  it('styles custom toolbar-select trigger/menu (no native select dependency)', () => {
    assert.match(css, /\.toolbar-select-trigger/);
    assert.match(css, /\.toolbar-select-menu/);
    assert.match(css, /background-color:\s*var\(--bg-elev\)/);
    assert.equal(css.includes('category-select option'), false);
    // Mount path must use custom markup helper (not a live <select class=…>).
    assert.match(toolbarSrc, /toolbarSelectMarkup\('category-select'/);
    assert.match(toolbarSrc, /toolbarSelectMarkup\('letter-select'/);
    assert.equal(/<select\s+class=/.test(toolbarSrc), false);
  });

  it('renders markup without native <select> and patches labels', () => {
    const html = toolbarSelectMarkup('letter-select', 'Літера');
    assert.match(html, /data-toolbar-select/);
    assert.match(html, /toolbar-select-trigger/);
    assert.equal(html.includes('<select'), false);
    assert.equal(toolbarSelectLabel([{ value: 'A', label: 'A' }], 'A'), 'A');

    // Lightweight DOM shim for patchToolbarSelect.
    const root = {
      hidden: false,
      dataset: {},
      querySelector(sel) {
        if (sel === '.toolbar-select-label') return this._label;
        if (sel === '.toolbar-select-menu') return this._menu;
        return null;
      },
      _label: { textContent: '' },
      _menu: { innerHTML: '' },
    };
    patchToolbarSelect(root, {
      options: [{ value: 'all', label: 'Всі' }, { value: 'B', label: 'B' }],
      value: 'B',
    });
    assert.equal(root.dataset.value, 'B');
    assert.equal(root._label.textContent, 'B');
    assert.match(root._menu.innerHTML, /data-value="B"/);
    assert.match(root._menu.innerHTML, /is-selected/);
  });
});
