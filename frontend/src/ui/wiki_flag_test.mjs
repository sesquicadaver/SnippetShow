/**
 * SS-20 / SS-25: man flag → argv helpers.
 * Run: node --test frontend/src/ui/wiki_flag_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { commandFromFlag, normalizeManFlag, shellSafeProgram } from './wiki.js';

describe('shellSafeProgram (SS-25)', () => {
  it('maps [ and test to /usr/bin binaries', () => {
    assert.equal(shellSafeProgram('['), '/usr/bin/[');
    assert.equal(shellSafeProgram('test'), '/usr/bin/test');
    assert.equal(shellSafeProgram('ls'), 'ls');
  });
});

describe('commandFromFlag (SS-25)', () => {
  it('builds /usr/bin/[ --help instead of [ --help', () => {
    assert.equal(commandFromFlag('[', '--help'), '/usr/bin/[ --help');
    assert.equal(commandFromFlag('[', '[ --help'), '/usr/bin/[ --help');
  });

  it('keeps ordinary programs unchanged', () => {
    assert.equal(commandFromFlag('2ping', '--adaptive'), '2ping --adaptive');
    assert.equal(normalizeManFlag('-a, --all'), '-a --all');
  });
});
