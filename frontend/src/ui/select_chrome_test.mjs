/**
 * SS-43: Section / letter <select> must use dark app chrome (not system white).
 * Run: node --test frontend/src/ui/select_chrome_test.mjs
 */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { describe, it } from 'node:test';
import { fileURLToPath } from 'node:url';

const css = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), '../styles/main.css'),
  'utf8',
);

describe('toolbar select chrome (SS-43)', () => {
  it('opts into dark color-scheme for native controls', () => {
    assert.match(css, /color-scheme:\s*dark/);
  });

  it('restyles category and letter selects (appearance + solid bg + options)', () => {
    assert.match(css, /\.toolbar-filters \.category-select,\s*\n\.toolbar-filters \.letter-select\s*\{/);
    assert.match(css, /appearance:\s*none/);
    assert.match(css, /background-color:\s*var\(--bg-elev\)/);
    assert.match(css, /\.category-select option,\s*\n\.toolbar-filters \.letter-select option/);
    assert.doesNotMatch(
      css.match(/\.toolbar \.search-input,[\s\S]*?\.toolbar \.btn \{[\s\S]*?\}/)?.[0] || '',
      /background:\s*var\(--bg-elev\)/,
    );
  });
});
