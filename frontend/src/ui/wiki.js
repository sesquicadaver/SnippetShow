import uk from '../i18n/uk.js';

/**
 * Center Wiki panel: man table / install hint / raw man modal.
 * @param {HTMLElement} host
 * @param {{
 *   detail: any|null,
 *   loading: boolean,
 *   flagFilter: string,
 *   rawOpen: boolean,
 *   rawText: string,
 *   rawLoading: boolean,
 *   onFlagFilter: (q:string)=>void,
 *   onOpenRaw: ()=>void,
 *   onCloseRaw: ()=>void,
 * }} state
 */
export function renderWiki(host, state) {
  const detail = state.detail;
  if (!detail && !state.loading) {
    host.innerHTML = `
      <section class="wiki-panel">
        <div class="wiki-empty">${esc(uk.wikiEmpty)}</div>
        <aside class="wiki-side-note">
          <h3>${esc(uk.snippetsPlaceholder)}</h3>
        </aside>
      </section>
    `;
    return;
  }

  if (state.loading || !detail) {
    host.innerHTML = `
      <section class="wiki-panel">
        <div class="wiki-empty">${esc(uk.wikiLoading)}</div>
      </section>
    `;
    return;
  }

  const p = detail.program;
  if (!p.isInstalled) {
    host.innerHTML = `
      <section class="wiki-panel">
        <header class="wiki-header">
          <h2>${esc(p.name)}</h2>
          <p class="wiki-sub">${esc(uk.installTitle)}</p>
        </header>
        <div class="install-block">
          <h3>${esc(uk.installSummary)}</h3>
          <p>${esc(p.summary || '—')}</p>
          <h3>${esc(uk.installCommand)}</h3>
          <pre class="install-cmd">${esc(detail.installCommand || '—')}</pre>
          <p class="muted">${esc(detail.cacheHint || uk.installHint)}</p>
        </div>
        <aside class="wiki-side-note">
          <h3>${esc(uk.snippetsPlaceholder)}</h3>
        </aside>
      </section>
    `;
    return;
  }

  const filter = (state.flagFilter || '').trim().toLowerCase();
  const cmds = (detail.commands || []).filter((c) => {
    if (!filter) return true;
    return `${c.flag} ${c.description}`.toLowerCase().includes(filter);
  });

  host.innerHTML = `
    <section class="wiki-panel">
      <header class="wiki-header">
        <div>
          <h2>${esc(p.name)}</h2>
          <p class="wiki-sub">${esc(p.summary || '')}${p.version ? ` · ${esc(p.version)}` : ''}</p>
        </div>
        <div class="wiki-tools">
          <input type="search" class="flag-filter" placeholder="${esc(uk.wikiFilterPlaceholder)}"
            value="${esc(state.flagFilter || '')}" autocomplete="off"/>
          <button type="button" class="btn" data-action="raw">${esc(uk.rawMan)}</button>
        </div>
      </header>
      <h3 class="wiki-section-title">${esc(uk.wikiTitle)}</h3>
      <div class="man-table-wrap">
        <table class="man-table">
          <thead>
            <tr>
              <th class="col-check">${esc(uk.colCheck)}</th>
              <th>${esc(uk.colFlag)}</th>
              <th>${esc(uk.colDesc)}</th>
              <th>${esc(uk.colActions)}</th>
            </tr>
          </thead>
          <tbody>
            ${cmds.length === 0 ? `
              <tr><td colspan="4" class="muted">${esc(uk.wikiNoFlags)}</td></tr>
            ` : cmds.map((c) => `
              <tr>
                <td class="col-check"><input type="checkbox" disabled title="${esc(uk.builderPlaceholder)}"/></td>
                <td class="flag"><code>${esc(c.flag)}</code></td>
                <td>${esc(c.description)}</td>
                <td class="actions"><span class="action-stub">${esc(uk.actionStub)}</span></td>
              </tr>
            `).join('')}
          </tbody>
        </table>
      </div>
      <aside class="wiki-side-note">
        <h3>${esc(uk.snippetsPlaceholder)}</h3>
      </aside>
      ${state.rawOpen ? renderRawModal(state) : ''}
    </section>
  `;

  host.querySelector('.flag-filter')?.addEventListener('input', (e) => {
    state.onFlagFilter(e.target.value);
  });
  host.querySelector('[data-action="raw"]')?.addEventListener('click', () => state.onOpenRaw());
  host.querySelector('[data-action="close-raw"]')?.addEventListener('click', () => state.onCloseRaw());
  host.querySelector('.raw-backdrop')?.addEventListener('click', () => state.onCloseRaw());
}

function renderRawModal(state) {
  return `
    <div class="raw-modal" role="dialog" aria-modal="true" aria-label="${esc(uk.rawManTitle)}">
      <div class="raw-backdrop"></div>
      <div class="raw-sheet">
        <header>
          <h3>${esc(uk.rawManTitle)}</h3>
          <button type="button" class="btn" data-action="close-raw">${esc(uk.rawManClose)}</button>
        </header>
        <pre class="raw-body">${state.rawLoading ? esc(uk.rawManLoading) : esc(state.rawText || uk.rawManUnavailable)}</pre>
      </div>
    </div>
  `;
}

function esc(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}
