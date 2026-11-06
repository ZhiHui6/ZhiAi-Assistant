const ENHANCED = new WeakMap();

function optionText(option) {
  return (option.textContent || '').replace(/\s+/g, ' ').trim();
}

export function enhanceSelect(select) {
  if (!select || ENHANCED.has(select)) return;
  select.classList.add('sel-native');

  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'sel-btn';
  btn.setAttribute('aria-haspopup', 'listbox');
  btn.setAttribute('aria-expanded', 'false');
  const label = document.createElement('span');
  label.className = 'sel-label';
  const caret = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  caret.setAttribute('class', 'icon sel-caret');
  caret.innerHTML = '<use href="#i-right"/>';
  btn.append(label, caret);
  select.after(btn);

  const state = { select, btn, label, menu: null, items: [], open: false, index: -1 };

  function syncLabel() {
    const option = select.options[select.selectedIndex];
    label.textContent = option ? optionText(option) : '';
    btn.disabled = !!select.disabled;
  }

  function rebuild() {
    if (!state.menu) return;
    state.menu.innerHTML = '';
    state.items = [];
    [...select.options].forEach((option, index) => {
      const item = document.createElement('button');
      item.type = 'button';
      item.className = 'sel-item';
      item.setAttribute('role', 'option');
      item.setAttribute('aria-selected', index === select.selectedIndex ? 'true' : 'false');
      if (index === select.selectedIndex) item.classList.add('active');
      const check = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
      check.setAttribute('class', 'icon');
      check.innerHTML = '<use href="#i-check"/>';
      const text = document.createElement('span');
      text.className = 'sel-item-text';
      text.textContent = optionText(option);
      item.append(check, text);
      item.addEventListener('click', (event) => {
        event.stopPropagation();
        choose(index);
      });
      item.addEventListener('mouseenter', () => setIndex(index));
      state.menu.appendChild(item);
      state.items.push(item);
    });
    setIndex(select.selectedIndex);
  }

  function setIndex(index) {
    state.index = index;
    state.items.forEach((item, i) => item.classList.toggle('focus', i === index));
  }

  function refresh() {
    syncLabel();
    if (state.open) rebuild();
  }

  function place() {
    const rect = btn.getBoundingClientRect();
    const menu = state.menu;
    const width = Math.min(Math.max(rect.width, 168), 360);
    menu.style.width = width + 'px';
    menu.style.left = Math.max(8, Math.min(rect.left, innerWidth - width - 8)) + 'px';
    const height = Math.min(menu.scrollHeight, 280);
    const below = innerHeight - rect.bottom - 8;
    if (below < height && rect.top > below) {
      menu.style.top = Math.max(8, rect.top - height - 6) + 'px';
    } else {
      menu.style.top = rect.bottom + 6 + 'px';
    }
  }

  function onDocDown(event) {
    if (state.menu && state.menu.contains(event.target)) return;
    if (btn.contains(event.target)) return;
    close();
  }

  function onKey(event) {
    if (event.key === 'Escape') {
      event.preventDefault();
      close();
      btn.focus();
      return;
    }
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      const step = event.key === 'ArrowDown' ? 1 : -1;
      const next = Math.min(state.items.length - 1, Math.max(0, state.index + step));
      setIndex(next);
      state.items[next]?.scrollIntoView({ block: 'nearest' });
      return;
    }
    if (event.key === 'Enter' && state.open) {
      event.preventDefault();
      if (state.index >= 0) choose(state.index);
    }
  }

  function open() {
    if (state.open || btn.disabled) return;
    if (!state.menu) {
      const menu = document.createElement('div');
      menu.className = 'sel-menu';
      menu.setAttribute('role', 'listbox');
      document.body.appendChild(menu);
      state.menu = menu;
    }
    rebuild();
    state.open = true;
    btn.setAttribute('aria-expanded', 'true');
    state.menu.classList.add('open');
    place();
    document.addEventListener('mousedown', onDocDown, true);
    document.addEventListener('keydown', onKey, true);
    window.addEventListener('resize', place);
    window.addEventListener('scroll', close, true);
  }

  function close() {
    if (!state.open) return;
    state.open = false;
    btn.setAttribute('aria-expanded', 'false');
    state.menu.classList.remove('open');
    document.removeEventListener('mousedown', onDocDown, true);
    document.removeEventListener('keydown', onKey, true);
    window.removeEventListener('resize', place);
    window.removeEventListener('scroll', close, true);
  }

  function choose(index) {
    if (index < 0 || index >= select.options.length) return;
    select.selectedIndex = index;
    syncLabel();
    close();
    select.dispatchEvent(new Event('input', { bubbles: true }));
    select.dispatchEvent(new Event('change', { bubbles: true }));
    btn.focus();
  }

  btn.addEventListener('click', () => (state.open ? close() : open()));
  btn.addEventListener('keydown', (event) => {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      if (!state.open) open();
    }
  });

  new MutationObserver(() => refresh()).observe(select, {
    childList: true,
    subtree: true,
    characterData: true,
  });

  ENHANCED.set(select, { refresh, close });
  refresh();
}

export function refreshSelect(select) {
  ENHANCED.get(select)?.refresh();
}
