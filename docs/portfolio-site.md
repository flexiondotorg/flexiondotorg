# Portfolio site design and architecture (`wimpys.world`)

The portfolio site at `https://wimpys.world` is a static web application hosted on GitHub Pages (`flexiondotorg.github.io`). It is built by `cmd/site` and `internal/site`, driven by the single `vitrine.yaml` configuration file shared with `cmd/vitrine`.

## Hosting and security

GitHub Pages serves the site from `flexiondotorg/flexiondotorg` with custom domain `wimpys.world`. Domain Name System (DNS) records live in Cloudflare:

- Apex `A` and `AAAA` records point `wimpys.world` to GitHub Pages.
- A `CNAME` record points `www.wimpys.world` to `flexiondotorg.github.io`.
- Cloudflare Redirect Rules permanently redirect `wimpysworld.link` (and `www.wimpysworld.link`) to `https://links.wimpys.world`, and `links.wimpys.world` to `https://wimpys.world/#links`, opening the site directly on the `Links` deck.
- GitHub Pages enforces HTTPS for the custom domain.

Deployment uses zero long-lived Personal Access Tokens (PATs). The scheduled `.github/workflows/pages.yml` workflow uses top-level `permissions: {}`, `contents: read` on the build job, and `pages: write` plus `id-token: write` on the deploy job with `GITHUB_TOKEN` only.

## Visual direction and hybrid tabletop architecture

The site renders a stripped-back, Balatro-inspired Top Trumps floating playing-card tabletop in **Catppuccin** palettes matching [docs/vitrine.md](vitrine.md). A single full-viewport fixed **PixiJS v8** canvas (`vendor/pixi.min.mjs`, vendored with zero npm build steps) runs behind a minimal HTML layer for the header, accessible focus buttons, quiet footer, and zoomed detail modal.

- **Minimal header and editorial intro**: Shows the brand mark, title, subtitle, and a three-way icon theme switch (**System**, **Light** for Catppuccin Latte, and **Dark** for Catppuccin Mocha) persisted in `localStorage`, followed by a balanced two-column intro bio divided by a centred vertical Catppuccin aurora bar directly above the card decks (collapsing to a left-accented stack on narrow viewports).
- **Subtle tabletop footer**: Pinned at the bottom of the page beneath a thin Catppuccin aurora fade rule and playing-card diamond pip, showing copyright, `BlueOak-1.0.0` licence link, stack credit (`Go, PixiJS, and Catppuccin`), repository link, subtle `Sponsor` link, and the last-updated date.
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

Content groups into three stacked card decks placed directly above the floating cards: `Projects`, `Links`, and `Media`. `vitrine.yaml` validates `site.default_deck` against `projects`, `links`, and `media`. Deep links via `#projects`, `#links`, `#media`, `?deck=<name>`, or the `links.wimpys.world` hostname select the initial deck automatically, and switching decks syncs the URL hash.

### Stacked deck tray

The tabletop displays three portrait `5:7` Catppuccin playing-card stacks (`Projects`, `Links`, `Media`) in a single row above the floating cards. As a deck deals out sequentially, its physical stack thins card by card and reveals a recessed dashed placeholder slot. Selecting another deck vacuums the dealt cards rapidly back into their home deck with arrival sparkles as the stack builds back up, then sequentially deals the newly chosen deck.

### Portrait `5:7` Top Trumps card face

Each card uses `224px x 314px` logical portrait playing-card proportions, drawn onto `2x` (`448px x 628px`) offscreen canvases and uploaded to mipmapped, linearly filtered `PIXI.Texture` sprites for crisp GPU anti-aliased rendering:

- **Top index row**: Language or category dot and label on the left, paired with a release tag, episode number, read time, or live badge in Fira Code on the right.
- **Upper illustration panel**: Clean unframed artwork using sanitized inline project SVG logos, PNG data URIs, acoustic waveforms for `jive-*` slugs, monogram fallbacks, podcast cover art, or inline Simple Icons brand marks.
- **Title and description**: Bold Work Sans title with multi-line descriptions sized dynamically to fill the space above the bottom stat band (`12.4px` on Projects cards so every project description fits in full).
- **Bottom-anchored Top Trumps stat rows**: One to two alternating tinted rows anchored to the card bottom with uppercase labels on the left and bold tabular Fira Code values on the right:
  - **Projects**: `STARS` and `UPDATED`.
  - **Links**: Live zero-secret stats where public endpoints exist (Discord `ONLINE`/`MEMBERS`, GitHub `STARS`/`FOLLOWERS`, Mastodon/GoToSocial `FOLLOWERS`/`POSTS`, Bluesky `FOLLOWERS`/`POSTS`, Twitch `FOLLOWERS`/`CHANNEL`, YouTube `SUBSCRIBERS`/`VIDEOS`, Steam `GAMES`/`LEVEL`, itch.io `GAMES`/`CREATOR`, Blog `LATEST`/`SITE`, Linux Matters `LATEST`/`PUBLISHED`), crisp single-row identity fields (`HANDLE`, `PROFILE`, `SITE`) for static destinations, or custom `site.links[].stats` pairs from `vitrine.yaml`.
  - **Media**: Individual Linux Matters podcast episodes (`DURATION` and `PUBLISHED`) and Wimpy's World blog posts (`PUBLISHED` and `SITE`), without aggregate hub cards.

### Physical pick-up, mid-air 3D flip, and portrait `5:7` back face

Clicking a card or activating its keyboard button proxy captures the card's live screen centre, rotation, scale, and `2x` front canvas snapshot, hides the PixiJS sprite on the same frame, and flies a 3D card (`460ms` cubic ease-out, `rotateY(0deg -> 180deg)`, subtle mid-air `rotateX` lift) to viewport centre at portrait `5:7` playing-card size (`min(440px, 90vw)` by `min(616px, 86vh)`). Once landed, the centred card keeps a calm reduced float (`±3px`, `±0.45deg` over `5.6s`) that pauses smoothly when hovering or focusing interactive links.

The flipped back face (`role="dialog"`, `aria-modal="true"`, `aria-labelledby="modal-title"`) keeps the playing-card border, top aurora bar, and an internal scroll area with individually clickable details:

- **Projects**: Clickable logo and title (`Homepage` or `RepoURL`), clickable `owner/repo` link, untruncated description, clickable star and release pills, icon action buttons (`Homepage` and `GitHub`), and an **Install** two-column grid of clickable Haberdasher package repository tiles (`https://<host>/#<slug>`) alongside a `Releases` archive box tile (`Direct downloads` or latest tag).
- **Links**: Brand-matched back face carrying each card's front accent on the top bar, brand icon, category badge, and primary visit button (with WCAG AA contrast tokens in both Mocha and Latte), full description, live Discord online and member counts or GitHub stats, and on the GitHub card a secondary `Sponsor on GitHub` button plus clickable active and past sponsor `@login` chips.
- **Media**: Individual Linux Matters Podcast episode cards with dedicated Linux Matters charcoal (`#141414`) and amber-gold (`#feb81e`) back-face branding in dark mode (and light Latte surfaces with `#feb81e` controls and deep amber text in light mode), mini HTML5 `<audio>` player, and single-row subscribe pills for Apple Podcasts, Spotify, YouTube, and RSS; plus individual Wimpy's World Blog post cards styled with their peach accent and in-browser Web Speech API text-to-speech mini player. Audio and speech stop cleanly when the card closes.

Visitors dismiss the card by clicking any non-interactive area of the flipped card, clicking outside it, or pressing `Esc`, which triggers a `340ms` reverse flight (`rotateY(180deg -> 0deg)`) back into the card's live tabletop slot before revealing the PixiJS sprite and clearing any lingering pointer selection. Focus stays trapped inside the back face while open, keyboard opens restore focus on `Esc`, and `prefers-reduced-motion: reduce` opens and closes cleanly without flight, spin, or centre float.

## Motion, interaction, and accessibility

- **Scroll-synced open canvas**: The PixiJS world container tracks `world.y = -window.scrollY` alongside a `#scroll-spacer` element so cards fly in freely from the viewport edges and scroll naturally without clipping.
- **Sequential deck deal-out and fast recall**: Cards launch farthest-first (`62ms` stagger) from their portrait `5:7` deck at deck scale, glide out with a gentle dealer wrist flick (`±28deg` to `±42deg`) and arrival sparkles, and leave a dashed placeholder behind. Switching decks reclaims dealt cards nearest-first (`30ms` stagger, `310ms` flight, `±22deg` to `±35deg` bank) with arrival sparkles as the home deck rebuilds, then chains straight into the newly selected deck's sequential deal.
- **Idle float and hover 3D perspective tilt**: Cards bob and rock gently (`±4px`, `±1.6deg`). Hovering or focusing a card pops it forward (`scale: 1.06`), raises its z-order, projects true 3D perspective corner tilt via `PIXI.PerspectiveMesh` (`±12deg` yaw, `±10deg` pitch, `850px` camera distance) with opposing shadow lift, and glides a restrained dual-diagonal Catppuccin satin sheen across the tilted card face.
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
