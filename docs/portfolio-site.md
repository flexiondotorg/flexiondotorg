# Portfolio site design and architecture (`wimpys.world`)

The portfolio site at `https://wimpys.world` is a static web application hosted on GitHub Pages (`flexiondotorg.github.io`). It replaces the current Tailor static holding page in `pages/` and shares the Go data packages built for Vitrine.

## Hosting and security

GitHub Pages serves the site from `flexiondotorg/flexiondotorg` with custom domain `wimpys.world`. Domain Name System (DNS) records live in Cloudflare:

- Apex `A` and `AAAA` records point `wimpys.world` to GitHub Pages.
- A `CNAME` record points `www.wimpys.world` to `flexiondotorg.github.io`.
- GitHub Pages enforces HTTPS for the custom domain.

Deployment uses zero long-lived Personal Access Tokens (PATs). The scheduled build workflow uses only the built-in `GITHUB_TOKEN` and standard OpenID Connect (OIDC) permissions for GitHub Pages.

## Visual direction

The site uses a dark collector's parlour and luxury Tarot deck direction that shares the **Obsidian and Gilt** palette from [docs/vitrine.md](vitrine.md). Visual treatment follows restrained Savile Row and collector craft: crisp hairline borders, subtle foil highlights, generous dark velvet space, and zero kitschy magician clipart.

| Role | Name | Hex values |
| --- | --- | --- |
| Page background | Velvet ground | `#09090b` |
| Primary card stock | Obsidian | `#121216` |
| Elevated surface | Obsidian elevated | `#19191f` |
| Outer card rim and primary ink | Ivory | `#f4efe4` |
| Secondary ink | Stone | `#b3aa99` |
| Muted ink | Ash | `#7d7668` |
| Gilt foil gradient (3-stop) | Gilt | `#f0d890`, `#cfa84f`, `#7a5e25` |
| Positive accent | Emerald | `#4fb885` |
| Alert accent | Ruby | `#c9545c` |
| Info or link accent | Sapphire | `#6c9bc9` |

## Card and deck anatomy

Content groups into stacked card decks: `Projects`, `Packages`, `Links`, and an optional `Media` deck. Each card has three visual states: a stacked back, a fanned front face, and a flipped detail face.

### Card back

Stacked decks show a black-and-gold filigree card back stamped with the deck title (`Projects`, `Packages`, `Links`, or `Media`). Selecting a deck fans its cards out onto the table.

### Card front

The front face combines Tarot proportion with Top Trumps scannability:

- **Outer rim and frame**: Crisp ivory outer edge (`#f4efe4`) framing a gilt hairline inner border over obsidian stock (`#121216`).
- **Corner index pips**: Glyph for the primary language and latest release tag positioned in the corners.
- **Centre crest**: Inline project logo resolved through the shared Haberdasher logo pipeline.
- **Title**: Project or entry name set cleanly below the crest.
- **Compact stats plaque**: Star count, latest release age, count of package hosts, and primary language.

### Flipped zoomed detail face

Selecting a fanned card brings it forward and flips it with `rotateY(180deg)` to expose a zoomed detail face:

- Full project or site description.
- Direct link to the latest GitHub release.
- Clickable package badges from Haberdasher linking to `https://<host>/#<package>` for every host that distributes the package.
- Homepage link and GitHub repository link.

Visitors dismiss the zoomed card by clicking outside the card bounds or pressing `Esc`.

## Interaction, physics, and accessibility

The front end uses vanilla JavaScript and Cascading Style Sheets (CSS) 3D transforms (`translate3d`, `rotateX`, and `rotateY`) with no heavy front-end framework.

- **Tactile decks and gentle orbit**: Stacked decks fan out into a slow floating orbit across the velvet stage.
- **Pointer-wake breeze with hover dampening**: Pointer motion applies a subtle wake impulse to nearby cards. Hovering over a card dampens its motion so the card settles cleanly under the cursor for reliable clicking.
- **Grid view toggle**: A one-click **Grid view** control switches immediately between the floating orbit and a flat scannable grid.
- **Keyboard navigation**: Visitors can move across decks and cards with `Tab` and arrow keys, open or flip a card with `Enter` or `Space`, and dismiss the zoomed face with `Esc`.
- **Automatic calm layout**: Touchscreens (coarse-pointer media queries) and `@media (prefers-reduced-motion: reduce)` default directly to the calm grid layout with orbital motion disabled.

## Data pipeline and deployment transition

The portfolio build reuses the Go `internal/` packages from Vitrine (`internal/config`, `internal/github`, `internal/haberdasher`, `internal/rss`, and optionally `internal/twitch`).

### Portfolio data assembly

The site generator combines two data streams into static HTML and JSON build assets:

1. **Haberdasher projects**: Every curated project in `haberdasher.yaml` enriched with metadata from the GitHub REST API and all eight `*.wimpys.world/haberdasher.json` manifests, sorted descending by latest release date on GitHub.
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

- **Local preview**: A local build and serve task (exposed via `just`) generates the site and serves it over HTTP for live browser inspection.
- **Playwright verification**: Automated browser checks verify layout, contrast, card flipping, outside-click and `Esc` dismissal, the **Grid view** toggle, keyboard navigation, and zero broken assets or console errors across desktop, mobile, and `@media (prefers-reduced-motion: reduce)` states.

## Open questions

These decisions stay open until Track 1 (`vitrine`) ships:

- **Initial deck state**: Decide whether the `Projects` deck loads already fanned in orbit or stacked until clicked.
- **Curated non-Haberdasher entries**: Finalise the exact list of external sites, podcasts, talks, and social cards included outside `haberdasher.yaml`.
- **Typography**: Choose between self-hosted serif and sans-serif fonts and a pure system font stack.
- **Rebuild cadence**: Set the cron schedule for the custom workflow for GitHub Pages.
