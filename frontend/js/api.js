import { t, tp } from './strings.js';

export function call(svc, method, ...args) {
  const g = window.go;
  if (!g || !g.service || !g.service[svc]) {
    throw new Error(t('后端服务尚未就绪：') + svc);
  }
  const fn = g.service[svc][method];
  if (typeof fn !== 'function') {
    throw new Error(tp('接口不存在：{svc}.{method}', { svc, method }));
  }
  return fn(...args);
}

export async function resolveLocale(value) {
  if (value === 'zh-CN' || value === 'zh-TW' || value === 'en') return value;
  return App.systemLocale().catch(() => 'zh-CN');
}

export const App = {
  mode: () => call('App', 'Mode'),
  signalReady: () => call('App', 'SignalReady'),
  getSettings: () => call('App', 'GetSettings'),
  defaults: () => call('App', 'Defaults'),
  preview: (s) => call('App', 'Preview', s),
  apply: (s) => call('App', 'Apply', s),
  revert: () => call('App', 'Revert'),
  locale: () => call('App', 'Locale'),
  systemLocale: () => call('App', 'SystemLocale'),
  closeSettings: () => call('App', 'CloseSettings'),
  fitSettingsWindow: (height) => call('App', 'FitSettingsWindow', height),
  openSettingsWindow: () => call('App', 'OpenSettingsWindow'),
  sidebarRunning: () => call('App', 'SidebarRunning'),
  updateSettings: (s) => call('App', 'UpdateSettings', s),
  usageAutoAll: () => call('App', 'UsageAutoAll'),
  setUsageAutoAll: (enabled) => call('App', 'SetUsageAutoAll', enabled),
  getMonitors: () => call('App', 'GetMonitors'),
  getAutostart: () => call('App', 'GetAutostart'),
  setAutostart: (v) => call('App', 'SetAutostart', v),
  openDataDir: () => call('App', 'OpenDataDir'),
  openExternal: (u) => call('App', 'OpenExternal', u),
  hotkeyInfo: () => call('App', 'HotkeyInfo'),
  refreshHolidays: () => call('App', 'RefreshHolidays'),
  debugLog: (m) => call('App', 'DebugLog', m),
  clipboardText: () => call('App', 'ClipboardText'),
  setClipboardText: (text) => call('App', 'SetClipboardText', text),
  setTimerSound: (kind, sound) => call('App', 'SetTimerSound', kind, sound),
  previewTimerSound: (sound) => call('App', 'PreviewTimerSound', sound),
  pickSound: () => call('App', 'PickSound'),
  aboutInfo: () => call('App', 'AboutInfo'),
  checkUpdate: () => call('App', 'CheckUpdate'),
  updateNotice: () => call('App', 'UpdateNotice'),
  dismissUpdate: (version) => call('App', 'DismissUpdate', version),
  getAutoUpdateCheck: () => call('App', 'GetAutoUpdateCheck'),
  setAutoUpdateCheck: (enabled) => call('App', 'SetAutoUpdateCheck', enabled),
  showSettingsPage: (page) => call('App', 'ShowSettingsPage', page),
  startPage: () => call('App', 'StartPage'),
  installUpdate: () => call('App', 'InstallUpdate'),
  resetAll: () => call('App', 'ResetAll'),
  dataLocation: () => call('App', 'DataLocation'),
  setDataLocation: (mode, custom) => call('App', 'SetDataLocation', mode, custom || ''),
  pickFolder: (title) => call('App', 'PickFolder', title),
  createBackupTo: (dir, categories) => call('App', 'CreateBackupTo', dir, categories || []),
  pickBackupFile: () => call('App', 'PickBackupFile'),
  backupContents: (path) => call('App', 'BackupContents', path),
  restoreBackup: (path, categories) => call('App', 'RestoreBackupData', path, categories || []),
  isDebug: () => call('App', 'Debug'),
  quit: () => call('App', 'Quit'),
};

export const WindowSvc = {
  getInfo: () => call('Window', 'GetInfo'),
  reveal: (view) => call('Window', 'Reveal', view),
  collapse: () => call('Window', 'Collapse'),
  toggle: () => call('Window', 'Toggle'),
  togglePin: () => call('Window', 'TogglePin'),
  setPinned: (v) => call('Window', 'SetPinned', v),
  activate: () => call('Window', 'Activate'),
};

export const Notes = {
  list: (kw, limit) => call('Notes', 'List', kw || '', limit || 0),
  create: (text, tags) => call('Notes', 'Create', text, tags || []),
  update: (id, text, tags, pinned) => call('Notes', 'Update', id, text, tags, pinned),
  remove: (id) => call('Notes', 'Delete', id),
  setTint: (id, tint) => call('Notes', 'SetTint', id, tint || ''),
};

export const Todos = {
  list: (filter) => call('Todos', 'List', filter),
  create: (input) => call('Todos', 'Create', input),
  update: (id, patch) => call('Todos', 'Update', id, patch),
  toggle: (id, done) => call('Todos', 'Toggle', id, done),
  remove: (id) => call('Todos', 'Delete', id),
  restore: (todo) => call('Todos', 'Restore', todo),
  stats: () => call('Todos', 'Stats'),
};

export const Calendar = {
  month: (y, m) => call('Calendar', 'Month', y, m),
  day: (date) => call('Calendar', 'Day', date),
  festivals: (y, m) => call('Calendar', 'Festivals', y, m),
  today: () => call('Calendar', 'Today'),
};

export const Shelf = {
  list: () => call('Shelf', 'List'),
  addPaths: (groupId, paths) => call('Shelf', 'AddPaths', groupId, paths),
  remove: (id) => call('Shelf', 'Remove', id),
  update: (id, name, args, workDir) => call('Shelf', 'Update', id, name, args, workDir),
  move: (id, groupId, index) => call('Shelf', 'Move', id, groupId, index),
  createGroup: (name) => call('Shelf', 'CreateGroup', name),
  renameGroup: (id, name) => call('Shelf', 'RenameGroup', id, name),
  toggleGroup: (id, collapsed) => call('Shelf', 'ToggleGroup', id, collapsed),
  deleteGroup: (id) => call('Shelf', 'DeleteGroup', id),
  launch: (id) => call('Shelf', 'Launch', id),
  reveal: (id) => call('Shelf', 'Reveal', id),
  pickFiles: (groupId) => call('Shelf', 'PickFiles', groupId),
  pickFolder: (groupId) => call('Shelf', 'PickFolder', groupId),
};

export const Launch = {
  list: () => call('Launch', 'List'),
  addPaths: (categoryId, paths) => call('Launch', 'AddPaths', categoryId, paths),
  remove: (id) => call('Launch', 'Remove', id),
  rename: (id, name) => call('Launch', 'Rename', id, name),
  move: (id, categoryId) => call('Launch', 'Move', id, categoryId),
  addCategory: (name) => call('Launch', 'AddCategory', name),
  renameCategory: (id, name) => call('Launch', 'RenameCategory', id, name),
  removeCategory: (id) => call('Launch', 'RemoveCategory', id),
  launch: (id) => call('Launch', 'Launch', id),
  reveal: (id) => call('Launch', 'Reveal', id),
  pick: (categoryId) => call('Launch', 'Pick', categoryId),
};

export const Timers = {
  state: () => call('Timers', 'State'),
  addAlarm: (at, label, repeat) => call('Timers', 'AddAlarm', at, label, repeat),
  toggleAlarm: (id, enabled) => call('Timers', 'ToggleAlarm', id, enabled),
  removeAlarm: (id) => call('Timers', 'RemoveAlarm', id),
  startCountdown: (seconds, label) => call('Timers', 'StartCountdown', seconds, label),
  pauseCountdown: () => call('Timers', 'PauseCountdown'),
  resumeCountdown: () => call('Timers', 'ResumeCountdown'),
  resetCountdown: () => call('Timers', 'ResetCountdown'),
  scheduleShutdown: (atUnix) => call('Timers', 'ScheduleShutdown', atUnix),
  cancelShutdown: () => call('Timers', 'CancelShutdown'),
};

export const Usage = {
  list: () => call('Usage', 'List'),
  refresh: (id) => call('Usage', 'Refresh', id),
  refreshAll: () => call('Usage', 'RefreshAll'),
  checkin: (id) => call('Usage', 'Checkin', id),
  checkinAll: () => call('Usage', 'CheckinAll'),
  signOut: (id) => call('Usage', 'SignOut', id),
  accountFull: (id) => call('Usage', 'AccountFull', id),
  openClient: (id) => call('Usage', 'OpenClient', id),
  startBrowserLogin: (id) => call('Usage', 'StartBrowserLogin', id),
  pollBrowserLogin: (id, loginId) => call('Usage', 'PollBrowserLogin', id, loginId),
  cancelBrowserLogin: (loginId) => call('Usage', 'CancelBrowserLogin', loginId),
  setAutoAll: (enabled) => call('Usage', 'SetAutoCheckinAll', enabled),
  reread: (id) => call('Usage', 'ReRead', id),
  ackNotice: () => call('Usage', 'AckNotice'),
  startWorkBuddyLogin: () => call('Usage', 'StartWorkBuddyLogin'),
  pollWorkBuddyLogin: (id) => call('Usage', 'PollWorkBuddyLogin', id),
};

export const Mount = {
  list: () => call('Mount', 'List'),
  add: (path) => call('Mount', 'Add', path),
  pick: () => call('Mount', 'Pick'),
  remove: (id) => call('Mount', 'Remove', id),
  children: (id, rel, opts) => call('Mount', 'Children', id, rel, opts),
  open: (id, rel) => call('Mount', 'Open', id, rel),
  reveal: (id, rel) => call('Mount', 'Reveal', id, rel),
};

export const Icons = {
  get: (paths) => call('Icons', 'Get', paths),
  getOne: (path) => call('Icons', 'GetOne', path),
};

export const Thumbs = {
  get: (paths) => call('Thumbs', 'Get', paths),
};

export function onEvent(name, handler) {
  if (!window.runtime || !window.runtime.EventsOn) return () => {};
  return window.runtime.EventsOn(name, handler);
}

export function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined && text !== null) node.textContent = text;
  return node;
}

export function svgIcon(id, cls) {
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('class', cls || 'icon');
  const use = document.createElementNS('http://www.w3.org/2000/svg', 'use');
  use.setAttribute('href', '#' + id);
  svg.appendChild(use);
  return svg;
}

export function pad2(n) {
  return String(n).padStart(2, '0');
}

export function todayStr(d) {
  const t = d || new Date();
  return `${t.getFullYear()}-${pad2(t.getMonth() + 1)}-${pad2(t.getDate())}`;
}

export function fmtDuration(total) {
  const sec = Math.max(0, Math.floor(total));
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = sec % 60;
  if (h > 0) return `${pad2(h)}:${pad2(m)}:${pad2(s)}`;
  return `${pad2(m)}:${pad2(s)}`;
}

export function fmtTarget(ms) {
  const target = new Date(ms);
  const now = new Date();
  const time = `${pad2(target.getHours())}:${pad2(target.getMinutes())}`;
  if (target.toDateString() === now.toDateString()) return tp('今天 {time}', { time });
  if (new Date(now.getTime() + 86400000).toDateString() === target.toDateString()) return tp('明天 {time}', { time });
  return tp('{m}月{d}日 {time}', { m: target.getMonth() + 1, d: target.getDate(), time });
}

export function relTime(iso) {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  const diff = (Date.now() - date.getTime()) / 1000;
  if (diff < 60) return t('刚刚');
  if (diff < 3600) return tp('{n} 分钟前', { n: Math.floor(diff / 60) });
  if (diff < 86400) return tp('{n} 小时前', { n: Math.floor(diff / 3600) });
  if (diff < 86400 * 7) return tp('{n} 天前', { n: Math.floor(diff / 86400) });
  return tp('{m}月{d}日', { m: date.getMonth() + 1, d: date.getDate() });
}

export function fmtSize(bytes) {
  if (!bytes && bytes !== 0) return '';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB';
  return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB';
}

export function debounce(fn, wait) {
  let timer = null;
  return (...args) => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => fn(...args), wait);
  };
}

let toastTimer = null;
export function toast(message, opts) {
  const node = document.getElementById('toast');
  if (!node) return;
  const options = opts || {};
  node.textContent = '';
  node.appendChild(el('span', 'toast-text', message));
  if (options.action) {
    const action = el('button', 'toast-action', options.action);
    action.addEventListener('click', () => {
      node.classList.remove('show');
      if (toastTimer) clearTimeout(toastTimer);
      if (options.onAction) options.onAction();
    });
    node.appendChild(action);
  }
  node.classList.add('show');
  if (toastTimer) clearTimeout(toastTimer);
  toastTimer = setTimeout(() => node.classList.remove('show'), options.duration || 1900);
}

export function closeContextMenu() {
  const existing = document.querySelector('.ctx-menu');
  if (existing) existing.remove();
}

export function openContextMenu(x, y, items) {
  closeContextMenu();
  const menu = el('div', 'ctx-menu');
  items.forEach((item) => {
    if (item.sep) {
      menu.appendChild(el('div', 'sep'));
      return;
    }
    const btn = el('button', item.danger ? 'danger' : '');
    if (item.disabled) btn.disabled = true;
    if (item.icon) btn.appendChild(svgIcon(item.icon, 'icon'));
    btn.appendChild(el('span', 'ctx-label', item.label));
    if (item.key) btn.appendChild(el('span', 'ctx-key', item.key));
    btn.addEventListener('mousedown', (e) => e.preventDefault());
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      closeContextMenu();
      try {
        const result = item.action();
        if (result && typeof result.catch === 'function') {
          result.catch((err) => toast(String(err).replace(/^Error:\s*/, '')));
        }
      } catch (err) {
        toast(String(err).replace(/^Error:\s*/, ''));
      }
    });
    menu.appendChild(btn);
  });
  menu.style.left = '0px';
  menu.style.top = '0px';
  document.body.appendChild(menu);
  const rect = menu.getBoundingClientRect();
  menu.style.left = Math.max(4, Math.min(x, window.innerWidth - rect.width - 6)) + 'px';
  menu.style.top = Math.max(4, Math.min(y, window.innerHeight - rect.height - 6)) + 'px';
  const dismiss = (e) => {
    if (!menu.contains(e.target)) {
      closeContextMenu();
      document.removeEventListener('mousedown', dismiss, true);
    }
  };
  setTimeout(() => document.addEventListener('mousedown', dismiss, true), 0);
}
