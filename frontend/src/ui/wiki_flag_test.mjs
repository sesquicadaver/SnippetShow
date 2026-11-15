/**
 * SS-20 / SS-25: man flag → argv helpers.
 * Run: node --test frontend/src/ui/wiki_flag_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import {
  commandFromFlag,
  expandBraceOptions,
  normalizeManFlag,
  renderDescriptionLead,
  shellSafeProgram,
} from './wiki.js';

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

describe('renderDescriptionLead (SS-41)', () => {
  it('renders summary before man section markup', () => {
    const html = renderDescriptionLead(
      { summary: 'network ping utility', version: '1.2', category: 'net' },
      { includeVersion: true, includeCategory: true },
    );
    assert.match(html, /wiki-desc/);
    assert.match(html, /network ping utility/);
    assert.match(html, /1\.2/);
    assert.match(html, /net/);
  });

  it('shows pending hint when summary is empty', () => {
    const html = renderDescriptionLead({ summary: '' }, {});
    assert.match(html, /is-pending/);
    assert.match(html, /збагачення/);
  });
});

describe('expandBraceOptions (SS-26)', () => {
  it('expands -ao{a|s|t|u} into four runnable flags', () => {
    const rows = expandBraceOptions('-ao{a|s|t|u}', 'set overwrite mode');
    assert.equal(rows.length, 4);
    assert.deepEqual(rows.map((r) => r.flag), ['-aoa', '-aos', '-aot', '-aou']);
    assert.ok(rows[0].description.includes('(варіант: a)'));
  });

  it('leaves non-brace flags alone', () => {
    const rows = expandBraceOptions('-v', 'verbose');
    assert.equal(rows.length, 1);
    assert.equal(rows[0].flag, '-v');
  });
});
