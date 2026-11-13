import './styles/main.css';
import uk from './i18n/uk.js';
import { renderShell } from './ui/shell.js';
import { AppInfo, PhaseStatus } from '../wailsjs/go/main/App';

const root = document.querySelector('#app');
root.textContent = uk.loading;

Promise.all([AppInfo(), PhaseStatus()])
  .then(([info, status]) => {
    renderShell(root, info, status);
  })
  .catch((err) => {
    console.error(err);
    renderShell(root, {
      name: uk.brand,
      subtitle: uk.subtitle,
      phase: '0',
      uiLocale: 'uk',
      frontend: 'vanilla-js',
      dbPath: '~/.local/share/linux-cmd-wiki/wiki.db',
    }, uk.phaseHint);
  });
