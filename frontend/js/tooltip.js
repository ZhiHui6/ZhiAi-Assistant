import { el } from './api.js';

let tipEl = null;
let tipTimer = null;
let activeNode = null;
let installed = false;
let previewEl = null;
const previews = new WeakMap();

export function hideTip() {
  if (tipTimer) {
    clearTimeout(tipTimer);
    tipTimer = null;
  }
  if (tipEl) tipEl.hidden = true;
  if (previewEl) {
    previewEl.hidden = true;
    previewEl.replaceChildren();
  }
}

export function setTipPreview(node, build) {
  previews.set(node, build);
}

function ensureTip() {
  if (!tipEl) {
    tipEl = el('div', 'dock-tip');
    tipEl.setAttribute('role', 'tooltip');
    tipEl.hidden = true;
    document.body.appendChild(tipEl);
  }
  return tipEl;
}

function ensurePreview() {
  if (!previewEl) {
    previewEl = el('div', 'tip-preview');
    previewEl.hidden = true;
    document.body.appendChild(previewEl);
  }
  return previewEl;
}

function placePreview(node) {
  const build = previews.get(node);
  if (!build) return;
  const content = build();
  if (!content) return;
  const box = ensurePreview();
  box.replaceChildren(content);
  box.hidden = false;
  box.style.left = '0px';
  box.style.top = '0px';
  const tip = tipEl.getBoundingClientRect();
  const bw = box.offsetWidth;
  const bh = box.offsetHeight;
  let left = tip.left + tip.width / 2 - bw / 2;
  left = Math.max(6, Math.min(left, window.innerWidth - bw - 6));
  let top = tip.top - bh - 8;
  if (top < 6) top = tip.bottom + 8;
  top = Math.min(top, window.innerHeight - bh - 6);
  box.style.left = left + 'px';
  box.style.top = top + 'px';
}

export function showTip(node, text, delay = 700) {
  hideTip();
  tipTimer = setTimeout(() => {
    tipTimer = null;
    const tip = ensureTip();
    tip.textContent = text;
    tip.hidden = false;
    tip.style.left = '0px';
    tip.style.top = '0px';
    const box = node.getBoundingClientRect();
    const tw = tip.offsetWidth;
    const th = tip.offsetHeight;
    const centerX = box.left + box.width / 2;
    let left = centerX - tw / 2;
    left = Math.max(6, Math.min(left, window.innerWidth - tw - 6));
    let pos = 'top';
    let top = box.top - th - 8;
    if (top < 6) {
      pos = 'bottom';
      top = box.bottom + 8;
    }
    top = Math.min(top, window.innerHeight - th - 6);
    const arrow = Math.round(Math.max(12, Math.min(centerX - left, tw - 12)));
    tip.dataset.pos = pos;
    tip.style.setProperty('--tip-arrow', arrow + 'px');
    tip.style.left = left + 'px';
    tip.style.top = top + 'px';
    placePreview(node);
  }, delay);
}

export function bindAriaTip(node) {
  node.addEventListener('mouseenter', () => {
    const label = node.getAttribute('aria-label');
    if (label) showTip(node, label);
  });
  node.addEventListener('mouseleave', hideTip);
}

function tipText(node) {
  const title = node.getAttribute('title');
  if (title) {
    node.setAttribute('data-tip', title);
    node.removeAttribute('title');
    return title;
  }
  return node.getAttribute('data-tip') || '';
}

function pick(e) {
  return e.target instanceof Element ? e.target.closest('[title], [data-tip]') : null;
}

export function installTips() {
  if (installed) return;
  installed = true;

  document.addEventListener('mouseover', (e) => {
    const node = pick(e);
    if (!node || node === activeNode) return;
    const text = tipText(node);
    if (!text) return;
    activeNode = node;
    showTip(node, text);
  });

  document.addEventListener('mouseout', (e) => {
    if (!activeNode) return;
    const next = e.relatedTarget;
    if (next instanceof Node && activeNode.contains(next)) return;
    activeNode = null;
    hideTip();
  });

  document.addEventListener('focusin', (e) => {
    const node = pick(e);
    if (!node || !node.matches(':focus-visible')) return;
    const text = tipText(node);
    if (!text) return;
    activeNode = null;
    showTip(node, text, 100);
  });

  document.addEventListener('focusout', () => {
    activeNode = null;
    hideTip();
  });

  document.addEventListener('mousedown', hideTip, true);
  document.addEventListener('keydown', hideTip, true);
  window.addEventListener('scroll', hideTip, true);
  window.addEventListener('blur', hideTip);
}
