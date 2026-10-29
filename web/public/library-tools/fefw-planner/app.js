/* 挖角手帐 · 本地复刻
   纯前端实现：数据来自 data.js（自原页面客户端 bundle 中提取），
   状态保存在 localStorage，可导出 / 导入 JSON。
   与原页面不同的仅剩 B 站云存档（需要平台登录与后端），本版改为本地存档。 */
(function () {
  'use strict';

  var DATA = window.FEFW_DATA;
  if (!DATA) {
    document.body.innerHTML = '<p style="padding:40px;color:#d4646c">缺少 data.js，无法启动。</p>';
    return;
  }

  var ROUTES = DATA.routes;
  var CHARS = DATA.characters;
  var CLASSES = DATA.classes;
  var CHAR_BY_ID = {};
  var CLASS_BY_ID = {};
  CHARS.forEach(function (c) { CHAR_BY_ID[c.id] = c; });
  CLASSES.forEach(function (c) { CLASS_BY_ID[c.id] = c; });

  var TIER_ORDER = ['基本職', '初級職', '中級職', '上級職', '最上級職', '神将職'];
  var TIER_LABEL = {
    '基本職': '基本职',
    '初級職': '初级职',
    '中級職': '中级职',
    '上級職': '上级职',
    '最上級職': '最上级职',
    '神将職': '神将职'
  };
  function tierLabel(tier) { return TIER_LABEL[tier] || tier; }

  // 得意技能：判断转职条件是否命中角色专长，命中即高亮该职业。
  var PROFICIENCIES = window.FEFW_PROFICIENCIES || {};
  var PROF_NAMES = ['剑术', '枪术', '斧术', '箭术', '格斗术', '黑魔术', '白魔术', '指挥术', '步兵术', '马术', '重装术', '飞行术'];

  function charProf(c) { return (c && PROFICIENCIES[c.name]) || []; }

  // 特技与可习得：由 web/scripts/build-fefw-data.mjs 从攻略站生成，页面不联网。
  var SKILL_SHEET = window.FEFW_SKILL_INDEX || {};
  var SKILL_HOLDERS = window.FEFW_SKILL_HOLDERS || { arts: {}, magic: {} };

  // 职业转职条件里的熟练 → 该职业可用的战技武器类型。
  var WEAPON_BY_PROF = {
    '剑术': '剑', '枪术': '枪', '斧术': '斧', '箭术': '弓', '格斗术': '手甲',
    '黑魔术': '魔法', '白魔术': '魔法',
  };

  function classWeapons(cls) {
    if (!cls || !cls.examSummary) return [];
    var found = [];
    Object.keys(WEAPON_BY_PROF).forEach(function (prof) {
      if (cls.examSummary.indexOf(prof) >= 0) {
        var weapon = WEAPON_BY_PROF[prof];
        if (found.indexOf(weapon) < 0) found.push(weapon);
      }
    });
    return found;
  }

  /** 该职业 + 该角色可用的战技 / 魔法名称。 */
  function classSkillHint(cls, c) {
    var sheet = SKILL_SHEET[c.name];
    var weapons = classWeapons(cls);
    if (!sheet || !weapons.length) return null;
    var arts = (sheet.arts || []).filter(function (a) { return weapons.indexOf(a.weapon) >= 0; });
    var magic = weapons.indexOf('魔法') >= 0 ? (sheet.magic || []) : [];
    if (!arts.length && !magic.length) return null;
    return { arts: arts, magic: magic };
  }

  function holdersOf(kind, name) {
    var list = (SKILL_HOLDERS[kind] || {})[name] || [];
    return list;
  }


  function sheetGroup(title, count, rows) {
    if (!count) return '';
    return '<details class="sheet-group"><summary>' + esc(title) + ' <span class="sheet-count">' + count + '</span></summary>' +
      '<ul class="sheet-list">' + rows + '</ul></details>';
  }

  function skillRows(list, kindOf, holdersKind) {
    return list.map(function (s) {
      var meta = kindOf ? kindOf(s) : '';
      return '<li class="sheet-item" data-holders="' + esc(holdersKind) + '" data-name="' + esc(s.name) + '">' +
        '<button type="button" class="sheet-row">' +
          '<b>' + esc(s.name) + '</b>' +
          (meta ? '<span class="sheet-badge">' + esc(meta) + '</span>' : '') +
          '<span class="sheet-text">' + esc(s.text || '') + '</span>' +
        '</button>' +
        '<p class="sheet-holders" hidden></p>' +
      '</li>';
    }).join('');
  }

  function toggleHolders(item) {
    var panel = item.querySelector('.sheet-holders');
    if (!panel) return;
    if (!panel.hidden) { panel.hidden = true; item.classList.remove('open'); return; }
    var names = holdersOf(item.getAttribute('data-holders'), item.getAttribute('data-name'));
    panel.textContent = names.length ? '可学：' + names.join('、') : '仅该角色可用（站点未记录其他习得者）';
    panel.hidden = false;
    item.classList.add('open');
  }

  /** 转职条件里出现的所有武器/兵种熟练，例：任选其一 剑术 D 斧术 D → [剑术, 斧术]。 */
  function classProfReq(cls) {
    if (!cls || !cls.examSummary) return [];
    var found = [];
    PROF_NAMES.forEach(function (name) {
      if (cls.examSummary.indexOf(name) >= 0) found.push(name);
    });
    return found;
  }

  /** 命中的得意技能列表；非空即视为适配。 */
  function classFit(cls, c) {
    var own = charProf(c);
    if (!own.length) return [];
    return classProfReq(cls).filter(function (name) { return own.indexOf(name) >= 0; });
  }


  function chapterLabel(chapter) {
    if (!chapter) return '';
    return /^第/.test(chapter) ? chapter : '第' + chapter;
  }
  var BASE_CLASS = null;
  for (var baseIndex = 0; baseIndex < CLASSES.length; baseIndex++) {
    if (CLASSES[baseIndex].tier === '基本職' && CLASSES[baseIndex].nameZh === '平民') {
      BASE_CLASS = CLASSES[baseIndex];
      break;
    }
  }
  var STATUSES = ['unset', 'main', 'bench', 'skip'];
  var STATUS_LABEL = { unset: '未规划', main: '主力', bench: '板凳', skip: '不挖角' };
  var STORAGE_KEY = 'fefw-planner-reproduction:v1';
  // Account-backed storage. The document lives in LifeOS (SQLite, per user) so
  // the plan survives a cleared browser and is shared by web and desktop; the
  // localStorage copy stays as an offline cache and a migration source.
  var TOOL_KEY = 'fefw-planner';
  var STATE_URL = '/api/v1/library-tools/' + TOOL_KEY + '/state';
  var sync = { mode: 'local', revision: 0, timer: null, busy: false, conflict: false };

  var root = document.documentElement;
  var $ = function (sel) { return document.querySelector(sel); };
  var railEl = $('#rail');
  var routeHeadEl = $('#routeHead');
  var rosterEl = $('#roster');
  var routeViewEl = $('#routeView');
  var overviewViewEl = $('#overviewView');
  var searchEl = $('#search');
  var hideSkipEl = $('#hideSkip');
  var fameFilterEl = $('#fameFilter');
  var saveTextEl = $('#saveText');
  var saveStateEl = $('#saveState');
  var drawerEl = $('#drawer');
  var backdropEl = $('#backdrop');
  var toastEl = $('#toast');

  var state = loadState();
  var view = 'route'; // 'route' | 'overview'
  var drawerCharId = null;
  var lastFocus = null;
  var toastTimer = null;
  var saveTimer = null;
  var syncText = '仅保存在本机';
  var memoRosterTimer = null;

  /* ── helpers ─────────────────────────────────────────────── */

  function esc(v) {
    return String(v == null ? '' : v)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function hexToRgba(hex, alpha) {
    var h = hex.replace('#', '');
    if (h.length === 3) h = h.split('').map(function (x) { return x + x; }).join('');
    var n = parseInt(h, 16);
    return 'rgba(' + ((n >> 16) & 255) + ', ' + ((n >> 8) & 255) + ', ' + (n & 255) + ', ' + alpha + ')';
  }

  function routeById(id) {
    for (var i = 0; i < ROUTES.length; i++) if (ROUTES[i].id === id) return ROUTES[i];
    return ROUTES[0];
  }

  function currentRoute() { return routeById(state.activeRoute); }

  function sanitizeClassPath(src) {
    var out = {};
    if (!src || typeof src !== 'object') return out;
    TIER_ORDER.forEach(function (tier) {
      if (tier === '基本職') return; // 基本职固定为平民，不参与选择
      var id = src[tier];
      if (id && CLASS_BY_ID[id] && CLASS_BY_ID[id].tier === tier) out[tier] = id;
    });
    return out;
  }

  function emptyPlan() { return { status: 'unset', classPath: {}, note: '' }; }

  function getPlan(routeId, charId) {
    var r = state.plans[routeId];
    if (!r || !r[charId]) return emptyPlan();
    var p = r[charId];
    var path = sanitizeClassPath(p.classPath);
    // 旧存档迁移：单职业 → 该职业所属阶段
    if (!Object.keys(path).length && p.classId && CLASS_BY_ID[p.classId] && CLASS_BY_ID[p.classId].tier !== '基本職') {
      path[CLASS_BY_ID[p.classId].tier] = p.classId;
    }
    return {
      status: STATUSES.indexOf(p.status) >= 0 ? p.status : 'unset',
      classPath: path,
      note: typeof p.note === 'string' ? p.note : ''
    };
  }

  function setPlan(routeId, charId, patch) {
    if (!state.plans[routeId]) state.plans[routeId] = {};
    var cur = getPlan(routeId, charId);
    var path = patch.classPath != null ? sanitizeClassPath(patch.classPath) : cur.classPath;
    state.plans[routeId][charId] = {
      status: patch.status != null ? patch.status : cur.status,
      classPath: path,
      note: patch.note != null ? patch.note : cur.note
    };
    queueSave();
  }

  function nextStatus(s) {
    var i = STATUSES.indexOf(s);
    return STATUSES[(i + 1) % STATUSES.length];
  }

  function rosterOf(routeId) {
    return CHARS.filter(function (c) { return c.routes[routeId].kind !== 'unavailable'; });
  }

  function countsOf(routeId) {
    var n = { main: 0, bench: 0, skip: 0, unset: 0 };
    rosterOf(routeId).forEach(function (c) {
      var p = getPlan(routeId, c.id);
      n[p.status] += 1;
    });
    return n;
  }

  function allRosterCounts() {
    var total = { main: 0, bench: 0, skip: 0, unset: 0 };
    ROUTES.forEach(function (r) {
      var n = countsOf(r.id);
      total.main += n.main; total.bench += n.bench; total.skip += n.skip; total.unset += n.unset;
    });
    total.chars = ROUTES.reduce(function (acc, r) { return acc + rosterOf(r.id).length; }, 0);
    return total;
  }

  /* ── persistence ─────────────────────────────────────────── */

  function defaultState() {
    return {
      version: 1,
      activeRoute: ROUTES[0].id,
      hideSkip: false,
      search: '',
      fameFilter: '',
      plans: {}
    };
  }

  function loadState() {
    var base = defaultState();
    try {
      var raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return base;
      var parsed = JSON.parse(raw);
      return sanitizeState(parsed, base) || base;
    } catch (e) {
      return base;
    }
  }

  function sanitizeState(src, base) {
    if (!src || typeof src !== 'object') return null;
    var out = base;
    var routeIds = ROUTES.map(function (r) { return r.id; });
    if (routeIds.indexOf(src.activeRoute) >= 0) out.activeRoute = src.activeRoute;
    out.hideSkip = !!src.hideSkip;
    out.search = typeof src.search === 'string' ? src.search : '';
    var ff = typeof src.fameFilter === 'string' ? src.fameFilter : '';
    out.fameFilter = ff === '' || ff === 'story' || /^(10|[2-9])$/.test(ff) ? ff : '';
    out.plans = {};
    if (src.plans && typeof src.plans === 'object') {
      Object.keys(src.plans).forEach(function (rid) {
        if (routeIds.indexOf(rid) < 0) return;
        var bucket = src.plans[rid];
        if (!bucket || typeof bucket !== 'object') return;
        var clean = {};
        Object.keys(bucket).forEach(function (cid) {
          if (!CHAR_BY_ID[cid]) return;
          var p = bucket[cid];
          if (!p || typeof p !== 'object') return;
          var cp = sanitizeClassPath(p.classPath);
          if (!Object.keys(cp).length && p.classId && CLASS_BY_ID[p.classId] && CLASS_BY_ID[p.classId].tier !== '基本職') {
            cp[CLASS_BY_ID[p.classId].tier] = p.classId;
          }
          clean[cid] = {
            status: STATUSES.indexOf(p.status) >= 0 ? p.status : 'unset',
            classPath: cp,
            note: typeof p.note === 'string' ? p.note.slice(0, 4000) : ''
          };
        });
        if (Object.keys(clean).length) out.plans[rid] = clean;
      });
    }
    return out;
  }

  function queueSave() {
    saveStateEl.classList.add('saving');
    saveTextEl.textContent = '保存中…';
    clearTimeout(saveTimer);
    saveTimer = setTimeout(function () {
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
        saveTextEl.textContent = '本地已保存 · ' + pad(new Date().getHours()) + ':' + pad(new Date().getMinutes());
        saveStateEl.classList.remove('saving');
      } catch (e) {
        saveTextEl.textContent = '浏览器禁止本地保存';
        saveStateEl.classList.remove('saving');
      }
    }, 160);
    schedulePush();
  }

  /* ── account sync ────────────────────────────────────────── */
  // mode: 'server' = logged in and account storage in use,
  //       'local'  = not logged in / unreachable, localStorage only.

  function syncClock() {
    return pad(new Date().getHours()) + ':' + pad(new Date().getMinutes());
  }

  function setSyncStatus(text) {
    syncText = text;
    var el = document.getElementById('syncStatus');
    if (el) el.textContent = text;
  }

  function schedulePush() {
    if (sync.mode !== 'server' || sync.conflict) return;
    clearTimeout(sync.timer);
    sync.timer = setTimeout(pushState, 900);
  }

  function pushState(forceRevision) {
    if (sync.mode !== 'server' || sync.conflict) return;
    if (sync.busy) { schedulePush(); return; }
    sync.busy = true;
    var revision = typeof forceRevision === 'number' ? forceRevision : sync.revision;
    fetch(STATE_URL, {
      method: 'PUT',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ state: state, revision: revision })
    }).then(function (res) {
      if (res.status === 401) {
        sync.mode = 'local';
        setSyncStatus('未登录 · 仅保存在本机');
        return null;
      }
      if (res.status === 409) {
        sync.conflict = true;
        setSyncStatus('账号存档已在别处更新 · 请载入账号版本');
        toast('规划已在别处更新，请载入账号版本或导出本机备份');
        if (view === 'overview') renderOverview();
        return null;
      }
      if (!res.ok) throw new Error('HTTP ' + res.status);
      return res.json();
    }).then(function (body) {
      if (!body || !body.data) return;
      sync.revision = body.data.revision || sync.revision;
      setSyncStatus('已保存到账号 · ' + syncClock());
    }).catch(function () {
      setSyncStatus('同步失败 · 已保存在本机');
    })['finally'](function () {
      sync.busy = false;
    });
  }

  function adoptServerState(stateDoc, revision) {
    var clean = sanitizeState(stateDoc, defaultState());
    if (!clean) { toast('账号存档无法读取'); return false; }
    state = clean;
    sync.revision = revision || 0;
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); } catch (e) {}
    searchEl.value = state.search;
    fameFilterEl.value = state.fameFilter;
    if (hideSkipEl) hideSkipEl.checked = state.hideSkip;
    renderRouteHead();
    renderToolbar();
    renderRoster();
    renderRail();
    if (view === 'overview') renderOverview();
    return true;
  }

  function loadAccountState() {
    fetch(STATE_URL, { credentials: 'same-origin' }).then(function (res) {
      if (res.status === 401) {
        sync.mode = 'local';
        setSyncStatus('未登录 · 仅保存在本机');
        return null;
      }
      if (res.status === 404) {
        sync.mode = 'server';
        sync.revision = 0;
        var hasLocal = Object.keys(state.plans || {}).length > 0;
        setSyncStatus(hasLocal ? '本机有规划 · 可上传到账号' : '账号暂无存档 · 编辑后自动保存到账号');
        if (view === 'overview') renderOverview();
        return null;
      }
      if (!res.ok) throw new Error('HTTP ' + res.status);
      return res.json();
    }).then(function (body) {
      if (!body || !body.data) return;
      sync.mode = 'server';
      if (adoptServerState(body.data.state, body.data.revision)) {
        setSyncStatus('已载入账号存档 · ' + syncClock());
        if (view === 'overview') renderOverview();
      }
    }).catch(function () {
      sync.mode = 'local';
      setSyncStatus('无法连接账号存档 · 仅保存在本机');
    });
  }

  function pad(n) { return (n < 10 ? '0' : '') + n; }

  /* ── toast ───────────────────────────────────────────────── */

  function toast(msg) {
    toastEl.textContent = msg;
    toastEl.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { toastEl.classList.remove('show'); }, 2400);
  }

  /* ── rail ────────────────────────────────────────────────── */

  function renderRail() {
    var html = '<p class="rail-label">路线</p>';
    ROUTES.forEach(function (r) {
      var n = countsOf(r.id);
      var active = view === 'route' && state.activeRoute === r.id;
      html +=
        '<button type="button" class="route-tab' + (active ? ' active' : '') + '" data-route="' + r.id + '"' +
        ' style="--route:' + r.color + ';--route-soft:' + hexToRgba(r.color, 0.14) + '"' +
        (active ? ' aria-current="page"' : '') + '>' +
          '<span class="rnum">' + esc(r.number) + '</span>' +
          '<span class="rname"><b>' + esc(r.name) + '线</b></span>' +
          '<span class="rstat"><b>' + n.main + '</b>主 <b>' + n.bench + '</b>板</span>' +
        '</button>';
    });
    html += '<div class="rail-divider"></div>';
    html += '<button type="button" class="overview-tab' + (view === 'overview' ? ' active' : '') + '" data-view="overview">' +
      '<span>四线总览</span><span>' + allRosterCounts().chars + ' 人次</span></button>';
    html += '<p class="rail-note">四线共 ' + CHARS.length + ' 名可选角色。原页面支持 B 站云存档；本复刻使用浏览器本地存档，可在顶部导出 JSON 备份。</p>';
    railEl.innerHTML = html;
  }

  /* ── route head + toolbar ────────────────────────────────── */

  function renderRouteHead() {
    var r = currentRoute();
    var roster = rosterOf(r.id);
    var story = roster.filter(function (c) { return c.routes[r.id].kind === 'story'; }).length;
    var n = countsOf(r.id);
    routeHeadEl.innerHTML =
      '<div class="route-meta">' +
        '<div>' +
          '<p class="eyebrow">四线规划 · 第 ' + esc(r.number) + ' 线</p>' +
          '<h2>' + esc(r.name) + '线</h2>' +
          '<p class="route-count"><b>' + roster.length + '</b> 位角色 · 剧情 <b>' + story + '</b> · 挖角 <b>' + (roster.length - story) + '</b></p>' +
        '</div>' +
        '<div class="stats">' +
          '<div class="stat main"><b>' + n.main + '</b><span>主力</span></div>' +
          '<div class="stat bench"><b>' + n.bench + '</b><span>板凳</span></div>' +
          '<div class="stat skip"><b>' + n.skip + '</b><span>不挖角</span></div>' +
          '<div class="stat unset"><b>' + n.unset + '</b><span>未规划</span></div>' +
        '</div>' +
      '</div>';

    root.style.setProperty('--route', r.color);
    root.style.setProperty('--route-soft', hexToRgba(r.color, 0.14));
  }

  function renderToolbar() {
    if (searchEl.value !== state.search) searchEl.value = state.search;
    hideSkipEl.checked = state.hideSkip;
    if (fameFilterEl.value !== state.fameFilter) fameFilterEl.value = state.fameFilter;
  }

  /* ── roster ──────────────────────────────────────────────── */

  function diffClass(d) {
    if (d === '低') return 'diff-low';
    if (d === '中') return 'diff-mid';
    if (d === '高') return 'diff-high';
    if (d === '超') return 'diff-max';
    return '';
  }

  function rowHTML(routeId, c) {
    var entry = c.routes[routeId];
    var plan = getPlan(routeId, c.id);
    var isStory = entry.kind === 'story';
    var meta = '';
    if (!isStory) {
      if (entry.support != null) meta += '<span class="tag support">支援 ' + entry.support + '</span>';
      if (entry.difficulty) meta += '<span class="tag ' + diffClass(entry.difficulty) + '">交涉 ' + esc(entry.difficulty) + '</span>';
      if (entry.chapter) meta += '<span class="tag plain">' + esc(chapterLabel(entry.chapter)) + (entry.date ? ' · ' + esc(entry.date) : '') + '</span>';
      if (entry.extra) meta += '<span class="tag warn" title="' + esc(entry.extra) + '">另有条件</span>';
    } else if (entry.storyText) {
      meta += '<span class="tag plain">' + esc(entry.storyText) + '</span>';
    }
    var mainMarks = '';
    ROUTES.forEach(function (r) {
      if (r.id === routeId) return;
      if (c.routes[r.id].kind === 'unavailable') return;
      if (getPlan(r.id, c.id).status !== 'main') return;
      mainMarks += '<span class="tag mainroute" title="' + esc(r.name) + '线主力"><i style="background:' + r.color + '"></i>主力·' + esc(r.name) + '线</span>';
    });
    if (mainMarks) meta = mainMarks + meta;

    var fame = isStory
      ? '<span class="fame story">剧情</span>'
      : '<span class="fame" title="名声门槛 Lv.' + esc(entry.fame == null ? '—' : entry.fame) + '">Lv.' + esc(entry.fame == null ? '—' : entry.fame) + '</span>';
    var names = classPathNames(plan.classPath);
    var hasPath = names.length > 0;
    if (hasPath && BASE_CLASS) names = [BASE_CLASS.nameZh || BASE_CLASS.name].concat(names);
    var cls = hasPath ? ' has-plan' : '';
    var planInner = hasPath ? esc(names.join(' → ')) : '职业规划';
    var planTitle = hasPath ? '职业路线：' + classPathTitle(plan.classPath) : '设置职业路线';
    return '' +
      '<article class="char-card is-' + plan.status + '" data-char="' + c.id + '" data-route="' + routeId + '">' +
        '<button type="button" class="char-portrait" data-act="open" aria-label="查看 ' + esc(c.name) + ' 的详情">' +
          '<img src="' + esc(c.image) + '" alt="" loading="lazy" width="56" height="60">' +
          '<span class="char-status-bar" aria-hidden="true"></span>' +
        '</button>' +
        '<div class="char-main">' +
          '<div class="char-top">' +
            '<button type="button" class="char-name" data-act="open">' +
              '<b>' + esc(c.name) + '</b>' +
            '</button>' +
            fame +
            '<button type="button" class="statchip" data-act="status" data-status="' + plan.status + '" aria-label="' + esc(c.name) + ' 的状态：' + STATUS_LABEL[plan.status] + '，点击切换">' + STATUS_LABEL[plan.status] + '</button>' +
          '</div>' +
          '<div class="char-meta">' + meta + '</div>' +
          personalSkillLine(c) +
          '<div class="char-foot">' +
            '<button type="button" class="planbtn' + cls + '" data-act="plan" title="' + esc(planTitle) + '">' + planInner + '</button>' +
            '<span class="notedot' + (plan.note ? ' on' : '') + '" aria-hidden="true"></span>' +
          '</div>' +
        '</div>' +
      '</article>';
  }

  function matchFilter(c) {
    var q = state.search.trim().toLowerCase();
    if (!q) return true;
    return c.name.toLowerCase().indexOf(q) >= 0 || c.jp.toLowerCase().indexOf(q) >= 0;
  }

  function matchFame(c, routeId) {
    if (!state.fameFilter) return true;
    var entry = c.routes[routeId];
    if (state.fameFilter === 'story') return entry.kind === 'story';
    return entry.kind === 'recruit' && String(entry.fame) === state.fameFilter;
  }

  function visiblePlan(c, routeId) {
    if (!state.hideSkip) return true;
    return getPlan(routeId, c.id).status !== 'skip';
  }

  function personalSkillLine(c) {
    var sheet = SKILL_SHEET[c.name];
    var skill = sheet && sheet.personal;
    if (!skill || !skill.name) return '';
    return '<div class="char-skill" title="' + esc(skill.text || '') + '">' +
      '<span class="char-skill-label">个人特技</span>' + esc(skill.name) + '</div>';
  }

  function renderRoster() {
    var routeId = state.activeRoute;
    var roster = rosterOf(routeId).filter(function (c) {
      return matchFilter(c) && visiblePlan(c, routeId) && matchFame(c, routeId);
    });
    roster.sort(function (a, b) {
      var ea = a.routes[routeId], eb = b.routes[routeId];
      if (ea.kind === 'story' && eb.kind !== 'story') return -1;
      if (eb.kind === 'story' && ea.kind !== 'story') return 1;
      var fa = ea.fame == null ? 99 : ea.fame;
      var fb = eb.fame == null ? 99 : eb.fame;
      if (fa !== fb) return fa - fb;
      return a.name.localeCompare(b.name, 'zh');
    });

    if (!roster.length) {
      rosterEl.innerHTML = '<div class="empty">没有匹配的角色。<button type="button" data-act="clear-search">清除筛选</button></div>';
      return;
    }

    var storyN = roster.filter(function (c) { return c.routes[routeId].kind === 'story'; }).length;
    var html = '<div class="roster-bar"><span>按名声门槛排序</span><span class="roster-count">' +
      roster.length + ' 位' + (state.fameFilter ? '' : ' · 剧情 ' + storyN + ' · 可挖角 ' + (roster.length - storyN)) +
      '</span></div><div class="char-grid">';
    roster.forEach(function (c) { html += rowHTML(routeId, c); });
    html += '</div>';
    rosterEl.innerHTML = html;
  }

  /* ── drawer ──────────────────────────────────────────────── */

  function classOptionsForTier(tier, selectedId, ownerChar) {
    var list = CLASSES.filter(function (c) { return c.tier === tier; }).slice().sort(function (a, b) {
      return (a.nameZh || a.name).localeCompare(b.nameZh || b.name, 'zh');
    });
    var html = '<option value="">未选择</option>';
    list.forEach(function (c) {
      var fit = ownerChar ? classFit(c, ownerChar) : [];
      html += '<option value="' + esc(c.id) + '"' + (c.id === selectedId ? ' selected' : '') +
        (fit.length ? ' data-fit="1"' : '') + '>' +
        esc(c.nameZh || c.name) + (c.gender ? '（' + esc(c.gender) + '）' : '') +
        (fit.length ? ' ★' + fit.join('/') : '') + '</option>';
    });
    return html;
  }

  function classPathNames(path) {
    var names = [];
    TIER_ORDER.forEach(function (tier) {
      var c = CLASS_BY_ID[path[tier]];
      if (c) names.push(c.nameZh || c.name);
    });
    return names;
  }

  function classPathTitle(path) {
    var parts = [];
    if (BASE_CLASS) parts.push(tierLabel('基本職') + ' ' + (BASE_CLASS.nameZh || BASE_CLASS.name));
    TIER_ORDER.forEach(function (tier) {
      if (tier === '基本職') return;
      var c = CLASS_BY_ID[path[tier]];
      if (c) parts.push(tierLabel(tier) + ' ' + (c.nameZh || c.name));
    });
    return parts.join(' → ');
  }

  function cumulativeGrowth(path) {
    var sum = {};
    TIER_ORDER.forEach(function (tier) {
      var c = CLASS_BY_ID[path[tier]];
      if (!c || !c.growth) return;
      Object.keys(c.growth).forEach(function (k) {
        sum[k] = (sum[k] || 0) + (c.growth[k] || 0);
      });
    });
    return sum;
  }

  function growthText(sum) {
    var gnames = { hp: 'HP', str: '力量', mag: '魔力', spd: '速度', dex: '技巧', def: '防守', res: '魔防', lck: '幸运', cha: '魅力' };
    return Object.keys(sum)
      .filter(function (k) { return sum[k]; })
      .map(function (k) { return (gnames[k] || k) + ' ' + (sum[k] > 0 ? '+' : '') + sum[k] + '%'; })
      .join(' · ');
  }

  // 选中职业后，直接列出该职业武器类型下这名角色能用的战技/魔法。
  function classHintHTML(cls, c) {
    var hint = classSkillHint(cls, c);
    if (!hint) return '';
    var bits = [];
    if (hint.arts.length) {
      bits.push('<span class="stage-hint-label">可用战技</span>' +
        hint.arts.map(function (a) { return '<span class="stage-hint-chip">' + esc(a.name) + '</span>'; }).join(''));
    }
    if (hint.magic.length) {
      bits.push('<span class="stage-hint-label">可用魔法</span>' +
        hint.magic.map(function (m) { return '<span class="stage-hint-chip">' + esc(m.name) + '</span>'; }).join(''));
    }
    return '<div class="stage-hint">' + bits.join('') + '</div>';
  }

  function stageReqHTML(cls) {
    if (!cls) return '';
    var bits = [];
    if (cls.examSummary && cls.examSummary !== '—') bits.push('转职条件 ' + cls.examSummary);
    if (cls.unlockSummary) bits.push('解锁 ' + cls.unlockSummary);
    if (!bits.length) return '';
    var text = bits.join(' · ');
    return '<p class="stage-req" title="' + esc(text) + '">' + esc(text) + '</p>';
  }

  function renderDrawer() {
    var c = CHAR_BY_ID[drawerCharId];
    if (!c) return;
    var routeId = state.activeRoute;
    var plan = getPlan(routeId, c.id);
    var entry = c.routes[routeId];
    var here = routeById(routeId);

    var html = '' +
      '<div class="drawer-top">' +
        '<p class="eyebrow" style="margin:0;font-size:10px;letter-spacing:.24em;color:var(--dim)">角色档案</p>' +
        '<button type="button" class="iconbtn" data-act="close" aria-label="关闭详情">✕</button>' +
      '</div>' +
      '<div class="dossier">' +
        '<div class="dossier-portrait"><img src="' + esc(c.image) + '" alt="' + esc(c.name) + ' 的头像"></div>' +
        '<div>' +
          '<h2>' + esc(c.name) + '</h2>' +

          '<span class="route-chip"><i></i>' + esc(here.name) + '线 · ' + (entry.kind === 'story' ? '剧情加入' : entry.kind === 'recruit' ? '挖角' : '本线不可加入') + '</span>' +
        '</div>' +
      '</div>';

    html += '<div class="status-picker" role="group" aria-label="规划状态">';
    STATUSES.forEach(function (s) {
      html += '<button type="button" data-act="set-status" data-status="' + s + '" aria-pressed="' + (plan.status === s) + '">' + STATUS_LABEL[s] + '</button>';
    });
    html += '</div>';

    var path = plan.classPath;
    var own = charProf(c);
    html += '<div class="dsection"><h3>职业路线</h3>';
    html += '<p class="prof-line"><span class="prof-label">得意技能</span>' +
      (own.length ? own.map(function (n) { return '<span class="prof-chip">' + esc(n) + '</span>'; }).join('') : '<span class="prof-none">未收录（可在 data-proficiencies.js 补充）</span>') +
      '<span class="prof-hint">带 ★ 的职业与得意技能匹配</span></p>';
    html += '<div class="classpath">';
    TIER_ORDER.forEach(function (tier) {
      if (tier === '基本職') return;
      var selected = path[tier] || '';
      var cls = selected ? CLASS_BY_ID[selected] : null;
      var fit = cls ? classFit(cls, c) : [];
      html += '<div class="stage' + (fit.length ? ' fit' : '') + '">' +
        '<div class="stage-head"><span class="stage-name">' + esc(tierLabel(tier)) + '</span>' +
        '<select class="class-select" data-act="stage" data-tier="' + esc(tier) + '" aria-label="' + esc(tierLabel(tier)) + '职业选择">' +
        classOptionsForTier(tier, selected, c) + '</select>' +
        stageReqHTML(cls) +
        '</div>' +
        (fit.length ? '<p class="stage-fit">得意技能适配：' + esc(fit.join('、')) + '</p>' : '') +
        classHintHTML(cls, c) +
        '</div>';
    });
    html += '</div>';
    var growth = growthText(cumulativeGrowth(path));
    if (growth) html += '<p class="class-summary"><b>路线累计成长</b> ' + esc(growth) + '</p>';
    if (Object.keys(path).length) {
      html += '<button type="button" class="path-clear" data-act="clear-path">清空整条职业路线</button>';
    }
    html += '</div>';

    var sheet = SKILL_SHEET[c.name];
    html += '<div class="dsection"><h3>特技与可习得</h3>';
    if (!sheet) {
      html += '<p class="quote">本站暂未收录这名角色的技能资料。</p>';
    } else {
      if (sheet.personal && sheet.personal.name) {
        html += '<p class="sheet-personal"><b>个人特技 · ' + esc(sheet.personal.name) + '</b>' + esc(sheet.personal.text || '') + '</p>';
      }
      (sheet.blood || []).filter(function (b) { return b && b.name; }).forEach(function (b) {
        html += '<p class="sheet-personal sheet-blood"><b>血印 · ' + esc(b.name) + '</b>' + esc(b.text || '') + '</p>';
      });
      html += sheetGroup('Lv 特技', (sheet.lv || []).length, skillRows(sheet.lv || [], function (v) {
        return v.level == null ? '等级待补' : 'Lv.' + v.level;
      }, 'lv'));
      html += sheetGroup('可习得技能', (sheet.skills || []).length, skillRows(sheet.skills || [], function (s) { return s.kind; }, 'skills'));
      html += sheetGroup('战技', (sheet.arts || []).length, skillRows(sheet.arts || [], function (a) {
        return [a.weapon, a.cost ? '消耗 ' + a.cost : '', a.might ? '威力 ' + a.might : '', a.hit ? '命中 ' + a.hit : ''].filter(Boolean).join(' · ');
      }, 'arts'));
      html += sheetGroup('魔法', (sheet.magic || []).length, skillRows(sheet.magic || [], function (m) {
        return [m.type, m.rank, m.uses ? '次数 ' + m.uses : '', m.range ? '射程 ' + m.range : ''].filter(Boolean).join(' · ');
      }, 'magic'));
    }
    html += '</div>';

    var neg = entry.negotiation || c.negotiation || '';
    html += '<div class="dsection"><h3>交涉条件</h3>';
    html += '<p class="quote">' + (neg ? esc(neg) : '无需额外交涉材料，按剧情或对话条件加入。') + '</p>';
    html += '</div>';

    html += '<div class="dsection"><h3>四线招募情报</h3><div class="routetable">';
    ROUTES.forEach(function (r) {
      var e = c.routes[r.id];
      var off = e.kind === 'unavailable';
      var kind = e.kind === 'story' ? '剧情加入' : e.kind === 'recruit' ? '挖角' : '不可加入';
      var detail = '';
      if (e.kind === 'story') {
        detail = e.storyText ? esc(e.storyText) : '剧情加入';
        if (!e.storyText && e.chapter) {
          detail += ' · 章节 <b>' + esc(chapterLabel(e.chapter)) + '</b>' + (e.date ? ' · 日期 <b>' + esc(e.date) + '</b>' : '');
        }
      } else if (e.kind === 'recruit') {
        var bits = [];
        if (e.fame != null) bits.push('名声 <b>Lv.' + e.fame + '</b>');
        if (e.support != null) bits.push('支援 <b>' + e.support + '</b>');
        if (e.difficulty) bits.push('交涉 <b>' + esc(e.difficulty) + '</b>');
        if (e.chapter) bits.push('章节 <b>' + esc(chapterLabel(e.chapter)) + '</b>');
        if (e.date) bits.push('日期 <b>' + esc(e.date) + '</b>');
        if (e.extra) bits.push(esc(e.extra));
        detail = bits.join(' · ');
      } else {
        detail = '本线不可加入';
      }
      html += '<div class="rt' + (off ? ' off' : '') + (r.id === routeId ? ' here' : '') + '">' +
        '<span class="dot" style="background:' + r.color + '"></span>' +
        '<span class="rn">' + esc(r.name) + '线<br><span style="color:var(--dim);font-size:9.5px">' + kind + '</span></span>' +
        '<span class="rd">' + detail + '</span>' +
      '</div>';
    });
    html += '</div></div>';

    var mainRoutes = [];
    html += '<div class="dsection"><h3>四线规划</h3>';
    ROUTES.forEach(function (r) {
      if (c.routes[r.id].kind === 'unavailable') return;
      if (getPlan(r.id, c.id).status === 'main') mainRoutes.push(r.name + '线');
    });
    html += '<p class="rp-summary">' + (mainRoutes.length
      ? '主力：<b>' + esc(mainRoutes.join('、')) + '</b>'
      : '尚未标记主力路线') + '</p>';
    html += '<div class="routeplans">';
    ROUTES.forEach(function (r) {
      var unavailable = c.routes[r.id].kind === 'unavailable';
      var p = getPlan(r.id, c.id);
      html += '<div class="rp-row' + (r.id === routeId ? ' here' : '') + '">' +
        '<span class="dot" style="background:' + r.color + '"></span>' +
        '<span class="rp-name">' + esc(r.name) + '线</span>';
      if (unavailable) {
        html += '<span class="rp-off">本线不可加入</span>';
      } else {
        html += '<button type="button" class="statchip rp-status" data-act="status-route" data-route="' + r.id + '" data-status="' + p.status + '" aria-label="' + esc(r.name) + '线状态：' + STATUS_LABEL[p.status] + '，点击切换">' + STATUS_LABEL[p.status] + '</button>';
      }
      html += '</div>';
    });
    html += '</div></div>';

    html += '<div class="dsection"><h3>备注</h3>' +
      '<textarea class="memo" rows="3" placeholder="记录挖角时机、需要准备的资金或素材…">' + esc(plan.note) + '</textarea>' +
      '</div>';

    html += '<p class="dfoot">本页数据仅保存在本机浏览器 · 顶部「导出 JSON」可备份或迁移</p>';

    drawerEl.innerHTML = html;
  }

  function renderDrawerKeepScroll() {
    var top = drawerEl.scrollTop;
    renderDrawer();
    drawerEl.scrollTop = top;
  }

  function writeHash(value) {
    try { history.replaceState(null, '', '#' + value); } catch (e) {}
  }

  function readHash() {
    var h = (location.hash || '').replace(/^#/, '');
    if (!h) return null;
    if (h === 'overview') return { view: 'overview' };
    var parts = h.split('/');
    var validRoute = ROUTES.some(function (r) { return r.id === parts[0]; });
    if (parts.length === 2 && validRoute && CHAR_BY_ID[parts[1]]) return { route: parts[0], char: parts[1] };
    if (parts.length === 1 && validRoute) return { route: parts[0] };
    return null;
  }

  function openDrawer(charId) {
    drawerCharId = charId;
    lastFocus = document.activeElement;
    writeHash(state.activeRoute + '/' + charId);
    renderDrawer();
    drawerEl.classList.add('open');
    drawerEl.setAttribute('aria-hidden', 'false');
    backdropEl.hidden = false;
    requestAnimationFrame(function () { backdropEl.classList.add('show'); });
    var closeBtn = drawerEl.querySelector('[data-act="close"]');
    if (closeBtn) closeBtn.focus();
    document.body.style.overflow = 'hidden';
  }

  function closeDrawer() {
    if (!drawerEl.classList.contains('open')) return;
    drawerEl.classList.remove('open');
    drawerEl.setAttribute('aria-hidden', 'true');
    backdropEl.classList.remove('show');
    document.body.style.overflow = '';
    drawerCharId = null;
    writeHash(state.activeRoute);
    setTimeout(function () { if (!drawerEl.classList.contains('open')) backdropEl.hidden = true; }, 220);
    if (lastFocus && lastFocus.focus) lastFocus.focus();
    lastFocus = null;
  }

  /* ── overview ────────────────────────────────────────────── */

  function renderOverview() {
    var totals = allRosterCounts();
    var html = '<div class="overview">' +
      '<h2>四线总览</h2>' +
      '<p class="lede">' + CHARS.length + ' 名角色 · 四线共 ' + totals.chars + ' 个可招募席位 · 规划保存在本机</p>' +
      '<div class="ovgrid">';

    ROUTES.forEach(function (r) {
      var n = countsOf(r.id);
      var roster = rosterOf(r.id);
      var planned = n.main + n.bench + n.skip;
      var pct = function (v) { return roster.length ? (v / roster.length) * 100 : 0; };
      html += '<article class="ovcard" style="--card-route:' + r.color + '">' +
        '<h3>' + esc(r.name) + '线</h3>' +
        '<p class="ovjp">第 ' + esc(r.number) + ' 线 · ' + roster.length + ' 位角色</p>' +
        '<div class="ovnums">' +
          '<div class="ovnum main"><b>' + n.main + '</b><span>主力</span></div>' +
          '<div class="ovnum bench"><b>' + n.bench + '</b><span>板凳</span></div>' +
          '<div class="ovnum skip"><b>' + n.skip + '</b><span>不挖角</span></div>' +
          '<div class="ovnum unset"><b>' + n.unset + '</b><span>未规划</span></div>' +
        '</div>' +
        '<div class="bar" role="img" aria-label="已规划 ' + planned + ' / ' + roster.length + '">' +
          '<i class="m" style="width:' + pct(n.main) + '%"></i>' +
          '<i class="b" style="width:' + pct(n.bench) + '%"></i>' +
          '<i class="s" style="width:' + pct(n.skip) + '%"></i>' +
        '</div>' +
        '<p class="ovfoot">已规划 ' + planned + ' / ' + roster.length + '</p>' +
      '</article>';
    });
    html += '</div>';

    // class distribution
    var dist = {};
    ROUTES.forEach(function (r) {
      var bucket = state.plans[r.id] || {};
      Object.keys(bucket).forEach(function (cid) {
        var cp = bucket[cid] && bucket[cid].classPath;
        if (!cp || typeof cp !== 'object') return;
        TIER_ORDER.forEach(function (tier) {
          var id = cp[tier];
          if (id && CLASS_BY_ID[id] && CLASS_BY_ID[id].tier === tier) dist[id] = (dist[id] || 0) + 1;
        });
      });
    });
    var entries = Object.keys(dist).map(function (id) { return { id: id, n: dist[id] }; })
      .sort(function (a, b) { return b.n - a.n || a.id.localeCompare(b.id); });
    var maxN = entries.length ? entries[0].n : 1;
    html += '<div class="panel"><h3>职业路线分布（按阶段汇总）</h3>';
    if (!entries.length) {
      html += '<p class="classdist"><span class="none">还没有设置职业路线。在角色详情里按阶段选择职业后会汇总到这里。</span></p>';
    } else {
      html += '<div class="classdist">';
      entries.slice(0, 10).forEach(function (e) {
        var c = CLASS_BY_ID[e.id];
        html += '<div class="cd"><span class="cdname">' + esc(c.nameZh || c.name) + ' <span style="color:var(--dim)">' + esc(tierLabel(c.tier)) + '</span></span>' +
          '<span class="cdbar"><i style="width:' + ((e.n / maxN) * 100) + '%"></i></span>' +
          '<span class="cdnum">' + e.n + '</span></div>';
      });
      html += '</div>';
    }
    html += '</div>';

    html += '<div class="panel"><h3>存档</h3><div class="savegrid">' +
      '<button class="btn" type="button" data-action="export">导出 JSON</button>' +
      '<button class="btn ghost" type="button" data-action="import">导入 JSON</button>' +
      '<button class="btn ghost quiet" type="button" data-action="reset">清空规划</button>' +
      '</div>' +
      '<p class="note-line" id="syncStatus">' + esc(syncText) + '</p>' +
      '<div class="savegrid">' +
      '<button class="btn ghost" type="button" data-action="sync-upload">上传本机到账号</button>' +
      '<button class="btn ghost" type="button" data-action="sync-reload">载入账号版本</button>' +
      '</div>' +
      '<p class="note-line">规划随账号保存（Web 与桌面端共用同一份），本机同时保留离线缓存。<br>' +
      '原页面另支持 <b>B 站云存档</b> 与 <b>二维码备份</b>：云存档需要 B 站登录与平台后端，二维码需要额外的扫码解码模块，这两项未包含在本复刻中。</p>' +
      '</div>';

    html += '</div>';
    overviewViewEl.innerHTML = html;
  }

  /* ── views ───────────────────────────────────────────────── */

  function switchView(next) {
    view = next;
    if (view === 'overview') {
      routeViewEl.hidden = true;
      overviewViewEl.hidden = false;
      renderOverview();
      writeHash('overview');
    } else {
      overviewViewEl.hidden = true;
      routeViewEl.hidden = false;
      renderRouteHead();
      renderToolbar();
      renderRoster();
      writeHash(state.activeRoute);
    }
    renderRail();
    document.getElementById('main').scrollTop = 0;
  }

  function switchRoute(routeId) {
    state.activeRoute = routeId;
    state.search = '';
    queueSave();
    switchView('route');
  }

  function rerenderCurrent() {
    renderRail();
    if (view === 'route') {
      renderRouteHead();
      renderRoster();
    } else {
      renderOverview();
    }
  }

  /* ── export / import / reset ─────────────────────────────── */

  function exportJSON() {
    var payload = {
      app: 'fefw-planner-reproduction',
      version: 1,
      exportedAt: new Date().toISOString(),
      note: '万缕千丝 · 挖角手帐本地复刻存档',
      state: state
    };
    var blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' });
    var url = URL.createObjectURL(blob);
    var d = new Date();
    var name = '挖角手帐-' + d.getFullYear() + pad(d.getMonth() + 1) + pad(d.getDate()) + '-' + pad(d.getHours()) + pad(d.getMinutes()) + '.json';
    var a = document.createElement('a');
    a.href = url;
    a.download = name;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
    toast('已导出 ' + name);
  }

  function importJSON(file) {
    var reader = new FileReader();
    reader.onload = function () {
      var parsed;
      try { parsed = JSON.parse(String(reader.result)); } catch (e) { toast('文件不是有效的 JSON'); return; }
      var src = parsed && parsed.state ? parsed.state : parsed;
      if (!src || typeof src !== 'object' || !src.plans) { toast('这不是挖角手帐的备份文件'); return; }
      if (!window.confirm('导入会覆盖当前规划，确定继续？')) return;
      var clean = sanitizeState(src, defaultState());
      if (!clean) { toast('备份内容无法读取'); return; }
      state = clean;
      try { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); } catch (e) {}
      saveTextEl.textContent = '已导入备份';
      schedulePush();
      drawerEl.classList.remove('open');
      drawerEl.setAttribute('aria-hidden', 'true');
      backdropEl.classList.remove('show');
      backdropEl.hidden = true;
      switchView('route');
      renderRouteHead();
      renderToolbar();
      renderRoster();
      renderRail();
      toast('已导入规划');
    };
    reader.onerror = function () { toast('读取文件失败'); };
    reader.readAsText(file);
  }

  function resetAll() {
    if (!window.confirm('清空全部四线规划？此操作不可撤销。')) return;
    state = defaultState();
    try { localStorage.removeItem(STORAGE_KEY); } catch (e) {}
    saveTextEl.textContent = '已清空规划';
    schedulePush();
    switchView('route');
    renderToolbar();
    renderRouteHead();
    renderRoster();
    renderRail();
    toast('已清空全部规划');
  }

  function uploadLocal() {
    if (sync.mode !== 'server') { toast('先登录 LifeOS 再上传'); return; }
    if (sync.revision > 0 && !window.confirm('账号里已有规划，上传会覆盖它。确定继续？')) return;
    sync.conflict = false;
    pushState(0);
    toast('正在上传本机规划');
  }

  function reloadAccount() {
    sync.conflict = false;
    if (sync.mode !== 'server') sync.mode = 'server';
    loadAccountState();
    toast('正在载入账号存档');
  }

  /* ── events ──────────────────────────────────────────────── */

  railEl.addEventListener('click', function (e) {
    var tab = e.target.closest('.route-tab');
    if (tab) { switchRoute(tab.getAttribute('data-route')); return; }
    var ov = e.target.closest('[data-view="overview"]');
    if (ov) switchView('overview');
  });

  rosterEl.addEventListener('click', function (e) {
    var clear = e.target.closest('[data-act="clear-search"]');
    if (clear) {
      state.search = '';
      state.fameFilter = '';
      searchEl.value = '';
      fameFilterEl.value = '';
      queueSave();
      renderRoster();
      return;
    }

    var row = e.target.closest('.char-card');
    if (!row) return;
    var routeId = row.getAttribute('data-route');
    var charId = row.getAttribute('data-char');
    var btn = e.target.closest('button');
    var act = btn && btn.getAttribute('data-act');

    if (act === 'status') {
      var p = getPlan(routeId, charId);
      setPlan(routeId, charId, { status: nextStatus(p.status) });
      var np = getPlan(routeId, charId);
      btn.setAttribute('data-status', np.status);
      btn.textContent = STATUS_LABEL[np.status];
      btn.setAttribute('aria-label', CHAR_BY_ID[charId].name + ' 的状态：' + STATUS_LABEL[np.status] + '，点击切换');
      row.className = row.className.replace(/\bis-(unset|main|bench|skip)\b/, 'is-' + np.status);
      renderRouteHead();
      renderRail();
      if (state.hideSkip) renderRoster();
      return;
    }
    if (act === 'plan') { openDrawer(charId); return; }
    if (act === 'open') { openDrawer(charId); return; }
    openDrawer(charId);
  });

  drawerEl.addEventListener('click', function (e) {
    var btn = e.target.closest('button');
    if (!btn) return;
    var act = btn.getAttribute('data-act');
    if (act === 'close') { closeDrawer(); return; }
    if (btn.classList.contains('sheet-row')) {
      var item = btn.closest('.sheet-item');
      if (item) { toggleHolders(item); return; }
    }
    if (act === 'status-route') {
      var rid = btn.getAttribute('data-route');
      var cur = getPlan(rid, drawerCharId);
      setPlan(rid, drawerCharId, { status: nextStatus(cur.status) });
      renderDrawerKeepScroll();
      renderRoster();
      renderRouteHead();
      renderRail();
      return;
    }
    if (act === 'clear-path') {
      setPlan(state.activeRoute, drawerCharId, { classPath: {} });
      renderDrawerKeepScroll();
      renderRoster();
      return;
    }
    if (act === 'set-status') {
      var s = btn.getAttribute('data-status');
      setPlan(state.activeRoute, drawerCharId, { status: s });
      drawerEl.querySelectorAll('.status-picker button').forEach(function (b) {
        b.setAttribute('aria-pressed', String(b.getAttribute('data-status') === s));
      });
      renderRoster();
      renderRouteHead();
      renderRail();
      return;
    }
  });

  drawerEl.addEventListener('change', function (e) {
    var sel = e.target.closest('[data-act="stage"]');
    if (!sel) return;
    var tier = sel.getAttribute('data-tier');
    var cur = getPlan(state.activeRoute, drawerCharId);
    var path = cur.classPath;
    if (sel.value) path[tier] = sel.value;
    else delete path[tier];
    setPlan(state.activeRoute, drawerCharId, { classPath: path });
    renderDrawerKeepScroll();
    var again = drawerEl.querySelector('[data-act="stage"][data-tier="' + tier + '"]');
    if (again) again.focus({ preventScroll: true });
    renderRoster();
  });

  var memoState = { timer: null };
  drawerEl.addEventListener('input', function (e) {
    var memo = e.target.closest('.memo');
    if (!memo) return;
    var value = memo.value;
    var routeId = state.activeRoute;
    var charId = drawerCharId;
    clearTimeout(memoState.timer);
    memoState.timer = setTimeout(function () {
      setPlan(routeId, charId, { note: value });
      var dot = rosterEl.querySelector('.char-card[data-char="' + charId + '"] .notedot');
      if (dot) dot.classList.toggle('on', !!value.trim());
    }, 260);
  });

  backdropEl.addEventListener('click', closeDrawer);

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') { closeDrawer(); return; }
    if (e.key === '/' && !/^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName)) {
      e.preventDefault();
      searchEl.focus();
    }
    if (e.key === 'Tab' && drawerEl.classList.contains('open')) {
      var focusables = drawerEl.querySelectorAll('button, select, textarea, [href]');
      if (!focusables.length) return;
      var first = focusables[0];
      var last = focusables[focusables.length - 1];
      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    }
  });

  searchEl.addEventListener('input', function () {
    state.search = searchEl.value;
    renderRoster();
    queueSave();
  });

  hideSkipEl.addEventListener('change', function () {
    state.hideSkip = hideSkipEl.checked;
    renderRoster();
    queueSave();
  });

  fameFilterEl.addEventListener('change', function () {
    state.fameFilter = fameFilterEl.value;
    renderRoster();
    queueSave();
  });

  document.addEventListener('click', function (e) {
    var btn = e.target.closest('[data-action]');
    if (!btn) return;
    var action = btn.getAttribute('data-action');
    if (action === 'export') exportJSON();
    else if (action === 'import') $('#importFile').click();
    else if (action === 'reset') resetAll();
    else if (action === 'sync-upload') uploadLocal();
    else if (action === 'sync-reload') reloadAccount();
  });

  $('#importFile').addEventListener('change', function (e) {
    var file = e.target.files && e.target.files[0];
    if (file) importJSON(file);
    e.target.value = '';
  });

  /* ── boot ────────────────────────────────────────────────── */

  saveTextEl.textContent = '本地已保存 · ' + pad(new Date().getHours()) + ':' + pad(new Date().getMinutes());

  try {
    if (new URLSearchParams(location.search).get('from') === 'library') {
      var backLink = document.getElementById('backLink');
      if (backLink) backLink.hidden = false;
    }
  } catch (e) {}

  loadAccountState();

  var deep = readHash();
  if (deep && deep.route) state.activeRoute = deep.route;
  switchView(deep && deep.view ? 'overview' : 'route');
  if (deep && deep.char) openDrawer(deep.char);
})();
