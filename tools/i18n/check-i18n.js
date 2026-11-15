const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '..', '..');
const dict = JSON.parse(fs.readFileSync(path.join(__dirname, 'strings.json'), 'utf8'));

const DYNAMIC_KEYS = [
  '立春', '雨水', '惊蛰', '春分', '谷雨', '立夏', '小满', '芒种', '夏至', '小暑', '大暑',
  '立秋', '处暑', '白露', '秋分', '寒露', '霜降', '立冬', '小雪', '大雪', '冬至', '小寒', '大寒',
  '元宵', '除夕', '重阳', '腊八', '七夕', '妇女', '儿童', '教师',
  '农历', '节气', '调休上班', '法定节假日', '休', '班',
];
const SENTINELS = ['已取消', '未选择文件'];

const HAN = /[\u4e00-\u9fa5]/;

function walk(dir, accept, out) {
  if (!fs.existsSync(dir)) return out;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(file, accept, out);
    else if (accept(entry.name)) out.push(file);
  }
  return out;
}

const uiFiles = walk(path.join(root, 'frontend'), (name) => (name.endsWith('.js') && name !== 'locales.js') || name === 'index.html', []);
const allFiles = uiFiles.concat(
  walk(path.join(root, 'internal'), (name) => name.endsWith('.go'), []),
  [path.join(root, 'main.go')]
);

function decode(text) {
  return text.replace(/\\n/g, '\n');
}

function add(map, text, file, normalize) {
  if (!text || text.includes('${')) return;
  const value = normalize ? decode(text).replace(/\s+/g, ' ').trim() : decode(text);
  if (!value || !HAN.test(value)) return;
  if (!map.has(value)) map.set(value, file);
}

function scanTemplates(text, add_) {
  let i = 0;
  while (i < text.length) {
    if (text[i] !== '`') {
      i += 1;
      continue;
    }
    i += 1;
    let chunk = '';
    let depth = 0;
    while (i < text.length) {
      const ch = text[i];
      if (ch === '\\') {
        chunk += text.slice(i, i + 2);
        i += 2;
        continue;
      }
      if (depth === 0 && ch === '`') break;
      if (ch === '$' && text[i + 1] === '{') {
        depth += 1;
        i += 2;
        continue;
      }
      if (depth > 0) {
        if (ch === '{') depth += 1;
        if (ch === '}') depth -= 1;
        i += 1;
        continue;
      }
      chunk += ch;
      i += 1;
    }
    i += 1;
    add_(chunk);
  }
}

const candidates = new Map();
for (const file of uiFiles) {
  const rel = path.relative(root, file).replace(/\\/g, '/');
  const text = fs.readFileSync(file, 'utf8');
  if (rel.endsWith('.html')) {
    for (const m of text.matchAll(/>([^<>]+)</g)) add(candidates, m[1], rel, true);
    for (const m of text.matchAll(/(?:placeholder|title|aria-label)="([^"]*)"/g)) add(candidates, m[1], rel, true);
    continue;
  }
  for (const m of text.matchAll(/'((?:[^'\\\n]|\\.)*)'/g)) add(candidates, m[1], rel, false);
  scanTemplates(text, (chunk) => add(candidates, chunk, rel, false));
}

const blob = allFiles.map((file) => fs.readFileSync(file, 'utf8')).join('\n');

const goDict = new Set();
for (const name of ['i18n.go', 'errors.go']) {
  const file = path.join(root, 'internal', 'i18n', name);
  if (!fs.existsSync(file)) continue;
  for (const m of fs.readFileSync(file, 'utf8').matchAll(/^\t"((?:[^"\\]|\\.)*)":/gm)) goDict.add(m[1]);
}

const goMissing = [];
for (const file of allFiles) {
  const rel = path.relative(root, file).replace(/\\/g, '/');
  if (!rel.endsWith('.go') || rel.startsWith('internal/i18n/')) continue;
  const text = fs.readFileSync(file, 'utf8');
  for (const m of text.matchAll(/(?:errors\.New|fmt\.Errorf|i18n\.TF?)\(\s*"((?:[^"\\\n]|\\.)*)"/g)) {
    const value = m[1];
    if (!HAN.test(value) || goDict.has(value)) continue;
    goMissing.push(`${rel}: ${value}`);
  }
}

const missing = [];
for (const [text, file] of candidates) {
  if (text in dict || SENTINELS.includes(text)) continue;
  missing.push(`${file}: ${text}`);
}

const unused = Object.keys(dict).filter((key) => {
  if (DYNAMIC_KEYS.includes(key)) return false;
  return !blob.includes(key) && !blob.includes(key.replace(/\n/g, '\\n'));
});

if (missing.length) {
  console.error(`✗ 界面文案缺少译文 ${missing.length} 条`);
  for (const item of missing) console.error('  ' + item);
} else {
  console.log('✓ 界面文案全部已收录');
}

if (goMissing.length) {
  console.error(`✗ 后端文案缺少译文 ${goMissing.length} 条`);
  for (const item of goMissing) console.error('  ' + item);
} else {
  console.log(`✓ 后端文案全部已收录（${goDict.size} 条）`);
}

if (unused.length) {
  console.log(`! 词典中可能有 ${unused.length} 条已废弃`);
  for (const item of unused) console.log('  ' + item);
} else {
  console.log('✓ 词典无废弃词条');
}

console.log(`词典 ${Object.keys(dict).length} 条，界面候选 ${candidates.size} 条，扫描 ${allFiles.length} 个文件`);
process.exit(missing.length || goMissing.length ? 1 : 0);
