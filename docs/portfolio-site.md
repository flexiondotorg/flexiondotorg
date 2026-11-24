# Portfolio site design and architecture (`wimpys.world`)

The portfolio site at `https://wimpys.world` is a static web application hosted on GitHub Pages (`flexiondotorg.github.io`). It replaces the current Tailor static holding page in `pages/` and shares the Go data packages built for Vitrine.

## Hosting and security

GitHub Pages serves the site from `flexiondotorg/flexiondotorg` with custom domain `wimpys.world`. Domain Name System (DNS) records live in Cloudflare:

- Apex `A` and `AAAA` records point `wimpys.world` to GitHub Pages.
- A `CNAME` record points `www.wimpys.world` to `flexiondotorg.github.io`.
- GitHub Pages enforces HTTPS for the custom domain.

Deployment uses zero long-lived Personal Access Tokens (PATs). The scheduled build workflow uses only the built-in `GITHUB_TOKEN` and standard OpenID Connect (OIDC) permissions for GitHub Pages.

## Visual direction

The site uses modern, clean, crisp **Catppuccin** floating cards that match the palette and typography in [docs/vitrine.md](vitrine.md). **Catppuccin Mocha** is the default dark theme, and a one-click toggle switches to **Catppuccin Latte** light mode while respecting `prefers-color-scheme` on initial load.

Typography uses **Work Sans** for headings and interface copy alongside **Fira Code** for release versions, star counts, and package metadata, with clean system font fallbacks. Every card uses a single crisp `1.5px` rounded border, generous breathing space, subtle Catppuccin aurora accents, and unframed project logos without ornate frames or Tarot filigree.

| Role | Catppuccin Mocha (dark) | Catppuccin Latte (light) |
| --- | --- | --- |
| Stage background | Crust (`#11111b`) | Crust (`#dce0e8`) |
| Primary card canvas | Base (`#1e1e2e`) | Base (`#eff1f5`) |
| Elevated surface | Surface0 (`#313244`) | Mantle (`#e6e9ef`) |
| Card border (`1.5px`) | Surface1 (`#45475a`) | Surface1 (`#bcc0cc`) |
| Primary ink | Text (`#cdd6f4`) | Text (`#4c4f69`) |
| Secondary ink | Subtext1 (`#bac2de`) | Subtext1 (`#5c5f77`) |
| Muted ink | Overlay0 (`#6c7086`) | Overlay0 (`#8c8fa1`) |
| Aurora accents | Mauve, Lavender, Blue, Teal (`#cba6f7`, `#b4befe`, `#89b4fa`, `#94e2d5`) | Mauve, Lavender, Blue, Teal (`#8839ef`, `#7287fd`, `#1e66f5`, `#179299`) |
| Positive accent | Green (`#a6e3a1`) | Green (`#40a02b`) |
| Alert accent | Red (`#f38ba8`) | Red (`#d20f39`) |
| Info or link accent | Sapphire (`#74c7ec`) | Sapphire (`#209fb5`) |

## Card and deck anatomy

Content groups into stacked card decks: `Projects`, `Packages`, `Links`, and an optional `Media` deck. Each card has three visual states: a stacked back, a fanned front face, and a flipped detail face.

### Card back

Stacked decks show a clean Catppuccin card back with a single `1.5px` border, a subtle Catppuccin aurora accent band, and the centred deck title (`Projects`, `Packages`, `Links`, or `Media`) in Work Sans. Selecting a deck fans its cards out onto the stage.

### Card front

The front face balances clean Catppuccin styling with rapid scannability:

- **Single crisp border**: Rounded `1.5px` border over Mocha Base (`#1e1e2e`) or Latte Base (`#eff1f5`) with no double frames or corner studs.
- **Top metadata row**: Primary language indicator and latest release tag set in Fira Code.
- **Unframed project logo**: Inline project logo placed cleanly on the card canvas with no medallion frame, resolved through the shared Haberdasher logo pipeline (`pages/icon.svg`, `pages/icon.png`, `icon.svg`, `icon.png`, configured PNG fallbacks, `logo_overrides`, and placeholder mark).
- **Title**: Project or entry name set cleanly in Work Sans below the logo.
- **Compact stats row**: Star count, latest release age, count of package hosts, and primary language.

### Flipped zoomed detail face

Selecting a fanned card brings it forward and flips it with `rotateY(180deg)` to expose a zoomed detail face:

- Full project or site description.
- Direct link to the latest GitHub release.
- Clickable Haberdasher package tiles linking to `https://<host>/#<package>` for every host that distributes the package.
- Homepage link and GitHub repository link.

Visitors dismiss the zoomed card by clicking outside the card bounds or pressing `Esc`.

## Interaction, physics, and accessibility

The front end uses vanilla JavaScript and Cascading Style Sheets (CSS) 3D transforms (`translate3d`, `rotateX`, and `rotateY`) with no heavy front-end framework.

- **Floating stacked decks and air-current physics**: Stacked decks fan out into gentle air-current drift across the stage.
- **Pointer-wake breeze with hover dampening**: Pointer motion applies a subtle wake impulse to nearby cards. Hovering over a card dampens its motion so the card settles cleanly under the pointer for reliable clicking.
- **Grid view and theme toggles**: One-click controls switch immediately between floating cards and a flat **Grid view**, and toggle between **Catppuccin Mocha** and **Catppuccin Latte**.
- **Keyboard navigation**: Visitors can move across decks and cards with `Tab` and arrow keys, open or flip a card with `Enter` or `Space`, and dismiss the zoomed face with `Esc`.
- **Automatic calm layout**: Touchscreens (coarse-pointer media queries) and `@media (prefers-reduced-motion: reduce)` default directly to the calm grid layout with air-current drift disabled.

## Data pipeline and deployment transition

The portfolio build reuses the Go `internal/` packages from Vitrine (`internal/config`, `internal/github`, `internal/haberdasher`, `internal/rss`, and optionally `internal/discord` and `internal/twitch`).

### Portfolio data assembly

The site generator combines two data streams into static HTML and JSON build assets:

1. **Haberdasher projects**: Every curated project in `haberdasher.yaml` enriched with metadata from the GitHub Representational State Transfer (REST) API and all eight `*.wimpys.world/haberdasher.json` manifests, sorted descending by latest release date on GitHub.
2. **Curated site entries**: Non-Haberdasher entries configured in YAML, such as the Linux Matters website (`https://linuxmatters.sh`) and the Wimpy's World blog (`https://wimpysworld.com`).

### Transition from Tailor Pages

Currently `.tailor.yml` sets `pages.enabled: true` (`generator: static`, `path: pages`, `cname: wimpys.world`) and manages `.github/workflows/tailor-pages.yml` (`alteration: always`) to publish the holding page from `pages/`.

When Track 2 goes live, the repository transitions in four steps:

1. Set `pages.enabled: false` in `.tailor.yml` so Tailor stops managing GitHub Pages and `tailor-pages.yml`.
2. Remove `.github/workflows/tailor-pages.yml` and the holding-page files that the generator replaces.
3. Add a custom workflow for GitHub Pages triggered on `main` pushes, a recurring schedule, and `workflow_dispatch`.
4. Build the site into a temporary output directory during the workflow and deploy it with `actions/upload-pages-artifact` and `actions/deploy-pages`.

The custom workflow uses minimal permissions (`contents: read` for the build job, plus `pages: write` and `id-token: write` for the deploy job). The workflow never commits generated HTML or build output to Git.

### Local preview and rendered verification

Development and Continuous Integration (CI) verification follow two checks before deployment:

- **Local preview**: A local build and serve task (exposed via `just pages`) generates the site and serves it over HTTP for live browser inspection.
- **Playwright verification**: Automated browser checks verify layout, contrast in both Catppuccin Mocha and Latte themes, card flipping, outside-click and `Esc` dismissal, the **Grid view** and theme toggles, keyboard navigation, and zero broken assets or console errors across desktop, mobile, and `@media (prefers-reduced-motion: reduce)` states.

## Open questions

These decisions stay open until Track 1 (`vitrine`) ships:

- **Initial deck state**: Decide whether the `Projects` deck loads already fanned on the stage or stacked until clicked.
- **Curated non-Haberdasher entries**: Finalise the exact list of external sites, podcasts, talks, and social cards included outside `haberdasher.yaml`.
- **Rebuild cadence**: Set the cron schedule for the custom workflow for GitHub Pages.
