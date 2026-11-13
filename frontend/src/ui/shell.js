import uk from '../i18n/uk.js';

/**
 * Renders the Phase 0 application shell (brand + status).
 * Full Wiki layout arrives in Phase 2.
 * @param {HTMLElement} root
 * @param {{ name?: string, subtitle?: string, dbPath?: string, uiLocale?: string, frontend?: string, phase?: string }} info
 * @param {string} status
 */
export function renderShell(root, info, status) {
  const brand = info?.name || uk.brand;
  const subtitle = info?.subtitle || uk.subtitle;
  const dbPath = info?.dbPath || '~/.local/share/linux-cmd-wiki/wiki.db';
  const locale = info?.uiLocale || 'uk';
  const frontend = info?.frontend || 'vanilla-js';

  root.innerHTML = `
    <main class="shell">
      <h1 class="brand">${escapeHtml(brand)}</h1>
      <p class="subtitle">${escapeHtml(subtitle)}</p>
      <div class="phase">
        <h2>${escapeHtml(uk.phaseShell)}</h2>
        <p>${escapeHtml(status || uk.phaseHint)}</p>
      </div>
      <div class="meta">
        <div><span>${escapeHtml(uk.dbPathLabel)}:</span> ${escapeHtml(dbPath)}</div>
        <div><span>${escapeHtml(uk.localeLabel)}:</span> ${escapeHtml(locale)}</div>
        <div><span>${escapeHtml(uk.frontendLabel)}:</span> ${escapeHtml(frontend)}</div>
      </div>
      <div class="badge">Phase ${escapeHtml(info?.phase || '0')}</div>
    </main>
  `;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}
