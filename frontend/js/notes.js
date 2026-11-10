import { Notes, debounce, el, relTime, svgIcon, toast } from './api.js';
import { onLocaleChange, t } from './strings.js';

const PARAGRAPH_MARK = '\n· ';
const TAG_RE = /(^|\s)#([\p{L}\p{N}_-]+)/gu;
const SAVE_DELAY = 450;

const NOTE_TINTS = [
  { tint: '', label: '默认' },
  { tint: '#2d5b90', label: '藏蓝' },
  { tint: '#2a6a54', label: '墨绿' },
  { tint: '#26707c', label: '墨青' },
  { tint: '#5b4680', label: '紫夜' },
  { tint: '#7d5a38', label: '暖褐' },
  { tint: '#8c4455', label: '绛红' },
  { tint: '#4b5672', label: '黛灰' },
  { tint: '#72662f', label: '橄榄' },
];

const LEGACY_TINTS = {
  '#152742': '#2d5b90',
  '#22456e': '#2d5b90',
  '#12312b': '#2a6a54',
  '#1d4a3c': '#2a6a54',
  '#0f3237': '#26707c',
  '#1a5058': '#26707c',
  '#2b1d3f': '#5b4680',
  '#3f3160': '#5b4680',
  '#33241a': '#7d5a38',
  '#4d3823': '#7d5a38',
  '#3a1d26': '#8c4455',
  '#552a36': '#8c4455',
  '#23283a': '#4b5672',
  '#363d58': '#4b5672',
  '#2e2a14': '#72662f',
  '#453f1e': '#72662f',
};

function parseTags(text) {
  const tags = [];
  for (const m of text.matchAll(TAG_RE)) tags.push(m[2]);
  return tags;
}

function autoGrow(ta) {
  ta.style.overflowY = 'hidden';
  ta.style.height = 'auto';
  ta.style.height = ta.scrollHeight + 1 + 'px';
  ta.style.overflowY = '';
}

function applyCardTint(card, tint) {
  const match = /^#([0-9a-f]{6})$/i.exec(tint || '');
  if (!match) {
    card.classList.remove('tinted');
    card.style.removeProperty('--note-rgb');
    return;
  }
  const n = parseInt(match[1], 16);
  card.classList.add('tinted');
  card.style.setProperty('--note-rgb', `${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}`);
}

function markTint(pop, tint) {
  pop.querySelectorAll('.tint-swatch').forEach((sw) => {
    sw.classList.toggle('active', (sw.dataset.tint || '') === (tint || ''));
  });
}

function insertParagraphMark(ta) {
  ta.setRangeText(PARAGRAPH_MARK, ta.selectionStart, ta.selectionEnd, 'end');
  ta.dispatchEvent(new Event('input', { bubbles: true }));
}

export function createNotes() {
  const addBtn = document.getElementById('noteAdd');
  const searchInput = document.getElementById('noteSearch');
  const clearBtn = document.getElementById('noteClear');
  const listEl = document.getElementById('noteList');

  let items = [];
  let keyword = '';
  const pending = new Set();

  function track(promise) {
    pending.add(promise);
    promise.finally(() => pending.delete(promise));
    return promise;
  }

  function cardNode(id) {
    return [...listEl.children].find((c) => c.dataset.noteId === id) || null;
  }

  function renderEmpty() {
    listEl.innerHTML = '';
    listEl.appendChild(el('div', 'empty', keyword ? t('没有匹配的随记') : t('还没有随记\n点「新建」记一条')));
  }

  function addToItems(note) {
    if (items.some((n) => n.id === note.id)) return;
    const pinnedCount = items.filter((n) => n.pinned).length;
    items = [...items.slice(0, pinnedCount), note, ...items.slice(pinnedCount)];
  }

  async function removeNote(note) {
    try {
      await Notes.remove(note.id);
    } catch (err) {
      toast(t('删除失败'));
      return;
    }
    items = items.filter((n) => n.id !== note.id);
    const card = cardNode(note.id);
    if (card) card.remove();
    if (!listEl.children.length) renderEmpty();
  }

  async function togglePin(note) {
    if (!items.some((n) => n.id === note.id)) return;
    try {
      await Notes.update(note.id, null, null, !note.pinned);
      await load();
    } catch (err) {
      toast(t('操作失败'));
    }
  }

  async function applyTint(note, tint, card, pop) {
    try {
      const saved = await Notes.setTint(note.id, tint);
      note.tint = saved.tint || '';
      applyCardTint(card, note.tint);
      markTint(pop, note.tint);
    } catch (err) {
      toast(t('保存失败'));
    }
  }

  function cardFor(note) {
    const ref = { note };
    const card = el('div', 'card');
    card.dataset.noteId = note.id;
    const row = el('div', 'card-row');
    const ta = el('textarea', 'note-text');
    ta.value = note.text;
    if (!note.text) ta.placeholder = t('随手记一条…（Ctrl+Enter 分段）');
    ta.rows = 1;
    ta.spellcheck = false;

    const timeEl = el('span', '', relTime(note.updatedAt || note.createdAt));
    const actions = el('div', 'card-actions');
    const pinBtn = el('button', 'icon-btn sm');
    pinBtn.classList.toggle('active', !!note.pinned);
    pinBtn.title = t(note.pinned ? '取消置顶' : '置顶');
    pinBtn.appendChild(svgIcon('i-tack'));
    pinBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (ref.note) togglePin(ref.note);
    });
    const delBtn = el('button', 'icon-btn sm note-del');
    delBtn.title = t('删除');
    delBtn.appendChild(svgIcon('i-trash'));
    delBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      clearTimer();
      if (ref.note) removeNote(ref.note);
    });
    actions.append(pinBtn, delBtn);

    const tintPop = el('div', 'card-tint');
    NOTE_TINTS.forEach(({ tint, label }) => {
      const btn = el('button', tint ? 'tint-swatch' : 'tint-swatch tint-default');
      btn.type = 'button';
      btn.dataset.tint = tint;
      btn.setAttribute('aria-label', t(label));
      if (tint) {
        const n = parseInt(tint.slice(1), 16);
        btn.style.setProperty('--sw-rgb', `${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}`);
      }
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        if (!ref.note) return;
        applyTint(ref.note, tint, card, tintPop);
      });
      tintPop.appendChild(btn);
    });
    if (note) {
      applyCardTint(card, note.tint);
      markTint(tintPop, note.tint);
    }

    const meta = el('div', 'card-meta card-meta-view');
    const tools = el('div', 'card-hover-tools');
    tools.append(tintPop, actions);
    meta.append(timeEl, tools);
    const tagBar = el('div', 'card-meta card-meta-edit');
    const tagHash = el('span', 'tag-hash');
    tagHash.appendChild(svgIcon('i-hash'));
    const chips = el('div', 'tag-chips');
    const tagInput = el('input', 'tag-input');
    tagInput.type = 'text';
    tagInput.spellcheck = false;
    tagInput.placeholder = t('添加标签，回车确认');
    tagBar.append(tagHash, chips, tagInput);

    const viewTags = el('div', 'note-tags-view');
    meta.insertBefore(viewTags, tools);

    row.append(ta);
    card.append(row, tagBar, meta);

    function renderTags() {
      chips.innerHTML = '';
      viewTags.innerHTML = '';
      const tags = ref.note.tags || [];
      tags.forEach((tag) => {
        const chip = el('span', 'tag-chip');
        chip.appendChild(el('span', '', '#' + tag));
        const remove = el('button', 'tag-x', '×');
        remove.type = 'button';
        remove.setAttribute('aria-label', t('移除标签'));
        remove.addEventListener('mousedown', (e) => e.preventDefault());
        remove.addEventListener('click', () => {
          saveTags((ref.note.tags || []).filter((item) => item !== tag));
        });
        chip.appendChild(remove);
        chips.appendChild(chip);

        viewTags.appendChild(el('span', 'tag-chip', '#' + tag));
      });
      viewTags.hidden = !tags.length;
    }

    async function saveTags(tags) {
      try {
        ref.note = await Notes.update(ref.note.id, null, tags, null);
        renderTags();
      } catch (err) {
        toast(t('保存失败'));
      }
    }

    function commitTag() {
      const raw = tagInput.value.trim().replace(/^#+/, '').toLowerCase();
      if (!raw) return;
      tagInput.value = '';
      const tags = ref.note.tags || [];
      if (tags.includes(raw)) return;
      saveTags([...tags, raw]);
    }

    tagInput.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter') return;
      e.preventDefault();
      commitTag();
    });
    tagInput.addEventListener('blur', commitTag);

    function enterEdit() {
      card.classList.add('editing');
    }

    ta.addEventListener('focus', enterEdit);
    tagInput.addEventListener('focus', enterEdit);
    card.addEventListener('focusout', (e) => {
      const next = e.relatedTarget;
      if (next === ta || next === tagInput) return;
      card.classList.remove('editing');
    });

    renderTags();

    let timer = null;

    function clearTimer() {
      if (!timer) return;
      clearTimeout(timer);
      timer = null;
    }

    async function commit() {
      const text = ta.value;
      if (text === ref.note.text) return;
      const merged = Array.from(new Set([...(ref.note.tags || []), ...parseTags(text)]));
      try {
        ref.note = await Notes.update(ref.note.id, text, merged, null);
        timeEl.textContent = relTime(ref.note.updatedAt);
        renderTags();
      } catch (err) {
        if (cardNode(ref.note.id)) toast(t('保存失败'));
      }
    }

    ta.addEventListener('input', () => {
      autoGrow(ta);
      clearTimer();
      timer = setTimeout(() => {
        timer = null;
        track(commit());
      }, SAVE_DELAY);
    });
    ta.addEventListener('blur', () => {
      clearTimer();
      track(commit());
    });
    ta.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter') return;
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        insertParagraphMark(ta);
        return;
      }
      if (e.shiftKey) return;
      e.preventDefault();
      ta.blur();
    });

    return { card, ta, commit };
  }

  async function addNote() {
    let saved;
    try {
      saved = await Notes.create('', []);
    } catch (err) {
      toast(t('保存失败'));
      return;
    }
    addToItems(saved);
    render();
    const card = cardNode(saved.id);
    const ta = card ? card.querySelector('.note-text') : null;
    if (ta) ta.focus();
  }

  function render() {
    listEl.innerHTML = '';
    if (!items.length) {
      renderEmpty();
      return;
    }
    items.forEach((note) => {
      const { card } = cardFor(note);
      listEl.appendChild(card);
      autoGrow(card.querySelector('.note-text'));
    });
  }

  async function load() {
    if (pending.size) await Promise.all([...pending]);
    try {
      items = await Notes.list(keyword, 0);
    } catch (err) {
      items = [];
      toast(t('读取随记失败'));
    }
    items.forEach((it) => {
      const next = LEGACY_TINTS[(it.tint || '').toLowerCase()];
      if (!next) return;
      it.tint = next;
      Notes.setTint(it.id, next).catch(() => {});
    });
    render();
  }

  function syncLabels() {
    addBtn.setAttribute('aria-label', t('新建随记'));
    searchInput.placeholder = t('搜索记事与标签');
    clearBtn.textContent = t('清除');
  }

  syncLabels();
  onLocaleChange(syncLabels);

  addBtn.addEventListener('click', addNote);

  const doSearch = debounce(async () => {
    keyword = searchInput.value.trim();
    clearBtn.hidden = keyword === '';
    await load();
  }, 180);

  searchInput.addEventListener('input', doSearch);
  clearBtn.addEventListener('click', () => {
    searchInput.value = '';
    keyword = '';
    clearBtn.hidden = true;
    load();
  });

  load();

  return { refresh: load };
}
