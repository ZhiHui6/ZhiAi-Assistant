import { App, Calendar, WindowSvc, onEvent, toast } from './api.js';
import { initClock } from './clock.js';
import { createCalendar } from './calendar.js';
import { createNotes } from './notes.js';
import { createTodos } from './todos.js';
import { createShelf } from './shelf.js';
import { createLaunch } from './launch.js';
import { createTimers } from './timers.js';
import { createShutdown } from './shutdown.js';
import { createUsage } from './usage.js';
import { captureStatic, dataText, getLocale, onLocaleChange, setLocale, t, tp } from './strings.js';

captureStatic();
setLocale(await App.locale().catch(() => 'zh-CN'));

const initialSettings = await App.getSettings().catch(() => null);

const sidebarRoot = document.getElementById('sidebarRoot');
const panel = document.getElementById('panel');
const tabsEl = document.getElementById('tabs');
const pinBtn = document.getElementById('btnPin');
const settingsBtn = document.getElementById('btnSettings');
const settingsDot = document.getElementById('settingsDot');
const collapseBtn = document.getElementById('btnCollapse');
const updateBar = document.getElementById('updateBar');
const updateBarText = document.getElementById('updateBarText');
const updateBarOpenBtn = document.getElementById('updateBarOpen');
const updateBarCloseBtn = document.getElementById('updateBarClose');
const edgeMark = document.getElementById('edgeMark');
const launchDock = document.getElementById('launchDock');
const shelfView = document.getElementById('view-shelf');
const appListEl = document.getElementById('launchList');
let launchGridCols = 5;

function updateLaunchCells() {
  if (!appListEl || !launchDock) return;
  const width = appListEl.clientWidth || 0;
  if (width <= 0) return;
  const cellW = Math.max(30, (width - (launchGridCols - 1) * 6) / launchGridCols);
  launchDock.style.setProperty('--launch-cell-w', cellW + 'px');
}

if (typeof ResizeObserver !== 'undefined' && appListEl) {
  new ResizeObserver(updateLaunchCells).observe(appListEl);
}
window.addEventListener('resize', updateLaunchCells);

let activeTab = 'notes';
let pinned = false;
let peekSetting = 6;
let windowDpi = 96;
let markTimer = null;
let idleMode = '';

const clock = initClock();

let todayDay = null;

function applyTodaySuffix() {
  if (!todayDay) return;
  const text = todayDay.kind === 'festival' || todayDay.kind === 'jieqi'
    ? dataText(todayDay.label)
    : dataText('农历' + todayDay.lunar);
  clock.setSuffix(text);
}

Calendar.today()
  .then((day) => {
    todayDay = day || null;
    applyTodaySuffix();
  })
  .catch(() => {});

const calendar = createCalendar({
  onSelect(dateStr) {
    todos.setDate(dateStr);
    if (dateStr) switchTab('todos');
  },
});

const notes = createNotes();

const todos = createTodos({
  onChanged() {
    calendar.refresh();
  },
});

const shelf = createShelf({ onChanged() {} });

const launcher = createLaunch();

const timers = createTimers();
const shutdown = createShutdown();
const usage = createUsage();

let pendingUpdateVersion = '';

function renderUpdateNotice(info) {
  if (!info || !info.latest) return;
  pendingUpdateVersion = info.latest;
  updateBarText.textContent = tp('发现新版本 v{v}', { v: info.latest });
  updateBar.hidden = false;
  settingsDot.hidden = false;
}

function clearUpdateNotice() {
  pendingUpdateVersion = '';
  updateBar.hidden = true;
  settingsDot.hidden = true;
}

onEvent('update:available', renderUpdateNotice);
App.updateNotice()
  .then((info) => {
    if (info) renderUpdateNotice(info);
  })
  .catch(() => {});
updateBarOpenBtn.addEventListener('click', () => {
  App.showSettingsPage('about').catch(() => {});
});
updateBarCloseBtn.addEventListener('click', () => {
  if (pendingUpdateVersion) App.dismissUpdate(pendingUpdateVersion).catch(() => {});
  clearUpdateNotice();
});

onLocaleChange(() => {
  applyTodaySuffix();
  clock.refresh();
  calendar.refresh();
  notes.refresh();
  todos.refresh();
  shelf.refresh();
  launcher.refresh();
  timers.refresh();
  shutdown.refresh();
  usage.refresh();
  if (!updateBar.hidden && pendingUpdateVersion) {
    updateBarText.textContent = tp('发现新版本 v{v}', { v: pendingUpdateVersion });
  }
  syncPinButton();
});

function syncPinButton() {
  pinBtn.classList.toggle('active', pinned);
  pinBtn.title = pinned ? t('恢复自动隐藏') : t('临时保持展开（取消/重启后恢复自动隐藏）');
}

function hexToRgb(hex) {
  const match = /^#?([0-9a-f]{6})$/i.exec(String(hex || ''));
  if (!match) return [10, 12, 17];
  const n = parseInt(match[1], 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

function applyAppearance(s) {
  if (!s) return;
  if (typeof s.opacity === 'number') {
    document.documentElement.style.setProperty('--tint-alpha', String(s.opacity));
  }
  if (typeof s.background === 'string') {
    document.documentElement.style.setProperty('--bg-rgb', hexToRgb(s.background).join(', '));
  }
  clock.set12h(!!s.clock12);
  const calendarCard = document.getElementById('calendarCard');
  if (calendarCard) calendarCard.hidden = s.showCalendar === false;
  if (launchDock) launchDock.hidden = !!s.hideLaunch;
  if (calendar) calendar.setShowDetails(!s.hideCalendarFestivals);

  const cols = Math.max(3, Math.min(8, Number(s.launchColumns) || 5));
  const rows = Math.max(1, Math.min(3, Number(s.launchRows) || 1));
  const icon = Math.max(24, Math.min(40, Number(s.launchIconSize) || 32));
  const hideNames = !!s.hideLaunchNames;
  const cell = icon + (hideNames ? 14 : 25);
  launchGridCols = cols;
  launchDock.classList.toggle('hide-names', hideNames);
  launchDock.classList.remove('rows-1', 'rows-2', 'rows-3');
  launchDock.classList.add('rows-' + rows);
  launchDock.style.setProperty('--launch-icon', icon + 'px');
  launchDock.style.setProperty('--launch-cell', cell + 'px');
  updateLaunchCells();
  panel.classList.toggle('edge-left', s.edge === 'left');
  sidebarRoot.classList.toggle('edge-left', s.edge === 'left');
  sidebarRoot.classList.toggle('no-motion', !s.animations);
  document.body.classList.toggle('no-motion', !s.animations);
  collapseBtn.hidden = !!s.edgeEnabled;
  pinBtn.hidden = !s.edgeEnabled;
  if (typeof s.peek === 'number') peekSetting = s.peek;
  updatePeekVar();
  applyShelfView(s);
}

const SHELF_SCALES = {
  compact: { cell: 48, icon: 24 },
  normal: { cell: 62, icon: 32 },
  large: { cell: 86, icon: 46 },
};

function applyShelfView(s) {
  const shelf = SHELF_SCALES[s.shelfScale] || SHELF_SCALES.normal;
  const mount = SHELF_SCALES[s.mountScale] || SHELF_SCALES.normal;
  document.documentElement.style.setProperty('--shelf-cell', shelf.cell + 'px');
  document.documentElement.style.setProperty('--shelf-icon', shelf.icon + 'px');
  document.documentElement.style.setProperty('--mount-cell', mount.cell + 'px');
  document.documentElement.style.setProperty('--mount-icon', mount.icon + 'px');
  sidebarRoot.classList.toggle('shelf-view-list', s.shelfView === 'list');
  sidebarRoot.classList.toggle('mount-view-grid', s.mountView === 'grid');
}

function updatePeekVar() {
  const value = Number(peekSetting);
  const visible = value >= 24 && value <= 40 ? value : 30;
  document.documentElement.style.setProperty('--peek-css', visible + 'px');
}

function applyWindowState(info) {
  if (!info) return;
  if (info.dpi) windowDpi = info.dpi;
  updatePeekVar();
  const manual = info.state === 'manual-idle';
  const auto = info.state === 'edge-idle';
  const closed = manual || auto;
  const left = info.edge === 'left';
  panel.classList.toggle('closed-right', closed && !left);
  panel.classList.toggle('closed-left', closed && left);
  panel.classList.toggle('edge-left', left);
  sidebarRoot.classList.toggle('edge-left', left);
  edgeMark.classList.toggle('manual', manual);
  edgeMark.classList.toggle('auto', auto);
  edgeMark.querySelector('use').setAttribute('href', left ? '#i-right' : '#i-left');

  const mode = manual ? 'manual' : auto ? 'auto' : '';
  if (mode === idleMode) return;
  idleMode = mode;
  if (markTimer) {
    clearTimeout(markTimer);
    markTimer = null;
  }
  edgeMark.hidden = true;
  if (!mode) return;
  const delay = sidebarRoot.classList.contains('no-motion') ? 0 : 220;
  markTimer = setTimeout(() => {
    edgeMark.hidden = false;
    markTimer = null;
  }, delay);
}

edgeMark.addEventListener('click', () => {
  if (!edgeMark.classList.contains('manual')) return;
  edgeMark.hidden = true;
  WindowSvc.reveal('').catch(() => {});
});

function commitActiveEditor() {
  const active = document.activeElement;
  if (active && (active.tagName === 'TEXTAREA' || active.tagName === 'INPUT')) {
    active.blur();
  }
}

function switchTab(tab) {
  commitActiveEditor();
  activeTab = tab;
  document.querySelectorAll('.tab').forEach((t) => t.classList.toggle('active', t.dataset.tab === tab));
  document.querySelectorAll('.view').forEach((v) => {
    v.classList.toggle('active', v.id === 'view-' + tab);
  });
}

tabsEl.addEventListener('click', (e) => {
  const tab = e.target.closest('.tab');
  if (tab) switchTab(tab.dataset.tab);
});

settingsBtn.addEventListener('click', () => {
  App.openSettingsWindow().catch(() => toast(t('无法打开设置窗口')));
});

collapseBtn.addEventListener('click', () => {
  WindowSvc.collapse().catch(() => {});
});

pinBtn.addEventListener('click', async () => {
  try {
    pinned = await WindowSvc.togglePin();
    syncPinButton();
  } catch (err) {
    toast(t('操作失败'));
  }
});

document.addEventListener(
  'mousedown',
  () => {
    WindowSvc.activate().catch(() => {});
  },
  true,
);

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    const menu = document.querySelector('.ctx-menu');
    if (menu) {
      menu.remove();
      return;
    }
    const active = document.activeElement;
    if (active && active.tagName === 'TEXTAREA' && active.value) {
      return;
    }
    if (active && active.tagName === 'TEXTAREA') active.blur();
  }
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault();
    switchTab('notes');
    const search = document.getElementById('noteSearch');
    if (search) search.focus();
  }
});

onEvent('state:changed', (info) => {
  if (!info) return;
  applyWindowState(info);
  pinned = !!info.pinned;
  syncPinButton();
});

onEvent('panel:shown', (payload) => {
  clock.refresh();
  setTimeout(() => reportLayout('shown'), 120);
  const view = payload && payload.view;
  if (view) switchTab(view);
  if (activeTab === 'todos') todos.refresh();
  if (activeTab === 'notes') notes.refresh();
  if (activeTab === 'shelf') shelf.refresh();
  if (activeTab === 'timers') timers.refresh();
  if (activeTab === 'shutdown') shutdown.refresh();
  if (activeTab === 'usage') usage.refresh();
  calendar.refresh();
});

onEvent('settings:changed', (s) => {
  applyAppearance(s);
  if (s && s.language) {
    App.locale()
      .then((loc) => {
        if (loc !== getLocale()) setLocale(loc);
      })
      .catch(() => {});
  }
  if (s && !s.edgeEnabled && pinned) {
    pinned = false;
    WindowSvc.setPinned(false).catch(() => {});
    syncPinButton();
  }
});
onEvent('settings:preview', applyAppearance);

onEvent('monitors:changed', () => {
  calendar.refresh();
});

onEvent('calendar:holidays', () => {
  calendar.refresh();
});

async function handleFileDrop(x, y, paths) {
  if (!paths || !paths.length) return;
  const target = document.elementFromPoint(x, y);
  const inLaunch = target && target.closest('#launchDock');
  if (inLaunch) {
    await launcher.handleDrop(x, y, paths);
    return;
  }

  switchTab('shelf');
  await shelf.handleDrop(x, y, paths);
}

if (window.runtime && typeof window.runtime.OnFileDrop === 'function') {
  window.runtime.OnFileDrop((x, y, paths) => {
    handleFileDrop(x, y, paths);
  }, true);
} else {
  onEvent('wails:file-drop', (data) => {
    if (data) handleFileDrop(data.x, data.y, data.paths);
  });
}

function isFileDrag(e) {
  return !!e.dataTransfer && Array.from(e.dataTransfer.types || []).indexOf('Files') !== -1;
}

function clearFileDropTargets() {
  document.body.classList.remove('file-drag-mount', 'file-drag-shelf');
  document.querySelectorAll('.shelf-group.file-drop-target').forEach((el) => {
    el.classList.remove('file-drop-target');
  });
}

window.addEventListener('dragover', (e) => {
  if (!isFileDrag(e)) return;
  const first = document.elementFromPoint(e.clientX, e.clientY);
  const inLaunch = !!first && !!first.closest('#launchDock');
  if (!inLaunch && !shelfView.classList.contains('active')) switchTab('shelf');
  const target = document.elementFromPoint(e.clientX, e.clientY);
  const inMount = !!target && !!target.closest('.mount-section');
  const inShelf = !!target && !!target.closest('.shelf-section');
  const groupEl = inShelf ? target.closest('.shelf-group') : null;
  const multiGroup = inShelf && document.querySelectorAll('.shelf-section .shelf-group').length > 1;
  document.body.classList.toggle('file-drag-mount', inMount);
  document.body.classList.toggle('file-drag-shelf', inShelf && !groupEl && !multiGroup);
  document.querySelectorAll('.shelf-group.file-drop-target').forEach((el) => {
    if (el !== groupEl) el.classList.remove('file-drop-target');
  });
  if (groupEl) groupEl.classList.add('file-drop-target');
  launchDock.classList.toggle('drag-over', inLaunch);
});

window.addEventListener('dragleave', (e) => {
  if (!e.relatedTarget) {
    launchDock.classList.remove('drag-over');
    clearFileDropTargets();
  }
});

window.addEventListener('drop', () => {
  launchDock.classList.remove('drag-over');
  clearFileDropTargets();
});

applyAppearance(initialSettings);

WindowSvc.getInfo()
  .then((info) => {
    applyWindowState(info);
    pinned = !!info.pinned;
    syncPinButton();
  })
  .catch(() => {});

function reportLayout(tag) {
  if (!window.__debug) return;
  try {
    const rects = {};
    document
      .querySelectorAll('.tab, #todoInput, #noteSearch, #btnPin, #btnSettings, #btnCollapse')
      .forEach((node, i) => {
        const r = node.getBoundingClientRect();
        if (r.width === 0 && r.height === 0) return;
        const key = (node.id || node.dataset.tab || node.className) + '#' + i;
        rects[key] = [Math.round(r.left), Math.round(r.top), Math.round(r.width), Math.round(r.height)];
      });
    App.debugLog(tag + ' ' + JSON.stringify({ dpr: window.devicePixelRatio, w: innerWidth, h: innerHeight, rects }));
  } catch (err) {
  }
}

App.isDebug()
  .then((v) => {
    window.__debug = !!v;
    if (window.__debug) setTimeout(() => reportLayout('ready'), 300);
  })
  .catch(() => {});

['notes', 'todos', 'shelf', 'timers', 'shutdown', 'usage'].forEach((tab) => {
  document.querySelector(`.tab[data-tab="${tab}"]`).addEventListener('click', () => {
    if (tab === 'shelf') shelf.refresh();
    if (tab === 'todos') todos.refresh();
    if (tab === 'notes') notes.refresh();
    if (tab === 'timers') timers.refresh();
    if (tab === 'shutdown') shutdown.refresh();
    if (tab === 'usage') usage.refresh();
  });
});

