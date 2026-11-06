import { App, Timers, el, fmtDuration, onEvent, svgIcon, toast } from './api.js';
import { dataText, t } from './strings.js';
import { enhanceSelect } from './select.js';

const SOUND_OPTIONS = [
  ['chime', '清脆铃'],
  ['bell', '大钟'],
  ['marimba', '木琴'],
  ['arpeggio', '三连音'],
  ['dawn', '晨光'],
  ['pulse', '脉冲'],
  ['drop', '水滴'],
  ['chord', '和弦'],
  ['wood', '木鱼'],
  ['dingdong', '叮咚'],
  ['none', '静音'],
];

const LEGACY_SOUNDS = {
  exclaim: 'chime',
  asterisk: 'dingdong',
  hand: 'pulse',
  question: 'arpeggio',
  ok: 'chord',
};

function normalizeSound(value) {
  if (value && value.startsWith('file:')) return value;
  if (LEGACY_SOUNDS[value]) return LEGACY_SOUNDS[value];
  if (SOUND_OPTIONS.some(([v]) => v === value)) return value;
  return 'chime';
}

function addCustomOption(selectEl, stored) {
  const path = stored.slice(5);
  const name = path.split(/[\\/]/).pop() || path;
  const option = document.createElement('option');
  option.value = stored;
  option.textContent = t('自定义：') + name;
  const choose = selectEl.querySelector('option[value="__choose__"]');
  selectEl.insertBefore(option, choose || null);
}

function setupSound(kind, selectEl, testBtn, initial) {
  let saved = normalizeSound(initial);
  selectEl.innerHTML = '';
  SOUND_OPTIONS.forEach(([value, label]) => {
    const option = document.createElement('option');
    option.value = value;
    option.textContent = t(label);
    selectEl.appendChild(option);
  });
  if (saved.startsWith('file:')) addCustomOption(selectEl, saved);
  const choose = document.createElement('option');
  choose.value = '__choose__';
  choose.textContent = t('自定义 WAV…');
  selectEl.appendChild(choose);
  selectEl.value = saved;

  selectEl.addEventListener('change', async () => {
    const value = selectEl.value;
    if (value === '__choose__') {
      try {
        const path = await App.pickSound();
        if (!path) throw new Error('未选择文件');
        const stored = 'file:' + path;
        Array.from(selectEl.options).forEach((option) => {
          if (option.value.startsWith('file:')) option.remove();
        });
        addCustomOption(selectEl, stored);
        selectEl.value = stored;
        await App.setTimerSound(kind, stored);
        saved = stored;
      } catch (err) {
        if (!/取消|未选择/.test(String(err))) toast(errText(err));
        selectEl.value = saved;
      }
      return;
    }
    try {
      await App.setTimerSound(kind, value);
      saved = value;
    } catch (err) {
      toast(errText(err));
      selectEl.value = saved;
    }
  });

  testBtn.addEventListener('click', () => {
    const value = selectEl.value;
    if (!value || value === '__choose__') return;
    App.previewTimerSound(value).catch(() => toast(t('无法播放该铃音')));
  });
}

function errText(err) {
  return dataText(String(err).replace(/^Error:\s*/, ''));
}

export function createTimers() {
  const alertEl = document.getElementById('timerAlert');
  const alertText = document.getElementById('timerAlertText');
  const alertClose = document.getElementById('timerAlertClose');
  const alarmTime = document.getElementById('alarmTime');
  const alarmLabel = document.getElementById('alarmLabel');
  const alarmRepeat = document.getElementById('alarmRepeat');
  const alarmAdd = document.getElementById('alarmAdd');
  const alarmList = document.getElementById('alarmList');
  const alarmSound = document.getElementById('alarmSound');
  const alarmSoundTest = document.getElementById('alarmSoundTest');
  const countdownSound = document.getElementById('countdownSound');
  const countdownSoundTest = document.getElementById('countdownSoundTest');
  enhanceSelect(alarmSound);
  enhanceSelect(countdownSound);
  const countdownValue = document.getElementById('countdownValue');
  const countdownPresets = document.getElementById('countdownPresets');
  const countdownHour = document.getElementById('countdownHour');
  const countdownMin = document.getElementById('countdownMin');
  const countdownSec = document.getElementById('countdownSec');
  const countdownToggle = document.getElementById('countdownToggle');
  const countdownReset = document.getElementById('countdownReset');

  let state = { alarms: [], countdown: {} };

  function countdownRemaining() {
    const c = state.countdown || {};
    if (c.running && c.endAt) {
      const end = Date.parse(c.endAt);
      if (!Number.isNaN(end)) return Math.max(0, Math.ceil((end - Date.now()) / 1000));
    }
    return Math.max(0, Number(c.remaining) || 0);
  }

  function updateLive() {
    const remain = countdownRemaining();
    countdownValue.textContent = fmtDuration(remain);
    const running = !!state.countdown?.running;
    countdownToggle.textContent = running ? t('暂停') : remain > 0 ? t('继续') : t('开始');
  }

  function renderAlarms() {
    alarmList.innerHTML = '';
    const alarms = state.alarms || [];
    if (!alarms.length) {
      alarmList.appendChild(el('div', 'empty', t('还没有闹钟')));
      return;
    }
    alarms.forEach((alarm) => {
      const row = el('div', 'alarm-row');
      row.appendChild(el('span', 'alarm-time', alarm.time));
      row.appendChild(el('span', 'alarm-label', alarm.label || t('闹钟')));
      row.appendChild(el('span', 'alarm-repeat', alarm.repeat ? t('每天') : t('一次')));

      const toggle = el('input');
      toggle.type = 'checkbox';
      toggle.checked = !!alarm.enabled;
      toggle.setAttribute('aria-label', t('启用闹钟'));
      toggle.addEventListener('change', async () => {
        try {
          await Timers.toggleAlarm(alarm.id, toggle.checked);
          await refresh();
        } catch (err) {
          toast(errText(err));
          toggle.checked = !toggle.checked;
        }
      });
      row.appendChild(toggle);

      const del = el('button', 'icon-btn sm');
      del.appendChild(svgIcon('i-trash'));
      del.setAttribute('aria-label', t('删除闹钟'));
      del.addEventListener('click', async () => {
        try {
          await Timers.removeAlarm(alarm.id);
          await refresh();
        } catch (err) {
          toast(errText(err));
        }
      });
      row.appendChild(del);
      alarmList.appendChild(row);
    });
  }

  function render() {
    renderAlarms();
    updateLive();
  }

  async function refresh() {
    try {
      state = await Timers.state();
    } catch (err) {
      toast(t('读取计时设置失败'));
      return;
    }
    render();
  }

  alarmAdd.addEventListener('click', async () => {
    const at = alarmTime.value;
    if (!at) {
      toast(t('请选择闹钟时间'));
      return;
    }
    try {
      await Timers.addAlarm(at, alarmLabel.value.trim(), alarmRepeat.checked);
      alarmLabel.value = '';
      await refresh();
    } catch (err) {
      toast(errText(err));
    }
  });

  countdownPresets.addEventListener('click', (e) => {
    const chip = e.target.closest('.chip');
    if (!chip) return;
    const total = Number(chip.dataset.min) || 0;
    countdownHour.value = String(Math.floor(total / 60));
    countdownMin.value = String(total % 60);
    countdownSec.value = '0';
  });

  countdownToggle.addEventListener('click', async () => {
    try {
      if (state.countdown?.running) {
        await Timers.pauseCountdown();
      } else if (countdownRemaining() > 0) {
        await Timers.resumeCountdown();
      } else {
        const hours = Number(countdownHour.value) || 0;
        const minutes = Number(countdownMin.value) || 0;
        const seconds = Number(countdownSec.value) || 0;
        const total = Math.floor(hours * 3600 + minutes * 60 + seconds);
        if (total < 1) {
          toast(t('请设置倒计时时长'));
          return;
        }
        await Timers.startCountdown(total, '');
      }
      await refresh();
    } catch (err) {
      toast(errText(err));
    }
  });

  countdownReset.addEventListener('click', async () => {
    try {
      await Timers.resetCountdown();
      await refresh();
    } catch (err) {
      toast(errText(err));
    }
  });

  alertClose.addEventListener('click', () => {
    alertEl.hidden = true;
  });

  onEvent('timer:fired', async (payload) => {
    if (payload?.label) {
      alertText.textContent = payload.label;
      alertEl.hidden = false;
    }
    toast(payload?.label || t('计时结束'));
    await refresh();
  });

  async function initSounds() {
    let settings = null;
    try {
      settings = await App.getSettings();
    } catch (err) {}
    setupSound('alarm', alarmSound, alarmSoundTest, settings?.alarmSound);
    setupSound('countdown', countdownSound, countdownSoundTest, settings?.countdownSound);
  }

  initSounds();
  setInterval(updateLive, 1000);
  refresh();

  return { refresh };
}
