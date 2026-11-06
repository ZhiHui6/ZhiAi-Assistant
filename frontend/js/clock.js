import { pad2 } from './api.js';
import { t, tp } from './strings.js';

const WEEK = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

export function initClock() {
  const timeEl = document.getElementById('clockTime');
  const subEl = document.getElementById('clockSub');
  let timer = null;
  let suffix = '';
  let clock12 = false;

  function render() {
    const now = new Date();
    const h = now.getHours();
    const minutes = pad2(now.getMinutes());
    if (clock12) {
      const half = h < 12 ? t('上午') : t('下午');
      const h12 = h % 12 === 0 ? 12 : h % 12;
      timeEl.innerHTML = `<span class="clock-half">${half}</span>${h12}:${minutes}`;
    } else {
      timeEl.textContent = `${pad2(h)}:${minutes}`;
    }
    subEl.textContent = `${tp('{y}年{m}月{d}日', { y: now.getFullYear(), m: now.getMonth() + 1, d: now.getDate() })} · ${t(WEEK[now.getDay()])}${suffix}`;
    if (timer) clearTimeout(timer);
    timer = setTimeout(render, 1000 - (Date.now() % 1000) + 8);
  }

  render();
  document.addEventListener('visibilitychange', render);
  return {
    refresh: render,
    set12h(on) {
      const next = !!on;
      if (next === clock12) return;
      clock12 = next;
      render();
    },
    setSuffix(text) {
      suffix = text ? ' · ' + text : '';
      render();
    },
  };
}

export function weekdayName(dateStr) {
  const d = new Date(dateStr + 'T00:00:00');
  return WEEK[d.getDay()];
}
