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
    foilAlpha: 0.13,
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
    foilAlpha: 0.11,
    swirlA: '#dce0e8',
    swirlB: '#d2d7e5',
    swirlC: '#c9d2e8',
    swirlAccent: '#d9ccf0',
  },
};

// Exact Haberdasher vendor platform colours from https://<host>.wimpys.world/logo.svg
const HOST_META = {
  apt: { label: 'APT', subtitle: 'Debian / Ubuntu', dark: '#E95420', light: '#E95420' },
  rpm: { label: 'RPM', subtitle: 'Fedora / RHEL', dark: '#EE0000', light: '#EE0000' },
  pkg: { label: 'PKG', subtitle: 'Arch Linux', dark: '#1793D1', light: '#1793D1' },
  apk: { label: 'APK', subtitle: 'Alpine Linux', dark: '#7FC0E3', light: '#0D597F' },
  flatpak: { label: 'Flatpak', subtitle: 'Linux Sandboxed', dark: '#4A90D9', light: '#4A90D9' },
  brew: { label: 'Homebrew', subtitle: 'macOS / Linux', dark: '#E2A843', light: '#A8700F' },
  scoop: { label: 'Scoop', subtitle: 'Windows', dark: '#F77FB5', light: '#D9488F' },
  nix: { label: 'Nix', subtitle: 'Nix Flakes', dark: '#7EBAE4', light: '#5277C3' },
};

const FLIP_IN_MS = 460;
const FLIP_OUT_MS = 340;
const DECK_STACK_SLICES = 7;
const DECK_STACK_MAX_LIFT = 14;
const DEAL_STAGGER_SEC = 0.062;
const RECALL_STAGGER_SEC = 0.03;
const RECALL_FLIGHT_SEC = 0.31;
const CHAIN_PAUSE_SEC = 0.09;
const TILT_MAX_Y = 12 * (Math.PI / 180);
const TILT_MAX_X = 10 * (Math.PI / 180);
const TILT_PERSPECTIVE_D = 850;
const FLAT_CARD_CORNERS = [
  -CARD_W / 2, -CARD_H / 2,
  CARD_W / 2, -CARD_H / 2,
  CARD_W / 2, CARD_H / 2,
  -CARD_W / 2, CARD_H / 2,
];

const SWIRL_PERIOD_SEC = 200 * Math.PI;
const SWIRL_STATIC_PHASE = 42.0;

function sharedSwirlPhase(reduced) {
  if (reduced) {
    return SWIRL_STATIC_PHASE;
  }
  return (Date.now() / 1000) % SWIRL_PERIOD_SEC;
}

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
  flipFloatWrap: document.getElementById('flip-float-wrap'),
  flipScene: document.getElementById('flip-scene'),
  flipCard: document.getElementById('flip-card'),
  flipFront: document.getElementById('flip-face-front'),
  flipBack: document.getElementById('flip-face-back'),
  themeButtons: Array.from(document.querySelectorAll('.theme-opt[data-theme-mode]')),
};

let svgMountSeq = 0;

const state = {
  data: null,
  themeMode: document.documentElement.dataset.themeMode || 'system',
  theme: document.documentElement.dataset.theme === 'latte' ? 'latte' : 'mocha',
  activeDeck: 'projects',
  modalOpen: false,
  modalPhase: 'closed', // 'closed' | 'flippingIn' | 'open' | 'flippingOut'
  modalAnimRaf: 0,
  mediaCleanup: null,
  activeCard: null,
  lastFocusedEl: null,
  openedByKeyboard: false,

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
  deckSliceTex: null,
  foilTexA: null,
  foilTexB: null,
  sparkleTex: null,

  // Live records
  deckRecords: [],
  cardRecords: [],
  recallingCards: [],
  pendingDeck: null,
  chainDelay: 0,
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
    deckW: 104,
    deckH: 146,
    deckGap: 28,
    cardScale: 1,
    cols: 4,
  },
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
  if (changed && state.modalPhase !== 'closed') {
    cancelCardModal(false);
  }
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

function buildFoilTexture(w, h, r, flipDiagonal) {
  const { canvas, ctx } = createCanvas(w, h);
  ctx.save();
  roundedRectPath(ctx, 0, 0, w, h, r);
  ctx.clip();

  const grad = flipDiagonal
    ? ctx.createLinearGradient(w, 0, 0, h)
    : ctx.createLinearGradient(0, 0, w, h);
  grad.addColorStop(0.0, 'rgba(245, 194, 231, 0.0)');
  grad.addColorStop(0.24, 'rgba(203, 166, 247, 0.24)');
  grad.addColorStop(0.40, 'rgba(137, 180, 250, 0.34)');
  grad.addColorStop(0.50, 'rgba(255, 255, 255, 0.42)');
  grad.addColorStop(0.60, 'rgba(148, 226, 213, 0.32)');
  grad.addColorStop(0.76, 'rgba(249, 226, 175, 0.20)');
  grad.addColorStop(1.0, 'rgba(245, 194, 231, 0.0)');
  ctx.fillStyle = grad;
  ctx.fillRect(0, 0, w, h);
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

function isLinuxMattersItem(deckKey, item) {
  if (!item) {
    return false;
  }
  const url = String(item.site_url || item.url || '').toLowerCase();
  if (deckKey === 'media' && item.kind === 'episode' && url.includes('linuxmatters')) {
    return true;
  }
  return deckKey === 'links' && (item.id === 'linux-matters' || url.includes('linuxmatters'));
}

function resolveCardAccent(deckKey, item, p) {
  if (deckKey === 'projects') {
    return item.language_color || p.mauve;
  }
  if (isLinuxMattersItem(deckKey, item)) {
    return p.name === 'latte' ? '#c98800' : '#feb81e';
  }
  return accentColor(p, item.accent);
}

function mediaHostLabel(item, fallback) {
  const raw = String((item && (item.site_url || item.url)) || '').trim();
  if (raw) {
    try {
      const host = new URL(raw).hostname.replace(/^www\./i, '');
      if (host) {
        return host;
      }
    } catch (_) {}
  }
  return fallback;
}

function extractCardTopTrumpsModel(deckKey, item, p) {
  const accent = resolveCardAccent(deckKey, item, p);

  if (deckKey === 'projects') {
    const rows = [
      {
        label: 'STARS',
        value: `★ ${item.stars_formatted || item.stars || 0}`,
        valueColor: p.text,
      },
      {
        label: 'UPDATED',
        value: item.release_age || '—',
        valueColor: item.fresh ? p.freshText : p.subtext,
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
    const icon = String(item.icon || '').toLowerCase();
    const isDiscord = icon === 'discord' || item.id === 'discord';
    const rows = [];

    if (Array.isArray(item.stats) && item.stats.length > 0) {
      for (const st of item.stats.slice(0, 2)) {
        rows.push({
          label: String(st.label || '').toUpperCase(),
          value: String(st.value || '—'),
          valueColor: st.highlight ? p.freshText : p.text,
        });
      }
    } else if (item.host_label) {
      rows.push({ label: 'SITE', value: item.host_label, valueColor: p.subtext });
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

  // Media deck (deduplicated 2-row layout so descriptions have room to breathe)
  const rows = [];
  if (item.kind === 'episode') {
    rows.push({ label: 'DURATION', value: item.duration || '—', valueColor: p.text });
    rows.push({ label: 'PUBLISHED', value: item.age || item.date_label || '—', valueColor: p.subtext });
  } else if (item.kind === 'podcast-hub') {
    const entries = item.entries || [];
    const latest = entries[0];
    rows.push({ label: 'LATEST', value: latest && latest.episode ? `#${latest.episode}` : (latest && latest.age) || '—', valueColor: p.text });
    rows.push({ label: 'LISTEN', value: `${(item.subscribe_links || []).length || 1} feeds`, valueColor: p.linkText });
  } else if (item.kind === 'post') {
    rows.push({ label: 'PUBLISHED', value: item.age || item.date_label || '—', valueColor: p.text });
    rows.push({ label: 'SITE', value: mediaHostLabel(item, 'blog.wimpys.world'), valueColor: p.subtext });
  } else if (item.kind === 'blog-hub') {
    const entries = item.entries || [];
    const latest = entries[0];
    rows.push({ label: 'LATEST', value: (latest && (latest.age || latest.date_label)) || '—', valueColor: p.text });
    rows.push({ label: 'SITE', value: mediaHostLabel(item, 'blog.wimpys.world'), valueColor: p.subtext });
  } else {
    rows.push({ label: 'PUBLISHED', value: item.age || item.date_label || '—', valueColor: p.text });
    rows.push({ label: 'SITE', value: 'YouTube', valueColor: p.subtext });
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

  // Compute bottom-anchored stat band first so the description uses every
  // available line inside the fixed 224 x 314 card geometry.
  const visibleRows = model.rows.slice(0, 2);
  const rowH = 25;
  const rowGap = 3;
  const rowW = CARD_W - padX * 2;
  const rowCount = visibleRows.length;
  const rowsBlockH = rowCount > 0 ? rowCount * rowH + (rowCount - 1) * rowGap : 0;
  const rowsStartY = rowCount > 0 ? 302 - rowsBlockH : 302;

  const isProjects = deckKey === 'projects';
  const descFontSize = isProjects ? 12.4 : 11.2;
  const descLineH = isProjects ? 16.5 : 15;
  ctx.font = `400 ${descFontSize}px ${FONT_SANS}`;
  ctx.fillStyle = p.subtext;
  const availDescH = Math.max(descLineH * 2, rowsStartY - 8 - cursorY);
  const maxDescLines = Math.max(2, Math.floor(availDescH / descLineH));
  const descLines = wrapLines(ctx, model.desc, textW, maxDescLines);
  for (const line of descLines) {
    ctx.fillText(line, padX, cursorY);
    cursorY += descLineH;
  }

  // 4. Bottom-anchored Top Trumps stat rows (1-2 rows ending at y = 302)
  for (let r = 0; r < rowCount; r += 1) {
    const row = visibleRows[r];
    const ry = rowsStartY + r * (rowH + rowGap);

    roundedRectPath(ctx, padX, ry, rowW, rowH, 6);
    ctx.fillStyle = r % 2 === 0 ? p.rowEven : p.rowOdd;
    ctx.fill();

    // Left uppercase label
    ctx.font = `600 9px ${FONT_MONO}`;
    ctx.fillStyle = p.muted;
    ctx.textAlign = 'left';
    ctx.textBaseline = 'middle';
    const labelStr = row.label;
    ctx.fillText(labelStr, padX + 8, ry + rowH / 2 + 0.5);
    const labelW = ctx.measureText(labelStr).width;

    // Right tabular value
    ctx.font = `600 10.5px ${FONT_MONO}`;
    ctx.fillStyle = row.valueColor || p.text;
    ctx.textAlign = 'right';
    const maxValW = Math.max(68, rowW - 22 - labelW);
    const valStr = truncateText(ctx, row.value, maxValW);
    ctx.fillText(valStr, padX + rowW - 8, ry + rowH / 2 + 0.5);
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
// Portrait 5:7 physical card-deck painters (placeholder, stack slice, card back)
// ---------------------------------------------------------------------------

function buildDeckSliceTexture(w, h, p) {
  const { canvas, ctx } = createCanvas(w, h);
  const radius = 11;
  roundedRectPath(ctx, 1, 1, w - 2, h - 2, radius);
  ctx.fillStyle = p.name === 'latte' ? p.surface0 : p.mantle;
  ctx.fill();
  ctx.strokeStyle = p.surface1;
  ctx.lineWidth = 1;
  ctx.stroke();

  // Subtle bottom card-edge highlight so stacked layers read clearly
  ctx.strokeStyle = p.name === 'latte' ? 'rgba(255, 255, 255, 0.55)' : 'rgba(205, 214, 244, 0.14)';
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(radius, h - 2.2);
  ctx.lineTo(w - radius, h - 2.2);
  ctx.stroke();

  return canvasToTexture(canvas);
}

function drawDeckPlaceholderCanvas(canvas, deck, totalCount, isActive, w, h, p) {
  const ctx = canvas.getContext('2d');
  ctx.save();
  ctx.setTransform(TEX_SCALE, 0, 0, TEX_SCALE, 0, 0);
  ctx.clearRect(0, 0, w, h);

  const acc = accentColor(p, deck.accent);
  const radius = 11;

  // Recessed tabletop card slot
  roundedRectPath(ctx, 1.5, 1.5, w - 3, h - 3, radius);
  ctx.fillStyle = p.name === 'latte' ? 'rgba(188, 192, 204, 0.34)' : 'rgba(17, 17, 27, 0.44)';
  ctx.fill();

  // Crisp dashed placeholder border
  ctx.save();
  ctx.setLineDash([6, 4]);
  ctx.strokeStyle = isActive ? acc : p.surface1;
  ctx.globalAlpha = isActive ? 0.85 : 0.65;
  ctx.lineWidth = isActive ? 1.8 : 1.3;
  ctx.stroke();
  ctx.restore();

  // Subtle corner registration pips
  ctx.fillStyle = isActive ? acc : p.muted;
  ctx.globalAlpha = isActive ? 0.72 : 0.4;
  ctx.font = `600 9px ${FONT_MONO}`;
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  ctx.fillText('◆', 9, 8);
  ctx.textAlign = 'right';
  ctx.textBaseline = 'bottom';
  ctx.fillText('◆', w - 9, h - 8);
  ctx.globalAlpha = 1;

  // Centred deck label + dealt indicator
  ctx.font = `700 ${w < 96 ? 12.5 : 14}px ${FONT_SANS}`;
  ctx.fillStyle = isActive ? p.text : p.subtext;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(deck.label, w / 2, h * 0.45);

  const subStr = `${totalCount} dealt`;
  ctx.font = `600 ${w < 96 ? 9 : 10}px ${FONT_MONO}`;
  const pillW = Math.min(w - 18, ctx.measureText(subStr).width + 14);
  const pillH = 18;
  const pillX = w / 2 - pillW / 2;
  const pillY = h * 0.65 - pillH / 2;
  roundedRectPath(ctx, pillX, pillY, pillW, pillH, 9);
  ctx.fillStyle = p.name === 'latte' ? 'rgba(239, 241, 245, 0.75)' : 'rgba(30, 30, 46, 0.72)';
  ctx.fill();
  ctx.strokeStyle = isActive ? acc : p.surface1;
  ctx.globalAlpha = isActive ? 0.65 : 0.5;
  ctx.lineWidth = 1;
  ctx.stroke();
  ctx.globalAlpha = 1;

  ctx.fillStyle = isActive ? acc : p.muted;
  ctx.fillText(subStr, w / 2, pillY + pillH / 2 + 0.5);

  ctx.restore();
}

function drawDeckTopCardCanvas(canvas, deck, isHighlighted, w, h, p) {
  const ctx = canvas.getContext('2d');
  ctx.save();
  ctx.setTransform(TEX_SCALE, 0, 0, TEX_SCALE, 0, 0);
  ctx.clearRect(0, 0, w, h);

  const acc = accentColor(p, deck.accent);
  const radius = 11;

  ctx.save();
  roundedRectPath(ctx, 1, 1, w - 2, h - 2, radius);
  ctx.fillStyle = isHighlighted ? p.base : p.mantle;
  ctx.fill();
  ctx.clip();

  // Balatro diagonal card-back diamond lattice
  ctx.strokeStyle = isHighlighted
    ? (p.name === 'latte' ? 'rgba(136, 57, 239, 0.13)' : 'rgba(203, 166, 247, 0.13)')
    : (p.name === 'latte' ? 'rgba(76, 79, 105, 0.07)' : 'rgba(205, 214, 244, 0.06)');
  ctx.lineWidth = 1;
  for (let x = -h; x < w + h; x += 12) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x + h, h);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(x + h, 0);
    ctx.lineTo(x, h);
    ctx.stroke();
  }

  // Top Catppuccin aurora bar
  const grad = ctx.createLinearGradient(0, 0, w, 0);
  grad.addColorStop(0, acc);
  grad.addColorStop(0.65, p.blue);
  grad.addColorStop(1, p.teal);
  ctx.fillStyle = grad;
  ctx.fillRect(0, 0, w, isHighlighted ? 4 : 3);

  // Ornate inner playing-card frame
  roundedRectPath(ctx, 6, 7.5, w - 12, h - 13.5, 7);
  ctx.strokeStyle = isHighlighted ? acc : p.surface1;
  ctx.globalAlpha = isHighlighted ? 0.55 : 0.72;
  ctx.lineWidth = 1;
  ctx.stroke();
  ctx.globalAlpha = 1;

  // Subtle corner card pips
  ctx.fillStyle = acc;
  ctx.globalAlpha = 0.78;
  ctx.font = `600 9px ${FONT_MONO}`;
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  ctx.fillText('◆', 10, 11);
  ctx.textAlign = 'right';
  ctx.textBaseline = 'bottom';
  ctx.fillText('◆', w - 10, h - 10);
  ctx.globalAlpha = 1;

  // Deck title
  ctx.font = `700 ${w < 96 ? 13 : 14.5}px ${FONT_SANS}`;
  ctx.fillStyle = p.text;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(deck.label, w / 2, h * 0.44);

  // Pill backing for dynamic PIXI.Text live count
  const pillW = w < 96 ? 50 : 58;
  const pillH = 20;
  const pillX = w / 2 - pillW / 2;
  const pillY = h * 0.65 - pillH / 2;
  roundedRectPath(ctx, pillX, pillY, pillW, pillH, 10);
  ctx.fillStyle = isHighlighted ? acc : p.base;
  ctx.fill();
  if (!isHighlighted) {
    ctx.strokeStyle = p.surface1;
    ctx.lineWidth = 1;
    ctx.stroke();
  }

  ctx.restore();

  // Outer crisp playing-card border
  roundedRectPath(ctx, 1, 1, w - 2, h - 2, radius);
  ctx.strokeStyle = isHighlighted ? acc : p.surface1;
  ctx.lineWidth = isHighlighted ? 2 : 1.5;
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
          uTime: { value: sharedSwirlPhase(isReducedMotion()), type: 'f32' },
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
  state.foilTexA = buildFoilTexture(CARD_W, CARD_H, CARD_RADIUS, false);
  state.foilTexB = buildFoilTexture(CARD_W, CARD_H, CARD_RADIUS, true);
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

function findDeckRecord(deckKey) {
  return state.deckRecords.find((d) => d.deck.key === deckKey) || null;
}

function effectiveDeckKey() {
  return state.pendingDeck || state.activeDeck;
}

function redrawDeckRecordTextures(drec, p) {
  const { deckW, deckH } = state.layout;
  const isSelected = effectiveDeckKey() === drec.deck.key;
  drawDeckPlaceholderCanvas(drec.placeholderCanvas, drec.deck, drec.totalCount, isSelected, deckW, deckH, p);
  drec.placeholderTex.source.update();
  drawDeckTopCardCanvas(drec.topCardCanvas, drec.deck, isSelected, deckW, deckH, p);
  drec.topCardTex.source.update();
  if (drec.countText) {
    drec.countText.style.fill = isSelected ? (p.name === 'latte' ? '#eff1f5' : '#11111b') : p.subtext;
  }
}

function syncDeckVisual(drec, p, dt, snap) {
  const ratio = drec.totalCount > 0
    ? Math.max(0, Math.min(1, drec.remainingCount / drec.totalCount))
    : 0;
  const targetLift = drec.remainingCount > 0
    ? Math.max(1.5, ratio * DECK_STACK_MAX_LIFT)
    : 0;
  const targetStackScale = drec.remainingCount > 0
    ? 0.93 + 0.07 * ratio
    : 0.93;

  if (snap) {
    drec.visualLift = targetLift;
    drec.stackScale = targetStackScale;
  } else {
    const step = Math.min(1, dt * 22);
    drec.visualLift += (targetLift - drec.visualLift) * step;
    drec.stackScale += (targetStackScale - drec.stackScale) * step;
  }

  const hasCards = drec.remainingCount > 0;
  const visibleSlices = drec.remainingCount > 1
    ? Math.max(1, Math.min(DECK_STACK_SLICES, Math.ceil(((drec.remainingCount - 1) / Math.max(1, drec.totalCount - 1)) * DECK_STACK_SLICES)))
    : 0;

  drec.stackGroup.visible = hasCards;
  for (let k = 0; k < drec.stackSlices.length; k += 1) {
    const sl = drec.stackSlices[k];
    const show = k < visibleSlices;
    sl.visible = show;
    if (show) {
      const frac = k / Math.max(1, visibleSlices);
      const liftY = -(frac * drec.visualLift);
      sl.position.set(0, liftY);
      const sliceScale = 0.93 + 0.07 * ((k + 1) / Math.max(1, DECK_STACK_SLICES)) * ratio;
      sl.scale.set(sliceScale);
    }
  }

  drec.topCardGroup.visible = hasCards;
  if (hasCards) {
    const catchScale = drec.catchBump > 0 ? Math.sin(drec.catchBump * Math.PI) * 0.05 : 0;
    const recoilY = drec.recoil > 0 ? Math.sin(drec.recoil * Math.PI) * 3.5 : 0;
    drec.topCardGroup.position.set(0, -drec.visualLift + recoilY);
    drec.topCardGroup.scale.set(drec.stackScale + catchScale);

    const countStr = String(drec.remainingCount);
    if (drec.lastRenderedCount !== countStr && drec.countText) {
      drec.countText.text = countStr;
      drec.lastRenderedCount = countStr;
    }
  }

  if (drec.shadowSprite) {
    drec.shadowSprite.alpha = hasCards ? p.shadowAlpha * 0.85 * (0.42 + 0.58 * ratio) : 0;
    drec.shadowSprite.y = 5 + drec.visualLift * 0.35;
  }
}

function refreshPixiForTheme() {
  const p = palette();
  updateBackgroundTheme();

  if (state.deckSliceTex) {
    state.deckSliceTex.destroy(true);
  }
  state.deckSliceTex = buildDeckSliceTexture(state.layout.deckW, state.layout.deckH, p);

  for (const drec of state.deckRecords) {
    for (const sl of drec.stackSlices) {
      sl.texture = state.deckSliceTex;
    }
    redrawDeckRecordTextures(drec, p);
    syncDeckVisual(drec, p, 0, true);
  }

  for (const rec of state.cardRecords) {
    rec.shadowSprite.alpha = p.shadowAlpha;
    if (rec.hoverOutline) {
      const corners = rec.isTilted ? projectCardCorners(rec.tiltX, rec.tiltY) : FLAT_CARD_CORNERS;
      drawCardHoverOutline(rec.hoverOutline, resolveCardAccent(rec.deckKey, rec.item, p), corners);
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
// Layout geometry (portrait 5:7 decks above floating cards, open tabletop)
// ---------------------------------------------------------------------------

function measureLayout() {
  const viewportW = window.innerWidth;
  const viewportH = window.innerHeight;
  const spacerRect = els.scrollSpacer ? els.scrollSpacer.getBoundingClientRect() : { top: 96 };
  const originY = Math.round(spacerRect.top + window.scrollY);

  const narrow = viewportW < 600;
  const deckGap = narrow ? 14 : 28;
  const deckW = narrow ? Math.max(78, Math.min(92, Math.floor((viewportW - 36 - deckGap * 2) / 3))) : 104;
  const deckH = Math.round(deckW * (CARD_H / CARD_W));

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

function buildDecks(preserveCounts) {
  if (!state.decksLayer) {
    return;
  }

  const prevCounts = new Map();
  if (preserveCounts) {
    for (const old of state.deckRecords) {
      prevCounts.set(old.deck.key, old.remainingCount);
    }
  }

  for (const old of state.deckRecords) {
    old.container.destroy({ children: true });
    old.placeholderTex.destroy(true);
    old.topCardTex.destroy(true);
  }
  state.deckRecords = [];
  if (state.deckShadowTex) {
    state.deckShadowTex.destroy(true);
  }
  if (state.deckSliceTex) {
    state.deckSliceTex.destroy(true);
  }

  const p = palette();
  const { viewportW, originY, deckW, deckH, deckGap } = state.layout;
  state.deckShadowTex = buildSoftShadowTexture(deckW, deckH, 11, 16);
  state.deckSliceTex = buildDeckSliceTexture(deckW, deckH, p);

  const totalW = DECKS.length * deckW + (DECKS.length - 1) * deckGap;
  const startX = (viewportW - totalW) / 2;
  const centerY = originY + DECK_STACK_MAX_LIFT + 8 + deckH / 2;

  if (els.a11yDecks) {
    els.a11yDecks.textContent = '';
  }

  const selectedKey = effectiveDeckKey();

  DECKS.forEach((deck, idx) => {
    const totalCount = deckItems(deck.key).length;
    const isSelected = selectedKey === deck.key;
    const remainingCount = prevCounts.has(deck.key) ? prevCounts.get(deck.key) : totalCount;

    const { canvas: placeholderCanvas } = createCanvas(deckW, deckH);
    drawDeckPlaceholderCanvas(placeholderCanvas, deck, totalCount, isSelected, deckW, deckH, p);
    const placeholderTex = canvasToTexture(placeholderCanvas);

    const { canvas: topCardCanvas } = createCanvas(deckW, deckH);
    drawDeckTopCardCanvas(topCardCanvas, deck, isSelected, deckW, deckH, p);
    const topCardTex = canvasToTexture(topCardCanvas);

    const container = new PIXI.Container();
    const homeX = startX + idx * (deckW + deckGap) + deckW / 2;
    const homeY = centerY;
    container.position.set(homeX, homeY);

    // 1. Recessed dashed placeholder mat always sits at deck origin
    const placeholderSprite = new PIXI.Sprite(placeholderTex);
    placeholderSprite.anchor.set(0.5);
    placeholderSprite.width = deckW;
    placeholderSprite.height = deckH;
    container.addChild(placeholderSprite);

    // 2. Dynamic drop shadow underneath physical stack
    const shadowSprite = new PIXI.Sprite(state.deckShadowTex);
    shadowSprite.anchor.set(0.5);
    shadowSprite.position.set(0, 6);
    shadowSprite.alpha = p.shadowAlpha * 0.85;
    container.addChild(shadowSprite);

    // 3. Layered card-edge stack group (thins card-by-card as deck deals out)
    const stackGroup = new PIXI.Container();
    const stackSlices = [];
    for (let s = 0; s < DECK_STACK_SLICES; s += 1) {
      const sl = new PIXI.Sprite(state.deckSliceTex);
      sl.anchor.set(0.5);
      sl.width = deckW;
      sl.height = deckH;
      stackGroup.addChild(sl);
      stackSlices.push(sl);
    }
    container.addChild(stackGroup);

    // 4. Top card-back sprite + live card counter
    const topCardGroup = new PIXI.Container();
    const topCardSprite = new PIXI.Sprite(topCardTex);
    topCardSprite.anchor.set(0.5);
    topCardSprite.width = deckW;
    topCardSprite.height = deckH;
    topCardGroup.addChild(topCardSprite);

    const countText = new PIXI.Text({
      text: String(remainingCount),
      style: {
        fontFamily: FONT_MONO,
        fontSize: deckW < 96 ? 10 : 11,
        fontWeight: '600',
        fill: isSelected ? (p.name === 'latte' ? '#eff1f5' : '#11111b') : p.subtext,
      },
    });
    countText.anchor.set(0.5);
    countText.position.set(0, Math.round(deckH * 0.15) + 0.5);
    countText.resolution = Math.min(window.devicePixelRatio || 1, 2);
    topCardGroup.addChild(countText);
    container.addChild(topCardGroup);

    state.decksLayer.addChild(container);

    const btn = el('button', 'a11y-deck-btn', `${deck.label} deck, ${totalCount} ${totalCount === 1 ? 'card' : 'cards'}`);
    btn.type = 'button';
    btn.dataset.deck = deck.key;
    btn.setAttribute('aria-pressed', String(isSelected));
    btn.setAttribute('aria-label', `${deck.label} deck (${totalCount} ${totalCount === 1 ? 'card' : 'cards'})`);
    btn.style.left = `${Math.round(homeX - deckW / 2)}px`;
    btn.style.top = `${Math.round(homeY - originY - deckH / 2 - DECK_STACK_MAX_LIFT / 2)}px`;
    btn.style.width = `${Math.round(deckW)}px`;
    btn.style.height = `${Math.round(deckH + DECK_STACK_MAX_LIFT / 2)}px`;

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

    const drec = {
      deck,
      totalCount,
      remainingCount,
      lastRenderedCount: String(remainingCount),
      visualLift: 0,
      stackScale: 1,
      placeholderCanvas,
      placeholderTex,
      placeholderSprite,
      topCardCanvas,
      topCardTex,
      topCardGroup,
      topCardSprite,
      countText,
      stackGroup,
      stackSlices,
      container,
      shadowSprite,
      btn,
      homeX,
      homeY,
      scale: 1,
      pressBounce: 0,
      recoil: 0,
      catchBump: 0,
      rot: (idx - 1) * 0.022,
    };
    syncDeckVisual(drec, p, 0, true);
    state.deckRecords.push(drec);
  });
}

function projectCardCorners(rotX, rotY) {
  const hw = CARD_W / 2;
  const hh = CARD_H / 2;
  const cosY = Math.cos(rotY);
  const sinY = Math.sin(rotY);
  const cosX = Math.cos(rotX);
  const sinX = Math.sin(rotX);
  const d = TILT_PERSPECTIVE_D;

  const raw = [
    [-hw, -hh],
    [hw, -hh],
    [hw, hh],
    [-hw, hh],
  ];
  const out = new Array(8);
  for (let i = 0; i < 4; i += 1) {
    const x0 = raw[i][0];
    const y0 = raw[i][1];
    const x1 = x0 * cosY;
    const z1 = -x0 * sinY;
    const y2 = y0 * cosX - z1 * sinX;
    const z2 = y0 * sinX + z1 * cosX;
    const scale = d / (d - z2);
    out[i * 2] = x1 * scale;
    out[i * 2 + 1] = y2 * scale;
  }
  return out;
}

function drawCardHoverOutline(graphics, accentHex, corners) {
  graphics.clear();
  if (!corners || corners === FLAT_CARD_CORNERS) {
    graphics.roundRect(-CARD_W / 2 - 1, -CARD_H / 2 - 1, CARD_W + 2, CARD_H + 2, CARD_RADIUS + 1);
    graphics.stroke({ width: 2.25, color: hexToNumber(accentHex), alpha: 0.88 });
    return;
  }

  // Trace the 4 perspective-projected corners with rounded bevels matching CARD_RADIUS
  const pts = [
    [corners[0] * 1.008, corners[1] * 1.006],
    [corners[2] * 1.008, corners[3] * 1.006],
    [corners[4] * 1.008, corners[5] * 1.006],
    [corners[6] * 1.008, corners[7] * 1.006],
  ];
  const r = CARD_RADIUS + 1;
  for (let i = 0; i < 4; i += 1) {
    const prev = pts[(i + 3) % 4];
    const curr = pts[i];
    const next = pts[(i + 1) % 4];
    const dx1 = prev[0] - curr[0];
    const dy1 = prev[1] - curr[1];
    const len1 = Math.hypot(dx1, dy1) || 1;
    const dx2 = next[0] - curr[0];
    const dy2 = next[1] - curr[1];
    const len2 = Math.hypot(dx2, dy2) || 1;
    const off1 = Math.min(r, len1 * 0.3) / len1;
    const off2 = Math.min(r, len2 * 0.3) / len2;
    const sx = curr[0] + dx1 * off1;
    const sy = curr[1] + dy1 * off1;
    const ex = curr[0] + dx2 * off2;
    const ey = curr[1] + dy2 * off2;
    if (i === 0) {
      graphics.moveTo(sx, sy);
    } else {
      graphics.lineTo(sx, sy);
    }
    graphics.quadraticCurveTo(curr[0], curr[1], ex, ey);
  }
  graphics.closePath();
  graphics.stroke({ width: 2.25, color: hexToNumber(accentHex), alpha: 0.88 });
}

function applyCardPerspective(rec, rotX, rotY, accentHex) {
  const isFlat = rotX === 0 && rotY === 0;
  const c = isFlat ? FLAT_CARD_CORNERS : projectCardCorners(rotX, rotY);
  rec.cardMesh.setCorners(c[0], c[1], c[2], c[3], c[4], c[5], c[6], c[7]);
  rec.foilMeshA.setCorners(c[0], c[1], c[2], c[3], c[4], c[5], c[6], c[7]);
  rec.foilMeshB.setCorners(c[0], c[1], c[2], c[3], c[4], c[5], c[6], c[7]);
  if (isFlat || rec.hoverOutline.alpha > 0.01) {
    drawCardHoverOutline(rec.hoverOutline, accentHex, c);
  }
}

function computeCardSlots(count) {
  const { viewportW, originY, deckH, effW, effH, cardGapX, cardGapY, cols } = state.layout;
  const effCols = Math.max(1, Math.min(cols, count));
  const rows = Math.ceil(count / effCols);
  const cardsTopY = originY + DECK_STACK_MAX_LIFT + 8 + deckH + 30;

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

function destroyCardRecord(rec) {
  if (!rec || rec.destroyed) {
    return;
  }
  rec.destroyed = true;
  rec.container.destroy({ children: true });
  rec.texture.destroy(true);
}

function flushRecallingCards() {
  for (const rec of state.recallingCards) {
    const homeDeck = findDeckRecord(rec.homeDeckKey || rec.deckKey);
    if (homeDeck) {
      homeDeck.remainingCount = Math.min(homeDeck.totalCount, homeDeck.remainingCount + 1);
    }
    destroyCardRecord(rec);
  }
  state.recallingCards = [];
}

function recallCurrentCards() {
  const reduced = isReducedMotion();
  if (els.a11yCards) {
    els.a11yCards.textContent = '';
  }

  if (reduced) {
    for (const rec of state.cardRecords) {
      destroyCardRecord(rec);
    }
    state.cardRecords = [];
    flushRecallingCards();
    for (const drec of state.deckRecords) {
      drec.remainingCount = drec.totalCount;
    }
    state.hoveredCardId = null;
    state.focusedCardId = null;
    return false;
  }

  // Reclaim nearest cards first so front rows clear a clean path for rows behind them.
  const launched = [];
  for (const rec of state.cardRecords) {
    if (!rec.launched) {
      // Never left its home deck; discard immediately and keep it counted in the stack.
      destroyCardRecord(rec);
      continue;
    }
    const homeDeck = findDeckRecord(rec.deckKey);
    const tx = homeDeck ? homeDeck.homeX : state.layout.viewportW / 2;
    const ty = homeDeck ? homeDeck.homeY : state.layout.originY + 90;
    launched.push({
      rec,
      dist: Math.hypot(rec.x - tx, rec.y - ty),
    });
  }
  launched.sort((a, b) => a.dist - b.dist || a.rec.index - b.rec.index);

  for (let queued = 0; queued < launched.length; queued += 1) {
    const { rec } = launched[queued];
    const homeDeck = findDeckRecord(rec.deckKey);
    const targetX = homeDeck ? homeDeck.homeX : state.layout.viewportW / 2;
    const dx = targetX - rec.x;
    const leanDir = dx === 0 ? (queued % 2 === 0 ? -1 : 1) : (dx > 0 ? 1 : -1);
    const bankDeg = 22 + Math.abs(seeded(rec.index + 13)) * 13;

    rec.recalling = true;
    rec.homeDeckKey = rec.deckKey;
    rec.recallFromX = rec.x;
    rec.recallFromY = rec.y;
    rec.recallFromRot = rec.rot;
    rec.recallFromScaleX = rec.scaleX;
    rec.recallFromScaleY = rec.scaleY;
    rec.recallDelay = queued * RECALL_STAGGER_SEC;
    rec.recallElapsed = 0;
    rec.recallDur = RECALL_FLIGHT_SEC;
    rec.recallSpin = leanDir * bankDeg * (Math.PI / 180);
    rec.recallArcX = leanDir * (18 + Math.abs(seeded(rec.index + 19)) * 24);
    rec.hoverOutline.alpha = 0;
    rec.foilMeshA.alpha = 0;
    rec.foilMeshB.alpha = 0;
    rec.tiltX = 0;
    rec.tiltY = 0;
    if (rec.isTilted) {
      applyCardPerspective(rec, 0, 0, rec.accent);
      rec.isTilted = false;
    }
    rec.container.zIndex = 140 + (launched.length - queued);
    state.recallingCards.push(rec);
  }

  state.cardRecords = [];
  state.hoveredCardId = null;
  state.focusedCardId = null;
  if (launched.length > 0) {
    state.chainDelay = CHAIN_PAUSE_SEC;
  }
  return launched.length > 0;
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

function spawnDeckArrivalSparkles(x, y, accentHex) {
  if (!state.particlesLayer || !state.sparkleTex || isReducedMotion()) {
    return;
  }
  const color = hexToNumber(accentHex);
  const count = 4;
  for (let i = 0; i < count; i += 1) {
    const sp = new PIXI.Sprite(state.sparkleTex);
    sp.anchor.set(0.5);
    sp.tint = color;
    const angle = (Math.PI * 2 * i) / count + seeded(x + y + i * 5) * 0.9;
    const dist = 24 + Math.abs(seeded(y + i * 7)) * 28;
    sp.position.set(x + Math.cos(angle) * dist, y + Math.sin(angle) * dist * 1.15);
    sp.scale.set(0.5 + Math.abs(seeded(i + 11)) * 0.45);
    sp.alpha = 0.95;
    state.particlesLayer.addChild(sp);

    state.particles.push({
      sprite: sp,
      vx: Math.cos(angle) * (55 + Math.abs(seeded(i + 13)) * 45),
      vy: Math.sin(angle) * (55 + Math.abs(seeded(i + 17)) * 45) - 24,
      vRot: (i % 2 === 0 ? 1 : -1) * 4.6,
      life: 0.34,
      maxLife: 0.34,
    });
  }
}

function dealDeck(deckKey) {
  if (!state.cardsLayer) {
    return;
  }

  for (const old of state.cardRecords) {
    destroyCardRecord(old);
  }
  state.cardRecords = [];
  state.hoveredCardId = null;
  state.focusedCardId = null;

  const items = deckItems(deckKey);
  const p = palette();
  const reduced = isReducedMotion();
  const { originY, cardScale, effW, effH, deckW } = state.layout;
  const slots = computeCardSlots(items.length);
  const homeDeck = findDeckRecord(deckKey);
  const deckScale = Math.max(0.32, (deckW / CARD_W) * cardScale);

  for (const drec of state.deckRecords) {
    if (drec.deck.key === deckKey) {
      drec.remainingCount = reduced ? 0 : items.length;
    } else {
      drec.remainingCount = drec.totalCount;
    }
    redrawDeckRecordTextures(drec, p);
    syncDeckVisual(drec, p, 0, true);
  }

  if (els.a11yCards) {
    els.a11yCards.textContent = '';
  }

  const originDeckX = homeDeck ? homeDeck.homeX : state.layout.viewportW / 2;
  const originDeckY = homeDeck ? homeDeck.homeY - homeDeck.visualLift : originY + 90;
  const originDeckRot = homeDeck ? homeDeck.rot : 0;

  // Deal farthest destinations first so later cards never cross over settled foreground cards.
  const dealOrderByIdx = new Array(items.length);
  items
    .map((_, idx) => ({
      idx,
      dist: Math.hypot(slots[idx].x - originDeckX, slots[idx].y - originDeckY),
    }))
    .sort((a, b) => b.dist - a.dist || b.idx - a.idx)
    .forEach((entry, order) => {
      dealOrderByIdx[entry.idx] = order;
    });

  items.forEach((item, idx) => {
    const slot = slots[idx];
    const dealOrder = dealOrderByIdx[idx];
    const cardId = `${deckKey}:${item.id || idx}`;
    const accent = resolveCardAccent(deckKey, item, p);
    const { canvas } = createCanvas(CARD_W, CARD_H);
    drawCardFaceToCanvas(canvas, deckKey, item, p, null);
    const texture = canvasToTexture(canvas);

    const container = new PIXI.Container();
    container.zIndex = reduced ? 20 + idx : 120 + dealOrder;

    const shadowSprite = new PIXI.Sprite(state.shadowTex);
    shadowSprite.anchor.set(0.5);
    shadowSprite.position.set(0, 10);
    shadowSprite.alpha = p.shadowAlpha;
    container.addChild(shadowSprite);

    const hw = CARD_W / 2;
    const hh = CARD_H / 2;

    const cardMesh = new PIXI.PerspectiveMesh({
      texture,
      verticesX: 12,
      verticesY: 16,
      x0: -hw,
      y0: -hh,
      x1: hw,
      y1: -hh,
      x2: hw,
      y2: hh,
      x3: -hw,
      y3: hh,
    });
    container.addChild(cardMesh);

    const foilMeshA = new PIXI.PerspectiveMesh({
      texture: state.foilTexA,
      verticesX: 8,
      verticesY: 10,
      x0: -hw,
      y0: -hh,
      x1: hw,
      y1: -hh,
      x2: hw,
      y2: hh,
      x3: -hw,
      y3: hh,
    });
    foilMeshA.alpha = 0;
    foilMeshA.blendMode = 'screen';
    container.addChild(foilMeshA);

    const foilMeshB = new PIXI.PerspectiveMesh({
      texture: state.foilTexB,
      verticesX: 8,
      verticesY: 10,
      x0: -hw,
      y0: -hh,
      x1: hw,
      y1: -hh,
      x2: hw,
      y2: hh,
      x3: -hw,
      y3: hh,
    });
    foilMeshB.alpha = 0;
    foilMeshB.blendMode = 'screen';
    container.addChild(foilMeshB);

    const hoverOutline = new PIXI.Graphics();
    drawCardHoverOutline(hoverOutline, accent, FLAT_CARD_CORNERS);
    hoverOutline.alpha = 0;
    container.addChild(hoverOutline);

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
      cardMesh,
      foilMeshA,
      foilMeshB,
      hoverOutline,
      btn: null,
      destroyed: false,

      homeX: slot.x,
      homeY: slot.y,
      baseRot: slot.baseRot,

      x: reduced ? slot.x : originDeckX,
      y: reduced ? slot.y : originDeckY,
      vx: 0,
      vy: 0,
      rot: reduced ? slot.baseRot : originDeckRot,
      vRot: 0,
      scaleX: reduced ? cardScale : deckScale,
      scaleY: reduced ? cardScale : deckScale,
      tiltX: 0,
      tiltY: 0,
      isTilted: false,

      dealOrder,
      dealDelay: reduced ? 0 : dealOrder * DEAL_STAGGER_SEC,
      launched: reduced,
      dealt: reduced,
      sparkled: reduced,
      pickedUp: false,

      phaseX: idx * 1.13 + 0.5,
      phaseY: idx * 0.89 + 1.7,
      phaseRot: idx * 1.41,
      freqX: 0.65 + (idx % 5) * 0.08,
      freqY: 0.78 + (idx % 4) * 0.09,
    };

    container.position.set(rec.x, rec.y);
    container.rotation = rec.rot;
    container.scale.set(rec.scaleX, rec.scaleY);
    container.visible = reduced;
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
      try {
        if (btn.matches(':focus-visible')) {
          state.focusedCardId = cardId;
        }
      } catch (_) {
        state.focusedCardId = cardId;
      }
    });
    btn.addEventListener('blur', () => {
      if (state.focusedCardId === cardId) {
        state.focusedCardId = null;
      }
    });
    btn.addEventListener('click', (ev) => {
      const byKeyboard = ev.detail === 0;
      triggerCardOpen(rec, btn, byKeyboard);
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

function syncNavDeckState(deckKey) {
  const navLinks = document.querySelectorAll('.site-nav [data-deck-nav]');
  for (const link of navLinks) {
    if (link.dataset.deckNav === deckKey) {
      link.setAttribute('aria-current', 'page');
    } else {
      link.removeAttribute('aria-current');
    }
  }
}

function updateDeckSelectionStyles() {
  const p = palette();
  const selectedKey = effectiveDeckKey();
  syncNavDeckState(selectedKey);
  for (const drec of state.deckRecords) {
    const isSelected = drec.deck.key === selectedKey;
    redrawDeckRecordTextures(drec, p);
    if (drec.btn) {
      drec.btn.setAttribute('aria-pressed', String(isSelected));
    }
  }
}

function normalizeDeckKey(raw) {
  const clean = String(raw || '').toLowerCase().replace(/^#/, '').trim();
  return DECKS.some((d) => d.key === clean) ? clean : '';
}

function resolveInitialDeck(data) {
  try {
    const params = new URLSearchParams(window.location.search);
    const fromQuery = normalizeDeckKey(params.get('deck'));
    if (fromQuery) {
      return fromQuery;
    }
  } catch (_) {
    // Ignore malformed query string.
  }

  const fromHash = normalizeDeckKey(window.location.hash);
  if (fromHash) {
    return fromHash;
  }

  const host = String(window.location.hostname || '').toLowerCase();
  if (host === 'links.wimpys.world') {
    return 'links';
  }

  return normalizeDeckKey(data && data.default_deck) || 'projects';
}

function syncDeckHash(deckKey) {
  try {
    const targetHash = `#${deckKey}`;
    if (window.location.hash !== targetHash && window.history && typeof window.history.replaceState === 'function') {
      const url = new URL(window.location.href);
      url.hash = targetHash;
      url.searchParams.delete('deck');
      window.history.replaceState(null, '', url.pathname + url.search + url.hash);
    }
  } catch (_) {
    // Ignore history API restrictions in opaque origins.
  }
}

function switchDeck(deckKey, opts) {
  if (!DECKS.some((d) => d.key === deckKey)) {
    return;
  }
  if (!opts || opts.syncHash !== false) {
    syncDeckHash(deckKey);
  }
  if (state.modalPhase !== 'closed') {
    cancelCardModal(false);
  }

  const targetDeck = findDeckRecord(deckKey);
  if (targetDeck && !isReducedMotion()) {
    targetDeck.pressBounce = 1;
  }

  // Clicking the already-active dealt deck while idle is a calm no-op.
  if (deckKey === state.activeDeck && state.recallingCards.length === 0 && !state.pendingDeck && state.cardRecords.length > 0) {
    return;
  }

  const wasSame = effectiveDeckKey() === deckKey;
  state.pendingDeck = deckKey;
  updateDeckSelectionStyles();

  if (isReducedMotion()) {
    recallCurrentCards();
    state.activeDeck = deckKey;
    state.pendingDeck = null;
    state.chainDelay = 0;
    dealDeck(deckKey);
    if (!wasSame) {
      const meta = DECKS.find((d) => d.key === deckKey);
      announce(`Dealt ${meta ? meta.label : deckKey} deck, ${deckItems(deckKey).length} cards.`);
    }
    return;
  }

  // If cards are currently on the table, vacuum them back into their home deck first.
  if (state.cardRecords.length > 0) {
    recallCurrentCards();
  }

  // If nothing is recalling, deal immediately.
  if (state.recallingCards.length === 0) {
    state.activeDeck = deckKey;
    state.pendingDeck = null;
    state.chainDelay = 0;
    dealDeck(deckKey);
  }

  if (!wasSame) {
    const meta = DECKS.find((d) => d.key === deckKey);
    announce(`Dealt ${meta ? meta.label : deckKey} deck, ${deckItems(deckKey).length} cards.`);
  }
}

function relayoutScene() {
  if (!state.app) {
    return;
  }
  if (state.modalPhase !== 'closed') {
    cancelCardModal(false);
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

  // Complete any in-flight recall cleanly on viewport resize so cards stay aligned.
  if (state.recallingCards.length > 0) {
    flushRecallingCards();
  }
  const queued = state.pendingDeck;
  state.pendingDeck = null;
  state.chainDelay = 0;

  measureLayout();
  buildDecks(true);

  if (queued) {
    state.activeDeck = queued;
    dealDeck(queued);
    return;
  }

  const { originY, cardScale, effW, effH } = state.layout;
  const slots = computeCardSlots(state.cardRecords.length);

  state.cardRecords.forEach((rec, idx) => {
    const slot = slots[idx];
    rec.homeX = slot.x;
    rec.homeY = slot.y;
    rec.baseRot = slot.baseRot;
    if (rec.launched) {
      rec.x = slot.x;
      rec.y = slot.y;
      rec.vx = 0;
      rec.vy = 0;
      rec.rot = slot.baseRot;
      rec.vRot = 0;
      rec.scaleX = cardScale;
      rec.scaleY = cardScale;
      rec.dealt = true;
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
// Frame ticker: shader swirl, deck stack physics, recall & sequential deal
// ---------------------------------------------------------------------------

function onTick(ticker) {
  const dt = Math.min(0.05, Math.max(0.001, ticker.deltaMS / 1000));
  const reduced = isReducedMotion();
  const p = palette();

  // Keep PixiJS world 1:1 locked to native page scroll
  if (state.world) {
    state.world.y = -window.scrollY;
  }

  // Sync swirling Balatro background shader to the shared cross-site clock
  if (state.bgFilter && state.bgFilter.resources && state.bgFilter.resources.balatroUniforms) {
    if (reduced) {
      state.bgFilter.resources.balatroUniforms.uniforms.uTime = SWIRL_STATIC_PHASE;
    } else if (!state.modalOpen) {
      state.bgFilter.resources.balatroUniforms.uniforms.uTime = sharedSwirlPhase(false);
    }
  }

  const t = performance.now() / 1000;
  const { cardScale, deckW } = state.layout;
  const deckScale = Math.max(0.32, (deckW / CARD_W) * cardScale);
  const selectedKey = effectiveDeckKey();

  // 1. Animate 3 physical playing-card decks + live stack thickness
  for (let i = 0; i < state.deckRecords.length; i += 1) {
    const drec = state.deckRecords[i];
    const isHovered = state.hoveredDeckKey === drec.deck.key;
    const isSelected = selectedKey === drec.deck.key;

    if (drec.pressBounce > 0) {
      drec.pressBounce = Math.max(0, drec.pressBounce - dt * 5.5);
    }
    if (drec.recoil > 0) {
      drec.recoil = Math.max(0, drec.recoil - dt * 9.5);
    }
    if (drec.catchBump > 0) {
      drec.catchBump = Math.max(0, drec.catchBump - dt * 9.0);
    }

    syncDeckVisual(drec, p, dt, reduced);

    const bounceOffset = reduced ? 0 : Math.sin(drec.pressBounce * Math.PI) * 4;
    const hoverLift = drec.remainingCount > 0 ? -4 : -2;
    const targetScale = reduced ? 1 : (isHovered ? 1.045 : (isSelected ? 1.015 : 1)) - drec.pressBounce * 0.035;
    const targetY = drec.homeY + (reduced ? 0 : (isHovered ? hoverLift : 0) + bounceOffset);
    const targetRot = reduced ? 0 : (isHovered ? (i - 1) * 0.035 : (i - 1) * 0.018);

    drec.scale += (targetScale - drec.scale) * Math.min(1, dt * 14);
    drec.container.y += (targetY - drec.container.y) * Math.min(1, dt * 14);
    drec.container.rotation += (targetRot - drec.container.rotation) * Math.min(1, dt * 14);
    drec.container.scale.set(drec.scale);
  }

  // 2. Fast magnetic recall: suck cards back into their home deck
  for (let i = state.recallingCards.length - 1; i >= 0; i -= 1) {
    const rec = state.recallingCards[i];
    if (rec.recallDelay > 0) {
      rec.recallDelay -= dt;
      continue;
    }

    rec.recallElapsed += dt;
    const u = Math.min(1, rec.recallElapsed / rec.recallDur);
    // Accelerating magnetic pull into the home deck
    const e = Math.pow(u, 1.55);

    const homeDeck = findDeckRecord(rec.homeDeckKey || rec.deckKey);
    const targetX = homeDeck ? homeDeck.homeX : state.layout.viewportW / 2;
    const targetY = homeDeck ? homeDeck.homeY - homeDeck.visualLift : state.layout.originY + 90;
    const targetRot = homeDeck ? homeDeck.rot : 0;

    const arc = Math.sin(u * Math.PI) * rec.recallArcX;
    rec.x = rec.recallFromX + (targetX - rec.recallFromX) * e + arc;
    rec.y = rec.recallFromY + (targetY - rec.recallFromY) * e;
    rec.rot = rec.recallFromRot + (targetRot + rec.recallSpin - rec.recallFromRot) * e;
    rec.scaleX = rec.recallFromScaleX + (deckScale - rec.recallFromScaleX) * e;
    rec.scaleY = rec.recallFromScaleY + (deckScale - rec.recallFromScaleY) * e;

    rec.container.position.set(rec.x, rec.y);
    rec.container.rotation = rec.rot;
    rec.container.scale.set(rec.scaleX, rec.scaleY);
    rec.container.alpha = u > 0.88 ? Math.max(0.25, 1 - (u - 0.88) / 0.16) : 1;

    if (u >= 1) {
      if (homeDeck) {
        homeDeck.remainingCount = Math.min(homeDeck.totalCount, homeDeck.remainingCount + 1);
        homeDeck.catchBump = 1;
        spawnDeckArrivalSparkles(homeDeck.homeX, homeDeck.homeY - homeDeck.visualLift, rec.accent);
      }
      destroyCardRecord(rec);
      state.recallingCards.splice(i, 1);
    }
  }

  // Chain the newly requested deck deal once all recalled cards land in their home deck
  if (state.recallingCards.length === 0 && state.pendingDeck) {
    if (state.chainDelay > 0) {
      state.chainDelay -= dt;
    } else {
      const nextDeck = state.pendingDeck;
      state.pendingDeck = null;
      state.activeDeck = nextDeck;
      dealDeck(nextDeck);
    }
  }

  // 3. Update sparkle particles
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

  // 4. Sequential deal-out from home deck & idle float
  for (let i = 0; i < state.cardRecords.length; i += 1) {
    const rec = state.cardRecords[i];
    const isHovered = !state.modalOpen && !rec.pickedUp && rec.dealt && (state.hoveredCardId === rec.id || state.focusedCardId === rec.id);

    if (reduced) {
      rec.x = rec.homeX;
      rec.y = rec.homeY;
      rec.rot = rec.baseRot;
      rec.scaleX = isHovered ? cardScale * 1.03 : cardScale;
      rec.scaleY = rec.scaleX;
      rec.tiltX = 0;
      rec.tiltY = 0;
      if (rec.isTilted) {
        applyCardPerspective(rec, 0, 0, resolveCardAccent(rec.deckKey, rec.item, p));
        rec.isTilted = false;
      }
      rec.container.visible = !rec.pickedUp;
      rec.container.position.set(rec.x, rec.y);
      rec.container.rotation = rec.rot;
      rec.container.scale.set(rec.scaleX, rec.scaleY);
      rec.container.zIndex = isHovered ? 200 : 20 + i;
      rec.hoverOutline.alpha = isHovered ? 1 : 0;
      rec.foilMeshA.alpha = 0;
      rec.foilMeshB.alpha = 0;
      continue;
    }

    // Sequential deck launch stagger
    if (!rec.launched) {
      if (rec.dealDelay > 0) {
        rec.dealDelay -= dt;
        continue;
      }
      rec.launched = true;
      const homeDeck = findDeckRecord(rec.deckKey);
      const launchX = homeDeck ? homeDeck.homeX : state.layout.viewportW / 2;
      const launchY = homeDeck ? homeDeck.homeY - homeDeck.visualLift : state.layout.originY + 90;
      if (homeDeck) {
        homeDeck.remainingCount = Math.max(0, homeDeck.remainingCount - 1);
        homeDeck.recoil = 1;
      }

      const dx = rec.homeX - launchX;
      const flickDir = dx === 0 ? (i % 2 === 0 ? 1 : -1) : (dx > 0 ? 1 : -1);
      const flickDeg = 28 + Math.abs(seeded(i + 11)) * 14;
      rec.x = launchX;
      rec.y = launchY;
      // Gentle dealer wrist flick angled along the card's travel vector
      rec.vx = dx * 0.55 + flickDir * 60;
      rec.vy = -120;
      rec.rot = flickDir * flickDeg * (Math.PI / 180);
      rec.vRot = 0;
      rec.scaleX = deckScale;
      rec.scaleY = deckScale;
      rec.container.position.set(rec.x, rec.y);
      rec.container.rotation = rec.rot;
      rec.container.scale.set(rec.scaleX, rec.scaleY);
      rec.container.visible = true;
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
    const targetRot = isHovered ? nx * 0.015 : idleRot;

    // Smooth unhurried spring glide for deal-in & gentle float
    const posK = rec.dealt ? (isHovered ? 150 : 62) : 115;
    const posDamp = rec.dealt ? (isHovered ? 18 : 11) : 13.5;
    const rotK = rec.dealt ? 95 : 105;
    const rotDamp = rec.dealt ? 13 : 13.0;

    rec.vx += ((targetX - rec.x) * posK - rec.vx * posDamp) * dt;
    rec.vy += ((targetY - rec.y) * posK - rec.vy * posDamp) * dt;
    rec.x += rec.vx * dt;
    rec.y += rec.vy * dt;

    rec.vRot += ((targetRot - rec.rot) * rotK - rec.vRot * rotDamp) * dt;
    rec.rot += rec.vRot * dt;

    // Mark dealt + burst subtle sparkles when card arrives at its destination
    if (!rec.dealt && Math.hypot(targetX - rec.x, targetY - rec.y) < 18 && Math.hypot(rec.vx, rec.vy) < 190) {
      rec.dealt = true;
      if (!rec.sparkled) {
        rec.sparkled = true;
        spawnDealSparkles(rec.homeX, rec.homeY, rec.accent);
      }
    }

    // Smooth scale-up from deck size during deal + true 3D perspective hover tilt
    const targetScale = cardScale * (isHovered ? 1.06 : 1);
    const targetTiltY = isHovered ? nx * TILT_MAX_Y : 0;
    const targetTiltX = isHovered ? -ny * TILT_MAX_X : 0;

    const scaleLerp = Math.min(1, dt * (rec.dealt ? 15 : 8.5));
    const lerp = Math.min(1, dt * 15);
    rec.scaleX += (targetScale - rec.scaleX) * scaleLerp;
    rec.scaleY += (targetScale - rec.scaleY) * scaleLerp;
    rec.tiltX += (targetTiltX - rec.tiltX) * lerp;
    rec.tiltY += (targetTiltY - rec.tiltY) * lerp;

    rec.container.position.set(rec.x, rec.y);
    rec.container.rotation = rec.rot;
    rec.container.scale.set(rec.scaleX, rec.scaleY);
    rec.container.zIndex = isHovered ? 200 : (rec.dealt ? 20 + i : 120 + (rec.dealOrder || i));

    // Dynamic shadow offset & hover glow
    const targetShadowX = isHovered ? -nx * 10 : 0;
    const targetShadowY = isHovered ? 18 - ny * 6 : 10;
    rec.shadowSprite.x += (targetShadowX - rec.shadowSprite.x) * lerp;
    rec.shadowSprite.y += (targetShadowY - rec.shadowSprite.y) * lerp;

    const targetOutline = isHovered ? 1 : 0;
    rec.hoverOutline.alpha += (targetOutline - rec.hoverOutline.alpha) * lerp;

    const cardAccent = resolveCardAccent(rec.deckKey, rec.item, p);
    if (Math.abs(rec.tiltX) > 0.0004 || Math.abs(rec.tiltY) > 0.0004) {
      applyCardPerspective(rec, rec.tiltX, rec.tiltY, cardAccent);
      rec.isTilted = true;
    } else if (rec.isTilted) {
      rec.tiltX = 0;
      rec.tiltY = 0;
      applyCardPerspective(rec, 0, 0, cardAccent);
      rec.isTilted = false;
    }

    // Restrained dual-diagonal satin sheen tracking pointer angle across tilted plane
    const targetFoilA = isHovered
      ? p.foilAlpha * Math.max(0.18, Math.min(1, 0.56 + (nx + ny) * 0.36))
      : 0;
    const targetFoilB = isHovered
      ? p.foilAlpha * Math.max(0.18, Math.min(1, 0.56 + (-nx + ny) * 0.36))
      : 0;
    rec.foilMeshA.alpha += (targetFoilA - rec.foilMeshA.alpha) * lerp;
    rec.foilMeshB.alpha += (targetFoilB - rec.foilMeshB.alpha) * lerp;
  }
}

// ---------------------------------------------------------------------------
// Detail Modal (minimal HTML layer with full project/link/media details)
// ---------------------------------------------------------------------------

function extractSVGThemeFills(rawSVG) {
  const result = { light: '', dark: '' };
  if (!rawSVG || typeof rawSVG !== 'string') {
    return result;
  }
  const styleMatch = rawSVG.match(/<style[^>]*>([\s\S]*?)<\/style>/i);
  if (!styleMatch) {
    return result;
  }
  const css = styleMatch[1];
  const darkMatch = css.match(/prefers-color-scheme\s*:\s*dark[\s\S]*?fill\s*:\s*(#[0-9a-f]{3,8})/i);
  const withoutDark = css.replace(/@media[^{]*\{[\s\S]*?\}\s*\}/gi, '');
  const lightMatch = withoutDark.match(/fill\s*:\s*(#[0-9a-f]{3,8})/i);
  if (lightMatch) {
    result.light = lightMatch[1];
  }
  if (darkMatch) {
    result.dark = darkMatch[1];
  } else if (result.light) {
    result.dark = result.light;
  }
  return result;
}

function mountSVG(container, rawSVG, opts) {
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
    if (opts && opts.stripStyle) {
      for (const st of Array.from(imported.querySelectorAll('style'))) {
        st.remove();
      }
    }
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
    coverImg.addEventListener('error', () => {
      container.textContent = '';
      container.classList.remove('is-cover');
      if (item.icon_svg) {
        container.classList.add('is-brand-icon');
        mountSVG(container, item.icon_svg);
      }
    });
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

function stopActiveMediaPlayer() {
  if (typeof state.mediaCleanup === 'function') {
    const fn = state.mediaCleanup;
    state.mediaCleanup = null;
    try {
      fn();
    } catch (_) {
      // Ignore cleanup errors on already-released media handles.
    }
  }
}

function getSavedVolume() {
  try {
    const raw = window.localStorage.getItem('wimpys-world-volume');
    if (raw !== null) {
      const parsed = parseFloat(raw);
      if (Number.isFinite(parsed) && parsed >= 0 && parsed <= 1) {
        return parsed;
      }
    }
  } catch (_) {
    // Ignore storage access errors.
  }
  return 0.85;
}

function saveVolume(val) {
  try {
    window.localStorage.setItem('wimpys-world-volume', String(Math.max(0, Math.min(1, val))));
  } catch (_) {
    // Ignore storage access errors.
  }
}

function formatAudioTime(sec) {
  if (!Number.isFinite(sec) || sec < 0) {
    return '00:00';
  }
  const total = Math.floor(sec);
  const hrs = Math.floor(total / 3600);
  const mins = Math.floor((total % 3600) / 60);
  const rem = total % 60;
  const pad = (n) => String(n).padStart(2, '0');
  return hrs > 0 ? `${hrs}:${pad(mins)}:${pad(rem)}` : `${pad(mins)}:${pad(rem)}`;
}

function makePlayerIconSVG(kind) {
  const ns = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('aria-hidden', 'true');
  svg.setAttribute('focusable', 'false');

  const add = (tag, attrs) => {
    const node = document.createElementNS(ns, tag);
    for (const [k, v] of Object.entries(attrs)) {
      node.setAttribute(k, v);
    }
    svg.appendChild(node);
  };

  if (kind === 'play') {
    add('path', { d: 'M8 5.2v13.6L19 12 8 5.2Z', fill: 'currentColor' });
  } else if (kind === 'pause') {
    add('rect', { x: '6.5', y: '5', width: '4', height: '14', rx: '1.2', fill: 'currentColor' });
    add('rect', { x: '13.5', y: '5', width: '4', height: '14', rx: '1.2', fill: 'currentColor' });
  } else if (kind === 'reset') {
    add('path', {
      d: 'M4.5 12a7.5 7.5 0 1 0 2.2-5.3L4.5 9M4.5 4.5V9H9',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2',
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
    });
  } else if (kind === 'muted') {
    add('path', { d: 'M11 5 6 9H2v6h4l5 4V5Z', fill: 'currentColor' });
    add('path', {
      d: 'm15.5 9.5 5 5m0-5-5 5',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2',
      'stroke-linecap': 'round',
    });
  } else {
    add('path', { d: 'M11 5 6 9H2v6h4l5 4V5Z', fill: 'currentColor' });
    add('path', {
      d: 'M15.5 8.5a5 5 0 0 1 0 7M18.5 5.5a9 9 0 0 1 0 13',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '1.9',
      'stroke-linecap': 'round',
    });
  }
  return svg;
}

function setRangeProgress(input, ratio) {
  const clamped = Math.max(0, Math.min(1, Number.isFinite(ratio) ? ratio : 0));
  input.style.setProperty('--range-pct', `${(clamped * 100).toFixed(2)}%`);
}

function buildPodcastAudioPlayer(audioURL, eyebrowText, durationHint) {
  const wrap = el('div', 'media-player');
  wrap.dataset.playerKind = 'podcast-audio';

  const top = el('div', 'player-top');
  top.appendChild(el('span', 'player-eyebrow', eyebrowText));
  wrap.appendChild(top);

  const center = el('div', 'player-center-controls');
  const playBtn = el('button', 'player-play-btn');
  playBtn.type = 'button';
  playBtn.setAttribute('aria-label', 'Play episode');
  playBtn.setAttribute('aria-pressed', 'false');
  playBtn.appendChild(makePlayerIconSVG('play'));
  center.appendChild(playBtn);
  wrap.appendChild(center);

  const progWrap = el('div', 'player-progress-wrap');
  const seek = document.createElement('input');
  seek.type = 'range';
  seek.className = 'player-range player-seek';
  seek.min = '0';
  seek.max = '1000';
  seek.step = '1';
  seek.value = '0';
  seek.setAttribute('aria-label', 'Episode playback position');
  setRangeProgress(seek, 0);

  const times = el('div', 'player-times');
  const elapsedEl = el('span', '', '00:00');
  const totalEl = el('span', '', durationHint || '--:--');
  times.appendChild(elapsedEl);
  times.appendChild(totalEl);
  progWrap.appendChild(seek);
  progWrap.appendChild(times);
  wrap.appendChild(progWrap);

  const volRow = el('div', 'player-volume-row');
  const muteBtn = el('button', 'player-mute-btn');
  muteBtn.type = 'button';
  muteBtn.setAttribute('aria-label', 'Mute audio');

  const volInput = document.createElement('input');
  volInput.type = 'range';
  volInput.className = 'player-range player-vol';
  volInput.min = '0';
  volInput.max = '1';
  volInput.step = '0.05';
  const initVol = getSavedVolume();
  volInput.value = String(initVol);
  volInput.setAttribute('aria-label', 'Audio volume');
  setRangeProgress(volInput, initVol);

  const volLabel = el('span', 'player-vol-label', `${Math.round(initVol * 100)}%`);
  volRow.appendChild(muteBtn);
  volRow.appendChild(volInput);
  volRow.appendChild(volLabel);
  wrap.appendChild(volRow);

  const statusEl = el('p', 'player-status');
  statusEl.hidden = true;
  wrap.appendChild(statusEl);

  const audio = new Audio();
  audio.preload = 'none';
  audio.volume = initVol;

  let isDragging = false;
  let lastNonZeroVol = initVol > 0 ? initVol : 0.85;
  let disposed = false;
  let hasAttemptedPlay = false;

  const setStatus = (msg) => {
    statusEl.textContent = msg;
    statusEl.hidden = !msg;
  };

  const syncPlayButton = (playing) => {
    playBtn.textContent = '';
    playBtn.appendChild(makePlayerIconSVG(playing ? 'pause' : 'play'));
    playBtn.setAttribute('aria-label', playing ? 'Pause episode' : 'Play episode');
    playBtn.setAttribute('aria-pressed', String(playing));
  };

  const syncVolumeUI = () => {
    const effective = audio.muted ? 0 : audio.volume;
    muteBtn.textContent = '';
    muteBtn.appendChild(makePlayerIconSVG(effective <= 0.001 ? 'muted' : 'volume'));
    muteBtn.setAttribute('aria-label', effective <= 0.001 ? 'Unmute audio' : 'Mute audio');
    volInput.value = String(effective);
    setRangeProgress(volInput, effective);
    volLabel.textContent = `${Math.round(effective * 100)}%`;
  };
  syncVolumeUI();

  playBtn.addEventListener('click', () => {
    if (disposed) {
      return;
    }
    if (audio.paused) {
      hasAttemptedPlay = true;
      if (!audio.src) {
        audio.src = audioURL;
      }
      setStatus('Loading audio...');
      const playPromise = audio.play();
      if (playPromise && typeof playPromise.then === 'function') {
        playPromise
          .then(() => {
            if (!disposed) {
              setStatus('');
              syncPlayButton(true);
            }
          })
          .catch(() => {
            if (!disposed) {
              syncPlayButton(false);
              setStatus('Unable to start audio playback.');
            }
          });
      }
    } else {
      audio.pause();
      syncPlayButton(false);
      setStatus('');
    }
  });

  seek.addEventListener('pointerdown', () => {
    isDragging = true;
  });
  seek.addEventListener('input', () => {
    const frac = Number(seek.value) / 1000;
    setRangeProgress(seek, frac);
    if (Number.isFinite(audio.duration) && audio.duration > 0) {
      elapsedEl.textContent = formatAudioTime(frac * audio.duration);
    }
  });
  const commitSeek = () => {
    isDragging = false;
    if (Number.isFinite(audio.duration) && audio.duration > 0) {
      const frac = Number(seek.value) / 1000;
      audio.currentTime = Math.max(0, Math.min(audio.duration, frac * audio.duration));
    }
  };
  seek.addEventListener('change', commitSeek);
  seek.addEventListener('pointerup', commitSeek);

  volInput.addEventListener('input', () => {
    const v = Math.max(0, Math.min(1, parseFloat(volInput.value) || 0));
    audio.muted = v <= 0.001;
    audio.volume = v;
    if (v > 0.001) {
      lastNonZeroVol = v;
      saveVolume(v);
    }
    syncVolumeUI();
  });

  muteBtn.addEventListener('click', () => {
    if (audio.muted || audio.volume <= 0.001) {
      audio.muted = false;
      audio.volume = lastNonZeroVol;
      saveVolume(lastNonZeroVol);
    } else {
      lastNonZeroVol = audio.volume;
      audio.muted = true;
    }
    syncVolumeUI();
  });

  audio.addEventListener('loadedmetadata', () => {
    if (disposed) {
      return;
    }
    if (Number.isFinite(audio.duration) && audio.duration > 0) {
      totalEl.textContent = formatAudioTime(audio.duration);
    }
  });

  audio.addEventListener('timeupdate', () => {
    if (disposed || isDragging || !Number.isFinite(audio.duration) || audio.duration <= 0) {
      return;
    }
    const frac = Math.max(0, Math.min(1, audio.currentTime / audio.duration));
    seek.value = String(Math.round(frac * 1000));
    setRangeProgress(seek, frac);
    elapsedEl.textContent = formatAudioTime(audio.currentTime);
  });

  audio.addEventListener('playing', () => {
    if (!disposed) {
      setStatus('');
      syncPlayButton(true);
    }
  });

  audio.addEventListener('waiting', () => {
    if (!disposed && !audio.paused) {
      setStatus('Buffering...');
    }
  });

  audio.addEventListener('pause', () => {
    if (!disposed) {
      syncPlayButton(false);
    }
  });

  audio.addEventListener('ended', () => {
    if (!disposed) {
      syncPlayButton(false);
      seek.value = '0';
      setRangeProgress(seek, 0);
      elapsedEl.textContent = '00:00';
      setStatus('');
    }
  });

  audio.addEventListener('error', () => {
    if (!disposed && hasAttemptedPlay) {
      syncPlayButton(false);
      setStatus('Audio stream unavailable.');
    }
  });

  state.mediaCleanup = () => {
    disposed = true;
    audio.pause();
    audio.removeAttribute('src');
    audio.load();
  };

  return wrap;
}

function splitSpeechChunks(rawText, maxLen) {
  const cleaned = String(rawText || '')
    .replace(/https?:\/\/\S+/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  if (!cleaned) {
    return [];
  }
  const sentences = cleaned.split(/(?<=[.!?])\s+/).filter(Boolean);
  const chunks = [];

  for (const sentence of sentences) {
    if (sentence.length <= maxLen) {
      chunks.push(sentence);
      continue;
    }
    const words = sentence.split(/\s+/);
    let buf = '';
    for (const word of words) {
      const candidate = buf ? `${buf} ${word}` : word;
      if (candidate.length <= maxLen || !buf) {
        buf = candidate;
      } else {
        chunks.push(buf);
        buf = word;
      }
    }
    if (buf) {
      chunks.push(buf);
    }
  }
  return chunks;
}

function pickSpeechVoice(synth) {
  if (!synth || typeof synth.getVoices !== 'function') {
    return null;
  }
  const voices = synth.getVoices() || [];
  if (voices.length === 0) {
    return null;
  }
  return (
    voices.find((v) => /^en-GB/i.test(v.lang) && /natural|neural|google|serena|daniel|martha/i.test(v.name))
    || voices.find((v) => /^en-GB/i.test(v.lang))
    || voices.find((v) => /^en/i.test(v.lang))
    || voices[0]
  );
}

function buildBlogSpeechPlayer(speechText) {
  if (!('speechSynthesis' in window) || typeof window.SpeechSynthesisUtterance !== 'function') {
    return null;
  }
  const chunks = splitSpeechChunks(speechText, 210);
  if (chunks.length === 0) {
    return null;
  }

  const synth = window.speechSynthesis;
  const wrap = el('div', 'media-player');
  wrap.dataset.playerKind = 'blog-tts';

  const rates = [1, 1.25, 1.5];
  let rateIdx = 0;

  const top = el('div', 'player-top');
  top.appendChild(el('span', 'player-eyebrow', 'Listen · Text-to-Speech'));
  const rateBtn = el('button', 'player-rate-btn', '1x');
  rateBtn.type = 'button';
  rateBtn.setAttribute('aria-label', 'Reading speed');
  top.appendChild(rateBtn);
  wrap.appendChild(top);

  const center = el('div', 'player-center-controls');
  const playBtn = el('button', 'player-play-btn');
  playBtn.type = 'button';
  playBtn.setAttribute('aria-label', 'Read post aloud');
  playBtn.setAttribute('aria-pressed', 'false');
  playBtn.appendChild(makePlayerIconSVG('play'));

  const resetBtn = el('button', 'player-reset-btn');
  resetBtn.type = 'button';
  resetBtn.hidden = true;
  resetBtn.setAttribute('aria-label', 'Restart reading from beginning');
  resetBtn.appendChild(makePlayerIconSVG('reset'));

  center.appendChild(playBtn);
  center.appendChild(resetBtn);
  wrap.appendChild(center);

  const progWrap = el('div', 'player-progress-wrap');
  const seek = document.createElement('input');
  seek.type = 'range';
  seek.className = 'player-range player-seek';
  seek.min = '0';
  seek.max = String(Math.max(0, chunks.length - 1));
  seek.step = '1';
  seek.value = '0';
  seek.setAttribute('aria-label', 'Reading progress');
  setRangeProgress(seek, 0);

  const times = el('div', 'player-times');
  const elapsedEl = el('span', '', `Part 1 / ${chunks.length}`);
  const totalEl = el('span', '', `${chunks.length} ${chunks.length === 1 ? 'part' : 'parts'}`);
  times.appendChild(elapsedEl);
  times.appendChild(totalEl);
  progWrap.appendChild(seek);
  progWrap.appendChild(times);
  wrap.appendChild(progWrap);

  const volRow = el('div', 'player-volume-row');
  const muteBtn = el('button', 'player-mute-btn');
  muteBtn.type = 'button';
  muteBtn.setAttribute('aria-label', 'Mute speech');

  const volInput = document.createElement('input');
  volInput.type = 'range';
  volInput.className = 'player-range player-vol';
  volInput.min = '0';
  volInput.max = '1';
  volInput.step = '0.05';
  let volume = getSavedVolume();
  let muted = volume <= 0.001;
  let lastNonZeroVol = volume > 0 ? volume : 0.85;
  volInput.value = String(volume);
  volInput.setAttribute('aria-label', 'Speech volume');
  setRangeProgress(volInput, volume);

  const volLabel = el('span', 'player-vol-label', `${Math.round(volume * 100)}%`);
  volRow.appendChild(muteBtn);
  volRow.appendChild(volInput);
  volRow.appendChild(volLabel);
  wrap.appendChild(volRow);

  const statusEl = el('p', 'player-status');
  statusEl.hidden = true;
  wrap.appendChild(statusEl);

  let currentIdx = 0;
  let isPlaying = false;
  let runToken = 0;
  let disposed = false;

  const setStatus = (msg) => {
    statusEl.textContent = msg;
    statusEl.hidden = !msg;
  };

  const syncPlayUI = () => {
    playBtn.textContent = '';
    playBtn.appendChild(makePlayerIconSVG(isPlaying ? 'pause' : 'play'));
    playBtn.setAttribute('aria-label', isPlaying ? 'Pause reading' : 'Read post aloud');
    playBtn.setAttribute('aria-pressed', String(isPlaying));
    resetBtn.hidden = !isPlaying && currentIdx === 0;
  };

  const syncProgressUI = () => {
    const denom = Math.max(1, chunks.length - 1);
    const ratio = chunks.length <= 1 ? (isPlaying ? 1 : 0) : currentIdx / denom;
    seek.value = String(currentIdx);
    setRangeProgress(seek, ratio);
    elapsedEl.textContent = `Part ${currentIdx + 1} / ${chunks.length}`;
  };

  const syncVolumeUI = () => {
    const eff = muted ? 0 : volume;
    muteBtn.textContent = '';
    muteBtn.appendChild(makePlayerIconSVG(eff <= 0.001 ? 'muted' : 'volume'));
    muteBtn.setAttribute('aria-label', eff <= 0.001 ? 'Unmute speech' : 'Mute speech');
    volInput.value = String(eff);
    setRangeProgress(volInput, eff);
    volLabel.textContent = `${Math.round(eff * 100)}%`;
  };
  syncVolumeUI();
  syncProgressUI();

  const speakFrom = (index) => {
    if (disposed) {
      return;
    }
    const token = ++runToken;
    synth.cancel();

    if (index >= chunks.length) {
      isPlaying = false;
      currentIdx = 0;
      syncPlayUI();
      syncProgressUI();
      setStatus('');
      return;
    }

    currentIdx = index;
    isPlaying = true;
    syncPlayUI();
    syncProgressUI();
    setStatus('');

    const utter = new window.SpeechSynthesisUtterance(chunks[index]);
    const voice = pickSpeechVoice(synth);
    if (voice) {
      utter.voice = voice;
      utter.lang = voice.lang;
    } else {
      utter.lang = 'en-GB';
    }
    utter.rate = rates[rateIdx];
    utter.volume = muted ? 0 : volume;

    utter.onend = () => {
      if (disposed || token !== runToken) {
        return;
      }
      speakFrom(index + 1);
    };

    utter.onerror = (ev) => {
      if (disposed || token !== runToken) {
        return;
      }
      if (ev && (ev.error === 'interrupted' || ev.error === 'canceled')) {
        return;
      }
      isPlaying = false;
      syncPlayUI();
      setStatus('No browser speech voice available.');
    };

    synth.speak(utter);
  };

  playBtn.addEventListener('click', () => {
    if (disposed) {
      return;
    }
    if (isPlaying) {
      runToken += 1;
      synth.cancel();
      isPlaying = false;
      syncPlayUI();
      setStatus('Paused');
    } else {
      speakFrom(currentIdx);
    }
  });

  resetBtn.addEventListener('click', () => {
    if (disposed) {
      return;
    }
    runToken += 1;
    synth.cancel();
    currentIdx = 0;
    isPlaying = false;
    syncPlayUI();
    syncProgressUI();
    setStatus('');
  });

  rateBtn.addEventListener('click', () => {
    rateIdx = (rateIdx + 1) % rates.length;
    rateBtn.textContent = `${rates[rateIdx]}x`;
    if (isPlaying) {
      speakFrom(currentIdx);
    }
  });

  seek.addEventListener('input', () => {
    currentIdx = Math.max(0, Math.min(chunks.length - 1, parseInt(seek.value, 10) || 0));
    syncProgressUI();
  });

  seek.addEventListener('change', () => {
    currentIdx = Math.max(0, Math.min(chunks.length - 1, parseInt(seek.value, 10) || 0));
    syncProgressUI();
    if (isPlaying) {
      speakFrom(currentIdx);
    }
  });

  volInput.addEventListener('input', () => {
    const v = Math.max(0, Math.min(1, parseFloat(volInput.value) || 0));
    volume = v;
    muted = v <= 0.001;
    if (v > 0.001) {
      lastNonZeroVol = v;
      saveVolume(v);
    }
    syncVolumeUI();
    if (isPlaying) {
      speakFrom(currentIdx);
    }
  });

  muteBtn.addEventListener('click', () => {
    if (muted || volume <= 0.001) {
      muted = false;
      volume = lastNonZeroVol;
      saveVolume(lastNonZeroVol);
    } else {
      lastNonZeroVol = volume;
      muted = true;
    }
    syncVolumeUI();
    if (isPlaying) {
      speakFrom(currentIdx);
    }
  });

  state.mediaCleanup = () => {
    disposed = true;
    runToken += 1;
    isPlaying = false;
    synth.cancel();
  };

  return wrap;
}

function makeActionIconSVG(kind) {
  const ns = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('aria-hidden', 'true');
  svg.setAttribute('focusable', 'false');

  const add = (tag, attrs) => {
    const node = document.createElementNS(ns, tag);
    for (const [k, v] of Object.entries(attrs)) {
      node.setAttribute(k, v);
    }
    svg.appendChild(node);
  };

  if (kind === 'github') {
    add('path', {
      d: 'M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12',
      fill: 'currentColor',
    });
  } else if (kind === 'globe') {
    add('circle', { cx: '12', cy: '12', r: '9.5', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.9' });
    add('ellipse', { cx: '12', cy: '12', rx: '4.2', ry: '9.5', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.8' });
    add('line', { x1: '2.8', y1: '12', x2: '21.2', y2: '12', stroke: 'currentColor', 'stroke-width': '1.8' });
  } else if (kind === 'package') {
    add('path', {
      d: 'M12 2.5 20.5 7.2v9.6L12 21.5 3.5 16.8V7.2L12 2.5Z',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '1.9',
      'stroke-linejoin': 'round',
    });
    add('path', {
      d: 'M3.8 7.4 12 12l8.2-4.6M12 12v9.3M7.8 4.9 16.2 9.6',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '1.8',
      'stroke-linecap': 'round',
      'stroke-linejoin': 'round',
    });
  }
  return svg;
}

function actionLinkWithIcon(href, className, iconKind, label) {
  const a = linkEl(href, className);
  const iconWrap = el('span', 'action-btn-icon');
  iconWrap.appendChild(makeActionIconSVG(iconKind));
  a.appendChild(iconWrap);
  a.appendChild(document.createTextNode(label));
  return a;
}

function buildSubscribeRow(links) {
  const row = el('div', 'subscribe-row');
  for (const sub of links) {
    const pill = linkEl(sub.url, 'subscribe-pill');
    if (sub.icon_svg) {
      const iconSpan = el('span', 'subscribe-icon');
      mountSVG(iconSpan, sub.icon_svg);
      pill.appendChild(iconSpan);
    }
    pill.appendChild(document.createTextNode(sub.label));
    row.appendChild(pill);
  }
  return row;
}

function resolveHostKey(hostRef) {
  const key = String((hostRef && hostRef.name) || '').toLowerCase().trim();
  if (HOST_META[key]) {
    return key;
  }
  const sub = String((hostRef && hostRef.host) || '').split('.')[0].toLowerCase().trim();
  if (HOST_META[sub]) {
    return sub;
  }
  return key || sub;
}

function resolveHostMeta(hostRef) {
  const key = resolveHostKey(hostRef);
  if (HOST_META[key]) {
    return HOST_META[key];
  }
  return {
    label: (hostRef.name || hostRef.host || 'PKG').toUpperCase(),
    subtitle: hostRef.host || 'Package repository',
  };
}

function buildModalHeader(deckKey, item, primaryURL) {
  const header = el('div', 'modal-header');
  const titleGroup = el('div', 'modal-title-group');

  const art = el('div', 'modal-art');
  mountArtworkDOM(art, item);
  if (primaryURL && isSafeURL(primaryURL)) {
    const artLink = linkEl(primaryURL, 'modal-art-link');
    artLink.setAttribute('aria-label', `Open ${item.name || item.title || 'item'}`);
    artLink.appendChild(art);
    titleGroup.appendChild(artLink);
  } else {
    titleGroup.appendChild(art);
  }

  const titleCol = el('div', 'modal-title-col');
  const titleText = item.name || item.title || 'Details';
  const h2 = el('h2', 'modal-title');
  h2.id = 'modal-title';
  if (primaryURL && isSafeURL(primaryURL)) {
    h2.appendChild(linkEl(primaryURL, 'modal-title-link', titleText));
  } else {
    h2.textContent = titleText;
  }
  titleCol.appendChild(h2);

  if (deckKey === 'projects') {
    if (item.full_name) {
      if (item.repo_url && isSafeURL(item.repo_url)) {
        titleCol.appendChild(linkEl(item.repo_url, 'modal-eyebrow-link', item.full_name));
      } else {
        titleCol.appendChild(el('p', 'modal-eyebrow', item.full_name));
      }
    }
  } else if (deckKey === 'links') {
    const row = el('div', 'modal-eyebrow-row');
    row.appendChild(el('span', 'modal-category-badge', item.category || 'Link'));
    if (item.host_label) {
      row.appendChild(el('span', 'modal-eyebrow', item.host_label));
    }
    titleCol.appendChild(row);
  } else if (deckKey === 'media') {
    const parts = [item.category || 'Media'];
    if (item.episode) {
      parts.push(`Episode #${item.episode}`);
    }
    if (item.date_label) {
      parts.push(item.date_label);
    }
    titleCol.appendChild(el('p', 'modal-eyebrow', parts.join(' · ')));
  }

  titleGroup.appendChild(titleCol);
  header.appendChild(titleGroup);
  return header;
}

function buildModalContent(deckKey, item) {
  const scroll = el('div', 'flip-back-scroll');

  let primaryURL = '';
  if (deckKey === 'projects') {
    primaryURL = (item.homepage && isSafeURL(item.homepage))
      ? item.homepage
      : ((item.repo_url && isSafeURL(item.repo_url)) ? item.repo_url : '');
  } else if (deckKey === 'links') {
    primaryURL = item.url && isSafeURL(item.url) ? item.url : '';
  } else if (deckKey === 'media') {
    if (item.kind === 'podcast-hub' || item.kind === 'blog-hub') {
      primaryURL = (item.site_url && isSafeURL(item.site_url)) ? item.site_url : item.url;
    } else {
      primaryURL = item.url;
    }
  }

  scroll.appendChild(buildModalHeader(deckKey, item, primaryURL));

  if (deckKey === 'projects') {
    if (item.description) {
      scroll.appendChild(el('p', 'modal-desc', item.description));
    }

    const stats = el('div', 'detail-stats');
    const starText = `★ ${item.stars_formatted || item.stars || 0} stars`;
    if (item.repo_url && isSafeURL(item.repo_url)) {
      stats.appendChild(linkEl(item.repo_url, 'detail-stat', starText));
    } else {
      stats.appendChild(el('span', 'detail-stat', starText));
    }

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
    const hasHomepage = Boolean(item.homepage && isSafeURL(item.homepage) && item.homepage !== item.repo_url);
    if (hasHomepage) {
      actions.appendChild(actionLinkWithIcon(item.homepage, 'action-btn is-primary', 'globe', 'Homepage'));
    }
    if (item.repo_url && isSafeURL(item.repo_url)) {
      actions.appendChild(actionLinkWithIcon(item.repo_url, `action-btn${hasHomepage ? '' : ' is-primary'}`, 'github', 'GitHub'));
    }
    const releasesURL = (item.release_url && isSafeURL(item.release_url))
      ? item.release_url
      : (item.repo_url && isSafeURL(item.repo_url) ? `${item.repo_url.replace(/\/$/, '')}/releases` : '');
    if (actions.childElementCount > 0) {
      scroll.appendChild(actions);
    }

    const hasHosts = Boolean(item.hosts && item.hosts.length > 0);
    if (hasHosts || releasesURL) {
      const installSec = el('div', 'detail-section');
      installSec.appendChild(el('h3', 'detail-section-title', 'Install'));
      const grid = el('div', 'host-tile-grid');
      const logos = (state.data && state.data.host_logos) || {};

      if (hasHosts) {
        for (const h of item.hosts) {
          const hostKey = resolveHostKey(h);
          const meta = resolveHostMeta(h);
          const tile = linkEl(h.url, 'host-tile');
          if (hostKey) {
            tile.dataset.host = hostKey;
          }
          const logoBox = el('span', 'host-tile-logo');
          if (logos[h.host]) {
            if (!HOST_META[hostKey]) {
              const extracted = extractSVGThemeFills(logos[h.host]);
              if (extracted.dark) {
                tile.style.setProperty('--host-color-dark', extracted.dark);
              }
              if (extracted.light) {
                tile.style.setProperty('--host-color-light', extracted.light);
              }
            }
            mountSVG(logoBox, logos[h.host], { stripStyle: true });
          }
          const copy = el('span', 'host-tile-copy');
          copy.appendChild(el('span', 'host-tile-name', meta.label));
          copy.appendChild(el('span', 'host-tile-sub', meta.subtitle));
          tile.appendChild(logoBox);
          tile.appendChild(copy);
          grid.appendChild(tile);
        }
      }

      if (releasesURL) {
        const relTile = linkEl(releasesURL, 'host-tile');
        relTile.dataset.host = 'releases';
        const relLogo = el('span', 'host-tile-logo');
        relLogo.appendChild(makeActionIconSVG('package'));
        const relCopy = el('span', 'host-tile-copy');
        relCopy.appendChild(el('span', 'host-tile-name', 'Releases'));
        relCopy.appendChild(el('span', 'host-tile-sub', item.latest_tag ? `${item.latest_tag} · Archives` : 'Direct downloads'));
        relTile.appendChild(relLogo);
        relTile.appendChild(relCopy);
        grid.appendChild(relTile);
      }

      installSec.appendChild(grid);
      scroll.appendChild(installSec);
    }
  } else if (deckKey === 'links') {
    if (item.description) {
      scroll.appendChild(el('p', 'modal-desc', item.description));
    }

    if (item.meta && item.meta.length > 0) {
      const linkStats = el('div', 'detail-stats');
      for (const m of item.meta) {
        const lower = String(m).toLowerCase();
        const isFresh = lower.includes('online') || lower.includes('active');
        linkStats.appendChild(el('span', `detail-stat${isFresh ? ' is-fresh' : ''}`, m));
      }
      scroll.appendChild(linkStats);
    }

    const linkActions = el('div', 'detail-actions');
    linkActions.appendChild(linkEl(item.url, 'action-btn is-primary', `Visit ${item.title} ↗`));
    const isGitHubCard = String(item.icon || '').toLowerCase() === 'github' || item.id === 'github';
    const sponsorsURL = state.data && state.data.profile && state.data.profile.sponsors_url;
    if (isGitHubCard && sponsorsURL && isSafeURL(sponsorsURL)) {
      linkActions.appendChild(linkEl(sponsorsURL, 'action-btn', 'Sponsor on GitHub ♥'));
    }
    scroll.appendChild(linkActions);

    if (item.sponsors && item.sponsors.length > 0) {
      const spSec = el('div', 'detail-section');
      spSec.appendChild(el('h3', 'detail-section-title', 'Active Sponsors'));
      const list = el('div', 'sponsor-chip-list');
      for (const sp of item.sponsors) {
        const href = (sp.url && isSafeURL(sp.url)) ? sp.url : `https://github.com/${encodeURIComponent(sp.login)}`;
        list.appendChild(linkEl(href, 'sponsor-chip', `@${sp.login}`));
      }
      spSec.appendChild(list);
      scroll.appendChild(spSec);
    }

    if (item.past_sponsors && item.past_sponsors.length > 0) {
      const pastSec = el('div', 'detail-section');
      pastSec.appendChild(el('h3', 'detail-section-title', 'Past Sponsors'));
      const list = el('div', 'sponsor-chip-list');
      for (const sp of item.past_sponsors) {
        const href = (sp.url && isSafeURL(sp.url)) ? sp.url : `https://github.com/${encodeURIComponent(sp.login)}`;
        list.appendChild(linkEl(href, 'sponsor-chip is-past', `@${sp.login}`));
      }
      pastSec.appendChild(list);
      scroll.appendChild(pastSec);
    }
  } else if (deckKey === 'media') {
    const mediaStats = el('div', 'detail-stats');
    if (item.episode) {
      mediaStats.appendChild(el('span', 'detail-stat', `#${item.episode}`));
    }
    if (item.duration) {
      mediaStats.appendChild(el('span', 'detail-stat', item.duration));
    } else if (item.read_time_minutes > 0) {
      mediaStats.appendChild(el('span', 'detail-stat', `${item.read_time_minutes} min read`));
    }
    if (item.date_label) {
      mediaStats.appendChild(el('span', 'detail-stat', item.date_label));
    } else if (item.age) {
      mediaStats.appendChild(el('span', 'detail-stat', item.age));
    }
    if (mediaStats.childElementCount > 0) {
      scroll.appendChild(mediaStats);
    }

    if (item.summary) {
      scroll.appendChild(el('p', 'modal-desc', item.summary));
    }

    if ((item.kind === 'episode' || item.kind === 'podcast-hub') && item.audio_url && isSafeURL(item.audio_url)) {
      const playerLabel = item.kind === 'podcast-hub' && item.episode
        ? `Play Latest · Episode #${item.episode}`
        : 'Play Episode';
      scroll.appendChild(buildPodcastAudioPlayer(item.audio_url, playerLabel, item.duration || ''));
    } else if (item.kind === 'post' && (item.speech_text || item.summary)) {
      const ttsPlayer = buildBlogSpeechPlayer(item.speech_text || item.summary);
      if (ttsPlayer) {
        scroll.appendChild(ttsPlayer);
      }
    }

    const mediaActions = el('div', 'detail-actions');
    let primaryLabel = 'Open ↗';
    let primaryHref = item.url;
    if (item.kind === 'episode') {
      primaryLabel = 'Episode Page ↗';
    } else if (item.kind === 'post') {
      primaryLabel = 'Read Post ↗';
    } else if (item.kind === 'podcast-hub') {
      primaryLabel = 'Visit Linux Matters ↗';
      primaryHref = (item.site_url && isSafeURL(item.site_url)) ? item.site_url : item.url;
    } else if (item.kind === 'blog-hub') {
      primaryLabel = 'Visit Blog ↗';
      primaryHref = (item.site_url && isSafeURL(item.site_url)) ? item.site_url : item.url;
    } else if (item.kind === 'video') {
      primaryLabel = 'Watch Video ↗';
    }
    mediaActions.appendChild(linkEl(primaryHref, 'action-btn is-primary', primaryLabel));
    if (item.site_url && item.site_url !== primaryHref && isSafeURL(item.site_url)) {
      mediaActions.appendChild(linkEl(item.site_url, 'action-btn', `All ${item.category || 'Entries'} ↗`));
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
        } else if (entry.read_time_minutes > 0) {
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

function captureCardFrontDataURL(rec) {
  if (!rec || !rec.canvas) {
    return '';
  }
  try {
    return rec.canvas.toDataURL('image/png');
  } catch (_) {
    return '';
  }
}

function mountFrontFaceSnapshot(rec) {
  if (!els.flipFront) {
    return;
  }
  els.flipFront.textContent = '';
  const dataURL = captureCardFrontDataURL(rec);
  if (dataURL) {
    const img = document.createElement('img');
    img.className = 'flip-front-img';
    img.src = dataURL;
    img.alt = '';
    img.draggable = false;
    els.flipFront.appendChild(img);
  }
}

function easeOutCubic(t) {
  return 1 - Math.pow(1 - t, 3);
}

function formatFlightTransform(dx, dy, rz, sx, sy, rx, ry) {
  return (
    `translate3d(${dx.toFixed(2)}px, ${dy.toFixed(2)}px, 0) `
    + `rotateZ(${rz.toFixed(4)}rad) `
    + `scale(${sx.toFixed(4)}, ${sy.toFixed(4)}) `
    + `rotateX(${rx.toFixed(2)}deg) `
    + `rotateY(${ry.toFixed(2)}deg)`
  );
}

function finishModalClose(restoreFocus) {
  stopActiveMediaPlayer();
  if (state.modalAnimRaf) {
    window.cancelAnimationFrame(state.modalAnimRaf);
    state.modalAnimRaf = 0;
  }

  const rec = state.activeCard;
  if (rec && !rec.destroyed) {
    rec.container.visible = true;
    rec.pickedUp = false;
  }

  state.activeCard = null;
  state.modalOpen = false;
  state.modalPhase = 'closed';
  state.hoveredCardId = null;
  state.focusedCardId = null;

  if (els.flipFloatWrap) {
    els.flipFloatWrap.classList.remove('is-floating');
  }
  if (els.flipCard) {
    els.flipCard.classList.remove('is-settled');
    els.flipCard.style.transform = '';
  }
  if (els.flipBack) {
    delete els.flipBack.dataset.cardAccent;
    delete els.flipBack.dataset.cardBrand;
  }
  if (els.modal) {
    els.modal.style.setProperty('--backdrop-progress', '0');
    els.modal.hidden = true;
  }
  document.body.classList.remove('modal-open');

  const returnTarget = state.lastFocusedEl;
  const shouldRestoreFocus = Boolean(restoreFocus && state.openedByKeyboard);
  state.lastFocusedEl = null;
  state.openedByKeyboard = false;
  if (shouldRestoreFocus && returnTarget && typeof returnTarget.focus === 'function') {
    returnTarget.focus();
  } else if (document.activeElement && typeof document.activeElement.blur === 'function') {
    document.activeElement.blur();
  }
}

function cancelCardModal(restoreFocus) {
  if (state.modalPhase === 'closed' && !state.modalOpen) {
    return;
  }
  finishModalClose(Boolean(restoreFocus));
}

function applyBackFaceBranding(deckKey, item) {
  if (!els.flipBack) {
    return;
  }
  delete els.flipBack.dataset.cardAccent;
  delete els.flipBack.dataset.cardBrand;

  if (isLinuxMattersItem(deckKey, item)) {
    els.flipBack.dataset.cardBrand = 'linux-matters';
    return;
  }
  if (deckKey === 'links' || (deckKey === 'media' && item && item.accent)) {
    const acc = String((item && item.accent) || '').toLowerCase().trim();
    if (acc) {
      els.flipBack.dataset.cardAccent = acc;
    }
  }
}

function triggerCardOpen(rec, triggerEl, byKeyboard) {
  if (state.modalPhase !== 'closed' || !els.modal || !els.flipFloatWrap || !els.flipCard || !els.flipBack) {
    return;
  }

  state.lastFocusedEl = triggerEl || document.activeElement;
  state.openedByKeyboard = Boolean(byKeyboard);
  state.activeCard = rec;
  state.modalOpen = true;
  state.hoveredCardId = null;
  state.focusedCardId = null;

  const startX = rec.x;
  const startY = rec.y - window.scrollY;
  const startRot = rec.rot;
  const cardScale = state.layout.cardScale || 1;
  const startW = CARD_W * (rec.scaleX || cardScale);
  const startH = CARD_H * (rec.scaleY || cardScale);

  mountFrontFaceSnapshot(rec);
  applyBackFaceBranding(rec.deckKey, rec.item);
  els.flipBack.textContent = '';
  els.flipBack.appendChild(buildModalContent(rec.deckKey, rec.item));
  els.flipFloatWrap.classList.remove('is-floating');
  els.flipCard.classList.remove('is-settled');
  els.flipCard.style.transform = '';

  els.modal.hidden = false;
  document.body.classList.add('modal-open');

  if (isReducedMotion() || !state.app) {
    if (rec && !rec.destroyed) {
      rec.container.visible = false;
      rec.pickedUp = true;
    }
    els.modal.style.setProperty('--backdrop-progress', '1');
    els.flipCard.style.transform = 'rotateY(180deg)';
    els.flipCard.classList.add('is-settled');
    state.modalPhase = 'open';
    els.flipBack.focus();
    return;
  }

  const rect = els.flipFloatWrap.getBoundingClientRect();
  const targetW = rect.width || Math.min(440, window.innerWidth * 0.9);
  const targetH = rect.height || Math.min(616, window.innerHeight * 0.86);
  const targetCx = rect.left + targetW / 2;
  const targetCy = rect.top + targetH / 2;

  const startDx = startX - targetCx;
  const startDy = startY - targetCy;
  const startSx = startW / targetW;
  const startSy = startH / targetH;

  // Place flying card on the exact tabletop slot and hide the PixiJS sprite on the same frame.
  els.modal.style.setProperty('--backdrop-progress', '0');
  els.flipCard.style.transform = formatFlightTransform(startDx, startDy, startRot, startSx, startSy, 0, 0);
  if (rec && !rec.destroyed) {
    rec.container.visible = false;
    rec.pickedUp = true;
  }

  state.modalPhase = 'flippingIn';
  const startTime = performance.now();

  const stepIn = (now) => {
    if (state.modalPhase !== 'flippingIn') {
      return;
    }
    const rawT = Math.min(1, Math.max(0, (now - startTime) / FLIP_IN_MS));
    const e = easeOutCubic(rawT);

    const dx = startDx * (1 - e);
    const dy = startDy * (1 - e);
    const sx = startSx + (1 - startSx) * e;
    const sy = startSy + (1 - startSy) * e;
    const rz = startRot * (1 - e);
    const ry = 180 * e;
    const rx = Math.sin(rawT * Math.PI) * 10;

    els.flipCard.style.transform = formatFlightTransform(dx, dy, rz, sx, sy, rx, ry);
    els.modal.style.setProperty('--backdrop-progress', e.toFixed(3));

    if (rawT < 1) {
      state.modalAnimRaf = window.requestAnimationFrame(stepIn);
      return;
    }

    state.modalAnimRaf = 0;
    state.modalPhase = 'open';
    els.flipCard.style.transform = 'rotateY(180deg)';
    els.flipCard.classList.add('is-settled');
    els.modal.style.setProperty('--backdrop-progress', '1');
    els.flipFloatWrap.classList.add('is-floating');
    els.flipBack.focus();
  };

  state.modalAnimRaf = window.requestAnimationFrame(stepIn);
}

function closeDetailModal() {
  if (state.modalPhase !== 'open' || !els.modal || !els.flipFloatWrap || !els.flipCard) {
    return;
  }
  stopActiveMediaPlayer();

  const rec = state.activeCard;
  if (isReducedMotion() || !state.app || !rec || rec.destroyed) {
    finishModalClose(true);
    return;
  }

  state.modalPhase = 'flippingOut';
  els.flipFloatWrap.classList.remove('is-floating');
  els.flipCard.classList.remove('is-settled');

  const rect = els.flipFloatWrap.getBoundingClientRect();
  const fromW = rect.width || Math.min(440, window.innerWidth * 0.9);
  const fromH = rect.height || Math.min(616, window.innerHeight * 0.86);
  const fromCx = rect.left + fromW / 2;
  const fromCy = rect.top + fromH / 2;
  const startTime = performance.now();

  const stepOut = (now) => {
    if (state.modalPhase !== 'flippingOut') {
      return;
    }
    if (!rec || rec.destroyed) {
      finishModalClose(true);
      return;
    }

    const rawT = Math.min(1, Math.max(0, (now - startTime) / FLIP_OUT_MS));
    const e = easeOutCubic(rawT);

    const cardScale = state.layout.cardScale || 1;
    const endX = rec.x;
    const endY = rec.y - window.scrollY;
    const endRot = rec.rot;
    const endSx = (CARD_W * (rec.scaleX || cardScale)) / fromW;
    const endSy = (CARD_H * (rec.scaleY || cardScale)) / fromH;

    const dx = (endX - fromCx) * e;
    const dy = (endY - fromCy) * e;
    const sx = 1 + (endSx - 1) * e;
    const sy = 1 + (endSy - 1) * e;
    const rz = endRot * e;
    const ry = 180 * (1 - e);
    const rx = Math.sin(rawT * Math.PI) * 8;

    els.flipCard.style.transform = formatFlightTransform(dx, dy, rz, sx, sy, rx, ry);
    els.modal.style.setProperty('--backdrop-progress', (1 - e).toFixed(3));

    if (rawT < 1) {
      state.modalAnimRaf = window.requestAnimationFrame(stepOut);
      return;
    }

    state.modalAnimRaf = 0;
    finishModalClose(true);
  };

  state.modalAnimRaf = window.requestAnimationFrame(stepOut);
}

function trapModalFocus(ev) {
  if (!state.modalOpen || ev.key !== 'Tab' || !els.flipBack) {
    return;
  }
  const focusable = els.flipBack.querySelectorAll(
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

  if (els.flipBack) {
    els.flipBack.addEventListener('click', (ev) => {
      if (state.modalPhase !== 'open') {
        return;
      }
      const sel = window.getSelection && window.getSelection();
      if (sel && String(sel).trim().length > 0) {
        return;
      }
      if (ev.target && ev.target.closest('a, button, input, select, textarea, label, .media-player')) {
        return;
      }
      closeDetailModal();
    });
  }

  window.addEventListener('beforeunload', stopActiveMediaPlayer);

  window.addEventListener('hashchange', () => {
    const fromHash = normalizeDeckKey(window.location.hash);
    if (fromHash && fromHash !== effectiveDeckKey()) {
      switchDeck(fromHash, { syncHash: false });
    }
  });

  for (const link of document.querySelectorAll('.site-nav [data-deck-nav]')) {
    link.addEventListener('click', (ev) => {
      if (ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey || ev.button !== 0) {
        return;
      }
      const key = normalizeDeckKey(link.dataset.deckNav);
      if (!key) {
        return;
      }
      ev.preventDefault();
      switchDeck(key);
    });
  }

  document.addEventListener('keydown', (ev) => {
    if (state.modalOpen && ev.key === 'Escape') {
      ev.preventDefault();
      if (state.modalPhase === 'flippingIn') {
        cancelCardModal(true);
      } else {
        closeDetailModal();
      }
      return;
    }
    trapModalFocus(ev);
  });
}

async function bootstrap(data) {
  state.data = data;
  state.activeDeck = resolveInitialDeck(data);

  syncNavDeckState(state.activeDeck);
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

  // Re-bake textures and re-sync spacer origin if web fonts finish loading after initial paint
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(() => {
      if (!state.app) {
        return;
      }
      const nextOriginY = els.scrollSpacer
        ? Math.round(els.scrollSpacer.getBoundingClientRect().top + window.scrollY)
        : state.layout.originY;
      if (Math.abs(nextOriginY - state.layout.originY) > 1) {
        relayoutScene();
      }
      refreshPixiForTheme();
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
