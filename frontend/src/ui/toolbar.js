import uk from '../i18n/uk.js';

const LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('');

/**
 * Builds the Wiki toolbar: mode radios, search, alphabet, update controls.
 * @param {HTMLElement} host
 * @param {{
 *   mode: string,
 *   letter: string,
 *   query: string,
 *   busy: boolean,
 *   status: string,
 *   progress: {done:number,total:number}|null,
 *   onMode: (mode:string)=>void,
 *   onLetter: (letter:string)=>void,
 *   onQuery: (query:string)=>void,
 *   onUpdate: ()=>void,
 *   onSyncCatalog: ()=>void,
 * }} state
 */
export function renderToolbar(host, state) {
  const progress = state.progress;
  const pct = progress && progress.total > 0
    ? Math.min(100, Math.round((100 * progress.done) / progress.total))
    : state.busy ? 15 : 0;

  host.innerHTML = `
    <div class="toolbar" role="toolbar" aria-label="Керування Wiki">
      <div class="toolbar-brand">
        <span class="toolbar-brand-name">${esc(uk.brand)}</span>
        <span class="toolbar-brand-sub">${esc(uk.subtitle)}</span>
      </div>
      <div class="toolbar-modes" role="radiogroup" aria-label="Режим списку">
        <label class="mode ${state.mode === 'installed' ? 'is-active' : ''}">
          <input type="radio" name="mode" value="installed" ${state.mode === 'installed' ? 'checked' : ''}/>
          ${esc(uk.modeInstalled)}
        </label>
        <label class="mode ${state.mode === 'all' ? 'is-active' : ''}">
          <input type="radio" name="mode" value="all" ${state.mode === 'all' ? 'checked' : ''}/>
          ${esc(uk.modeAll)}
        </label>
      </div>
      <div class="toolbar-search">
        <input type="search" class="search-input" placeholder="${esc(uk.searchPlaceholder)}"
          value="${esc(state.query)}" autocomplete="off" spellcheck="false"/>
      </div>
      <div class="toolbar-actions">
        <button type="button" class="btn btn-primary" data-action="update" ${state.busy ? 'disabled' : ''}>
          ${esc(state.busy ? uk.updating : uk.updateIndex)}
        </button>
        <button type="button" class="btn" data-action="catalog" ${state.busy ? 'disabled' : ''} title="${esc(uk.modeAll)}">
          ${esc(uk.syncCatalog)}
        </button>
      </div>
      <div class="toolbar-status" aria-live="polite">
        <div class="progress-track" ${state.busy ? '' : 'hidden'}>
          <div class="progress-bar" style="width:${pct}%"></div>
        </div>
        <span class="status-text">${esc(state.status || uk.statusIdle)}</span>
      </div>
      <div class="alphabet" role="group" aria-label="Алфавітний фільтр">
        <button type="button" class="alpha ${state.letter === 'all' ? 'is-active' : ''}" data-letter="all">${esc(uk.alphabetAll)}</button>
        ${LETTERS.map((L) => `
          <button type="button" class="alpha ${state.letter.toUpperCase() === L ? 'is-active' : ''}" data-letter="${L}">${L}</button>
        `).join('')}
        <button type="button" class="alpha ${state.letter === '0-9' ? 'is-active' : ''}" data-letter="0-9">${esc(uk.alphabetDigits)}</button>
      </div>
    </div>
  `;

  host.querySelectorAll('input[name="mode"]').forEach((el) => {
    el.addEventListener('change', () => state.onMode(el.value));
  });
  const search = host.querySelector('.search-input');
  search.addEventListener('input', () => state.onQuery(search.value));
  host.querySelector('[data-action="update"]').addEventListener('click', () => state.onUpdate());
  host.querySelector('[data-action="catalog"]').addEventListener('click', () => state.onSyncCatalog());
  host.querySelectorAll('[data-letter]').forEach((btn) => {
    btn.addEventListener('click', () => state.onLetter(btn.getAttribute('data-letter')));
  });
}

function esc(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}
