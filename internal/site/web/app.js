import * as PIXI from './vendor/pixi.min.mjs';

const DECKS = [
  { key: 'projects', label: 'Projects', accent: 'mauve' },
  { key: 'links', label: 'Links', accent: 'blue' },
  { key: 'media', label: 'Media', accent: 'teal' },
];

// Portrait 5:7 Top Trumps playing-card geometry (logical px, baked at 2x).
const CARD_W = 224;
const CARD_H = 314;
const CARD_RADIUS = 14;
const TEX_SCALE = 2;

const PALETTES = {
  mocha: {
    name: 'mocha',
    crust: '#11111b',
    crustHex: 0x11111b,
    mantle: '#181825',
    base: '#1e1e2e',
    surface0: '#313244',
    surface1: '#45475a',
    surface2: '#585b70',
    text: '#cdd6f4',
    subtext: '#bac2de',
    muted: '#a6adc8',
    mauve: '#cba6f7',
    lavender: '#b4befe',
    blue: '#89b4fa',
    sapphire: '#74c7ec',
    teal: '#94e2d5',
    green: '#a6e3a1',
    yellow: '#f9e2af',
    peach: '#fab387',
    red: '#f38ba8',
    pink: '#f5c2e7',
    linkText: '#89b4fa',
    freshText: '#a6e3a1',
    freshBg: 'rgba(166, 227, 161, 0.14)',
    rowEven: 'rgba(24, 24, 37, 0.82)',
    rowOdd: 'rgba(49, 50, 68, 0.44)',
    innerFrame: 'rgba(205, 214, 244, 0.08)',
    badgeBg: '#181825',
    shadowAlpha: 0.42,
    foilAlpha: 0.25,
    swirlA: '#11111b',
    swirlB: '#181828',
    swirlC: '#231f38',
    swirlAccent: '#302b4d',
  },
  latte: {
    name: 'latte',
    crust: '#dce0e8',
    crustHex: 0xdce0e8,
    mantle: '#e6e9ef',
    base: '#eff1f5',
    surface0: '#ccd0da',
    surface1: '#bcc0cc',
    surface2: '#acb0be',
    text: '#4c4f69',
    subtext: '#5c5f77',
    muted: '#5c5f77',
    mauve: '#8839ef',
    lavender: '#7287fd',
    blue: '#1e66f5',
    sapphire: '#209fb5',
    teal: '#179299',
    green: '#40a02b',
    yellow: '#df8e1d',
    peach: '#fe640b',
    red: '#d20f39',
    pink: '#ea76cb',
    linkText: '#1752c8',
    freshText: '#2a6f1d',
    freshBg: 'rgba(64, 160, 43, 0.14)',
    rowEven: 'rgba(230, 233, 239, 0.92)',
    rowOdd: 'rgba(204, 208, 218, 0.48)',
    innerFrame: 'rgba(76, 79, 105, 0.09)',
    badgeBg: '#e6e9ef',
    shadowAlpha: 0.22,
    foilAlpha: 0.2,
    swirlA: '#dce0e8',
    swirlB: '#d2d7e5',
    swirlC: '#c9d2e8',
    swirlAccent: '#d9ccf0',
  },
};

const HOST_DOT_ACCENTS = [
  'red',
  'blue',
  'sapphire',
  'teal',
  'mauve',
  'peach',
  'green',
  'lavender',
];

const BALATRO_FRAG = `
precision highp float;

in vec2 vTextureCoord;
out vec4 finalColor;

uniform sampler2D uTexture;
uniform vec4 uInputSize;
uniform float uTime;
uniform vec2 uResolution;
uniform vec3 uColorA;
uniform vec3 uColorB;
uniform vec3 uColorC;
uniform vec3 uAccent;

void main(void) {
  vec4 src = texture(uTexture, vTextureCoord);
  vec2 res = max(uResolution, vec2(1.0));
  vec2 fragPx = vTextureCoord * uInputSize.xy;
  vec2 uv = (fragPx - 0.5 * res) / min(res.x, res.y);
  float dist = length(uv);
  float angle = atan(uv.y, uv.x);
  float twist = angle + 0.95 / (dist + 0.55) + uTime * 0.12;
  vec2 p = vec2(cos(twist), sin(twist)) * dist * 1.45;

  for (int i = 0; i < 4; i++) {
    float fi = float(i) * 0.65;
    p += 0.28 * sin(p.yx * 2.15 + vec2(uTime * 0.35 + fi, -uTime * 0.28 + fi));
  }

  float band1 = 0.5 + 0.5 * sin(p.x * 2.6 + p.y * 1.8 + uTime * 0.18);
  float band2 = 0.5 + 0.5 * cos(p.y * 3.1 - p.x * 1.4 - uTime * 0.14);
  float ribbon = smoothstep(0.68, 0.98, band1 * band2);

  vec3 col = mix(uColorA, uColorB, smoothstep(0.15, 0.85, band1));
  col = mix(col, uColorC, smoothstep(0.25, 0.92, band2) * 0.62);
  col = mix(col, uAccent, ribbon * 0.45);

  float vignette = smoothstep(1.38, 0.22, dist);
  col = mix(uColorA, col, 0.52 + 0.48 * vignette);
  finalColor = vec4(col, src.a);
}
`;

const FONT_SANS = "'Work Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif";
const FONT_MONO = "'Fira Code', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace";

const colorSchemeMQ = window.matchMedia ? window.matchMedia('(prefers-color-scheme: light)') : { matches: false };
const reducedMotionMQ = window.matchMedia ? window.matchMedia('(prefers-reduced-motion: reduce)') : { matches: false };

const els = {
  pixiRoot: document.getElementById('pixi-root'),
  scrollSpacer: document.getElementById('scroll-spacer'),
  a11yDecks: document.getElementById('a11y-decks'),
  a11yCards: document.getElementById('a11y-cards'),
  liveStatus: document.getElementById('live-status'),
  modal: document.getElementById('card-modal'),
  modalCard: document.getElementById('modal-card'),
  themeButtons: Array.from(document.querySelectorAll('.theme-opt[data-theme-mode]')),
};

let svgMountSeq = 0;

const state = {
  data: null,
  themeMode: document.documentElement.dataset.themeMode || 'system',
  theme: document.documentElement.dataset.theme === 'latte' ? 'latte' : 'mocha',
  activeDeck: 'projects',
  modalOpen: false,
  lastFocusedEl: null,
  flippingCard: null,

  // PixiJS scene objects
  app: null,
  bgSprite: null,
  bgFilter: null,
  world: null,
  decksLayer: null,
  cardsLayer: null,
  particlesLayer: null,
  shadowTex: null,
  deckShadowTex: null,
  foilTex: null,
  sparkleTex: null,

  // Live records
  deckRecords: [],
  cardRecords: [],
  exitingCards: [],
  particles: [],
  artworkCache: new Map(),

  // Pointer & layout
  pointer: { x: 0, y: 0, active: false },
  hoveredDeckKey: null,
  hoveredCardId: null,
  focusedCardId: null,
  layout: {
    originY: 96,
    viewportW: window.innerWidth,
    viewportH: window.innerHeight,
    deckW: 176,
    deckH: 94,
    cardScale: 1,
    cols: 4,
  },
  shaderTime: 0,
  swirlBoost: 0,
  resizeTimer: 0,
};

function palette() {
  return PALETTES[state.theme] || PALETTES.mocha;
}

function accentColor(p, name) {
  if (!name) {
    return p.mauve;
  }
  const key = String(name).toLowerCase().trim();
  return p[key] || p.mauve;
}

function hexToRgb01(hex) {
  const clean = String(hex || '#11111b').replace('#', '');
  const n = parseInt(clean, 16);
  return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}

function hexToNumber(hex) {
  return parseInt(String(hex || '#cba6f7').replace('#', ''), 16);
}

function seeded(n) {
  const x = Math.sin(n * 12.9898 + 78.233) * 43758.5453;
  return (x - Math.floor(x)) * 2 - 1;
}

function isReducedMotion() {
  return Boolean(reducedMotionMQ && reducedMotionMQ.matches);
}

function announce(msg) {
  if (!els.liveStatus) {
    return;
  }
  els.liveStatus.textContent = '';
  window.setTimeout(() => {
    els.liveStatus.textContent = msg;
  }, 20);
}

function isSafeURL(raw) {
  if (typeof raw !== 'string') {
    return false;
  }
  const trimmed = raw.trim().toLowerCase();
  return trimmed.startsWith('https://') || trimmed.startsWith('http://') || trimmed.startsWith('mailto:');
}

function isSafeImageURI(raw) {
  if (typeof raw !== 'string') {
    return false;
  }
  const trimmed = raw.trim().toLowerCase();
  if (trimmed.startsWith('https://') || trimmed.startsWith('http://')) {
    return true;
  }
  return trimmed.startsWith('data:image/') && trimmed.includes(';base64,');
}

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) {
    node.className = className;
  }
  if (text !== undefined && text !== null) {
    node.textContent = String(text);
  }
  return node;
}

function linkEl(href, className, text) {
  const a = el('a', className, text);
  if (isSafeURL(href)) {
    a.href = href;
    a.target = '_blank';
    a.rel = 'noopener noreferrer';
  } else {
    a.href = '#';
  }
  return a;
}

function deckItems(deckKey) {
  if (!state.data) {
    return [];
  }
  if (deckKey === 'projects') {
    const list = (state.data.projects || []).slice();
    const sortMode = String(state.data.default_sort || 'stars').toLowerCase();
    if (sortMode === 'release') {
      list.sort((a, b) => {
        if ((b.published_unix || 0) !== (a.published_unix || 0)) {
          return (b.published_unix || 0) - (a.published_unix || 0);
        }
        if ((b.stars || 0) !== (a.stars || 0)) {
          return (b.stars || 0) - (a.stars || 0);
        }
        return String(a.name || '').localeCompare(String(b.name || ''));
      });
    } else {
      list.sort((a, b) => {
        if ((b.stars || 0) !== (a.stars || 0)) {
          return (b.stars || 0) - (a.stars || 0);
        }
        if ((b.published_unix || 0) !== (a.published_unix || 0)) {
          return (b.published_unix || 0) - (a.published_unix || 0);
        }
        return String(a.name || '').localeCompare(String(b.name || ''));
      });
    }
    return list;
  }
  if (deckKey === 'links') {
    return state.data.links || [];
  }
  if (deckKey === 'media') {
    return state.data.media || [];
  }
  return [];
}

// ---------------------------------------------------------------------------
// Theme management (3-way System / Light / Dark)
// ---------------------------------------------------------------------------

function resolveThemeFromMode(mode) {
  if (mode === 'light') {
    return 'latte';
  }
  if (mode === 'dark') {
    return 'mocha';
  }
  return colorSchemeMQ.matches ? 'latte' : 'mocha';
}

function syncThemeControls() {
  for (const btn of els.themeButtons) {
    const isCurrent = btn.dataset.themeMode === state.themeMode;
    btn.setAttribute('aria-pressed', String(isCurrent));
  }
}

function applyThemeMode(mode, shouldAnnounce) {
  if (mode !== 'system' && mode !== 'light' && mode !== 'dark') {
    mode = 'system';
  }
  state.themeMode = mode;
  const nextTheme = resolveThemeFromMode(mode);
  const changed = nextTheme !== state.theme;
  state.theme = nextTheme;

  document.documentElement.dataset.themeMode = mode;
  document.documentElement.dataset.theme = nextTheme;
  try {
    localStorage.setItem('wimpys-world-theme', mode);
  } catch (_) {
    // Ignore storage quota or privacy-mode errors.
  }

  syncThemeControls();
  if (changed && state.app) {
    refreshPixiForTheme();
  }
  if (shouldAnnounce) {
    const label = mode === 'system'
      ? `System theme (${nextTheme === 'latte' ? 'Catppuccin Latte' : 'Catppuccin Mocha'})`
      : (nextTheme === 'latte' ? 'Catppuccin Latte light theme' : 'Catppuccin Mocha dark theme');
    announce(`Switched to ${label}.`);
  }
}

// ---------------------------------------------------------------------------
// Canvas & SVG helpers (high-DPI 2x textures with crisp typography & logos)
// ---------------------------------------------------------------------------

function createCanvas(w, h) {
  const canvas = document.createElement('canvas');
  canvas.width = Math.round(w * TEX_SCALE);
  canvas.height = Math.round(h * TEX_SCALE);
  const ctx = canvas.getContext('2d');
  ctx.scale(TEX_SCALE, TEX_SCALE);
  return { canvas, ctx };
}

function canvasToTexture(canvas) {
  const source = new PIXI.CanvasSource({
    resource: canvas,
    resolution: TEX_SCALE,
    scaleMode: 'linear',
    autoGenerateMipmaps: true,
  });
  return new PIXI.Texture({ source });
}

function roundedRectPath(ctx, x, y, w, h, r) {
  const rr = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + rr, y);
  ctx.arcTo(x + w, y, x + w, y + h, rr);
  ctx.arcTo(x + w, y + h, x, y + h, rr);
  ctx.arcTo(x, y + h, x, y, rr);
  ctx.arcTo(x, y, x + w, y, rr);
  ctx.closePath();
}

function truncateText(ctx, text, maxW) {
  const str = String(text || '').trim();
  if (!str) {
    return '';
  }
  if (ctx.measureText(str).width <= maxW) {
    return str;
  }
  const ellipsis = '…';
  let lo = 0;
  let hi = str.length;
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    if (ctx.measureText(str.slice(0, mid).trimEnd() + ellipsis).width <= maxW) {
      lo = mid;
    } else {
      hi = mid - 1;
    }
  }
  return str.slice(0, Math.max(1, lo)).trimEnd() + ellipsis;
}

function wrapLines(ctx, text, maxW, maxLines) {
  const words = String(text || '').trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return [];
  }
  const lines = [];
  let current = '';
  for (let i = 0; i < words.length; i += 1) {
    const candidate = current ? `${current} ${words[i]}` : words[i];
    if (ctx.measureText(candidate).width <= maxW || !current) {
      current = candidate;
    } else {
      lines.push(current);
      if (lines.length === maxLines - 1) {
        const rest = words.slice(i).join(' ');
        lines.push(truncateText(ctx, rest, maxW));
        return lines;
      }
      current = words[i];
    }
  }
  if (current) {
    lines.push(truncateText(ctx, current, maxW));
  }
  return lines.slice(0, maxLines);
}

// Prepares standalone inline SVG markup for Image rasterization onto 2D canvas:
// ensures xmlns, resolves CSS var(--token) and currentColor, and sets raster size.
function prepareSVGForImage(rawSVG, accentHex, p) {
  if (!rawSVG || typeof rawSVG !== 'string') {
    return '';
  }
  let svg = rawSVG.trim();
  if (!svg.toLowerCase().startsWith('<svg')) {
    return '';
  }

  if (!/^<svg\b[^>]*\bxmlns=/i.test(svg)) {
    svg = svg.replace(/^<svg\b/i, '<svg xmlns="http://www.w3.org/2000/svg"');
  }

  const varMap = {
    '--crust': p.crust,
    '--mantle': p.mantle,
    '--base': p.base,
    '--surface0': p.surface0,
    '--surface1': p.surface1,
    '--surface2': p.surface2,
    '--text': p.text,
    '--subtext': p.subtext,
    '--muted-text': p.muted,
    '--mauve': p.mauve,
    '--lavender': p.lavender,
    '--blue': p.blue,
    '--sapphire': p.sapphire,
    '--teal': p.teal,
    '--green': p.green,
    '--yellow': p.yellow,
    '--peach': p.peach,
    '--red': p.red,
    '--pink': p.pink,
    '--accent-text': accentHex || p.mauve,
    '--link-text': p.linkText,
    '--fresh-text': p.freshText,
  };

  svg = svg.replace(/var\((--[a-z0-9-]+)\)/gi, (_, name) => varMap[name.toLowerCase()] || p.text);
  svg = svg.replace(/currentColor/gi, accentHex || p.mauve);

  const openTagMatch = svg.match(/^<svg\b[^>]*>/i);
  if (openTagMatch) {
    let openTag = openTagMatch[0];
    if (!/\bwidth=/i.test(openTag)) {
      openTag = openTag.replace(/^<svg\b/i, '<svg width="128"');
    }
    if (!/\bheight=/i.test(openTag)) {
      openTag = openTag.replace(/^<svg\b/i, '<svg height="128"');
    }
    svg = openTag + svg.slice(openTagMatch[0].length);
  }
  return svg;
}

function loadSVGAsImage(rawSVG, accentHex, p) {
  return new Promise((resolve) => {
    const svgText = prepareSVGForImage(rawSVG, accentHex, p);
    if (!svgText) {
      resolve(null);
      return;
    }
    const blob = new Blob([svgText], { type: 'image/svg+xml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const img = new Image();
    img.decoding = 'async';
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      resolve(null);
    };
    img.src = url;
  });
}

function loadBitmapImage(src) {
  return new Promise((resolve) => {
    if (!isSafeImageURI(src)) {
      resolve(null);
      return;
    }
    const img = new Image();
    if (src.startsWith('http://') || src.startsWith('https://')) {
      img.crossOrigin = 'anonymous';
    }
    img.decoding = 'async';
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

async function resolveCardArtwork(deckKey, item, p) {
  const accent = resolveCardAccent(deckKey, item, p);
  const cacheKey = `${p.name}|${deckKey}|${item.id || item.name || item.title}`;
  if (state.artworkCache.has(cacheKey)) {
    return state.artworkCache.get(cacheKey);
  }

  let result = null;
  if (item.cover_url && isSafeImageURI(item.cover_url)) {
    const cover = await loadBitmapImage(item.cover_url);
    if (cover) {
      result = { img: cover, isCover: true };
    }
  }
  if (!result && item.logo_svg) {
    const svgImg = await loadSVGAsImage(item.logo_svg, accent, p);
    if (svgImg) {
      result = { img: svgImg, isCover: false };
    }
  }
  if (!result && item.logo_data_uri && isSafeImageURI(item.logo_data_uri)) {
    const bmp = await loadBitmapImage(item.logo_data_uri);
    if (bmp) {
      result = { img: bmp, isCover: false };
    }
  }
  if (!result && item.icon_svg) {
    const iconImg = await loadSVGAsImage(item.icon_svg, accent, p);
    if (iconImg) {
      result = { img: iconImg, isCover: false, isIcon: true };
    }
  }

  state.artworkCache.set(cacheKey, result);
  return result;
}

// ---------------------------------------------------------------------------
// Shared procedural textures (soft drop shadows, Balatro foil sheen, sparkles)
// ---------------------------------------------------------------------------

function buildSoftShadowTexture(w, h, r, pad) {
  const totalW = w + pad * 2;
  const totalH = h + pad * 2;
  const { canvas, ctx } = createCanvas(totalW, totalH);
  ctx.save();
  ctx.shadowColor = 'rgba(0, 0, 0, 0.85)';
  ctx.shadowBlur = pad * 0.72;
  ctx.shadowOffsetX = 0;
  ctx.shadowOffsetY = 4;
  ctx.fillStyle = 'rgba(0, 0, 0, 0.65)';
  roundedRectPath(ctx, pad, pad, w, h, r);
  ctx.fill();
  ctx.restore();
  return canvasToTexture(canvas);
}

function buildFoilTexture(w, h, r) {
  const { canvas, ctx } = createCanvas(w, h);
  ctx.save();
  roundedRectPath(ctx, 0, 0, w, h, r);
  ctx.clip();

  const grad = ctx.createLinearGradient(0, 0, w, h);
  grad.addColorStop(0.0, 'rgba(245, 194, 231, 0.0)');
  grad.addColorStop(0.18, 'rgba(203, 166, 247, 0.55)');
  grad.addColorStop(0.36, 'rgba(137, 180, 250, 0.72)');
  grad.addColorStop(0.5, 'rgba(255, 255, 255, 0.92)');
  grad.addColorStop(0.64, 'rgba(148, 226, 213, 0.72)');
  grad.addColorStop(0.82, 'rgba(249, 226, 175, 0.48)');
  grad.addColorStop(1.0, 'rgba(245, 194, 231, 0.0)');
  ctx.fillStyle = grad;
  ctx.fillRect(0, 0, w, h);

  // Fine holographic micro-lines
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.16)';
  ctx.lineWidth = 1;
  for (let d = -h; d < w + h; d += 12) {
    ctx.beginPath();
    ctx.moveTo(d, 0);
    ctx.lineTo(d - h * 0.4, h);
    ctx.stroke();
  }
  ctx.restore();
  return canvasToTexture(canvas);
}

function buildSparkleTexture() {
  const s = 20;
  const { canvas, ctx } = createCanvas(s, s);
  const c = s / 2;
  ctx.fillStyle = '#ffffff';
  ctx.beginPath();
  ctx.moveTo(c, 1);
  ctx.quadraticCurveTo(c + 1.5, c - 1.5, s - 1, c);
  ctx.quadraticCurveTo(c + 1.5, c + 1.5, c, s - 1);
  ctx.quadraticCurveTo(c - 1.5, c + 1.5, 1, c);
  ctx.quadraticCurveTo(c - 1.5, c - 1.5, c, 1);
  ctx.closePath();
  ctx.fill();
  return canvasToTexture(canvas);
}

// ---------------------------------------------------------------------------
// Top Trumps card face painter (224 x 314 @ 2x)
// ---------------------------------------------------------------------------

function resolveCardAccent(deckKey, item, p) {
  if (deckKey === 'projects') {
    return item.language_color || p.mauve;
  }
  return accentColor(p, item.accent);
}

function extractCardTopTrumpsModel(deckKey, item, p) {
  const totalHosts = (state.data && state.data.packages && state.data.packages.length) || 8;
  const accent = resolveCardAccent(deckKey, item, p);

  if (deckKey === 'projects') {
    const hostCount = (item.hosts || []).length;
    const rows = [
      {
        label: 'STARS',
        value: `★ ${item.stars_formatted || item.stars || 0}`,
        valueColor: p.text,
      },
      {
        label: 'RELEASE',
        value: item.latest_tag || '—',
        valueColor: item.fresh ? p.freshText : p.text,
      },
      {
        label: 'UPDATED',
        value: item.release_age || '—',
        valueColor: p.subtext,
      },
      {
        label: 'PACKAGES',
        value: hostCount > 0 ? `${hostCount}/${totalHosts} hosts` : 'Source',
        valueColor: hostCount > 0 ? p.text : p.subtext,
        hostDots: hostCount,
      },
    ];
    return {
      accent,
      topLeft: item.language || item.owner || 'GitHub',
      dotColor: item.language_color || p.mauve,
      topBadge: item.latest_tag || '',
      topBadgeFresh: Boolean(item.fresh),
      title: item.name || item.full_name || 'Project',
      desc: item.description || item.full_name || '',
      rows,
    };
  }

  if (deckKey === 'links') {
    const rows = [
      { label: 'CATEGORY', value: item.category || 'Link', valueColor: p.text },
      { label: 'HOST', value: item.host_label || 'External', valueColor: p.subtext },
    ];

    const s = (state.data && state.data.stats) || {};
    const icon = String(item.icon || '').toLowerCase();
    const isDiscord = icon === 'discord' || item.id === 'discord';
    const isSponsors = icon === 'heart' || item.id === 'github-sponsors' || String(item.url || '').includes('github.com/sponsors/');

    if (isDiscord && (s.discord_online > 0 || s.discord_members > 0)) {
      rows.push({
        label: 'ONLINE',
        value: s.discord_online > 0 ? s.discord_online.toLocaleString('en-GB') : '—',
        valueColor: p.freshText,
      });
      rows.push({
        label: 'MEMBERS',
        value: s.discord_members > 0 ? s.discord_members.toLocaleString('en-GB') : '—',
        valueColor: p.text,
      });
    } else if (isSponsors && (s.active_sponsors > 0 || s.past_sponsors > 0)) {
      rows.push({
        label: 'ACTIVE',
        value: String(s.active_sponsors || 0),
        valueColor: s.active_sponsors > 0 ? p.freshText : p.subtext,
      });
      rows.push({
        label: 'PAST SPONSORS',
        value: String(s.past_sponsors || 0),
        valueColor: p.text,
      });
    } else {
      rows.push({ label: 'DECK', value: 'Links', valueColor: p.subtext });
      rows.push({ label: 'OPEN', value: 'Details ↗', valueColor: p.linkText });
    }

    return {
      accent,
      topLeft: item.category || 'Link',
      dotColor: accent,
      topBadge: item.badge || '',
      topBadgeFresh: isDiscord && Boolean(item.badge),
      title: item.title || 'Link',
      desc: item.description || item.host_label || '',
      rows,
    };
  }

  // Media deck
  const rows = [];
  if (item.kind === 'episode') {
    rows.push({ label: 'EPISODE', value: item.episode ? `#${item.episode}` : '—', valueColor: p.text });
    rows.push({ label: 'DURATION', value: item.duration || '—', valueColor: p.text });
    rows.push({ label: 'PUBLISHED', value: item.age || item.date_label || '—', valueColor: p.subtext });
    rows.push({ label: 'SHOW', value: 'Linux Matters', valueColor: p.subtext });
  } else if (item.kind === 'podcast-hub') {
    const entries = item.entries || [];
    const latest = entries[0];
    rows.push({ label: 'LATEST', value: latest && latest.episode ? `#${latest.episode}` : (latest && latest.age) || '—', valueColor: p.text });
    rows.push({ label: 'EPISODES', value: entries.length > 0 ? `${entries.length} recent` : 'Feed', valueColor: p.text });
    rows.push({ label: 'LISTEN', value: `${(item.subscribe_links || []).length || 1} feeds`, valueColor: p.linkText });
    rows.push({ label: 'SHOW', value: 'Linux Matters', valueColor: p.subtext });
  } else if (item.kind === 'post') {
    rows.push({ label: 'READ TIME', value: item.read_time_minutes > 0 ? `${item.read_time_minutes} min` : 'Article', valueColor: p.text });
    rows.push({ label: 'PUBLISHED', value: item.age || item.date_label || '—', valueColor: p.subtext });
    rows.push({ label: 'SITE', value: 'wimpysworld.com', valueColor: p.subtext });
    rows.push({ label: 'FORMAT', value: 'Blog Post', valueColor: p.subtext });
  } else if (item.kind === 'blog-hub') {
    const entries = item.entries || [];
    const latest = entries[0];
    rows.push({ label: 'POSTS', value: entries.length > 0 ? `${entries.length} recent` : 'Feed', valueColor: p.text });
    rows.push({ label: 'LATEST', value: (latest && (latest.age || latest.date_label)) || '—', valueColor: p.text });
    rows.push({ label: 'SITE', value: 'wimpysworld.com', valueColor: p.subtext });
    rows.push({ label: 'FEED', value: 'RSS', valueColor: p.linkText });
  } else {
    rows.push({ label: 'PUBLISHED', value: item.age || item.date_label || '—', valueColor: p.text });
    rows.push({ label: 'DATE', value: item.date_label || '—', valueColor: p.subtext });
    rows.push({ label: 'SITE', value: 'YouTube', valueColor: p.subtext });
    rows.push({ label: 'FORMAT', value: 'Video', valueColor: p.subtext });
  }

  return {
    accent,
    topLeft: item.category || 'Media',
    dotColor: accent,
    topBadge: item.badge || '',
    topBadgeFresh: false,
    title: item.title || 'Media',
    desc: item.summary || item.subtitle || '',
    rows,
  };
}

function drawCardFaceToCanvas(canvas, deckKey, item, p, artwork) {
  const ctx = canvas.getContext('2d');
  ctx.save();
  ctx.setTransform(TEX_SCALE, 0, 0, TEX_SCALE, 0, 0);
  ctx.clearRect(0, 0, CARD_W, CARD_H);

  const model = extractCardTopTrumpsModel(deckKey, item, p);
  const padX = 14;

  // Card body clipped to rounded playing-card silhouette
  ctx.save();
  roundedRectPath(ctx, 0.75, 0.75, CARD_W - 1.5, CARD_H - 1.5, CARD_RADIUS);
  ctx.fillStyle = p.base;
  ctx.fill();
  ctx.clip();

  // Top Catppuccin aurora accent band (4px)
  const topGrad = ctx.createLinearGradient(0, 0, CARD_W, 0);
  topGrad.addColorStop(0, model.accent);
  topGrad.addColorStop(0.6, p.blue);
  topGrad.addColorStop(1, p.teal);
  ctx.fillStyle = topGrad;
  ctx.fillRect(0, 0, CARD_W, 4);

  // Subtle vertical sheen tint from top
  const bgTint = ctx.createLinearGradient(0, 4, 0, 124);
  bgTint.addColorStop(0, p.name === 'latte' ? 'rgba(255,255,255,0.45)' : 'rgba(255,255,255,0.03)');
  bgTint.addColorStop(1, 'rgba(255,255,255,0)');
  ctx.fillStyle = bgTint;
  ctx.fillRect(0, 4, CARD_W, 120);

  // Subtle Top Trumps inner hairline frame
  roundedRectPath(ctx, 5.5, 8.5, CARD_W - 11, CARD_H - 14, 10);
  ctx.strokeStyle = p.innerFrame;
  ctx.lineWidth = 1;
  ctx.stroke();

  // 1. Top index row (y = 14..32)
  const topY = 23;
  ctx.fillStyle = model.dotColor;
  ctx.beginPath();
  ctx.arc(padX + 4, topY, 4, 0, Math.PI * 2);
  ctx.fill();

  ctx.font = `500 10.5px ${FONT_MONO}`;
  ctx.fillStyle = p.subtext;
  ctx.textBaseline = 'middle';
  ctx.textAlign = 'left';
  const maxLeftW = model.topBadge ? 74 : CARD_W - padX * 2 - 14;
  ctx.fillText(truncateText(ctx, model.topLeft, maxLeftW), padX + 12, topY + 0.5);

  if (model.topBadge) {
    ctx.font = `600 10px ${FONT_MONO}`;
    const badgeStr = truncateText(ctx, model.topBadge, 110);
    const textW = ctx.measureText(badgeStr).width;
    const pillW = textW + 12;
    const pillH = 18;
    const pillX = CARD_W - padX - pillW;
    const pillY = topY - pillH / 2;

    roundedRectPath(ctx, pillX, pillY, pillW, pillH, 9);
    ctx.fillStyle = model.topBadgeFresh ? p.freshBg : p.badgeBg;
    ctx.fill();
    ctx.strokeStyle = model.topBadgeFresh ? p.green : p.surface1;
    ctx.lineWidth = 1;
    ctx.stroke();

    ctx.fillStyle = model.topBadgeFresh ? p.freshText : p.text;
    ctx.textAlign = 'center';
    ctx.fillText(badgeStr, pillX + pillW / 2, topY + 0.5);
  }

  // 2. Upper unframed illustration panel (y = 38..108)
  const artCx = CARD_W / 2;
  const artCy = 73;
  const halo = ctx.createRadialGradient(artCx, artCy, 4, artCx, artCy, 38);
  halo.addColorStop(0, p.name === 'latte' ? 'rgba(136, 57, 239, 0.10)' : 'rgba(203, 166, 247, 0.13)');
  halo.addColorStop(1, 'rgba(0, 0, 0, 0)');
  ctx.fillStyle = halo;
  ctx.beginPath();
  ctx.arc(artCx, artCy, 38, 0, Math.PI * 2);
  ctx.fill();

  if (artwork && artwork.img) {
    const img = artwork.img;
    const box = artwork.isCover ? 62 : (artwork.isIcon ? 44 : 58);
    const iw = img.naturalWidth || img.width || box;
    const ih = img.naturalHeight || img.height || box;
    const ratio = Math.min(box / iw, box / ih);
    const dw = iw * ratio;
    const dh = ih * ratio;
    const dx = artCx - dw / 2;
    const dy = artCy - dh / 2;

    if (artwork.isCover) {
      ctx.save();
      roundedRectPath(ctx, artCx - box / 2, artCy - box / 2, box, box, 12);
      ctx.clip();
      ctx.drawImage(img, artCx - box / 2, artCy - box / 2, box, box);
      ctx.restore();
    } else {
      ctx.drawImage(img, dx, dy, dw, dh);
    }
  } else {
    // Crisp fallback initial badge while artwork loads
    roundedRectPath(ctx, artCx - 24, artCy - 24, 48, 48, 12);
    ctx.fillStyle = p.surface0;
    ctx.fill();
    ctx.strokeStyle = model.accent;
    ctx.lineWidth = 1.5;
    ctx.stroke();
    ctx.font = `700 18px ${FONT_SANS}`;
    ctx.fillStyle = p.text;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    const initial = (model.title || 'W').slice(0, 2).toUpperCase();
    ctx.fillText(initial, artCx, artCy + 1);
  }

  // 3. Bold Work Sans title + crisp 2-line description (y = 114..180)
  const textW = CARD_W - padX * 2;
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  ctx.font = `700 15.5px ${FONT_SANS}`;
  ctx.fillStyle = p.text;

  const titleLines = wrapLines(ctx, model.title, textW, 2);
  let cursorY = 114;
  if (titleLines.length === 1) {
    ctx.fillText(titleLines[0], padX, cursorY);
    cursorY += 22;
  } else {
    ctx.font = `700 14px ${FONT_SANS}`;
    const tightTitle = wrapLines(ctx, model.title, textW, 2);
    for (const line of tightTitle) {
      ctx.fillText(line, padX, cursorY);
      cursorY += 17;
    }
    cursorY += 3;
  }

  ctx.font = `400 11.2px ${FONT_SANS}`;
  ctx.fillStyle = p.subtext;
  const maxDescLines = titleLines.length > 1 ? 2 : 2;
  const descLines = wrapLines(ctx, model.desc, textW, maxDescLines);
  for (const line of descLines) {
    ctx.fillText(line, padX, cursorY);
    cursorY += 15;
  }

  // 4. Lower Top Trumps stat rows (4 rows, y = 190..302)
  const rowsStartY = 190;
  const rowH = 25;
  const rowGap = 3;
  const rowW = CARD_W - padX * 2;

  for (let r = 0; r < Math.min(4, model.rows.length); r += 1) {
    const row = model.rows[r];
    const ry = rowsStartY + r * (rowH + rowGap);

    roundedRectPath(ctx, padX, ry, rowW, rowH, 6);
    ctx.fillStyle = r % 2 === 0 ? p.rowEven : p.rowOdd;
    ctx.fill();

    // Left uppercase label
    ctx.font = `600 9px ${FONT_MONO}`;
    ctx.fillStyle = p.muted;
    ctx.textAlign = 'left';
    ctx.textBaseline = 'middle';
    ctx.fillText(row.label, padX + 8, ry + rowH / 2 + 0.5);

    // Right tabular value (plus optional tiny coloured host dots on PACKAGES)
    ctx.font = `600 10.5px ${FONT_MONO}`;
    ctx.fillStyle = row.valueColor || p.text;
    ctx.textAlign = 'right';
    const valStr = truncateText(ctx, row.value, rowW - 82);
    ctx.fillText(valStr, padX + rowW - 8, ry + rowH / 2 + 0.5);

    if (row.hostDots && row.hostDots > 0) {
      const valW = ctx.measureText(valStr).width;
      const dotCount = Math.min(8, row.hostDots);
      const dotStep = 6.5;
      const startX = padX + rowW - 8 - valW - 7 - (dotCount - 1) * dotStep;
      if (startX > padX + 64) {
        for (let d = 0; d < dotCount; d += 1) {
          ctx.fillStyle = accentColor(p, HOST_DOT_ACCENTS[d % HOST_DOT_ACCENTS.length]);
          ctx.beginPath();
          ctx.arc(startX + d * dotStep, ry + rowH / 2, 2.3, 0, Math.PI * 2);
          ctx.fill();
        }
      }
    }
  }

  ctx.restore(); // End clip

  // Outer crisp 1.5px border
  roundedRectPath(ctx, 0.75, 0.75, CARD_W - 1.5, CARD_H - 1.5, CARD_RADIUS);
  ctx.strokeStyle = p.surface1;
  ctx.lineWidth = 1.5;
  ctx.stroke();

  ctx.restore();
}

// ---------------------------------------------------------------------------
// Tactile stacked deck texture painter (Projects, Links, Media)
// ---------------------------------------------------------------------------

function drawDeckStackCanvas(canvas, deck, count, isActive, w, h, p) {
  const ctx = canvas.getContext('2d');
  ctx.save();
  ctx.setTransform(TEX_SCALE, 0, 0, TEX_SCALE, 0, 0);
  ctx.clearRect(0, 0, w, h);

  const acc = accentColor(p, deck.accent);
  const stackPad = 8;
  const topW = w - 6;
  const topH = h - stackPad - 2;
  const radius = 13;

  // Draw 3 layered lower card edges to look like a real physical deck stack
  for (let layer = 3; layer >= 1; layer -= 1) {
    const oy = layer * 2.6;
    const inset = layer * 1.6;
    roundedRectPath(ctx, 3 + inset, oy + 1, topW - inset * 2, topH, radius);
    ctx.fillStyle = layer % 2 === 0 ? p.surface0 : p.mantle;
    ctx.fill();
    ctx.strokeStyle = p.surface1;
    ctx.lineWidth = 1;
    ctx.stroke();
  }

  // Top card of stack
  ctx.save();
  roundedRectPath(ctx, 3, 1, topW, topH, radius);
  ctx.fillStyle = isActive ? p.base : p.mantle;
  ctx.fill();
  ctx.clip();

  // Subtle Balatro diagonal card-back diamond motif
  ctx.strokeStyle = isActive
    ? (p.name === 'latte' ? 'rgba(136, 57, 239, 0.11)' : 'rgba(203, 166, 247, 0.11)')
    : (p.name === 'latte' ? 'rgba(76, 79, 105, 0.06)' : 'rgba(205, 214, 244, 0.05)');
  ctx.lineWidth = 1;
  for (let x = -topH; x < topW + topH; x += 14) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x + topH, topH);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(x + topH, 0);
    ctx.lineTo(x, topH);
    ctx.stroke();
  }

  // Top aurora bar
  const grad = ctx.createLinearGradient(3, 0, 3 + topW, 0);
  grad.addColorStop(0, acc);
  grad.addColorStop(1, p.blue);
  ctx.fillStyle = grad;
  ctx.fillRect(3, 1, topW, isActive ? 4.5 : 3);

  // Subtle inner card-back inset border
  roundedRectPath(ctx, 8, 7, topW - 10, topH - 11, 9);
  ctx.strokeStyle = isActive ? acc : p.innerFrame;
  ctx.globalAlpha = isActive ? 0.35 : 1;
  ctx.lineWidth = 1;
  ctx.stroke();
  ctx.globalAlpha = 1;

  // Deck title
  ctx.font = `700 ${w < 130 ? 13.5 : 16}px ${FONT_SANS}`;
  ctx.fillStyle = p.text;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(deck.label, w / 2, topH * 0.42);

  // Card count pill
  const countStr = `${count} ${count === 1 ? 'card' : 'cards'}`;
  ctx.font = `600 ${w < 130 ? 10 : 11}px ${FONT_MONO}`;
  const pillW = Math.min(topW - 20, ctx.measureText(countStr).width + 18);
  const pillH = 20;
  const pillX = w / 2 - pillW / 2;
  const pillY = topH * 0.68 - pillH / 2;

  roundedRectPath(ctx, pillX, pillY, pillW, pillH, 10);
  ctx.fillStyle = isActive ? acc : p.base;
  ctx.fill();
  if (!isActive) {
    ctx.strokeStyle = p.surface1;
    ctx.lineWidth = 1;
    ctx.stroke();
  }

  ctx.fillStyle = isActive ? (p.name === 'latte' ? '#eff1f5' : '#11111b') : p.subtext;
  ctx.fillText(countStr, w / 2, pillY + pillH / 2 + 0.5);

  ctx.restore();

  // Crisp outer border
  roundedRectPath(ctx, 3, 1, topW, topH, radius);
  ctx.strokeStyle = isActive ? acc : p.surface1;
  ctx.lineWidth = isActive ? 2 : 1.5;
  ctx.stroke();

  ctx.restore();
}

// ---------------------------------------------------------------------------
// PixiJS v8 Tabletop Initialization
// ---------------------------------------------------------------------------

async function initPixi() {
  if (!els.pixiRoot) {
    return false;
  }

  const p = palette();
  const app = new PIXI.Application();

  try {
    await app.init({
      preference: 'webgl',
      antialias: true,
      autoDensity: true,
      resolution: Math.min(window.devicePixelRatio || 1, 2),
      width: window.innerWidth,
      height: window.innerHeight,
      background: p.crustHex,
      backgroundAlpha: 1,
    });
  } catch (_) {
    try {
      await app.init({
        antialias: true,
        autoDensity: true,
        resolution: Math.min(window.devicePixelRatio || 1, 2),
        width: window.innerWidth,
        height: window.innerHeight,
        background: p.crustHex,
        backgroundAlpha: 1,
      });
    } catch (_) {
      return false;
    }
  }

  els.pixiRoot.textContent = '';
  els.pixiRoot.appendChild(app.canvas);
  state.app = app;

  // 1. Subtle Balatro Catppuccin swirling background shader
  const bgSprite = new PIXI.Sprite(PIXI.Texture.WHITE);
  bgSprite.width = window.innerWidth;
  bgSprite.height = window.innerHeight;
  bgSprite.tint = p.crustHex;

  try {
    const filter = PIXI.Filter.from({
      gl: {
        vertex: PIXI.defaultFilterVert,
        fragment: BALATRO_FRAG,
      },
      resources: {
        balatroUniforms: {
          uTime: { value: 0, type: 'f32' },
          uResolution: { value: new Float32Array([window.innerWidth, window.innerHeight]), type: 'vec2<f32>' },
          uColorA: { value: new Float32Array(hexToRgb01(p.swirlA)), type: 'vec3<f32>' },
          uColorB: { value: new Float32Array(hexToRgb01(p.swirlB)), type: 'vec3<f32>' },
          uColorC: { value: new Float32Array(hexToRgb01(p.swirlC)), type: 'vec3<f32>' },
          uAccent: { value: new Float32Array(hexToRgb01(p.swirlAccent)), type: 'vec3<f32>' },
        },
      },
    });
    bgSprite.filterArea = new PIXI.Rectangle(0, 0, window.innerWidth, window.innerHeight);
    bgSprite.filters = [filter];
    state.bgFilter = filter;
  } catch (_) {
    state.bgFilter = null;
  }

  state.bgSprite = bgSprite;
  app.stage.addChild(bgSprite);

  // 2. Scroll-synced world container
  const world = new PIXI.Container();
  world.sortableChildren = true;
  app.stage.addChild(world);
  state.world = world;

  const decksLayer = new PIXI.Container();
  decksLayer.zIndex = 10;
  world.addChild(decksLayer);
  state.decksLayer = decksLayer;

  const cardsLayer = new PIXI.Container();
  cardsLayer.zIndex = 20;
  cardsLayer.sortableChildren = true;
  world.addChild(cardsLayer);
  state.cardsLayer = cardsLayer;

  const particlesLayer = new PIXI.Container();
  particlesLayer.zIndex = 30;
  world.addChild(particlesLayer);
  state.particlesLayer = particlesLayer;

  // Shared procedural textures
  state.shadowTex = buildSoftShadowTexture(CARD_W, CARD_H, CARD_RADIUS, 26);
  state.foilTex = buildFoilTexture(CARD_W, CARD_H, CARD_RADIUS);
  state.sparkleTex = buildSparkleTexture();

  app.ticker.add(onTick);
  return true;
}

function updateBackgroundTheme() {
  const p = palette();
  if (state.app && state.app.renderer && state.app.renderer.background) {
    state.app.renderer.background.color = p.crustHex;
  }
  if (state.bgSprite) {
    state.bgSprite.tint = p.crustHex;
  }
  if (state.bgFilter && state.bgFilter.resources && state.bgFilter.resources.balatroUniforms) {
    const u = state.bgFilter.resources.balatroUniforms.uniforms;
    u.uColorA.set(hexToRgb01(p.swirlA));
    u.uColorB.set(hexToRgb01(p.swirlB));
    u.uColorC.set(hexToRgb01(p.swirlC));
    u.uAccent.set(hexToRgb01(p.swirlAccent));
  }
}

function refreshPixiForTheme() {
  const p = palette();
  updateBackgroundTheme();

  for (const drec of state.deckRecords) {
    const count = deckItems(drec.deck.key).length;
    drawDeckStackCanvas(drec.canvas, drec.deck, count, state.activeDeck === drec.deck.key, state.layout.deckW, state.layout.deckH, p);
    drec.texture.source.update();
    if (drec.shadowSprite) {
      drec.shadowSprite.alpha = p.shadowAlpha * 0.85;
    }
  }

  for (const rec of state.cardRecords) {
    rec.shadowSprite.alpha = p.shadowAlpha;
    if (rec.hoverOutline) {
      drawCardHoverOutline(rec.hoverOutline, resolveCardAccent(rec.deckKey, rec.item, p));
    }
    drawCardFaceToCanvas(rec.canvas, rec.deckKey, rec.item, p, null);
    rec.texture.source.update();
    resolveCardArtwork(rec.deckKey, rec.item, p).then((art) => {
      if (rec.destroyed || state.theme !== p.name) {
        return;
      }
      drawCardFaceToCanvas(rec.canvas, rec.deckKey, rec.item, p, art);
      rec.texture.source.update();
    });
  }
}

// ---------------------------------------------------------------------------
// Layout geometry (decks above floating cards, open zero-chrome tabletop)
// ---------------------------------------------------------------------------

function measureLayout() {
  const viewportW = window.innerWidth;
  const viewportH = window.innerHeight;
  const spacerRect = els.scrollSpacer ? els.scrollSpacer.getBoundingClientRect() : { top: 96 };
  const originY = Math.round(spacerRect.top + window.scrollY);

  const narrow = viewportW < 600;
  const deckGap = narrow ? 10 : 20;
  const deckW = narrow ? Math.max(96, Math.floor((viewportW - 28 - deckGap * 2) / 3)) : 176;
  const deckH = narrow ? 84 : 96;

  const usableW = Math.min(1260, Math.max(260, viewportW - (narrow ? 24 : 56)));
  const cardGapX = narrow ? 16 : 26;
  const cardGapY = narrow ? 22 : 30;
  const cardScale = viewportW < 340 ? Math.max(0.84, (viewportW - 28) / CARD_W) : 1;
  const effW = CARD_W * cardScale;
  const effH = CARD_H * cardScale;
  const cols = Math.max(1, Math.min(5, Math.floor((usableW + cardGapX) / (effW + cardGapX))));

  state.layout = {
    originY,
    viewportW,
    viewportH,
    deckW,
    deckH,
    deckGap,
    cardScale,
    effW,
    effH,
    cardGapX,
    cardGapY,
    cols,
  };
}

function buildDecks() {
  if (!state.decksLayer) {
    return;
  }

  for (const old of state.deckRecords) {
    old.container.destroy({ children: true });
    old.texture.destroy(true);
  }
  state.deckRecords = [];
  if (state.deckShadowTex) {
    state.deckShadowTex.destroy(true);
  }

  const p = palette();
  const { viewportW, originY, deckW, deckH, deckGap } = state.layout;
  state.deckShadowTex = buildSoftShadowTexture(deckW, deckH, 13, 18);

  const totalW = DECKS.length * deckW + (DECKS.length - 1) * deckGap;
  const startX = (viewportW - totalW) / 2;
  const centerY = originY + 12 + deckH / 2;

  if (els.a11yDecks) {
    els.a11yDecks.textContent = '';
  }

  DECKS.forEach((deck, idx) => {
    const count = deckItems(deck.key).length;
    const isActive = state.activeDeck === deck.key;
    const { canvas } = createCanvas(deckW, deckH);
    drawDeckStackCanvas(canvas, deck, count, isActive, deckW, deckH, p);
    const texture = canvasToTexture(canvas);

    const container = new PIXI.Container();
    const homeX = startX + idx * (deckW + deckGap) + deckW / 2;
    const homeY = centerY;
    container.position.set(homeX, homeY);

    const shadowSprite = new PIXI.Sprite(state.deckShadowTex);
    shadowSprite.anchor.set(0.5);
    shadowSprite.position.set(0, 7);
    shadowSprite.alpha = p.shadowAlpha * 0.85;
    container.addChild(shadowSprite);

    const sprite = new PIXI.Sprite(texture);
    sprite.anchor.set(0.5);
    sprite.width = deckW;
    sprite.height = deckH;
    container.addChild(sprite);

    state.decksLayer.addChild(container);

    const btn = el('button', 'a11y-deck-btn', `${deck.label} deck, ${count} ${count === 1 ? 'card' : 'cards'}`);
    btn.type = 'button';
    btn.dataset.deck = deck.key;
    btn.setAttribute('aria-pressed', String(isActive));
    btn.setAttribute('aria-label', `${deck.label} deck (${count} ${count === 1 ? 'card' : 'cards'})`);
    btn.style.left = `${Math.round(homeX - deckW / 2)}px`;
    btn.style.top = `${Math.round(homeY - originY - deckH / 2)}px`;
    btn.style.width = `${Math.round(deckW)}px`;
    btn.style.height = `${Math.round(deckH)}px`;

    btn.addEventListener('pointerenter', () => {
      state.hoveredDeckKey = deck.key;
    });
    btn.addEventListener('pointerleave', () => {
      if (state.hoveredDeckKey === deck.key) {
        state.hoveredDeckKey = null;
      }
    });
    btn.addEventListener('focus', () => {
      state.hoveredDeckKey = deck.key;
    });
    btn.addEventListener('blur', () => {
      if (state.hoveredDeckKey === deck.key) {
        state.hoveredDeckKey = null;
      }
    });
    btn.addEventListener('click', () => {
      switchDeck(deck.key);
    });
    if (els.a11yDecks) {
      els.a11yDecks.appendChild(btn);
    }

    state.deckRecords.push({
      deck,
      canvas,
      texture,
      container,
      shadowSprite,
      sprite,
      btn,
      homeX,
      homeY,
      scale: 1,
      pressBounce: 0,
      rot: (idx - 1) * 0.018,
    });
  });
}

function drawCardHoverOutline(graphics, accentHex) {
  graphics.clear();
  graphics.roundRect(-CARD_W / 2 - 1, -CARD_H / 2 - 1, CARD_W + 2, CARD_H + 2, CARD_RADIUS + 1);
  graphics.stroke({ width: 2.5, color: hexToNumber(accentHex), alpha: 0.92 });
}

function computeCardSlots(count) {
  const { viewportW, originY, deckH, effW, effH, cardGapX, cardGapY, cols } = state.layout;
  const effCols = Math.max(1, Math.min(cols, count));
  const rows = Math.ceil(count / effCols);
  const cardsTopY = originY + 12 + deckH + 34;

  const slots = [];
  let maxBottom = cardsTopY + 180;

  for (let i = 0; i < count; i += 1) {
    const row = Math.floor(i / effCols);
    const col = i % effCols;
    const itemsInRow = row === rows - 1 ? count - row * effCols : effCols;
    const rowWidth = itemsInRow * effW + Math.max(0, itemsInRow - 1) * cardGapX;
    const startX = (viewportW - rowWidth) / 2;

    // Gentle playing-card fan arc across each row
    const normCol = itemsInRow > 1 ? (col / (itemsInRow - 1)) * 2 - 1 : 0;
    const arcY = normCol * normCol * 10 + seeded(i + 3) * 4;
    const baseRot = (normCol * 1.35 + seeded(i + 17) * 0.85) * (Math.PI / 180);

    const cx = startX + col * (effW + cardGapX) + effW / 2;
    const cy = cardsTopY + row * (effH + cardGapY) + effH / 2 + arcY;
    slots.push({ x: cx, y: cy, baseRot });

    const bottom = cy + effH / 2;
    if (bottom > maxBottom) {
      maxBottom = bottom;
    }
  }

  const spacerHeight = Math.max(460, Math.ceil(maxBottom - originY + 42));
  if (els.scrollSpacer) {
    els.scrollSpacer.style.height = `${spacerHeight}px`;
  }
  return slots;
}

function whipOutCurrentCards() {
  const reduced = isReducedMotion();
  for (let i = 0; i < state.cardRecords.length; i += 1) {
    const rec = state.cardRecords[i];
    if (reduced) {
      rec.destroyed = true;
      rec.container.destroy({ children: true });
      rec.texture.destroy(true);
      continue;
    }
    const side = i % 2 === 0 ? -1 : 1;
    rec.exiting = true;
    rec.exitVx = side * (2250 + Math.abs(seeded(i + 9)) * 450);
    rec.exitVy = -180 + seeded(i + 23) * 260;
    rec.exitVRot = side * (8.5 + Math.abs(seeded(i + 13)) * 3.5);
    rec.exitLife = 0.3;
    rec.container.zIndex = 5;
    state.exitingCards.push(rec);
  }
  state.cardRecords = [];
  state.hoveredCardId = null;
  state.focusedCardId = null;
}

function spawnDealSparkles(x, y, accentHex) {
  if (!state.particlesLayer || !state.sparkleTex || isReducedMotion()) {
    return;
  }
  const color = hexToNumber(accentHex);
  const count = 5;
  for (let i = 0; i < count; i += 1) {
    const sp = new PIXI.Sprite(state.sparkleTex);
    sp.anchor.set(0.5);
    sp.tint = color;
    const angle = (Math.PI * 2 * i) / count + seeded(x + i) * 0.6;
    const dist = 54 + Math.abs(seeded(y + i * 3)) * 52;
    sp.position.set(x + Math.cos(angle) * dist * 0.7, y + Math.sin(angle) * dist * 0.9);
    sp.scale.set(0.55 + Math.abs(seeded(i + 7)) * 0.5);
    sp.alpha = 0.95;
    state.particlesLayer.addChild(sp);

    state.particles.push({
      sprite: sp,
      vx: Math.cos(angle) * (48 + Math.abs(seeded(i + 19)) * 42),
      vy: Math.sin(angle) * (48 + Math.abs(seeded(i + 29)) * 42) - 18,
      vRot: (i % 2 === 0 ? 1 : -1) * 3.8,
      life: 0.44,
      maxLife: 0.44,
    });
  }
}

function dealDeck(deckKey) {
  if (!state.cardsLayer) {
    return;
  }

  whipOutCurrentCards();

  const items = deckItems(deckKey);
  const p = palette();
  const reduced = isReducedMotion();
  const { viewportW, originY, cardScale, effW, effH } = state.layout;
  const slots = computeCardSlots(items.length);

  if (!reduced) {
    state.swirlBoost = 2.4;
  }

  if (els.a11yCards) {
    els.a11yCards.textContent = '';
  }

  items.forEach((item, idx) => {
    const slot = slots[idx];
    const cardId = `${deckKey}:${item.id || idx}`;
    const accent = resolveCardAccent(deckKey, item, p);
    const { canvas } = createCanvas(CARD_W, CARD_H);
    drawCardFaceToCanvas(canvas, deckKey, item, p, null);
    const texture = canvasToTexture(canvas);

    const container = new PIXI.Container();
    container.zIndex = 20 + idx;

    const shadowSprite = new PIXI.Sprite(state.shadowTex);
    shadowSprite.anchor.set(0.5);
    shadowSprite.position.set(0, 10);
    shadowSprite.alpha = p.shadowAlpha;
    container.addChild(shadowSprite);

    const cardSprite = new PIXI.Sprite(texture);
    cardSprite.anchor.set(0.5);
    cardSprite.width = CARD_W;
    cardSprite.height = CARD_H;
    container.addChild(cardSprite);

    const foilMask = new PIXI.Graphics();
    foilMask.roundRect(-CARD_W / 2, -CARD_H / 2, CARD_W, CARD_H, CARD_RADIUS);
    foilMask.fill(0xffffff);
    container.addChild(foilMask);

    const foilSprite = new PIXI.Sprite(state.foilTex);
    foilSprite.anchor.set(0.5);
    foilSprite.width = CARD_W * 1.45;
    foilSprite.height = CARD_H * 1.45;
    foilSprite.alpha = 0;
    foilSprite.blendMode = 'screen';
    foilSprite.mask = foilMask;
    container.addChild(foilSprite);

    const hoverOutline = new PIXI.Graphics();
    drawCardHoverOutline(hoverOutline, accent);
    hoverOutline.alpha = 0;
    container.addChild(hoverOutline);

    // Casino-dealer alternating off-screen fling start pose
    const side = idx % 2 === 0 ? -1 : 1;
    const startX = side < 0 ? -280 : viewportW + 280;
    const startY = slot.y + seeded(idx + 5) * 85 - 35;
    const spinDeg = 220 + Math.abs(seeded(idx + 11)) * 40;
    const startRot = side * spinDeg * (Math.PI / 180);

    const rec = {
      id: cardId,
      deckKey,
      item,
      index: idx,
      accent,
      canvas,
      texture,
      container,
      shadowSprite,
      cardSprite,
      foilSprite,
      hoverOutline,
      btn: null,
      destroyed: false,

      homeX: slot.x,
      homeY: slot.y,
      baseRot: slot.baseRot,

      x: reduced ? slot.x : startX,
      y: reduced ? slot.y : startY,
      vx: 0,
      vy: 0,
      rot: reduced ? slot.baseRot : startRot,
      vRot: 0,
      scaleX: cardScale,
      scaleY: cardScale,
      skewX: 0,
      skewY: 0,

      dealDelay: reduced ? 0 : idx * 0.038,
      dealt: reduced,
      sparkled: reduced,
      flipState: 'idle', // 'idle' | 'closing' | 'hidden'
      flipProgress: 0,

      phaseX: idx * 1.13 + 0.5,
      phaseY: idx * 0.89 + 1.7,
      phaseRot: idx * 1.41,
      freqX: 0.65 + (idx % 5) * 0.08,
      freqY: 0.78 + (idx % 4) * 0.09,
    };

    container.position.set(rec.x, rec.y);
    container.rotation = rec.rot;
    container.scale.set(cardScale);
    state.cardsLayer.addChild(container);

    // Accessible aligned button proxy for pointer & keyboard interaction
    const label = buildCardAriaLabel(deckKey, item);
    const btn = el('button', 'a11y-card-btn', label);
    btn.type = 'button';
    btn.dataset.cardId = cardId;
    btn.setAttribute('aria-label', label);
    btn.style.left = `${Math.round(slot.x - effW / 2)}px`;
    btn.style.top = `${Math.round(slot.y - originY - effH / 2)}px`;
    btn.style.width = `${Math.round(effW)}px`;
    btn.style.height = `${Math.round(effH)}px`;

    btn.addEventListener('pointerenter', () => {
      state.hoveredCardId = cardId;
    });
    btn.addEventListener('pointerleave', () => {
      if (state.hoveredCardId === cardId) {
        state.hoveredCardId = null;
      }
    });
    btn.addEventListener('focus', () => {
      state.focusedCardId = cardId;
    });
    btn.addEventListener('blur', () => {
      if (state.focusedCardId === cardId) {
        state.focusedCardId = null;
      }
    });
    btn.addEventListener('click', () => {
      triggerCardOpen(rec, btn);
    });
    btn.addEventListener('keydown', (ev) => {
      handleCardKeyNav(ev, idx);
    });

    if (els.a11yCards) {
      els.a11yCards.appendChild(btn);
    }
    rec.btn = btn;
    state.cardRecords.push(rec);

    // Async artwork hydration
    resolveCardArtwork(deckKey, item, p).then((art) => {
      if (rec.destroyed || state.theme !== p.name) {
        return;
      }
      drawCardFaceToCanvas(rec.canvas, deckKey, item, p, art);
      rec.texture.source.update();
    });
  });
}

function buildCardAriaLabel(deckKey, item) {
  if (deckKey === 'projects') {
    const hosts = (item.hosts || []).length;
    return `${item.name}, ${item.language || 'project'}, ${item.stars_formatted || item.stars || 0} stars${hosts > 0 ? `, ${hosts} package hosts` : ''}. Open details.`;
  }
  if (deckKey === 'links') {
    return `${item.title} (${item.category || 'Link'})${item.badge ? `, ${item.badge}` : ''}. Open details.`;
  }
  return `${item.title} (${item.category || 'Media'})${item.badge ? `, ${item.badge}` : ''}. Open details.`;
}

function handleCardKeyNav(ev, index) {
  const keys = ['ArrowRight', 'ArrowLeft', 'ArrowDown', 'ArrowUp', 'Home', 'End'];
  if (!keys.includes(ev.key) || !els.a11yCards) {
    return;
  }
  const btns = Array.from(els.a11yCards.querySelectorAll('.a11y-card-btn'));
  if (btns.length === 0) {
    return;
  }
  const cols = state.layout.cols || 1;
  let next = index;
  if (ev.key === 'ArrowRight') {
    next = (index + 1) % btns.length;
  } else if (ev.key === 'ArrowLeft') {
    next = (index - 1 + btns.length) % btns.length;
  } else if (ev.key === 'ArrowDown') {
    next = Math.min(btns.length - 1, index + cols);
  } else if (ev.key === 'ArrowUp') {
    next = Math.max(0, index - cols);
  } else if (ev.key === 'Home') {
    next = 0;
  } else if (ev.key === 'End') {
    next = btns.length - 1;
  }
  if (next !== index && btns[next]) {
    ev.preventDefault();
    btns[next].focus();
  }
}

function switchDeck(deckKey) {
  if (!DECKS.some((d) => d.key === deckKey)) {
    return;
  }
  const wasSame = state.activeDeck === deckKey;
  state.activeDeck = deckKey;

  const p = palette();
  for (const drec of state.deckRecords) {
    const isActive = drec.deck.key === deckKey;
    const count = deckItems(drec.deck.key).length;
    drawDeckStackCanvas(drec.canvas, drec.deck, count, isActive, state.layout.deckW, state.layout.deckH, p);
    drec.texture.source.update();
    if (drec.btn) {
      drec.btn.setAttribute('aria-pressed', String(isActive));
    }
    if (isActive && !isReducedMotion()) {
      drec.pressBounce = 1;
    }
  }

  dealDeck(deckKey);
  if (!wasSame) {
    const meta = DECKS.find((d) => d.key === deckKey);
    announce(`Dealt ${meta ? meta.label : deckKey} deck, ${deckItems(deckKey).length} cards.`);
  }
}

function relayoutScene() {
  if (!state.app) {
    return;
  }
  const w = window.innerWidth;
  const h = window.innerHeight;
  state.app.renderer.resize(w, h);

  if (state.bgSprite) {
    state.bgSprite.width = w;
    state.bgSprite.height = h;
    if (state.bgFilter) {
      state.bgSprite.filterArea = new PIXI.Rectangle(0, 0, w, h);
      state.bgFilter.resources.balatroUniforms.uniforms.uResolution.set([w, h]);
    }
  }

  measureLayout();
  buildDecks();

  const { originY, cardScale, effW, effH } = state.layout;
  const slots = computeCardSlots(state.cardRecords.length);
  const reduced = isReducedMotion();

  state.cardRecords.forEach((rec, idx) => {
    const slot = slots[idx];
    rec.homeX = slot.x;
    rec.homeY = slot.y;
    rec.baseRot = slot.baseRot;
    if (reduced || !rec.dealing) {
      rec.x = slot.x;
      rec.y = slot.y;
      rec.vx = 0;
      rec.vy = 0;
      rec.rot = slot.baseRot;
      rec.vRot = 0;
      rec.container.position.set(rec.x, rec.y);
      rec.container.rotation = rec.rot;
      rec.container.scale.set(cardScale);
    }
    if (rec.btn) {
      rec.btn.style.left = `${Math.round(slot.x - effW / 2)}px`;
      rec.btn.style.top = `${Math.round(slot.y - originY - effH / 2)}px`;
      rec.btn.style.width = `${Math.round(effW)}px`;
      rec.btn.style.height = `${Math.round(effH)}px`;
    }
  });
}

// ---------------------------------------------------------------------------
// Frame ticker: shader swirl, casino deal springs, idle float, hover 3D tilt
// ---------------------------------------------------------------------------

function onTick(ticker) {
  const dt = Math.min(0.05, Math.max(0.001, ticker.deltaMS / 1000));
  const reduced = isReducedMotion();
  const p = palette();

  // Keep PixiJS world 1:1 locked to native page scroll
  if (state.world) {
    state.world.y = -window.scrollY;
  }

  // Advance swirling Balatro background shader (frozen when reduced motion)
  if (!reduced && !state.modalOpen) {
    state.swirlBoost = Math.max(0, state.swirlBoost - dt * 2.8);
    state.shaderTime += dt * (1 + state.swirlBoost);
    if (state.bgFilter && state.bgFilter.resources && state.bgFilter.resources.balatroUniforms) {
      state.bgFilter.resources.balatroUniforms.uniforms.uTime = state.shaderTime;
    }
  }

  const t = performance.now() / 1000;
  const { cardScale } = state.layout;

  // Animate 3 tactile deck stacks
  for (let i = 0; i < state.deckRecords.length; i += 1) {
    const drec = state.deckRecords[i];
    const isHovered = state.hoveredDeckKey === drec.deck.key;
    const isActive = state.activeDeck === drec.deck.key;

    if (drec.pressBounce > 0) {
      drec.pressBounce = Math.max(0, drec.pressBounce - dt * 5.5);
    }
    const bounceOffset = reduced ? 0 : Math.sin(drec.pressBounce * Math.PI) * 5;
    const targetScale = reduced ? 1 : (isHovered ? 1.045 : (isActive ? 1.02 : 1)) - drec.pressBounce * 0.04;
    const targetY = drec.homeY + (reduced ? 0 : (isHovered ? -4 : (isActive ? -2 : 0)) + bounceOffset);
    const targetRot = reduced ? 0 : (isHovered ? (i - 1) * 0.032 : (i - 1) * 0.014);

    drec.scale += (targetScale - drec.scale) * Math.min(1, dt * 14);
    drec.container.y += (targetY - drec.container.y) * Math.min(1, dt * 14);
    drec.container.rotation += (targetRot - drec.container.rotation) * Math.min(1, dt * 14);
    drec.container.scale.set(drec.scale);
  }

  // Whip off outgoing cards on deck change
  for (let i = state.exitingCards.length - 1; i >= 0; i -= 1) {
    const rec = state.exitingCards[i];
    rec.exitLife -= dt;
    rec.x += rec.exitVx * dt;
    rec.y += rec.exitVy * dt;
    rec.rot += rec.exitVRot * dt;
    rec.container.position.set(rec.x, rec.y);
    rec.container.rotation = rec.rot;
    rec.container.alpha = Math.max(0, rec.exitLife / 0.3);
    if (rec.exitLife <= 0) {
      rec.destroyed = true;
      rec.container.destroy({ children: true });
      rec.texture.destroy(true);
      state.exitingCards.splice(i, 1);
    }
  }

  // Update sparkle particles
  for (let i = state.particles.length - 1; i >= 0; i -= 1) {
    const pt = state.particles[i];
    pt.life -= dt;
    pt.sprite.x += pt.vx * dt;
    pt.sprite.y += pt.vy * dt;
    pt.sprite.rotation += pt.vRot * dt;
    const ratio = Math.max(0, pt.life / pt.maxLife);
    pt.sprite.alpha = ratio;
    pt.sprite.scale.set(ratio * 0.9);
    if (pt.life <= 0) {
      pt.sprite.destroy();
      state.particles.splice(i, 1);
    }
  }

  // Animate active deck cards
  for (let i = 0; i < state.cardRecords.length; i += 1) {
    const rec = state.cardRecords[i];
    const isHovered = !state.modalOpen && (state.hoveredCardId === rec.id || state.focusedCardId === rec.id);

    // Handle brisk flip collapse when opening detail modal
    if (rec.flipState === 'closing') {
      rec.flipProgress += dt / 0.15;
      if (rec.flipProgress >= 1) {
        rec.flipProgress = 1;
        rec.flipState = 'hidden';
        rec.container.scale.set(0.03 * cardScale, cardScale * 1.06);
        openDetailModal(rec.deckKey, rec.item, state.lastFocusedEl);
      } else {
        const sx = Math.max(0.03, Math.cos(rec.flipProgress * Math.PI * 0.5));
        rec.container.scale.set(sx * cardScale * 1.07, cardScale * 1.07);
      }
      continue;
    }

    if (rec.flipState === 'hidden') {
      continue;
    }

    if (reduced) {
      rec.container.position.set(rec.homeX, rec.homeY);
      rec.container.rotation = rec.baseRot;
      rec.container.scale.set(isHovered ? cardScale * 1.03 : cardScale);
      rec.container.skew.set(0, 0);
      rec.container.zIndex = isHovered ? 200 : 20 + i;
      rec.hoverOutline.alpha = isHovered ? 1 : 0;
      rec.foilSprite.alpha = 0;
      continue;
    }

    // Staggered casino deal wait
    if (rec.dealDelay > 0) {
      rec.dealDelay -= dt;
      continue;
    }

    // Target idle harmonic float (±4px bob, ±1.6deg sway)
    const bobX = isHovered ? 0 : Math.sin(t * rec.freqX + rec.phaseX) * 3.5;
    const bobY = isHovered ? -8 : Math.cos(t * rec.freqY + rec.phaseY) * 4.0;
    const targetX = rec.homeX + bobX;
    const targetY = rec.homeY + bobY;

    // Pointer-normalized offset [-1, 1] for Balatro 3D hover tilt
    let nx = 0;
    let ny = 0;
    if (isHovered && state.pointer.active) {
      const screenX = rec.homeX;
      const screenY = rec.homeY - window.scrollY;
      nx = Math.max(-1, Math.min(1, (state.pointer.x - screenX) / (CARD_W * 0.55)));
      ny = Math.max(-1, Math.min(1, (state.pointer.y - screenY) / (CARD_H * 0.55)));
    }

    const idleRot = rec.baseRot + Math.sin(t * 0.95 + rec.phaseRot) * (1.6 * (Math.PI / 180));
    const targetRot = isHovered ? nx * 0.045 : idleRot;

    // Underdamped spring physics for snappy deal-in overshoot & smooth float
    const posK = rec.dealt ? (isHovered ? 150 : 62) : 185;
    const posDamp = rec.dealt ? (isHovered ? 18 : 11) : 16.5;
    const rotK = rec.dealt ? 95 : 170;
    const rotDamp = rec.dealt ? 13 : 15.5;

    rec.vx += ((targetX - rec.x) * posK - rec.vx * posDamp) * dt;
    rec.vy += ((targetY - rec.y) * posK - rec.vy * posDamp) * dt;
    rec.x += rec.vx * dt;
    rec.y += rec.vy * dt;

    rec.vRot += ((targetRot - rec.rot) * rotK - rec.vRot * rotDamp) * dt;
    rec.rot += rec.vRot * dt;

    // Mark dealt + burst subtle sparkles when card snaps into slot
    if (!rec.dealt && Math.hypot(targetX - rec.x, targetY - rec.y) < 18 && Math.hypot(rec.vx, rec.vy) < 220) {
      rec.dealt = true;
      if (!rec.sparkled) {
        rec.sparkled = true;
        spawnDealSparkles(rec.homeX, rec.homeY, rec.accent);
      }
    }

    // Hover scale, perspective tilt skew, shadow elevation, & foil sheen
    const targetScaleX = cardScale * (isHovered ? 1.07 - Math.abs(nx) * 0.025 : 1);
    const targetScaleY = cardScale * (isHovered ? 1.07 - Math.abs(ny) * 0.02 : 1);
    const targetSkewX = isHovered ? -ny * 0.06 : 0;
    const targetSkewY = isHovered ? nx * 0.055 : 0;

    const lerp = Math.min(1, dt * 15);
    rec.scaleX += (targetScaleX - rec.scaleX) * lerp;
    rec.scaleY += (targetScaleY - rec.scaleY) * lerp;
    rec.skewX += (targetSkewX - rec.skewX) * lerp;
    rec.skewY += (targetSkewY - rec.skewY) * lerp;

    rec.container.position.set(rec.x, rec.y);
    rec.container.rotation = rec.rot;
    rec.container.scale.set(rec.scaleX, rec.scaleY);
    rec.container.skew.set(rec.skewX, rec.skewY);
    rec.container.zIndex = isHovered ? 200 : 20 + i;

    // Dynamic shadow offset & hover glow
    const targetShadowX = isHovered ? -nx * 12 : 0;
    const targetShadowY = isHovered ? 20 - ny * 7 : 10;
    rec.shadowSprite.x += (targetShadowX - rec.shadowSprite.x) * lerp;
    rec.shadowSprite.y += (targetShadowY - rec.shadowSprite.y) * lerp;

    const targetOutline = isHovered ? 1 : 0;
    rec.hoverOutline.alpha += (targetOutline - rec.hoverOutline.alpha) * lerp;

    const targetFoil = isHovered ? p.foilAlpha : 0;
    rec.foilSprite.alpha += (targetFoil - rec.foilSprite.alpha) * lerp;
    if (rec.foilSprite.alpha > 0.01) {
      rec.foilSprite.x = nx * 42;
      rec.foilSprite.y = ny * 48;
    }
  }
}

// ---------------------------------------------------------------------------
// Detail Modal (minimal HTML layer with full project/link/media details)
// ---------------------------------------------------------------------------

function mountSVG(container, rawSVG) {
  if (!container || !rawSVG || typeof rawSVG !== 'string') {
    return false;
  }
  try {
    svgMountSeq += 1;
    const suffix = `-m${svgMountSeq}`;
    let uniqueSVG = rawSVG
      .replace(/id="(vlogo-[^"]+)"/g, `id="$1${suffix}"`)
      .replace(/url\(#(vlogo-[^)]+)\)/g, `url(#$1${suffix})`)
      .replace(/href="#(vlogo-[^"]+)"/g, `href="#$1${suffix}"`);
    if (/^<svg\b(?![^>]*\bxmlns=)/i.test(uniqueSVG.trim())) {
      uniqueSVG = uniqueSVG.replace(/^(\s*<svg\b)/i, '$1 xmlns="http://www.w3.org/2000/svg"');
    }

    const parser = new DOMParser();
    const doc = parser.parseFromString(uniqueSVG, 'image/svg+xml');
    const svgRoot = doc.documentElement;
    if (!svgRoot || svgRoot.nodeName.toLowerCase() !== 'svg' || svgRoot.querySelector('parsererror')) {
      return false;
    }
    const imported = document.importNode(svgRoot, true);
    imported.setAttribute('aria-hidden', 'true');
    imported.setAttribute('focusable', 'false');
    container.appendChild(imported);
    return true;
  } catch (_) {
    return false;
  }
}

function mountArtworkDOM(container, item) {
  if (item.cover_url && isSafeImageURI(item.cover_url)) {
    container.classList.add('is-cover');
    const coverImg = document.createElement('img');
    coverImg.src = item.cover_url;
    coverImg.alt = '';
    coverImg.loading = 'lazy';
    coverImg.decoding = 'async';
    container.appendChild(coverImg);
    return;
  }
  if (item.logo_svg && mountSVG(container, item.logo_svg)) {
    return;
  }
  if (item.logo_data_uri && isSafeImageURI(item.logo_data_uri)) {
    const img = document.createElement('img');
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

function buildSubscribeRow(links) {
  const row = el('div', 'subscribe-row');
  for (const sub of links) {
    const pill = linkEl(sub.url, 'subscribe-pill');
    if (sub.icon_svg) {
      const iconSpan = el('span');
      mountSVG(iconSpan, sub.icon_svg);
      pill.appendChild(iconSpan);
    }
    pill.appendChild(document.createTextNode(sub.label));
    row.appendChild(pill);
  }
  return row;
}

function buildModalContent(deckKey, item) {
  const scroll = el('div', 'modal-scroll');

  const header = el('div', 'modal-header');
  const titleGroup = el('div', 'modal-title-group');
  const art = el('div', 'modal-art');
  mountArtworkDOM(art, item);

  const titleCol = el('div');
  const titleText = item.name || item.title || 'Details';
  const h2 = el('h2', 'modal-title', titleText);
  h2.id = 'modal-title';
  titleCol.appendChild(h2);

  let eyebrow = '';
  if (deckKey === 'projects') {
    eyebrow = item.full_name || '';
  } else if (deckKey === 'links') {
    eyebrow = (item.category || 'Link') + (item.host_label ? ` · ${item.host_label}` : '');
  } else if (deckKey === 'media') {
    const parts = [item.category || 'Media'];
    if (item.episode) {
      parts.push(`Episode #${item.episode}`);
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

  const closeBtn = el('button', 'modal-close', '×');
  closeBtn.type = 'button';
  closeBtn.setAttribute('aria-label', 'Close details');
  closeBtn.addEventListener('click', closeDetailModal);

  header.appendChild(titleGroup);
  header.appendChild(closeBtn);
  scroll.appendChild(header);

  if (deckKey === 'projects') {
    if (item.description) {
      scroll.appendChild(el('p', 'modal-desc', item.description));
    }

    const stats = el('div', 'detail-stats');
    stats.appendChild(el('span', 'detail-stat', `★ ${item.stars_formatted || item.stars || 0} stars`));
    if (item.language) {
      const langChip = el('span', 'detail-stat');
      const dot = el('span', 'lang-dot');
      dot.style.backgroundColor = item.language_color || 'var(--mauve)';
      langChip.appendChild(dot);
      langChip.appendChild(document.createTextNode(item.language));
      stats.appendChild(langChip);
    }
    if (item.latest_tag) {
      const relLabel = item.latest_tag + (item.release_age ? ` · ${item.release_age}` : '');
      const cls = `detail-stat${item.fresh ? ' is-fresh' : ''}`;
      if (item.release_url && isSafeURL(item.release_url)) {
        stats.appendChild(linkEl(item.release_url, cls, `Release ${relLabel}`));
      } else {
        stats.appendChild(el('span', cls, `Release ${relLabel}`));
      }
    }
    scroll.appendChild(stats);

    const actions = el('div', 'detail-actions');
    if (item.homepage && isSafeURL(item.homepage) && item.homepage !== item.repo_url) {
      actions.appendChild(linkEl(item.homepage, 'action-btn is-primary', 'Visit Homepage'));
    }
    if (item.repo_url && isSafeURL(item.repo_url)) {
      actions.appendChild(linkEl(item.repo_url, `action-btn${actions.childElementCount === 0 ? ' is-primary' : ''}`, 'View on GitHub'));
    }
    if (item.release_url && isSafeURL(item.release_url)) {
      actions.appendChild(linkEl(item.release_url, 'action-btn', `Latest Release (${item.latest_tag})`));
    }
    scroll.appendChild(actions);

    if (item.hosts && item.hosts.length > 0) {
      const hostSec = el('div', 'detail-section');
      hostSec.appendChild(el('h3', 'detail-section-title', 'Install via Haberdasher Packages'));
      const grid = el('div', 'host-tile-grid');
      const logos = (state.data && state.data.host_logos) || {};

      for (const h of item.hosts) {
        const tile = linkEl(h.url, 'host-tile');
        const logoBox = el('span', 'host-tile-logo');
        if (logos[h.host]) {
          mountSVG(logoBox, logos[h.host]);
        }
        tile.appendChild(logoBox);
        tile.appendChild(el('span', '', h.name));
        grid.appendChild(tile);
      }
      hostSec.appendChild(grid);
      scroll.appendChild(hostSec);
    }
  } else if (deckKey === 'links') {
    if (item.description) {
      scroll.appendChild(el('p', 'modal-desc', item.description));
    }
    if (item.meta && item.meta.length > 0) {
      const linkStats = el('div', 'detail-stats');
      for (const m of item.meta) {
        linkStats.appendChild(el('span', 'detail-stat', m));
      }
      scroll.appendChild(linkStats);
    }
    const linkActions = el('div', 'detail-actions');
    linkActions.appendChild(linkEl(item.url, 'action-btn is-primary', `Open ${item.title}`));
    scroll.appendChild(linkActions);
  } else if (deckKey === 'media') {
    if (item.summary) {
      scroll.appendChild(el('p', 'modal-desc', item.summary));
    }

    const mediaStats = el('div', 'detail-stats');
    if (item.episode) {
      mediaStats.appendChild(el('span', 'detail-stat', `#${item.episode}`));
    }
    if (item.duration) {
      mediaStats.appendChild(el('span', 'detail-stat', item.duration));
    }
    if (item.read_time_minutes > 0) {
      mediaStats.appendChild(el('span', 'detail-stat', `${item.read_time_minutes} min read`));
    }
    if (item.age) {
      mediaStats.appendChild(el('span', 'detail-stat', item.age));
    }
    if (mediaStats.childElementCount > 0) {
      scroll.appendChild(mediaStats);
    }

    const mediaActions = el('div', 'detail-actions');
    let primaryLabel = 'Open';
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
      mediaActions.appendChild(linkEl(item.site_url, 'action-btn', `All ${item.category || 'Entries'}`));
    }
    scroll.appendChild(mediaActions);

    if (item.subscribe_links && item.subscribe_links.length > 0) {
      const subSec = el('div', 'detail-section');
      subSec.appendChild(el('h3', 'detail-section-title', 'Subscribe'));
      subSec.appendChild(buildSubscribeRow(item.subscribe_links));
      scroll.appendChild(subSec);
    }

    if (item.entries && item.entries.length > 0) {
      const entrySec = el('div', 'detail-section');
      entrySec.appendChild(el('h3', 'detail-section-title', item.kind === 'podcast-hub' ? 'Recent Episodes' : 'Recent Posts'));
      const ul = el('ul', 'media-entry-list');
      for (const entry of item.entries) {
        const li = el('li', 'media-entry-item');
        const a = linkEl(entry.url, 'media-entry-link');
        const head = el('div', 'media-entry-head');
        const prefix = entry.episode ? `#${entry.episode} ` : '';
        head.appendChild(el('span', 'media-entry-title', prefix + entry.title));

        const metaBits = [];
        if (entry.duration) {
          metaBits.push(entry.duration);
        }
        if (entry.read_time_minutes > 0) {
          metaBits.push(`${entry.read_time_minutes} min read`);
        }
        if (entry.date_label) {
          metaBits.push(entry.date_label);
        }
        if (metaBits.length > 0) {
          head.appendChild(el('span', 'media-entry-meta', metaBits.join(' · ')));
        }
        a.appendChild(head);

        const summaryText = entry.subtitle || entry.summary;
        if (summaryText) {
          a.appendChild(el('p', 'media-entry-summary', summaryText));
        }
        li.appendChild(a);
        ul.appendChild(li);
      }
      entrySec.appendChild(ul);
      scroll.appendChild(entrySec);
    }
  }

  return scroll;
}

function triggerCardOpen(rec, triggerEl) {
  if (state.modalOpen || state.flippingCard) {
    return;
  }
  state.lastFocusedEl = triggerEl || document.activeElement;
  if (isReducedMotion() || !state.app) {
    openDetailModal(rec.deckKey, rec.item, state.lastFocusedEl);
    return;
  }
  state.flippingCard = rec;
  rec.flipState = 'closing';
  rec.flipProgress = 0;
  rec.container.zIndex = 240;
}

function openDetailModal(deckKey, item, triggerEl) {
  if (!els.modal || !els.modalCard) {
    return;
  }
  state.lastFocusedEl = triggerEl || state.lastFocusedEl || document.activeElement;
  state.modalOpen = true;

  els.modalCard.textContent = '';
  els.modalCard.appendChild(buildModalContent(deckKey, item));
  els.modal.hidden = false;
  document.body.classList.add('modal-open');

  const closeBtn = els.modalCard.querySelector('.modal-close');
  if (closeBtn) {
    closeBtn.focus();
  }
}

function closeDetailModal() {
  if (!state.modalOpen || !els.modal) {
    return;
  }
  state.modalOpen = false;
  els.modal.hidden = true;
  document.body.classList.remove('modal-open');

  if (state.flippingCard) {
    const rec = state.flippingCard;
    rec.flipState = 'idle';
    rec.flipProgress = 0;
    if (!rec.destroyed) {
      rec.container.scale.set(state.layout.cardScale);
    }
    state.flippingCard = null;
  }

  if (state.lastFocusedEl && typeof state.lastFocusedEl.focus === 'function') {
    state.lastFocusedEl.focus();
  }
  state.lastFocusedEl = null;
}

function trapModalFocus(ev) {
  if (!state.modalOpen || ev.key !== 'Tab' || !els.modalCard) {
    return;
  }
  const focusable = els.modalCard.querySelectorAll(
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
  );
  if (focusable.length === 0) {
    return;
  }
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  if (ev.shiftKey && document.activeElement === first) {
    ev.preventDefault();
    last.focus();
  } else if (!ev.shiftKey && document.activeElement === last) {
    ev.preventDefault();
    first.focus();
  }
}

// ---------------------------------------------------------------------------
// Events & Bootstrap
// ---------------------------------------------------------------------------

function bindEvents() {
  for (const btn of els.themeButtons) {
    btn.addEventListener('click', () => {
      applyThemeMode(btn.dataset.themeMode || 'system', true);
    });
  }

  if (colorSchemeMQ && typeof colorSchemeMQ.addEventListener === 'function') {
    colorSchemeMQ.addEventListener('change', () => {
      if (state.themeMode === 'system') {
        applyThemeMode('system', false);
      }
    });
  }

  if (reducedMotionMQ && typeof reducedMotionMQ.addEventListener === 'function') {
    reducedMotionMQ.addEventListener('change', () => {
      relayoutScene();
    });
  }

  window.addEventListener('pointermove', (ev) => {
    state.pointer.x = ev.clientX;
    state.pointer.y = ev.clientY;
    state.pointer.active = true;
  }, { passive: true });

  window.addEventListener('pointerleave', () => {
    state.pointer.active = false;
  });

  window.addEventListener('scroll', () => {
    if (state.world) {
      state.world.y = -window.scrollY;
    }
  }, { passive: true });

  window.addEventListener('resize', () => {
    window.clearTimeout(state.resizeTimer);
    state.resizeTimer = window.setTimeout(relayoutScene, 100);
  });

  if (els.modal) {
    els.modal.addEventListener('click', (ev) => {
      if (ev.target === els.modal) {
        closeDetailModal();
      }
    });
  }

  document.addEventListener('keydown', (ev) => {
    if (state.modalOpen && ev.key === 'Escape') {
      ev.preventDefault();
      closeDetailModal();
      return;
    }
    trapModalFocus(ev);
  });
}

async function bootstrap(data) {
  state.data = data;
  const candidateDeck = String(data.default_deck || 'projects').toLowerCase();
  state.activeDeck = DECKS.some((d) => d.key === candidateDeck) ? candidateDeck : 'projects';

  syncThemeControls();
  bindEvents();

  if (document.fonts && document.fonts.ready) {
    await Promise.race([
      document.fonts.ready,
      new Promise((resolve) => window.setTimeout(resolve, 450)),
    ]);
  }

  const ok = await initPixi();
  if (!ok) {
    return;
  }

  measureLayout();
  buildDecks();
  dealDeck(state.activeDeck);

  // Re-bake textures cleanly if web fonts finish loading slightly after initial paint
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(() => {
      if (state.app) {
        refreshPixiForTheme();
      }
    });
  }
}

function loadData() {
  const inlineNode = document.getElementById('site-data');
  if (inlineNode && inlineNode.textContent && inlineNode.textContent.trim().length > 0) {
    try {
      bootstrap(JSON.parse(inlineNode.textContent));
      return;
    } catch (_) {
      // Fall back to fetching data.json below.
    }
  }

  fetch('./data.json')
    .then((resp) => {
      if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}`);
      }
      return resp.json();
    })
    .then(bootstrap)
    .catch(() => {
      if (els.scrollSpacer) {
        els.scrollSpacer.textContent = 'Unable to load portfolio data.';
      }
    });
}

loadData();
