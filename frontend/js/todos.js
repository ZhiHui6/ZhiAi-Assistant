import { Todos, el, svgIcon, todayStr, toast } from './api.js';
import { t, tp } from './strings.js';

export function createTodos({ onChanged }) {
  const input = document.getElementById('todoInput');
  const addBtn = document.getElementById('todoAdd');
  const scopesEl = document.getElementById('todoScopes');
  const dateChip = document.getElementById('dateChip');
  const listEl = document.getElementById('todoList');

  let scope = 'today';
  let dateFilter = null;
  let items = [];
  let stats = null;
  let loadSeq = 0;
  let doneLimit = 50;

  function currentFilter() {
    if (dateFilter) return { scope: 'date', date: dateFilter, limit: 0 };
    if (scope === 'done') return { scope, date: '', limit: doneLimit };
    return { scope, date: '', limit: 0 };
  }

  function applyScope(next) {
    scope = next;
    dateFilter = null;
    doneLimit = 50;
    dateChip.hidden = true;
    [...scopesEl.querySelectorAll('.chip')].forEach((c) => {
      if (c.id !== 'dateChip') c.classList.toggle('active', c.dataset.scope === scope);
    });
  }

  async function load(keepScroll) {
    const seq = ++loadSeq;
    const top = keepScroll ? listEl.scrollTop : 0;
    const [listRes, statsRes] = await Promise.allSettled([Todos.list(currentFilter()), Todos.stats()]);
    if (seq !== loadSeq) return;
    items = listRes.status === 'fulfilled' ? listRes.value : [];
    if (statsRes.status === 'fulfilled') stats = statsRes.value;
    render();
    renderCounts();
    renderMore();
    listEl.scrollTop = top;
  }

  function renderMore() {
    if (dateFilter || scope !== 'done' || !items.length || !stats || stats.done <= items.length) return;
    const more = el('button', 'todo-more', tp('显示更早的 {n} 条', { n: stats.done - items.length }));
    more.addEventListener('click', () => {
      doneLimit += 50;
      load(true);
    });
    listEl.appendChild(more);
  }

  function renderCounts() {
    const values = stats
      ? {
          today: stats.today + stats.todayDone,
          week: stats.week,
          overdue: stats.overdue,
          open: stats.open,
          done: stats.done,
        }
      : {};
    scopesEl.querySelectorAll('.chip[data-scope]').forEach((chip) => {
      const count = chip.querySelector('.chip-count');
      if (!count) return;
      const value = values[chip.dataset.scope];
      count.textContent = value ? String(value) : '';
      if (chip.dataset.scope === 'overdue') chip.classList.toggle('alert', value > 0);
    });
  }

  function render() {
    listEl.innerHTML = '';
    if (!items.length) {
      const messages = {
        today: t('今天没有到期的待办\n添加一条开始吧'),
        week: t('本周暂无待办'),
        overdue: t('没有逾期的待办'),
        open: t('没有未完成的待办'),
        done: t('还没有完成的待办'),
        date: t('这天没有安排'),
      };
      listEl.appendChild(el('div', 'empty', messages[dateFilter ? 'date' : scope] || t('暂无待办')));
      return;
    }
    items.forEach((t) => listEl.appendChild(todoRow(t)));
  }

  function revealItem(id) {
    const row = listEl.querySelector(`.todo[data-id="${id}"]`);
    if (!row) return;
    row.scrollIntoView({ block: 'nearest' });
    row.classList.add('flash');
    setTimeout(() => row.classList.remove('flash'), 1200);
  }

  function todoRow(todo) {
    const row = el('div', 'todo');
    row.dataset.id = todo.id;
    if (todo.done) row.classList.add('done');

    const check = el('button', 'todo-check');
    check.appendChild(svgIcon('i-check'));
    check.title = todo.done ? t('标记为未完成') : t('标记为完成');
    check.addEventListener('click', async () => {
      try {
        await Todos.toggle(todo.id, !todo.done);
        await load(true);
        onChanged && onChanged();
      } catch (err) {
        toast(t('操作失败'));
      }
    });

    const body = el('div', 'todo-body');
    const title = el('div', 'todo-title', todo.title);
    title.addEventListener('click', () => startEdit(title, todo));

    const sub = el('div', 'todo-sub');
    const prio = el('i', 'prio p' + (todo.priority ?? 1));
    prio.title = [t('低优先级'), t('中优先级'), t('高优先级')][todo.priority] || '';
    sub.appendChild(prio);
    if (todo.dueDate) {
      const due = el('span', 'due', formatDue(todo.dueDate));
      const today = todayStr();
      if (!todo.done && todo.dueDate < today) due.classList.add('overdue');
      else if (todo.dueDate === today) due.classList.add('today');
      sub.appendChild(due);
    }
    if (todo.group) sub.appendChild(el('span', 'tag', todo.group));
    if (todo.note) sub.appendChild(el('span', 'todo-note', todo.note.slice(0, 40)));

    const actions = el('div', 'card-actions');
    const del = el('button', 'icon-btn sm');
    del.appendChild(svgIcon('i-trash'));
    del.title = t('删除');
    del.addEventListener('click', async (e) => {
      e.stopPropagation();
      try {
        await Todos.remove(todo.id);
        await load(true);
        onChanged && onChanged();
        toast(tp('已删除「{title}」', { title: todo.title.slice(0, 12) }), {
          action: t('撤销'),
          duration: 4200,
          onAction: async () => {
            try {
              await Todos.restore(todo);
              await load(true);
              onChanged && onChanged();
            } catch (err) {
              toast(t('撤销失败'));
            }
          },
        });
      } catch (err) {
        toast(t('删除失败'));
      }
    });
    actions.appendChild(del);

    body.appendChild(title);
    body.appendChild(sub);
    row.appendChild(check);
    row.appendChild(body);
    row.appendChild(actions);
    return row;
  }

  function formatDue(dateStr) {
    const d = new Date(dateStr + 'T00:00:00');
    const today = todayStr();
    if (dateStr === today) return t('今天');
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    if (dateStr === todayStr(tomorrow)) return t('明天');
    return `${d.getMonth() + 1}/${d.getDate()}`;
  }

  function startEdit(titleEl, todo) {
    const inputEl = el('input');
    inputEl.value = todo.title;
    inputEl.style.cssText = 'width:100%;background:rgba(0,0,0,0.25);border:1px solid rgba(255,255,255,0.12);border-radius:6px;padding:3px 6px;color:inherit;font-family:inherit;font-size:12.8px;outline:none';
    titleEl.replaceWith(inputEl);
    inputEl.focus();
    inputEl.setSelectionRange(inputEl.value.length, inputEl.value.length);
    let done = false;
    const finish = async (commit) => {
      if (done) return;
      done = true;
      if (commit && inputEl.value.trim() && inputEl.value.trim() !== todo.title) {
        try {
          await Todos.update(todo.id, { title: inputEl.value.trim() });
          onChanged && onChanged();
        } catch (err) {
          toast(t('保存失败'));
        }
      }
      await load(true);
    };
    inputEl.addEventListener('blur', () => finish(true));
    inputEl.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        finish(false);
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        finish(true);
      }
    });
  }

  async function add() {
    const title = input.value.trim();
    if (!title) {
      input.focus();
      return;
    }
    let targetScope = scope;
    if (!dateFilter && (targetScope === 'overdue' || targetScope === 'done')) targetScope = 'today';
    const dueDate = dateFilter ? dateFilter : targetScope === 'open' ? null : todayStr();
    try {
      const created = await Todos.create({ title, priority: 1, dueDate, group: '' });
      input.value = '';
      if (!dateFilter) applyScope(targetScope);
      await load(true);
      revealItem(created.id);
      onChanged && onChanged();
    } catch (err) {
      toast(t('添加失败'));
    }
    input.focus();
  }

  addBtn.addEventListener('click', add);
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      add();
    }
  });

  scopesEl.addEventListener('click', (e) => {
    const chip = e.target.closest('.chip');
    if (!chip || !chip.dataset.scope || chip.id === 'dateChip') return;
    applyScope(chip.dataset.scope);
    load();
  });

  dateChip.addEventListener('click', () => {
    applyScope(scope);
    load();
  });

  load();

  return {
    refresh: load,
    focusInput: () => input.focus(),
    setDate(dateStr) {
      if (dateStr) {
        dateFilter = dateStr;
        dateChip.hidden = false;
        dateChip.textContent = `${dateStr.slice(5)} ×`;
        [...scopesEl.querySelectorAll('.chip')].forEach((c) => c.classList.remove('active'));
      } else {
        applyScope(scope);
      }
      load();
    },
  };
}
