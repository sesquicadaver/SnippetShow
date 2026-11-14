import uk from '../i18n/uk.js';

/**
 * Coalesce highlight work onto one animation frame (SS-30).
 * @param {(cb:()=>void)=>unknown} schedule requestAnimationFrame-like
 * @param {(text:string)=>void} apply
 * @returns {(text:string)=>void}
 */
export function createHighlightScheduler(schedule, apply) {
  let pending = null;
  let token = null;
  return (text) => {
    pending = text;
    if (token != null) return;
    token = schedule(() => {
      token = null;
      const value = pending;
      pending = null;
      apply(value);
    });
  };
}

/**
 * Right Script Builder panel: draft lines, reorder, bash editor, save/export/run.
 * Mounts once; later paints patch in place and keep the same textarea node (SS-30).
 * @param {HTMLElement} host
 * @param {object} state
 */
export function renderBuilder(host, state) {
  host._builderState = state;
  const draft = state.draft || { lines: [], count: 0, body: '' };
  const body = state.editorBody != null ? state.editorBody : (draft.body || '');
  const open = state.builderOpen !== false;

  if (!host.dataset.ready) {
    mountBuilderShell(host);
    host.dataset.ready = '1';
    bindBuilderChrome(host);
  }

  const panel = host.querySelector('.builder-panel');
  panel?.classList.toggle('is-open', open);
  panel?.classList.toggle('is-collapsed', !open);

  const toggle = host.querySelector('[data-action="toggle-builder"]');
  if (toggle) {
    toggle.title = open ? uk.builderCollapse : uk.builderExpand;
    toggle.textContent = open ? uk.builderCollapse : uk.builderExpand;
  }

  const countEl = host.querySelector('.builder-sub');
  if (countEl) {
    const n = draft.count || (draft.lines || []).length || 0;
    countEl.textContent = fmt(uk.builderDraftCount, { n });
  }

  const bodyEl = host.querySelector('.builder-body');
  if (!bodyEl) return;

  if (!open) {
    bodyEl.hidden = true;
    bodyEl.innerHTML = '';
    host._builderOpenMounted = false;
    return;
  }

  bodyEl.hidden = false;

  if (!host._builderOpenMounted) {
    bodyEl.innerHTML = openBuilderBodyHTML();
    bindBuilderBody(host);
    host._builderOpenMounted = true;
    host._hlSchedule = createHighlightScheduler(
      (cb) => requestAnimationFrame(cb),
      (text) => {
        const hl = host.querySelector('.bash-highlight');
        if (hl) hl.innerHTML = highlightBash(text);
      },
    );
  }

  patchBuilderBody(host, state, body);
}

function mountBuilderShell(host) {
  host.innerHTML = `
    <section class="builder-panel is-open">
      <header class="builder-header">
        <div>
          <h2>${esc(uk.builderTitle)}</h2>
          <p class="builder-sub"></p>
        </div>
        <button type="button" class="btn btn-tiny" data-action="toggle-builder"></button>
      </header>
      <div class="builder-body"></div>
    </section>
  `;
}

function openBuilderBodyHTML() {
  return `
    <div class="builder-lines">
      <h3>${esc(uk.builderLines)}</h3>
      <div class="builder-lines-content"></div>
    </div>
    <div class="builder-editor-wrap">
      <h3>${esc(uk.builderEditor)}</h3>
      <p class="muted builder-editor-hint">${esc(uk.builderEditorHint || uk.builderHint)}</p>
      <div class="bash-editor">
        <pre class="bash-highlight" aria-hidden="true"></pre>
        <textarea class="bash-input" spellcheck="false" autocomplete="off"></textarea>
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
      <div class="builder-saved-content"></div>
    </div>
  `;
}

function bindBuilderChrome(host) {
  host.querySelector('[data-action="toggle-builder"]')?.addEventListener('click', () => {
    host._builderState?.onToggle?.();
  });
}

function bindBuilderBody(host) {
  const ta = host.querySelector('.bash-input');
  ta?.addEventListener('input', () => {
    host._hlSchedule?.(ta.value);
    host._builderState?.onEditorInput?.(ta.value);
  });
  ta?.addEventListener('scroll', () => {
    const hl = host.querySelector('.bash-highlight');
    if (hl) {
      hl.scrollTop = ta.scrollTop;
      hl.scrollLeft = ta.scrollLeft;
    }
  });
  // Body is source of truth on blur: backend ApplyBody keeps blank lines (H3).
  // Skip commit when focus moves to builder actions — mousedown preventDefault keeps
  // focus in the textarea so Run/Export read the latest body (SS-21).
  ta?.addEventListener('blur', (e) => {
    const next = e.relatedTarget;
    if (next && host.querySelector('.builder-actions')?.contains(next)) {
      return;
    }
    host._builderState?.onEditorCommit?.(ta.value);
  });

  // Keep textarea focus when pressing action buttons so blur→repaint cannot
  // destroy the button before click or drop in-progress editor edits (SS-21).
  host.querySelector('.builder-actions')?.addEventListener('mousedown', (e) => {
    if (e.button === 0 && e.target.closest('button[data-action]')) {
      e.preventDefault();
    }
  });

  const readBody = () => {
    const input = host.querySelector('.bash-input');
    return input ? input.value : '';
  };

  host.querySelector('[data-action="script-copy"]')?.addEventListener('click', () => {
    host._builderState?.onCopy?.(readBody());
  });
  host.querySelector('[data-action="script-run"]')?.addEventListener('click', () => {
    host._builderState?.onRun?.(readBody());
  });
  host.querySelector('[data-action="script-export"]')?.addEventListener('click', () => {
    host._builderState?.onExport?.(readBody());
  });
  host.querySelector('[data-action="script-save"]')?.addEventListener('click', () => {
    host._builderState?.onSave?.(readBody());
  });
  host.querySelector('[data-action="script-clear"]')?.addEventListener('click', () => {
    host._builderState?.onClear?.();
  });
}

function patchBuilderBody(host, state, body) {
  const draft = state.draft || { lines: [], count: 0, body: '' };
  const lines = draft.lines || [];
  const scripts = state.savedScripts || [];

  const linesContent = host.querySelector('.builder-lines-content');
  if (linesContent) {
    linesContent.innerHTML = lines.length === 0
      ? `<p class="muted">${esc(uk.builderEmpty)}</p>`
      : `
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
      `;
    bindLineEvents(host, linesContent);
  }

  const ta = host.querySelector('.bash-input');
  const hl = host.querySelector('.bash-highlight');
  const focused = ta && typeof document !== 'undefined' && document.activeElement === ta;
  if (ta && !focused && ta.value !== body) {
    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    ta.value = body;
    // Best-effort caret restore when programmatically syncing while unfocused.
    if (typeof start === 'number' && typeof end === 'number') {
      try {
        ta.setSelectionRange(start, end);
      } catch {
        // ignore
      }
    }
  }
  if (hl && !focused) {
    hl.innerHTML = highlightBash(ta ? ta.value : body);
  } else if (hl && focused && ta) {
    host._hlSchedule?.(ta.value);
  }

  const savedContent = host.querySelector('.builder-saved-content');
  if (savedContent) {
    savedContent.innerHTML = scripts.length === 0
      ? `<p class="muted">${esc(uk.builderSavedEmpty)}</p>`
      : `
        <ul class="saved-list">
          ${scripts.map((s) => `
            <li>
              <button type="button" class="btn btn-link" data-action="script-load" data-id="${s.id}">${esc(s.name)}</button>
              <button type="button" class="btn btn-tiny btn-danger" data-action="script-delete" data-id="${s.id}">${esc(uk.builderDeleteSaved)}</button>
            </li>
          `).join('')}
        </ul>
      `;
    savedContent.querySelectorAll('[data-action="script-load"]').forEach((btn) => {
      btn.addEventListener('click', () => host._builderState?.onLoad?.(Number(btn.dataset.id)));
    });
    savedContent.querySelectorAll('[data-action="script-delete"]').forEach((btn) => {
      btn.addEventListener('click', () => host._builderState?.onDeleteSaved?.(Number(btn.dataset.id)));
    });
  }
}

function bindLineEvents(host, linesContent) {
  linesContent.querySelectorAll('[data-action="line-up"]').forEach((btn) => {
    btn.addEventListener('click', () => host._builderState?.onMove?.(Number(btn.dataset.index), 'up'));
  });
  linesContent.querySelectorAll('[data-action="line-down"]').forEach((btn) => {
    btn.addEventListener('click', () => host._builderState?.onMove?.(Number(btn.dataset.index), 'down'));
  });
  linesContent.querySelectorAll('[data-action="line-remove"]').forEach((btn) => {
    btn.addEventListener('click', () => host._builderState?.onRemove?.(Number(btn.dataset.index)));
  });

  let dragIndex = null;
  linesContent.querySelectorAll('.builder-line').forEach((el) => {
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
      host._builderState?.onReorder?.(dragIndex, to);
    });
  });
}

/** Minimal bash highlight: shebang, comments, strings, common keywords. */
export function highlightBash(src) {
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
