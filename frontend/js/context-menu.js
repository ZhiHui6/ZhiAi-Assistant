import { App, closeContextMenu, openContextMenu } from './api.js';
import { t } from './strings.js';

let editState = null;

function isTextEditable(node) {
  if (!node) return false;
  if (node.isContentEditable) return true;
  if (node.tagName === 'TEXTAREA') return true;
  if (node.tagName !== 'INPUT') return false;
  const type = (node.getAttribute('type') || 'text').toLowerCase();
  return ['text', 'search', 'url', 'tel', 'email', 'password', 'number'].includes(type);
}

function selectedText() {
  const sel = window.getSelection();
  return sel ? sel.toString() : '';
}

function restoreField() {
  if (!editState) return null;
  const node = editState.field;
  if (!node || !node.isConnected) return null;
  node.focus();
  if (editState.start !== null && typeof node.setSelectionRange === 'function') {
    try {
      node.setSelectionRange(editState.start, editState.end, editState.dir);
    } catch (err) {}
  }
  return node;
}

function execCommand(cmd) {
  const node = restoreField();
  if (!node) return;
  try {
    document.execCommand(cmd, false);
  } catch (err) {}
}

async function pasteText() {
  let text = '';
  try {
    text = await App.clipboardText();
  } catch (err) {}
  if (!text) return;
  const node = restoreField();
  if (!node) return;
  node.focus();
  let ok = false;
  try {
    ok = document.execCommand('insertText', false, text);
  } catch (err) {}
  if (ok) return;
  if (node.isContentEditable) {
    const escaped = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    document.execCommand('insertHTML', false, escaped);
    return;
  }
  if (typeof node.setRangeText === 'function' && node.selectionStart !== null) {
    node.setRangeText(text, node.selectionStart, node.selectionEnd, 'end');
    node.dispatchEvent(new Event('input', { bubbles: true }));
  }
}

function selectAll() {
  const node = restoreField();
  if (!node) return;
  if (node.isContentEditable) {
    document.execCommand('selectAll');
    return;
  }
  try {
    node.select();
  } catch (err) {}
}

function showEditMenu(field, x, y) {
  let start = null;
  let end = null;
  let dir = 'none';
  if (typeof field.selectionStart === 'number') {
    start = field.selectionStart;
    end = field.selectionEnd;
    dir = field.selectionDirection || 'none';
  }
  const hasSelection = field.isContentEditable ? !!selectedText() : start !== null && start !== end;
  editState = { field, start, end, dir };

  openContextMenu(x, y, [
    { label: t('撤销'), key: 'Ctrl+Z', icon: 'i-undo', action: () => execCommand('undo') },
    { label: t('重做'), key: 'Ctrl+Y', icon: 'i-redo', action: () => execCommand('redo') },
    { sep: true },
    { label: t('剪切'), key: 'Ctrl+X', icon: 'i-cut', disabled: !hasSelection, action: () => execCommand('cut') },
    { label: t('复制'), key: 'Ctrl+C', icon: 'i-copy', disabled: !hasSelection, action: () => execCommand('copy') },
    { label: t('粘贴'), key: 'Ctrl+V', icon: 'i-paste', action: pasteText },
    { sep: true },
    { label: t('全选'), key: 'Ctrl+A', icon: 'i-select-all', action: selectAll },
  ]);
}

export function installContextMenu() {
  document.addEventListener('contextmenu', (e) => {
    if (e.defaultPrevented) return;
    const target = e.target instanceof Element ? e.target : null;
    const field = target ? target.closest('input, textarea, [contenteditable="true"]') : null;
    if (field && isTextEditable(field)) {
      e.preventDefault();
      showEditMenu(field, e.clientX, e.clientY);
      return;
    }
    if (!field) {
      const text = selectedText();
      if (text.trim()) {
        e.preventDefault();
        openContextMenu(e.clientX, e.clientY, [
          {
            label: t('复制'),
            key: 'Ctrl+C',
            icon: 'i-copy',
            action: () => App.setClipboardText(text).catch(() => {}),
          },
        ]);
        return;
      }
    }
    e.preventDefault();
  });

  document.addEventListener(
    'keydown',
    (e) => {
      if (e.key === 'Escape' && document.querySelector('.ctx-menu')) {
        e.preventDefault();
        e.stopPropagation();
        closeContextMenu();
      }
    },
    true,
  );
}
