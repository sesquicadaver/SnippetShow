/**
 * SS-23: navigator virtual scroll must not use a stale first-render state.
 * Run: node --test frontend/src/ui/navigator_test.mjs
 */
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { resolveNavState, shouldShowFavoriteControl, windowRange } from './navigator.js';

describe('windowRange', () => {
  it('clamps to program count', () => {
    const { start, end } = windowRange(0, 360, 5, 36, 12);
    assert.equal(start, 0);
    assert.equal(end, 5);
  });

  it('advances start with scrollTop and overscan', () => {
    const { start, end } = windowRange(36 * 40, 360, 500, 36, 12);
    assert.equal(start, 40 - 12);
    assert.ok(end > start);
    assert.ok(end <= 500);
  });
});

describe('resolveNavState (SS-23)', () => {
  it('reads host._navState so scroll ignores closed-over first render', () => {
    const staleFirstRender = { programs: [], loading: true };
    const host = { _navState: { programs: [{ name: '2ping' }], loading: false } };
    // Old bug: paintWindow(host, staleFirstRender) preferred the closed-over arg.
    const buggy = staleFirstRender || host._navState;
    assert.equal(buggy.programs.length, 0);
    const fixed = resolveNavState(host);
    assert.equal(fixed.programs.length, 1);
    assert.equal(fixed.programs[0].name, '2ping');
  });

  it('returns null when host has no stored state', () => {
    assert.equal(resolveNavState({}), null);
    assert.equal(resolveNavState(null), null);
  });
});

describe('shouldShowFavoriteControl (SS-34)', () => {
  it('shows for installed PATH commands', () => {
    assert.equal(shouldShowFavoriteControl({ kind: 'command', isInstalled: true }), true);
    assert.equal(shouldShowFavoriteControl({ isInstalled: true }), true);
  });

  it('hides for packages and uninstalled rows', () => {
    assert.equal(shouldShowFavoriteControl({ kind: 'package', isInstalled: true }), false);
    assert.equal(shouldShowFavoriteControl({ kind: 'command', isInstalled: false }), false);
    assert.equal(shouldShowFavoriteControl(null), false);
  });
});
