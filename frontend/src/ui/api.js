/**
 * Thin wrappers around Wails Go bindings with safe fallbacks for browser smoke.
 */

async function call(name, ...args) {
  const fn = window?.go?.main?.App?.[name];
  if (typeof fn !== 'function') {
    throw new Error(`Wails binding missing: ${name}`);
  }
  return fn(...args);
}

export function hasWails() {
  return typeof window?.go?.main?.App?.AppInfo === 'function';
}

export const AppAPI = {
  AppInfo: () => call('AppInfo'),
  PhaseStatus: () => call('PhaseStatus'),
  EnsureIndexed: () => call('EnsureIndexed'),
  ScanInstalled: () => call('ScanInstalled'),
  UpdateIndex: () => call('UpdateIndex'),
  SyncRepoCatalog: () => call('SyncRepoCatalog'),
  GetProgram: (name, parseMan) => call('GetProgram', name, parseMan),
  GetRawMan: (name) => call('GetRawMan', name),
  Search: (query) => call('Search', query),
  ListInstalled: () => call('ListInstalled'),
  ListPrograms: (q) => call('ListPrograms', q),
  SearchPrograms: (q) => call('SearchPrograms', q),
  DetectPackageManager: () => call('DetectPackageManager'),
  InstallCommandFor: (pkg) => call('InstallCommandFor', pkg),
  ListSnippets: (programID) => call('ListSnippets', programID),
  CreateSnippet: (inPayload) => call('CreateSnippet', inPayload),
  UpdateSnippet: (id, inPayload) => call('UpdateSnippet', id, inPayload),
  DeleteSnippet: (id) => call('DeleteSnippet', id),
  AnalyzeCommand: (command) => call('AnalyzeCommand', command),
  CopyText: (text) => call('CopyText', text),
  RunCommand: (req) => call('RunCommand', req),
  AddToScript: (command) => call('AddToScript', command),
  ScriptDraft: () => call('ScriptDraft'),
};

/**
 * Subscribe to index progress events when Wails runtime is present.
 * @param {(payload: any) => void} handler
 * @returns {() => void} unsubscribe
 */
export function onIndexProgress(handler) {
  const runtime = window?.runtime;
  if (!runtime?.EventsOn) {
    return () => {};
  }
  const off = runtime.EventsOn('index:progress', handler);
  return typeof off === 'function' ? off : () => runtime.EventsOff?.('index:progress');
}
