# Portfolio site design and architecture (`wimpys.world`)

The portfolio site at `https://wimpys.world` is a static web application hosted on GitHub Pages (`flexiondotorg.github.io`). It is built by `cmd/site` and `internal/site`, driven by the single `vitrine.yaml` configuration file shared with `cmd/vitrine`.

## Hosting and security

GitHub Pages serves the site from `flexiondotorg/flexiondotorg` with custom domain `wimpys.world`. Domain Name System (DNS) records live in Cloudflare:

- Apex `A` and `AAAA` records point `wimpys.world` to GitHub Pages.
- A `CNAME` record points `www.wimpys.world` to `flexiondotorg.github.io`.
- GitHub Pages enforces HTTPS for the custom domain.

Deployment uses zero long-lived Personal Access Tokens (PATs). The scheduled `.github/workflows/pages.yml` workflow uses top-level `permissions: {}`, `contents: read` on the build job, and `pages: write` plus `id-token: write` on the deploy job with `GITHUB_TOKEN` only.

## Visual direction and hybrid tabletop architecture

The site renders a stripped-back, Balatro-inspired Top Trumps floating playing-card tabletop in **Catppuccin** palettes matching [docs/vitrine.md](vitrine.md). A single full-viewport fixed **PixiJS v8** canvas (`vendor/pixi.min.mjs`, vendored with zero npm build steps) runs behind a minimal HTML layer for the header, accessible focus buttons, quiet footer, and zoomed detail modal.

- **Minimal header**: Shows the brand mark, title, subtitle, and a three-way icon theme switch (**System** monitor icon, **Light** sun icon for Catppuccin Latte, and **Dark** moon icon for Catppuccin Mocha) persisted in `localStorage`.
- **Zero-chrome open stage**: There are no stats pills, duplicate deck tabs, sort toggles, view-mode toggles, drift-pause buttons, section headings, captions, or stage container borders. Project order is driven strictly by `vitrine.yaml`.
- **Swirling Catppuccin shader**: A custom PixiJS v8 WebGL GLSL ES 3.00 fragment shader paints a subtle domain-warped Catppuccin background that pulses gently on deck deals and falls back cleanly if shaders are unavailable.
- **Typography and contrast**: Cards and HTML chrome use **Work Sans** for titles and copy alongside **Fira Code** (`tabular-nums`) for structured stats, meeting Web Content Accessibility Guidelines (WCAG) AA contrast in both Mocha and Latte.

| Role | Catppuccin Mocha (dark) | Catppuccin Latte (light) |
| --- | --- | --- |
| Stage background | Crust (`#11111b`) | Crust (`#dce0e8`) |
| Primary card canvas | Base (`#1e1e2e`) | Base (`#eff1f5`) |
| Elevated surface | Surface0 (`#313244`) | Mantle (`#e6e9ef`) |
| Card border (`1.5px`) | Surface1 (`#45475a`) | Surface1 (`#bcc0cc`) |
| Primary ink | Text (`#cdd6f4`) | Text (`#4c4f69`) |
| Secondary ink | Subtext1 (`#bac2de`) | Subtext1 (`#5c5f77`) |
| Link ink (WCAG AA) | Blue (`#89b4fa`) | Deep Blue (`#1752c8`) |
| Fresh release ink | Green (`#a6e3a1`) | Deep Green (`#2a6f1d`) |
| Aurora accents | Mauve, Lavender, Blue, Teal (`#cba6f7`, `#b4befe`, `#89b4fa`, `#94e2d5`) | Mauve, Lavender, Blue, Teal (`#8839ef`, `#7287fd`, `#1e66f5`, `#179299`) |

## Decks and Top Trumps playing cards

Content groups into three stacked card decks placed directly above the floating cards: `Projects`, `Links`, and `Media`. `vitrine.yaml` validates `site.default_deck` against `projects`, `links`, and `media`.

### Stacked deck tray

The tabletop displays three tactile multi-layer Catppuccin card-back stacks (`Projects`, `Links`, `Media`) in a single row above the floating cards. Selecting a stack whips the outgoing cards off alternating viewport edges and deals the new deck in immediately.

### Portrait `5:7` Top Trumps card face

Each card uses `224px x 314px` logical portrait playing-card proportions, drawn onto `2x` (`448px x 628px`) offscreen canvases and uploaded to mipmapped, linearly filtered `PIXI.Texture` sprites for crisp GPU anti-aliased rendering:

- **Top index row**: Language or category dot and label on the left, paired with a release tag, episode number, read time, or live badge in Fira Code on the right.
- **Upper illustration panel**: Clean unframed artwork using sanitized inline project SVG logos, PNG data URIs, acoustic waveforms for `jive-*` slugs, monogram fallbacks, podcast cover art, or inline Simple Icons brand marks.
- **Title and description**: Bold Work Sans title with a two-line summary.
- **Lower Top Trumps stat rows**: Four alternating tinted rows with uppercase labels on the left and bold tabular Fira Code values on the right:
  - **Projects**: `STARS`, `RELEASE` (green highlight when fresh), `UPDATED`, and `PACKAGES` (`N/8 hosts` with coloured host dots, or `Source`).
  - **Links**: `CATEGORY`, `HOST`, plus live `ONLINE` and `MEMBERS` for Discord or `ACTIVE` and `PAST SPONSORS` for GitHub Sponsors.
  - **Media**: Tailored rows for podcast episodes (`EPISODE`, `DURATION`, `PUBLISHED`, `SHOW`), podcast and blog hubs (`LATEST`, `EPISODES` or `POSTS`, `LISTEN` or `SITE`), and blog posts (`READ TIME`, `PUBLISHED`, `SITE`, `FORMAT`).

### Physical pick-up, mid-air 3D flip, and portrait `5:7` back face

Clicking a card or activating its keyboard button proxy captures the card's live screen centre, rotation, scale, and `2x` front canvas snapshot, hides the PixiJS sprite on the same frame, and flies a 3D card (`460ms` cubic ease-out, `rotateY(0deg -> 180deg)`, subtle mid-air `rotateX` lift) to viewport centre at portrait `5:7` playing-card size (`min(440px, 90vw)` by `min(616px, 86vh)`). Once landed, the centred card keeps a calm reduced float (`±3px`, `±0.45deg` over `5.6s`) that pauses smoothly when hovering or focusing interactive links.

The flipped back face (`role="dialog"`, `aria-modal="true"`, `aria-labelledby="modal-title"`) keeps the playing-card border, top aurora bar, and an internal scroll area with individually clickable details:

- **Projects**: Clickable logo and title (`Homepage` or `RepoURL`), clickable `owner/repo` link, untruncated description, clickable star and release pills, action buttons (`Homepage`, `GitHub`, `Releases`), and a two-column grid of clickable Haberdasher package repository tiles (`https://<host>/#<slug>`) with each host's inline SVG logo, bold host label (`APT`, `RPM`, `PKG`, `APK`, `Flatpak`, `Homebrew`, `Scoop`, `Nix`), and format subtitle, or a `Source & Releases` card link when no package hosts exist.
- **Links**: Clickable brand icon and title, category badge, full description, live Discord online and member counts or clickable GitHub Sponsors `@login` handles (active and past), and a primary visit button.
- **Media**: Linux Matters Podcast hub (clickable cover art and title, subscribe pills for Apple Podcasts, Spotify, YouTube, and RSS, and clickable recent episodes), podcast episode cards, and Wimpy's World Blog hub and post cards.

Visitors dismiss the card by clicking outside it, clicking the close button, or pressing `Esc`, which triggers a `340ms` reverse flight (`rotateY(180deg -> 0deg)`) back into the card's live tabletop slot before revealing the PixiJS sprite and restoring focus. Focus stays trapped inside the back face while open, and `prefers-reduced-motion: reduce` opens and closes cleanly without flight, spin, or centre float.

## Motion, interaction, and accessibility

- **Scroll-synced open canvas**: The PixiJS world container tracks `world.y = -window.scrollY` alongside a `#scroll-spacer` element so cards fly in freely from the viewport edges and scroll naturally without clipping.
- **Casino-dealer deal-in**: Cards fling rapidly (`38ms` stagger) from alternating left and right viewport edges with alternating spin (`±220deg` to `±260deg`) and snap into gently fanned slots via underdamped springs with subtle deal sparkles.
- **Idle float and hover 3D tilt**: Cards bob and rock gently (`±4px`, `±1.6deg`). Hovering or focusing a card pops it forward (`scale: 1.07`), raises its z-order, tracks cursor tilt, and sweeps a Catppuccin holographic foil sheen across the face.
- **Keyboard and screen-reader layer**: Transparent focusable buttons in `#a11y-layer` align 1:1 with the three decks and active cards, supporting `Tab`, arrow keys, `Home`, `End`, `Enter`, and `Space` with visible `:focus-visible` outlines.
- **Silent OS reduced motion**: Under `@media (prefers-reduced-motion: reduce)`, spin, deal flights, idle bob, shader motion, sparkles, and flip transitions stop automatically with no visible toggle button.

## Single-config data pipeline (`cmd/site` and `internal/site`)

Both `cmd/vitrine` and `cmd/site` read `vitrine.yaml` via `internal/config` (`SiteConfig` and `SiteLink` under `site:`).

`cmd/site` accepts `-config`, `-out`, `-verbose`, and `-version`. It reuses `internal/haberdasher`, `internal/github`, `internal/discord`, and `internal/rss` to assemble data, passes it to `internal/site.Build`, and emits static output (`index.html`, `data.json`, `style.css`, `app.js`, `icon.svg`, `vendor/pixi.min.mjs`, `vendor/pixi.LICENSE`, `.nojekyll`, and `CNAME`) via `internal/site.Write` using `//go:embed web/* web/vendor/*`.

## Deployment workflow and Tailor transition

`.github/workflows/pages.yml` builds and deploys the site on `main` pushes, every 6 hours at 15 minutes past the hour (`cron: "15 */6 * * *"`), and `workflow_dispatch`. Generated HTML is built into `${RUNNER_TEMP}/site` and uploaded via `actions/upload-pages-artifact` and `actions/deploy-pages`; it is never committed to Git.

To retire the legacy Tailor holding page in `pages/`, set `pages.enabled: false` in `.tailor.yml`, remove `.github/workflows/tailor-pages.yml` and the holding-page files, and verify with `tailor measure`.

## Local preview and verification

- **Local preview**: Run `just site` to build the portfolio site from `vitrine.yaml` and serve it locally at `http://127.0.0.1:18475`.
- **Automated checks**: Run `golangci-lint fmt --config .golangci.yml`, `just build`, `just test`, `go test -race ./...`, `golangci-lint run --config .golangci.yml ./...`, and `actionlint` before pushing.
- **Rendered verification**: Inspect the live preview in headless Chromium across desktop, mobile, Mocha, Latte, and `prefers-reduced-motion: reduce` states.
