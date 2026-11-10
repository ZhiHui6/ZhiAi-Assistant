import { App, Icons, Mount, Shelf, Thumbs, el, fmtSize, openContextMenu, svgIcon, toast } from './api.js';
import { t, tp } from './strings.js';

export function createShelf({ onChanged }) {
  const listEl = document.getElementById('shelfList');
  const hintEl = document.getElementById('shelfHint');
  const crumbEl = document.getElementById('crumb');
  const backBtn = document.getElementById('mountBack');
  const addGroupBtn = document.getElementById('shelfAddGroup');
  const mountAddBtn = document.getElementById('mountAdd');
  const iconCache = new Map();
  const thumbCache = new Map();
  const IMAGE_RE = /\.(png|jpe?g|gif)$/i;
  let data = { groups: [], items: [] };
  let mounts = [];
  let browsing = null;

  async function load() {
    browsing = null;
    const [shelf, mountList] = await Promise.all([
      Shelf.list().catch(() => null),
      Mount.list().catch(() => []),
    ]);
    if (shelf) {
      data = shelf;
    } else {
      toast(t('读取收纳区失败'));
      data = { groups: [], items: [] };
    }
    mounts = Array.isArray(mountList) ? mountList : [];
    syncHead();
    render();
    loadIcons(data.items.filter((it) => it.kind !== 'url').map((it) => it.path));
    loadThumbs(data.items.filter((it) => IMAGE_RE.test(it.path || '')).map((it) => it.path));
  }

  function loadIcons(paths) {
    const missing = [];
    const seen = new Set();
    paths.forEach((path) => {
      if (!path) return;
      const key = path.toLowerCase();
      if (iconCache.has(key) || seen.has(key)) return;
      seen.add(key);
      missing.push(path);
    });
    if (!missing.length) {
      applyIcons();
      return;
    }
    (async () => {
      for (let i = 0; i < missing.length; i += 12) {
        const batch = missing.slice(i, i + 12);
        try {
          const map = await Icons.get(batch);
          const keys = map ? Object.keys(map) : [];
          if (window.__debug) App.debugLog(`icons batch=${batch.length} got=${keys.length}`);
          Object.entries(map || {}).forEach(([k, v]) => iconCache.set(k, v));
        } catch (err) {
          if (window.__debug) App.debugLog('icons failed: ' + String(err));
          break;
        }
      }
      applyIcons();
    })();
  }

  function loadThumbs(paths) {
    const missing = [];
    const seen = new Set();
    paths.forEach((path) => {
      if (!path) return;
      const key = path.toLowerCase();
      if (thumbCache.has(key) || seen.has(key)) return;
      seen.add(key);
      missing.push(path);
    });
    if (!missing.length) {
      applyThumbs();
      return;
    }
    (async () => {
      for (let i = 0; i < missing.length; i += 8) {
        const batch = missing.slice(i, i + 8);
        try {
          const map = await Thumbs.get(batch);
          Object.entries(map || {}).forEach(([k, v]) => thumbCache.set(k, v));
        } catch (err) {
          if (window.__debug) App.debugLog('thumbs failed: ' + String(err));
          break;
        }
      }
      applyThumbs();
    })();
  }

  function applyThumbs() {
    listEl.querySelectorAll('img[data-thumb-path]').forEach((img) => {
      const src = thumbCache.get(img.dataset.thumbPath.toLowerCase());
      if (!src || img.dataset.thumbApplied) return;
      img.src = src;
      img.dataset.thumbApplied = '1';
      img.classList.add('shelf-thumb');
    });
  }

  function applyIcons() {
    let applied = 0;
    listEl.querySelectorAll('img[data-path]').forEach((img) => {
      if (img.dataset.thumbApplied) return;
      const key = img.dataset.path.toLowerCase();
      const src = iconCache.get(key);
      if (src && img.src !== src) {
        img.src = src;
        applied += 1;
      }
    });
    if (window.__debug) App.debugLog(`icons applied=${applied} imgNodes=${listEl.querySelectorAll('img[data-path]').length}`);
  }

  function itemsOf(groupId) {
    return data.items.filter((it) => it.groupId === groupId).sort((a, b) => a.order - b.order);
  }

  let draggedId = null;
  let suppressClickUntil = 0;
  let dropLine = null;

  function hideDropLine() {
    if (dropLine) dropLine.style.display = 'none';
  }

  function ensureDropLine() {
    if (!dropLine) {
      dropLine = el('div', 'shelf-drop-line');
      document.body.appendChild(dropLine);
    }
    return dropLine;
  }

  function clearDropTargets() {
    listEl.querySelectorAll('.drop-target').forEach((node) => node.classList.remove('drop-target'));
  }

  function gridIndex(grid, x, y, skipId) {
    const nodes = [...grid.querySelectorAll('.shelf-item')].filter((node) => node.dataset.itemId !== skipId);
    let index = nodes.length;
    for (let i = 0; i < nodes.length; i++) {
      const rect = nodes[i].getBoundingClientRect();
      if (y < rect.top + rect.height / 2) {
        index = i;
        break;
      }
      if (y <= rect.bottom && x < rect.left + rect.width / 2) {
        index = i;
        break;
      }
    }
    return index;
  }

  function showDropLine(grid, index) {
    const line = ensureDropLine();
    const nodes = [...grid.querySelectorAll('.shelf-item')].filter((node) => node.dataset.itemId !== draggedId);
    line.style.display = 'block';
    const gridRect = grid.getBoundingClientRect();
    if (grid.closest('.shelf-view-list')) {
      const top = index < nodes.length
        ? nodes[index].getBoundingClientRect().top
        : (nodes.length ? nodes[nodes.length - 1].getBoundingClientRect().bottom : gridRect.top);
      line.style.left = gridRect.left + 4 + 'px';
      line.style.width = gridRect.width - 8 + 'px';
      line.style.height = '2px';
      line.style.top = top - 1 + 'px';
      return;
    }
    line.style.width = '2px';
    if (!nodes.length) {
      line.style.left = gridRect.left + 6 + 'px';
      line.style.top = gridRect.top + 4 + 'px';
      line.style.height = '24px';
      return;
    }
    if (index < nodes.length) {
      const rect = nodes[index].getBoundingClientRect();
      line.style.left = rect.left - 4 + 'px';
      line.style.top = rect.top + 2 + 'px';
      line.style.height = rect.height - 4 + 'px';
    } else {
      const rect = nodes[nodes.length - 1].getBoundingClientRect();
      line.style.left = rect.right + 2 + 'px';
      line.style.top = rect.top + 2 + 'px';
      line.style.height = rect.height - 4 + 'px';
    }
  }

  function attachGroupDrop(groupEl, head, grid, group) {
    const isInternal = () => !!draggedId;
    const setTarget = (on) => groupEl.classList.toggle('drop-target', on);
    const onOver = (e) => {
      if (!isInternal()) return;
      e.preventDefault();
      e.stopPropagation();
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
      setTarget(true);
      const onGrid = !!grid && !!e.target.closest && !!e.target.closest('.grid');
      if (onGrid) {
        showDropLine(grid, gridIndex(grid, e.clientX, e.clientY, draggedId));
      } else {
        hideDropLine();
      }
    };
    const onLeave = (e) => {
      if (!isInternal()) return;
      const next = e.relatedTarget;
      if (next && groupEl.contains(next)) return;
      setTarget(false);
      hideDropLine();
    };
    const onDrop = async (e) => {
      if (!isInternal()) return;
      e.preventDefault();
      e.stopPropagation();
      const id = draggedId;
      const onGrid = !!grid && !!e.target.closest && !!e.target.closest('.grid');
      const index = onGrid
        ? gridIndex(grid, e.clientX, e.clientY, id)
        : itemsOf(group.id).filter((it) => it.id !== id).length;
      draggedId = null;
      suppressClickUntil = Date.now() + 500;
      setTarget(false);
      hideDropLine();
      try {
        await Shelf.move(id, group.id, index);
        await load();
        onChanged && onChanged();
      } catch (err) {
        toast(t('移动失败'));
      }
    };

    groupEl.addEventListener('dragover', onOver);
    groupEl.addEventListener('dragleave', onLeave);
    groupEl.addEventListener('drop', onDrop);
  }

  function syncHead() {
    if (browsing) {
      hintEl.textContent = '';
      crumbEl.hidden = false;
      backBtn.hidden = false;
      addGroupBtn.hidden = true;
      mountAddBtn.hidden = true;
      return;
    }
    hintEl.textContent = t('拖入文件收藏，挂载可直接浏览');
    crumbEl.hidden = true;
    backBtn.hidden = true;
    addGroupBtn.hidden = false;
    mountAddBtn.hidden = false;
  }

  function render() {
    listEl.innerHTML = '';
    renderMounts();
    const section = el('div', 'shelf-section');
    listEl.appendChild(section);
    const title = el('div', 'shelf-title');
    const titleIcon = el('span', 'shelf-title-icon');
    titleIcon.appendChild(svgIcon('i-bookmark'));
    title.appendChild(titleIcon);
    title.appendChild(el('span', '', t('文件收纳')));
    title.appendChild(el('span', 'spacer'));
    const shelfViewBtn = viewButton();
    bindViewButton(shelfViewBtn, 'shelfView', 'shelfScale');
    title.appendChild(shelfViewBtn);
    section.appendChild(title);
    if (!data.groups.length) {
      section.classList.add('is-empty');
      section.appendChild(el('div', 'empty', t('拖入文件即自动分组')));
      return;
    }
    data.groups.forEach((group) => {
      const groupEl = el('div', 'shelf-group');
      groupEl.dataset.groupId = group.id;
      section.appendChild(groupEl);

      const head = el('div', 'group-head');
      head.dataset.groupId = group.id;

      const toggle = el('button', 'icon-btn sm');
      toggle.appendChild(svgIcon(group.collapsed ? 'i-right' : 'i-left'));
      toggle.title = group.collapsed ? t('展开') : t('收起');
      toggle.addEventListener('click', async () => {
        await Shelf.toggleGroup(group.id, !group.collapsed);
        load();
      });

      const nameEl = el('span', '', group.name);
      nameEl.style.cursor = 'text';
      nameEl.title = t('双击重命名');
      nameEl.addEventListener('dblclick', () => startRenameGroup(head, nameEl, group));

      const count = el('span', 'count');
      const bracket = el('span', 'count-num', String(itemsOf(group.id).length));
      count.appendChild(bracket);
      count.appendChild(el('span', 'count-unit', t('项')));
      const spacer = el('span', 'spacer');

      head.appendChild(toggle);
      head.appendChild(nameEl);
      head.appendChild(spacer);
      head.appendChild(count);
      groupEl.appendChild(head);

      head.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        openContextMenu(e.clientX, e.clientY, [
          { label: t('重命名分组'), icon: 'i-edit', action: () => startRenameGroup(head, nameEl, group) },
          {
            label: t('删除分组'),
            icon: 'i-trash',
            danger: true,
            action: async () => {
              await Shelf.deleteGroup(group.id);
              await load();
              onChanged && onChanged();
            },
          },
        ]);
      });

      if (group.collapsed) {
        attachGroupDrop(groupEl, head, null, group);
        return;
      }

      const grid = el('div', 'grid');
      grid.dataset.groupId = group.id;
      const items = itemsOf(group.id);
      if (!items.length) {
        const hint = el('div', 'empty', t('把文件拖到这里'));
        hint.style.gridColumn = '1 / -1';
        grid.appendChild(hint);
      } else {
        items.forEach((it) => grid.appendChild(renderItem(it, group)));
      }
      groupEl.appendChild(grid);
      attachGroupDrop(groupEl, head, grid, group);
    });
  }

  function renderMounts() {
    const section = el('div', 'mount-section');
    const head = el('div', 'mount-title');
    const titleIcon = el('span', 'mount-title-icon');
    titleIcon.appendChild(svgIcon('i-folder-open'));
    head.appendChild(titleIcon);
    head.appendChild(el('span', '', t('挂载目录')));
    head.appendChild(el('span', 'spacer'));
    const mountViewBtn = viewButton();
    bindViewButton(mountViewBtn, 'mountView', 'mountScale');
    head.appendChild(mountViewBtn);
    section.appendChild(head);

    const box = el('div', 'mount-list');
    if (!mounts.length) {
      box.appendChild(el('div', 'mount-empty', t('拖入文件夹以挂载')));
    }
    mounts.forEach((m) => {
      const row = el('div', 'entry-row mount-row');
      row.title = m.path;
      const icon = el('div', 'shelf-fallback');
      icon.appendChild(svgIcon('i-folder'));
      const nameWrap = el('div', 'entry-name');
      nameWrap.appendChild(el('div', '', m.label || m.path));
      const sub = el('div', 'entry-meta', m.path);
      sub.style.opacity = '0.7';
      nameWrap.appendChild(sub);
      const chev = el('div', 'entry-chev');
      chev.appendChild(svgIcon('i-right'));

      row.appendChild(icon);
      row.appendChild(nameWrap);
      row.appendChild(chev);
      row.addEventListener('click', () => openDir(m, '', []));
      row.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        openContextMenu(e.clientX, e.clientY, [
          {
            label: t('在资源管理器中打开'),
            icon: 'i-folder',
            action: () => Mount.reveal(m.id, '').catch(() => toast(t('打开失败'))),
          },
          {
            label: t('取消挂载'),
            icon: 'i-trash',
            danger: true,
            action: async () => {
              await Mount.remove(m.id);
              await load();
            },
          },
        ]);
      });
      box.appendChild(row);
    });
    section.appendChild(box);
    listEl.appendChild(section);
  }

  async function openDir(mount, rel, stack) {
    browsing = { mount, rel, stack };
    syncHead();
    renderCrumb();
    listEl.innerHTML = '<div class="skeleton">' + t('读取中…') + '</div>';
    let entries = [];
    try {
      entries = await Mount.children(mount.id, rel, {
        showHidden: false,
        sortBy: 'name',
        desc: false,
        limit: 400,
        offset: 0,
      });
    } catch (err) {
      listEl.innerHTML = '';
      listEl.appendChild(el('div', 'empty', t('无法读取该目录')));
      return;
    }
    renderEntries(entries);
  }

  function renderCrumb() {
    if (!browsing) return;
    const { mount, stack } = browsing;
    crumbEl.hidden = false;
    crumbEl.innerHTML = '';
    const rootBtn = el('button', '', mount.label || t('根目录'));
    rootBtn.addEventListener('click', () => openDir(mount, '', []));
    crumbEl.appendChild(rootBtn);
    stack.forEach((seg, idx) => {
      crumbEl.appendChild(el('span', '', '›'));
      const btn = el('button', '', seg.name);
      btn.addEventListener('click', () => openDir(mount, seg.rel, stack.slice(0, idx + 1)));
      crumbEl.appendChild(btn);
    });
  }

  function renderEntries(entries) {
    const { mount, rel, stack } = browsing;
    hintEl.textContent = tp('{n} 项', { n: entries.length });
    listEl.innerHTML = '';
    if (!entries.length) {
      listEl.appendChild(el('div', 'empty', t('这个文件夹是空的')));
      return;
    }
    entries.forEach((entry) => {
      const row = el('div', 'entry-row');
      const icon = el('img', 'entry-icon');
      icon.dataset.path = entry.path;
      const fallback = () => {
        const box = el('div', 'shelf-fallback');
        box.style.width = '20px';
        box.style.height = '20px';
        box.appendChild(svgIcon(entry.isDir ? 'i-folder' : 'i-file'));
        icon.replaceWith(box);
      };
      icon.addEventListener('error', fallback);
      const cached = iconCache.get(entry.path.toLowerCase());
      if (cached) icon.src = cached;

      const name = el('div', 'entry-name', entry.name);
      const meta = el('div', 'entry-meta', entry.isDir ? '' : fmtSize(entry.size));

      row.appendChild(icon);
      row.appendChild(name);
      row.appendChild(meta);

      if (entry.isDir) {
        const chev = el('div', 'entry-chev');
        chev.appendChild(svgIcon('i-right'));
        row.appendChild(chev);
      }

      row.addEventListener('click', async () => {
        if (entry.isDir) {
          const nextRel = rel ? `${rel}\\${entry.name}` : entry.name;
          await openDir(mount, nextRel, [...stack, { name: entry.name, rel: nextRel }]);
          return;
        }
        try {
          await Mount.open(mount.id, rel ? `${rel}\\${entry.name}` : entry.name);
        } catch (err) {
          toast(t('无法打开'));
        }
      });

      row.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        const targetRel = rel ? `${rel}\\${entry.name}` : entry.name;
        openContextMenu(e.clientX, e.clientY, [
          {
            label: t('打开'),
            icon: 'i-play',
            action: () => Mount.open(mount.id, targetRel).catch(() => toast(t('无法打开'))),
          },
          {
            label: t('在资源管理器中显示'),
            icon: 'i-folder',
            action: () => Mount.reveal(mount.id, targetRel).catch(() => toast(t('打开失败'))),
          },
          { sep: true },
          {
            label: t('收纳到侧边栏'),
            icon: 'i-plus',
            action: async () => {
              try {
                await Shelf.addPaths('', [entry.path]);
                toast(t('已收纳'));
              } catch (err) {
                toast(String(err).replace(/^Error:\s*/, ''));
              }
            },
          },
        ]);
      });

      listEl.appendChild(row);
    });

    loadIcons(entries.map((entry) => entry.path));
  }

  function renderItem(item, group) {
    const node = el('div', 'shelf-item');
    node.dataset.itemId = item.id;
    node.title = item.path;
    node.draggable = true;

    let iconEl;
    if (item.kind === 'url') {
      iconEl = el('div', 'shelf-fallback', 'URL');
    } else {
      iconEl = el('img', 'shelf-icon');
      iconEl.draggable = false;
      iconEl.dataset.path = item.path;
      const cached = iconCache.get(item.path.toLowerCase());
      if (cached) iconEl.src = cached;
      if (IMAGE_RE.test(item.path || '')) {
        iconEl.dataset.thumbPath = item.path;
        const thumb = thumbCache.get(item.path.toLowerCase());
        if (thumb) {
          iconEl.src = thumb;
          iconEl.dataset.thumbApplied = '1';
          iconEl.classList.add('shelf-thumb');
        }
      }
      iconEl.addEventListener('error', () => {
        const fallback = el('div', 'shelf-fallback', (item.path.split('.').pop() || '?').slice(0, 4));
        iconEl.replaceWith(fallback);
      });
    }

    const nameEl = el('div', 'shelf-name', item.name);
    node.appendChild(iconEl);
    node.appendChild(nameEl);

    node.addEventListener('dragstart', (e) => {
      draggedId = item.id;
      suppressClickUntil = Date.now() + 500;
      node.classList.add('dragging');
      if (e.dataTransfer) {
        e.dataTransfer.effectAllowed = 'move';
        try {
          e.dataTransfer.setData('text/plain', item.id);
        } catch (err) {}
      }
      hideDropLine();
    });

    node.addEventListener('dragend', () => {
      node.classList.remove('dragging');
      draggedId = null;
      suppressClickUntil = Math.max(suppressClickUntil, Date.now() + 150);
      clearDropTargets();
      hideDropLine();
    });

    node.addEventListener('click', async () => {
      if (Date.now() < suppressClickUntil) return;
      try {
        await Shelf.launch(item.id);
      } catch (err) {
        toast(t('无法打开：') + String(err).replace(/^Error:\s*/, ''));
      }
    });

    node.addEventListener('contextmenu', (e) => {
      e.preventDefault();
      e.stopPropagation();
      openContextMenu(e.clientX, e.clientY, [
        {
          label: t('打开'),
          icon: 'i-play',
          action: async () => {
            try {
              await Shelf.launch(item.id);
            } catch (err) {
              toast(t('无法打开'));
            }
          },
        },
        {
          label: t('在资源管理器中显示'),
          icon: 'i-folder',
          action: async () => {
            try {
              await Shelf.reveal(item.id);
            } catch (err) {
              toast(t('定位失败'));
            }
          },
        },
        { sep: true },
        { label: t('重命名'), icon: 'i-edit', action: () => startRenameItem(node, nameEl, item) },
        { sep: true },
        {
          label: t('从收纳区移除'),
          icon: 'i-trash',
          danger: true,
          action: async () => {
            await Shelf.remove(item.id);
            await load();
            onChanged && onChanged();
          },
        },
      ]);
    });

    return node;
  }

  function startRenameItem(node, nameEl, item) {
    const input = el('input');
    input.value = item.name;
    input.className = 'shelf-name';
    input.style.cssText = 'width:100%;background:rgba(0,0,0,0.3);border:1px solid rgba(255,255,255,0.16);border-radius:5px;padding:1px 3px;color:inherit;font-family:inherit;font-size:10.5px;outline:none;text-align:center';
    nameEl.replaceWith(input);
    input.focus();
    input.select();
    let done = false;
    const finish = async (commit) => {
      if (done) return;
      done = true;
      if (commit && input.value.trim() && input.value.trim() !== item.name) {
        try {
          await Shelf.update(item.id, input.value.trim(), null, null);
          onChanged && onChanged();
        } catch (err) {
          toast(t('重命名失败'));
        }
      }
      await load();
    };
    input.addEventListener('blur', () => finish(true));
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') finish(false);
      if (e.key === 'Enter') finish(true);
    });
  }

  function startRenameGroup(head, nameEl, group) {
    const input = el('input');
    input.value = group.name;
    input.style.cssText = 'width:110px;background:rgba(0,0,0,0.3);border:1px solid rgba(255,255,255,0.16);border-radius:5px;padding:1px 5px;color:inherit;font-family:inherit;font-size:11.5px;outline:none';
    nameEl.replaceWith(input);
    input.focus();
    input.select();
    let done = false;
    const finish = async (commit) => {
      if (done) return;
      done = true;
      if (commit && input.value.trim() && input.value.trim() !== group.name) {
        await Shelf.renameGroup(group.id, input.value.trim());
        onChanged && onChanged();
      }
      await load();
    };
    input.addEventListener('blur', () => finish(true));
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') finish(false);
      if (e.key === 'Enter') finish(true);
    });
  }

  function groupIdAt(x, y) {
    const target = document.elementFromPoint(x, y);
    if (!target) return null;
    const holder = target.closest('[data-group-id]');
    if (!holder) return null;
    return holder.dataset.groupId;
  }

  async function handleDrop(x, y, paths) {
    if (!paths || !paths.length) return;
    const target = document.elementFromPoint(x, y);
    if (target && target.closest('.mount-section')) {
      let mounted = 0;
      for (const p of paths) {
        try {
          await Mount.Add(p);
          mounted += 1;
        } catch (err) {
        }
      }
      if (!mounted) {
        toast(t('只能挂载文件夹'));
        return;
      }
      await load();
      toast(tp('已挂载 {n} 个文件夹', { n: mounted }));
      return;
    }
    let groupId = groupIdAt(x, y);
    if (!groupId) {
      if (!data.groups.length) {
        try {
          const created = await Shelf.createGroup(t('新分组'));
          groupId = created && created.id;
        } catch (err) {
        }
      } else {
        groupId = data.groups[0].id;
      }
    }
    if (!groupId) {
      toast(t('请先创建一个分组'));
      return;
    }
    try {
      const added = await Shelf.addPaths(groupId, paths);
      toast(tp('已收纳 {n} 项', { n: added ? added.length : 0 }));
      await load();
      onChanged && onChanged();
    } catch (err) {
      toast(String(err).replace(/^Error:\s*/, ''));
    }
  }

  function viewButton() {
    const btn = el('button', 'icon-btn sm view-btn');
    btn.title = t('显示方式');
    btn.appendChild(svgIcon('i-grid'));
    return btn;
  }

  function bindViewButton(btn, viewKey, scaleKey) {
    btn.addEventListener('click', async () => {
      let current;
      try {
        current = await App.getSettings();
      } catch (err) {
        toast(t('操作失败'));
        return;
      }
      const apply = (patch) => {
        App.updateSettings({ ...current, ...patch }).catch(() => toast(t('操作失败')));
      };
      const mark = (on) => (on ? '• ' : '');
      const rect = btn.getBoundingClientRect();
      openContextMenu(Math.max(4, rect.right - 150), rect.bottom + 4, [
        { label: mark(current[viewKey] !== 'list') + t('网格'), action: () => apply({ [viewKey]: 'grid' }) },
        { label: mark(current[viewKey] === 'list') + t('列表'), action: () => apply({ [viewKey]: 'list' }) },
        { sep: true },
        { label: mark(current[scaleKey] === 'compact') + t('紧凑'), action: () => apply({ [scaleKey]: 'compact' }) },
        { label: mark(current[scaleKey] === 'normal' || !current[scaleKey]) + t('标准'), action: () => apply({ [scaleKey]: 'normal' }) },
        { label: mark(current[scaleKey] === 'large') + t('大图标'), action: () => apply({ [scaleKey]: 'large' }) },
      ]);
    });
  }

  addGroupBtn.addEventListener('click', async () => {
    const group = await Shelf.createGroup(t('新分组'));
    await load();
    const head = listEl.querySelector(`.group-head[data-group-id="${group.id}"] span`);
    if (head) {
      head.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
    }
  });

  mountAddBtn.addEventListener('click', async () => {
    try {
      const m = await Mount.pick();
      if (m && m.id) {
        await load();
        toast(t('已挂载'));
      }
    } catch (err) {
      if (!/取消/.test(String(err))) toast(t('挂载失败'));
    }
  });

  backBtn.addEventListener('click', () => {
    if (!browsing) return;
    const stack = browsing.stack;
    if (!stack.length) {
      load();
      return;
    }
    const parentRel = stack.length > 1 ? stack[stack.length - 2].rel : '';
    openDir(browsing.mount, parentRel, stack.slice(0, stack.length - 1));
  });

  load();

  return { refresh: load, handleDrop };
}
