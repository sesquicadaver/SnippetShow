import uk from '../i18n/uk.js';

const ROW_H = 36;
const OVERSCAN = 12;

/**
 * Visible index range for the virtualized navigator window.
 * Exported for unit tests (SS-23).
 * @param {number} scrollTop
 * @param {number} viewH
 * @param {number} count
 * @param {number} [rowH]
 * @param {number} [overscan]
 * @returns {{start:number,end:number}}
 */
export function windowRange(scrollTop, viewH, count, rowH = ROW_H, overscan = OVERSCAN) {
  const safeCount = Math.max(0, count | 0);
  const safeView = Math.max(1, viewH || 400);
  const start = Math.max(0, Math.floor((scrollTop || 0) / rowH) - overscan);
  const visible = Math.ceil(safeView / rowH) + overscan * 2;
  const end = Math.min(safeCount, start + visible);
  return { start, end };
}

/**
 * Always prefer the latest state stored on the host.
 * Scroll handlers must not close over the first renderNavigator state (SS-23).
 * @param {HTMLElement|{_navState?: object}} host
 */
export function resolveNavState(host) {
  return host?._navState ?? null;
}

/**
 * Virtualized left-panel program list.
 * @param {HTMLElement} host
 * @param {{
 *   programs: Array<{id:number,name:string,summary:string,isInstalled:boolean}>,
 *   selected: string|null,
 *   mode: string,
 *   loading: boolean,
 *   onSelect: (name:string)=>void,
 * }} state
 */
export function renderNavigator(host, state) {
  const programs = state.programs || [];
  const emptyHint = state.mode === 'all' ? uk.emptyAllHint : uk.emptyInstalledHint;

  // Keep latest state before any early return so scroll repaints stay current (SS-23).
  host._navState = state;

  if (!host.dataset.ready) {
    host.innerHTML = `
      <div class="nav-panel">
        <div class="nav-head">
          <span class="nav-count"></span>
        </div>
        <div class="nav-table-head" aria-hidden="true">
          <span>${esc(uk.colName)}</span>
          <span>${esc(uk.colSummary)}</span>
          <span>${esc(uk.colStatus)}</span>
        </div>
        <div class="nav-scroll" tabindex="0">
          <div class="nav-spacer"></div>
          <div class="nav-window"></div>
        </div>
        <div class="nav-empty" hidden></div>
      </div>
    `;
    host.dataset.ready = '1';
    const scroll = host.querySelector('.nav-scroll');
    // Do not close over `state` — first call is often loading/empty.
    scroll.addEventListener('scroll', () => paintWindow(host), { passive: true });
  }

  const countEl = host.querySelector('.nav-count');
  const emptyEl = host.querySelector('.nav-empty');
  const scroll = host.querySelector('.nav-scroll');
  countEl.textContent = fmt(uk.countPrograms, { n: programs.length });

  if (state.loading) {
    emptyEl.hidden = false;
    emptyEl.textContent = uk.statusLoadingList;
    scroll.hidden = true;
    return;
  }

  if (programs.length === 0) {
    emptyEl.hidden = false;
    emptyEl.innerHTML = `<strong>${esc(uk.emptyList)}</strong><p>${esc(emptyHint)}</p>`;
    scroll.hidden = true;
    return;
  }

  emptyEl.hidden = true;
  scroll.hidden = false;
  paintWindow(host);
}

function paintWindow(host) {
  const st = resolveNavState(host);
  if (!st) return;
  const programs = st.programs || [];
  const scroll = host.querySelector('.nav-scroll');
  const spacer = host.querySelector('.nav-spacer');
  const win = host.querySelector('.nav-window');
  if (!scroll || !spacer || !win) return;

  spacer.style.height = `${programs.length * ROW_H}px`;
  const viewH = scroll.clientHeight || 400;
  const { start, end } = windowRange(scroll.scrollTop, viewH, programs.length);

  win.style.transform = `translateY(${start * ROW_H}px)`;
  const rows = [];
  for (let i = start; i < end; i += 1) {
    const p = programs[i];
    const selected = st.selected === p.name;
    const installed = !!p.isInstalled;
    rows.push(`
      <button type="button" class="nav-row ${selected ? 'is-selected' : ''} ${installed ? 'is-installed' : 'is-missing'}"
        data-name="${esc(p.name)}" style="height:${ROW_H}px">
        <span class="nav-name" title="${esc(p.name)}">${esc(p.name)}</span>
        <span class="nav-summary" title="${esc(p.summary || '')}">${esc(p.summary || '—')}</span>
        <span class="nav-status">${esc(installed ? uk.statusInstalled : uk.statusNotInstalled)}</span>
      </button>
    `);
  }
  win.innerHTML = rows.join('');
  win.querySelectorAll('[data-name]').forEach((btn) => {
    btn.addEventListener('click', () => st.onSelect(btn.getAttribute('data-name')));
  });
}

function fmt(template, vars) {
  return template.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ''));
}

function esc(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}
