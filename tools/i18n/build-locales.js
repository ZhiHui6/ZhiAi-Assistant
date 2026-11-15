const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..', '..');
const source = path.join(__dirname, 'strings.json');
const target = path.join(root, 'frontend/js/locales.js');

const entries = JSON.parse(fs.readFileSync(source, 'utf8'));
const PLACEHOLDER = /\{(\w+)\}/g;

function placeholders(text) {
  return Array.from(text.matchAll(PLACEHOLDER), (m) => m[1]).sort().join(',');
}

const en = {};
const tw = {};
const problems = [];

for (const [key, value] of Object.entries(entries)) {
  if (!Array.isArray(value) || value.length !== 2 || !value[0] || !value[1]) {
    problems.push(`${key} 缺少 [en, zh-TW] 译文`);
    continue;
  }
  const [a, b] = value;
  const base = placeholders(key);
  if (placeholders(a) !== base) problems.push(`${key} 的英文占位符不一致`);
  if (placeholders(b) !== base) problems.push(`${key} 的繁體占位符不一致`);
  en[key] = a;
  tw[key] = b;
}

if (problems.length) {
  for (const item of problems) console.error('✗ ' + item);
  console.error(`共 ${problems.length} 处问题，未写入 locales.js`);
  process.exit(1);
}

const out =
  'export const EN = ' + JSON.stringify(en, null, 1) + ';\n\n' +
  'export const ZH_TW = ' + JSON.stringify(tw, null, 1) + ';\n';
fs.writeFileSync(target, out, 'utf8');
console.log(`strings.json: ${Object.keys(en).length} 条 → frontend/js/locales.js`);
