import uk from '../i18n/uk.js';

/**
 * Right Script Builder panel: draft lines, reorder, bash editor, save/export/run.
 * @param {HTMLElement} host
 * @param {object} state
 */
export function renderBuilder(host, state) {
  const draft = state.draft || { lines: [], count: 0, body: '' };
  const lines = draft.lines || [];
  const body = state.editorBody != null ? state.editorBody : (draft.body || '');
  const scripts = state.savedScripts || [];
  const open = state.builderOpen !== false;

  host.innerHTML = `
    <section class="builder-panel ${open ? 'is-open' : 'is-collapsed'}">
      <header class="builder-header">
        <div>
          <h2>${esc(uk.builderTitle)}</h2>
          <p class="builder-sub">${esc(fmt(uk.builderDraftCount, { n: draft.count || lines.length || 0 }))}</p>
        </div>
        <button type="button" class="btn btn-tiny" data-action="toggle-builder"
          title="${esc(open ? uk.builderCollapse : uk.builderExpand)}">
          ${esc(open ? uk.builderCollapse : uk.builderExpand)}
        </button>
      </header>
      ${open ? `
        <div class="builder-lines">
          <h3>${esc(uk.builderLines)}</h3>
          ${lines.length === 0 ? `<p class="muted">${esc(uk.builderEmpty)}</p>` : `
            <ul class="builder-line-list">
              ${lines.map((line, i) => `
                <li class="builder-line" draggable="true" data-index="${i}">
                  <code>${esc(line)}</code>
                  <div class="builder-line-actions">
                    <button type="button" class="btn btn-tiny" data-action="line-up" data-index="${i}" ${i === 0 ? 'disabled' : ''}>${esc(uk.builderUp)}</button>
                    <button type="button" class="btn btn-tiny" data-action="line-down" data-index="${i}" ${i >= lines.length - 1 ? 'disabled' : ''}>${esc(uk.builderDown)}</button>
                    <button type="button" class="btn btn-tiny btn-danger" data-action="line-remove" data-index="${i}">${esc(uk.builderRemove)}</button>
                  </div>
                </li>
              `).join('')}
            </ul>
          `}
        </div>
        <div class="builder-editor-wrap">
          <h3>${esc(uk.builderEditor)}</h3>
          <div class="bash-editor">
            <pre class="bash-highlight" aria-hidden="true">${highlightBash(body)}</pre>
            <textarea class="bash-input" spellcheck="false" autocomplete="off">${esc(body)}</textarea>
          </div>
        </div>
        <div class="builder-actions">
          <button type="button" class="btn" data-action="script-copy">${esc(uk.builderCopy)}</button>
          <button type="button" class="btn" data-action="script-run">${esc(uk.builderRun)}</button>
          <button type="button" class="btn" data-action="script-export">${esc(uk.builderExport)}</button>
          <button type="button" class="btn btn-accent" data-action="script-save">${esc(uk.builderSave)}</button>
          <button type="button" class="btn" data-action="script-clear">${esc(uk.builderClear)}</button>
        </div>
        <div class="builder-saved">
          <h3>${esc(uk.builderSaved)}</h3>
          ${scripts.length === 0 ? `<p class="muted">${esc(uk.builderSavedEmpty)}</p>` : `
            <ul class="saved-list">
              ${scripts.map((s) => `
                <li>
                  <button type="button" class="btn btn-link" data-action="script-load" data-id="${s.id}">${esc(s.name)}</button>
                  <button type="button" class="btn btn-tiny btn-danger" data-action="script-delete" data-id="${s.id}">${esc(uk.builderDeleteSaved)}</button>
                </li>
              `).join('')}
            </ul>
          `}
        </div>
      ` : ''}
    </section>
  `;

  bindBuilderEvents(host, state);
}

function bindBuilderEvents(host, state) {
  host.querySelector('[data-action="toggle-builder"]')?.addEventListener('click', () => state.onToggle?.());

  const ta = host.querySelector('.bash-input');
  const hl = host.querySelector('.bash-highlight');
  ta?.addEventListener('input', () => {
    if (hl) hl.innerHTML = highlightBash(ta.value);
    state.onEditorInput?.(ta.value);
  });
  ta?.addEventListener('scroll', () => {
    if (hl) {
      hl.scrollTop = ta.scrollTop;
      hl.scrollLeft = ta.scrollLeft;
    }
  });
  ta?.addEventListener('blur', () => {
    state.onEditorCommit?.(ta.value);
  });

  host.querySelectorAll('[data-action="line-up"]').forEach((btn) => {
    btn.addEventListener('click', () => state.onMove?.(Number(btn.dataset.index), 'up'));
  });
  host.querySelectorAll('[data-action="line-down"]').forEach((btn) => {
    btn.addEventListener('click', () => state.onMove?.(Number(btn.dataset.index), 'down'));
  });
  host.querySelectorAll('[data-action="line-remove"]').forEach((btn) => {
    btn.addEventListener('click', () => state.onRemove?.(Number(btn.dataset.index)));
  });

  host.querySelector('[data-action="script-copy"]')?.addEventListener('click', () => {
    state.onCopy?.(ta?.value || '');
  });
  host.querySelector('[data-action="script-run"]')?.addEventListener('click', () => {
    state.onRun?.(ta?.value || '');
  });
  host.querySelector('[data-action="script-export"]')?.addEventListener('click', () => {
    state.onExport?.(ta?.value || '');
  });
  host.querySelector('[data-action="script-save"]')?.addEventListener('click', () => {
    state.onSave?.(ta?.value || '');
  });
  host.querySelector('[data-action="script-clear"]')?.addEventListener('click', () => state.onClear?.());

  host.querySelectorAll('[data-action="script-load"]').forEach((btn) => {
    btn.addEventListener('click', () => state.onLoad?.(Number(btn.dataset.id)));
  });
  host.querySelectorAll('[data-action="script-delete"]').forEach((btn) => {
    btn.addEventListener('click', () => state.onDeleteSaved?.(Number(btn.dataset.id)));
  });

  // Lightweight DnD reorder (P4-2).
  let dragIndex = null;
  host.querySelectorAll('.builder-line').forEach((el) => {
    el.addEventListener('dragstart', () => {
      dragIndex = Number(el.dataset.index);
      el.classList.add('is-dragging');
    });
    el.addEventListener('dragend', () => {
      el.classList.remove('is-dragging');
      dragIndex = null;
    });
    el.addEventListener('dragover', (e) => {
      e.preventDefault();
    });
    el.addEventListener('drop', (e) => {
      e.preventDefault();
      const to = Number(el.dataset.index);
      if (dragIndex == null || Number.isNaN(to) || dragIndex === to) return;
      state.onReorder?.(dragIndex, to);
    });
  });
}

/** Minimal bash highlight: shebang, comments, strings, common keywords. */
function highlightBash(src) {
  const text = String(src ?? '');
  return text.split('\n').map((line) => {
    if (line.startsWith('#!')) {
      return `<span class="tok-shebang">${esc(line)}</span>`;
    }
    const trimmed = line.trimStart();
    if (trimmed.startsWith('#')) {
      return `<span class="tok-comment">${esc(line)}</span>`;
    }
    let out = esc(line);
    out = out.replace(
      /(&quot;.*?&quot;|&#39;.*?&#39;)/g,
      '<span class="tok-string">$1</span>',
    );
    out = out.replace(
      /\b(if|then|else|fi|for|while|do|done|case|esac|function|return|exit|echo|export|local)\b/g,
      '<span class="tok-kw">$1</span>',
    );
    return out || ' ';
  }).join('\n');
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
