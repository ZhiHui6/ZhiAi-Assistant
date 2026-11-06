import { Launch, Icons, el, openContextMenu, toast } from './api.js';
import { dataText, t, tp } from './strings.js';
import { hideTip, showTip } from './tooltip.js';

const CAT_KEY = 'launchCategory';
const LEGACY_CAT_KEY = 'appDockCategory';

function errText(err) {
  return dataText(String(err).replace(/^Error:\s*/, ''));
}

function saveActive(id) {
  try {
    localStorage.setItem(CAT_KEY, id);
  } catch (err) {}
}

function loadActive() {
  try {
    return localStorage.getItem(CAT_KEY) || localStorage.getItem(LEGACY_CAT_KEY) || '';
  } catch (err) {
    return '';
  }
}

export function createLaunch() {
  const dockEl = document.getElementById('launchDock');
  const listEl = document.getElementById('launchList');
  const catsEl = document.getElementById('launchCats');
  const addBtn = document.getElementById('launchAdd');
  const catAddBtn = document.getElementById('launchCatAdd');
  const iconCache = new Map();
  let state = { categories: [], items: [] };
  let activeCategory = loadActive();

  async function load() {
    try {
      state = await Launch.list();
    } catch (err) {
      toast(t('读取程序列表失败'));
      state = { categories: [], items: [] };
    }
    if (!state.categories.some((c) => c.id === activeCategory)) {
      activeCategory = state.categories[0] ? state.categories[0].id : '';
      saveActive(activeCategory);
    }
    renderCats();
    render();
    loadIcons();
  }

  function loadIcons() {
    const missing = state.items.map((it) => it.path).filter((p) => !iconCache.has(p.toLowerCase()));
    if (!missing.length) return;
    (async () => {
      for (let i = 0; i < missing.length; i += 12) {
        const batch = missing.slice(i, i + 12);
        try {
          const map = await Icons.get(batch);
          Object.entries(map || {}).forEach(([k, v]) => iconCache.set(k, v));
        } catch (err) {
          break;
        }
      }
      applyIcons();
    })();
  }

  function applyIcons() {
    listEl.querySelectorAll('img[data-path]').forEach((img) => {
      const src = iconCache.get(img.dataset.path.toLowerCase());
      if (src && img.src !== src) img.src = src;
    });
  }

  function renderCats() {
    catsEl.innerHTML = '';
    state.categories.forEach((category) => {
      const chip = el('button', 'chip' + (category.id === activeCategory ? ' active' : ''), category.name);
      chip.dataset.catId = category.id;
      chip.addEventListener('click', () => {
        if (activeCategory === category.id) return;
        activeCategory = category.id;
        saveActive(activeCategory);
        hideTip();
        renderCats();
        render();
      });
      chip.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        e.stopPropagation();
        openContextMenu(e.clientX, e.clientY, [
          { label: t('重命名分类'), icon: 'i-edit', action: () => editCategory(category) },
          {
            label: t('删除分类'),
            icon: 'i-trash',
            danger: true,
            disabled: state.categories.length <= 1,
            action: async () => {
              try {
                await Launch.removeCategory(category.id);
                await load();
              } catch (err) {
                toast(errText(err));
              }
            },
          },
        ]);
      });
      catsEl.appendChild(chip);
    });
  }

  function addCategoryInline() {
    if (catsEl.querySelector('.launch-cat-input')) return;
    const input = el('input', 'launch-cat-input');
    input.placeholder = t('分类名');
    catsEl.appendChild(input);
    catsEl.scrollLeft = catsEl.scrollWidth;
    input.focus();
    let done = false;
    const finish = async (commit) => {
      if (done) return;
      done = true;
      const name = input.value.trim();
      if (commit && name) {
        try {
          const category = await Launch.addCategory(name);
          activeCategory = category.id;
          saveActive(activeCategory);
        } catch (err) {
          toast(errText(err));
        }
      }
      await load();
    };
    input.addEventListener('blur', () => finish(true));
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        finish(false);
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        finish(true);
      }
    });
  }

  function editCategory(category) {
    if (!category) return;
    const chip = catsEl.querySelector(`[data-cat-id="${category.id}"]`);
    if (!chip) return;
    const input = el('input', 'launch-cat-input');
    input.value = category.name;
    chip.replaceWith(input);
    input.focus();
    input.select();
    let done = false;
    const finish = async (commit) => {
      if (done) return;
      done = true;
      const name = input.value.trim();
      if (!commit || !name || name === category.name) {
        input.replaceWith(chip);
        return;
      }
      try {
        await Launch.renameCategory(category.id, name);
      } catch (err) {
        toast(errText(err));
      }
      await load();
    };
    input.addEventListener('blur', () => finish(true));
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        finish(false);
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        finish(true);
      }
    });
  }

  function categoryItems() {
    return state.items.filter((item) => item.categoryId === activeCategory);
  }

  function render() {
    hideTip();
    listEl.innerHTML = '';
    const items = categoryItems();
    listEl.classList.toggle('is-empty', !items.length);
    if (!items.length) {
      listEl.appendChild(el('div', 'launch-empty', t('把程序拖到这里，或点「程序」添加')));
      return;
    }
    items.forEach((item) => listEl.appendChild(renderItem(item)));
  }

  function renderItem(item) {
    const node = el('div', 'shelf-item');
    node.dataset.itemId = item.id;

    const iconEl = el('img', 'shelf-icon');
    iconEl.dataset.path = item.path;
    const cached = iconCache.get(item.path.toLowerCase());
    if (cached) iconEl.src = cached;
    iconEl.addEventListener('error', () => {
      const fallback = el('div', 'shelf-fallback', (item.path.split('.').pop() || 'app').slice(0, 4));
      iconEl.replaceWith(fallback);
    });

    const nameEl = el('div', 'shelf-name', item.name);
    node.appendChild(iconEl);
    node.appendChild(nameEl);

    node.addEventListener('mouseenter', () => {
      if (dockEl.classList.contains('hide-names') || nameEl.scrollWidth > nameEl.clientWidth + 1) {
        showTip(node, item.name);
      }
    });
    node.addEventListener('mouseleave', hideTip);

    node.addEventListener('click', async () => {
      hideTip();
      try {
        await Launch.launch(item.id);
      } catch (err) {
        toast(t('无法启动：') + errText(err));
      }
    });

    node.addEventListener('contextmenu', (e) => {
      e.preventDefault();
      e.stopPropagation();
      hideTip();
      const menu = [
        {
          label: t('启动'),
          icon: 'i-play',
          action: () => Launch.launch(item.id).catch(() => toast(t('无法启动'))),
        },
        {
          label: t('在资源管理器中显示'),
          icon: 'i-folder',
          action: () => Launch.reveal(item.id).catch(() => toast(t('定位失败'))),
        },
        { sep: true },
        { label: t('重命名'), icon: 'i-edit', action: () => startRename(nameEl, item) },
      ];
      const others = state.categories.filter((category) => category.id !== item.categoryId);
      if (others.length) {
        menu.push({ sep: true });
        others.forEach((category) => {
          menu.push({
            label: tp('移到「{name}」', { name: category.name }),
            icon: 'i-right',
            action: async () => {
              try {
                await Launch.move(item.id, category.id);
                await load();
              } catch (err) {
                toast(errText(err));
              }
            },
          });
        });
      }
      menu.push({ sep: true });
      menu.push({
        label: t('移除程序'),
        icon: 'i-trash',
        danger: true,
        action: async () => {
          await Launch.remove(item.id);
          await load();
        },
      });
      openContextMenu(e.clientX, e.clientY, menu);
    });

    return node;
  }

  function startRename(nameEl, item) {
    const input = el('input');
    input.value = item.name;
    input.className = 'shelf-name';
    input.style.cssText =
      'width:100%;background:rgba(0,0,0,0.3);border:1px solid rgba(255,255,255,0.16);border-radius:5px;padding:1px 3px;color:inherit;font-family:inherit;font-size:10.5px;outline:none;text-align:center';
    nameEl.replaceWith(input);
    input.focus();
    input.select();
    let done = false;
    const finish = async (commit) => {
      if (done) return;
      done = true;
      const name = input.value.trim();
      if (!commit || name === item.name) {
        input.replaceWith(nameEl);
        return;
      }
      try {
        await Launch.rename(item.id, name);
      } catch (err) {
        toast(t('重命名失败'));
      }
      await load();
    };
    input.addEventListener('blur', () => finish(true));
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') finish(false);
      if (e.key === 'Enter') finish(true);
    });
  }

  async function handleDrop(x, y, paths) {
    if (!paths || !paths.length) return;
    try {
      const added = await Launch.addPaths(activeCategory, paths);
      toast(tp('已添加 {n} 个程序', { n: added ? added.length : 0 }));
      await load();
    } catch (err) {
      toast(errText(err));
    }
  }

  catAddBtn.addEventListener('click', () => {
    hideTip();
    addCategoryInline();
  });

  addBtn.addEventListener('click', async () => {
    try {
      const added = await Launch.pick(activeCategory);
      if (added && added.length) {
        await load();
        toast(tp('已添加 {n} 个程序', { n: added.length }));
      }
    } catch (err) {
      if (!/取消/.test(String(err))) toast(errText(err));
    }
  });

  window.addEventListener('resize', hideTip);
  load();

  return { refresh: load, handleDrop };
}
