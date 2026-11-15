/**
 * SS-27 / D13: man parse only for favorites that need it.
 * SS-52: warm-start bootstrap still queues Section enrich when categories empty.
 * Run: node --test frontend/src/ui/shell_select_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { programNeedsManParse, shouldQueueBootstrapEnrich } from './shell.js';

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

describe('shouldQueueBootstrapEnrich (SS-52)', () => {
  it('queues after names-only first run (added > 0)', () => {
    assert.equal(shouldQueueBootstrapEnrich(12, [], 12), true);
    assert.equal(shouldQueueBootstrapEnrich(1, ['utils'], 1), true);
  });

  it('queues on warm start when programs exist but categories are empty', () => {
    assert.equal(shouldQueueBootstrapEnrich(0, [], 40), true);
    assert.equal(shouldQueueBootstrapEnrich(0, null, 3), true);
  });

  it('skips when categories already filled and nothing was added', () => {
    assert.equal(shouldQueueBootstrapEnrich(0, ['utils', 'admin'], 40), false);
  });

  it('skips empty DB (no programs, nothing added)', () => {
    assert.equal(shouldQueueBootstrapEnrich(0, [], 0), false);
  });
});
