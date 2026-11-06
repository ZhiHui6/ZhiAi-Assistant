import { EN, ZH_TW } from './locales.js';

const DICTS = { en: EN, 'zh-TW': ZH_TW };
const ATTRS = ['placeholder', 'title', 'aria-label'];
const entries = [];
const hooks = [];
let locale = 'zh-CN';

export function t(source) {
  const dict = DICTS[locale];
  return (dict && dict[source]) || source;
}

export function tp(source, params) {
  return t(source).replace(/\{(\w+)\}/g, (match, key) => (params[key] ?? match));
}

export function captureStatic(root = document) {
  if (entries.length) return;
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) {
    const node = walker.currentNode;
    const raw = node.nodeValue;
    if (!raw || !/[\u4e00-\u9fa5]/.test(raw)) continue;
    entries.push({ node, original: raw.replace(/\s+/g, ' ').trim() });
  }
  for (const el of root.querySelectorAll(ATTRS.map((attr) => `[${attr}]`).join(','))) {
    for (const attr of ATTRS) {
      const value = el.getAttribute(attr);
      if (!value || !/[\u4e00-\u9fa5]/.test(value)) continue;
      entries.push({ el, attr, original: value.replace(/\s+/g, ' ').trim() });
    }
  }
}

function applyStatic() {
  const dict = DICTS[locale];
  for (const entry of entries) {
    const value = (dict && dict[entry.original]) || entry.original;
    if (entry.node) {
      if (!entry.node.isConnected) continue;
      const raw = entry.node.nodeValue || '';
      const lead = raw.match(/^\s*/)[0];
      const trail = raw.match(/\s*$/)[0];
      entry.node.nodeValue = lead + value + trail;
    } else if (entry.el.isConnected) {
      entry.el.setAttribute(entry.attr, value);
    }
  }
}

const LUNAR_MONTH = { 正: 1, 二: 2, 三: 3, 四: 4, 五: 5, 六: 6, 七: 7, 八: 8, 九: 9, 十: 10, 十一: 11, 十二: 12, 冬: 11, 腊: 12 };
const LUNAR_DAY = {
  初一: 1, 初二: 2, 初三: 3, 初四: 4, 初五: 5, 初六: 6, 初七: 7, 初八: 8, 初九: 9, 初十: 10,
  十一: 11, 十二: 12, 十三: 13, 十四: 14, 十五: 15, 十六: 16, 十七: 17, 十八: 18, 十九: 19, 二十: 20,
  廿一: 21, 廿二: 22, 廿三: 23, 廿四: 24, 廿五: 25, 廿六: 26, 廿七: 27, 廿八: 28, 廿九: 29, 三十: 30,
};

function lunarNumber(text) {
  const m = text.match(/^(十[一二]?|冬|腊|正|[二三四五六七八九])月(.+)$/);
  if (!m || !LUNAR_DAY[m[2]]) return text;
  return `${LUNAR_MONTH[m[1]]}/${LUNAR_DAY[m[2]]}`;
}

export function dataText(source) {
  if (!source) return source;
  const dict = DICTS[locale];
  if (!dict) return source;
  if (dict[source]) return dict[source];
  const parts = source.split(' · ').map((part) => {
    if (dict[part]) return dict[part];
    let m = part.match(/^农历(.+)$/);
    if (m) return `${dict['农历']} ${lunarNumber(m[1])}`;
    m = part.match(/^(.+?)节气$/);
    if (m && dict[m[1]]) return `${dict[m[1]]} ${dict['节气']}`;
    m = part.match(/^(.+?)调休上班$/);
    if (m) return `${dataText(m[1])} ${dict['调休上班']}`;
    m = part.match(/^(.+?)节$/);
    if (m && dict[m[1]]) return dict[m[1]];
    if (/[初十廿]/.test(part)) return lunarNumber(part);
    return part;
  });
  return parts.join(' · ');
}

export function setLocale(next) {
  locale = DICTS[next] ? next : 'zh-CN';
  applyStatic();
  for (const fn of hooks) fn(locale);
}

export function getLocale() {
  return locale;
}

export function onLocaleChange(fn) {
  hooks.push(fn);
}
