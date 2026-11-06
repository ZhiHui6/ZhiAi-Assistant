import { Calendar, el, pad2, todayStr, toast } from './api.js';
import { dataText, t, tp } from './strings.js';

const WEEK = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

let tipEl = null;
let tipTimer = null;

function hideTip() {
  if (tipTimer) {
    clearTimeout(tipTimer);
    tipTimer = null;
  }
  if (tipEl) tipEl.hidden = true;
}

function ensureTip() {
  if (!tipEl) {
    tipEl = el('div', 'cal-tip');
    tipEl.setAttribute('role', 'tooltip');
    tipEl.hidden = true;
    document.body.appendChild(tipEl);
  }
  return tipEl;
}

function showTip(btn, text) {
  hideTip();
  tipTimer = setTimeout(() => {
    tipTimer = null;
    const tip = ensureTip();
    tip.textContent = text;
    tip.hidden = false;
    tip.style.left = '0px';
    tip.style.top = '0px';
    const box = btn.getBoundingClientRect();
    const tw = tip.offsetWidth;
    const th = tip.offsetHeight;
    let left = box.left + box.width / 2 - tw / 2;
    left = Math.max(6, Math.min(left, window.innerWidth - tw - 6));
    let top = box.top - th - 6;
    if (top < 6) top = box.bottom + 6;
    top = Math.min(top, window.innerHeight - th - 6);
    tip.style.left = left + 'px';
    tip.style.top = top + 'px';
  }, 100);
}

export function createCalendar({ onSelect }) {
  const titleEl = document.getElementById('calTitle');
  const gridEl = document.getElementById('calGrid');
  const today = new Date();
  let year = today.getFullYear();
  let month = today.getMonth();
  let selected = null;
  let summary = new Map();
  let festivals = {};
  let showDetails = true;

  function monthArg(offset) {
    const d = new Date(year, month + offset, 1);
    return { year: d.getFullYear(), month: d.getMonth() + 1 };
  }

  async function load() {
    const prev = monthArg(-1);
    const next = monthArg(1);
    const [list, cur, before, after] = await Promise.all([
      Calendar.month(year, month + 1).catch(() => []),
      Calendar.festivals(year, month + 1).catch(() => ({})),
      Calendar.festivals(prev.year, prev.month).catch(() => ({})),
      Calendar.festivals(next.year, next.month).catch(() => ({})),
    ]);
    summary = new Map();
    (list || []).forEach((s) => summary.set(s.date, s));
    festivals = Object.assign({}, before || {}, cur || {}, after || {});
    render();
  }

  function render() {
    hideTip();
    const card = document.getElementById('calendarCard');
    if (card) card.classList.toggle('compact', !showDetails);
    titleEl.textContent = tp('{y}年{m}月', { y: year, m: month + 1 });
    gridEl.innerHTML = '';
    const firstWeekday = (new Date(year, month, 1).getDay() + 6) % 7;
    const daysInMonth = new Date(year, month + 1, 0).getDate();
    const totalCells = Math.ceil((firstWeekday + daysInMonth) / 7) * 7;
    const todayKey = todayStr();

    for (let i = 0; i < totalCells; i++) {
      const dayNum = i - firstWeekday + 1;
      const inMonth = dayNum >= 1 && dayNum <= daysInMonth;
      const dateObj = new Date(year, month, dayNum);
      const cellDate = todayStr(dateObj);
      const btn = el('button', 'cal-day');
      if (!inMonth) btn.classList.add('other');
      if (cellDate === todayKey) btn.classList.add('today');
      if (cellDate === selected) btn.classList.add('selected');
      btn.appendChild(el('span', 'cal-num', String(dateObj.getDate())));

      const fest = festivals[cellDate];
      if (fest) {
        let festEl = null;
        if (fest.label && showDetails) {
          festEl = el('span', 'cal-fest cal-fest-' + (fest.kind || 'lunar'), dataText(fest.label));
          btn.appendChild(festEl);
        }
        if (fest.holiday) btn.classList.add('is-holiday');
        const detail = dataText(fest.detail || fest.label);
        if (detail) {
          const text = `${tp('{m}月{d}日', { m: dateObj.getMonth() + 1, d: dateObj.getDate() })} ${t(WEEK[dateObj.getDay()])} · ${detail}`;
          const needsTip = () => !showDetails || (festEl && festEl.scrollWidth > festEl.clientWidth + 1);
          btn.addEventListener('mouseenter', () => {
            if (needsTip()) showTip(btn, text);
          });
          btn.addEventListener('mouseleave', hideTip);
          btn.addEventListener('focus', () => {
            if (needsTip()) showTip(btn, text);
          });
          btn.addEventListener('blur', hideTip);
        }
      }

      const info = summary.get(cellDate);
      if (info && (info.open > 0 || info.done > 0)) {
        const dots = el('span', 'dots');
        if (info.overdue > 0) dots.appendChild(el('i', 'dot overdue'));
        else if (info.open > 0) dots.appendChild(el('i', 'dot'));
        if (info.done > 0) dots.appendChild(el('i', 'dot done'));
        btn.appendChild(dots);
      }

      btn.addEventListener('click', () => {
        if (selected === cellDate) {
          selected = null;
          onSelect && onSelect(null);
        } else {
          selected = cellDate;
          onSelect && onSelect(cellDate);
        }
        render();
      });
      gridEl.appendChild(btn);
    }
  }

  function shiftMonth(delta) {
    month += delta;
    if (month < 0) {
      month = 11;
      year -= 1;
    } else if (month > 11) {
      month = 0;
      year += 1;
    }
    load();
  }

  document.getElementById('calPrev').addEventListener('click', () => shiftMonth(-1));
  document.getElementById('calNext').addEventListener('click', () => shiftMonth(1));
  document.getElementById('calToday').addEventListener('click', () => {
    const now = new Date();
    year = now.getFullYear();
    month = now.getMonth();
    selected = todayStr();
    onSelect && onSelect(selected);
    load();
  });

  load();

  window.addEventListener('resize', hideTip);

  return {
    refresh: load,
    setShowDetails(show) {
      const next = show !== false;
      if (next === showDetails) return;
      showDetails = next;
      hideTip();
      render();
    },
    select(dateStr) {
      selected = dateStr;
      if (dateStr) {
        const d = new Date(dateStr + 'T00:00:00');
        year = d.getFullYear();
        month = d.getMonth();
      }
      load();
    },
    clear() {
      selected = null;
      render();
    },
  };
}
