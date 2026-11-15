import uk from '../i18n/uk.js';

const LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('');

/**
 * Options for the alphabet letter filter select (SS-36d).
 * @returns {Array<{value:string,label:string}>}
 */
export function letterFilterOptions() {
  return [
    { value: 'all', label: uk.alphabetAll },
    ...LETTERS.map((L) => ({ value: L, label: L })),
    { value: '0-9', label: uk.alphabetDigits },
  ];
}

/**
 * Normalize toolbar letter state for the select value (SS-36d).
 * @param {string|null|undefined} letter
 * @returns {string}
 */
export function normalizeLetterValue(letter) {
  const raw = String(letter ?? 'all').trim();
  if (!raw || raw === 'all') return 'all';
  if (raw === '0-9') return '0-9';
  const upper = raw.toUpperCase();
  if (LETTERS.includes(upper)) return upper;
  return 'all';
}

/**
 * SS-30: do not overwrite the live search field while the user is typing.
 * @param {boolean} inputFocused
 * @param {string} currentValue
 * @param {string} nextQuery
 * @returns {boolean}
 */
export function shouldSyncSearchValue(inputFocused, currentValue, nextQuery) {
  if (inputFocused) return false;
  return String(currentValue ?? '') !== String(nextQuery ?? '');
}

/**
 * SS-45: markup for a CSS-themed toolbar dropdown (replaces native <select>).
 * @param {string} className
 * @param {string} ariaLabel
 * @returns {string}
 */
export function toolbarSelectMarkup(className, ariaLabel) {
  return `
    <div class="toolbar-select ${esc(className)}" data-toolbar-select>
      <button type="button" class="toolbar-select-trigger" aria-haspopup="listbox"
        aria-expanded="false" aria-label="${esc(ariaLabel)}">
        <span class="toolbar-select-label"></span>
      </button>
      <ul class="toolbar-select-menu" role="listbox" hidden></ul>
    </div>
  `;
}

/**
 * Resolve label for a value from option list.
 * @param {Array<{value:string,label:string}>} options
 * @param {string} value
 * @returns {string}
 */
export function toolbarSelectLabel(options, value) {
  const hit = (options || []).find((o) => o.value === value);
  return hit ? hit.label : String(value ?? '');
}

/**
 * Rebuild options + current label for a custom toolbar select (SS-45).
 * @param {HTMLElement|null} root
 * @param {{options:Array<{value:string,label:string}>, value:string, hidden?: boolean}} state
 */
export function patchToolbarSelect(root, state) {
  if (!root) return;
  const options = state.options || [];
  const value = String(state.value ?? '');
  root.hidden = !!state.hidden;
  root.dataset.value = value;
  const label = root.querySelector('.toolbar-select-label');
  if (label) label.textContent = toolbarSelectLabel(options, value) || '—';
  const menu = root.querySelector('.toolbar-select-menu');
  if (!menu) return;
  menu.innerHTML = options.map((o) => `
    <li role="option" class="toolbar-select-option${o.value === value ? ' is-selected' : ''}"
      data-value="${esc(o.value)}" aria-selected="${o.value === value ? 'true' : 'false'}">${esc(o.label)}</li>
  `).join('');
}

/**
 * Close all open toolbar selects under host.
 * @param {HTMLElement} host
 */
export function closeToolbarSelects(host) {
  host.querySelectorAll('[data-toolbar-select].is-open').forEach((el) => {
    el.classList.remove('is-open');
    const trigger = el.querySelector('.toolbar-select-trigger');
    const menu = el.querySelector('.toolbar-select-menu');
    if (trigger) trigger.setAttribute('aria-expanded', 'false');
    if (menu) menu.hidden = true;
  });
}

/**
 * Builds / updates the Wiki toolbar: mode radios, search, letter select, update controls.
 * First call mounts DOM; later calls patch in place so the search input keeps focus (SS-30).
 * Brand = SnippetShow only (no subtitle). Letter filter = compact dropdown (SS-36d / SS-45).
 * SS-40 restores a single coherent composition (segmented modes, no wrap sprawl).
 * @param {HTMLElement} host
 * @param {{
 *   mode: string,
 *   letter: string,
 *   query: string,
 *   category: string,
 *   categories: string[],
 *   noMan: boolean,
 *   busy: boolean,
 *   status: string,
 *   progress: {done:number,total:number}|null,
 *   onMode: (mode:string)=>void,
 *   onLetter: (letter:string)=>void,
 *   onQuery: (query:string)=>void,
 *   onCategory: (category:string)=>void,
 *   onNoMan: (on:boolean)=>void,
 *   onUpdate: ()=>void,
 *   onCancel: ()=>void,
 *   onSyncCatalog: ()=>void,
 *   onExportBackup: ()=>void,
 *   onImportBackup: ()=>void,
 * }} state
 */
export function renderToolbar(host, state) {
  host._toolbarState = state;

  if (!host.dataset.ready) {
    mountToolbar(host);
    host.dataset.ready = '1';
    bindToolbarEvents(host);
  }

  patchToolbar(host, state);
}

function mountToolbar(host) {
  host.innerHTML = `
    <div class="toolbar" role="toolbar" aria-label="Керування Wiki">
      <div class="toolbar-brand">
        <span class="toolbar-brand-name">${esc(uk.brand)}</span>
      </div>
      <div class="toolbar-modes" role="radiogroup" aria-label="Режим списку">
        <label class="mode" data-mode="installed">
          <input type="radio" name="mode" value="installed"/>
          ${esc(uk.modeInstalled)}
        </label>
        <label class="mode" data-mode="favorites">
          <input type="radio" name="mode" value="favorites"/>
          ${esc(uk.modeFavorites)}
        </label>
        <label class="mode" data-mode="all">
          <input type="radio" name="mode" value="all"/>
          ${esc(uk.modeAll)}
        </label>
      </div>
      <div class="toolbar-search">
        <input type="search" class="search-input" placeholder="${esc(uk.searchPlaceholder)}"
          value="" autocomplete="off" spellcheck="false"/>
      </div>
      <div class="toolbar-filters">
        ${toolbarSelectMarkup('category-select', uk.filterCategoryAll)}
        <label class="noman-filter">
          <input type="checkbox" class="noman-input"/>
          ${esc(uk.filterNoMan)}
        </label>
        ${toolbarSelectMarkup('letter-select', uk.alphabetFilter || 'Літера')}
      </div>
      <div class="toolbar-actions">
        <button type="button" class="btn btn-primary" data-action="update"></button>
        <button type="button" class="btn btn-danger" data-action="cancel" hidden
          title="${esc(uk.cancelIndex)}">${esc(uk.cancelIndex)}</button>
        <button type="button" class="btn" data-action="catalog" title="${esc(uk.modeAll)}">
          ${esc(uk.syncCatalog)}
        </button>
        <button type="button" class="btn" data-action="export-backup" title="${esc(uk.exportBackup)}">
          ${esc(uk.exportBackup)}
        </button>
        <button type="button" class="btn" data-action="import-backup" title="${esc(uk.importBackup)}">
          ${esc(uk.importBackup)}
        </button>
      </div>
      <div class="toolbar-status" aria-live="polite">
        <div class="progress-track" hidden>
          <div class="progress-bar" style="width:0%"></div>
        </div>
        <span class="status-text"></span>
      </div>
    </div>
  `;
}

function bindToolbarEvents(host) {
  host.querySelectorAll('input[name="mode"]').forEach((el) => {
    el.addEventListener('change', () => host._toolbarState?.onMode?.(el.value));
  });
  const search = host.querySelector('.search-input');
  search.addEventListener('input', () => host._toolbarState?.onQuery?.(search.value));
  host.querySelector('.noman-input')?.addEventListener('change', (e) => {
    host._toolbarState?.onNoMan?.(!!e.target.checked);
  });

  bindToolbarSelect(host, host.querySelector('.category-select'), (value) => {
    host._toolbarState?.onCategory?.(value);
  });
  bindToolbarSelect(host, host.querySelector('.letter-select'), (value) => {
    host._toolbarState?.onLetter?.(value);
  });

  host.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeToolbarSelects(host);
  });
  // Outside clicks once (SS-45 / SS-46). Resolve Element so text-node targets
  // still hit .closest; ignore events that originate inside a toolbar-select.
  if (!host._toolbarOutsideBound) {
    host._toolbarOutsideBound = true;
    document.addEventListener('pointerdown', (e) => {
      if (!host.isConnected) return;
      const el = eventElement(e);
      if (el && host.contains(el) && el.closest('[data-toolbar-select]')) return;
      closeToolbarSelects(host);
    });
  }

  host.querySelector('[data-action="update"]').addEventListener('click', () => {
    host._toolbarState?.onUpdate?.();
  });
  host.querySelector('[data-action="cancel"]').addEventListener('click', () => {
    host._toolbarState?.onCancel?.();
  });
  host.querySelector('[data-action="catalog"]').addEventListener('click', () => {
    host._toolbarState?.onSyncCatalog?.();
  });
  host.querySelector('[data-action="export-backup"]').addEventListener('click', () => {
    host._toolbarState?.onExportBackup?.();
  });
  host.querySelector('[data-action="import-backup"]').addEventListener('click', () => {
    host._toolbarState?.onImportBackup?.();
  });
}

/**
 * Resolve the Element for a pointer/mouse event (Text nodes have no .closest).
 * @param {Event} e
 * @returns {Element|null}
 */
export function eventElement(e) {
  const t = e?.target;
  if (!t || typeof t !== 'object') return null;
  if (typeof t.closest === 'function') return /** @type {Element} */ (t);
  const parent = t.parentElement;
  if (parent && typeof parent.closest === 'function') return parent;
  return null;
}

/**
 * Open or close one toolbar select (SS-45 / SS-46).
 * @param {HTMLElement} host
 * @param {HTMLElement} root
 * @param {boolean} open
 */
export function setToolbarSelectOpen(host, root, open) {
  closeToolbarSelects(host);
  if (!open || !root) return;
  const trigger = root.querySelector('.toolbar-select-trigger');
  const menu = root.querySelector('.toolbar-select-menu');
  root.classList.add('is-open');
  if (trigger) trigger.setAttribute('aria-expanded', 'true');
  if (menu) menu.hidden = false;
}

/**
 * @param {HTMLElement} host
 * @param {HTMLElement|null} root
 * @param {(value:string)=>void} onChange
 */
function bindToolbarSelect(host, root, onChange) {
  if (!root) return;
  const trigger = root.querySelector('.toolbar-select-trigger');
  const menu = root.querySelector('.toolbar-select-menu');
  if (!trigger || !menu) return;

  // pointerdown + stopPropagation so the document outside-closer does not race
  // the toggle (mousedown/pointerdown fires before click).
  trigger.addEventListener('pointerdown', (e) => {
    e.preventDefault();
    e.stopPropagation();
    const willOpen = !root.classList.contains('is-open');
    setToolbarSelectOpen(host, root, willOpen);
  });

  // Activate on pointerdown (SS-47): stopPropagation keeps the document
  // outside-closer from racing; preventDefault + a one-shot capture click
  // eater stops the following click from hitting .nav-row after the menu closes.
  menu.addEventListener('pointerdown', (e) => {
    e.preventDefault();
    e.stopPropagation();
    const el = eventElement(e);
    const opt = el?.closest?.('[data-value]');
    if (!opt || !menu.contains(opt)) return;
    const value = opt.getAttribute('data-value') || '';
    root.dataset.value = value;
    closeToolbarSelects(host);
    onChange?.(value);
    eatNextClick();
  });
}

/**
 * Swallow the click that follows a menu option pointerdown so it cannot
 * activate the programs list under the closed listbox (SS-47).
 */
export function eatNextClick() {
  let done = false;
  const finish = () => {
    if (done) return;
    done = true;
    document.removeEventListener('click', eat, true);
  };
  const eat = (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    finish();
  };
  document.addEventListener('click', eat, true);
  // Failsafe if pointer cancels without a click.
  setTimeout(finish, 500);
}

function patchToolbar(host, state) {
  const progress = state.progress;
  const pct = progress && progress.total > 0
    ? Math.min(100, Math.round((100 * progress.done) / progress.total))
    : state.busy ? 15 : 0;

  host.querySelectorAll('[data-mode]').forEach((label) => {
    const mode = label.getAttribute('data-mode');
    const active = state.mode === mode;
    label.classList.toggle('is-active', active);
    const input = label.querySelector('input');
    if (input) input.checked = active;
  });

  const search = host.querySelector('.search-input');
  if (search && shouldSyncSearchValue(
    document.activeElement === search,
    search.value,
    state.query,
  )) {
    search.value = state.query || '';
  }

  const cats = state.categories || [];
  const curCat = state.category || '';
  const categoryRoot = host.querySelector('.category-select');
  // Avoid clobbering an open menu mid-interaction (SS-46).
  if (!(categoryRoot?.classList.contains('is-open'))) {
    patchToolbarSelect(categoryRoot, {
      options: [{ value: '', label: uk.filterCategoryAll }]
        .concat(cats.map((c) => ({ value: c, label: c }))),
      value: curCat,
      hidden: state.mode === 'all',
    });
  } else if (state.mode === 'all') {
    // Still honor hide when switching to packages mode while open.
    categoryRoot.hidden = true;
    closeToolbarSelects(host);
  }

  const noMan = host.querySelector('.noman-input');
  if (noMan) {
    noMan.checked = !!state.noMan;
    noMan.closest('.noman-filter').hidden = state.mode === 'all';
  }

  const letterVal = normalizeLetterValue(state.letter);
  const letterRoot = host.querySelector('.letter-select');
  if (!(letterRoot?.classList.contains('is-open'))) {
    patchToolbarSelect(letterRoot, {
      options: letterFilterOptions(),
      value: letterVal,
    });
  }

  const updateBtn = host.querySelector('[data-action="update"]');
  if (updateBtn) {
    updateBtn.disabled = !!state.busy;
    updateBtn.textContent = state.busy ? uk.updating : uk.updateIndex;
  }

  const cancelBtn = host.querySelector('[data-action="cancel"]');
  if (cancelBtn) cancelBtn.hidden = !state.busy;

  host.querySelectorAll('[data-action="catalog"], [data-action="export-backup"], [data-action="import-backup"]')
    .forEach((btn) => {
      btn.disabled = !!state.busy;
    });

  const track = host.querySelector('.progress-track');
  const bar = host.querySelector('.progress-bar');
  // Show hairline during names-pass busy or background enrich progress (SS-36b).
  if (track) track.hidden = !(state.busy || state.progress);
  if (bar) bar.style.width = `${pct}%`;

  const status = host.querySelector('.status-text');
  if (status) status.textContent = state.status || uk.statusIdle;
}

function esc(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}
