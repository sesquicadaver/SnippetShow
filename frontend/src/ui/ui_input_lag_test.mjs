/**
 * SS-30: search/toolbar must not remount on query; list soft-refresh;
 * bash highlight updates coalesce on rAF.
 * Run: node --test frontend/src/ui/ui_input_lag_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import {
  letterFilterOptions,
  normalizeLetterValue,
  shouldSyncSearchValue,
} from './toolbar.js';
import {
  SEARCH_DEBOUNCE_MS,
  shouldShowListLoading,
} from './shell.js';
import {
  createHighlightScheduler,
  highlightBash,
} from './builder.js';

describe('shouldSyncSearchValue (SS-30)', () => {
  it('never overwrites the field while focused', () => {
    assert.equal(shouldSyncSearchValue(true, 'curl', 'curl -'), false);
    assert.equal(shouldSyncSearchValue(true, '', 'x'), false);
  });

  it('syncs when unfocused and values differ', () => {
    assert.equal(shouldSyncSearchValue(false, 'old', 'new'), true);
    assert.equal(shouldSyncSearchValue(false, 'same', 'same'), false);
  });
});

describe('letter filter select (SS-36d)', () => {
  it('exposes Всі, A–Z, and 0–9 options', () => {
    const opts = letterFilterOptions();
    assert.equal(opts[0].value, 'all');
    assert.equal(opts[opts.length - 1].value, '0-9');
    assert.equal(opts.length, 28);
    assert.ok(opts.some((o) => o.value === 'A'));
    assert.ok(opts.some((o) => o.value === 'Z'));
  });

  it('normalizes letter state for the select value', () => {
    assert.equal(normalizeLetterValue(''), 'all');
    assert.equal(normalizeLetterValue('b'), 'B');
    assert.equal(normalizeLetterValue('0-9'), '0-9');
    assert.equal(normalizeLetterValue('??'), 'all');
  });
});

describe('search debounce / list loading (SS-30)', () => {
  it('uses a debounce above a single keystroke burst', () => {
    assert.ok(SEARCH_DEBOUNCE_MS >= 250);
  });

  it('shows loading overlay only when the list is empty', () => {
    assert.equal(shouldShowListLoading(0), true);
    assert.equal(shouldShowListLoading(12), false);
  });
});

describe('createHighlightScheduler (SS-30)', () => {
  it('coalesces multiple updates into one apply', () => {
    const queued = [];
    const schedule = (cb) => {
      queued.push(cb);
      return queued.length;
    };
    const applied = [];
    const push = createHighlightScheduler(schedule, (text) => applied.push(text));

    push('a');
    push('ab');
    push('abc');
    assert.equal(queued.length, 1);
    assert.deepEqual(applied, []);

    queued[0]();
    assert.deepEqual(applied, ['abc']);

    push('abcd');
    assert.equal(queued.length, 2);
    queued[1]();
    assert.deepEqual(applied, ['abc', 'abcd']);
  });
});

describe('highlightBash', () => {
  it('marks shebang and comments', () => {
    const html = highlightBash('#!/bin/bash\n# hi\necho ok');
    assert.match(html, /tok-shebang/);
    assert.match(html, /tok-comment/);
    assert.match(html, /tok-kw/);
  });
});
