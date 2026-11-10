import { t } from './strings.js';

const ENHANCED = new WeakSet();

function makeStep(dir) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = `num-step ${dir}`;
  btn.tabIndex = -1;
  btn.setAttribute('aria-label', dir === 'up' ? t('增加') : t('减少'));
  btn.innerHTML = '<svg class="icon" viewBox="0 0 10 10"><path d="M2.2 6.4 5 3.6l2.8 2.8"/></svg>';
  return btn;
}

export function enhanceNumber(input) {
  if (!(input instanceof Element) || ENHANCED.has(input)) return;
  ENHANCED.add(input);
  input.classList.add('num-field');

  const wrap = document.createElement('span');
  wrap.className = 'num-wrap';
  input.parentNode.insertBefore(wrap, input);
  wrap.appendChild(input);

  const steps = document.createElement('span');
  steps.className = 'num-steps';
  const up = makeStep('up');
  const down = makeStep('down');
  steps.append(up, down);
  wrap.appendChild(steps);

  const nudge = (delta) => {
    if (input.disabled) return;
    if (delta > 0) input.stepUp();
    else input.stepDown();
    input.dispatchEvent(new Event('input', { bubbles: true }));
    input.dispatchEvent(new Event('change', { bubbles: true }));
  };

  up.addEventListener('click', () => nudge(1));
  down.addEventListener('click', () => nudge(-1));
  [up, down].forEach((btn) => btn.addEventListener('mousedown', (event) => event.preventDefault()));
}
