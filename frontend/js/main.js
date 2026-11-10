import { App } from './api.js';
import { installContextMenu } from './context-menu.js';
import { installTips } from './tooltip.js';

installContextMenu();
installTips();

(async () => {
  const mode = await App.mode().catch(() => 'sidebar');
  document.body.classList.add(mode === 'settings' ? 'mode-settings' : 'mode-sidebar');

  if (mode === 'settings') {
    await import('./settings-window.js');
  } else {
    await import('./sidebar-app.js');
  }
  document.body.classList.remove('pending');
  App.signalReady().catch(() => {});
})();
