# Portfolio site design and architecture (`wimpys.world`)

The portfolio site at `https://wimpys.world` is a static web application hosted on GitHub Pages (`flexiondotorg.github.io`). It is built by `cmd/site` and `internal/site`, driven by the single `vitrine.yaml` configuration file shared with `cmd/vitrine`.

## Hosting and security

GitHub Pages serves the site from `flexiondotorg/flexiondotorg` with custom domain `wimpys.world`. Domain Name System (DNS) records live in Cloudflare:

- Apex `A` and `AAAA` records point `wimpys.world` to GitHub Pages.
- A `CNAME` record points `www.wimpys.world` to `flexiondotorg.github.io`.
- GitHub Pages enforces HTTPS for the custom domain.

Deployment uses zero long-lived Personal Access Tokens (PATs). The scheduled `.github/workflows/pages.yml` workflow uses top-level `permissions: {}`, `contents: read` on the build job, and `pages: write` plus `id-token: write` on the deploy job with `GITHUB_TOKEN` only.

## Visual direction

The site uses clean **Catppuccin** floating cards matching [docs/vitrine.md](vitrine.md). **Catppuccin Mocha** is the default dark theme, and a one-click toggle switches to **Catppuccin Latte** light mode while respecting `prefers-color-scheme` on first load.

Typography uses **Work Sans** for headings and interface copy alongside **Fira Code** (`tabular-nums`) for release versions, star counts, and package metadata, with system font fallbacks. Every card uses a single `1.5px` rounded border (`14px` to `16px` radius), subtle Catppuccin aurora accents, and unframed artwork without double borders or corner studs. Both themes enforce Web Content Accessibility Guidelines (WCAG) AA contrast ratios across text and interactive controls.

| Role | Catppuccin Mocha (dark) | Catppuccin Latte (light) |
| --- | --- | --- |
| Stage background | Crust (`#11111b`) | Crust (`#dce0e8`) |
| Primary card canvas | Base (`#1e1e2e`) | Base (`#eff1f5`) |
| Elevated surface | Surface0 (`#313244`) | Mantle (`#e6e9ef`) |
| Card border (`1.5px`) | Surface1 (`#45475a`) | Surface1 (`#bcc0cc`) |
| Primary ink | Text (`#cdd6f4`) | Text (`#4c4f69`) |
| Secondary ink | Subtext1 (`#bac2de`) | Subtext1 (`#5c5f77`) |
| Aurora accents | Mauve, Lavender, Blue, Teal (`#cba6f7`, `#b4befe`, `#89b4fa`, `#94e2d5`) | Mauve, Lavender, Blue, Teal (`#8839ef`, `#7287fd`, `#1e66f5`, `#179299`) |
| Positive accent | Green (`#a6e3a1`) | Green (`#40a02b`) |
| Alert accent | Red (`#f38ba8`) | Red (`#d20f39`) |
| Info or link accent | Sapphire (`#74c7ec`) | Sapphire (`#209fb5`) |

## Card and deck anatomy

Content groups into four stacked card decks: `Projects`, `Packages`, `Links`, and `Media`. Each card has three visual states: a stacked deck back, a fanned front face, and a flipped zoomed detail face.

### Card back and deck dock

The dock displays tactile three-layer Catppuccin card-back stacks for `Projects`, `Packages`, `Links`, and `Media`. On desktop load, `Projects` starts fanned across the stage while `Packages`, `Links`, and `Media` wait stacked in the dock. Selecting any stacked deck fans its cards onto the stage.

### Card front

- **Single crisp border**: Rounded `1.5px` border over Mocha Base (`#1e1e2e`) or Latte Base (`#eff1f5`) with a thin aurora top band.
- **Top metadata row**: Primary language or category on the left, and latest release tag, package count, or live badge in Fira Code on the right.
- **Unframed artwork**: Sanitized inline project or host SVG logo, PNG data URI, acoustic waveform for `jive-*` slugs, monogram fallback, podcast cover art, or inline Simple Icons brand mark.
- **Title and summary**: Name set in Work Sans with a two-line summary.
- **Compact footer**: Star count, release age, host count, package previews, or duration and reading time in Fira Code.

### Flipped zoomed detail face

Selecting a card opens a centred modal dialog that flips with CSS 3D `rotateY(180deg)` to reveal the detail face:

- **Projects**: Full description, star count, language, latest release tag and age with direct release link, Homepage and GitHub links, and clickable Haberdasher package tiles (`https://<host>/#<slug>`) with each host's inline SVG logo.
- **Packages**: Host logo, host URL, package count, and clickable links to `https://<host>/#<pkgName>` for every published package on that host.
- **Links**: Full description, category, live Discord or GitHub Sponsors metadata when available, and primary open link.
- **Media**: Linux Matters Podcast hub (cover art, recent episodes, and subscribe pills for Apple Podcasts, Spotify, YouTube, and RSS), Wimpy's World Blog hub (recent posts with reading time and summaries), and individual episode and post cards.

Visitors dismiss the modal by clicking outside the card, clicking the close button, or pressing `Esc`. Focus stays trapped inside the modal while open and returns to the triggering card on close.

## Interaction, physics, and accessibility

The front end uses vanilla JavaScript and CSS 3D transforms (`translate3d`, `rotate`, and `rotateY`) with no framework runtime.

- **Air-current constellation**: Desktop views compute a non-overlapping constellation layout that adapts to viewport width, with cards drifting gently via `requestAnimationFrame`.
- **Pointer-wake breeze with hover dampening**: Pointer motion applies a soft wake impulse to nearby cards. Hovering or focusing a card dampens its motion to zero and lifts it for reliable clicking.
- **Toolbar controls**: One-click buttons switch decks, sort `Projects` by `Stars` (default) or `Latest release`, toggle between `Floating` and `Grid` views, pause or resume drift, and switch between `Mocha` and `Latte`.
- **Keyboard navigation**: Visitors move across cards with `Tab` and arrow keys, open a card with `Enter` or `Space`, and dismiss details with `Esc`.
- **Automatic calm layout**: Coarse-pointer devices (`@media (pointer: coarse)`) and `@media (prefers-reduced-motion: reduce)` default directly to the calm `Grid` layout with no animation loop running.

## Single-config data pipeline (`cmd/site` and `internal/site`)

Both `cmd/vitrine` and `cmd/site` read `vitrine.yaml` via `internal/config` (`SiteConfig` and `SiteLink` under `site:`).

`cmd/site` accepts `-config`, `-out`, `-verbose`, and `-version`. It reuses `internal/haberdasher`, `internal/github`, `internal/discord`, and `internal/rss` to assemble data, passes it to `internal/site.Build`, and emits static output (`index.html`, `data.json`, `style.css`, `app.js`, `icon.svg`, `.nojekyll`, and `CNAME`) via `internal/site.Write` using `//go:embed web/*`.

Curated external entries live under `site.links` in `vitrine.yaml`:

- **Sites**: Wimpy's World Blog, Linux Matters, Oval Tutu.
- **Community**: Wimpy's World Discord (enriched with live online and member counts), Matrix.
- **Social**: Mastodon, Bluesky, LinkedIn.
- **Video**: YouTube, Twitch.
- **Gaming**: Steam, itch.io.
- **Support**: GitHub Sponsors (enriched with live active and past sponsor counts).

## Deployment workflow and Tailor transition

`.github/workflows/pages.yml` builds and deploys the site on `main` pushes, every 6 hours at 15 minutes past the hour (`cron: "15 */6 * * *"`), and `workflow_dispatch`. Generated HTML is built into `${RUNNER_TEMP}/site` and uploaded via `actions/upload-pages-artifact` and `actions/deploy-pages`; it is never committed to Git.

To retire the legacy Tailor holding page in `pages/`, set `pages.enabled: false` in `.tailor.yml`, remove `.github/workflows/tailor-pages.yml` and the holding-page files, and verify with `tailor measure`.

## Local preview and verification

- **Local preview**: Run `just site` to build the portfolio site from `vitrine.yaml` and serve it locally at `http://127.0.0.1:18475`.
- **Automated checks**: Run `just build`, `just test`, `go test -race ./...`, `golangci-lint run --config .golangci.yml ./...`, and `actionlint` before pushing.
- **Rendered verification**: Inspect the live preview in headless Chromium across desktop, mobile, Mocha, Latte, and `prefers-reduced-motion: reduce` states.
