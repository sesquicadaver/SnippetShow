/**
 * SS-27: selectProgram must not force man parse / list reload for cached rows.
 * Run: node --test frontend/src/ui/shell_select_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { programNeedsManParse } from './shell.js';

describe('programNeedsManParse (SS-27)', () => {
  it('parses unknown rows (missing from current list)', () => {
    assert.equal(programNeedsManParse(null), true);
    assert.equal(programNeedsManParse(undefined), true);
  });

  it('parses installed rows without cached man', () => {
    assert.equal(programNeedsManParse({ isInstalled: true, manParsed: false }), true);
  });

  it('skips already-parsed installed rows', () => {
    assert.equal(programNeedsManParse({ isInstalled: true, manParsed: true }), false);
  });

  it('skips uninstalled catalog rows', () => {
    assert.equal(programNeedsManParse({ isInstalled: false, manParsed: false }), false);
  });
});
