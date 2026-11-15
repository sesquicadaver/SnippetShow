/**
 * SS-20 / SS-25: man flag → argv helpers.
 * Run: node --test frontend/src/ui/wiki_flag_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import {
  bareCommandLine,
  commandFromFlag,
  expandBraceOptions,
  normalizeManFlag,
  renderDescriptionLead,
  renderManTableBody,
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

describe('bare command row (SS-44)', () => {
  it('bareCommandLine is shell-safe program alone', () => {
    assert.equal(bareCommandLine('clear'), 'clear');
    assert.equal(bareCommandLine('['), '/usr/bin/[');
  });

  it('always renders + Script / checkbox for bare cmd when flags empty', () => {
    const html = renderManTableBody('clear', [], new Set(), { sourceEmpty: true, filterActive: false });
    assert.match(html, /data-bare="1"/);
    assert.match(html, /data-action="check-script"[^>]*data-cmd="clear"/);
    assert.match(html, /data-action="script"/);
    assert.match(html, /Базова команда/);
    assert.match(html, /Прапорців|прапорців/);
  });

  it('keeps bare row above flags and dedupes identical bare lines', () => {
    const html = renderManTableBody(
      'ls',
      [{ flag: '-l', description: 'long' }, { flag: '', description: 'noop' }],
      new Set(['ls']),
      { sourceEmpty: false, filterActive: false },
    );
    assert.match(html, /man-bare-row[\s\S]*data-cmd="ls"[\s\S]*checked/);
    assert.match(html, /data-cmd="ls -l"/);
    // empty flag expands to bare — must not duplicate a second bare action row
    assert.equal((html.match(/data-bare="1"/g) || []).length, 1);
  });
});
