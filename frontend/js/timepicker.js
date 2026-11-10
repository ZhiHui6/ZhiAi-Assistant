import { t } from './strings.js';

const ENHANCED = new WeakMap();

const pad = (n) => String(n).padStart(2, '0');

function parseValue(value) {
  const m = /^(\d{1,2}):(\d{1,2})/.exec(value || '');
  if (!m) return null;
  const h = Number(m[1]);
  const min = Number(m[2]);
  if (h > 23 || min > 59) return null;
  return { h, m: min };
}

function clockIcon() {
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('class', 'icon tp-icon');
  svg.setAttribute('viewBox', '0 0 16 16');
  svg.innerHTML = '<circle cx="8" cy="8" r="6.1"/><path d="M8 4.7v3.5l2.3 1.4"/>';
  return svg;
}

export function enhanceTime(input, options) {
  if (!(input instanceof Element) || ENHANCED.has(input)) return;
  const iconOnly = !!(options && options.iconOnly);
  input.classList.add('tp-native');

  const label = document.createElement('span');
  label.className = 'tp-label';

  let btn;
  let box;
  if (iconOnly) {
    box = document.createElement('span');
    box.className = 'tp-field';
    btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'tp-trigger';
    btn.append(clockIcon());
    box.append(label, btn);
    input.after(box);
  } else {
    btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'tp-btn';
    btn.append(label, clockIcon());
    input.after(btn);
    box = btn;
  }
  btn.setAttribute('aria-haspopup', 'dialog');
  btn.setAttribute('aria-expanded', 'false');

  const state = { open: false, pop: null, hourCol: null, minCol: null, hourItems: [], minItems: [], h: 8, m: 0 };

  function syncLabel() {
    const parsed = parseValue(input.value);
    label.textContent = parsed ? `${pad(parsed.h)}:${pad(parsed.m)}` : '-- : --';
    box.classList.toggle('tp-empty', !parsed);
  }

  function current() {
    return parseValue(input.value) || { h: state.h, m: state.m };
  }

  function mark(h, m) {
    state.hourItems.forEach((item, i) => item.classList.toggle('active', i === h));
    state.minItems.forEach((item, i) => item.classList.toggle('active', i === m));
  }

  function commit(h, m) {
    input.value = `${pad(h)}:${pad(m)}`;
    state.h = h;
    state.m = m;
    syncLabel();
    mark(h, m);
    input.dispatchEvent(new Event('input', { bubbles: true }));
    input.dispatchEvent(new Event('change', { bubbles: true }));
  }

  function makeItem(text, onPick) {
    const item = document.createElement('button');
    item.type = 'button';
    item.className = 'tp-item';
    item.textContent = text;
    item.addEventListener('click', (event) => {
      event.stopPropagation();
      onPick();
    });
    return item;
  }

  function build() {
    const pop = document.createElement('div');
    pop.className = 'tp-pop';
    pop.setAttribute('role', 'dialog');

    const head = document.createElement('div');
    head.className = 'tp-head';
    const hourLabel = document.createElement('span');
    hourLabel.textContent = t('时');
    const minLabel = document.createElement('span');
    minLabel.textContent = t('分');
    head.append(hourLabel, minLabel);

    const cols = document.createElement('div');
    cols.className = 'tp-cols';
    state.hourCol = document.createElement('div');
    state.hourCol.className = 'tp-col';
    state.minCol = document.createElement('div');
    state.minCol.className = 'tp-col';

    for (let i = 0; i < 24; i += 1) {
      const item = makeItem(pad(i), () => commit(i, current().m));
      state.hourItems.push(item);
      state.hourCol.appendChild(item);
    }
    for (let i = 0; i < 60; i += 1) {
      const item = makeItem(pad(i), () => {
        commit(current().h, i);
        close(true);
      });
      state.minItems.push(item);
      state.minCol.appendChild(item);
    }

    cols.append(state.hourCol, state.minCol);
    pop.append(head, cols);
    document.body.appendChild(pop);
    state.pop = pop;
  }

  function scrollToActive() {
    const center = (col, items) => {
      const active = items.find((item) => item.classList.contains('active'));
      if (!active) return;
      col.scrollTop = active.offsetTop - col.clientHeight / 2 + active.offsetHeight / 2;
    };
    center(state.hourCol, state.hourItems);
    center(state.minCol, state.minItems);
  }

  function place() {
    const rect = box.getBoundingClientRect();
    const pop = state.pop;
    const width = 168;
    pop.style.width = width + 'px';
    pop.style.left = Math.max(8, Math.min(rect.left, innerWidth - width - 8)) + 'px';
    const height = Math.min(pop.scrollHeight, 236);
    const below = innerHeight - rect.bottom - 8;
    if (below < height && rect.top > below) {
      pop.style.top = Math.max(8, rect.top - height - 6) + 'px';
    } else {
      pop.style.top = rect.bottom + 6 + 'px';
    }
  }

  function onDocDown(event) {
    if (state.pop && state.pop.contains(event.target)) return;
    if (btn.contains(event.target)) return;
    close();
  }

  function onKey(event) {
    if (event.key === 'Escape') {
      event.preventDefault();
      close();
      btn.focus();
    }
  }

  function onScroll(event) {
    if (state.pop && state.pop.contains(event.target)) return;
    close();
  }

  function open() {
    if (state.open) return;
    if (!state.pop) build();
    const parsed = parseValue(input.value);
    const now = new Date();
    state.h = parsed ? parsed.h : now.getHours();
    state.m = parsed ? parsed.m : now.getMinutes();
    mark(state.h, state.m);
    state.open = true;
    btn.setAttribute('aria-expanded', 'true');
    state.pop.classList.add('open');
    place();
    scrollToActive();
    document.addEventListener('mousedown', onDocDown, true);
    document.addEventListener('keydown', onKey, true);
    window.addEventListener('resize', place);
    window.addEventListener('scroll', onScroll, true);
  }

  function close(refocus) {
    if (!state.open) return;
    state.open = false;
    btn.setAttribute('aria-expanded', 'false');
    state.pop.classList.remove('open');
    document.removeEventListener('mousedown', onDocDown, true);
    document.removeEventListener('keydown', onKey, true);
    window.removeEventListener('resize', place);
    window.removeEventListener('scroll', onScroll, true);
    if (refocus) btn.focus();
  }

  btn.addEventListener('click', () => (state.open ? close() : open()));
  btn.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' || event.key === ' ' || event.key === 'ArrowDown') {
      event.preventDefault();
      if (!state.open) open();
    }
  });

  syncLabel();
  ENHANCED.set(input, { syncLabel, close });
}
