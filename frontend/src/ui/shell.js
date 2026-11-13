import uk from '../i18n/uk.js';
import { AppAPI, hasWails, onIndexProgress } from './api.js';
import { renderToolbar } from './toolbar.js';
import { renderNavigator } from './navigator.js';
import { renderWiki } from './wiki.js';
import { renderBuilder } from './builder.js';

/**
 * Phase 4 Wiki shell: toolbar + navigator + center + Script Builder + backup.
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
        <aside class="app-builder" id="builder" aria-label="${esc(uk.builderTitle)}"></aside>
      </div>
    </div>
  `;

  const app = new WikiApp(root, info, status);
  app.mount();
}

class WikiApp {
  /**
   * @param {HTMLElement} root
   * @param {object} info
   * @param {string} statusHint
   */
  constructor(root, info, statusHint) {
    this.root = root;
    this.info = info;
    this.toolbarEl = root.querySelector('#toolbar');
    this.navEl = root.querySelector('#navigator');
    this.wikiEl = root.querySelector('#wiki');
    this.builderEl = root.querySelector('#builder');

    this.mode = 'installed';
    this.letter = 'all';
    this.query = '';
    this.programs = [];
    this.selected = null;
    this.detail = null;
    this.listLoading = false;
    this.detailLoading = false;
    this.busy = false;
    this.status = statusHint || uk.statusIdle;
    this.progress = null;
    this.flagFilter = '';
    this.rawOpen = false;
    this.rawText = '';
    this.rawLoading = false;
    this.toast = '';
    this._toastTimer = null;
    this.confirm = null;
    this.snippetForm = { open: false, id: 0, title: '', command: '', tags: '', error: '' };
    this.draft = { lines: [], count: 0, body: '' };
    this.editorBody = null;
    this.builderOpen = true;
    this.savedScripts = [];
    this.savedScriptId = 0;
    this._searchTimer = null;
    this._unsubProgress = null;
    this._reqList = 0;
    this._reqDetail = 0;
    this._editorTimer = null;
  }

  mount() {
    this._unsubProgress = onIndexProgress((p) => {
      this.progress = p || null;
      if (p?.message) this.status = p.message;
      this.paintToolbar();
    });
    this.paintAll();
    this.refreshDraft();
    this.refreshSavedScripts();
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
      onCancel: () => this.cancelIndex(),
      onSyncCatalog: () => this.runCatalog(),
      onExportBackup: () => this.exportBackup(),
      onImportBackup: () => this.importBackup(),
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
      draftLines: this.draft?.lines || [],
      onCopy: (cmd) => this.copyText(cmd),
      onRun: (cmd) => this.runCommand(cmd, false),
      onAddScript: (cmd) => this.addToScript(cmd),
      onRemoveScript: (cmd) => this.removeFromScript(cmd),
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
        const kind = this.confirm?.kind || 'command';
        this.confirm = null;
        if (kind === 'script') {
          this.runScript(cmd, true);
        } else {
          this.runCommand(cmd, true);
        }
      },
      onConfirmCancel: () => {
        this.confirm = null;
        this.paintWiki();
      },
    });
  }

  paintBuilder() {
    renderBuilder(this.builderEl, {
      draft: this.draft,
      editorBody: this.editorBody != null ? this.editorBody : this.draft.body,
      builderOpen: this.builderOpen,
      savedScripts: this.savedScripts,
      onToggle: () => {
        this.builderOpen = !this.builderOpen;
        this.root.querySelector('.app-body')?.classList.toggle('builder-collapsed', !this.builderOpen);
        this.paintBuilder();
      },
      onEditorInput: (body) => {
        this.editorBody = body;
      },
      onEditorCommit: (body) => this.commitEditor(body),
      onMove: (index, dir) => this.moveLine(index, dir),
      onRemove: (index) => this.removeLine(index),
      onReorder: (from, to) => this.reorderLines(from, to),
      onCopy: (body) => this.copyText(body || this.draft.body),
      onRun: (body) => this.runScript(body, false),
      onExport: (body) => this.exportScript(body),
      onSave: (body) => this.saveScript(body),
      onClear: () => this.clearDraft(),
      onLoad: (id) => this.loadSavedScript(id),
      onDeleteSaved: (id) => this.deleteSavedScript(id),
    });
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
      this.detailLoading = false;
      if (detail?.parseWarning) {
        const warn = fmt(uk.statusParseWarning, { msg: detail.parseWarning });
        this.status = warn;
        this.showToast(warn);
      } else {
        this.status = fmt(uk.selected, { name });
        this.paintToolbar();
        this.paintWiki();
      }
    } catch (err) {
      console.error(err);
      if (req !== this._reqDetail) return;
      this.detail = null;
      this.detailLoading = false;
      this.status = `${uk.statusError}: ${err?.message || err}`;
      this.paintToolbar();
      this.paintWiki();
    }
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
        this.confirm = { command: cmd, reasons: res.reasons || [], kind: 'command' };
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

  async runScript(body, confirmed) {
    if (!hasWails()) return;
    const command = body || this.draft.body || '';
    try {
      const res = await AppAPI.RunScript({ command, confirmed: !!confirmed });
      if (res?.needsConfirm) {
        this.confirm = { command, reasons: res.reasons || [], kind: 'script' };
        this.paintWiki();
        return;
      }
      if (res?.started) {
        this.showToast(uk.toastRunStarted);
      }
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async addToScript(cmd) {
    if (!cmd || !hasWails()) return;
    try {
      if (this.editorBody != null) {
        await AppAPI.SetScriptBody(this.editorBody);
      }
      const draft = await AppAPI.AddToScript(cmd);
      this.applyDraft(draft);
      this.paintWiki();
      this.showToast(fmt(uk.toastScriptAdded, { n: this.draft.count }));
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async removeFromScript(cmd) {
    if (!cmd || !hasWails()) return;
    try {
      if (this.editorBody != null) {
        await AppAPI.SetScriptBody(this.editorBody);
      }
      const draft = await AppAPI.RemoveFromScript(cmd);
      this.applyDraft(draft);
      this.paintWiki();
      this.showToast(fmt(uk.toastScriptRemoved, { n: this.draft.count }));
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  applyDraft(draft) {
    this.draft = draft || { lines: [], count: 0, body: '' };
    this.editorBody = this.draft.body || '';
    this.paintBuilder();
  }

  async refreshDraft() {
    if (!hasWails()) return;
    try {
      const draft = await AppAPI.ScriptDraft();
      this.applyDraft(draft);
    } catch {
      // ignore
    }
  }

  async refreshSavedScripts() {
    if (!hasWails()) return;
    try {
      const list = await AppAPI.ListCustomScripts();
      this.savedScripts = Array.isArray(list) ? list : [];
      this.paintBuilder();
    } catch {
      // ignore
    }
  }

  async commitEditor(body) {
    if (!hasWails()) return;
    try {
      const draft = await AppAPI.SetScriptBody(body);
      this.applyDraft(draft);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async moveLine(index, dir) {
    if (!hasWails()) return;
    try {
      const draft = await AppAPI.MoveScriptLine(index, dir);
      this.applyDraft(draft);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async removeLine(index) {
    if (!hasWails()) return;
    try {
      const draft = await AppAPI.RemoveScriptLine(index);
      this.applyDraft(draft);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async reorderLines(from, to) {
    if (!hasWails() || from === to) return;
    const lines = [...(this.draft.lines || [])];
    if (from < 0 || from >= lines.length || to < 0 || to >= lines.length) return;
    const [item] = lines.splice(from, 1);
    lines.splice(to, 0, item);
    const body = `${this.draft.shebang || '#!/usr/bin/env bash'}\n${lines.join('\n')}\n`;
    await this.commitEditor(body);
  }

  async clearDraft() {
    if (!hasWails()) return;
    this.savedScriptId = 0;
    const draft = await AppAPI.ClearScriptDraft();
    this.applyDraft(draft);
  }

  async exportScript(body) {
    if (!hasWails()) return;
    try {
      const path = await AppAPI.ExportScriptDialog(body || this.draft.body || '');
      if (path) this.showToast(fmt(uk.toastScriptExported, { path }));
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async saveScript(body) {
    if (!hasWails()) return;
    const name = window.prompt(uk.builderSavePrompt, uk.builderSaveDefault);
    if (name == null) return;
    const trimmed = name.trim();
    if (!trimmed) {
      this.showToast(uk.builderSaveValidation);
      return;
    }
    try {
      const cs = await AppAPI.SaveCustomScript(this.savedScriptId || 0, {
        name: trimmed,
        scriptBody: body || this.draft.body || '',
      });
      this.savedScriptId = cs?.id || 0;
      await this.refreshSavedScripts();
      this.showToast(uk.toastScriptSaved);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async loadSavedScript(id) {
    if (!hasWails()) return;
    try {
      const draft = await AppAPI.LoadCustomScript(id);
      this.savedScriptId = id;
      this.applyDraft(draft);
      this.showToast(uk.toastScriptLoaded);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async deleteSavedScript(id) {
    if (!hasWails()) return;
    if (!window.confirm(uk.builderDeleteConfirm)) return;
    try {
      await AppAPI.DeleteCustomScript(id);
      if (this.savedScriptId === id) this.savedScriptId = 0;
      await this.refreshSavedScripts();
      this.showToast(uk.toastScriptDeleted);
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async exportBackup() {
    if (!hasWails()) return;
    try {
      const path = await AppAPI.ExportBackupDialog();
      if (path) this.showToast(fmt(uk.toastBackupExported, { path }));
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
    }
  }

  async importBackup() {
    if (!hasWails()) return;
    try {
      const res = await AppAPI.ImportBackupDialog();
      if (!res) return;
      this.showToast(fmt(uk.toastBackupImported, {
        sn: res.snippetsAdded || 0,
        sc: res.scriptsAdded || 0,
      }));
      await this.refreshSavedScripts();
      await this.reloadList();
      if (this.selected) await this.reloadDetail();
    } catch (err) {
      this.showToast(`${uk.statusError}: ${err?.message || err}`);
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

  async cancelIndex() {
    if (!this.busy || !hasWails()) return;
    try {
      await AppAPI.CancelIndex();
      this.status = uk.statusCancelled;
      this.paintToolbar();
    } catch (err) {
      console.error(err);
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
      const msg = err?.message || String(err);
      this.status = /скасов/i.test(msg) ? uk.statusCancelled : `${uk.statusError}: ${msg}`;
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
      const msg = err?.message || String(err);
      this.status = /скасов/i.test(msg) ? uk.statusCancelled : `${uk.statusError}: ${msg}`;
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
