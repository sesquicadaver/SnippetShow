/**
 * SS-43 / SS-45 / SS-46 / SS-47: Section / letter filters — dark chrome, open, hit-test.
 * SS-45 replaces native <select> with custom toolbar-select under WebKitGTK.
 * SS-46: menus must not be clipped by .toolbar overflow; toggle must open.
 * SS-47: toolbar stacks above .app-body so listbox options receive clicks.
 * Run: node --test frontend/src/ui/select_chrome_test.mjs
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { describe, it } from 'node:test';
import { fileURLToPath } from 'node:url';
import {
  closeToolbarSelects,
  eatNextClick,
  eventElement,
  patchToolbarSelect,
  setToolbarSelectOpen,
  toolbarSelectLabel,
  toolbarSelectMarkup,
} from './toolbar.js';

const rootDir = join(dirname(fileURLToPath(import.meta.url)), '../..');
const css = readFileSync(join(rootDir, 'src/styles/main.css'), 'utf8');
const toolbarSrc = readFileSync(join(rootDir, 'src/ui/toolbar.js'), 'utf8');

/**
 * Minimal classList shim for open/close tests.
 * @param {string[]} initial
 */
function classListShim(initial = []) {
  const set = new Set(initial);
  return {
    contains: (c) => set.has(c),
    add: (c) => { set.add(c); },
    remove: (c) => { set.delete(c); },
    toggle: (c, force) => {
      if (force === true) set.add(c);
      else if (force === false) set.delete(c);
      else if (set.has(c)) set.delete(c);
      else set.add(c);
      return set.has(c);
    },
  };
}

describe('toolbar select chrome (SS-45 / SS-46)', () => {
  it('opts into dark color-scheme for the document', () => {
    assert.match(css, /color-scheme:\s*dark/);
  });

  it('styles custom toolbar-select trigger/menu (no native select dependency)', () => {
    assert.match(css, /\.toolbar-select-trigger/);
    assert.match(css, /\.toolbar-select-menu/);
    assert.match(css, /background-color:\s*var\(--bg-elev\)/);
    assert.equal(css.includes('category-select option'), false);
    assert.match(toolbarSrc, /toolbarSelectMarkup\('category-select'/);
    assert.match(toolbarSrc, /toolbarSelectMarkup\('letter-select'/);
    assert.equal(/<select\s+class=/.test(toolbarSrc), false);
  });

  it('keeps .toolbar overflow visible so absolute menus are not clipped (SS-46)', () => {
    // Base rule must not clip dropdowns (strip comments — they may mention the bug).
    const toolbarBlock = css.match(/\.toolbar\s*\{[^}]*\}/);
    assert.ok(toolbarBlock, 'expected .toolbar rule');
    const decls = toolbarBlock[0].replace(/\/\*[\s\S]*?\*\//g, '');
    assert.match(decls, /overflow:\s*visible/);
    assert.equal(/overflow:\s*hidden/.test(decls), false);
  });

  it('stacks .toolbar above .app-body so overflowing menus receive hits (SS-47)', () => {
    const toolbarBlock = css.match(/\.toolbar\s*\{[^}]*\}/);
    assert.ok(toolbarBlock, 'expected .toolbar rule');
    const decls = toolbarBlock[0].replace(/\/\*[\s\S]*?\*\//g, '');
    assert.match(decls, /z-index:\s*30/);
    assert.match(css, /\.toolbar-select-menu\s*\{[^}]*pointer-events:\s*auto/s);
    assert.match(css, /\.toolbar-select-option\s*\{[^}]*pointer-events:\s*auto/s);
    assert.match(toolbarSrc, /eatNextClick/);
    assert.match(toolbarSrc, /menu\.addEventListener\('pointerdown'/);
  });

  it('eatNextClick swallows the following capture-phase click (SS-47)', () => {
    const calls = [];
    const prev = globalThis.document;
    const listeners = new Map();
    globalThis.document = {
      addEventListener(type, fn, opts) {
        listeners.set(type, { fn, opts });
      },
      removeEventListener(type, fn) {
        const hit = listeners.get(type);
        if (hit && hit.fn === fn) listeners.delete(type);
      },
    };
    try {
      eatNextClick();
      assert.equal(listeners.has('click'), true);
      assert.equal(listeners.get('click').opts, true);
      const ev = {
        preventDefault() { calls.push('prevent'); },
        stopPropagation() { calls.push('stop'); },
      };
      listeners.get('click').fn(ev);
      assert.deepEqual(calls, ['prevent', 'stop']);
      assert.equal(listeners.has('click'), false);
    } finally {
      globalThis.document = prev;
    }
  });

  it('renders markup without native <select> and patches labels', () => {
    const html = toolbarSelectMarkup('letter-select', 'Літера');
    assert.match(html, /data-toolbar-select/);
    assert.match(html, /toolbar-select-trigger/);
    assert.equal(html.includes('<select'), false);
    assert.equal(toolbarSelectLabel([{ value: 'A', label: 'A' }], 'A'), 'A');

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

  it('opens and closes toolbar select (SS-46)', () => {
    const trigger = {
      attrs: { 'aria-expanded': 'false' },
      setAttribute(k, v) { this.attrs[k] = v; },
    };
    const menu = { hidden: true };
    const root = {
      classList: classListShim(),
      querySelector(sel) {
        if (sel === '.toolbar-select-trigger') return trigger;
        if (sel === '.toolbar-select-menu') return menu;
        return null;
      },
    };
    const host = {
      querySelectorAll() {
        return root.classList.contains('is-open') ? [root] : [];
      },
    };

    setToolbarSelectOpen(host, root, true);
    assert.equal(root.classList.contains('is-open'), true);
    assert.equal(trigger.attrs['aria-expanded'], 'true');
    assert.equal(menu.hidden, false);

    closeToolbarSelects(host);
    assert.equal(root.classList.contains('is-open'), false);
    assert.equal(trigger.attrs['aria-expanded'], 'false');
    assert.equal(menu.hidden, true);
  });

  it('eventElement resolves Element and text-node parent', () => {
    const el = { closest() { return null; } };
    assert.equal(eventElement({ target: el }), el);

    const parent = { closest() { return null; } };
    const text = { parentElement: parent };
    assert.equal(eventElement({ target: text }), parent);
    assert.equal(eventElement({ target: null }), null);
  });
});
