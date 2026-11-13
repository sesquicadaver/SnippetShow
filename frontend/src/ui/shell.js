import uk from '../i18n/uk.js';
import { AppAPI, hasWails, onIndexProgress } from './api.js';
import { renderToolbar } from './toolbar.js';
import { renderNavigator } from './navigator.js';
import { renderWiki } from './wiki.js';

/**
 * Phase 3 Wiki shell: toolbar + navigator + center (man/snippets/actions) + builder draft count.
 * @param {HTMLElement} root
 * @param {{ name?: string, subtitle?: string, phase?: string }} info
 * @param {string} status
 */
export function renderShell(root, info, status) {
  root.innerHTML = `
    <div class="app-frame">
      <header class="app-toolbar" id="toolbar"></header>
      <div class="app-body">
        <aside class="app-nav" id="navigator" aria-label="Список програм"></aside>
        <main class="app-wiki" id="wiki" aria-label="Wiki"></main>
        <aside class="app-builder" id="builder" aria-label="${esc(uk.builderPlaceholder)}">
          <div class="builder-placeholder">
            <h2>${esc(uk.builderPlaceholder)}</h2>
            <p>${esc(uk.builderHint)}</p>
            <p class="builder-draft" id="builder-draft"></p>
            <p class="muted">${esc(status || uk.phaseHint)}</p>
          </div>
        </aside>
      </div>
    </div>
  `;

  const app = new WikiApp(root, info);
  app.mount();
}

class WikiApp {
  /**
   * @param {HTMLElement} root
   * @param {object} info
   */
  constructor(root, info) {
    this.root = root;
    this.info = info;
    this.toolbarEl = root.querySelector('#toolbar');
    this.navEl = root.querySelector('#navigator');
    this.wikiEl = root.querySelector('#wiki');
    this.builderDraftEl = root.querySelector('#builder-draft');

    this.mode = 'installed';
    this.letter = 'all';
    this.query = '';
    this.programs = [];
    this.selected = null;
    this.detail = null;
    this.listLoading = false;
    this.detailLoading = false;
    this.busy = false;
    this.status = uk.statusIdle;
    this.progress = null;
    this.flagFilter = '';
    this.rawOpen = false;
    this.rawText = '';
    this.rawLoading = false;
    this.toast = '';
    this._toastTimer = null;
    this.confirm = null;
    this.snippetForm = { open: false, id: 0, title: '', command: '', tags: '', error: '' };
    this.draftCount = 0;
    this._searchTimer = null;
    this._unsubProgress = null;
    this._reqList = 0;
    this._reqDetail = 0;
  }

  mount() {
    this._unsubProgress = onIndexProgress((p) => {
      this.progress = p || null;
      if (p?.message) this.status = p.message;
      this.paintToolbar();
    });
    this.paintAll();
    this.refreshDraft();
    this.bootstrap();
  }

  async bootstrap() {
    if (!hasWails()) {
      this.status = 'Wails API недоступний (перегляд без бекенду)';
      this.paintToolbar();
      return;
    }
    this.busy = true;
    this.status = uk.statusLoadingList;
    this.paintToolbar();
    try {
      await AppAPI.EnsureIndexed();
    } catch (err) {
      console.error(err);
      this.status = `${uk.statusError}: ${err?.message || err}`;
    }
    this.busy = false;
    await this.reloadList();
  }

  paintAll() {
    this.paintToolbar();
    this.paintNavigator();
    this.paintWiki();
    this.paintBuilder();
  }

  paintToolbar() {
    renderToolbar(this.toolbarEl, {
      mode: this.mode,
      letter: this.letter,
      query: this.query,
      busy: this.busy,
      status: this.status,
      progress: this.progress,
      onMode: (mode) => {
        this.mode = mode;
        this.reloadList();
      },
      onLetter: (letter) => {
        this.letter = letter;
        this.reloadList();
      },
      onQuery: (query) => {
        this.query = query;
        clearTimeout(this._searchTimer);
        this._searchTimer = setTimeout(() => this.reloadList(), 180);
        this.paintToolbar();
      },
      onUpdate: () => this.runUpdate(),
      onSyncCatalog: () => this.runCatalog(),
    });
  }

  paintNavigator() {
    renderNavigator(this.navEl, {
      programs: this.programs,
      selected: this.selected,
      mode: this.mode,
      loading: this.listLoading,
      onSelect: (name) => this.selectProgram(name),
    });
  }

  paintWiki() {
    renderWiki(this.wikiEl, {
      detail: this.detail,
      loading: this.detailLoading,
      flagFilter: this.flagFilter,
      rawOpen: this.rawOpen,
      rawText: this.rawText,
      rawLoading: this.rawLoading,
      toast: this.toast,
      confirm: this.confirm,
      snippetForm: this.snippetForm,
      onFlagFilter: (q) => {
        this.flagFilter = q;
        this.paintWiki();
      },
      onOpenRaw: () => this.openRaw(),
      onCloseRaw: () => {
        this.rawOpen = false;
        this.paintWiki();
      },
      onCopy: (cmd) => this.copyText(cmd),
      onRun: (cmd) => this.runCommand(cmd, false),
      onAddScript: (cmd) => this.addToScript(cmd),
      onSnippetAdd: () => {
        this.snippetForm = { open: true, id: 0, title: '', command: '', tags: '', error: '' };
        this.paintWiki();
      },
      onSnippetCancel: () => {
        this.snippetForm = { open: false, id: 0, title: '', command: '', tags: '', error: '' };
        this.paintWiki();
      },
      onSnippetEdit: (id) => this.openSnippetEdit(id),
      onSnippetDelete: (id) => this.deleteSnippet(id),
      onSnippetSave: (fields) => this.saveSnippet(fields),
      onConfirmRun: () => {
        const cmd = this.confirm?.command || '';
        this.confirm = null;
        this.runCommand(cmd, true);
      },
      onConfirmCancel: () => {
        this.confirm = null;
        this.paintWiki();
      },
    });
  }

  paintBuilder() {
    if (!this.builderDraftEl) return;
    this.builderDraftEl.textContent = fmt(uk.builderDraftCount, { n: this.draftCount });
  }

  listQuery() {
    return {
      mode: this.mode,
      letter: this.letter,
      query: this.query,
    };
  }

  async reloadList() {
    if (!hasWails()) return;
    const req = ++this._reqList;
    this.listLoading = true;
    this.paintNavigator();
    try {
      const rows = await AppAPI.ListPrograms(this.listQuery());
      if (req !== this._reqList) return;
      this.programs = Array.isArray(rows) ? rows : [];
      this.status = fmt(uk.countPrograms, { n: this.programs.length });
    } catch (err) {
      console.error(err);
      if (req !== this._reqList) return;
      this.programs = [];
      this.status = `${uk.statusError}: ${err?.message || err}`;
    }
    this.listLoading = false;
    this.paintToolbar();
    this.paintNavigator();
  }

  async selectProgram(name) {
    this.selected = name;
    this.flagFilter = '';
    this.rawOpen = false;
    this.rawText = '';
    this.snippetForm = { open: false, id: 0, title: '', command: '', tags: '', error: '' };
    this.confirm = null;
    this.detailLoading = true;
    this.paintNavigator();
    this.paintWiki();
    const req = ++this._reqDetail;
    try {
      const detail = await AppAPI.GetProgram(name, true);
      if (req !== this._reqDetail) return;
      this.detail = detail;
      this.status = fmt(uk.selected, { name });
    } catch (err) {
      console.error(err);
      if (req !== this._reqDetail) return;
      this.detail = null;
      this.status = `${uk.statusError}: ${err?.message || err}`;
    }
    this.detailLoading = false;
    this.paintToolbar();
    this.paintWiki();
    this.reloadList();
  }

  async openRaw() {
    if (!this.selected) return;
    this.rawOpen = true;
    this.rawLoading = true;
    this.paintWiki();
    try {
      this.rawText = await AppAPI.GetRawMan(this.selected);
    } catch (err) {
      this.rawText = err?.message || uk.rawManUnavailable;
    }
    this.rawLoading = false;
    this.paintWiki();
  }

  showToast(msg) {
    this.toast = msg;
    this.paintWiki();
    clearTimeout(this._toastTimer);
    this._toastTimer = setTimeout(() => {
      this.toast = '';
      this.paintWiki();
    }, 1800);
  }

  async copyText(cmd) {
    if (!cmd || !hasWails()) return;
    try {
      await AppAPI.CopyText(cmd);
      this.showToast(uk.toastCopied);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async runCommand(cmd, confirmed) {
    if (!cmd || !hasWails()) return;
    try {
      const res = await AppAPI.RunCommand({ command: cmd, confirmed: !!confirmed });
      if (res?.needsConfirm) {
        this.confirm = { command: cmd, reasons: res.reasons || [] };
        this.paintWiki();
        return;
      }
      if (res?.started) {
        this.showToast(uk.toastRunStarted);
        this.status = uk.toastRunStarted;
        this.paintToolbar();
      }
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async addToScript(cmd) {
    if (!cmd || !hasWails()) return;
    try {
      const draft = await AppAPI.AddToScript(cmd);
      this.draftCount = draft?.count || 0;
      this.paintBuilder();
      this.showToast(fmt(uk.toastScriptAdded, { n: this.draftCount }));
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async refreshDraft() {
    if (!hasWails()) return;
    try {
      const draft = await AppAPI.ScriptDraft();
      this.draftCount = draft?.count || 0;
      this.paintBuilder();
    } catch {
      // ignore
    }
  }

  openSnippetEdit(id) {
    const sn = (this.detail?.snippets || []).find((s) => s.id === id);
    if (!sn) return;
    this.snippetForm = {
      open: true,
      id: sn.id,
      title: sn.title || '',
      command: sn.command || '',
      tags: sn.tags || '',
      error: '',
    };
    this.paintWiki();
  }

  async saveSnippet(fields) {
    if (!this.detail?.program?.id || !hasWails()) return;
    const title = (fields.title || '').trim();
    const command = (fields.command || '').trim();
    const tags = (fields.tags || '').trim();
    if (!title || !command) {
      this.snippetForm = { ...this.snippetForm, open: true, error: uk.snippetValidation };
      this.paintWiki();
      return;
    }
    try {
      if (this.snippetForm.id) {
        await AppAPI.UpdateSnippet(this.snippetForm.id, {
          programId: this.detail.program.id,
          title,
          command,
          tags,
        });
      } else {
        await AppAPI.CreateSnippet({
          programId: this.detail.program.id,
          title,
          command,
          tags,
        });
      }
      this.snippetForm = { open: false, id: 0, title: '', command: '', tags: '', error: '' };
      await this.reloadDetail();
      this.showToast(uk.toastSnippetSaved);
    } catch (err) {
      this.snippetForm = { ...this.snippetForm, open: true, error: err?.message || String(err) };
      this.paintWiki();
    }
  }

  async deleteSnippet(id) {
    if (!hasWails()) return;
    if (!window.confirm(uk.snippetDeleteConfirm)) return;
    try {
      await AppAPI.DeleteSnippet(id);
      await this.reloadDetail();
      this.showToast(uk.toastSnippetDeleted);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async reloadDetail() {
    if (!this.selected || !hasWails()) return;
    try {
      this.detail = await AppAPI.GetProgram(this.selected, false);
      this.paintWiki();
    } catch (err) {
      this.status = `${uk.statusError}: ${err?.message || err}`;
      this.paintToolbar();
    }
  }

  async runUpdate() {
    if (this.busy || !hasWails()) return;
    this.busy = true;
    this.progress = { done: 0, total: 1 };
    this.status = uk.updating;
    this.paintToolbar();
    try {
      const res = await AppAPI.UpdateIndex();
      const n = (res?.added || 0) + (res?.updated || 0) + (res?.markedGone || 0);
      this.status = fmt(uk.statusUpdated, { n });
    } catch (err) {
      console.error(err);
      this.status = `${uk.statusError}: ${err?.message || err}`;
    }
    this.busy = false;
    this.progress = null;
    this.paintToolbar();
    await this.reloadList();
  }

  async runCatalog() {
    if (this.busy || !hasWails()) return;
    this.busy = true;
    this.progress = { done: 0, total: 1 };
    this.status = uk.updating;
    this.mode = 'all';
    this.paintToolbar();
    try {
      const res = await AppAPI.SyncRepoCatalog();
      if (res?.status === 'cache_empty') {
        this.status = res.hint || uk.statusCacheEmpty;
      } else {
        this.status = fmt(uk.statusCatalog, { n: res?.added || 0 });
      }
    } catch (err) {
      console.error(err);
      this.status = `${uk.statusError}: ${err?.message || err}`;
    }
    this.busy = false;
    this.progress = null;
    this.paintToolbar();
    await this.reloadList();
  }
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
