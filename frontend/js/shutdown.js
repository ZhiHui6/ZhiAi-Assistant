import { Timers, fmtDuration, fmtTarget, toast } from './api.js';
import { dataText, t, tp } from './strings.js';

function errText(err) {
  return dataText(String(err).replace(/^Error:\s*/, ''));
}

export function createShutdown() {
  const active = document.getElementById('shutdownActive');
  const activeText = document.getElementById('shutdownActiveText');
  const cancelBtn = document.getElementById('shutdownCancel');
  const minuteInput = document.getElementById('shutdownMin');
  const byMinutes = document.getElementById('shutdownByMinutes');
  const atInput = document.getElementById('shutdownAt');
  const byTime = document.getElementById('shutdownByTime');
  const confirm = document.getElementById('shutdownConfirm');
  const confirmText = document.getElementById('shutdownConfirmText');
  const confirmOk = document.getElementById('shutdownConfirmBtn');
  const confirmCancel = document.getElementById('shutdownConfirmCancel');

  let plan = {};
  let pending = 0;

  function renderActive() {
    const at = plan.active && plan.at ? Date.parse(plan.at) : NaN;
    if (Number.isNaN(at)) {
      active.hidden = true;
      return;
    }
    const left = Math.max(0, Math.ceil((at - Date.now()) / 1000));
    activeText.textContent = tp('已安排 {time} 关机（还剩 {left}）', { time: fmtTarget(at), left: fmtDuration(left) });
    active.hidden = false;
  }

  function renderConfirm() {
    if (!pending) {
      confirm.hidden = true;
      return;
    }
    const minutes = Math.max(1, Math.round((pending - Date.now()) / 60000));
    confirmText.textContent = tp('将在 {time} 关闭电脑，约 {n} 分钟后。', { time: fmtTarget(pending), n: minutes });
    confirm.hidden = false;
  }

  async function refresh() {
    try {
      const state = await Timers.state();
      plan = (state && state.shutdown) || {};
    } catch (err) {
      toast(t('读取关机计划失败'));
      return;
    }
    renderActive();
  }

  function tick() {
    if (active.hidden || !plan.active || !plan.at) return;
    const at = Date.parse(plan.at);
    if (Number.isNaN(at)) return;
    if (at <= Date.now()) {
      plan = {};
      active.hidden = true;
      return;
    }
    renderActive();
  }

  byMinutes.addEventListener('click', () => {
    const minutes = Number(minuteInput.value) || 0;
    if (minutes < 1) {
      toast(t('请输入关机倒计时分钟数'));
      return;
    }
    pending = Date.now() + minutes * 60000;
    renderConfirm();
  });

  byTime.addEventListener('click', () => {
    const value = atInput.value;
    if (!value) {
      toast(t('请选择关机时间'));
      return;
    }
    const parts = value.split(':').map(Number);
    const target = new Date();
    target.setHours(parts[0] || 0, parts[1] || 0, 0, 0);
    if (target.getTime() <= Date.now()) target.setDate(target.getDate() + 1);
    pending = target.getTime();
    renderConfirm();
  });

  confirmCancel.addEventListener('click', () => {
    pending = 0;
    renderConfirm();
  });

  confirmOk.addEventListener('click', async () => {
    if (!pending) return;
    try {
      await Timers.scheduleShutdown(Math.floor(pending / 1000));
      pending = 0;
      renderConfirm();
      await refresh();
      toast(t('已安排系统关机'));
    } catch (err) {
      toast(errText(err));
    }
  });

  cancelBtn.addEventListener('click', async () => {
    try {
      await Timers.cancelShutdown();
      await refresh();
      toast(t('已取消关机计划'));
    } catch (err) {
      toast(errText(err));
    }
  });

  setInterval(tick, 1000);
  refresh();

  return { refresh };
}
