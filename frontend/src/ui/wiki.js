import uk from '../i18n/uk.js';

/**
 * Center Wiki panel: man table, snippets CRUD, Copy/Run/+Script actions.
 * @param {HTMLElement} host
 * @param {object} state
 */
export function renderWiki(host, state) {
  const detail = state.detail;
  if (!detail && !state.loading) {
    host.innerHTML = `
      <section class="wiki-panel">
        <div class="wiki-empty">${esc(uk.wikiEmpty)}</div>
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
          <div class="row-actions">
            ${actionButtons(detail.installCommand || '', 'install')}
          </div>
          <p class="muted">${esc(detail.cacheHint || uk.installHint)}</p>
        </div>
        ${renderSnippetsBlock(detail, state)}
        ${state.confirm ? renderConfirm(state.confirm) : ''}
        ${state.toast ? `<div class="toast" role="status">${esc(state.toast)}</div>` : ''}
      </section>
    `;
    bindWikiEvents(host, state, detail);
    return;
  }

  const filter = (state.flagFilter || '').trim().toLowerCase();
  const draftSet = new Set((state.draftLines || []).map((l) => String(l).trim()).filter(Boolean));
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
      ${detail.parseWarning ? `<p class="form-error" role="status">${esc(fmt(uk.statusParseWarning, { msg: detail.parseWarning }))}</p>` : ''}
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
            ` : cmds.map((c) => {
              const line = commandFromFlag(p.name, c.flag);
              const checked = draftSet.has(line) ? 'checked' : '';
              return `
              <tr>
                <td class="col-check"><input type="checkbox" data-action="check-script" data-cmd="${esc(line)}" ${checked} title="${esc(uk.actionScript)}"/></td>
                <td class="flag"><code>${esc(c.flag)}</code></td>
                <td>${esc(c.description)}</td>
                <td class="actions">${actionButtons(line, `man-${c.id}`)}</td>
              </tr>`;
            }).join('')}
          </tbody>
        </table>
      </div>
      ${renderSnippetsBlock(detail, state)}
      ${state.rawOpen ? renderRawModal(state) : ''}
      ${state.confirm ? renderConfirm(state.confirm) : ''}
      ${state.toast ? `<div class="toast" role="status">${esc(state.toast)}</div>` : ''}
    </section>
  `;

  bindWikiEvents(host, state, detail);
}

function renderSnippetsBlock(detail, state) {
  const snippets = detail.snippets || [];
  const form = state.snippetForm || { open: false, id: 0, title: '', command: '', tags: '' };
  return `
    <aside class="snippets-panel">
      <header class="snippets-header">
        <h3>${esc(uk.snippetsTitle)}</h3>
        <button type="button" class="btn btn-accent" data-action="snippet-add">${esc(uk.snippetAdd)}</button>
      </header>
      ${form.open ? renderSnippetForm(form) : ''}
      <div class="man-table-wrap snippets-table-wrap">
        <table class="man-table snippets-table">
          <thead>
            <tr>
              <th>${esc(uk.snippetColTitle)}</th>
              <th>${esc(uk.snippetColCommand)}</th>
              <th>${esc(uk.snippetColTags)}</th>
              <th>${esc(uk.colActions)}</th>
            </tr>
          </thead>
          <tbody>
            ${snippets.length === 0 ? `
              <tr><td colspan="4" class="muted">${esc(uk.snippetsEmpty)}</td></tr>
            ` : snippets.map((s) => `
              <tr>
                <td>${esc(s.title)}</td>
                <td class="flag"><code>${esc(s.command)}</code></td>
                <td class="muted">${esc(s.tags || '—')}</td>
                <td class="actions">
                  ${actionButtons(s.command, `sn-${s.id}`)}
                  <button type="button" class="btn btn-tiny" data-action="snippet-edit" data-id="${s.id}">${esc(uk.snippetEdit)}</button>
                  <button type="button" class="btn btn-tiny btn-danger" data-action="snippet-delete" data-id="${s.id}">${esc(uk.snippetDelete)}</button>
                </td>
              </tr>
            `).join('')}
          </tbody>
        </table>
      </div>
    </aside>
  `;
}

function renderSnippetForm(form) {
  const isEdit = !!form.id;
  return `
    <form class="snippet-form" data-action="snippet-save">
      <h4>${esc(isEdit ? uk.snippetEditTitle : uk.snippetAddTitle)}</h4>
      <label>${esc(uk.snippetFieldTitle)}
        <input name="title" required value="${esc(form.title || '')}" autocomplete="off"/>
      </label>
      <label>${esc(uk.snippetFieldCommand)}
        <input name="command" required value="${esc(form.command || '')}" autocomplete="off" class="mono"/>
      </label>
      <label>${esc(uk.snippetFieldTags)}
        <input name="tags" value="${esc(form.tags || '')}" autocomplete="off"/>
      </label>
      ${form.error ? `<p class="form-error">${esc(form.error)}</p>` : ''}
      <div class="form-actions">
        <button type="submit" class="btn btn-accent">${esc(uk.snippetSave)}</button>
        <button type="button" class="btn" data-action="snippet-cancel">${esc(uk.snippetCancel)}</button>
      </div>
    </form>
  `;
}

function actionButtons(command, key) {
  const payload = esc(command || '');
  return `
    <div class="action-group" data-cmd="${payload}">
      <button type="button" class="btn btn-tiny" data-action="copy" data-key="${esc(key)}" title="${esc(uk.actionCopy)}">${esc(uk.actionCopy)}</button>
      <button type="button" class="btn btn-tiny" data-action="run" data-key="${esc(key)}" title="${esc(uk.actionRun)}">${esc(uk.actionRun)}</button>
      <button type="button" class="btn btn-tiny" data-action="script" data-key="${esc(key)}" title="${esc(uk.actionScript)}">${esc(uk.actionScript)}</button>
    </div>
  `;
}

/**
 * Man OPTIONS often list alternatives as "-a, --all" or leave a trailing comma
 * ("--audible,"). Shell argv must not keep those commas (SS-20 / 2ping fail).
 * @param {string} flag
 * @returns {string}
 */
export function normalizeManFlag(flag) {
  return String(flag || '')
    .split(/,\s*(?=[-−])/)
    .map((s) => s.trim().replace(/[,，]+$/g, ''))
    .filter(Boolean)
    .join(' ');
}

function commandFromFlag(program, flag) {
  const f = normalizeManFlag(flag);
  if (!f) return program;
  if (f.startsWith(program + ' ') || f === program) return f;
  return `${program} ${f}`;
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

function renderConfirm(confirm) {
  const reasons = (confirm.reasons || []).join(', ');
  return `
    <div class="confirm-modal" role="dialog" aria-modal="true">
      <div class="confirm-backdrop" data-action="confirm-cancel"></div>
      <div class="confirm-sheet">
        <h3>${esc(uk.confirmTitle)}</h3>
        <p>${esc(uk.confirmBody)}</p>
        <p class="confirm-reasons">${esc(uk.confirmReasons)}: <strong>${esc(reasons)}</strong></p>
        <pre class="confirm-cmd">${esc(confirm.command || '')}</pre>
        <div class="form-actions">
          <button type="button" class="btn btn-danger" data-action="confirm-run">${esc(uk.confirmRun)}</button>
          <button type="button" class="btn" data-action="confirm-cancel">${esc(uk.confirmCancel)}</button>
        </div>
      </div>
    </div>
  `;
}

function bindWikiEvents(host, state, detail) {
  host.querySelector('.flag-filter')?.addEventListener('input', (e) => {
    state.onFlagFilter(e.target.value);
  });
  host.querySelector('[data-action="raw"]')?.addEventListener('click', () => state.onOpenRaw());
  host.querySelector('[data-action="close-raw"]')?.addEventListener('click', () => state.onCloseRaw());
  host.querySelector('.raw-backdrop')?.addEventListener('click', () => state.onCloseRaw());

  host.querySelectorAll('[data-action="copy"]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const cmd = btn.closest('[data-cmd]')?.getAttribute('data-cmd') || '';
      state.onCopy?.(cmd);
    });
  });
  host.querySelectorAll('[data-action="run"]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const cmd = btn.closest('[data-cmd]')?.getAttribute('data-cmd') || '';
      state.onRun?.(cmd);
    });
  });
  host.querySelectorAll('[data-action="script"]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const cmd = btn.closest('[data-cmd]')?.getAttribute('data-cmd') || '';
      state.onAddScript?.(cmd);
    });
  });
  host.querySelectorAll('[data-action="check-script"]').forEach((box) => {
    box.addEventListener('change', () => {
      const cmd = box.getAttribute('data-cmd') || '';
      if (box.checked) {
        state.onAddScript?.(cmd);
      } else {
        state.onRemoveScript?.(cmd);
      }
    });
  });

  host.querySelector('[data-action="snippet-add"]')?.addEventListener('click', () => state.onSnippetAdd?.());
  host.querySelector('[data-action="snippet-cancel"]')?.addEventListener('click', () => state.onSnippetCancel?.());
  host.querySelectorAll('[data-action="snippet-edit"]').forEach((btn) => {
    btn.addEventListener('click', () => state.onSnippetEdit?.(Number(btn.dataset.id)));
  });
  host.querySelectorAll('[data-action="snippet-delete"]').forEach((btn) => {
    btn.addEventListener('click', () => state.onSnippetDelete?.(Number(btn.dataset.id)));
  });
  const form = host.querySelector('form.snippet-form');
  form?.addEventListener('submit', (e) => {
    e.preventDefault();
    const fd = new FormData(form);
    state.onSnippetSave?.({
      title: String(fd.get('title') || ''),
      command: String(fd.get('command') || ''),
      tags: String(fd.get('tags') || ''),
    });
  });

  host.querySelector('[data-action="confirm-run"]')?.addEventListener('click', () => state.onConfirmRun?.());
  host.querySelectorAll('[data-action="confirm-cancel"]').forEach((el) => {
    el.addEventListener('click', () => state.onConfirmCancel?.());
  });

  // silence unused
  void detail;
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
