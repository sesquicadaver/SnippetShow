/**
 * SS-27 / D13: man parse only for favorites that need it.
 * Run: node --test frontend/src/ui/shell_select_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { programNeedsManParse } from './shell.js';

describe('programNeedsManParse (SS-27 / D13)', () => {
  it('skips unknown / non-favorite rows', () => {
    assert.equal(programNeedsManParse(null), false);
    assert.equal(programNeedsManParse(undefined), false);
    assert.equal(programNeedsManParse({ isInstalled: true, manParsed: false, isFavorite: false }), false);
  });

  it('parses favorite installed rows without cached man', () => {
    assert.equal(programNeedsManParse({
      isInstalled: true, isFavorite: true, manParsed: false, manStatus: 'unknown',
    }), true);
  });

  it('skips already-parsed or absent favorites', () => {
    assert.equal(programNeedsManParse({
      isInstalled: true, isFavorite: true, manParsed: true,
    }), false);
    assert.equal(programNeedsManParse({
      isInstalled: true, isFavorite: true, manParsed: false, manStatus: 'absent',
    }), false);
  });

  it('skips uninstalled catalog rows', () => {
    assert.equal(programNeedsManParse({ isInstalled: false, manParsed: false, isFavorite: true }), false);
  });
});
