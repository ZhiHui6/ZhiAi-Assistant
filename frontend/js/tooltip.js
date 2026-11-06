import { el } from './api.js';

let tipEl = null;
let tipTimer = null;

export function hideTip() {
  if (tipTimer) {
    clearTimeout(tipTimer);
    tipTimer = null;
  }
  if (tipEl) tipEl.hidden = true;
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

export function showTip(node, text) {
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
    let left = box.left + box.width / 2 - tw / 2;
    left = Math.max(6, Math.min(left, window.innerWidth - tw - 6));
    let top = box.top - th - 6;
    if (top < 6) top = box.bottom + 6;
    top = Math.min(top, window.innerHeight - th - 6);
    tip.style.left = left + 'px';
    tip.style.top = top + 'px';
  }, 350);
}

export function bindAriaTip(node) {
  node.addEventListener('mouseenter', () => {
    const label = node.getAttribute('aria-label');
    if (label) showTip(node, label);
  });
  node.addEventListener('mouseleave', hideTip);
}
