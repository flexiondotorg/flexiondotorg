(function () {
  'use strict';

  var DECKS = [
    { key: 'projects', label: 'Projects', hint: 'Select any project card to flip for full details, releases, and Haberdasher package tiles.' },
    { key: 'packages', label: 'Packages', hint: 'First-party Haberdasher package repositories. Select a host to browse every published package.' },
    { key: 'links', label: 'Links', hint: 'Sites, community hubs, social profiles, video channels, gaming profiles, and sponsorship.' },
    { key: 'media', label: 'Media', hint: 'Linux Matters podcast episodes, subscribe links, and recent posts from the blog.' }
  ];

  var CARD_W = 262;
  var CARD_H = 258;
  var CARD_GAP_X = 24;
  var CARD_GAP_Y = 26;
  var STAGE_PAD_X = 24;
  var STAGE_PAD_Y = 26;

  var svgMountSeq = 0;

  var coarsePointerMQ = window.matchMedia ? window.matchMedia('(pointer: coarse)') : { matches: false };
  var reducedMotionMQ = window.matchMedia ? window.matchMedia('(prefers-reduced-motion: reduce)') : { matches: false };

  var state = {
    data: null,
    activeDeck: 'projects',
    sortMode: 'stars',
    viewMode: (coarsePointerMQ.matches || reducedMotionMQ.matches) ? 'grid' : 'floating',
    driftPaused: false,
    modalOpen: false,
    lastFocusedEl: null,
    cards: [],
    rafId: 0,
    lastFrameTime: 0,
    pointer: { active: false, x: 0, y: 0, vx: 0, vy: 0, lastX: 0, lastY: 0 },
    resizeTimer: 0
  };

  var els = {
    headerStats: document.getElementById('header-stats'),
    deckNav: document.getElementById('deck-nav'),
    sortControls: document.getElementById('sort-controls'),
    btnSortStars: document.getElementById('btn-sort-stars'),
    btnSortRelease: document.getElementById('btn-sort-release'),
    btnViewFloating: document.getElementById('btn-view-floating'),
    btnViewGrid: document.getElementById('btn-view-grid'),
    btnDriftToggle: document.getElementById('btn-drift-toggle'),
    btnThemeToggle: document.getElementById('btn-theme-toggle'),
    themeLabel: document.getElementById('theme-label'),
    activeHeading: document.getElementById('active-deck-heading'),
    activeHint: document.getElementById('active-deck-hint'),
    stage: document.getElementById('stage'),
    deckDock: document.getElementById('deck-dock'),
    liveStatus: document.getElementById('live-status'),
    modal: document.getElementById('card-modal'),
    modalFlipper: document.getElementById('modal-flipper'),
    modalFront: document.getElementById('modal-front'),
    modalBack: document.getElementById('modal-back')
  };

  function announce(msg) {
    if (!els.liveStatus) {
      return;
    }
    els.liveStatus.textContent = '';
    window.setTimeout(function () {
      els.liveStatus.textContent = msg;
    }, 20);
  }

  function isSafeURL(raw) {
    if (typeof raw !== 'string') {
      return false;
    }
    var trimmed = raw.trim().toLowerCase();
    return trimmed.indexOf('https://') === 0 || trimmed.indexOf('http://') === 0 || trimmed.indexOf('mailto:') === 0;
  }

  function isSafeImageURI(raw) {
    if (typeof raw !== 'string') {
      return false;
    }
    var trimmed = raw.trim().toLowerCase();
    if (trimmed.indexOf('https://') === 0 || trimmed.indexOf('http://') === 0) {
      return true;
    }
    return trimmed.indexOf('data:image/') === 0 && trimmed.indexOf(';base64,') !== -1;
  }

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) {
      node.className = className;
    }
    if (text !== undefined && text !== null) {
      node.textContent = String(text);
    }
    return node;
  }

  function linkEl(href, className, text) {
    var a = el('a', className, text);
    if (isSafeURL(href)) {
      a.href = href;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
    } else {
      a.href = '#';
    }
    return a;
  }

  // Safely parses inline SVG markup and namespaces any internal id/url(#...)
  // references per mount so simultaneous stage + modal renders never collide.
  function mountSVG(container, rawSVG) {
    if (!container || !rawSVG || typeof rawSVG !== 'string') {
      return false;
    }
    try {
      svgMountSeq += 1;
      var suffix = '-m' + svgMountSeq;
      var uniqueSVG = rawSVG
        .replace(/id="(vlogo-[^"]+)"/g, 'id="$1' + suffix + '"')
        .replace(/url\(#(vlogo-[^)]+)\)/g, 'url(#$1' + suffix + ')')
        .replace(/href="#(vlogo-[^"]+)"/g, 'href="#$1' + suffix + '"');
      if (/^<svg\b(?![^>]*\bxmlns=)/i.test(uniqueSVG.trim())) {
        uniqueSVG = uniqueSVG.replace(/^(\s*<svg\b)/i, '$1 xmlns="http://www.w3.org/2000/svg"');
      }

      var parser = new DOMParser();
      var doc = parser.parseFromString(uniqueSVG, 'image/svg+xml');
      var svgRoot = doc.documentElement;
      if (!svgRoot || svgRoot.nodeName.toLowerCase() !== 'svg') {
        return false;
      }
      if (svgRoot.querySelector('parsererror')) {
        return false;
      }
      var imported = document.importNode(svgRoot, true);
      imported.setAttribute('aria-hidden', 'true');
      imported.setAttribute('focusable', 'false');
      container.appendChild(imported);
      return true;
    } catch (_) {
      return false;
    }
  }

  function mountArtwork(container, item) {
    if (item.cover_url && isSafeImageURI(item.cover_url)) {
      container.classList.add('is-cover');
      var coverImg = document.createElement('img');
      coverImg.src = item.cover_url;
      coverImg.alt = '';
      coverImg.loading = 'lazy';
      coverImg.decoding = 'async';
      container.appendChild(coverImg);
      return;
    }
    if (item.logo_svg) {
      if (mountSVG(container, item.logo_svg)) {
        return;
      }
    }
    if (item.logo_data_uri && isSafeImageURI(item.logo_data_uri)) {
      var img = document.createElement('img');
      img.src = item.logo_data_uri;
      img.alt = '';
      img.loading = 'lazy';
      img.decoding = 'async';
      container.appendChild(img);
      return;
    }
    if (item.icon_svg) {
      container.classList.add('is-brand-icon');
      mountSVG(container, item.icon_svg);
    }
  }

  function deckItems(deckKey) {
    if (!state.data) {
      return [];
    }
    if (deckKey === 'projects') {
      var list = (state.data.projects || []).slice();
      if (state.sortMode === 'release') {
        list.sort(function (a, b) {
          if (b.published_unix !== a.published_unix) {
            return (b.published_unix || 0) - (a.published_unix || 0);
          }
          if (b.stars !== a.stars) {
            return (b.stars || 0) - (a.stars || 0);
          }
          return String(a.name || '').localeCompare(String(b.name || ''));
        });
      } else {
        list.sort(function (a, b) {
          if (b.stars !== a.stars) {
            return (b.stars || 0) - (a.stars || 0);
          }
          if (b.published_unix !== a.published_unix) {
            return (b.published_unix || 0) - (a.published_unix || 0);
          }
          return String(a.name || '').localeCompare(String(b.name || ''));
        });
      }
      return list;
    }
    if (deckKey === 'packages') {
      return state.data.packages || [];
    }
    if (deckKey === 'links') {
      return state.data.links || [];
    }
    if (deckKey === 'media') {
      return state.data.media || [];
    }
    return [];
  }

  function renderHeaderStats() {
    if (!els.headerStats || !state.data) {
      return;
    }
    els.headerStats.textContent = '';
    var s = state.data.stats || {};
    var pills = [];
    if (s.total_stars > 0) {
      pills.push({ value: s.stars_formatted || String(s.total_stars), label: 'GitHub stars' });
    }
    if (s.project_count > 0) {
      pills.push({ value: String(s.project_count), label: 'projects' });
    }
    if (s.host_count > 0) {
      pills.push({ value: String(s.host_count), label: 'package hosts' });
    }
    if (s.active_sponsors > 0) {
      pills.push({ value: String(s.active_sponsors), label: 'active sponsors' });
    } else if (s.past_sponsors > 0) {
      pills.push({ value: String(s.past_sponsors), label: 'past sponsors' });
    }
    if (s.discord_online > 0) {
      pills.push({ value: String(s.discord_online), label: 'online on Discord' });
    }

    pills.forEach(function (p) {
      var pill = el('div', 'stat-pill');
      pill.appendChild(el('span', 'stat-pill-value', p.value));
      pill.appendChild(el('span', 'stat-pill-label', p.label));
      els.headerStats.appendChild(pill);
    });
  }

  function renderDeckControls() {
    if (els.deckNav) {
      els.deckNav.textContent = '';
      DECKS.forEach(function (deck) {
        var count = deckItems(deck.key).length;
        var btn = el('button', 'deck-tab');
        btn.type = 'button';
        btn.dataset.deck = deck.key;
        btn.setAttribute('aria-pressed', String(state.activeDeck === deck.key));
        btn.appendChild(el('span', '', deck.label));
        btn.appendChild(el('span', 'deck-tab-count', count));
        btn.addEventListener('click', function () {
          switchDeck(deck.key, null);
        });
        els.deckNav.appendChild(btn);
      });
    }

    if (els.deckDock) {
      els.deckDock.textContent = '';
      DECKS.forEach(function (deck) {
        var count = deckItems(deck.key).length;
        var isFanned = state.activeDeck === deck.key;
        var btn = el('button', 'deck-stack-btn' + (isFanned ? ' is-fanned' : ''));
        btn.type = 'button';
        btn.dataset.deck = deck.key;
        btn.setAttribute('aria-pressed', String(isFanned));
        btn.appendChild(el('span', 'deck-stack-title', deck.label));
        btn.appendChild(el('span', 'deck-stack-meta', count + (count === 1 ? ' card' : ' cards')));
        btn.appendChild(el('span', 'deck-stack-state', isFanned ? 'Fanned' : 'Fan deck'));
        btn.addEventListener('click', function () {
          switchDeck(deck.key, btn);
        });
        els.deckDock.appendChild(btn);
      });
    }

    var deckMeta = DECKS.find(function (d) { return d.key === state.activeDeck; }) || DECKS[0];
    if (els.activeHeading) {
      els.activeHeading.textContent = deckMeta.label;
    }
    if (els.activeHint) {
      els.activeHint.textContent = deckMeta.hint;
    }
    if (els.sortControls) {
      els.sortControls.hidden = state.activeDeck !== 'projects';
    }
    if (els.btnSortStars) {
      els.btnSortStars.setAttribute('aria-pressed', String(state.sortMode === 'stars'));
    }
    if (els.btnSortRelease) {
      els.btnSortRelease.setAttribute('aria-pressed', String(state.sortMode === 'release'));
    }
    if (els.btnViewFloating) {
      els.btnViewFloating.setAttribute('aria-pressed', String(state.viewMode === 'floating'));
    }
    if (els.btnViewGrid) {
      els.btnViewGrid.setAttribute('aria-pressed', String(state.viewMode === 'grid'));
    }
    if (els.btnDriftToggle) {
      var showDrift = state.viewMode === 'floating' && !reducedMotionMQ.matches;
      els.btnDriftToggle.hidden = !showDrift;
      els.btnDriftToggle.setAttribute('aria-pressed', String(state.driftPaused));
      els.btnDriftToggle.textContent = state.driftPaused ? 'Resume drift' : 'Pause drift';
    }
  }

  function buildCardElement(deckKey, item, index) {
    var btn = el('button', 'portfolio-card');
    btn.type = 'button';
    btn.dataset.deck = deckKey;
    btn.dataset.index = String(index);
    btn.dataset.id = item.id || String(index);
    if (item.accent) {
      btn.classList.add('accent-' + item.accent);
    }

    var top = el('div', 'card-top');
    var topLeft = el('span', 'card-meta-left');
    var topRight = el('span', 'card-meta-right');
    var art = el('div', 'card-artwork');
    var body = el('div', 'card-body');
    var footer = el('div', 'card-footer');

    if (deckKey === 'projects') {
      if (item.language) {
        var dot = el('span', 'lang-dot');
        dot.style.backgroundColor = item.language_color || 'var(--mauve)';
        topLeft.appendChild(dot);
        topLeft.appendChild(document.createTextNode(item.language));
      } else {
        topLeft.textContent = item.owner || 'GitHub';
      }
      if (item.latest_tag) {
        var tagBadge = el('span', 'card-badge' + (item.fresh ? ' is-fresh' : ''), item.latest_tag);
        topRight.appendChild(tagBadge);
      }

      mountArtwork(art, item);
      body.appendChild(el('h3', 'card-title', item.name));
      body.appendChild(el('p', 'card-desc', item.description || item.full_name));

      var fLeft = el('span', 'card-footer-group');
      fLeft.appendChild(el('span', 'card-footer-item card-footer-strong', '★ ' + (item.stars_compact || item.stars || 0)));
      if (item.release_age) {
        fLeft.appendChild(el('span', 'card-footer-item', item.release_age));
      }
      footer.appendChild(fLeft);

      var hostCount = (item.hosts || []).length;
      var totalHosts = (state.data.packages || []).length || 8;
      footer.appendChild(el('span', 'card-footer-item', hostCount > 0 ? (hostCount + '/' + totalHosts + ' pkgs') : 'Source'));
      btn.setAttribute('aria-label', item.name + ', ' + (item.language || 'project') + ', ' + (item.stars_formatted || item.stars || 0) + ' stars. Open details.');
    } else if (deckKey === 'packages') {
      topLeft.textContent = item.display_name || item.name;
      topRight.appendChild(el('span', 'card-badge', item.package_count + ' pkgs'));
      mountArtwork(art, item);
      body.appendChild(el('h3', 'card-title', item.host));
      body.appendChild(el('p', 'card-desc', item.description));

      var chips = el('div', 'host-chip-row');
      var pkgs = item.packages || [];
      pkgs.slice(0, 2).forEach(function (pkg) {
        chips.appendChild(el('span', 'mini-chip', pkg.name));
      });
      if (pkgs.length > 2) {
        chips.appendChild(el('span', 'mini-chip', '+' + (pkgs.length - 2)));
      }
      footer.appendChild(chips);
      btn.setAttribute('aria-label', item.display_name + ' repository at ' + item.host + ', ' + item.package_count + ' packages. Open details.');
    } else if (deckKey === 'links') {
      topLeft.textContent = item.category || 'Link';
      if (item.badge) {
        topRight.appendChild(el('span', 'card-badge', item.badge));
      }
      mountArtwork(art, item);
      body.appendChild(el('h3', 'card-title', item.title));
      body.appendChild(el('p', 'card-desc', item.description || item.host_label));

      footer.appendChild(el('span', 'card-footer-item', item.host_label || item.url));
      footer.appendChild(el('span', 'card-footer-item card-footer-strong', 'Details'));
      btn.setAttribute('aria-label', item.title + ' (' + (item.category || 'Link') + '). Open details.');
    } else if (deckKey === 'media') {
      topLeft.textContent = item.category || 'Media';
      if (item.badge) {
        topRight.appendChild(el('span', 'card-badge', item.badge));
      }
      mountArtwork(art, item);
      body.appendChild(el('h3', 'card-title', item.title));
      body.appendChild(el('p', 'card-desc', item.summary || item.subtitle || ''));

      var mLeft = el('span', 'card-footer-group');
      if (item.duration) {
        mLeft.appendChild(el('span', 'card-footer-item card-footer-strong', item.duration));
      } else if (item.read_time_minutes > 0) {
        mLeft.appendChild(el('span', 'card-footer-item card-footer-strong', item.read_time_minutes + ' min read'));
      } else if (item.kind === 'podcast-hub') {
        mLeft.appendChild(el('span', 'card-footer-item card-footer-strong', 'Subscribe & episodes'));
      } else if (item.kind === 'blog-hub') {
        mLeft.appendChild(el('span', 'card-footer-item card-footer-strong', 'Recent posts'));
      }
      footer.appendChild(mLeft);
      if (item.age || item.date_label) {
        footer.appendChild(el('span', 'card-footer-item', item.age || item.date_label));
      }
      btn.setAttribute('aria-label', item.title + ' (' + (item.category || 'Media') + '). Open details.');
    }

    top.appendChild(topLeft);
    top.appendChild(topRight);
    btn.appendChild(top);
    btn.appendChild(art);
    btn.appendChild(body);
    btn.appendChild(footer);

    btn.addEventListener('click', function () {
      openCardModal(deckKey, item, btn);
    });

    btn.addEventListener('keydown', function (ev) {
      handleCardKeyNav(ev, index);
    });

    return btn;
  }

  function handleCardKeyNav(ev, index) {
    var keys = ['ArrowRight', 'ArrowLeft', 'ArrowDown', 'ArrowUp', 'Home', 'End'];
    if (keys.indexOf(ev.key) === -1) {
      return;
    }
    var cardButtons = Array.prototype.slice.call(els.stage.querySelectorAll('.portfolio-card'));
    if (cardButtons.length === 0) {
      return;
    }
    var cols = Math.max(1, Math.floor((els.stage.clientWidth - STAGE_PAD_X * 2 + CARD_GAP_X) / (CARD_W + CARD_GAP_X)));
    var next = index;
    if (ev.key === 'ArrowRight') {
      next = (index + 1) % cardButtons.length;
    } else if (ev.key === 'ArrowLeft') {
      next = (index - 1 + cardButtons.length) % cardButtons.length;
    } else if (ev.key === 'ArrowDown') {
      next = Math.min(cardButtons.length - 1, index + cols);
    } else if (ev.key === 'ArrowUp') {
      next = Math.max(0, index - cols);
    } else if (ev.key === 'Home') {
      next = 0;
    } else if (ev.key === 'End') {
      next = cardButtons.length - 1;
    }
    if (next !== index && cardButtons[next]) {
      ev.preventDefault();
      cardButtons[next].focus();
    }
  }

  // Deterministic pseudo-random helper in [-1, 1] for calm, reproducible air currents.
  function seeded(n) {
    var x = Math.sin(n * 12.9898 + 78.233) * 43758.5453;
    return (x - Math.floor(x)) * 2 - 1;
  }

  function computeConstellation(cardRecords, stageWidth, originPoint) {
    var count = cardRecords.length;
    if (count === 0) {
      els.stage.style.minHeight = '260px';
      return;
    }

    var usableW = Math.max(CARD_W + STAGE_PAD_X * 2, stageWidth);
    var cols = Math.max(1, Math.floor((usableW - STAGE_PAD_X * 2 + CARD_GAP_X) / (CARD_W + CARD_GAP_X)));
    cols = Math.min(cols, count, 4);

    var rows = Math.ceil(count / cols);
    var maxBottom = 0;

    for (var i = 0; i < count; i += 1) {
      var row = Math.floor(i / cols);
      var col = i % cols;
      var itemsInRow = row === rows - 1 ? count - row * cols : cols;
      var rowWidth = itemsInRow * CARD_W + Math.max(0, itemsInRow - 1) * CARD_GAP_X;
      var startX = Math.max(STAGE_PAD_X, (usableW - rowWidth) / 2);

      var waveY = Math.sin((col + 0.5) * 1.35 + row * 0.9) * 11;
      var waveX = (row % 2 === 1 && itemsInRow < cols) ? 0 : seeded(i + 1) * 6;
      var homeX = Math.max(12, Math.min(usableW - CARD_W - 12, startX + col * (CARD_W + CARD_GAP_X) + waveX));
      var homeY = STAGE_PAD_Y + row * (CARD_H + CARD_GAP_Y) + waveY + 10;
      var baseRot = seeded(i + 17) * 2.1;

      var rec = cardRecords[i];
      rec.homeX = homeX;
      rec.homeY = homeY;
      rec.baseRot = baseRot;

      if (originPoint) {
        rec.x = originPoint.x;
        rec.y = originPoint.y;
        rec.rot = seeded(i + 31) * 10;
      } else {
        rec.x = homeX;
        rec.y = homeY;
        rec.vx = 0;
        rec.vy = 0;
        rec.rot = baseRot;
      }
      rec.positioned = true;
      rec.el.style.transform = 'translate3d(' + rec.x.toFixed(2) + 'px, ' + rec.y.toFixed(2) + 'px, 0) rotate(' + rec.rot.toFixed(2) + 'deg)';

      if (homeY + CARD_H > maxBottom) {
        maxBottom = homeY + CARD_H;
      }
    }

    els.stage.style.minHeight = Math.max(460, Math.ceil(maxBottom + STAGE_PAD_Y + 18)) + 'px';
  }

  function renderStage(originEl) {
    stopAnimation();
    if (!els.stage) {
      return;
    }

    var items = deckItems(state.activeDeck);
    els.stage.textContent = '';
    els.stage.classList.toggle('is-floating', state.viewMode === 'floating');
    els.stage.classList.toggle('is-grid', state.viewMode === 'grid');
    els.stage.style.minHeight = '';

    state.cards = [];

    if (items.length === 0) {
      els.stage.appendChild(el('p', 'deck-hint', 'No cards available in this deck.'));
      return;
    }

    var fragment = document.createDocumentFragment();
    items.forEach(function (item, idx) {
      var cardEl = buildCardElement(state.activeDeck, item, idx);
      var rec = {
        el: cardEl,
        homeX: 0,
        homeY: 0,
        x: 0,
        y: 0,
        vx: 0,
        vy: 0,
        rot: 0,
        baseRot: 0,
        phaseX: idx * 1.15 + 0.4,
        phaseY: idx * 0.87 + 1.3,
        phaseRot: idx * 1.42,
        freqX: 0.42 + (idx % 5) * 0.07,
        freqY: 0.51 + (idx % 4) * 0.08,
        settled: false,
        damp: 0,
        positioned: false
      };

      if (state.viewMode === 'floating') {
        cardEl.addEventListener('pointerenter', function () {
          rec.settled = true;
          cardEl.style.zIndex = '40';
        });
        cardEl.addEventListener('pointerleave', function () {
          rec.settled = false;
          cardEl.style.zIndex = '';
        });
        cardEl.addEventListener('focus', function () {
          rec.settled = true;
          cardEl.style.zIndex = '45';
        });
        cardEl.addEventListener('blur', function () {
          rec.settled = false;
          cardEl.style.zIndex = '';
        });
      }

      state.cards.push(rec);
      fragment.appendChild(cardEl);
    });

    els.stage.appendChild(fragment);

    if (state.viewMode === 'floating') {
      var originPoint = null;
      if (originEl && !reducedMotionMQ.matches) {
        var stageRect = els.stage.getBoundingClientRect();
        var origRect = originEl.getBoundingClientRect();
        originPoint = {
          x: Math.max(12, Math.min(stageRect.width - CARD_W - 12, origRect.left - stageRect.left + origRect.width / 2 - CARD_W / 2)),
          y: Math.max(12, Math.min(stageRect.height - CARD_H, origRect.top - stageRect.top - 40))
        };
      }
      computeConstellation(state.cards, els.stage.clientWidth, originPoint);
      startAnimation();
    }
  }

  function startAnimation() {
    if (state.viewMode !== 'floating' || state.driftPaused || reducedMotionMQ.matches || state.modalOpen) {
      return;
    }
    if (state.rafId) {
      return;
    }
    state.lastFrameTime = performance.now();
    state.rafId = window.requestAnimationFrame(tickPhysics);
  }

  function stopAnimation() {
    if (state.rafId) {
      window.cancelAnimationFrame(state.rafId);
      state.rafId = 0;
    }
  }

  function tickPhysics(now) {
    state.rafId = 0;
    if (state.viewMode !== 'floating' || state.driftPaused || reducedMotionMQ.matches || state.modalOpen) {
      return;
    }

    var dt = Math.min(0.05, Math.max(0.004, (now - state.lastFrameTime) / 1000));
    state.lastFrameTime = now;
    var t = now / 1000;

    var wakeX = state.pointer.x;
    var wakeY = state.pointer.y;
    var wakeVX = state.pointer.vx;
    var wakeVY = state.pointer.vy;
    var wakeActive = state.pointer.active && (Math.abs(wakeVX) > 0.4 || Math.abs(wakeVY) > 0.4);

    // Decay pointer wake velocity between pointermove events.
    state.pointer.vx *= 0.84;
    state.pointer.vy *= 0.84;

    for (var i = 0; i < state.cards.length; i += 1) {
      var c = state.cards[i];
      var targetDamp = c.settled ? 1 : 0;
      c.damp += (targetDamp - c.damp) * Math.min(1, dt * 10);

      var driftScale = 1 - c.damp;
      var airX = Math.sin(t * c.freqX + c.phaseX) * 7.5 * driftScale;
      var airY = Math.cos(t * c.freqY + c.phaseY) * 6.5 * driftScale;
      var targetX = c.homeX + airX;
      var targetY = c.homeY + airY;
      var targetRot = (c.baseRot + Math.sin(t * 0.55 + c.phaseRot) * 1.15) * driftScale;

      // Apply soft radial pointer-wake breeze when not hovered/focused.
      if (wakeActive && !c.settled) {
        var cx = c.x + CARD_W / 2;
        var cy = c.y + CARD_H / 2;
        var dx = cx - wakeX;
        var dy = cy - wakeY;
        var dist = Math.hypot(dx, dy);
        if (dist > 8 && dist < 230) {
          var falloff = (1 - dist / 230) * (1 - dist / 230);
          c.vx += (wakeVX * 0.42 + (dx / dist) * 14) * falloff * dt * 10;
          c.vy += (wakeVY * 0.42 + (dy / dist) * 14) * falloff * dt * 10;
        }
      }

      // Spring towards air-current target with smooth damping.
      var stiffness = c.settled ? 14 : 6.2;
      var damping = c.settled ? 0.72 : 0.88;
      c.vx = (c.vx + (targetX - c.x) * stiffness * dt) * Math.pow(damping, dt * 60);
      c.vy = (c.vy + (targetY - c.y) * stiffness * dt) * Math.pow(damping, dt * 60);

      // Bound velocity so cards stay composed on the stage.
      var speed = Math.hypot(c.vx, c.vy);
      if (speed > 260) {
        c.vx = (c.vx / speed) * 260;
        c.vy = (c.vy / speed) * 260;
      }

      c.x += c.vx * dt;
      c.y += c.vy * dt;
      c.rot += (targetRot - c.rot) * Math.min(1, dt * 8);

      var lift = c.settled ? -5 : 0;
      c.el.style.transform =
        'translate3d(' + c.x.toFixed(2) + 'px, ' + (c.y + lift).toFixed(2) + 'px, 0) rotate(' + c.rot.toFixed(2) + 'deg)';
    }

    state.rafId = window.requestAnimationFrame(tickPhysics);
  }

  function switchDeck(deckKey, originEl) {
    if (state.activeDeck === deckKey) {
      return;
    }
    state.activeDeck = deckKey;
    renderDeckControls();
    renderStage(originEl);
    var meta = DECKS.find(function (d) { return d.key === deckKey; });
    announce('Showing ' + (meta ? meta.label : deckKey) + ' deck, ' + deckItems(deckKey).length + ' cards.');
  }

  function setSortMode(mode) {
    if (state.sortMode === mode) {
      return;
    }
    state.sortMode = mode;
    renderDeckControls();
    if (state.activeDeck === 'projects') {
      renderStage(null);
      announce('Projects sorted by ' + (mode === 'release' ? 'latest release.' : 'GitHub stars.'));
    }
  }

  function setViewMode(mode) {
    if (state.viewMode === mode) {
      return;
    }
    state.viewMode = mode;
    renderDeckControls();
    renderStage(null);
    announce('Switched to ' + (mode === 'floating' ? 'Floating cards view.' : 'Grid view.'));
  }

  function toggleDrift() {
    state.driftPaused = !state.driftPaused;
    renderDeckControls();
    if (state.driftPaused) {
      stopAnimation();
      announce('Card drift paused.');
    } else {
      startAnimation();
      announce('Card drift resumed.');
    }
  }

  function syncThemeUI() {
    var current = document.documentElement.dataset.theme === 'latte' ? 'latte' : 'mocha';
    if (els.themeLabel) {
      els.themeLabel.textContent = current === 'latte' ? 'Latte' : 'Mocha';
    }
    if (els.btnThemeToggle) {
      els.btnThemeToggle.setAttribute('aria-label', current === 'latte' ? 'Switch to Catppuccin Mocha dark theme' : 'Switch to Catppuccin Latte light theme');
    }
  }

  function toggleTheme() {
    var next = document.documentElement.dataset.theme === 'latte' ? 'mocha' : 'latte';
    document.documentElement.dataset.theme = next;
    try {
      localStorage.setItem('wimpys-world-theme', next);
    } catch (_) {}
    syncThemeUI();
    announce('Switched theme to Catppuccin ' + (next === 'latte' ? 'Latte.' : 'Mocha.'));
  }

  function buildSubscribeRow(links) {
    var row = el('div', 'subscribe-row');
    links.forEach(function (sub) {
      var pill = linkEl(sub.url, 'subscribe-pill');
      if (sub.icon_svg) {
        var iconSpan = el('span');
        mountSVG(iconSpan, sub.icon_svg);
        pill.appendChild(iconSpan);
      }
      pill.appendChild(document.createTextNode(sub.label));
      row.appendChild(pill);
    });
    return row;
  }

  function buildModalBack(deckKey, item) {
    var scroll = el('div', 'modal-scroll');

    // Header
    var header = el('div', 'modal-header');
    var titleGroup = el('div', 'modal-title-group');
    var art = el('div', 'modal-art');
    mountArtwork(art, item);

    var titleCol = el('div');
    var titleText = item.name || item.title || item.host || 'Details';
    var h2 = el('h2', 'modal-title', titleText);
    h2.id = 'modal-title';
    titleCol.appendChild(h2);

    var eyebrow = '';
    if (deckKey === 'projects') {
      eyebrow = item.full_name || '';
    } else if (deckKey === 'packages') {
      eyebrow = (item.display_name || item.name) + ' · ' + item.host;
    } else if (deckKey === 'links') {
      eyebrow = (item.category || 'Link') + (item.host_label ? ' · ' + item.host_label : '');
    } else if (deckKey === 'media') {
      var parts = [item.category || 'Media'];
      if (item.episode) {
        parts.push('Episode #' + item.episode);
      }
      if (item.date_label) {
        parts.push(item.date_label);
      }
      eyebrow = parts.join(' · ');
    }
    if (eyebrow) {
      titleCol.appendChild(el('p', 'modal-eyebrow', eyebrow));
    }

    titleGroup.appendChild(art);
    titleGroup.appendChild(titleCol);

    var closeBtn = el('button', 'modal-close', '×');
    closeBtn.type = 'button';
    closeBtn.setAttribute('aria-label', 'Close details');
    closeBtn.addEventListener('click', closeCardModal);

    header.appendChild(titleGroup);
    header.appendChild(closeBtn);
    scroll.appendChild(header);

    // Deck-specific detail body
    if (deckKey === 'projects') {
      if (item.description) {
        scroll.appendChild(el('p', 'modal-desc', item.description));
      }

      var stats = el('div', 'detail-stats');
      stats.appendChild(el('span', 'detail-stat', '★ ' + (item.stars_formatted || item.stars || 0) + ' stars'));
      if (item.language) {
        var langChip = el('span', 'detail-stat');
        var dot = el('span', 'lang-dot');
        dot.style.backgroundColor = item.language_color || 'var(--mauve)';
        langChip.appendChild(dot);
        langChip.appendChild(document.createTextNode(item.language));
        stats.appendChild(langChip);
      }
      if (item.latest_tag) {
        var relLabel = item.latest_tag + (item.release_age ? ' · ' + item.release_age : '');
        if (item.release_url && isSafeURL(item.release_url)) {
          stats.appendChild(linkEl(item.release_url, 'detail-stat', 'Release ' + relLabel));
        } else {
          stats.appendChild(el('span', 'detail-stat', 'Release ' + relLabel));
        }
      }
      scroll.appendChild(stats);

      var actions = el('div', 'detail-actions');
      if (item.homepage && isSafeURL(item.homepage) && item.homepage !== item.repo_url) {
        actions.appendChild(linkEl(item.homepage, 'action-btn is-primary', 'Visit Homepage'));
      }
      if (item.repo_url && isSafeURL(item.repo_url)) {
        actions.appendChild(linkEl(item.repo_url, 'action-btn' + (actions.childElementCount === 0 ? ' is-primary' : ''), 'View on GitHub'));
      }
      if (item.release_url && isSafeURL(item.release_url)) {
        actions.appendChild(linkEl(item.release_url, 'action-btn', 'Latest Release (' + item.latest_tag + ')'));
      }
      scroll.appendChild(actions);

      if (item.hosts && item.hosts.length > 0) {
        var hostSec = el('div', 'detail-section');
        hostSec.appendChild(el('h3', 'detail-section-title', 'Install via Haberdasher Packages'));
        var grid = el('div', 'host-tile-grid');
        var logos = (state.data && state.data.host_logos) || {};

        item.hosts.forEach(function (h) {
          var tile = linkEl(h.url, 'host-tile');
          var logoBox = el('span', 'host-tile-logo');
          if (logos[h.host]) {
            mountSVG(logoBox, logos[h.host]);
          }
          tile.appendChild(logoBox);
          tile.appendChild(el('span', '', h.name));
          grid.appendChild(tile);
        });
        hostSec.appendChild(grid);
        scroll.appendChild(hostSec);
      }
    } else if (deckKey === 'packages') {
      if (item.description) {
        scroll.appendChild(el('p', 'modal-desc', item.description));
      }

      var pkgStats = el('div', 'detail-stats');
      pkgStats.appendChild(el('span', 'detail-stat', item.package_count + ' published packages'));
      pkgStats.appendChild(el('span', 'detail-stat', item.host));
      scroll.appendChild(pkgStats);

      var pkgActions = el('div', 'detail-actions');
      pkgActions.appendChild(linkEl(item.url, 'action-btn is-primary', 'Open ' + item.host));
      scroll.appendChild(pkgActions);

      if (item.packages && item.packages.length > 0) {
        var pkgSec = el('div', 'detail-section');
        pkgSec.appendChild(el('h3', 'detail-section-title', 'Published Packages'));
        var pkgGrid = el('div', 'pkg-link-grid');
        item.packages.forEach(function (pkg) {
          pkgGrid.appendChild(linkEl(pkg.url, 'pkg-link-chip', pkg.name));
        });
        pkgSec.appendChild(pkgGrid);
        scroll.appendChild(pkgSec);
      }
    } else if (deckKey === 'links') {
      if (item.description) {
        scroll.appendChild(el('p', 'modal-desc', item.description));
      }
      if (item.meta && item.meta.length > 0) {
        var linkStats = el('div', 'detail-stats');
        item.meta.forEach(function (m) {
          linkStats.appendChild(el('span', 'detail-stat', m));
        });
        scroll.appendChild(linkStats);
      }
      var linkActions = el('div', 'detail-actions');
      linkActions.appendChild(linkEl(item.url, 'action-btn is-primary', 'Open ' + item.title));
      scroll.appendChild(linkActions);
    } else if (deckKey === 'media') {
      if (item.summary) {
        scroll.appendChild(el('p', 'modal-desc', item.summary));
      }

      var mediaStats = el('div', 'detail-stats');
      if (item.episode) {
        mediaStats.appendChild(el('span', 'detail-stat', '#' + item.episode));
      }
      if (item.duration) {
        mediaStats.appendChild(el('span', 'detail-stat', item.duration));
      }
      if (item.read_time_minutes > 0) {
        mediaStats.appendChild(el('span', 'detail-stat', item.read_time_minutes + ' min read'));
      }
      if (item.age) {
        mediaStats.appendChild(el('span', 'detail-stat', item.age));
      }
      if (mediaStats.childElementCount > 0) {
        scroll.appendChild(mediaStats);
      }

      var mediaActions = el('div', 'detail-actions');
      var primaryLabel = 'Open';
      if (item.kind === 'episode') {
        primaryLabel = 'Listen to Episode';
      } else if (item.kind === 'post') {
        primaryLabel = 'Read Article';
      } else if (item.kind === 'podcast-hub') {
        primaryLabel = 'Visit Linux Matters';
      } else if (item.kind === 'blog-hub') {
        primaryLabel = 'Visit Blog';
      } else if (item.kind === 'video') {
        primaryLabel = 'Watch Video';
      }
      mediaActions.appendChild(linkEl(item.url, 'action-btn is-primary', primaryLabel));
      if (item.site_url && item.site_url !== item.url && isSafeURL(item.site_url)) {
        mediaActions.appendChild(linkEl(item.site_url, 'action-btn', 'All ' + (item.category || 'Entries')));
      }
      scroll.appendChild(mediaActions);

      if (item.subscribe_links && item.subscribe_links.length > 0) {
        var subSec = el('div', 'detail-section');
        subSec.appendChild(el('h3', 'detail-section-title', 'Subscribe'));
        subSec.appendChild(buildSubscribeRow(item.subscribe_links));
        scroll.appendChild(subSec);
      }

      if (item.entries && item.entries.length > 0) {
        var entrySec = el('div', 'detail-section');
        entrySec.appendChild(el('h3', 'detail-section-title', item.kind === 'podcast-hub' ? 'Recent Episodes' : 'Recent Posts'));
        var ul = el('ul', 'media-entry-list');
        item.entries.forEach(function (entry) {
          var li = el('li', 'media-entry-item');
          var a = linkEl(entry.url, 'media-entry-link');
          var head = el('div', 'media-entry-head');
          var titlePrefix = entry.episode ? ('#' + entry.episode + ' ') : '';
          head.appendChild(el('span', 'media-entry-title', titlePrefix + entry.title));

          var metaBits = [];
          if (entry.duration) {
            metaBits.push(entry.duration);
          }
          if (entry.read_time_minutes > 0) {
            metaBits.push(entry.read_time_minutes + ' min read');
          }
          if (entry.date_label) {
            metaBits.push(entry.date_label);
          }
          if (metaBits.length > 0) {
            head.appendChild(el('span', 'media-entry-meta', metaBits.join(' · ')));
          }
          a.appendChild(head);

          var summaryText = entry.subtitle || entry.summary;
          if (summaryText) {
            a.appendChild(el('p', 'media-entry-summary', summaryText));
          }
          li.appendChild(a);
          ul.appendChild(li);
        });
        entrySec.appendChild(ul);
        scroll.appendChild(entrySec);
      }
    }

    return scroll;
  }

  function openCardModal(deckKey, item, triggerEl) {
    if (!els.modal || !els.modalFlipper || !els.modalFront || !els.modalBack) {
      return;
    }

    state.lastFocusedEl = triggerEl || document.activeElement;
    state.modalOpen = true;
    stopAnimation();

    els.modalFront.textContent = '';
    var frontArt = el('div', 'modal-art');
    mountArtwork(frontArt, item);
    els.modalFront.appendChild(frontArt);
    els.modalFront.appendChild(el('p', 'modal-title', item.name || item.title || item.host || ''));

    els.modalBack.textContent = '';
    els.modalBack.appendChild(buildModalBack(deckKey, item));

    els.modalFlipper.classList.remove('is-flipped');
    els.modal.hidden = false;
    document.body.classList.add('modal-open');

    if (reducedMotionMQ.matches) {
      els.modalFlipper.classList.add('is-flipped');
    } else {
      window.requestAnimationFrame(function () {
        window.requestAnimationFrame(function () {
          els.modalFlipper.classList.add('is-flipped');
        });
      });
    }

    var closeBtn = els.modalBack.querySelector('.modal-close');
    if (closeBtn) {
      closeBtn.focus();
    }
  }

  function closeCardModal() {
    if (!state.modalOpen || !els.modal) {
      return;
    }
    state.modalOpen = false;
    els.modalFlipper.classList.remove('is-flipped');
    els.modal.hidden = true;
    document.body.classList.remove('modal-open');

    if (state.lastFocusedEl && typeof state.lastFocusedEl.focus === 'function') {
      state.lastFocusedEl.focus();
    }
    state.lastFocusedEl = null;
    startAnimation();
  }

  function trapModalFocus(ev) {
    if (!state.modalOpen || ev.key !== 'Tab') {
      return;
    }
    var focusable = els.modalBack.querySelectorAll(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    );
    if (focusable.length === 0) {
      return;
    }
    var first = focusable[0];
    var last = focusable[focusable.length - 1];
    if (ev.shiftKey && document.activeElement === first) {
      ev.preventDefault();
      last.focus();
    } else if (!ev.shiftKey && document.activeElement === last) {
      ev.preventDefault();
      first.focus();
    }
  }

  function bindEvents() {
    if (els.btnViewFloating) {
      els.btnViewFloating.addEventListener('click', function () { setViewMode('floating'); });
    }
    if (els.btnViewGrid) {
      els.btnViewGrid.addEventListener('click', function () { setViewMode('grid'); });
    }
    if (els.btnDriftToggle) {
      els.btnDriftToggle.addEventListener('click', toggleDrift);
    }
    if (els.btnThemeToggle) {
      els.btnThemeToggle.addEventListener('click', toggleTheme);
    }
    if (els.btnSortStars) {
      els.btnSortStars.addEventListener('click', function () { setSortMode('stars'); });
    }
    if (els.btnSortRelease) {
      els.btnSortRelease.addEventListener('click', function () { setSortMode('release'); });
    }

    if (els.stage) {
      els.stage.addEventListener('pointermove', function (ev) {
        if (state.viewMode !== 'floating' || state.driftPaused || reducedMotionMQ.matches) {
          return;
        }
        var rect = els.stage.getBoundingClientRect();
        var px = ev.clientX - rect.left;
        var py = ev.clientY - rect.top;
        if (state.pointer.active) {
          state.pointer.vx = Math.max(-45, Math.min(45, px - state.pointer.lastX));
          state.pointer.vy = Math.max(-45, Math.min(45, py - state.pointer.lastY));
        }
        state.pointer.x = px;
        state.pointer.y = py;
        state.pointer.lastX = px;
        state.pointer.lastY = py;
        state.pointer.active = true;
      });
      els.stage.addEventListener('pointerleave', function () {
        state.pointer.active = false;
        state.pointer.vx = 0;
        state.pointer.vy = 0;
      });
    }

    if (els.modal) {
      els.modal.addEventListener('click', function (ev) {
        if (ev.target === els.modal) {
          closeCardModal();
        }
      });
    }

    document.addEventListener('keydown', function (ev) {
      if (state.modalOpen && ev.key === 'Escape') {
        ev.preventDefault();
        closeCardModal();
        return;
      }
      trapModalFocus(ev);
    });

    window.addEventListener('resize', function () {
      if (state.viewMode !== 'floating') {
        return;
      }
      window.clearTimeout(state.resizeTimer);
      state.resizeTimer = window.setTimeout(function () {
        computeConstellation(state.cards, els.stage.clientWidth, null);
      }, 120);
    });

    document.addEventListener('visibilitychange', function () {
      if (document.visibilityState === 'hidden') {
        stopAnimation();
      } else {
        startAnimation();
      }
    });

    if (reducedMotionMQ && typeof reducedMotionMQ.addEventListener === 'function') {
      reducedMotionMQ.addEventListener('change', function (ev) {
        if (ev.matches) {
          stopAnimation();
          state.viewMode = 'grid';
          renderDeckControls();
          renderStage(null);
        }
      });
    }
  }

  function bootstrap(data) {
    state.data = data;
    state.activeDeck = data.default_deck || 'projects';
    state.sortMode = data.default_sort || 'stars';

    syncThemeUI();
    renderHeaderStats();
    renderDeckControls();
    renderStage(null);
    bindEvents();
  }

  function loadData() {
    var inlineNode = document.getElementById('site-data');
    if (inlineNode && inlineNode.textContent && inlineNode.textContent.trim().length > 0) {
      try {
        bootstrap(JSON.parse(inlineNode.textContent));
        return;
      } catch (_) {}
    }

    fetch('./data.json')
      .then(function (resp) {
        if (!resp.ok) {
          throw new Error('HTTP ' + resp.status);
        }
        return resp.json();
      })
      .then(bootstrap)
      .catch(function () {
        if (els.stage) {
          els.stage.textContent = 'Unable to load portfolio data.';
        }
      });
  }

  loadData();
})();
