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
 * Builds / updates the Wiki toolbar: mode radios, search, letter select, update controls.
 * First call mounts DOM; later calls patch in place so the search input keeps focus (SS-30).
 * Brand = SnippetShow only (no subtitle). Letter filter = compact `<select>` (SS-36d).
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
  const letterOpts = letterFilterOptions()
    .map((o) => `<option value="${esc(o.value)}">${esc(o.label)}</option>`)
    .join('');

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
        <select class="category-select" aria-label="${esc(uk.filterCategoryAll)}"></select>
        <label class="noman-filter">
          <input type="checkbox" class="noman-input"/>
          ${esc(uk.filterNoMan)}
        </label>
        <select class="letter-select" aria-label="${esc(uk.alphabetFilter || 'Літера')}">
          ${letterOpts}
        </select>
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
  host.querySelector('.category-select')?.addEventListener('change', (e) => {
    host._toolbarState?.onCategory?.(e.target.value);
  });
  host.querySelector('.noman-input')?.addEventListener('change', (e) => {
    host._toolbarState?.onNoMan?.(!!e.target.checked);
  });
  host.querySelector('.letter-select')?.addEventListener('change', (e) => {
    host._toolbarState?.onLetter?.(e.target.value);
  });
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

  const catSel = host.querySelector('.category-select');
  if (catSel) {
    const cats = state.categories || [];
    const cur = state.category || '';
    const opts = [`<option value="">${esc(uk.filterCategoryAll)}</option>`]
      .concat(cats.map((c) => `<option value="${esc(c)}" ${c === cur ? 'selected' : ''}>${esc(c)}</option>`));
    catSel.innerHTML = opts.join('');
    catSel.value = cur;
    catSel.hidden = state.mode === 'all';
  }
  const noMan = host.querySelector('.noman-input');
  if (noMan) {
    noMan.checked = !!state.noMan;
    noMan.closest('.noman-filter').hidden = state.mode === 'all';
  }

  const letterSel = host.querySelector('.letter-select');
  if (letterSel && document.activeElement !== letterSel) {
    letterSel.value = normalizeLetterValue(state.letter);
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
