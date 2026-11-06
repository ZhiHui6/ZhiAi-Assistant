import { App, onEvent, resolveLocale } from './api.js';
import { captureStatic, getLocale, onLocaleChange, setLocale, t, tp } from './strings.js';
import { enhanceSelect, refreshSelect } from './select.js';
import { bindAriaTip } from './tooltip.js';

captureStatic();
const bootSettings = await App.getSettings().catch(() => null);
setLocale(await resolveLocale(bootSettings && bootSettings.language));

const languageEl = document.getElementById('setLanguage');
const widthRange = document.getElementById('setWidth');
const widthNum = document.getElementById('setWidthNum');
const widthHint = document.getElementById('widthHint');
const opacityRange = document.getElementById('setOpacity');
const opacityNum = document.getElementById('setOpacityNum');
const backgroundEl = document.getElementById('setBackground');
const bgOut = document.getElementById('setBgOut');
const bgReset = document.getElementById('setBgReset');
const bgSwatches = document.getElementById('bgSwatches');
const peekRange = document.getElementById('setPeek');
const peekNum = document.getElementById('setPeekNum');
const edgeEl = document.getElementById('setEdge');
const monitorEl = document.getElementById('setMonitor');
const hotkeyEl = document.getElementById('setHotkey');
const hotkeyReset = document.getElementById('setHotkeyReset');
const edgeEnabledEl = document.getElementById('setEdgeEnabled');
const autostartEl = document.getElementById('setAutostart');
const animationsEl = document.getElementById('setAnimations');
const clock12El = document.getElementById('setClock12');
const showCalendarEl = document.getElementById('setShowCalendar');
const calendarFestivalsEl = document.getElementById('setCalendarFestivals');
const launchColsEl = document.getElementById('setLaunchCols');
const launchColsNum = document.getElementById('setLaunchColsNum');
const launchRowsEl = document.getElementById('setLaunchRows');
const launchRowsNum = document.getElementById('setLaunchRowsNum');
const launchIconEl = document.getElementById('setLaunchIcon');
const launchColsReset = document.getElementById('setLaunchColsReset');
const launchRowsReset = document.getElementById('setLaunchRowsReset');
const launchIconReset = document.getElementById('setLaunchIconReset');
const launchIconNum = document.getElementById('setLaunchIconNum');
const launchNamesEl = document.getElementById('setLaunchNames');
const showLaunchEl = document.getElementById('setShowLaunch');
const usageAutoMasterEl = document.getElementById('usageAutoMaster');
const switchEls = Array.from(document.querySelectorAll('#pageSettings .switch'));
const stateEls = switchEls.map((input) => document.getElementById(`${input.id}State`));
const statusEl = document.getElementById('sStatus');
const titleEl = document.getElementById('sTitle');
const widthReset = document.getElementById('setWidthReset');
const opacityReset = document.getElementById('setOpacityReset');
const peekReset = document.getElementById('setPeekReset');
const navEl = document.getElementById('sNav');
const navItems = navEl ? Array.from(navEl.querySelectorAll('.s-nav-item[data-target]')) : [];
const pageButtons = navEl ? Array.from(navEl.querySelectorAll('.s-nav-page')) : [];
const navSections = navItems
  .map((item) => document.getElementById(item.dataset.target))
  .filter(Boolean);
const pages = {
  settings: document.getElementById('pageSettings'),
  help: document.getElementById('pageHelp'),
  about: document.getElementById('pageAbout'),
};
const aboutMetaEl = document.getElementById('aboutMeta');
const aboutVersionEl = document.getElementById('aboutVersion');
const checkUpdateBtn = document.getElementById('checkUpdate');
const autoUpdateEl = document.getElementById('autoUpdate');
const updateStatusEl = document.getElementById('updateStatus');
const updateDetailEl = document.getElementById('updateDetail');
const updateLatestEl = document.getElementById('updateLatest');
const updateDateEl = document.getElementById('updateDate');
const updateNotesEl = document.getElementById('updateNotes');
const installUpdateBtn = document.getElementById('installUpdate');
const openReleaseBtn = document.getElementById('openRelease');
const updateConfirmEl = document.getElementById('updateConfirm');
const updateConfirmYes = document.getElementById('updateConfirmYes');
const updateConfirmNo = document.getElementById('updateConfirmNo');
const openProjectBtn = document.getElementById('openProject');
const openIssuesBtn = document.getElementById('openIssues');
const resetAllBtn = document.getElementById('resetAll');
const resetModal = document.getElementById('resetModal');
const resetConfirmInput = document.getElementById('resetConfirmInput');
const resetConfirmBtn = document.getElementById('resetConfirm');
const resetCancelBtn = document.getElementById('resetCancel');
const resetErrorEl = document.getElementById('resetError');
const dataDirModeEl = document.getElementById('setDataDirMode');
const dataDirCustomEl = document.getElementById('setDataDirCustom');
const dataDirEffectiveEl = document.getElementById('dataDirEffective');
const pickDataDirBtn = document.getElementById('pickDataDir');
const dataDirModal = document.getElementById('dataDirModal');
const dataDirSummaryEl = document.getElementById('dataDirSummary');
const dataDirErrorEl = document.getElementById('dataDirError');
const dataDirConfirmBtn = document.getElementById('dataDirConfirm');
const dataDirCancelBtn = document.getElementById('dataDirCancel');
const restoreSettingsBtn = document.getElementById('restoreSettings');
const restoreDefaultsModal = document.getElementById('restoreDefaultsModal');
const restoreDefaultsCancelBtn = document.getElementById('restoreDefaultsCancel');
const restoreDefaultsConfirmBtn = document.getElementById('restoreDefaultsConfirm');
const backupNowBtn = document.getElementById('backupNow');
const backupHintEl = document.getElementById('backupHint');
const restoreNowBtn = document.getElementById('restoreNow');
const backupModal = document.getElementById('backupModal');
const backupTitleEl = document.getElementById('backupTitle');
const backupSummaryEl = document.getElementById('backupSummary');
const backupListEl = document.getElementById('backupList');
const backupAllBtn = document.getElementById('backupAll');
const backupNoneBtn = document.getElementById('backupNone');
const backupErrorEl = document.getElementById('backupError');
const backupConfirmBtn = document.getElementById('backupConfirm');
const backupCancelBtn = document.getElementById('backupCancel');

const BACKUP_CATEGORIES = [
  { id: 'todos', name: '待办', desc: '任务、完成记录与逾期' },
  { id: 'notes', name: '随记', desc: '内容、标签与颜色' },
  { id: 'shelf', name: '收纳与挂载', desc: '收藏分组、条目与挂载目录' },
  { id: 'launch', name: '快捷启动', desc: '程序图标、分类与排列' },
  { id: 'timers', name: '计时与关机', desc: '闹钟、倒计时与关机安排' },
  { id: 'usage', name: '用量记录', desc: '各平台额度与签到状态' },
  { id: 'usageAuth', name: '用量登录凭据', desc: '本机加密，换机后需重新登录' },
  { id: 'settings', name: '设置与外观', desc: '面板外观、开关与热键' },
  { id: 'holidays', name: '节假日数据', desc: '离线日历所需的节假日缓存' },
];

let saved = null;
let monitorList = [];
let defaults = null;
let maxWidth = 560;
let previewTimer = null;
let persistTimer = null;
let persistPromise = null;
let statusTimer = null;
let sidebarRunning = false;
let resetBusy = false;
let restoreBusy = false;
let aboutLoaded = false;
let latestInfo = null;
let links = { project: '', releases: '', issues: '' };
let dataDirInfo = null;
let dataDirPending = null;
let dataDirBusy = false;
let backupBusy = false;
let backupMode = 'export';
let backupPath = '';
let backupLegacy = false;
let backupCounts = new Map();
let backupSelected = new Set();
let suppressPersist = false;

function clampWidth(v) {
  const n = Number(v);
  if (!Number.isFinite(n)) return 340;
  return Math.min(maxWidth, Math.max(340, Math.round(n)));
}

function clampRange(v, min, max, fallback, step) {
  const n = Number(v);
  if (!Number.isFinite(n) || n <= 0) return fallback;
  const snapped = step > 1 ? Math.round(n / step) * step : Math.round(n);
  return Math.min(max, Math.max(min, snapped));
}

function widthHintText() {
  return tp('展开宽度 {min}–{max} px，拖动预览', { min: 340, max: maxWidth });
}

const PAIRS = [
  { key: 'width', range: widthRange, num: widthNum, hint: widthHint, clamp: clampWidth },
  { key: 'peek', range: peekRange, num: peekNum, clamp: (v) => clampRange(v, 24, 40, 30, 1) },
  { key: 'opacity', range: opacityRange, num: opacityNum, clamp: (v) => clampRange(v, 25, 90, 80, 1) },
  { key: 'cols', range: launchColsEl, num: launchColsNum, clamp: (v) => clampRange(v, 3, 8, 5, 1) },
  { key: 'rows', range: launchRowsEl, num: launchRowsNum, clamp: (v) => clampRange(v, 1, 3, 1, 1) },
  { key: 'icon', range: launchIconEl, num: launchIconNum, clamp: (v) => clampRange(v, 24, 40, 32, 2) },
];
const pairOf = Object.fromEntries(PAIRS.map((pair) => [pair.key, pair]));

function setPairValue(pair, value) {
  const next = pair.clamp(value);
  pair.range.value = String(next);
  pair.num.value = String(next);
  if (pair.hint) pair.hint.textContent = widthHintText();
}

function syncPairFromRange(pair) {
  pair.num.value = String(pair.clamp(pair.range.value));
  if (pair.hint) pair.hint.textContent = widthHintText();
}

function syncPairFromNum(pair) {
  pair.range.value = String(pair.clamp(pair.num.value));
  if (pair.hint) pair.hint.textContent = widthHintText();
}

function bindPairs() {
  PAIRS.forEach((pair) => {
    pair.range.addEventListener('input', () => {
      syncPairFromRange(pair);
      schedulePreview();
    });
    pair.range.addEventListener('change', persistNow);
    pair.num.addEventListener('input', () => {
      syncPairFromNum(pair);
      schedulePreview();
    });
    pair.num.addEventListener('blur', () => {
      setPairValue(pair, pair.num.value);
      persistNow();
    });
    pair.num.addEventListener('focus', () => pair.num.select());
    pair.num.addEventListener('click', () => pair.num.select());
    pair.num.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        persistNow();
      }
    });
  });
}

function setBackgroundDisplay(value) {
  const hex = /^#[0-9a-f]{6}$/i.test(String(value || '')) ? String(value).toLowerCase() : '#0a0c11';
  backgroundEl.value = hex;
  bgOut.textContent = hex.toUpperCase();
  bgSwatches.querySelectorAll('.bg-swatch').forEach((swatch) => {
    swatch.classList.toggle('active', swatch.dataset.bg.toLowerCase() === hex);
  });
}

function syncOutputs() {
  document.documentElement.style.setProperty('--tint-alpha', String(pairOf.opacity.clamp(opacityRange.value) / 100));
}

function paintSwitchStates() {
  switchEls.forEach((input, idx) => {
    const out = stateEls[idx];
    if (!out) return;
    out.textContent = input.checked ? t('开') : t('关');
    out.classList.toggle('on', input.checked);
  });
}

function fitWindow() {
  App.fitSettingsWindow(0).catch(() => {});
}

function renderMonitors() {
  const current = monitorEl.value;
  monitorEl.innerHTML = '';
  const auto = document.createElement('option');
  auto.value = '-1';
  auto.textContent = t('主显示器');
  monitorEl.appendChild(auto);
  monitorList.forEach((m, idx) => {
    const opt = document.createElement('option');
    opt.value = String(m.Index);
    opt.textContent = `${t('显示器')} ${idx + 1} · ${m.Rect.W}×${m.Rect.H}${m.Primary ? t('（主）') : ''}`;
    monitorEl.appendChild(opt);
  });
  if (current) monitorEl.value = current;
}

function collect() {
  return {
    edge: edgeEl.value,
    panelWidth: pairOf.width.clamp(widthRange.value),
    peek: pairOf.peek.clamp(peekRange.value),
    targetMonitor: Number(monitorEl.value),
    hotkey: hotkeyEl.value.trim() || 'ctrl+alt+space',
    edgeEnabled: edgeEnabledEl.checked,
    autostart: autostartEl.checked,
    opacity: pairOf.opacity.clamp(opacityRange.value) / 100,
    background: backgroundEl.value,
    animations: !animationsEl.checked,
    clock12: clock12El.checked,
    showClock: saved ? saved.showClock : true,
    showCalendar: showCalendarEl.checked,
    hideCalendarFestivals: !calendarFestivalsEl.checked,
    alarmSound: saved ? saved.alarmSound : 'chime',
    countdownSound: saved ? saved.countdownSound : 'chime',
    launchColumns: pairOf.cols.clamp(launchColsEl.value),
    launchRows: pairOf.rows.clamp(launchRowsEl.value),
    launchIconSize: pairOf.icon.clamp(launchIconEl.value),
    hideLaunchNames: !launchNamesEl.checked,
    hideLaunch: !showLaunchEl.checked,
    language: languageEl.value,
  };
}

function schedulePreview() {
  syncOutputs();
  if (window.__debug) App.debugLog(`preview range=${widthRange.value} num=${widthNum.value}`);
  if (previewTimer) clearTimeout(previewTimer);
  previewTimer = setTimeout(() => {
    App.preview(collect()).catch(() => {});
  }, 120);
  schedulePersist();
}

function flashStatus(text) {
  statusEl.textContent = text;
  if (statusTimer) clearTimeout(statusTimer);
  statusTimer = setTimeout(() => {
    statusTimer = null;
    if (statusEl.textContent === text) statusEl.textContent = '';
  }, 1600);
}

function schedulePersist() {
  if (resetBusy) return;
  if (persistTimer) clearTimeout(persistTimer);
  persistTimer = setTimeout(() => {
    persistNow();
  }, 450);
}

async function persistNow() {
  if (resetBusy || suppressPersist) return;
  if (persistTimer) {
    clearTimeout(persistTimer);
    persistTimer = null;
  }
  const run = (async () => {
    try {
      const result = await App.apply(collect());
      if (result) saved = result;
      const loc = await resolveLocale((result && result.language) || languageEl.value);
      if (loc !== getLocale()) setLocale(loc);
      flashStatus(t('已自动保存'));
    } catch (err) {
      statusEl.textContent = t('保存失败：') + friendlyError(err);
    }
  })();
  persistPromise = run;
  try {
    await run;
  } finally {
    if (persistPromise === run) persistPromise = null;
  }
}

async function closeWindow() {
  if (resetBusy || restoreBusy) return;
  if (previewTimer) {
    clearTimeout(previewTimer);
    previewTimer = null;
  }
  if (persistTimer) {
    clearTimeout(persistTimer);
    persistTimer = null;
    await persistNow();
  } else if (persistPromise) {
    await persistPromise;
  }
  App.closeSettings().catch(() => {});
}

function openResetModal() {
  if (resetBusy) return;
  resetConfirmInput.value = '';
  resetConfirmInput.disabled = false;
  resetCancelBtn.disabled = false;
  resetConfirmBtn.textContent = t('确认重置');
  resetConfirmBtn.disabled = true;
  resetErrorEl.hidden = true;
  resetModal.hidden = false;
  resetConfirmInput.focus();
}

function closeResetModal() {
  if (resetBusy) return;
  resetModal.hidden = true;
}

function openRestoreDefaults() {
  if (!defaults || restoreBusy) return;
  restoreDefaultsModal.hidden = false;
}

function closeRestoreDefaults() {
  if (restoreBusy) return;
  restoreDefaultsModal.hidden = true;
}

async function restoreDefaultSettings() {
  if (!defaults || restoreBusy) return;
  restoreBusy = true;
  restoreDefaultsConfirmBtn.disabled = true;
  restoreDefaultsCancelBtn.disabled = true;
  try {
    saved = defaults;
    fillControls(defaults);
    if (usageAutoMasterEl && !usageAutoMasterEl.disabled) {
      usageAutoMasterEl.checked = false;
      await App.setUsageAutoAll(false).catch(() => {});
      paintSwitchStates();
    }
    await persistNow();
    restoreDefaultsModal.hidden = true;
    flashStatus(t('已恢复默认设置'));
  } finally {
    restoreBusy = false;
    restoreDefaultsConfirmBtn.disabled = false;
    restoreDefaultsCancelBtn.disabled = false;
  }
}

async function performReset() {
  if (resetBusy || resetConfirmInput.value.trim() !== t('完全重置')) return;
  resetBusy = true;
  if (previewTimer) {
    clearTimeout(previewTimer);
    previewTimer = null;
  }
  if (persistTimer) {
    clearTimeout(persistTimer);
    persistTimer = null;
  }
  if (persistPromise) {
    await persistPromise;
  }
  try {
    localStorage.clear();
  } catch (err) {}
  resetConfirmInput.disabled = true;
  resetCancelBtn.disabled = true;
  resetConfirmBtn.disabled = true;
  resetConfirmBtn.textContent = t('正在重置…');
  resetErrorEl.hidden = true;
  statusEl.textContent = t('正在重置…');
  try {
    await App.resetAll();
    statusEl.textContent = t('已重置，正在重启…');
    resetModal.hidden = true;
    setTimeout(() => App.closeSettings().catch(() => {}), 900);
  } catch (err) {
    resetBusy = false;
    resetConfirmInput.disabled = false;
    resetCancelBtn.disabled = false;
    resetConfirmBtn.textContent = t('确认重置');
    resetConfirmBtn.disabled = resetConfirmInput.value.trim() !== t('完全重置');
    resetErrorEl.textContent = friendlyError(err);
    resetErrorEl.hidden = false;
    statusEl.textContent = t('重置失败');
  }
}

function dataDirPathOf(mode, custom) {
  if (!dataDirInfo) return '';
  if (mode === 'portable') return dataDirInfo.portableDir;
  if (mode === 'custom') return String(custom || '').trim();
  return dataDirInfo.defaultDir;
}

function samePath(a, b) {
  const clean = (p) => String(p || '').replace(/[\\/]+$/, '').replace(/\//g, '\\').toLowerCase();
  const left = clean(a);
  return left !== '' && left === clean(b);
}

function sizeText(bytes) {
  if (bytes >= 1048576) return tp('{size} MB', { size: (bytes / 1048576).toFixed(1) });
  if (bytes >= 1024) return tp('{size} KB', { size: Math.round(bytes / 1024) });
  return tp('{size} B', { size: bytes });
}

function renderDataDir() {
  if (!dataDirInfo) {
    dataDirEffectiveEl.textContent = t('读取存储位置失败');
    return;
  }
  dataDirModeEl.value = dataDirInfo.mode;
  refreshSelect(dataDirModeEl);
  dataDirCustomEl.value = dataDirInfo.custom || '';
  dataDirCustomEl.disabled = dataDirInfo.mode !== 'custom';
  pickDataDirBtn.disabled = dataDirInfo.mode !== 'custom';
  const effectiveText = dataDirInfo.fellBack
    ? tp('{path}（自定义不可用，已回退）', { path: dataDirInfo.effective })
    : tp('当前生效：{path}', { path: dataDirInfo.effective });
  dataDirEffectiveEl.textContent = effectiveText;
  dataDirEffectiveEl.setAttribute('aria-label', effectiveText);
}

async function loadDataDir() {
  dataDirInfo = await App.dataLocation().catch(() => null);
  renderDataDir();
}

function openDataDirModal(mode, custom) {
  const target = dataDirPathOf(mode, custom);
  if (!target) return;
  dataDirPending = { mode, custom: mode === 'custom' ? target : '' };
  dataDirErrorEl.hidden = true;
  dataDirSummaryEl.textContent = tp('将从「{from}」复制到「{to}」，约 {size}。原位置保留不动，完成后自动重启侧边栏与本窗口。', {
    from: dataDirInfo.effective,
    to: target,
    size: sizeText(dataDirInfo.bytes),
  });
  dataDirConfirmBtn.disabled = false;
  dataDirCancelBtn.disabled = false;
  dataDirConfirmBtn.textContent = t('确认更改');
  dataDirModal.hidden = false;
}

function closeDataDirModal() {
  if (dataDirBusy) return;
  dataDirModal.hidden = true;
  dataDirPending = null;
  renderDataDir();
}

async function applyDataDir() {
  if (!dataDirPending || dataDirBusy) return;
  const { mode, custom } = dataDirPending;
  if (samePath(dataDirPathOf(mode, custom), dataDirInfo.effective)) {
    dataDirModal.hidden = true;
    dataDirPending = null;
    try {
      await App.setDataLocation(mode, custom);
      await loadDataDir();
      flashStatus(t('已自动保存'));
    } catch (err) {
      statusEl.textContent = friendlyError(err);
    }
    return;
  }
  dataDirBusy = true;
  dataDirConfirmBtn.disabled = true;
  dataDirCancelBtn.disabled = true;
  dataDirConfirmBtn.textContent = t('正在迁移…');
  try {
    await App.setDataLocation(mode, custom);
    suppressPersist = true;
    dataDirModal.hidden = true;
    statusEl.textContent = t('存储位置已更新，正在重启…');
    setTimeout(() => App.closeSettings().catch(() => {}), 1400);
  } catch (err) {
    dataDirErrorEl.textContent = friendlyError(err);
    dataDirErrorEl.hidden = false;
    dataDirConfirmBtn.disabled = false;
    dataDirCancelBtn.disabled = false;
    dataDirConfirmBtn.textContent = t('确认更改');
  } finally {
    dataDirBusy = false;
  }
}

function baseName(path) {
  const parts = String(path || '').split(/[\\/]/);
  return parts[parts.length - 1] || String(path || '');
}

function backupAvailableIds() {
  return backupMode === 'import' ? Array.from(backupCounts.keys()) : BACKUP_CATEGORIES.map((c) => c.id);
}

function syncBackupActions() {
  const empty = backupSelected.size === 0;
  backupConfirmBtn.disabled = backupBusy || empty;
  backupCancelBtn.disabled = backupBusy;
  backupAllBtn.disabled = backupBusy;
  backupNoneBtn.disabled = backupBusy;
}

function renderBackupList() {
  const ids = backupAvailableIds();
  backupListEl.innerHTML = '';
  BACKUP_CATEGORIES.filter((c) => ids.includes(c.id)).forEach((c) => {
    const item = document.createElement('label');
    item.className = 'backup-item';
    const box = document.createElement('input');
    box.type = 'checkbox';
    box.checked = backupSelected.has(c.id);
    box.addEventListener('change', () => {
      if (box.checked) backupSelected.add(c.id);
      else backupSelected.delete(c.id);
      syncBackupActions();
    });
    const text = document.createElement('span');
    text.className = 'backup-item-text';
    const name = document.createElement('span');
    name.className = 'backup-item-name';
    name.textContent = t(c.name);
    const desc = document.createElement('span');
    desc.className = 'backup-item-desc';
    desc.textContent = t(c.desc);
    text.append(name, desc);
    item.append(box, text);
    if (backupMode === 'import') {
      const count = document.createElement('span');
      count.className = 'backup-item-count';
      count.textContent = tp('{n} 个文件', { n: backupCounts.get(c.id) || 0 });
      item.append(count);
    }
    backupListEl.append(item);
  });
  syncBackupActions();
}

function renderBackupModal() {
  const importMode = backupMode === 'import';
  backupTitleEl.textContent = importMode ? t('导入备份') : t('导出备份');
  backupSummaryEl.textContent = importMode
    ? (backupLegacy
      ? tp('备份包含 {n} 类内容（旧版备份按文件识别）。只覆盖所选类别，其余数据保留。', { n: backupSelected.size })
      : tp('备份包含 {n} 类内容。只覆盖所选类别，其余数据保留。', { n: backupSelected.size }))
    : t('选择要写入备份的内容，未勾选的不会打包。');
  backupConfirmBtn.textContent = importMode ? t('确认导入') : t('选择文件夹并导出');
  renderBackupList();
}

function openBackupModal(mode) {
  backupMode = mode;
  backupErrorEl.hidden = true;
  renderBackupModal();
  backupModal.hidden = false;
}

function startBackup() {
  if (backupBusy) return;
  backupPath = '';
  backupLegacy = false;
  backupCounts = new Map();
  backupSelected = new Set(BACKUP_CATEGORIES.map((c) => c.id));
  openBackupModal('export');
}

async function pickRestoreFile() {
  if (backupBusy) return;
  const path = await App.pickBackupFile().catch(() => '');
  if (!path) return;
  try {
    const contents = await App.backupContents(path);
    backupPath = path;
    backupLegacy = !!contents.legacy;
    backupCounts = new Map((contents.categories || []).map((c) => [c.id, c.files]));
    backupSelected = new Set(backupCounts.keys());
    openBackupModal('import');
  } catch (err) {
    statusEl.textContent = friendlyError(err);
  }
}

function closeBackupModal() {
  if (backupBusy) return;
  backupModal.hidden = true;
  backupPath = '';
  backupSelected.clear();
}

async function confirmBackup() {
  if (backupBusy || backupSelected.size === 0) return;
  backupBusy = true;
  backupErrorEl.hidden = true;
  const label = backupConfirmBtn.textContent;
  backupConfirmBtn.textContent = t('正在处理…');
  syncBackupActions();
  try {
    if (backupMode === 'export') {
      const dir = await App.pickFolder(t('选择备份保存文件夹')).catch(() => '');
      if (!dir) return;
      const info = await App.createBackupTo(dir, Array.from(backupSelected));
      backupModal.hidden = true;
      backupHintEl.textContent = tp('已备份 {files} 项 · {size}', { files: info.files, size: sizeText(info.bytes) });
      flashStatus(tp('已生成 {name}', { name: baseName(info.path) }));
    } else {
      await App.restoreBackup(backupPath, Array.from(backupSelected));
      suppressPersist = true;
      backupModal.hidden = true;
      statusEl.textContent = t('数据已导入，正在重启…');
      setTimeout(() => App.closeSettings().catch(() => {}), 1400);
    }
  } catch (err) {
    backupErrorEl.textContent = friendlyError(err);
    backupErrorEl.hidden = false;
  } finally {
    backupBusy = false;
    backupConfirmBtn.textContent = label;
    syncBackupActions();
  }
}

function commitNow() {
  schedulePreview();
  persistNow();
}

async function init() {
  resetConfirmInput.placeholder = t('完全重置');
  bindAriaTip(dataDirEffectiveEl);
  let settings;
  let monitors = [];
  try {
    settings = await App.getSettings();
    monitors = (await App.getMonitors()) || [];
    defaults = await App.defaults();
  } catch (err) {
    statusEl.textContent = t('读取设置失败');
    return;
  }
  saved = settings;

  let screenW = 0;
  monitors.forEach((m) => {
    if (m.Rect && m.Rect.W > screenW) screenW = m.Rect.W;
  });
  maxWidth = Math.min(560, screenW > 240 ? screenW : 560);
  widthRange.max = String(maxWidth);
  widthNum.max = String(maxWidth);

  monitorList = monitors;
  renderMonitors();
  fillControls(settings);

  sidebarRunning = await App.sidebarRunning().catch(() => false);
  await initUsageAuto();
  await loadDataDir();
}

function fillControls(s) {
  setPairValue(pairOf.width, s.panelWidth);
  setPairValue(pairOf.opacity, (s.opacity ?? 0.8) * 100);
  setBackgroundDisplay(s.background);
  setPairValue(pairOf.peek, s.peek);
  edgeEl.value = s.edge || 'right';
  languageEl.value = s.language || 'auto';
  hotkeyEl.value = s.hotkey || 'ctrl+alt+space';
  edgeEnabledEl.checked = !!s.edgeEnabled;
  autostartEl.checked = !!s.autostart;
  animationsEl.checked = !s.animations;
  clock12El.checked = !!s.clock12;
  showCalendarEl.checked = s.showCalendar !== false;
  calendarFestivalsEl.checked = !s.hideCalendarFestivals;
  setPairValue(pairOf.cols, s.launchColumns);
  setPairValue(pairOf.rows, s.launchRows);
  setPairValue(pairOf.icon, s.launchIconSize);
  launchNamesEl.checked = !s.hideLaunchNames;
  showLaunchEl.checked = !s.hideLaunch;
  monitorEl.value = String(s.targetMonitor ?? -1);
  refreshSelect(edgeEl);
  refreshSelect(languageEl);
  refreshSelect(monitorEl);
  paintSwitchStates();
  syncOutputs();
}

bindPairs();
languageEl.addEventListener('change', persistNow);
backgroundEl.addEventListener('input', () => {
  bgOut.textContent = backgroundEl.value.toUpperCase();
  bgSwatches.querySelectorAll('.bg-swatch').forEach((swatch) => {
    swatch.classList.toggle('active', swatch.dataset.bg.toLowerCase() === backgroundEl.value);
  });
  schedulePreview();
});
backgroundEl.addEventListener('change', persistNow);
bgSwatches.addEventListener('click', (e) => {
  const swatch = e.target.closest('.bg-swatch');
  if (!swatch) return;
  setBackgroundDisplay(swatch.dataset.bg);
  commitNow();
});

bgSwatches.querySelectorAll('.bg-swatch').forEach((swatch) => bindAriaTip(swatch));
edgeEl.addEventListener('change', commitNow);
edgeEnabledEl.addEventListener('change', commitNow);
animationsEl.addEventListener('change', commitNow);
clock12El.addEventListener('change', commitNow);
showCalendarEl.addEventListener('change', commitNow);
calendarFestivalsEl.addEventListener('change', commitNow);
launchNamesEl.addEventListener('change', commitNow);
showLaunchEl.addEventListener('change', commitNow);
monitorEl.addEventListener('change', commitNow);
autostartEl.addEventListener('change', commitNow);
hotkeyEl.addEventListener('change', commitNow);
pages.settings.addEventListener('change', paintSwitchStates);

dataDirModeEl.addEventListener('change', () => {
  const mode = dataDirModeEl.value;
  if (mode === 'custom') {
    dataDirCustomEl.disabled = false;
    pickDataDirBtn.disabled = false;
    if (dataDirInfo && dataDirInfo.custom) openDataDirModal('custom', dataDirInfo.custom);
    else dataDirCustomEl.focus();
    return;
  }
  if (!dataDirInfo || mode === dataDirInfo.mode) return;
  openDataDirModal(mode, '');
});
dataDirCustomEl.addEventListener('change', () => {
  if (dataDirModeEl.value === 'custom') openDataDirModal('custom', dataDirCustomEl.value);
});
dataDirCustomEl.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && dataDirModeEl.value === 'custom') {
    e.preventDefault();
    openDataDirModal('custom', dataDirCustomEl.value);
  }
});
pickDataDirBtn.addEventListener('click', async () => {
  const path = await App.pickFolder(t('选择数据存放文件夹')).catch(() => '');
  if (!path) return;
  dataDirCustomEl.value = path;
  dataDirModeEl.value = 'custom';
  refreshSelect(dataDirModeEl);
  openDataDirModal('custom', path);
});
dataDirConfirmBtn.addEventListener('click', applyDataDir);
dataDirCancelBtn.addEventListener('click', closeDataDirModal);
dataDirModal.addEventListener('mousedown', (e) => {
  if (e.target === dataDirModal) closeDataDirModal();
});
backupNowBtn.addEventListener('click', startBackup);
restoreNowBtn.addEventListener('click', pickRestoreFile);
backupConfirmBtn.addEventListener('click', confirmBackup);
backupCancelBtn.addEventListener('click', closeBackupModal);
backupAllBtn.addEventListener('click', () => {
  backupSelected = new Set(backupAvailableIds());
  renderBackupList();
});
backupNoneBtn.addEventListener('click', () => {
  backupSelected.clear();
  renderBackupList();
});
backupModal.addEventListener('mousedown', (e) => {
  if (e.target === backupModal) closeBackupModal();
});

async function refreshUsageAuto() {
  if (!usageAutoMasterEl) return;
  const on = await App.usageAutoAll().catch(() => null);
  if (on === null) return;
  usageAutoMasterEl.checked = !!on;
  paintSwitchStates();
}

async function initUsageAuto() {
  if (!usageAutoMasterEl) return;
  await refreshUsageAuto();
  usageAutoMasterEl.disabled = !sidebarRunning;
  usageAutoMasterEl.addEventListener('change', async () => {
    const next = usageAutoMasterEl.checked;
    try {
      await App.setUsageAutoAll(next);
      flashStatus(t('已自动保存'));
    } catch (err) {
      usageAutoMasterEl.checked = !next;
      statusEl.textContent = friendlyError(err);
    }
  });
}

window.addEventListener('focus', () => {
  refreshUsageAuto();
});

function restoreWidth() {
  if (!defaults) return;
  setPairValue(pairOf.width, defaults.panelWidth);
  commitNow();
}

function restoreOpacity() {
  if (!defaults) return;
  setPairValue(pairOf.opacity, (defaults.opacity ?? 0.8) * 100);
  commitNow();
}

function restoreBackground() {
  if (!defaults) return;
  setBackgroundDisplay(defaults.background || '#0a0c11');
  commitNow();
}

function restorePeek() {
  if (!defaults) return;
  setPairValue(pairOf.peek, defaults.peek ?? 30);
  commitNow();
}

function restoreHotkey() {
  if (!defaults) return;
  hotkeyEl.value = defaults.hotkey || 'ctrl+alt+space';
  commitNow();
}

function restoreLaunchCols() {
  if (!defaults) return;
  setPairValue(pairOf.cols, defaults.launchColumns);
  commitNow();
}

function restoreLaunchRows() {
  if (!defaults) return;
  setPairValue(pairOf.rows, defaults.launchRows);
  commitNow();
}

function restoreLaunchIcon() {
  if (!defaults) return;
  setPairValue(pairOf.icon, defaults.launchIconSize);
  commitNow();
}

widthReset.addEventListener('click', restoreWidth);
opacityReset.addEventListener('click', restoreOpacity);
bgReset.addEventListener('click', restoreBackground);
peekReset.addEventListener('click', restorePeek);
hotkeyReset.addEventListener('click', restoreHotkey);
launchColsReset.addEventListener('click', restoreLaunchCols);
launchRowsReset.addEventListener('click', restoreLaunchRows);
launchIconReset.addEventListener('click', restoreLaunchIcon);
document.getElementById('sClose').addEventListener('click', closeWindow);
restoreSettingsBtn.addEventListener('click', openRestoreDefaults);
restoreDefaultsCancelBtn.addEventListener('click', closeRestoreDefaults);
restoreDefaultsConfirmBtn.addEventListener('click', restoreDefaultSettings);
restoreDefaultsModal.addEventListener('mousedown', (e) => {
  if (e.target === restoreDefaultsModal) closeRestoreDefaults();
});
document.getElementById('setDataDir').addEventListener('click', () => App.openDataDir().catch(() => {}));
resetAllBtn.addEventListener('click', openResetModal);
resetCancelBtn.addEventListener('click', closeResetModal);
resetModal.addEventListener('mousedown', (e) => {
  if (e.target === resetModal) closeResetModal();
});
resetConfirmInput.addEventListener('input', () => {
  resetConfirmBtn.disabled = resetBusy || resetConfirmInput.value.trim() !== t('完全重置');
});
resetConfirmInput.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && !resetConfirmBtn.disabled) {
    e.preventDefault();
    performReset();
  }
});
resetConfirmBtn.addEventListener('click', performReset);
document.addEventListener('keydown', (e) => {
  if (e.key !== 'Escape') return;
  if (!backupModal.hidden) closeBackupModal();
  else if (!restoreDefaultsModal.hidden) closeRestoreDefaults();
  else if (!dataDirModal.hidden) closeDataDirModal();
  else if (!resetModal.hidden) closeResetModal();
  else closeWindow();
});

function openLink(url) {
  if (url) App.openExternal(url).catch(() => {});
}

function friendlyError(err) {
  return String(err).replace(/^Error:\s*/, '');
}

function showUpdateSummary(info) {
  if (!info || !info.latest) return;
  latestInfo = info;
  updateStatusEl.textContent = tp('发现新版本 v{v}', { v: info.latest });
  if (!info.canInstall) updateStatusEl.textContent += t(' · 可前往发布页手动下载');
  updateLatestEl.textContent = tp('最新版本 v{v}', { v: info.latest });
  updateDateEl.textContent = info.publishedAt || '';
  updateNotesEl.textContent = info.notes || t('该版本没有提供更新说明。');
  updateDetailEl.hidden = false;
  installUpdateBtn.hidden = !info.canInstall;
}

async function loadAbout() {
  if (!aboutLoaded) {
    try {
      const info = await App.aboutInfo();
      aboutLoaded = true;
      links = {
        project: info.projectUrl || '',
        releases: info.releasesUrl || '',
        issues: info.issuesUrl || '',
      };
      aboutVersionEl.textContent = 'v' + info.version;
      aboutMetaEl.textContent = `v${info.version} · ${info.arch}`;
    } catch (err) {
      aboutVersionEl.textContent = '—';
      aboutMetaEl.textContent = t('版本信息不可用');
      updateStatusEl.textContent = t('无法读取版本信息');
    }
  }
  autoUpdateEl.checked = await App.getAutoUpdateCheck().catch(() => true);
  const notice = await App.updateNotice().catch(() => null);
  if (notice) showUpdateSummary(notice);
}

function syncTitle() {
  if (!titleEl) return;
  const active = pageButtons.find((item) => item.classList.contains('active'));
  titleEl.textContent = active ? active.textContent.trim() : t('设置');
}

function switchPage(name) {
  if (!pages[name]) return;
  pageButtons.forEach((item) => {
    item.classList.toggle('active', item.dataset.page === name);
  });
  Object.keys(pages).forEach((key) => {
    pages[key].hidden = key !== name;
  });
  document.querySelector('.win-body').scrollTop = 0;
  syncTitle();
  if (name === 'about') loadAbout();
}

async function checkUpdate() {
  checkUpdateBtn.disabled = true;
  updateStatusEl.textContent = t('正在检查更新…');
  updateDetailEl.hidden = true;
  updateConfirmEl.hidden = true;
  latestInfo = null;
  try {
    const info = await App.checkUpdate();
    latestInfo = info;
    if (!info.hasUpdate) {
      updateStatusEl.textContent = tp('已是最新版本（v{v}）', { v: info.current });
      return;
    }
    updateStatusEl.textContent = tp('发现新版本 v{v}', { v: info.latest });
    updateLatestEl.textContent = tp('最新版本 v{v}', { v: info.latest });
    updateDateEl.textContent = info.publishedAt || '';
    updateNotesEl.textContent = info.notes || t('该版本没有提供更新说明。');
    updateDetailEl.hidden = false;
    installUpdateBtn.hidden = !info.canInstall;
    if (!info.canInstall) {
      updateStatusEl.textContent += t(' · 发布包未提供校验文件，请前往发布页下载');
    }
  } catch (err) {
    updateStatusEl.textContent = friendlyError(err);
  } finally {
    checkUpdateBtn.disabled = false;
  }
}

async function installUpdate() {
  if (!latestInfo) return;
  updateConfirmEl.hidden = true;
  installUpdateBtn.hidden = true;
  checkUpdateBtn.disabled = true;
  updateStatusEl.textContent = t('正在下载并校验更新…');
  try {
    await App.installUpdate();
    updateStatusEl.textContent = t('更新完成，正在重启侧边栏…');
    setTimeout(() => App.closeSettings().catch(() => {}), 900);
  } catch (err) {
    updateStatusEl.textContent = friendlyError(err);
    checkUpdateBtn.disabled = false;
    if (latestInfo && latestInfo.canInstall) installUpdateBtn.hidden = false;
  }
}

[edgeEl, languageEl, monitorEl, dataDirModeEl].forEach(enhanceSelect);

function setActiveNav(id) {
  navItems.forEach((item) => item.classList.toggle('active', item.dataset.target === id));
}

navEl.addEventListener('click', (e) => {
  const page = e.target.closest('.s-nav-page');
  if (page) {
    switchPage(page.dataset.page);
    return;
  }
  const item = e.target.closest('.s-nav-item[data-target]');
  if (!item) return;
  if (pages.settings.hidden) switchPage('settings');
  setActiveNav(item.dataset.target);
  const section = document.getElementById(item.dataset.target);
  if (section) section.scrollIntoView({ behavior: 'smooth', block: 'start' });
});

const winBody = document.querySelector('.win-body');
let spyTick = false;
winBody.addEventListener('scroll', () => {
  if (spyTick || pages.settings.hidden) return;
  spyTick = true;
  requestAnimationFrame(() => {
    spyTick = false;
    const line = winBody.getBoundingClientRect().top + 60;
    let current = navSections[0] && navSections[0].id;
    for (const section of navSections) {
      if (section.getBoundingClientRect().top <= line) current = section.id;
    }
    setActiveNav(current);
  });
});

checkUpdateBtn.addEventListener('click', checkUpdate);
autoUpdateEl.addEventListener('change', () => {
  App.setAutoUpdateCheck(autoUpdateEl.checked).catch(() => {});
});
onEvent('settings:page', (page) => {
  if (typeof page === 'string') switchPage(page);
});
installUpdateBtn.addEventListener('click', () => {
  updateConfirmEl.hidden = false;
  installUpdateBtn.hidden = true;
});
updateConfirmNo.addEventListener('click', () => {
  updateConfirmEl.hidden = true;
  installUpdateBtn.hidden = false;
});
updateConfirmYes.addEventListener('click', installUpdate);
openReleaseBtn.addEventListener('click', () => {
  openLink((latestInfo && latestInfo.pageUrl) || links.releases);
});
openProjectBtn.addEventListener('click', () => openLink(links.project));
openIssuesBtn.addEventListener('click', () => openLink(links.issues));

App.isDebug()
  .then((v) => {
    window.__debug = !!v;
  })
  .catch(() => {});

onEvent('settings:changed', (next) => {
  if (!next || !next.language) return;
  App.locale()
    .then((loc) => {
      if (loc !== getLocale()) setLocale(loc);
    })
    .catch(() => {});
});

onLocaleChange(() => {
  resetConfirmInput.placeholder = t('完全重置');
  widthHint.textContent = widthHintText();
  syncOutputs();
  paintSwitchStates();
  renderDataDir();
  renderMonitors();
  syncTitle();
  if (!backupModal.hidden) renderBackupModal();
  if (latestInfo && !updateDetailEl.hidden) showUpdateSummary(latestInfo);
});

init()
  .then(() => {
    syncTitle();
    fitWindow();
    return App.startPage().catch(() => '');
  })
  .then((page) => {
    if (page) switchPage(page);
  })
  .catch(() => {
    statusEl.textContent = t('初始化失败');
  });
