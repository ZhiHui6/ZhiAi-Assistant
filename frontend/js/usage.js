import { Usage, App, el, svgIcon, toast, openContextMenu, onEvent, todayStr } from './api.js';
import { dataText, t, tp, onLocaleChange } from './strings.js';
import { bindAriaTip } from './tooltip.js';

const PLATFORMS = [
  { id: 'qoder', name: 'Qoder', letter: 'Q', color: '#6ea8ff', url: 'https://qoder.com.cn', usageUrl: 'https://qoder.cn/account/usage' },
  { id: 'workbuddy', name: 'WorkBuddy', letter: 'W', color: '#4ade80', url: 'https://www.workbuddy.cn', usageUrl: 'https://www.workbuddy.cn/profile/plans-usage' },
  { id: 'trae', name: 'Trae', letter: 'T', color: '#fbbf24', url: 'https://www.trae.cn', usageUrl: 'https://www.trae.cn/dashboard' },
];

const numberFormat = new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 2 });

function fmt(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return numberFormat.format(n);
}

function errText(err) {
  return dataText(String(err).replace(/^Error:\s*/, ''));
}

function sourceLabel(st) {
  if (st.authState !== 'ok') return t('未登录');
  if (st.source === 'browser') return t('浏览器登录');
  if (st.source === 'local') return t('客户端载入');
  return t('未登录');
}

function authMessage(st) {
  if (st.lastError) return dataText(st.lastError);
  if (st.signedOut) return t('已退出登录，可重新登录');
  if (st.authState === 'expired') return t('登录已过期，请重新登录');
  if (st.authState === 'encrypted') return t('登录信息已加密，请在客户端重新登录');
  if (st.authState === 'blocked') return t('平台拒绝了当前请求');
  if (st.authState === 'missing') return t('未找到客户端登录态');
  return t('尚未获取额度');
}

function updatedLabel(value) {
  if (!value) return { time: '', full: '' };
  const at = new Date(value);
  if (Number.isNaN(at.getTime())) return { time: '', full: '' };
  const pad = (n) => String(n).padStart(2, '0');
  const clock = pad(at.getHours()) + ':' + pad(at.getMinutes());
  const now = new Date();
  const sameDay = at.toDateString() === now.toDateString();
  const date = pad(at.getMonth() + 1) + '-' + pad(at.getDate());
  return { time: sameDay ? clock : date + ' ' + clock, full: date + ' ' + clock };
}

function dateOnly(value) {
  const text = String(value || '');
  if (!text) return '';
  return text.length > 10 ? text.slice(0, 10) : text;
}

export function createUsage() {
  const listEl = document.getElementById('usageList');
  const refreshAllBtn = document.getElementById('usageRefreshAll');
  const autoAllEl = document.getElementById('usageAutoAll');
  const autoAllStateEl = document.getElementById('usageAutoAllState');
  const checkinAllBtn = document.getElementById('usageCheckinAll');
  let state = { platforms: {}, autoCheckin: {}, noticeAcked: false };
  const accountFull = {};
  let modalEl = null;
  let refreshingAll = false;
  let checkingAll = false;
  let activated = false;
  let pollTimer = null;
  let activeLoginAbort = null;

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  function closeModal() {
    stopPolling();
    if (activeLoginAbort) {
      const abort = activeLoginAbort;
      activeLoginAbort = null;
      abort();
    }
    if (modalEl) {
      modalEl.remove();
      modalEl = null;
    }
  }

  function showModal(node) {
    closeModal();
    modalEl = node;
    document.body.appendChild(node);
  }

  function shell(title, body, actions, onMask) {
    const mask = el('div', 'modal-mask usage-modal-mask');
    mask.addEventListener('mousedown', (e) => {
      if (e.target === mask) {
        if (onMask) onMask();
        else closeModal();
      }
    });
    const card = el('div', 'modal-card usage-modal-card');
    card.appendChild(el('h3', '', title));
    body.forEach((node) => card.appendChild(node));
    card.appendChild(actions);
    mask.appendChild(card);
    return mask;
  }

  function actionRow() {
    return el('div', 'modal-actions');
  }

  function showNotice() {
    const text = el('p', 'modal-warn');
    text.textContent =
      t('这些平台没有公开的第三方额度接口。本功能复用本机客户端登录态读取额度并签到，属于非官方方式，可能随时失效或违反平台条款。');
    const text2 = el('p', 'modal-warn');
    text2.textContent =
      t('凭据只在本机读取；保存的登录凭据使用 Windows DPAPI 加密，不会回显、上传或写入普通设置文件。');
    const actions = actionRow();
    const cancel = el('button', 'btn ghost', t('取消'));
    const ok = el('button', 'btn primary', t('我已知晓'));
    cancel.addEventListener('click', closeModal);
    ok.addEventListener('click', async () => {
      ok.disabled = true;
      try {
        state = await Usage.ackNotice();
        closeModal();
        render();
        maybeRefresh();
      } catch (err) {
        ok.disabled = false;
        toast(errText(err));
      }
    });
    actions.appendChild(cancel);
    actions.appendChild(ok);
    showModal(shell(t('使用前确认'), [text, text2], actions, closeModal));
  }

  function ensureAck() {
    if (state.noticeAcked) return true;
    showNotice();
    return false;
  }

  async function openWorkBuddyLogin() {
    if (!ensureAck()) return;
    const hint = el('p', 'modal-warn');
    hint.textContent = t('将在浏览器打开 WorkBuddy 授权页。完成登录后会自动保存凭据并刷新额度，不会读取本机加密文件。');
    const status = el('p', 'usage-oauth-status', t('正在获取登录链接…'));
    const actions = actionRow();
    const cancel = el('button', 'btn ghost', t('取消'));
    cancel.addEventListener('click', closeModal);
    actions.appendChild(cancel);
    const mask = shell(t('WorkBuddy 浏览器登录'), [hint, status], actions, closeModal);
    showModal(mask);
    let startedAt = Date.now();
    let ttl = 10 * 60 * 1000;
    let busy = false;
    try {
      const start = await Usage.startWorkBuddyLogin();
      if (!start || !start.url) throw new Error('未获取到登录地址');
      if (start.expiresIn) ttl = Number(start.expiresIn) * 1000;
      startedAt = Date.now();
      await App.openExternal(start.url).catch(() => {});
      status.textContent = t('等待浏览器完成登录…');
      pollTimer = setInterval(async () => {
        if (busy || !modalEl) return;
        busy = true;
        try {
          const result = await Usage.pollWorkBuddyLogin(start.id);
          if (!result) return;
          if (result.done) {
            stopPolling();
            if (result.error) {
              status.textContent = result.error;
              cancel.textContent = t('关闭');
            } else {
              status.textContent = t('登录成功，正在刷新额度…');
              setTimeout(() => {
                closeModal();
                refreshOne('workbuddy');
              }, 700);
            }
          } else if (result.error) {
            status.textContent = result.error;
          }
          if (Date.now() - startedAt > ttl) {
            stopPolling();
            status.textContent = t('登录已超时，请重新发起');
          }
        } catch (err) {
          stopPolling();
          status.textContent = errText(err);
          cancel.textContent = t('关闭');
        } finally {
          busy = false;
        }
      }, 2000);
    } catch (err) {
      status.textContent = errText(err);
      cancel.textContent = t('关闭');
    }
  }

  async function openBrowserLogin(platform) {
    if (!ensureAck()) return;
    const hint = el('p', 'modal-warn');
    hint.textContent = tp('将打开独立浏览器窗口访问 {name} 登录页，登录成功后自动读取额度并关闭窗口。', { name: platform.name });
    const status = el('p', 'usage-oauth-status', t('正在启动登录窗口…'));
    const actions = actionRow();
    const cancel = el('button', 'btn ghost', t('取消'));
    actions.appendChild(cancel);
    let loginId = '';
    let busy = false;
    let startedAt = Date.now();
    let ttl = 5 * 60 * 1000;
    const abort = () => {
      stopPolling();
      if (loginId) {
        Usage.cancelBrowserLogin(loginId).catch(() => {});
        loginId = '';
      }
    };
    const close = () => {
      abort();
      activeLoginAbort = null;
      closeModal();
    };
    cancel.addEventListener('click', close);
    showModal(shell(tp('{name} 浏览器登录', { name: platform.name }), [hint, status], actions, close));
    try {
      const start = await Usage.startBrowserLogin(platform.id);
      if (!start || !start.id) throw new Error(t('未获取到登录会话'));
      loginId = start.id;
      activeLoginAbort = () => {
        if (loginId) Usage.cancelBrowserLogin(loginId).catch(() => {});
      };
      if (start.expiresIn) ttl = Number(start.expiresIn) * 1000;
      startedAt = Date.now();
      status.textContent = t('请在浏览器的登录窗口中完成登录…');
      pollTimer = setInterval(async () => {
        if (busy || !modalEl) return;
        busy = true;
        try {
          const result = await Usage.pollBrowserLogin(platform.id, loginId);
          if (!result) return;
          if (result.done) {
            stopPolling();
            activeLoginAbort = null;
            loginId = '';
            if (result.error) {
              status.textContent = dataText(result.error);
              cancel.textContent = t('关闭');
            } else {
              status.textContent = t('登录成功，正在刷新额度…');
              setTimeout(() => {
                closeModal();
                refreshOne(platform.id);
              }, 600);
            }
          } else if (result.error) {
            status.textContent = dataText(result.error);
          }
          if (Date.now() - startedAt > ttl) {
            stopPolling();
            status.textContent = t('登录已超时，请重新发起');
          }
        } catch (err) {
          stopPolling();
          status.textContent = errText(err);
          cancel.textContent = t('关闭');
        } finally {
          busy = false;
        }
      }, 1500);
    } catch (err) {
      status.textContent = errText(err);
      cancel.textContent = t('关闭');
    }
  }

  async function refreshAll(quiet) {
    if (refreshingAll) return;
    if (!ensureAck()) return;
    const loggedCount = PLATFORMS.filter((platform) => state.platforms[platform.id]?.authState === 'ok').length;
    if (!loggedCount) {
      if (!quiet) toast(t('请先登录后再刷新'));
      return;
    }
    const skipped = PLATFORMS.length - loggedCount;
    refreshingAll = true;
    refreshAllBtn.disabled = true;
    refreshAllBtn.textContent = t('刷新中…');
    try {
      state = await Usage.refreshAll();
      render();
      if (skipped > 0 && !quiet) toast(tp('{n} 个平台未登录，已跳过', { n: skipped }));
    } catch (err) {
      if (!quiet) toast(errText(err));
    } finally {
      refreshingAll = false;
      refreshAllBtn.disabled = false;
      refreshAllBtn.textContent = t('刷新全部');
    }
  }

  async function refreshOne(id) {
    const stateOne = await Usage.refresh(id).catch((err) => {
      toast(errText(err));
      return null;
    });
    if (stateOne) {
      state = stateOne;
      render();
    }
  }

  async function checkin(platform) {
    if (!ensureAck()) return;
    try {
      state = await Usage.checkin(platform.id);
      render();
      const st = state.platforms[platform.id] || {};
      if (st.lastError) toast(st.lastError);
      else toast(t('签到成功'));
    } catch (err) {
      toast(errText(err));
    }
  }

  function pendingCheckins() {
    const today = todayStr();
    return PLATFORMS.filter((platform) => {
      const st = state.platforms[platform.id];
      if (!st || st.authState !== 'ok') return false;
      if (st.lastCheckinDate === today) return false;
      const snap = st.snapshot || {};
      return !snap.checkedIn;
    });
  }

  async function checkinAll() {
    if (checkingAll) return;
    if (!ensureAck()) return;
    const targets = pendingCheckins();
    if (!targets.length) {
      toast(t('暂无可签到平台'));
      return;
    }
    checkingAll = true;
    checkinAllBtn.disabled = true;
    checkinAllBtn.textContent = t('签到中…');
    try {
      state = await Usage.checkinAll();
      render();
      const done = targets.filter((platform) => {
        const st = state.platforms[platform.id] || {};
        return st.lastCheckinDate === todayStr() || (st.snapshot || {}).checkedIn;
      }).length;
      if (done > 0) {
        toast(tp('已签到 {n} 个平台', { n: done }));
      } else {
        const failed = targets.map((platform) => state.platforms[platform.id]).find((st) => st && st.lastError);
        toast(failed ? dataText(failed.lastError) : t('签到失败'));
      }
    } catch (err) {
      toast(errText(err));
    } finally {
      checkingAll = false;
      checkinAllBtn.disabled = false;
      checkinAllBtn.textContent = t('一键签到');
    }
  }

  async function openClient(platform) {
    try {
      await Usage.openClient(platform.id);
      toast(tp('已启动 {name}', { name: platform.name }));
    } catch (err) {
      toast(errText(err));
    }
  }

  async function reread(platform) {
    if (!ensureAck()) return;
    try {
      state = await Usage.reread(platform.id);
      render();
      const st = state.platforms[platform.id] || {};
      if (st.authState !== 'ok') toast(authMessage(st));
    } catch (err) {
      toast(errText(err));
    }
  }

  function confirmSignOut(platform) {
    const hint = el('p', 'modal-warn');
    hint.textContent = t('退出后会删除本机保存的凭据，并清空已读取的额度；本机客户端自身的登录状态不受影响。');
    const error = el('p', 'modal-error');
    error.hidden = true;
    const actions = actionRow();
    const cancel = el('button', 'btn ghost', t('取消'));
    const ok = el('button', 'btn danger', t('确认退出'));
    cancel.addEventListener('click', closeModal);
    ok.addEventListener('click', async () => {
      ok.disabled = true;
      cancel.disabled = true;
      try {
        state = await Usage.signOut(platform.id);
        closeModal();
        render();
        toast(tp('已退出 {name}', { name: platform.name }));
      } catch (err) {
        ok.disabled = false;
        cancel.disabled = false;
        error.textContent = errText(err);
        error.hidden = false;
      }
    });
    actions.appendChild(cancel);
    actions.appendChild(ok);
    showModal(shell(tp('退出 {name} 登录', { name: platform.name }), [hint, error], actions, closeModal));
  }

  function cardMenu(platform, st, event) {
    event.preventDefault();
    event.stopPropagation();
    const items = [
      { label: t('打开官网'), icon: 'i-right', action: () => App.openExternal(platform.url).catch(() => {}) },
      { label: t('打开客户端'), icon: 'i-play', action: () => openClient(platform) },
    ];
    if (st.source === 'browser' || st.source === 'local') {
      items.push({ sep: true });
      items.push({
        label: t('退出登录'),
        icon: 'i-signout',
        danger: true,
        action: () => confirmSignOut(platform),
      });
    }
    openContextMenu(event.clientX, event.clientY, items);
  }

  function renderCard(platform) {
    const st = state.platforms[platform.id] || {
      snapshot: {},
      source: 'none',
      authState: 'missing',
    };
    const snap = st.snapshot || {};
    const card = el('div', 'usage-card');
    card.style.setProperty('--brand', platform.color);

    const head = el('div', 'usage-card-head');
    const logo = el('div', 'usage-logo', platform.letter);
    const title = el('div', 'usage-title');
    const nameRow = el('div', 'usage-name');
    nameRow.appendChild(el('span', 'usage-platform', platform.name));
    nameRow.appendChild(el('span', 'usage-source', '[' + sourceLabel(st) + ']'));
    title.appendChild(nameRow);
    if (snap.account) {
      const accountRow = el('div', 'usage-account');
      accountRow.appendChild(el('span', 'usage-account-text', accountFull[platform.id] || snap.account));
      const eye = el('button', 'icon-btn sm usage-account-eye');
      eye.appendChild(svgIcon(accountFull[platform.id] ? 'i-eye-off' : 'i-eye'));
      eye.setAttribute('aria-label', t(accountFull[platform.id] ? '隐藏账号' : '显示完整账号'));
      eye.title = t(accountFull[platform.id] ? '隐藏账号' : '显示完整账号');
      eye.addEventListener('click', async () => {
        if (accountFull[platform.id]) {
          delete accountFull[platform.id];
          render();
          return;
        }
        try {
          const full = await Usage.accountFull(platform.id);
          if (full) accountFull[platform.id] = full;
          render();
        } catch (err) {
          toast(errText(err));
        }
      });
      accountRow.appendChild(eye);
      title.appendChild(accountRow);
    }
    const refresh = el('button', 'icon-btn sm usage-refresh');
    refresh.appendChild(svgIcon('i-refresh'));
    refresh.setAttribute('aria-label', t('刷新'));
    refresh.addEventListener('click', () => refreshOne(platform.id));
    head.appendChild(logo);
    head.appendChild(title);
    if (!st.signedOut && (st.source === 'browser' || st.source === 'local')) {
      const signout = el('button', 'icon-btn sm usage-signout');
      signout.appendChild(svgIcon('i-signout'));
      signout.setAttribute('aria-label', t('退出登录'));
      signout.title = t('退出登录');
      signout.addEventListener('click', () => confirmSignOut(platform));
      head.appendChild(signout);
    }
    head.appendChild(refresh);
    card.appendChild(head);

    card.addEventListener('contextmenu', (e) => cardMenu(platform, st, e));

    const total = Number(snap.total) || 0;
    const remaining = Number(snap.remaining) || 0;
    const percent = Math.max(0, Math.min(1, Number(snap.percent) || (total > 0 ? (total - remaining) / total : 0)));

    if (total > 0) {
      const bar = el('div', 'usage-bar');
      const fill = el('i');
      fill.style.width = Math.round(percent * 100) + '%';
      bar.appendChild(fill);
      card.appendChild(bar);

      const nums = el('div', 'usage-nums');
      nums.appendChild(el('span', 'usage-remain', fmt(remaining)));
      const usageClick = el('span', 'usage-nums-click');
      nums.appendChild(el('span', 'usage-total-sep', '/'));
      usageClick.appendChild(el('span', 'usage-total', `${fmt(total)} ${snap.unit || ''}`));
      usageClick.title = t('打开官网查询用量');
      usageClick.addEventListener('click', () => App.openExternal(platform.usageUrl || platform.url).catch(() => {}));
      nums.appendChild(usageClick);
      nums.appendChild(el('span', 'usage-pct', Math.round(percent * 100) + '%'));
      card.appendChild(nums);

      const meta = el('div', 'usage-meta');
      if (snap.plan) meta.appendChild(el('span', '', snap.plan));
      if (snap.resetAt) meta.appendChild(el('span', '', t('重置 ') + dateOnly(snap.resetAt)));
      card.appendChild(meta);
    } else {
      if (st.authState === 'ok') {
        card.appendChild(el('div', 'usage-empty', authMessage(st)));
      }
      const actions = el('div', 'usage-actions');
      const loginBtn = el('button', 'btn sm primary', t('浏览器登录'));
      loginBtn.addEventListener('click', () =>
        platform.id === 'workbuddy' ? openWorkBuddyLogin() : openBrowserLogin(platform),
      );
      actions.appendChild(loginBtn);
      const rereadBtn = el('button', 'btn sm ghost', t('客户端载入'));
      rereadBtn.addEventListener('click', () => reread(platform));
      actions.appendChild(rereadBtn);
      card.appendChild(actions);
    }

    const check = el('div', 'usage-check');
    if (snap.checkedIn) {
      check.appendChild(el('span', 'usage-badge', t('今日已签到')));
    } else if (snap.checkinAvailable && st.authState === 'ok') {
      const label = snap.dailyCredit ? tp('签到 +{n}', { n: fmt(snap.dailyCredit) }) : t('签到');
      const btn = el('button', 'btn sm primary', label);
      btn.addEventListener('click', () => checkin(platform));
      check.appendChild(btn);
    } else if (total > 0) {
      check.appendChild(el('span', 'usage-empty', dataText(snap.message) || t('暂无可领取签到')));
    }
    if (total > 0) {
      check.classList.add('has-time');
      const updated = updatedLabel(snap.updatedAt);
      const updatedEl = el('span', 'usage-updated');
      if (updated.time) {
        updatedEl.appendChild(el('span', 'usage-updated-label', t('更新于')));
        updatedEl.appendChild(el('span', 'usage-updated-time', updated.time));
        updatedEl.title = updated.full;
      } else {
        updatedEl.textContent = t('尚未刷新');
      }
      check.appendChild(updatedEl);
    }
    if (check.childNodes.length) card.appendChild(check);

    if (st.lastError && total > 0) {
      card.appendChild(el('div', 'usage-error', st.lastError));
    }
    return card;
  }

  function render() {
    listEl.innerHTML = '';
    PLATFORMS.forEach((platform) => listEl.appendChild(renderCard(platform)));
    syncAutoAll();
  }

  function syncAutoAll() {
    const ids = PLATFORMS.map((platform) => platform.id);
    const on = ids.filter((id) => !!state.autoCheckin[id]).length;
    const stateText = on === 0 ? t('自动签到已关闭') : on === ids.length ? t('自动签到已开启') : t('自动签到部分开启');
    autoAllStateEl.textContent = stateText;
    autoAllEl.dataset.state = on === 0 ? 'off' : on === ids.length ? 'on' : 'part';
    autoAllEl.setAttribute('aria-label', tp('{state}，可在设置中修改', { state: stateText }));
  }

  function maybeRefresh() {
    if (!state.noticeAcked) return;
    const now = Date.now();
    const stale = PLATFORMS.filter((platform) => {
      const st = state.platforms[platform.id];
      if (!st || st.authState !== 'ok') return false;
      if (!st.lastFetchAt) return true;
      const at = Date.parse(st.lastFetchAt);
      return !Number.isFinite(at) || now - at > 10 * 60 * 1000;
    });
    if (!stale.length) return;
    (async () => {
      for (const platform of stale) {
        await refreshOne(platform.id);
      }
    })();
  }

  async function activate() {
    try {
      state = await Usage.list();
    } catch (err) {
      toast(t('读取用量失败'));
    }
    render();
    if (!state.noticeAcked) {
      showNotice();
      return;
    }
    if (!activated) {
      activated = true;
      maybeRefresh();
    }
  }

  refreshAllBtn.addEventListener('click', () => refreshAll(false));
  checkinAllBtn.addEventListener('click', checkinAll);
  bindAriaTip(autoAllEl);
  onLocaleChange(syncAutoAll);

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && modalEl) {
      e.stopPropagation();
      closeModal();
    }
  });

  onEvent('usage:changed', (next) => {
    if (!next) return;
    state = next;
    render();
  });

  return { refresh: activate };
}
