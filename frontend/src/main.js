import './styles/main.css';
import uk from './i18n/uk.js';
import { renderShell } from './ui/shell.js';
import { AppAPI, hasWails } from './ui/api.js';

const root = document.querySelector('#app');
root.textContent = uk.loading;

function fallbackInfo() {
  return {
    name: uk.brand,
    subtitle: uk.subtitle,
    phase: '3',
    uiLocale: 'uk',
    frontend: 'vanilla-js',
    dbPath: '~/.local/share/linux-cmd-wiki/wiki.db',
  };
}

if (!hasWails()) {
  renderShell(root, fallbackInfo(), uk.phaseHint);
} else {
  Promise.all([AppAPI.AppInfo(), AppAPI.PhaseStatus()])
    .then(([info, status]) => {
      renderShell(root, info, status);
    })
    .catch((err) => {
      console.error(err);
      renderShell(root, fallbackInfo(), uk.phaseHint);
    });
}
