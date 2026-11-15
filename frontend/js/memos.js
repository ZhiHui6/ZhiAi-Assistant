import { Memos, debounce, el, fmtSize, openContextMenu, relTime, svgIcon, toast } from './api.js';
import { onLocaleChange, t, tp } from './strings.js';
import { setTipPreview, hideTip } from './tooltip.js';

const PARAGRAPH_MARK = '\n· ';
const TAG_RE = /(^|\s)#([\p{L}\p{N}_-]+)/gu;
const SAVE_DELAY = 450;
const PREVIEW_FRAME = 30;
const PREVIEW_FPS = 30;
const PREVIEW_WIDTH = 208;
const PREVIEW_HEIGHT = 117;
const PREVIEW_TEXT_CHARS = 800;
const PREVIEW_TEXT_BYTES = 256 * 1024;
const BYTE_UPLOAD_LIMIT = 32 * 1024 * 1024;
const PASTE_EXT = {
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/gif': 'gif',
  'image/webp': 'webp',
  'image/bmp': 'bmp',
  'image/svg+xml': 'svg',
  'video/mp4': 'mp4',
  'video/webm': 'webm',
  'audio/mpeg': 'mp3',
  'audio/wav': 'wav',
  'text/plain': 'txt',
};
let pathDropStamp = 0;

const MEMO_TINTS = [
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

const ATTACH_ICONS = {
  image: 'i-image',
  audio: 'i-music',
  video: 'i-video',
  text: 'i-file',
  link: 'i-link',
  file: 'i-file',
};

function errText(err) {
  return String(err).replace(/^Error:\s*/, '');
}

function mediaURL(path) {
  return '/memo-media?p=' + encodeURIComponent(path);
}

function base64Of(bytes) {
  let out = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    out += String.fromCharCode.apply(null, bytes.subarray(i, i + chunk));
  }
  return btoa(out);
}

function payloadName(file) {
  const name = file.name || '';
  if (/\.[a-z0-9]{1,8}$/i.test(name)) return name;
  const ext = PASTE_EXT[file.type] || 'bin';
  return 'paste-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 6) + '.' + ext;
}

function clipboardFiles(event) {
  const data = event.clipboardData;
  if (!data) return [];
  const files = [];
  for (const item of Array.from(data.items || [])) {
    if (item.kind !== 'file') continue;
    const file = item.getAsFile();
    if (file) files.push(file);
  }
  if (!files.length && data.files) files.push(...Array.from(data.files));
  return files;
}

function hasFileDrag(event) {
  const types = event.dataTransfer && event.dataTransfer.types;
  return !!types && Array.from(types).includes('Files');
}

function previewFrame() {
  const box = el('div', 'tip-preview-media');
  box.style.width = PREVIEW_WIDTH + 'px';
  box.style.height = PREVIEW_HEIGHT + 'px';
  box.style.overflow = 'hidden';
  box.style.borderRadius = '7px';
  box.style.background = 'rgba(255, 255, 255, 0.04)';
  return box;
}

function fillFrame(node) {
  node.style.display = 'block';
  node.style.width = '100%';
  node.style.height = '100%';
  node.style.objectFit = 'cover';
  return node;
}

function videoPreview(url) {
  const wrap = previewFrame();
  const canvas = fillFrame(el('canvas', 'tip-preview-img'));
  canvas.width = 320;
  canvas.height = 180;
  const video = document.createElement('video');
  video.className = 'tip-preview-src';
  video.src = url;
  video.muted = true;
  video.preload = 'auto';
  video.playsInline = true;
  video.addEventListener('loadedmetadata', () => {
    const span = video.duration || 0;
    const target = Math.min(PREVIEW_FRAME / PREVIEW_FPS, Math.max(0, span - 0.1));
    video.currentTime = target;
  }, { once: true });
  video.addEventListener('seeked', () => {
    const w = video.videoWidth;
    const h = video.videoHeight;
    const ctx = w && h ? canvas.getContext('2d') : null;
    if (!ctx) return;
    canvas.width = w;
    canvas.height = h;
    ctx.drawImage(video, 0, 0, w, h);
    wrap.replaceChildren(canvas);
    video.removeAttribute('src');
    video.load();
  }, { once: true });
  wrap.append(canvas, video);
  return wrap;
}

function attachmentPreview(att) {
  if (!att.path || att.kind === 'link') return null;
  const url = mediaURL(att.path);
  if (att.kind === 'image') {
    const wrap = previewFrame();
    const img = fillFrame(el('img', 'tip-preview-img'));
    img.alt = '';
    img.src = url;
    img.addEventListener('error', hideTip, { once: true });
    wrap.appendChild(img);
    return wrap;
  }
  if (att.kind === 'video') return videoPreview(url);
  if (att.kind === 'text' && (!att.size || att.size <= PREVIEW_TEXT_BYTES)) {
    const wrap = previewFrame();
    const pre = el('pre', 'tip-preview-text', '…');
    pre.style.margin = '0';
    pre.style.boxSizing = 'border-box';
    pre.style.width = '100%';
    pre.style.height = '100%';
    pre.style.padding = '6px 7px';
    pre.style.overflow = 'hidden';
    pre.style.font = '10px/1.5 ui-monospace, Consolas, monospace';
    pre.style.color = 'var(--text-dim, #9aa4b4)';
    pre.style.whiteSpace = 'pre-wrap';
    pre.style.wordBreak = 'break-all';
    wrap.appendChild(pre);
    fetch(url)
      .then((res) => (res.ok ? res.text() : ''))
      .then((text) => {
        pre.textContent = text.slice(0, PREVIEW_TEXT_CHARS);
      })
      .catch(() => {
        pre.textContent = '';
      });
    return wrap;
  }
  return null;
}

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
    card.style.removeProperty('--memo-rgb');
    return;
  }
  const n = parseInt(match[1], 16);
  card.classList.add('tinted');
  card.style.setProperty('--memo-rgb', `${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}`);
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

export function createMemos() {
  const addBtn = document.getElementById('memoAdd');
  const searchInput = document.getElementById('memoSearch');
  const clearBtn = document.getElementById('memoClear');
  const listEl = document.getElementById('memoList');
  const linkModal = document.getElementById('memoLinkModal');
  const linkInput = document.getElementById('memoLinkInput');
  const linkError = document.getElementById('memoLinkError');
  const linkConfirm = document.getElementById('memoLinkConfirm');
  const linkCancel = document.getElementById('memoLinkCancel');

  let items = [];
  let keyword = '';
  let linkRef = null;
  const pending = new Set();
  const live = new Map();

  function track(promise) {
    pending.add(promise);
    promise.finally(() => pending.delete(promise));
    return promise;
  }

  function cardNode(memoId) {
    return [...listEl.children].find((c) => c.dataset.memoId === memoId) || null;
  }

  function renderEmpty() {
    listEl.innerHTML = '';
    listEl.appendChild(el('div', 'empty', keyword ? t('没有匹配的速记') : t('还没有速记\n点「新建」记一条')));
  }

  function addToItems(memo) {
    if (items.some((n) => n.id === memo.id)) return;
    const pinnedCount = items.filter((n) => n.pinned).length;
    items = [...items.slice(0, pinnedCount), memo, ...items.slice(pinnedCount)];
  }

  function applyMemo(memo) {
    const index = items.findIndex((n) => n.id === memo.id);
    if (index >= 0) items[index] = memo;
    const entry = live.get(memo.id);
    if (!entry) return;
    entry.ref.note = memo;
    entry.renderAttachments();
    entry.timeEl.textContent = relTime(memo.updatedAt);
  }

  function reportSkipped(skipped) {
    if (!skipped || !skipped.length) return;
    toast(skipped.join('；'), { duration: 3600 });
  }

  async function removeMemo(memo) {
    try {
      await Memos.remove(memo.id);
    } catch (err) {
      toast(t('删除失败'));
      return;
    }
    items = items.filter((n) => n.id !== memo.id);
    live.delete(memo.id);
    const card = cardNode(memo.id);
    if (card) card.remove();
    if (!listEl.children.length) renderEmpty();
  }

  async function togglePin(memo) {
    if (!items.some((n) => n.id === memo.id)) return;
    try {
      await Memos.update(memo.id, null, null, !memo.pinned);
      await load();
    } catch (err) {
      toast(t('操作失败'));
    }
  }

  async function applyTint(memo, tint, card, pop) {
    try {
      const saved = await Memos.setTint(memo.id, tint);
      memo.tint = saved.tint || '';
      applyCardTint(card, memo.tint);
      markTint(pop, memo.tint);
    } catch (err) {
      toast(t('保存失败'));
    }
  }

  async function openAttachment(memo, att) {
    try {
      await Memos.openAttachment(memo.id, att.id);
    } catch (err) {
      toast(errText(err));
    }
  }

  async function removeAttachment(memo, att) {
    try {
      const saved = await Memos.removeAttachment(memo.id, att.id);
      applyMemo(saved);
    } catch (err) {
      toast(errText(err));
    }
  }

  async function pickAttachments(ref) {
    try {
      const result = await Memos.pickAttachments(ref.note.id);
      if (!result || !result.memo) return;
      applyMemo(result.memo);
      if (result.added) toast(tp('已添加 {n} 个附件', { n: result.added }));
      reportSkipped(result.skipped);
    } catch (err) {
      const message = errText(err);
      if (/cancel|no file selected|已取消|未选择文件|未選擇檔案/i.test(message)) return;
      toast(message, { duration: 3200 });
    }
  }

  async function addPaths(memoId, paths) {
    if (!paths || !paths.length) return;
    pathDropStamp = Date.now();
    try {
      const result = await Memos.addPaths(memoId, paths);
      if (!result || !result.memo) return;
      applyMemo(result.memo);
      if (result.added) toast(tp('已添加 {n} 个附件', { n: result.added }));
      reportSkipped(result.skipped);
    } catch (err) {
      toast(errText(err), { duration: 3200 });
    }
  }

  async function uploadFiles(ref, files) {
    const items = [];
    let tooLarge = false;
    for (const file of files) {
      if (!file || !file.size) continue;
      if (file.size > BYTE_UPLOAD_LIMIT) {
        tooLarge = true;
        continue;
      }
      try {
        const buffer = await file.arrayBuffer();
        items.push({ name: payloadName(file), data: base64Of(new Uint8Array(buffer)) });
      } catch (err) {
        console.warn('attachment read failed', err);
      }
    }
    if (tooLarge) toast(t('文件过大，请改用拖入文件方式添加'), { duration: 3200 });
    if (!items.length) return;
    try {
      const result = await Memos.addPayloads(ref.note.id, items);
      if (!result || !result.memo) return;
      applyMemo(result.memo);
      if (result.added) toast(tp('已添加 {n} 个附件', { n: result.added }));
      reportSkipped(result.skipped);
    } catch (err) {
      toast(errText(err), { duration: 3200 });
    }
  }

  function openAttachMenu(event, ref) {
    openContextMenu(event.clientX, event.clientY, [
      { label: t('选择文件…'), icon: 'i-file', action: () => pickAttachments(ref) },
      { label: t('添加链接…'), icon: 'i-link', action: () => openLinkModal(ref) },
    ]);
  }

  function openLinkModal(ref) {
    linkRef = ref;
    linkInput.value = '';
    linkError.hidden = true;
    linkConfirm.disabled = true;
    linkModal.hidden = false;
    linkInput.focus();
  }

  function closeLinkModal() {
    linkModal.hidden = true;
    linkRef = null;
  }

  async function commitLink() {
    if (!linkRef || linkConfirm.disabled) return;
    const url = linkInput.value.trim();
    if (!url) return;
    linkConfirm.disabled = true;
    try {
      const saved = await Memos.addLink(linkRef.note.id, url);
      applyMemo(saved);
      closeLinkModal();
    } catch (err) {
      linkError.textContent = errText(err);
      linkError.hidden = false;
      linkConfirm.disabled = false;
    }
  }

  function attachmentNode(ref, att) {
    const chip = el('div', att.kind === 'link' ? 'memo-attach is-link' : 'memo-attach');
    const open = el('button', 'memo-attach-main');
    open.type = 'button';
    open.appendChild(svgIcon(ATTACH_ICONS[att.kind] || 'i-file'));
    open.appendChild(el('span', 'memo-attach-name', att.name || t('附件')));
    if (att.kind !== 'link' && att.size) open.appendChild(el('span', 'memo-attach-size', fmtSize(att.size)));
    open.title = att.kind === 'link' ? att.url || att.name : att.name || '';
    setTipPreview(open, () => attachmentPreview(att));
    open.addEventListener('click', (e) => {
      e.stopPropagation();
      openAttachment(ref.note, att);
    });
    const remove = el('button', 'memo-attach-x', '×');
    remove.type = 'button';
    remove.title = t('移除附件');
    remove.setAttribute('aria-label', t('移除附件'));
    remove.addEventListener('mousedown', (e) => e.preventDefault());
    remove.addEventListener('click', (e) => {
      e.stopPropagation();
      removeAttachment(ref.note, att);
    });
    chip.append(open, remove);
    return chip;
  }

  function cardFor(memo) {
    const ref = { note: memo };
    const card = el('div', 'memo');
    card.dataset.memoId = memo.id;
    const row = el('div', 'memo-row');
    const ta = el('textarea', 'memo-text');
    ta.value = memo.text;
    if (!memo.text) ta.placeholder = t('随手速记…（Ctrl+Enter 分段）');
    ta.rows = 1;
    ta.spellcheck = false;

    ta.addEventListener('paste', (e) => {
      const files = clipboardFiles(e);
      if (!files.length) return;
      e.preventDefault();
      uploadFiles(ref, files);
    });

    let dropTimer = null;
    card.addEventListener('dragover', (e) => {
      if (!hasFileDrag(e)) return;
      e.preventDefault();
      card.classList.add('memo-drop-target');
    });
    card.addEventListener('dragleave', (e) => {
      if (e.relatedTarget instanceof Node && card.contains(e.relatedTarget)) return;
      card.classList.remove('memo-drop-target');
    });
    card.addEventListener('drop', (e) => {
      const files = Array.from((e.dataTransfer && e.dataTransfer.files) || []);
      card.classList.remove('memo-drop-target');
      if (!files.length) return;
      e.preventDefault();
      const stamp = Date.now();
      if (dropTimer) clearTimeout(dropTimer);
      dropTimer = setTimeout(() => {
        dropTimer = null;
        if (pathDropStamp > stamp) return;
        uploadFiles(ref, files);
      }, 400);
    });

    const timeEl = el('span', '', relTime(memo.updatedAt || memo.createdAt));
    const actions = el('div', 'memo-actions');

    const attachBtn = el('button', 'icon-btn sm memo-attach-btn');
    attachBtn.title = t('添加附件');
    attachBtn.setAttribute('aria-label', t('添加附件'));
    attachBtn.appendChild(svgIcon('i-paperclip'));
    attachBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (ref.note) openAttachMenu(e, ref);
    });

    const pinBtn = el('button', 'icon-btn sm');
    pinBtn.classList.toggle('active', !!memo.pinned);
    pinBtn.title = t(memo.pinned ? '取消置顶' : '置顶');
    pinBtn.appendChild(svgIcon('i-tack'));
    pinBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (ref.note) togglePin(ref.note);
    });

    const delBtn = el('button', 'icon-btn sm memo-del');
    delBtn.title = t('删除');
    delBtn.appendChild(svgIcon('i-trash'));
    delBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      clearTimer();
      if (ref.note) removeMemo(ref.note);
    });
    actions.append(attachBtn, pinBtn, delBtn);

    const tintPop = el('div', 'memo-tint');
    MEMO_TINTS.forEach(({ tint, label }) => {
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
    if (memo) {
      applyCardTint(card, memo.tint);
      markTint(tintPop, memo.tint);
    }

    const meta = el('div', 'memo-meta memo-meta-view');
    const tools = el('div', 'memo-hover-tools');
    tools.append(tintPop, actions);
    meta.append(timeEl, tools);
    const tagBar = el('div', 'memo-meta memo-meta-edit');
    const tagHash = el('span', 'tag-hash');
    tagHash.appendChild(svgIcon('i-hash'));
    const chips = el('div', 'tag-chips');
    const tagInput = el('input', 'tag-input');
    tagInput.type = 'text';
    tagInput.spellcheck = false;
    tagInput.placeholder = t('添加标签，回车确认');
    tagBar.append(tagHash, chips, tagInput);

    const viewTags = el('div', 'memo-tags-view');
    meta.insertBefore(viewTags, tools);

    const attachList = el('div', 'memo-attach-list');

    row.append(ta);
    card.append(row, attachList, tagBar, meta);

    function renderAttachments() {
      attachList.innerHTML = '';
      const attachments = ref.note.attachments || [];
      attachList.hidden = !attachments.length;
      attachments.forEach((att) => attachList.appendChild(attachmentNode(ref, att)));
    }

    live.set(memo.id, { ref, renderAttachments, timeEl });
    renderAttachments();

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
        ref.note = await Memos.update(ref.note.id, null, tags, null);
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
        ref.note = await Memos.update(ref.note.id, text, merged, null);
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

  async function addMemo() {
    let saved;
    try {
      saved = await Memos.create('', []);
    } catch (err) {
      toast(t('保存失败'));
      return;
    }
    addToItems(saved);
    render();
    const card = cardNode(saved.id);
    const ta = card ? card.querySelector('.memo-text') : null;
    if (ta) ta.focus();
  }

  function render() {
    listEl.innerHTML = '';
    live.clear();
    if (!items.length) {
      renderEmpty();
      return;
    }
    items.forEach((memo) => {
      const { card } = cardFor(memo);
      listEl.appendChild(card);
      autoGrow(card.querySelector('.memo-text'));
    });
  }

  async function load() {
    if (pending.size) await Promise.all([...pending]);
    try {
      items = await Memos.list(keyword, 0);
    } catch (err) {
      items = [];
      toast(t('读取速记失败'));
    }
    items.forEach((it) => {
      const next = LEGACY_TINTS[(it.tint || '').toLowerCase()];
      if (!next) return;
      it.tint = next;
      Memos.setTint(it.id, next).catch(() => {});
    });
    render();
  }

  function syncLabels() {
    addBtn.setAttribute('aria-label', t('新建速记'));
    searchInput.placeholder = t('搜索速记与标签');
    clearBtn.textContent = t('清除');
  }

  syncLabels();
  onLocaleChange(syncLabels);

  addBtn.addEventListener('click', addMemo);

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

  linkInput.addEventListener('input', () => {
    linkError.hidden = true;
    linkConfirm.disabled = linkInput.value.trim() === '';
  });
  linkInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      commitLink();
    }
  });
  linkConfirm.addEventListener('click', commitLink);
  linkCancel.addEventListener('click', closeLinkModal);
  linkModal.addEventListener('mousedown', (e) => {
    if (e.target === linkModal) closeLinkModal();
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !linkModal.hidden) {
      e.stopPropagation();
      closeLinkModal();
    }
  });

  load();

  return { refresh: load, addPaths };
}
